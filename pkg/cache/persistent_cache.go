package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// PersistentCache is a file-based implementation of Cache
type PersistentCache struct {
	directory  string
	ttl        time.Duration
	maxEntries int
	hashFunc   func(*core.Prompt) string
	mu         sync.RWMutex
}

// cacheMetadata contains metadata about the cache
type cacheMetadata struct {
	Entries    map[string]time.Time `json:"entries"`
	LastPurged time.Time            `json:"last_purged"`
}

// NewPersistentCache creates a new persistent cache
func NewPersistentCache(options ...PersistentCacheOption) (*PersistentCache, error) {
	cache := &PersistentCache{
		directory:  filepath.Join(os.TempDir(), "gollem-cache"),
		ttl:        time.Hour,
		maxEntries: 1000,
		hashFunc:   defaultHashFunc,
	}
	
	for _, option := range options {
		option(cache)
	}
	
	// Create the cache directory if it doesn't exist
	if err := os.MkdirAll(cache.directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	// Load the metadata
	if err := cache.loadMetadata(); err != nil {
		// If the metadata file doesn't exist, create it
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load metadata: %w", err)
		}
		
		if err := cache.saveMetadata(&cacheMetadata{
			Entries:    make(map[string]time.Time),
			LastPurged: time.Now(),
		}); err != nil {
			return nil, fmt.Errorf("failed to save metadata: %w", err)
		}
	}
	
	return cache, nil
}

// PersistentCacheOption is a function that configures a PersistentCache
type PersistentCacheOption func(*PersistentCache)

// WithDirectory sets the directory for the cache
func WithDirectory(directory string) PersistentCacheOption {
	return func(c *PersistentCache) {
		c.directory = directory
	}
}

// WithPersistentTTL sets the time-to-live for cache entries
func WithPersistentTTL(ttl time.Duration) PersistentCacheOption {
	return func(c *PersistentCache) {
		c.ttl = ttl
	}
}

// WithPersistentMaxEntries sets the maximum number of entries in the cache
func WithPersistentMaxEntries(max int) PersistentCacheOption {
	return func(c *PersistentCache) {
		c.maxEntries = max
	}
}

// WithPersistentHashFunc sets the function used to hash prompts
func WithPersistentHashFunc(hashFunc func(*core.Prompt) string) PersistentCacheOption {
	return func(c *PersistentCache) {
		c.hashFunc = hashFunc
	}
}

// Get retrieves a cached response for a prompt
func (c *PersistentCache) Get(ctx context.Context, prompt *core.Prompt) (*core.Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	key := c.hashFunc(prompt)
	
	// Load the metadata
	metadata, err := c.loadMetadata()
	if err != nil {
		return nil, false
	}
	
	// Check if the entry exists and hasn't expired
	timestamp, exists := metadata.Entries[key]
	if !exists || time.Since(timestamp) > c.ttl {
		return nil, false
	}
	
	// Load the response from the file
	response, err := c.loadResponse(key)
	if err != nil {
		return nil, false
	}
	
	return response, true
}

// Set stores a response for a prompt
func (c *PersistentCache) Set(ctx context.Context, prompt *core.Prompt, response *core.Response) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.hashFunc(prompt)
	
	// Load the metadata
	metadata, err := c.loadMetadata()
	if err != nil {
		metadata = &cacheMetadata{
			Entries:    make(map[string]time.Time),
			LastPurged: time.Now(),
		}
	}
	
	// Check if we need to purge old entries
	if len(metadata.Entries) >= c.maxEntries || time.Since(metadata.LastPurged) > 24*time.Hour {
		c.purgeOldEntries(metadata)
	}
	
	// Save the response to a file
	if err := c.saveResponse(key, response); err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}
	
	// Update the metadata
	metadata.Entries[key] = time.Now()
	
	// Save the metadata
	if err := c.saveMetadata(metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	
	return nil
}

// Invalidate removes a cached response for a prompt
func (c *PersistentCache) Invalidate(ctx context.Context, prompt *core.Prompt) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	key := c.hashFunc(prompt)
	
	// Load the metadata
	metadata, err := c.loadMetadata()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}
	
	// Check if the entry exists
	if _, exists := metadata.Entries[key]; !exists {
		return nil
	}
	
	// Delete the response file
	responsePath := filepath.Join(c.directory, key+".json")
	if err := os.Remove(responsePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete response file: %w", err)
	}
	
	// Update the metadata
	delete(metadata.Entries, key)
	
	// Save the metadata
	if err := c.saveMetadata(metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	
	return nil
}

// Clear removes all cached responses
func (c *PersistentCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Remove all files in the cache directory
	files, err := ioutil.ReadDir(c.directory)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}
	
	for _, file := range files {
		if err := os.Remove(filepath.Join(c.directory, file.Name())); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	
	// Create a new metadata file
	if err := c.saveMetadata(&cacheMetadata{
		Entries:    make(map[string]time.Time),
		LastPurged: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	
	return nil
}

// loadMetadata loads the cache metadata
func (c *PersistentCache) loadMetadata() (*cacheMetadata, error) {
	metadataPath := filepath.Join(c.directory, "metadata.json")
	
	data, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}
	
	var metadata cacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	
	return &metadata, nil
}

// saveMetadata saves the cache metadata
func (c *PersistentCache) saveMetadata(metadata *cacheMetadata) error {
	metadataPath := filepath.Join(c.directory, "metadata.json")
	
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := ioutil.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	
	return nil
}

// loadResponse loads a response from a file
func (c *PersistentCache) loadResponse(key string) (*core.Response, error) {
	responsePath := filepath.Join(c.directory, key+".json")
	
	data, err := ioutil.ReadFile(responsePath)
	if err != nil {
		return nil, err
	}
	
	var response core.Response
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return &response, nil
}

// saveResponse saves a response to a file
func (c *PersistentCache) saveResponse(key string, response *core.Response) error {
	responsePath := filepath.Join(c.directory, key+".json")
	
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	
	if err := ioutil.WriteFile(responsePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	
	return nil
}

// purgeOldEntries removes old entries from the cache
func (c *PersistentCache) purgeOldEntries(metadata *cacheMetadata) {
	// Find expired entries
	var expiredKeys []string
	now := time.Now()
	
	for key, timestamp := range metadata.Entries {
		if now.Sub(timestamp) > c.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// Delete expired entries
	for _, key := range expiredKeys {
		delete(metadata.Entries, key)
		
		// Delete the response file
		responsePath := filepath.Join(c.directory, key+".json")
		os.Remove(responsePath) // Ignore errors
	}
	
	// If we still have too many entries, delete the oldest ones
	if len(metadata.Entries) > c.maxEntries {
		// Find the oldest entries
		type keyTimestamp struct {
			key       string
			timestamp time.Time
		}
		
		entries := make([]keyTimestamp, 0, len(metadata.Entries))
		for key, timestamp := range metadata.Entries {
			entries = append(entries, keyTimestamp{key, timestamp})
		}
		
		// Sort by timestamp (oldest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].timestamp.Before(entries[j].timestamp)
		})
		
		// Delete the oldest entries
		for i := 0; i < len(entries)-(c.maxEntries/2); i++ {
			key := entries[i].key
			delete(metadata.Entries, key)
			
			// Delete the response file
			responsePath := filepath.Join(c.directory, key+".json")
			os.Remove(responsePath) // Ignore errors
		}
	}
	
	// Update the last purged timestamp
	metadata.LastPurged = now
}

// CacheMiddleware is middleware that adds caching to an LLM provider
type CacheMiddleware struct {
	provider core.LLMProvider
	cache    Cache
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(provider core.LLMProvider, cache Cache) *CacheMiddleware {
	return &CacheMiddleware{
		provider: provider,
		cache:    cache,
	}
}

// Name returns the name of the provider
func (m *CacheMiddleware) Name() string {
	return m.provider.Name() + "_cached"
}

// Generate generates a response for the given prompt
func (m *CacheMiddleware) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Check if the response is cached
	if response, found := m.cache.Get(ctx, prompt); found {
		return response, nil
	}
	
	// Generate a response
	response, err := m.provider.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	// Cache the response
	if err := m.cache.Set(ctx, prompt, response); err != nil {
		// Log the error but don't fail the request
		fmt.Printf("Failed to cache response: %v\n", err)
	}
	
	return response, nil
}

// GenerateStream generates a streaming response for the given prompt
func (m *CacheMiddleware) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	// Streaming responses are not cached
	return m.provider.GenerateStream(ctx, prompt)
}
