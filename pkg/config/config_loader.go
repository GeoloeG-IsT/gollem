package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/custom"
)

// EnvVarPrefix is the prefix for environment variables
const EnvVarPrefix = "GOLLEM_"

// LoadConfigWithEnv loads the configuration from a file and overrides values with environment variables
func LoadConfigWithEnv(path string) (*Config, error) {
	// Load the base configuration
	config, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	
	// Override with environment variables
	if err := overrideWithEnv(config); err != nil {
		return nil, fmt.Errorf("failed to override config with environment variables: %w", err)
	}
	
	return config, nil
}

// overrideWithEnv overrides configuration values with environment variables
func overrideWithEnv(config *Config) error {
	// Override default provider
	if val := os.Getenv(EnvVarPrefix + "DEFAULT_PROVIDER"); val != "" {
		config.DefaultProvider = val
	}
	
	// Override provider configurations
	for name, provider := range config.Providers {
		// API Key
		envKey := EnvVarPrefix + strings.ToUpper(name) + "_API_KEY"
		if val := os.Getenv(envKey); val != "" {
			provider.APIKey = val
			config.Providers[name] = provider
		}
		
		// Model
		envKey = EnvVarPrefix + strings.ToUpper(name) + "_MODEL"
		if val := os.Getenv(envKey); val != "" {
			provider.Model = val
			config.Providers[name] = provider
		}
		
		// Endpoint
		envKey = EnvVarPrefix + strings.ToUpper(name) + "_ENDPOINT"
		if val := os.Getenv(envKey); val != "" {
			provider.Endpoint = val
			config.Providers[name] = provider
		}
	}
	
	// Override cache configuration
	if val := os.Getenv(EnvVarPrefix + "CACHE_ENABLED"); val != "" {
		config.Cache.Enabled = val == "true" || val == "1" || val == "yes"
	}
	
	if val := os.Getenv(EnvVarPrefix + "CACHE_TYPE"); val != "" {
		config.Cache.Type = val
	}
	
	if val := os.Getenv(EnvVarPrefix + "CACHE_TTL"); val != "" {
		var ttl int
		if _, err := fmt.Sscanf(val, "%d", &ttl); err == nil {
			config.Cache.TTL = ttl
		}
	}
	
	// Override RAG configuration
	if val := os.Getenv(EnvVarPrefix + "RAG_ENABLED"); val != "" {
		config.RAG.Enabled = val == "true" || val == "1" || val == "yes"
	}
	
	if val := os.Getenv(EnvVarPrefix + "RAG_VECTOR_STORE"); val != "" {
		config.RAG.VectorStore = val
	}
	
	if val := os.Getenv(EnvVarPrefix + "RAG_EMBEDDINGS"); val != "" {
		config.RAG.Embeddings = val
	}
	
	// Override tracing configuration
	if val := os.Getenv(EnvVarPrefix + "TRACING_ENABLED"); val != "" {
		config.Tracing.Enabled = val == "true" || val == "1" || val == "yes"
	}
	
	if val := os.Getenv(EnvVarPrefix + "TRACING_TYPE"); val != "" {
		config.Tracing.Type = val
	}
	
	if val := os.Getenv(EnvVarPrefix + "TRACING_ENDPOINT"); val != "" {
		config.Tracing.Endpoint = val
	}
	
	// Override custom provider paths
	if val := os.Getenv(EnvVarPrefix + "CUSTOM_PROVIDER_PATHS"); val != "" {
		config.CustomProviderPaths = strings.Split(val, ":")
	}
	
	return nil
}

// CreateRegistryWithConfig creates a provider registry from the configuration
func CreateRegistryWithConfig(configPath string) (*core.Registry, error) {
	// Load the configuration
	config, err := LoadConfigWithEnv(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create the registry
	registry, err := config.CreateRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}
	
	// Load custom providers
	if len(config.CustomProviderPaths) > 0 {
		loader := custom.NewProviderLoader(config.CustomProviderPaths)
		if err := loader.LoadProviders(registry); err != nil {
			return nil, fmt.Errorf("failed to load custom providers: %w", err)
		}
	}
	
	return registry, nil
}

// FindConfigFile searches for a configuration file in common locations
func FindConfigFile() (string, error) {
	// Check environment variable
	if path := os.Getenv(EnvVarPrefix + "CONFIG"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// Check common locations
	locations := []string{
		"./gollem.json",
		"./config/gollem.json",
		"~/.gollem/config.json",
		"/etc/gollem/config.json",
	}
	
	for _, loc := range locations {
		// Expand home directory
		if strings.HasPrefix(loc, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			loc = filepath.Join(home, loc[2:])
		}
		
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}
	
	return "", fmt.Errorf("no configuration file found")
}

// CreateDefaultConfigFile creates a default configuration file at the specified path
func CreateDefaultConfigFile(path string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create the default configuration
	config := DefaultConfig()
	
	// Save the configuration
	if err := SaveConfig(config, path); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	return nil
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Check if the default provider exists
	if _, exists := config.Providers[config.DefaultProvider]; !exists {
		return fmt.Errorf("default provider %s does not exist", config.DefaultProvider)
	}
	
	// Validate provider configurations
	for name, provider := range config.Providers {
		if provider.Type == "" {
			return fmt.Errorf("provider %s has no type", name)
		}
		
		// Validate API key for providers that require it
		if provider.Type != "custom" && provider.APIKey == "" {
			return fmt.Errorf("provider %s requires an API key", name)
		}
	}
	
	return nil
}

// MergeConfigs merges two configurations, with the second taking precedence
func MergeConfigs(base, override *Config) *Config {
	result := *base
	
	// Override default provider
	if override.DefaultProvider != "" {
		result.DefaultProvider = override.DefaultProvider
	}
	
	// Merge providers
	for name, provider := range override.Providers {
		result.Providers[name] = provider
	}
	
	// Merge cache configuration
	if override.Cache.Type != "" {
		result.Cache.Type = override.Cache.Type
	}
	
	if override.Cache.TTL != 0 {
		result.Cache.TTL = override.Cache.TTL
	}
	
	if override.Cache.MaxEntries != 0 {
		result.Cache.MaxEntries = override.Cache.MaxEntries
	}
	
	// Merge RAG configuration
	if override.RAG.VectorStore != "" {
		result.RAG.VectorStore = override.RAG.VectorStore
	}
	
	if override.RAG.Embeddings != "" {
		result.RAG.Embeddings = override.RAG.Embeddings
	}
	
	if override.RAG.ChunkSize != 0 {
		result.RAG.ChunkSize = override.RAG.ChunkSize
	}
	
	if override.RAG.ChunkOverlap != 0 {
		result.RAG.ChunkOverlap = override.RAG.ChunkOverlap
	}
	
	// Merge tracing configuration
	if override.Tracing.Type != "" {
		result.Tracing.Type = override.Tracing.Type
	}
	
	if override.Tracing.Endpoint != "" {
		result.Tracing.Endpoint = override.Tracing.Endpoint
	}
	
	if override.Tracing.SampleRate != 0 {
		result.Tracing.SampleRate = override.Tracing.SampleRate
	}
	
	// Merge custom provider paths
	if len(override.CustomProviderPaths) > 0 {
		result.CustomProviderPaths = append(result.CustomProviderPaths, override.CustomProviderPaths...)
	}
	
	return &result
}
