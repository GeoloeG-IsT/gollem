package providers_test

import (
	"context"
	"io"
	"testing"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
)

// TestOpenAIProvider tests the OpenAI provider implementation
func TestOpenAIProvider(t *testing.T) {
	// Skip this test in CI environments or when API key is not available
	t.Skip("Skipping OpenAI provider test as it requires API credentials")

	// Create a provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: "test-api-key",
		Model:  "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Check the name
	if provider.Name() != "openai" {
		t.Fatalf("Provider name is incorrect: %s", provider.Name())
	}

	// Test error cases
	_, err = openai.NewProvider(openai.Config{
		APIKey: "",
	})
	if err == nil {
		t.Fatal("No error when creating provider with empty API key")
	}
}

// TestMockProvider tests a mock provider for testing purposes
func TestMockProvider(t *testing.T) {
	// Create a mock provider
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Check the name
	if provider.Name() != "mock_provider" {
		t.Fatalf("Provider name is incorrect: %s", provider.Name())
	}

	// Generate a response
	ctx := context.Background()
	prompt := core.NewPrompt("Test prompt")
	response, err := provider.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate response: %v", err)
	}

	// Check the response
	if response.Text != "Mock response for: Test prompt" {
		t.Fatalf("Response text is incorrect: %s", response.Text)
	}
	if response.TokensUsed.Total != 20 {
		t.Fatalf("Response tokens used is incorrect: %d", response.TokensUsed.Total)
	}
	if response.FinishReason != "stop" {
		t.Fatalf("Response finish reason is incorrect: %s", response.FinishReason)
	}
	if response.ModelInfo.Name != "mock-model" {
		t.Fatalf("Response model info name is incorrect: %s", response.ModelInfo.Name)
	}
	if response.ModelInfo.Provider != "mock_provider" {
		t.Fatalf("Response model info provider is incorrect: %s", response.ModelInfo.Provider)
	}
	if response.ProviderInfo.Name != "mock_provider" {
		t.Fatalf("Response provider info name is incorrect: %s", response.ProviderInfo.Name)
	}
	if response.ProviderInfo.Version != "1.0.0" {
		t.Fatalf("Response provider info version is incorrect: %s", response.ProviderInfo.Version)
	}

	// Generate a streaming response
	stream, err := provider.GenerateStream(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to generate stream: %v", err)
	}

	// Read the stream
	var streamText string
	var chunkCount int
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}
		streamText += chunk.Text
		chunkCount++

		// Check the last chunk
		if chunk.IsFinal {
			if chunk.FinishReason != "stop" {
				t.Fatalf("Final chunk finish reason is incorrect: %s", chunk.FinishReason)
			}
		}
	}

	// Check the stream
	if streamText != "Mock response for: Test prompt" {
		t.Fatalf("Stream text is incorrect: %s", streamText)
	}
	if chunkCount != 4 {
		t.Fatalf("Stream chunk count is incorrect: %d", chunkCount)
	}

	// Close the stream
	err = stream.Close()
	if err != nil {
		t.Fatalf("Failed to close stream: %v", err)
	}
}

// MockProvider is a mock implementation of the LLMProvider interface
type MockProvider struct {
	name string
}

// Name returns the name of the provider
func (p *MockProvider) Name() string {
	return p.name
}

// Generate generates a response for the given prompt
func (p *MockProvider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
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
