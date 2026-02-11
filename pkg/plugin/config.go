// Package plugin - Config for executor plugins
package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the executors configuration
type Config struct {
	Executors map[string]ExecutorConfig `yaml:"executors"`
}

// ExecutorConfig represents a single executor configuration
type ExecutorConfig struct {
	Type string `yaml:"type"` // "http"
	URL  string `yaml:"url"`
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".partir", "config.yaml")
}

// LoadConfig loads the config from the default path
func LoadConfig() (*Config, error) {
	return LoadConfigFrom(DefaultConfigPath())
}

// LoadConfigFrom loads the config from a specific path
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{Executors: make(map[string]ExecutorConfig)}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Executors == nil {
		cfg.Executors = make(map[string]ExecutorConfig)
	}

	return &cfg, nil
}

// SaveConfig saves the config to the default path
func (c *Config) Save() error {
	return c.SaveTo(DefaultConfigPath())
}

// SaveTo saves the config to a specific path
func (c *Config) SaveTo(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// AddExecutor adds an executor to the config
func (c *Config) AddExecutor(name, url string) {
	c.Executors[name] = ExecutorConfig{
		Type: "http",
		URL:  url,
	}
}

// RemoveExecutor removes an executor from the config
func (c *Config) RemoveExecutor(name string) {
	delete(c.Executors, name)
}

// GetExecutor gets an executor config by name
func (c *Config) GetExecutor(name string) (ExecutorConfig, bool) {
	exec, ok := c.Executors[name]
	return exec, ok
}

// ListExecutors returns all executor names
func (c *Config) ListExecutors() []string {
	names := make([]string, 0, len(c.Executors))
	for name := range c.Executors {
		names = append(names, name)
	}
	return names
}

// NewHTTPClientFromConfig creates an HTTP client from config
func NewHTTPClientFromConfig(name string, cfg ExecutorConfig) (*HTTPClient, error) {
	if cfg.Type != "http" {
		return nil, fmt.Errorf("unsupported executor type: %s", cfg.Type)
	}
	return NewHTTPClient(name, cfg.URL), nil
}
