package query

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// CachedBackend wraps a Backend with TTL-based caching.
type CachedBackend struct {
	Backend Backend
	TTL     time.Duration

	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	result    *QueryResult
	expiresAt time.Time
}

func NewCachedBackend(backend Backend, ttl time.Duration) *CachedBackend {
	return &CachedBackend{
		Backend: backend,
		TTL:     ttl,
		cache:   make(map[string]*cacheEntry),
	}
}

func (c *CachedBackend) Execute(ctx context.Context, connection string, query string) (*QueryResult, error) {
	key := cacheKey(connection, query)

	// Check cache.
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.result, nil
	}

	// Cache miss or expired — execute.
	result, err := c.Backend.Execute(ctx, connection, query)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.TTL),
	}
	c.mu.Unlock()

	return result, nil
}

// Invalidate clears all cached entries.
func (c *CachedBackend) Invalidate() {
	c.mu.Lock()
	c.cache = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

func cacheKey(connection, query string) string {
	h := sha256.Sum256([]byte(connection + "\x00" + query))
	return fmt.Sprintf("%x", h)
}
