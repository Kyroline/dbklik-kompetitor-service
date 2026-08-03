// Package cache defines a driver-agnostic cache contract. Concrete drivers
// (Redis, in-memory, etc.) are provided by each module's infrastructure layer
// and satisfy this interface, following the same extension-point pattern as
// repositories.
package cache

import (
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
}

type item struct {
	value     interface{}
	expiresAt time.Time
	hasTTL    bool
}

// MemoryCache is a minimal in-process Cache implementation, useful as a
// default driver or in tests when no external cache is configured.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]item)}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if it.hasTTL && time.Now().After(it.expiresAt) {
		return nil, false
	}
	return it.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	it := item{value: value}
	if ttl > 0 {
		it.hasTTL = true
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
