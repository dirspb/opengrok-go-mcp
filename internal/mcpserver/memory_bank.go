package mcpserver

import "sync"

type MemoryBank struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewMemoryBank() *MemoryBank {
	return &MemoryBank{store: make(map[string]string)}
}

func (m *MemoryBank) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = value
}

func (m *MemoryBank) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.store[key]
	return v, ok
}

func (m *MemoryBank) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
}

func (m *MemoryBank) List() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copy := make(map[string]string, len(m.store))
	for k, v := range m.store {
		copy[k] = v
	}
	return copy
}

func (m *MemoryBank) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]string)
}
