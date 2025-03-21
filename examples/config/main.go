package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/anthropic"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/google"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/llama"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/mistral"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
	"github.com/joho/godotenv"
)

// Config represents a simplified version of the configuration
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
	Cache           map[string]interface{}    `json:"cache"`
	RAG             map[string]interface{}    `json:"rag"`
	Tracing         map[string]interface{}    `json:"tracing"`
}

// ProviderConfig represents the configuration for an LLM provider
type ProviderConfig struct {
	Type       string                 `json:"type"`
	APIKey     string                 `json:"api_key"`
	Model      string                 `json:"model"`
	Endpoint   string                 `json:"endpoint,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

func main() {
	// Load .env file for API keys
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Get API keys from environment variables
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Println("Warning: OPENAI_API_KEY not found in environment, using dummy key")
		openaiAPIKey = "dummy_openai_api_key_12345"
	}

	mistralAPIKey := os.Getenv("MISTRAL_API_KEY")
	if mistralAPIKey == "" {
		log.Println("Warning: MISTRAL_API_KEY not found in environment, using dummy key")
		mistralAPIKey = "dummy_mistral_api_key_12345"
	}

	anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicAPIKey == "" {
		log.Println("Warning: ANTHROPIC_API_KEY not found in environment, using dummy key")
		anthropicAPIKey = "dummy_anthropic_api_key_12345"
	}

	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		log.Println("Warning: GOOGLE_API_KEY not found in environment, using dummy key")
		googleAPIKey = "dummy_google_api_key_12345"
	}

	llamaAPIKey := os.Getenv("LLAMA_API_KEY")
	if llamaAPIKey == "" {
		log.Println("Warning: LLAMA_API_KEY not found in environment, using dummy key")
		llamaAPIKey = "dummy_llama_api_key_12345"
	}

	// Load configuration from config.json
	configPath := filepath.Join(".", "config.json")
	fmt.Println("Loading configuration from:", configPath)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Update API keys in the configuration
	if provider, ok := cfg.Providers["openai"]; ok {
		provider.APIKey = openaiAPIKey
		cfg.Providers["openai"] = provider
	}

	if provider, ok := cfg.Providers["mistral"]; ok {
		provider.APIKey = mistralAPIKey
		cfg.Providers["mistral"] = provider
	}

	if provider, ok := cfg.Providers["anthropic"]; ok {
		provider.APIKey = anthropicAPIKey
		cfg.Providers["anthropic"] = provider
	}

	if provider, ok := cfg.Providers["google"]; ok {
		provider.APIKey = googleAPIKey
		cfg.Providers["google"] = provider
	}

	if provider, ok := cfg.Providers["llama"]; ok {
		provider.APIKey = llamaAPIKey
		cfg.Providers["llama"] = provider
	}

	// Print configuration details
	fmt.Println("Configuration loaded successfully!")
	fmt.Println("Default Provider:", cfg.DefaultProvider)
	fmt.Println("Available Providers:", getProviderNames(cfg.Providers))
	fmt.Println("Cache Enabled:", cfg.Cache["enabled"])
	fmt.Println("RAG Enabled:", cfg.RAG["enabled"])
	fmt.Println("Tracing Enabled:", cfg.Tracing["enabled"])

	// Get the default provider configuration
	providerConfig := cfg.Providers[cfg.DefaultProvider]
	
	// Create the appropriate provider based on the type
	var provider core.LLMProvider
	
	switch providerConfig.Type {
	case "openai":
		provider, err = openai.NewProvider(openai.Config{
			APIKey:   providerConfig.APIKey,
			Model:    providerConfig.Model,
			Endpoint: providerConfig.Endpoint,
		})
	case "mistral":
		provider, err = mistral.NewProvider(mistral.Config{
			APIKey:   providerConfig.APIKey,
			Model:    providerConfig.Model,
			Endpoint: providerConfig.Endpoint,
		})
	case "anthropic":
		provider, err = anthropic.NewProvider(anthropic.Config{
			APIKey:   providerConfig.APIKey,
			Model:    providerConfig.Model,
			Endpoint: providerConfig.Endpoint,
			Version:  providerConfig.Version,
		})
	case "google":
		provider, err = google.NewProvider(google.Config{
			APIKey:   providerConfig.APIKey,
			Model:    providerConfig.Model,
			Endpoint: providerConfig.Endpoint,
		})
	case "llama":
		provider, err = llama.NewProvider(llama.Config{
			APIKey:   providerConfig.APIKey,
			Model:    providerConfig.Model,
			Endpoint: providerConfig.Endpoint,
		})
	default:
		log.Fatalf("Unsupported provider type: %s", providerConfig.Type)
	}
	
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create a prompt
	prompt := core.NewPrompt("What is the capital of Japan?")

	// Create a context
	ctx := context.Background()

	// Generate a response
	fmt.Println("\nGenerating response using provider:", provider.Name())
	response, err := provider.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate response: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)
	fmt.Println("Finish reason:", response.FinishReason)
}

// Helper function to get provider names
func getProviderNames(providers map[string]ProviderConfig) []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
