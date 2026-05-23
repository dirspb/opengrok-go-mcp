package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheGetSetDelete(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("key1", "value1")
	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit for key1")
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %v", val)
	}

	c.Delete("key1")
	_, ok = c.Get("key1")
	if ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestCacheClear(t *testing.T) {
	c := New(1 * time.Minute)

	c.Set("a", 1)
	c.Set("b", 2)
	if c.Size() != 2 {
		t.Fatalf("expected size 2, got %d", c.Size())
	}

	c.Clear()
	if c.Size() != 0 {
		t.Fatalf("expected size 0 after clear, got %d", c.Size())
	}

	_, ok := c.Get("a")
	if ok {
		t.Fatal("expected cache miss after clear")
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	c := New(50 * time.Millisecond)

	c.Set("key", "value")
	val, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit before TTL expiration")
	}
	if val != "value" {
		t.Fatalf("expected value, got %v", val)
	}

	time.Sleep(60 * time.Millisecond)
	_, ok = c.Get("key")
	if ok {
		t.Fatal("expected cache miss after TTL expiration")
	}
}

func TestCacheSetWithTTLOverridesDefault(t *testing.T) {
	c := New(1 * time.Minute)

	c.SetWithTTL("key", "value", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected cache miss after custom TTL expiration")
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := New(1 * time.Minute)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			c.Set(key, n)
			c.Get(key)
			c.Delete(key)
		}(i)
	}

	wg.Wait()
}

func TestCacheSize(t *testing.T) {
	c := New(1 * time.Minute)

	if c.Size() != 0 {
		t.Fatalf("expected size 0 for empty cache, got %d", c.Size())
	}

	c.Set("a", 1)
	c.Set("b", 2)
	if c.Size() != 2 {
		t.Fatalf("expected size 2, got %d", c.Size())
	}

	c.Set("a", 3)
	if c.Size() != 2 {
		t.Fatalf("expected size 2 after overwrite, got %d", c.Size())
	}
}

func TestCacheTrimToSizeEvictsLeastRecentlyUsed(t *testing.T) {
	c := New(1 * time.Minute)
	c.Set("oldest", 1)
	c.Set("hot", 2)
	c.Set("newest", 3)

	if _, ok := c.Get("hot"); !ok {
		t.Fatal("hot entry is missing before trim")
	}

	c.TrimToSize(2)

	if c.Size() != 2 {
		t.Fatalf("size after trim = %d, want 2", c.Size())
	}
	if _, ok := c.Get("oldest"); ok {
		t.Fatal("oldest entry survived trim, want it evicted")
	}
	if _, ok := c.Get("hot"); !ok {
		t.Fatal("recently accessed entry was evicted")
	}
	if _, ok := c.Get("newest"); !ok {
		t.Fatal("newest entry was evicted")
	}
}
