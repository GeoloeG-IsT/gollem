package core

import (
	"fmt"
	"sync"
)

// Registry is used to register and retrieve LLM providers
type Registry struct {
	providers map[string]LLMProvider
	factories map[string]ProviderFactory
	mu        sync.RWMutex
}

// ProviderFactory is a function that creates a provider from a configuration
type ProviderFactory func(config map[string]interface{}) (LLMProvider, error)

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]LLMProvider),
		factories: make(map[string]ProviderFactory),
	}
}

// RegisterProvider registers a provider with the registry
func (r *Registry) RegisterProvider(provider LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// GetProvider retrieves a provider from the registry
func (r *Registry) GetProvider(name string) (LLMProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, exists := r.providers[name]
	return provider, exists
}

// RegisterFactory registers a provider factory with the registry
func (r *Registry) RegisterFactory(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// CreateProvider creates a provider using a registered factory
func (r *Registry) CreateProvider(name string, config map[string]interface{}) (LLMProvider, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no factory registered for provider: %s", name)
	}
	
	provider, err := factory(config)
	if err != nil {
		return nil, err
	}
	
	r.RegisterProvider(provider)
	return provider, nil
}
