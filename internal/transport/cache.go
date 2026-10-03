package transport

import (
	"sync"
	"time"
)

// cacheEntry holds cached data with its last update time.
type cacheEntry struct {
	data        interface{}
	lastUpdated time.Time
}

// cache provides thread-safe caching for multiple data types.
type cache struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]cacheEntry
}

// newCache creates a new cache with the specified TTL.
func newCache(ttl time.Duration) *cache {
	return &cache{
		ttl:  ttl,
		data: make(map[string]cacheEntry),
	}
}

// get retrieves data from cache if it exists and has not expired.
func (c *cache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || time.Since(entry.lastUpdated) >= c.ttl {
		return nil, false
	}
	return entry.data, true
}

// set stores data in cache with the current timestamp.
func (c *cache) set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheEntry{
		data:        data,
		lastUpdated: time.Now(),
	}
}

// Cache key constants for different data types.
const (
	CacheKeyDelegated    = "delegated"
	CacheKeyExtended     = "extended"
	CacheKeyAssigned     = "assigned"
	CacheKeyIPv6Assigned = "ipv6-assigned"
	CacheKeyLegacy       = "legacy"
	CacheKeyTransfers    = "transfers"
	CacheKeyChanges      = "changes"
)

// CacheKeyIRR returns the cache key for an IRR database of the given object type.
func CacheKeyIRR(objType string) string {
	return "irr:" + objType
}

// CacheGet retrieves data from the client's cache if it exists and has not
// expired. It is the exported accessor used by the root-package caching layer
// (the cache type itself is unexported, so subpackages cannot reach into it).
func (c *Client) CacheGet(key string) (interface{}, bool) {
	if c.cache == nil {
		return nil, false
	}
	return c.cache.get(key)
}

// CacheSet stores data in the client's cache with the current timestamp.
func (c *Client) CacheSet(key string, data interface{}) {
	if c.cache == nil {
		return
	}
	c.cache.set(key, data)
}
