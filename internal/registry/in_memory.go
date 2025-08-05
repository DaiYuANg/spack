package registry

import (
	"fmt"
	"github.com/eko/gocache/lib/v4/cache"
	"sync"
)

type InMemoryRegistry struct {
	mu   sync.RWMutex
	data map[string]*Entry
	cm   *cache.Cache[string]
}

func NewInMemoryRegistry(cm *cache.Cache[string]) *InMemoryRegistry {
	return &InMemoryRegistry{
		data: make(map[string]*Entry),
		cm:   cm,
	}
}

func (r *InMemoryRegistry) Register(path string, entry *Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[path]; exists {
		return fmt.Errorf("path already registered: %s", path)
	}

	r.data[path] = entry
	return nil
}

func (r *InMemoryRegistry) Lookup(path string) (*Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.data[path]
	return entry, ok
}

func (r *InMemoryRegistry) List() map[string]*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本防止外部改动原始数据
	copied := make(map[string]*Entry, len(r.data))
	for k, v := range r.data {
		copied[k] = v
	}
	return copied
}
