# API Documentation

This document provides detailed information about the Gollem API.

## Core Package

The core package contains the fundamental interfaces and types used throughout Gollem.

### Interfaces

#### LLMProvider

The `LLMProvider` interface defines the methods that all LLM providers must implement.

```go
type LLMProvider interface {
    // Name returns the name of the provider
    Name() string
    
    // Generate generates a response for the given prompt
    Generate(ctx context.Context, prompt *Prompt) (*Response, error)
    
    // GenerateStream generates a streaming response for the given prompt
    GenerateStream(ctx context.Context, prompt *Prompt) (ResponseStream, error)
}
```

#### ResponseStream

The `ResponseStream` interface defines methods for handling streaming responses.

```go
type ResponseStream interface {
    // Next returns the next chunk of the response
    Next() (*ResponseChunk, error)
    
    // Close closes the stream
    Close() error
}
```

#### Cache

The `Cache` interface defines methods for caching responses.

```go
type Cache interface {
    // Get retrieves a cached response for a prompt
    Get(ctx context.Context, prompt *Prompt) (*Response, bool)
    
    // Set stores a response for a prompt
    Set(ctx context.Context, prompt *Prompt, response *Response) error
    
    // Invalidate removes a cached response for a prompt
    Invalidate(ctx context.Context, prompt *Prompt) error
    
    // Clear removes all cached responses
    Clear(ctx context.Context) error
}
```

### Types

#### Prompt

The `Prompt` type represents a prompt to be sent to an LLM.

```go
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
```

#### Response

The `Response` type represents a response from an LLM.

```go
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
```

#### ResponseChunk

The `ResponseChunk` type represents a chunk of a streaming response.

```go
type ResponseChunk struct {
    // Text is the chunk text
    Text string
    
    // IsFinal indicates if this is the final chunk
    IsFinal bool
    
    // FinishReason indicates why generation stopped (only set if IsFinal is true)
    FinishReason string
}
```

#### TokenUsage

The `TokenUsage` type contains information about token usage.

```go
type TokenUsage struct {
    // Prompt is the number of tokens in the prompt
    Prompt int
    
    // Completion is the number of tokens in the completion
    Completion int
    
    // Total is the total number of tokens
    Total int
}
```

#### ModelInfo

The `ModelInfo` type contains information about the model.

```go
type ModelInfo struct {
    // Name is the name of the model
    Name string
    
    // Provider is the name of the provider
    Provider string
    
    // Version is the version of the model
    Version string
}
```

#### ProviderInfo

The `ProviderInfo` type contains information about the provider.

```go
type ProviderInfo struct {
    // Name is the name of the provider
    Name string
    
    // Version is the version of the provider
    Version string
}
```

## Registry

The `Registry` is used to register and retrieve LLM providers.

```go
// NewRegistry creates a new registry
registry := core.NewRegistry()

// Register a provider
registry.RegisterProvider(provider)

// Get a provider
provider, exists := registry.GetProvider("openai")

// Register a factory
registry.RegisterFactory("openai", func(config map[string]interface{}) (core.LLMProvider, error) {
    return openai.NewProvider(config)
})

// Create a provider using a factory
provider, err := registry.CreateProvider("openai", config)
```

## Configuration

The configuration package provides utilities for loading and managing configuration.

```go
// Load configuration from a file
config, err := config.LoadConfig("config.json")

// Load configuration with environment variable overrides
config, err := config.LoadConfigWithEnv("config.json")

// Create a configuration manager
manager, err := config.NewConfigManager("config.json")

// Get the default provider
name, providerConfig, err := manager.GetDefaultProvider()

// Update a provider configuration
err := manager.UpdateProvider("openai", config.ProviderConfig{
    Type:   "openai",
    APIKey: "your-api-key",
    Model:  "gpt-4",
})

// Enable or disable features
err := manager.EnableCache(true)
err := manager.EnableRAG(true)
err := manager.EnableTracing(true)
```

## Error Handling

Gollem uses standard Go error handling. Most functions return an error as their last return value, which should be checked.

```go
response, err := provider.Generate(ctx, prompt)
if err != nil {
    // Handle error
    log.Fatalf("Failed to generate response: %v", err)
}
```

Common errors include:
- API authentication errors
- Rate limiting errors
- Context timeout errors
- Invalid configuration errors
- Network errors

## Context Usage

Gollem uses Go's context package for cancellation, timeouts, and passing request-scoped values.

```go
// Create a context with a timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Generate a response with the context
response, err := provider.Generate(ctx, prompt)
```

## Thread Safety

All Gollem components are designed to be thread-safe and can be used concurrently from multiple goroutines.
