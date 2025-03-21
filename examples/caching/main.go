package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/GeoloeG-IsT/gollem/pkg/cache"
	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not found in environment, using dummy key")
		apiKey = "dummy_openai_api_key_12345"
	}

	// Create a provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: apiKey,
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
