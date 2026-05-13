package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// Cache is a generic thread-safe in-memory TTL cache.
type Cache[K comparable, V any] struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[K]entry[V]
}

// New creates a Cache with the given TTL duration.
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		ttl: ttl,
		m:   make(map[K]entry[V]),
	}
}

// Get returns the cached value for key if present and not expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	e, ok := c.m[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(e.expiresAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores value under key with the cache TTL.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	c.m[key] = entry[V]{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
