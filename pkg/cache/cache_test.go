package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/GeoloeG-IsT/gollem/pkg/cache"
	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// TestMemoryCache tests the memory cache implementation
func TestMemoryCache(t *testing.T) {
	// Create a cache
	memCache := cache.NewMemoryCache(
		cache.WithTTL(time.Second),
		cache.WithMaxEntries(10),
	)

	// Create a prompt and response
	prompt := core.NewPrompt("Test prompt")
	response := &core.Response{
		Text: "Test response",
	}

	// Set the response in the cache
	ctx := context.Background()
	err := memCache.Set(ctx, prompt, response)
	if err != nil {
		t.Fatalf("Failed to set response in cache: %v", err)
	}

	// Get the response from the cache
	cachedResponse, found := memCache.Get(ctx, prompt)
	if !found {
		t.Fatal("Response not found in cache")
	}

	// Check if it's the same response
	if cachedResponse.Text != response.Text {
		t.Fatalf("Cached response text is incorrect: %s", cachedResponse.Text)
	}

	// Invalidate the response
	err = memCache.Invalidate(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to invalidate response: %v", err)
	}

	// Try to get the response again
	_, found = memCache.Get(ctx, prompt)
	if found {
		t.Fatal("Response found in cache after invalidation")
	}

	// Set multiple responses
	for i := 0; i < 5; i++ {
		p := core.NewPrompt(fmt.Sprintf("Prompt %d", i))
		r := &core.Response{
			Text: fmt.Sprintf("Response %d", i),
		}
		err := memCache.Set(ctx, p, r)
		if err != nil {
			t.Fatalf("Failed to set response %d in cache: %v", i, err)
		}
	}

	// Clear the cache
	err = memCache.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Try to get a response
	_, found = memCache.Get(ctx, prompt)
	if found {
		t.Fatal("Response found in cache after clearing")
	}

	// Test TTL expiration
	err = memCache.Set(ctx, prompt, response)
	if err != nil {
		t.Fatalf("Failed to set response in cache: %v", err)
	}

	// Wait for the TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the response
	_, found = memCache.Get(ctx, prompt)
	if found {
		t.Fatal("Response found in cache after TTL expiration")
	}
}

// TestCacheMiddleware tests the cache middleware
func TestCacheMiddleware(t *testing.T) {
	// Create a mock provider
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Create a cache
	memCache := cache.NewMemoryCache()

	// Create a cache middleware
	middleware := cache.NewCacheMiddleware(provider, memCache)

	// Check the name
	if middleware.Name() != "mock_provider_cached" {
		t.Fatalf("Middleware name is incorrect: %s", middleware.Name())
	}

	// Generate a response
	ctx := context.Background()
	prompt := core.NewPrompt("Test prompt")
	response, err := middleware.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	// Check the response
	if response.Text != "Mock response for: Test prompt" {
		t.Fatalf("Response text is incorrect: %s", response.Text)
	}

	// Generate the same response again (should be cached)
	provider.callCount = 0
	response, err = middleware.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	// Check the response
	if response.Text != "Mock response for: Test prompt" {
		t.Fatalf("Response text is incorrect: %s", response.Text)
	}

	// Check that the provider wasn't called
	if provider.callCount != 0 {
		t.Fatalf("Provider was called %d times, expected 0", provider.callCount)
	}

	// Generate a streaming response (should not be cached)
	stream, err := middleware.GenerateStream(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate stream: %v", err)
	}

	// Read the stream
	var streamText string
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}
		streamText += chunk.Text
	}

	// Check the stream text
	if streamText != "Mock response for: Test prompt" {
		t.Fatalf("Stream text is incorrect: %s", streamText)
	}

	// Check that the provider was called
	if provider.callCount != 1 {
		t.Fatalf("Provider was called %d times, expected 1", provider.callCount)
	}
}

// MockProvider is a mock implementation of the LLMProvider interface
type MockProvider struct {
	name      string
	callCount int
}

// Name returns the name of the provider
func (p *MockProvider) Name() string {
	return p.name
}

// Generate generates a response for the given prompt
func (p *MockProvider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	p.callCount++
	return &core.Response{
		Text: "Mock response for: " + prompt.Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     10,
			Completion: 10,
			Total:      20,
		},
		FinishReason: "stop",
		ModelInfo: &core.ModelInfo{
			Name:     "mock-model",
			Provider: p.name,
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    p.name,
			Version: "1.0.0",
		},
	}, nil
}

// GenerateStream generates a streaming response for the given prompt
func (p *MockProvider) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	p.callCount++
	return &MockResponseStream{
		chunks: []string{
			"Mock ",
			"response ",
			"for: ",
			prompt.Text,
		},
	}, nil
}

// MockResponseStream is a mock implementation of the ResponseStream interface
type MockResponseStream struct {
	chunks []string
	index  int
}

// Next returns the next chunk of the response
func (s *MockResponseStream) Next() (*core.ResponseChunk, error) {
	if s.index >= len(s.chunks) {
		return nil, io.EOF
	}

	chunk := &core.ResponseChunk{
		Text:    s.chunks[s.index],
		IsFinal: s.index == len(s.chunks)-1,
	}

	if chunk.IsFinal {
		chunk.FinishReason = "stop"
	}

	s.index++
	return chunk, nil
}

// Close closes the stream
func (s *MockResponseStream) Close() error {
	return nil
}
