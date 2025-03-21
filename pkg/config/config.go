package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/gollem/pkg/core"
)

// Config represents the configuration for the gollem package
type Config struct {
	// DefaultProvider is the name of the default provider to use
	DefaultProvider string `json:"default_provider"`
	
	// Providers is a map of provider configurations
	Providers map[string]ProviderConfig `json:"providers"`
	
	// Cache configuration
	Cache CacheConfig `json:"cache"`
	
	// RAG configuration
	RAG RAGConfig `json:"rag"`
	
	// Tracing configuration
	Tracing TracingConfig `json:"tracing"`
	
	// CustomProviderPaths is a list of paths to search for custom providers
	CustomProviderPaths []string `json:"custom_provider_paths"`
}

// ProviderConfig represents the configuration for an LLM provider
type ProviderConfig struct {
	// Type is the type of provider (e.g., "openai", "anthropic", etc.)
	Type string `json:"type"`
	
	// APIKey is the API key for the provider
	APIKey string `json:"api_key"`
	
	// Model is the model to use
	Model string `json:"model"`
	
	// Endpoint is the API endpoint to use (optional)
	Endpoint string `json:"endpoint,omitempty"`
	
	// Parameters contains additional provider-specific parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// CacheConfig represents the configuration for caching
type CacheConfig struct {
	// Enabled indicates whether caching is enabled
	Enabled bool `json:"enabled"`
	
	// Type is the type of cache to use (e.g., "memory", "redis", etc.)
	Type string `json:"type"`
	
	// TTL is the time-to-live for cache entries in seconds
	TTL int `json:"ttl"`
	
	// MaxEntries is the maximum number of entries in the cache
	MaxEntries int `json:"max_entries"`
	
	// Parameters contains additional cache-specific parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// RAGConfig represents the configuration for RAG (Retrieval Augmented Generation)
type RAGConfig struct {
	// Enabled indicates whether RAG is enabled
	Enabled bool `json:"enabled"`
	
	// VectorStore is the type of vector store to use
	VectorStore string `json:"vector_store"`
	
	// Embeddings is the type of embeddings to use
	Embeddings string `json:"embeddings"`
	
	// ChunkSize is the size of chunks for document splitting
	ChunkSize int `json:"chunk_size"`
	
	// ChunkOverlap is the overlap between chunks
	ChunkOverlap int `json:"chunk_overlap"`
	
	// Parameters contains additional RAG-specific parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// TracingConfig represents the configuration for tracing
type TracingConfig struct {
	// Enabled indicates whether tracing is enabled
	Enabled bool `json:"enabled"`
	
	// Type is the type of tracing to use
	Type string `json:"type"`
	
	// Endpoint is the endpoint for the tracing service
	Endpoint string `json:"endpoint,omitempty"`
	
	// SampleRate is the rate at which to sample traces
	SampleRate float64 `json:"sample_rate"`
	
	// Parameters contains additional tracing-specific parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return &config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// CreateRegistry creates a provider registry from the configuration
func (c *Config) CreateRegistry() (*core.Registry, error) {
	registry := core.NewRegistry()
	
	// Register built-in provider factories
	registerBuiltInProviderFactories(registry)
	
	// Register custom provider factories
	if err := c.registerCustomProviderFactories(registry); err != nil {
		return nil, err
	}
	
	// Create providers from the configuration
	for name, providerConfig := range c.Providers {
		config := map[string]interface{}{
			"api_key":  providerConfig.APIKey,
			"model":    providerConfig.Model,
			"endpoint": providerConfig.Endpoint,
		}
		
		// Add additional parameters
		for k, v := range providerConfig.Parameters {
			config[k] = v
		}
		
		_, err := registry.CreateProvider(providerConfig.Type, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
		}
	}
	
	return registry, nil
}

// registerBuiltInProviderFactories registers the built-in provider factories
func registerBuiltInProviderFactories(registry *core.Registry) {
	// This would be implemented in a real package to register all the built-in providers
	// For example:
	// registry.RegisterFactory("openai", openai.NewProvider)
	// registry.RegisterFactory("anthropic", anthropic.NewProvider)
	// etc.
}

// registerCustomProviderFactories registers custom provider factories
func (c *Config) registerCustomProviderFactories(registry *core.Registry) error {
	for _, path := range c.CustomProviderPaths {
		// Check if the path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		
		// Walk the directory and look for Go files
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip directories
			if info.IsDir() {
				return nil
			}
			
			// Skip non-Go files
			if !strings.HasSuffix(filePath, ".go") {
				return nil
			}
			
			// In a real implementation, this would dynamically load the Go file
			// and register any provider factories it contains
			// This is a complex operation that would require reflection or code generation
			
			return nil
		})
		
		if err != nil {
			return fmt.Errorf("failed to walk custom provider path %s: %w", path, err)
		}
	}
	
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:   "openai",
				APIKey: "",
				Model:  "gpt-4",
			},
		},
		Cache: CacheConfig{
			Enabled:    true,
			Type:       "memory",
			TTL:        3600,
			MaxEntries: 1000,
		},
		RAG: RAGConfig{
			Enabled:      false,
			VectorStore:  "memory",
			Embeddings:   "openai",
			ChunkSize:    1000,
			ChunkOverlap: 200,
		},
		Tracing: TracingConfig{
			Enabled:    false,
			Type:       "console",
			SampleRate: 1.0,
		},
		CustomProviderPaths: []string{
			"./custom_providers",
		},
	}
}
