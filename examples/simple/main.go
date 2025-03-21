package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// Create a prompt
	prompt := core.NewPrompt("What is the capital of France?")

	// Generate a response
	ctx := context.Background()
	response, err := provider.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)
	fmt.Println("Finish reason:", response.FinishReason)
}
