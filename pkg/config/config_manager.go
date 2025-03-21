package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// ConfigManager manages configuration for the gollem package
type ConfigManager struct {
	config     *Config
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) (*ConfigManager, error) {
	// If no path is provided, try to find a config file
	if configPath == "" {
		var err error
		configPath, err = FindConfigFile()
		if err != nil {
			// Create a default config in the user's home directory
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			
			configPath = filepath.Join(home, ".gollem", "config.json")
			if err := CreateDefaultConfigFile(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", err)
			}
		}
	}
	
	// Load the configuration
	config, err := LoadConfigWithEnv(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Validate the configuration
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &ConfigManager{
		config:     config,
		configPath: configPath,
	}, nil
}

// GetConfig returns the current configuration
func (m *ConfigManager) GetConfig() *Config {
	return m.config
}

// SaveConfig saves the current configuration to the file
func (m *ConfigManager) SaveConfig() error {
	return SaveConfig(m.config, m.configPath)
}

// CreateRegistry creates a provider registry from the configuration
func (m *ConfigManager) CreateRegistry() (*core.Registry, error) {
	return m.config.CreateRegistry()
}

// UpdateProvider updates a provider configuration
func (m *ConfigManager) UpdateProvider(name string, config ProviderConfig) error {
	m.config.Providers[name] = config
	return m.SaveConfig()
}

// RemoveProvider removes a provider configuration
func (m *ConfigManager) RemoveProvider(name string) error {
	delete(m.config.Providers, name)
	return m.SaveConfig()
}

// SetDefaultProvider sets the default provider
func (m *ConfigManager) SetDefaultProvider(name string) error {
	if _, exists := m.config.Providers[name]; !exists {
		return fmt.Errorf("provider %s does not exist", name)
	}
	
	m.config.DefaultProvider = name
	return m.SaveConfig()
}

// EnableCache enables or disables caching
func (m *ConfigManager) EnableCache(enabled bool) error {
	m.config.Cache.Enabled = enabled
	return m.SaveConfig()
}

// EnableRAG enables or disables RAG
func (m *ConfigManager) EnableRAG(enabled bool) error {
	m.config.RAG.Enabled = enabled
	return m.SaveConfig()
}

// EnableTracing enables or disables tracing
func (m *ConfigManager) EnableTracing(enabled bool) error {
	m.config.Tracing.Enabled = enabled
	return m.SaveConfig()
}

// AddCustomProviderPath adds a path to search for custom providers
func (m *ConfigManager) AddCustomProviderPath(path string) error {
	// Check if the path already exists
	for _, p := range m.config.CustomProviderPaths {
		if p == path {
			return nil
		}
	}
	
	m.config.CustomProviderPaths = append(m.config.CustomProviderPaths, path)
	return m.SaveConfig()
}

// RemoveCustomProviderPath removes a path from the custom provider paths
func (m *ConfigManager) RemoveCustomProviderPath(path string) error {
	var paths []string
	for _, p := range m.config.CustomProviderPaths {
		if p != path {
			paths = append(paths, p)
		}
	}
	
	m.config.CustomProviderPaths = paths
	return m.SaveConfig()
}

// GetProviderNames returns a list of all provider names
func (m *ConfigManager) GetProviderNames() []string {
	names := make([]string, 0, len(m.config.Providers))
	for name := range m.config.Providers {
		names = append(names, name)
	}
	return names
}

// GetDefaultProvider returns the default provider configuration
func (m *ConfigManager) GetDefaultProvider() (string, ProviderConfig, error) {
	name := m.config.DefaultProvider
	provider, exists := m.config.Providers[name]
	if !exists {
		return "", ProviderConfig{}, fmt.Errorf("default provider %s does not exist", name)
	}
	
	return name, provider, nil
}

// GetProviderConfig returns a provider configuration
func (m *ConfigManager) GetProviderConfig(name string) (ProviderConfig, error) {
	provider, exists := m.config.Providers[name]
	if !exists {
		return ProviderConfig{}, fmt.Errorf("provider %s does not exist", name)
	}
	
	return provider, nil
}

// ExportConfig exports the configuration as JSON
func (m *ConfigManager) ExportConfig() (string, error) {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return string(data), nil
}

// ImportConfig imports a configuration from JSON
func (m *ConfigManager) ImportConfig(jsonStr string) error {
	var config Config
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Validate the configuration
	if err := ValidateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	m.config = &config
	return m.SaveConfig()
}
