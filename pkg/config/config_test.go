package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/gollem/pkg/config"
)

// TestConfigLoading tests the configuration loading functionality
func TestConfigLoading(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	
	// Create a default config
	defaultConfig := config.DefaultConfig()
	
	// Save the config
	err := config.SaveConfig(defaultConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Load the config
	loadedConfig, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Check the config
	if loadedConfig.DefaultProvider != defaultConfig.DefaultProvider {
		t.Fatalf("Default provider is incorrect: %s", loadedConfig.DefaultProvider)
	}
	
	if len(loadedConfig.Providers) != len(defaultConfig.Providers) {
		t.Fatalf("Providers count is incorrect: %d", len(loadedConfig.Providers))
	}
	
	// Test environment variable overrides
	os.Setenv("GOLLEM_DEFAULT_PROVIDER", "test_provider")
	os.Setenv("GOLLEM_OPENAI_API_KEY", "test_api_key")
	
	// Load the config with environment variables
	envConfig, err := config.LoadConfigWithEnv(configPath)
	if err != nil {
		t.Fatalf("Failed to load config with env: %v", err)
	}
	
	// Check the overrides
	if envConfig.DefaultProvider != "test_provider" {
		t.Fatalf("Environment override for default provider failed: %s", envConfig.DefaultProvider)
	}
	
	if envConfig.Providers["openai"].APIKey != "test_api_key" {
		t.Fatalf("Environment override for API key failed: %s", envConfig.Providers["openai"].APIKey)
	}
	
	// Clean up
	os.Unsetenv("GOLLEM_DEFAULT_PROVIDER")
	os.Unsetenv("GOLLEM_OPENAI_API_KEY")
}

// TestConfigManager tests the configuration manager
func TestConfigManager(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	
	// Create a default config
	defaultConfig := config.DefaultConfig()
	
	// Save the config
	err := config.SaveConfig(defaultConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Create a config manager
	manager, err := config.NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}
	
	// Get the config
	cfg := manager.GetConfig()
	if cfg == nil {
		t.Fatal("Config is nil")
	}
	
	// Update a provider
	err = manager.UpdateProvider("test_provider", config.ProviderConfig{
		Type:   "openai",
		APIKey: "test_api_key",
		Model:  "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to update provider: %v", err)
	}
	
	// Set as default provider
	err = manager.SetDefaultProvider("test_provider")
	if err != nil {
		t.Fatalf("Failed to set default provider: %v", err)
	}
	
	// Get the updated config
	cfg = manager.GetConfig()
	if cfg.DefaultProvider != "test_provider" {
		t.Fatalf("Default provider is incorrect: %s", cfg.DefaultProvider)
	}
	
	// Get provider names
	names := manager.GetProviderNames()
	hasTestProvider := false
	for _, name := range names {
		if name == "test_provider" {
			hasTestProvider = true
			break
		}
	}
	if !hasTestProvider {
		t.Fatal("test_provider not found in provider names")
	}
	
	// Get default provider
	name, providerConfig, err := manager.GetDefaultProvider()
	if err != nil {
		t.Fatalf("Failed to get default provider: %v", err)
	}
	if name != "test_provider" {
		t.Fatalf("Default provider name is incorrect: %s", name)
	}
	if providerConfig.Type != "openai" {
		t.Fatalf("Default provider type is incorrect: %s", providerConfig.Type)
	}
	
	// Remove a provider
	err = manager.RemoveProvider("test_provider")
	if err != nil {
		t.Fatalf("Failed to remove provider: %v", err)
	}
	
	// Try to get the removed provider
	_, err = manager.GetProviderConfig("test_provider")
	if err == nil {
		t.Fatal("No error when getting removed provider")
	}
	
	// Enable/disable features
	err = manager.EnableCache(false)
	if err != nil {
		t.Fatalf("Failed to disable cache: %v", err)
	}
	
	err = manager.EnableRAG(true)
	if err != nil {
		t.Fatalf("Failed to enable RAG: %v", err)
	}
	
	err = manager.EnableTracing(true)
	if err != nil {
		t.Fatalf("Failed to enable tracing: %v", err)
	}
	
	// Check the config
	cfg = manager.GetConfig()
	if cfg.Cache.Enabled {
		t.Fatal("Cache is enabled after disabling")
	}
	if !cfg.RAG.Enabled {
		t.Fatal("RAG is disabled after enabling")
	}
	if !cfg.Tracing.Enabled {
		t.Fatal("Tracing is disabled after enabling")
	}
	
	// Test custom provider paths
	err = manager.AddCustomProviderPath("/path/to/custom/providers")
	if err != nil {
		t.Fatalf("Failed to add custom provider path: %v", err)
	}
	
	err = manager.RemoveCustomProviderPath("/path/to/custom/providers")
	if err != nil {
		t.Fatalf("Failed to remove custom provider path: %v", err)
	}
	
	// Export and import config
	exportedConfig, err := manager.ExportConfig()
	if err != nil {
		t.Fatalf("Failed to export config: %v", err)
	}
	
	// Create a new manager
	newManager, err := config.NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create new config manager: %v", err)
	}
	
	// Import the config
	err = newManager.ImportConfig(exportedConfig)
	if err != nil {
		t.Fatalf("Failed to import config: %v", err)
	}
	
	// Check the imported config
	importedCfg := newManager.GetConfig()
	if importedCfg.Cache.Enabled {
		t.Fatal("Imported config has cache enabled")
	}
	if !importedCfg.RAG.Enabled {
		t.Fatal("Imported config has RAG disabled")
	}
	if !importedCfg.Tracing.Enabled {
		t.Fatal("Imported config has tracing disabled")
	}
}

// TestConfigValidation tests the configuration validation
func TestConfigValidation(t *testing.T) {
	// Create a valid config
	validConfig := config.DefaultConfig()
	
	// Validate the config
	err := config.ValidateConfig(validConfig)
	if err != nil {
		t.Fatalf("Validation failed for valid config: %v", err)
	}
	
	// Create an invalid config (non-existent default provider)
	invalidConfig := config.DefaultConfig()
	invalidConfig.DefaultProvider = "non_existent"
	
	// Validate the invalid config
	err = config.ValidateConfig(invalidConfig)
	if err == nil {
		t.Fatal("No error when validating config with non-existent default provider")
	}
	
	// Create an invalid config (provider with no type)
	invalidConfig2 := config.DefaultConfig()
	invalidConfig2.Providers["invalid"] = config.ProviderConfig{
		APIKey: "test_api_key",
	}
	
	// Validate the invalid config
	err = config.ValidateConfig(invalidConfig2)
	if err == nil {
		t.Fatal("No error when validating config with provider with no type")
	}
	
	// Create an invalid config (provider with no API key)
	invalidConfig3 := config.DefaultConfig()
	invalidConfig3.Providers["invalid"] = config.ProviderConfig{
		Type: "openai",
	}
	
	// Validate the invalid config
	err = config.ValidateConfig(invalidConfig3)
	if err == nil {
		t.Fatal("No error when validating config with provider with no API key")
	}
}
