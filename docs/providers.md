# Provider Documentation

This document provides detailed information about the LLM providers supported by Gollem.

## Supported Providers

Gollem supports the following LLM providers out of the box:

- OpenAI
- Anthropic
- Google
- Llama
- Mistral

Additionally, Gollem allows you to implement and load custom providers.

## OpenAI Provider

The OpenAI provider supports OpenAI's GPT models.

### Configuration

```json
{
  "providers": {
    "openai": {
      "type": "openai",
      "api_key": "your-api-key",
      "model": "gpt-4",
      "organization": "your-organization-id",  // Optional
      "base_url": "https://api.openai.com/v1", // Optional
      "timeout": 30                            // Optional, in seconds
    }
  }
}
```

### Environment Variables

- `GOLLEM_OPENAI_API_KEY`: API key
- `GOLLEM_OPENAI_MODEL`: Model name
- `GOLLEM_OPENAI_ORGANIZATION`: Organization ID
- `GOLLEM_OPENAI_BASE_URL`: Base URL
- `GOLLEM_OPENAI_TIMEOUT`: Timeout in seconds

### Supported Models

- `gpt-4`
- `gpt-4-turbo`
- `gpt-3.5-turbo`
- And other models supported by the OpenAI API

### Usage Example

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
)

provider, err := openai.NewProvider(openai.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4",
})
```

## Anthropic Provider

The Anthropic provider supports Anthropic's Claude models.

### Configuration

```json
{
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "your-api-key",
      "model": "claude-2",
      "base_url": "https://api.anthropic.com", // Optional
      "timeout": 30                            // Optional, in seconds
    }
  }
}
```

### Environment Variables

- `GOLLEM_ANTHROPIC_API_KEY`: API key
- `GOLLEM_ANTHROPIC_MODEL`: Model name
- `GOLLEM_ANTHROPIC_BASE_URL`: Base URL
- `GOLLEM_ANTHROPIC_TIMEOUT`: Timeout in seconds

### Supported Models

- `claude-2`
- `claude-instant-1`
- And other models supported by the Anthropic API

### Usage Example

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/providers/anthropic"
)

provider, err := anthropic.NewProvider(anthropic.Config{
    APIKey: "your-api-key",
    Model:  "claude-2",
})
```

## Google Provider

The Google provider supports Google's Gemini models.

### Configuration

```json
{
  "providers": {
    "google": {
      "type": "google",
      "api_key": "your-api-key",
      "model": "gemini-pro",
      "project_id": "your-project-id",       // Optional
      "location": "us-central1",             // Optional
      "timeout": 30                          // Optional, in seconds
    }
  }
}
```

### Environment Variables

- `GOLLEM_GOOGLE_API_KEY`: API key
- `GOLLEM_GOOGLE_MODEL`: Model name
- `GOLLEM_GOOGLE_PROJECT_ID`: Project ID
- `GOLLEM_GOOGLE_LOCATION`: Location
- `GOLLEM_GOOGLE_TIMEOUT`: Timeout in seconds

### Supported Models

- `gemini-pro`
- `gemini-ultra`
- And other models supported by the Google AI API

### Usage Example

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/providers/google"
)

provider, err := google.NewProvider(google.Config{
    APIKey: "your-api-key",
    Model:  "gemini-pro",
})
```

## Llama Provider

The Llama provider supports running Llama models locally or via API.

### Configuration

```json
{
  "providers": {
    "llama": {
      "type": "llama",
      "model_path": "/path/to/model.bin",    // For local models
      "api_url": "http://localhost:8080",     // For API access
      "context_size": 4096,                   // Optional
      "threads": 4                            // Optional
    }
  }
}
```

### Environment Variables

- `GOLLEM_LLAMA_MODEL_PATH`: Path to the model file
- `GOLLEM_LLAMA_API_URL`: API URL
- `GOLLEM_LLAMA_CONTEXT_SIZE`: Context size
- `GOLLEM_LLAMA_THREADS`: Number of threads

### Usage Example

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/providers/llama"
)

// For local models
provider, err := llama.NewProvider(llama.Config{
    ModelPath: "/path/to/model.bin",
    Threads:   4,
})

// For API access
provider, err := llama.NewProvider(llama.Config{
    APIURL: "http://localhost:8080",
})
```

## Mistral Provider

The Mistral provider supports Mistral AI's models.

### Configuration

```json
{
  "providers": {
    "mistral": {
      "type": "mistral",
      "api_key": "your-api-key",
      "model": "mistral-medium",
      "base_url": "https://api.mistral.ai", // Optional
      "timeout": 30                         // Optional, in seconds
    }
  }
}
```

### Environment Variables

- `GOLLEM_MISTRAL_API_KEY`: API key
- `GOLLEM_MISTRAL_MODEL`: Model name
- `GOLLEM_MISTRAL_BASE_URL`: Base URL
- `GOLLEM_MISTRAL_TIMEOUT`: Timeout in seconds

### Supported Models

- `mistral-tiny`
- `mistral-small`
- `mistral-medium`
- And other models supported by the Mistral AI API

### Usage Example

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/providers/mistral"
)

provider, err := mistral.NewProvider(mistral.Config{
    APIKey: "your-api-key",
    Model:  "mistral-medium",
})
```

## Custom Providers

Gollem allows you to implement and load custom providers.

### Implementing a Custom Provider

To implement a custom provider, create a Go package that implements the `core.LLMProvider` interface:

```go
package myprovider

import (
    "context"
    "github.com/GeoloeG-IsT/gollem/pkg/core"
)

type MyProvider struct {
    // Provider-specific fields
}

func NewProvider(config map[string]interface{}) (*MyProvider, error) {
    // Initialize the provider with the config
    return &MyProvider{}, nil
}

func (p *MyProvider) Name() string {
    return "myprovider"
}

func (p *MyProvider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
    // Implement generation logic
}

func (p *MyProvider) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
    // Implement streaming logic
}
```

### Loading Custom Providers

To load custom providers, specify the paths to the provider packages in the configuration:

```json
{
  "custom_provider_paths": [
    "/path/to/custom/providers"
  ]
}
```

The directory structure should be:

```
/path/to/custom/providers/
  ├── provider1/
  │   └── provider1.go
  └── provider2/
      └── provider2.go
```

Each provider package must export a `NewProvider` function that takes a `map[string]interface{}` configuration and returns a `core.LLMProvider` and an error.

### Using Custom Providers

Once loaded, custom providers can be used like built-in providers:

```go
// Get the provider from the registry
provider, exists := registry.GetProvider("myprovider")
if !exists {
    // Handle error
}

// Or create it with a configuration
provider, err := registry.CreateProvider("myprovider", config)
if err != nil {
    // Handle error
}
```

## Provider Middleware

Gollem supports middleware for providers, which can be used to add functionality like caching, tracing, or rate limiting.

### Caching Middleware

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/cache"
)

// Create a cache
memCache := cache.NewMemoryCache()

// Create a provider
provider, err := openai.NewProvider(config)
if err != nil {
    // Handle error
}

// Wrap the provider with caching
cachedProvider := cache.NewCacheMiddleware(provider, memCache)
```

### Tracing Middleware

```go
import (
    "github.com/GeoloeG-IsT/gollem/pkg/tracing"
)

// Create a tracer
tracer := tracing.NewConsoleTracer()

// Create a provider
provider, err := openai.NewProvider(config)
if err != nil {
    // Handle error
}

// Wrap the provider with tracing
tracedProvider := tracing.NewLLMTracer(provider, tracer)
```

## Provider Selection

Gollem allows you to select providers at runtime based on configuration:

```go
// Load configuration
manager, err := config.NewConfigManager("config.json")
if err != nil {
    // Handle error
}

// Get the default provider
name, providerConfig, err := manager.GetDefaultProvider()
if err != nil {
    // Handle error
}

// Create a registry
registry := core.NewRegistry()

// Create the provider
provider, err := registry.CreateProvider(providerConfig.Type, providerConfig)
if err != nil {
    // Handle error
}
```

This allows you to switch providers by changing the configuration without modifying your code.
