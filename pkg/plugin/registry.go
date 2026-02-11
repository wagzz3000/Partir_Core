package plugin

import (
	"fmt"
	"sync"
)

// Registry manages registered plugins
type Registry struct {
	mu        sync.RWMutex
	plugins   map[string]Plugin
	trustRoot string // Hex-encoded public key for verification
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// SetTrustRoot sets the public key for verifying plugin manifests
func (r *Registry) SetTrustRoot(pubKeyHex string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trustRoot = pubKeyHex
}

// Register adds a plugin to the registry
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify signature if trust root is configured
	if r.trustRoot != "" {
		if plugin.Manifest() == nil {
			return fmt.Errorf("plugin %q missing manifest", plugin.ID())
		}
		if err := plugin.Manifest().VerifySignature(r.trustRoot); err != nil {
			return fmt.Errorf("plugin %q signature verification failed: %w", plugin.ID(), err)
		}
	}

	id := plugin.ID()
	if _, exists := r.plugins[id]; exists {
		return fmt.Errorf("plugin %q already registered", id)
	}

	r.plugins[id] = plugin
	return nil
}

// Get retrieves a plugin by ID
func (r *Registry) Get(id string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", id)
	}
	return plugin, nil
}

// List returns all registered plugins
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListCapabilities returns all job types across all plugins
func (r *Registry) ListCapabilities() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	caps := make(map[string][]string)
	for id, p := range r.plugins {
		caps[id] = p.Capabilities()
	}
	return caps
}

// FindByCapability finds plugins that support a given job type
func (r *Registry) FindByCapability(jobType string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Plugin
	for _, p := range r.plugins {
		for _, cap := range p.Capabilities() {
			if cap == jobType {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// Unregister removes a plugin from the registry
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, id)
}
