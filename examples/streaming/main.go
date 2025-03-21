package main

import (
	"context"
	"fmt"
	"io"
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
	prompt := core.NewPrompt("Write a short story about a robot learning to feel emotions.")

	// Generate a streaming response
	ctx := context.Background()
	stream, err := provider.GenerateStream(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate streaming response: %v", err)
	}
	defer stream.Close()

	// Process the streaming response
	fmt.Println("Streaming response:")
	fmt.Println("-------------------")
	
	var fullText string
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading from stream: %v", err)
		}

		// Print the chunk as it arrives
		fmt.Print(chunk.Text)
		fullText += chunk.Text
		
		// If this is the final chunk, print finish information
		if chunk.IsFinal {
			fmt.Printf("\nFinish reason: %s\n", chunk.FinishReason)
		}
	}
	
	fmt.Println("\n-------------------")
	fmt.Printf("Total response length: %d characters\n", len(fullText))
}
