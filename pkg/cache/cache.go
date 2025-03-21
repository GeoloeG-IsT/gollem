package cache

import (
	"context"
	"sync"
	"time"

	"github.com/user/gollem/pkg/core"
)

// Cache defines the interface for caching LLM responses
type Cache interface {
	// Get retrieves a cached response for a prompt
	Get(ctx context.Context, prompt *core.Prompt) (*core.Response, bool)
	
	// Set stores a response for a prompt
	Set(ctx context.Context, prompt *core.Prompt, response *core.Response) error
	
	// Invalidate removes a cached response for a prompt
	Invalidate(ctx context.Context, prompt *core.Prompt) error
	
	// Clear removes all cached responses
	Clear(ctx context.Context) error
}

// MemoryCache is an in-memory implementation of Cache
type MemoryCache struct {
	entries     map[string]cacheEntry
	mu          sync.RWMutex
	ttl         time.Duration
	maxEntries  int
	hashFunc    func(*core.Prompt) string
}

type cacheEntry struct {
	response  *core.Response
	timestamp time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(options ...MemoryCacheOption) *MemoryCache {
	cache := &MemoryCache{
		entries:    make(map[string]cacheEntry),
		ttl:        time.Hour,
		maxEntries: 1000,
		hashFunc:   defaultHashFunc,
	}
	
	for _, option := range options {
		option(cache)
	}
	
	return cache
}

// MemoryCacheOption is a function that configures a MemoryCache
type MemoryCacheOption func(*MemoryCache)

// WithTTL sets the time-to-live for cache entries
func WithTTL(ttl time.Duration) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.ttl = ttl
	}
}

// WithMaxEntries sets the maximum number of entries in the cache
func WithMaxEntries(max int) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.maxEntries = max
	}
}

// WithHashFunc sets the function used to hash prompts
func WithHashFunc(hashFunc func(*core.Prompt) string) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.hashFunc = hashFunc
	}
}

// Get retrieves a cached response for a prompt
func (c *MemoryCache) Get(ctx context.Context, prompt *core.Prompt) (*core.Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key := c.hashFunc(prompt)
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}
	
	// Check if the entry has expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}
	
	return entry.response, true
}

// Set stores a response for a prompt
func (c *MemoryCache) Set(ctx context.Context, prompt *core.Prompt, response *core.Response) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// If we've reached the maximum number of entries, remove the oldest one
	if len(c.entries) >= c.maxEntries {
		var oldestKey string
		var oldestTime time.Time
		
		// Find the oldest entry
		for key, entry := range c.entries {
			if oldestKey == "" || entry.timestamp.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.timestamp
			}
		}
		
		// Remove the oldest entry
		delete(c.entries, oldestKey)
	}
	
	key := c.hashFunc(prompt)
	c.entries[key] = cacheEntry{
		response:  response,
		timestamp: time.Now(),
	}
	
	return nil
}

// Invalidate removes a cached response for a prompt
func (c *MemoryCache) Invalidate(ctx context.Context, prompt *core.Prompt) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.hashFunc(prompt)
	delete(c.entries, key)
	
	return nil
}

// Clear removes all cached responses
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]cacheEntry)
	
	return nil
}

// defaultHashFunc is a simple hash function for prompts
func defaultHashFunc(prompt *core.Prompt) string {
	// In a real implementation, this would use a proper hashing algorithm
	// For simplicity, we're just using the prompt text
	return prompt.Text
}
