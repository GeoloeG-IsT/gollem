package main

import (
	"context"
	"fmt"
	"log"

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
