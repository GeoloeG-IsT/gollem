package cache

import (
	"context"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// CacheMiddleware is a middleware that caches responses from an LLM provider
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

// Generate generates a response for a prompt, using the cache if available
func (m *CacheMiddleware) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	// Check if the response is in the cache
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
		// In a real implementation, this would use a proper logger
		// fmt.Printf("Failed to cache response: %v\n", err)
	}

	return response, nil
}

// GenerateStream generates a streaming response for a prompt
func (m *CacheMiddleware) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	// Streaming responses are not cached
	return m.provider.GenerateStream(ctx, prompt)
}
