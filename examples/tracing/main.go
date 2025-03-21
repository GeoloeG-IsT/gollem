package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
	"github.com/GeoloeG-IsT/gollem/pkg/tracing"
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

	// Create a console tracer
	consoleTracer := tracing.NewConsoleTracer()

	// Create a file tracer
	fileTracer, err := tracing.NewFileTracer("trace.log")
	if err != nil {
		log.Fatalf("Failed to create file tracer: %v", err)
	}
	defer fileTracer.Close()

	// Wrap the provider with console tracing
	tracedProvider := tracing.NewLLMTracer(provider, consoleTracer)

	// Create a prompt
	prompt := core.NewPrompt("What is the capital of France?")

	// Create a context
	ctx := context.Background()

	// Start a custom span
	ctx, _ = consoleTracer.StartSpan(ctx, "example_operation")

	// Add attributes to the span
	consoleTracer.SetAttribute(ctx, "example_attribute", "example_value")

	// Add an event to the span
	consoleTracer.AddEvent(ctx, "example_event", map[string]interface{}{
		"event_key": "event_value",
	})

	// Generate a response
	fmt.Println("Generating response...")
	response, err := tracedProvider.Generate(ctx, prompt)
	if err != nil {
		consoleTracer.EndSpan(ctx, tracing.SpanStatusError)
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)

	// End the custom span
	consoleTracer.EndSpan(ctx, tracing.SpanStatusOK)

	// Generate a streaming response
	fmt.Println("\nGenerating streaming response...")
	stream, err := tracedProvider.GenerateStream(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate stream: %v", err)
	}

	// Read the stream
	fmt.Println("Streaming response:")
	var streamText string
	for {
		chunk, err := stream.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Fatalf("Failed to read stream: %v", err)
		}
		fmt.Print(chunk.Text)
		streamText += chunk.Text
		if chunk.IsFinal {
			fmt.Printf("\nFinish reason: %s\n", chunk.FinishReason)
		}
	}

	// Close the stream
	stream.Close()

	// Flush the tracers
	consoleTracer.Flush()
	fileTracer.Flush()

	fmt.Println("\nTracing information has been logged to the console and trace.log file.")
}
