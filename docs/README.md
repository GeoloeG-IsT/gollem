# Gollem - Go Large Language Model Interface

[![Go Reference](https://pkg.go.dev/badge/github.com/GeoloeG-IsT/gollem.svg)](https://pkg.go.dev/github.com/GeoloeG-IsT/gollem)
[![Go Report Card](https://goreportcard.com/badge/github.com/GeoloeG-IsT/gollem)](https://goreportcard.com/report/github.com/GeoloeG-IsT/gollem)
[![License](https://img.shields.io/github/license/GeoloeG-IsT/gollem)](https://github.com/GeoloeG-IsT/gollem/blob/main/LICENSE)

Gollem is a comprehensive Go package that provides a high-level interface for interacting with Large Language Models (LLMs). It supports multiple LLM providers, offers advanced features like prompt optimization and caching, and includes components for building RAG (Retrieval Augmented Generation) applications.

## Features

- **Multiple Provider Support**: Integrations with OpenAI, Anthropic, Google, Llama, Mistral, and more
- **Custom Provider Support**: Define your own provider implementations in a configured folder
- **JSON Configuration**: Configure all necessary parameters via a simple JSON file
- **Advanced Features**:
  - Prompt optimization
  - Response caching
  - Structured output handling with JSON schema validation
  - Streaming responses
- **RAG Architecture**: Complete set of components for building RAG applications
- **Tracing**: Comprehensive tracing capabilities compatible with Arize Phoenix

## Installation

```bash
go get github.com/GeoloeG-IsT/gollem
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/GeoloeG-IsT/gollem/pkg/config"
	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

func main() {
	// Load configuration
	manager, err := config.NewConfigManager("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get the default provider
	name, providerConfig, err := manager.GetDefaultProvider()
	if err != nil {
		log.Fatalf("Failed to get default provider: %v", err)
	}

	// Create a registry
	registry := core.NewRegistry()

	// Create a provider
	provider, err := registry.CreateProvider(providerConfig.Type, providerConfig)
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
	fmt.Println(response.Text)
}
```

## Configuration

Gollem can be configured via a JSON file. Here's an example configuration:

```json
{
  "default_provider": "openai",
  "providers": {
    "openai": {
      "type": "openai",
      "api_key": "your-api-key",
      "model": "gpt-4"
    },
    "anthropic": {
      "type": "anthropic",
      "api_key": "your-api-key",
      "model": "claude-2"
    }
  },
  "cache": {
    "enabled": true,
    "ttl": 3600,
    "max_entries": 1000
  },
  "rag": {
    "enabled": true,
    "embedding_provider": "openai",
    "embedding_model": "text-embedding-ada-002",
    "chunk_size": 1000,
    "chunk_overlap": 200
  },
  "tracing": {
    "enabled": true,
    "type": "console"
  },
  "custom_provider_paths": [
    "/path/to/custom/providers"
  ]
}
```

Environment variables can be used to override configuration values:

- `GOLLEM_DEFAULT_PROVIDER`: Override the default provider
- `GOLLEM_<PROVIDER>_API_KEY`: Override the API key for a provider
- `GOLLEM_<PROVIDER>_MODEL`: Override the model for a provider
- `GOLLEM_CACHE_ENABLED`: Enable or disable caching
- `GOLLEM_RAG_ENABLED`: Enable or disable RAG
- `GOLLEM_TRACING_ENABLED`: Enable or disable tracing

## Documentation

For more detailed documentation, see the following:

- [API Documentation](./docs/api.md)
- [Provider Documentation](./docs/providers.md)
- [RAG Documentation](./docs/rag.md)
- [Tracing Documentation](./docs/tracing.md)
- [Examples](./examples)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
