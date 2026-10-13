package search

import (
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMaxEntries limits in-memory cache capacity to avoid unbounded memory growth.
	DefaultMaxEntries = 1000
)

type cacheItem struct {
	result    SearchResult
	expiresAt time.Time
}

// Cache provides a thread-safe in-memory TTL cache for search queries.
type Cache struct {
	mu         sync.RWMutex
	items      map[string]cacheItem
	ttl        time.Duration
	maxEntries int
	stopCh     chan struct{}
	stopped    bool
}

// NewCache creates a new in-memory TTL cache with the specified duration.
func NewCache(ttl time.Duration, maxEntries ...int) *Cache {
	if ttl <= 0 {
		ttl = 20 * time.Minute
	}
	limit := DefaultMaxEntries
	if len(maxEntries) > 0 && maxEntries[0] > 0 {
		limit = maxEntries[0]
	}

	c := &Cache{
		items:      make(map[string]cacheItem),
		ttl:        ttl,
		maxEntries: limit,
		stopCh:     make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// NormalizeQuery standardizes query strings by lowercasing, trimming, and collapsing whitespace.
func NormalizeQuery(q string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(q)), " "))
}

// Get retrieves a cached search result if present and unexpired.
func (c *Cache) Get(query string) (SearchResult, bool) {
	key := NormalizeQuery(query)
	if key == "" {
		return SearchResult{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return SearchResult{}, false
	}

	if time.Now().After(item.expiresAt) {
		return SearchResult{}, false
	}

	return item.result, true
}

// Set stores a search result in the cache with the configured TTL, evicting if capacity is reached.
func (c *Cache) Set(query string, res SearchResult) {
	key := NormalizeQuery(query)
	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If at capacity and adding a new key, purge expired entries first
	if _, exists := c.items[key]; !exists && len(c.items) >= c.maxEntries {
		now := time.Now()
		for k, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, k)
			}
		}

		// If still at capacity, evict the entry with the earliest expiry
		if len(c.items) >= c.maxEntries {
			var oldestKey string
			var earliestExpiry time.Time
			for k, item := range c.items {
				if oldestKey == "" || item.expiresAt.Before(earliestExpiry) {
					oldestKey = k
					earliestExpiry = item.expiresAt
				}
			}
			if oldestKey != "" {
				delete(c.items, oldestKey)
			}
		}
	}

	c.items[key] = cacheItem{
		result:    res,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(query string) {
	key := NormalizeQuery(query)
	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Len returns the count of items currently in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// PurgeExpired explicitly removes all expired items from the cache.
func (c *Cache) PurgeExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	purged := 0
	for k, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, k)
			purged++
		}
	}
	return purged
}

// Close stops the background cleanup loop.
func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.stopped {
		c.stopped = true
		close(c.stopCh)
	}
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.PurgeExpired()
		case <-c.stopCh:
			return
		}
	}
}
