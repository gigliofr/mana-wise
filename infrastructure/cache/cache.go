package cache

import (
	"sync"
	"time"
)

// entry holds a cached value with its expiry.
type entry struct {
	value     interface{}
	expiresAt time.Time
}

// Cache is a thread-safe in-memory TTL cache.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// New creates an empty Cache. Start the optional janitor with StartJanitor.
func New() *Cache {
	return &Cache{entries: make(map[string]entry)}
}

// Set stores a value under key for the given TTL duration.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
}

// Get retrieves a value. Returns (nil, false) if not found or expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// StartJanitor starts a background goroutine that evicts expired entries
// every interval. The goroutine stops when the provided done channel is closed.
func (c *Cache) StartJanitor(interval time.Duration, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				c.evict()
			}
		}
	}()
}

func (c *Cache) evict() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}
