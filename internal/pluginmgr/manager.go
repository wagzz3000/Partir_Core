package pluginmgr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// RegistryEntry describes a plugin in the remote registry
type RegistryEntry struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	DownloadURL string   `json:"download_url"`
	SHA256      string   `json:"sha256"`
	Signature   string   `json:"signature,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// InstalledPlugin tracks a locally installed plugin
type InstalledPlugin struct {
	Slug        string    `json:"slug"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installed_at"`
	SHA256      string    `json:"sha256"`
}

// Manager handles plugin lifecycle operations
type Manager struct {
	pluginDir   string
	registryURL string
	client      *http.Client
}

// NewManager creates a new plugin manager
func NewManager(pluginDir, registryURL string) *Manager {
	return &Manager{
		pluginDir:   pluginDir,
		registryURL: registryURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Install downloads, verifies, and installs a plugin
func (m *Manager) Install(ctx context.Context, slug string) (*InstalledPlugin, error) {
	// Fetch registry entry
	entry, err := m.fetchRegistryEntry(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry entry: %w", err)
	}

	// Download plugin binary
	data, err := m.download(ctx, entry.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}

	// Verify SHA-256
	hash := sha256.Sum256(data)
	actualHash := hex.EncodeToString(hash[:])
	if entry.SHA256 != "" && actualHash != entry.SHA256 {
		return nil, fmt.Errorf("SHA-256 mismatch: expected %s, got %s", entry.SHA256, actualHash)
	}

	// Write to plugin directory
	if err := os.MkdirAll(m.pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin dir: %w", err)
	}

	pluginPath := filepath.Join(m.pluginDir, slug)
	if err := os.WriteFile(pluginPath, data, 0755); err != nil {
		return nil, fmt.Errorf("failed to write plugin: %w", err)
	}

	// Record installation
	installed := &InstalledPlugin{
		Slug:        slug,
		Version:     entry.Version,
		Path:        pluginPath,
		InstalledAt: time.Now(),
		SHA256:      actualHash,
	}

	if err := m.saveManifest(installed); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	return installed, nil
}

// Update checks for and applies updates to an installed plugin
func (m *Manager) Update(ctx context.Context, slug string) (*InstalledPlugin, error) {
	// Load current manifest
	current, err := m.loadManifest(slug)
	if err != nil {
		return nil, fmt.Errorf("plugin %q is not installed", slug)
	}

	// Fetch latest from registry
	entry, err := m.fetchRegistryEntry(ctx, slug)
	if err != nil {
		return nil, err
	}

	if entry.Version == current.Version {
		return current, nil // Already up to date
	}

	// Re-install with latest
	return m.Install(ctx, slug)
}

// ListRemote fetches the full plugin registry
func (m *Manager) ListRemote(ctx context.Context) ([]RegistryEntry, error) {
	url := m.registryURL + "/plugins"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var entries []RegistryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode registry: %w", err)
	}

	return entries, nil
}

// ListInstalled returns all locally installed plugins
func (m *Manager) ListInstalled() ([]InstalledPlugin, error) {
	manifestDir := filepath.Join(m.pluginDir, ".manifests")
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []InstalledPlugin
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(manifestDir, e.Name()))
		if err != nil {
			continue
		}
		var p InstalledPlugin
		if json.Unmarshal(data, &p) == nil {
			plugins = append(plugins, p)
		}
	}
	return plugins, nil
}

// --- Internal helpers ---

func (m *Manager) fetchRegistryEntry(ctx context.Context, slug string) (*RegistryEntry, error) {
	url := fmt.Sprintf("%s/plugins/%s", m.registryURL, slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin %q not found (status %d)", slug, resp.StatusCode)
	}

	var entry RegistryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (m *Manager) saveManifest(p *InstalledPlugin) error {
	manifestDir := filepath.Join(m.pluginDir, ".manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(manifestDir, p.Slug+".json"), data, 0644)
}

func (m *Manager) loadManifest(slug string) (*InstalledPlugin, error) {
	path := filepath.Join(m.pluginDir, ".manifests", slug+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p InstalledPlugin
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
