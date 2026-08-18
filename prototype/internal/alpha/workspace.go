package alpha

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Workspace manages local rulebook editing before promotion
type Workspace struct {
	BaseDir string
}

// NewWorkspace creates a new workspace manager
func NewWorkspace(baseDir string) *Workspace {
	return &Workspace{BaseDir: baseDir}
}

// Init creates a new rulebook workspace
func (w *Workspace) Init(name string) (*Rulebook, error) {
	dir := filepath.Join(w.BaseDir, name)

	// Create directory structure
	dirs := []string{
		dir,
		filepath.Join(dir, "schemas"),
		filepath.Join(dir, "catalogs"),
		filepath.Join(dir, "rules"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create rulebook
	rb := NewRulebook(name)

	// Write manifest
	if err := w.WriteManifest(name, rb); err != nil {
		return nil, err
	}

	// Write default compat
	if err := w.WriteCompat(name, rb.Compat); err != nil {
		return nil, err
	}

	return rb, nil
}

// WriteManifest writes the manifest to the workspace
func (w *Workspace) WriteManifest(name string, rb *Rulebook) error {
	path := filepath.Join(w.BaseDir, name, "alpha_manifest.json")
	data, err := json.MarshalIndent(rb.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WriteCompat writes the compat configuration
func (w *Workspace) WriteCompat(name string, compat Compat) error {
	path := filepath.Join(w.BaseDir, name, "compat.json")
	data, err := json.MarshalIndent(compat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal compat: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Load loads a rulebook from the workspace
func (w *Workspace) Load(name string) (*Rulebook, error) {
	dir := filepath.Join(w.BaseDir, name)

	rb := &Rulebook{
		Schemas:  make(map[string]interface{}),
		Catalogs: make(map[string][]Catalog),
	}

	// Load manifest
	manifestPath := filepath.Join(dir, "alpha_manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	if err := json.Unmarshal(manifestData, &rb.Manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Load compat
	compatPath := filepath.Join(dir, "compat.json")
	if compatData, err := os.ReadFile(compatPath); err == nil {
		if err := json.Unmarshal(compatData, &rb.Compat); err != nil {
			return nil, fmt.Errorf("failed to parse compat: %w", err)
		}
	}

	// Load schemas
	schemasDir := filepath.Join(dir, "schemas")
	if entries, err := os.ReadDir(schemasDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			schemaName := entry.Name()
			schemaPath := filepath.Join(schemasDir, schemaName)
			schemaData, err := os.ReadFile(schemaPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read schema %s: %w", schemaName, err)
			}

			var schema interface{}
			ext := filepath.Ext(schemaName)
			if ext == ".yaml" || ext == ".yml" {
				if err := yaml.Unmarshal(schemaData, &schema); err != nil {
					return nil, fmt.Errorf("failed to parse schema %s: %w", schemaName, err)
				}
			} else {
				if err := json.Unmarshal(schemaData, &schema); err != nil {
					return nil, fmt.Errorf("failed to parse schema %s: %w", schemaName, err)
				}
			}
			rb.Schemas[schemaName] = schema
		}
	}

	// Load catalogs
	catalogsDir := filepath.Join(dir, "catalogs")
	if entries, err := os.ReadDir(catalogsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			catalogName := entry.Name()
			catalogPath := filepath.Join(catalogsDir, catalogName)
			catalogData, err := os.ReadFile(catalogPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read catalog %s: %w", catalogName, err)
			}

			var catalogs []Catalog
			ext := filepath.Ext(catalogName)
			if ext == ".yaml" || ext == ".yml" {
				if err := yaml.Unmarshal(catalogData, &catalogs); err != nil {
					return nil, fmt.Errorf("failed to parse catalog %s: %w", catalogName, err)
				}
			} else {
				if err := json.Unmarshal(catalogData, &catalogs); err != nil {
					return nil, fmt.Errorf("failed to parse catalog %s: %w", catalogName, err)
				}
			}
			baseName := catalogName[:len(catalogName)-len(ext)]
			rb.Catalogs[baseName] = catalogs
		}
	}

	// Load rules
	rulesPath := filepath.Join(dir, "rules", "rules.json")
	if rulesData, err := os.ReadFile(rulesPath); err == nil {
		if err := json.Unmarshal(rulesData, &rb.Rules); err != nil {
			return nil, fmt.Errorf("failed to parse rules: %w", err)
		}
	}

	return rb, nil
}

// ArchiveVersion saves a frozen copy of the rulebook to the versions directory
func (w *Workspace) ArchiveVersion(name string, rb *Rulebook) error {
	version := rb.Manifest.Version
	if version == "" {
		return fmt.Errorf("cannot archive unversioned rulebook")
	}

	// Create versions directory
	versionDir := filepath.Join(w.BaseDir, name, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Save full rulebook bundle
	data, err := rb.ToJSON()
	if err != nil {
		return err
	}

	path := filepath.Join(versionDir, "rulebook.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write versioned rulebook: %w", err)
	}

	return nil
}

// LoadVersion loads a specific version of a rulebook
func (w *Workspace) LoadVersion(name, version string) (*Rulebook, error) {
	// If version is "latest" or empty, load current workspace
	if version == "" || version == "latest" {
		return w.Load(name)
	}

	// Try to load from versions directory
	path := filepath.Join(w.BaseDir, name, "versions", version, "rulebook.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("version %s not found for rulebook %s", version, name)
		}
		return nil, err
	}

	return FromJSON(data)
}

// Build compiles YAML/JSON sources into a canonical JSON bundle
func (w *Workspace) Build(name string) ([]byte, error) {
	rb, err := w.Load(name)
	if err != nil {
		return nil, err
	}

	hash, err := rb.ComputeHash()
	if err != nil {
		return nil, err
	}
	rb.Manifest.Hash = hash

	return rb.ToJSON()
}

// Exists checks if a workspace exists
func (w *Workspace) Exists(name string) bool {
	path := filepath.Join(w.BaseDir, name, "alpha_manifest.json")
	_, err := os.Stat(path)
	return err == nil
}
