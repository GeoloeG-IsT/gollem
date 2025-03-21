package main

import (
	"context"
	"fmt"
	"log"

	"github.com/GeoloeG-IsT/gollem/pkg/cache"
	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
)

func main() {
	// Create a provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: "your-api-key", // Replace with your actual API key
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create a cache
	memCache := cache.NewMemoryCache(
		cache.WithTTL(3600),
		cache.WithMaxEntries(1000),
	)

	// Wrap the provider with caching
	cachedProvider := cache.NewCacheMiddleware(provider, memCache)

	// Create a prompt
	prompt := core.NewPrompt("What is the capital of France?")

	// Generate a response (this will be cached)
	ctx := context.Background()
	response, err := cachedProvider.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("First response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)

	// Generate the same response again (should be retrieved from cache)
	fmt.Println("\nGenerating the same response again (should be from cache)...")
	response, err = cachedProvider.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("Second response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)

	// Create a different prompt
	prompt2 := core.NewPrompt("What is the capital of Germany?")

	// Generate a response for the new prompt
	fmt.Println("\nGenerating response for a different prompt...")
	response, err = cachedProvider.Generate(ctx, prompt2)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)
}
