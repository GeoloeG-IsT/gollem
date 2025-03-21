package core

import (
	"context"
)

// LLMProvider defines the interface for all LLM providers
type LLMProvider interface {
	// Name returns the name of the provider
	Name() string
	
	// Generate generates a response for the given prompt
	Generate(ctx context.Context, prompt *Prompt) (*Response, error)
	
	// GenerateStream generates a streaming response for the given prompt
	GenerateStream(ctx context.Context, prompt *Prompt) (ResponseStream, error)
}

// ResponseStream defines the interface for streaming responses
type ResponseStream interface {
	// Next returns the next chunk of the response
	Next() (*ResponseChunk, error)
	
	// Close closes the stream
	Close() error
}

// Prompt represents a prompt to be sent to an LLM
type Prompt struct {
	// Text is the main prompt text
	Text string
	
	// SystemMessage is an optional system message
	SystemMessage string
	
	// Temperature controls randomness (0.0-1.0)
	Temperature float64
	
	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int
	
	// TopP controls diversity via nucleus sampling (0.0-1.0)
	TopP float64
	
	// FrequencyPenalty reduces repetition (0.0-2.0)
	FrequencyPenalty float64
	
	// PresencePenalty encourages new topics (0.0-2.0)
	PresencePenalty float64
	
	// StopSequences are sequences that stop generation
	StopSequences []string
	
	// Schema is an optional JSON schema for structured output
	Schema interface{}
	
	// AdditionalParams contains provider-specific parameters
	AdditionalParams map[string]interface{}
}

// NewPrompt creates a new prompt with default settings
func NewPrompt(text string) *Prompt {
	return &Prompt{
		Text:             text,
		Temperature:      0.7,
		MaxTokens:        1024,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		AdditionalParams: make(map[string]interface{}),
	}
}

// Response represents a response from an LLM
type Response struct {
	// Text is the response text
	Text string
	
	// StructuredOutput contains parsed structured output
	StructuredOutput interface{}
	
	// TokensUsed contains token usage information
	TokensUsed *TokenUsage
	
	// FinishReason indicates why generation stopped
	FinishReason string
	
	// ModelInfo contains information about the model
	ModelInfo *ModelInfo
	
	// ProviderInfo contains information about the provider
	ProviderInfo *ProviderInfo
}

// ResponseChunk represents a chunk of a streaming response
type ResponseChunk struct {
	// Text is the chunk text
	Text string
	
	// IsFinal indicates if this is the final chunk
	IsFinal bool
	
	// FinishReason indicates why generation stopped (only set if IsFinal is true)
	FinishReason string
}

// TokenUsage contains information about token usage
type TokenUsage struct {
	// Prompt is the number of tokens in the prompt
	Prompt int
	
	// Completion is the number of tokens in the completion
	Completion int
	
	// Total is the total number of tokens
	Total int
}

// ModelInfo contains information about the model
type ModelInfo struct {
	// Name is the name of the model
	Name string
	
	// Provider is the name of the provider
	Provider string
	
	// Version is the version of the model
	Version string
}

// ProviderInfo contains information about the provider
type ProviderInfo struct {
	// Name is the name of the provider
	Name string
	
	// Version is the version of the provider
	Version string
}
