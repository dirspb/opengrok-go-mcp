// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key       string
	value     any
	expiresAt time.Time
}

// Cache is an in-memory key-value store with TTL expiration and LRU eviction.
type Cache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	order      *list.List
	defaultTTL time.Duration
}

// New creates a Cache with the given default TTL.
func New(defaultTTL time.Duration) *Cache {
	return &Cache{
		items:      make(map[string]*list.Element),
		order:      list.New(),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a value from the cache. The second return value is false
// if the key is missing or has expired. Expired entries are deleted on access.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*entry)
	if time.Now().After(e.expiresAt) {
		c.removeElement(el)
		return nil, false
	}
	c.order.MoveToFront(el)
	return e.value, true
}

// Set stores a value in the cache using the default TTL.
func (c *Cache) Set(key string, value any) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with an explicit TTL.
func (c *Cache) SetWithTTL(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		e := el.Value.(*entry)
		e.value = value
		e.expiresAt = time.Now().Add(ttl)
		c.order.MoveToFront(el)
		return
	}
	e := &entry{key: key, value: value, expiresAt: time.Now().Add(ttl)}
	el := c.order.PushFront(e)
	c.items[key] = el
}

// EvictOldest removes the least-recently-used entry.
func (c *Cache) EvictOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el := c.order.Back(); el != nil {
		c.removeElement(el)
	}
}

// TrimToSize drops expired entries, then evicts least-recently-used entries
// until the cache contains no more than maxSize items.
func (c *Cache) TrimToSize(maxSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if maxSize < 0 {
		maxSize = 0
	}
	now := time.Now()
	for el := c.order.Back(); el != nil; {
		previous := el.Prev()
		if !now.Before(el.Value.(*entry).expiresAt) {
			c.removeElement(el)
		}
		el = previous
	}
	for len(c.items) > maxSize {
		c.removeElement(c.order.Back())
	}
}

func (c *Cache) removeElement(el *list.Element) {
	c.order.Remove(el)
	e := el.Value.(*entry)
	delete(c.items, e.key)
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElement(el)
	}
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.order.Init()
}

// Size returns the number of items currently in the cache.
func (c *Cache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}
