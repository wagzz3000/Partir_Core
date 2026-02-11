package beta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Workspace manages local Beta rulebook editing
type Workspace struct {
	BaseDir string
}

// NewWorkspace creates a new workspace manager
func NewWorkspace(baseDir string) *Workspace {
	return &Workspace{BaseDir: baseDir}
}

// Init creates a new Beta rulebook workspace
func (w *Workspace) Init(name string) (*Rulebook, error) {
	dir := filepath.Join(w.BaseDir, name)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	rb := NewRulebook(name)

	// Add default effects as examples
	rb.RenderVocab.Effects = []Effect{
		{
			ID:   "pulse",
			Name: "Pulse",
			Type: "glow",
			Parameters: map[string]EffectParam{
				"intensity": {Type: "float", Default: 0.5, Min: 0.0, Max: 1.0},
				"speed":     {Type: "float", Default: 1.0, Min: 0.1, Max: 5.0},
			},
			Composable: true,
		},
		{
			ID:   "wave",
			Name: "Wave",
			Type: "distortion",
			Parameters: map[string]EffectParam{
				"amplitude": {Type: "float", Default: 0.1, Min: 0.0, Max: 1.0},
				"frequency": {Type: "float", Default: 2.0, Min: 0.1, Max: 10.0},
			},
			Composable: true,
		},
	}

	// Add default palette
	rb.RenderVocab.Palettes = []Palette{
		{
			ID:     "default",
			Name:   "Default",
			Colors: []string{"#1a1a2e", "#16213e", "#0f3460", "#e94560"},
		},
	}

	// Write manifest and content
	if err := w.Save(name, rb); err != nil {
		return nil, err
	}

	return rb, nil
}

// Save saves the rulebook to the workspace
func (w *Workspace) Save(name string, rb *Rulebook) error {
	dir := filepath.Join(w.BaseDir, name)

	// Write manifest
	manifestPath := filepath.Join(dir, "beta_manifest.json")
	manifestData, err := json.MarshalIndent(rb.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return err
	}

	// Write render_vocab
	vocabPath := filepath.Join(dir, "render_vocab.json")
	vocabData, err := json.MarshalIndent(rb.RenderVocab, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal render_vocab: %w", err)
	}
	if err := os.WriteFile(vocabPath, vocabData, 0644); err != nil {
		return err
	}

	// Write style_rules
	rulesPath := filepath.Join(dir, "style_rules.json")
	rulesData, err := json.MarshalIndent(rb.StyleRules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal style_rules: %w", err)
	}
	return os.WriteFile(rulesPath, rulesData, 0644)
}

// Load loads a rulebook from the workspace
func (w *Workspace) Load(name string) (*Rulebook, error) {
	dir := filepath.Join(w.BaseDir, name)

	rb := &Rulebook{
		Schemas: make(map[string]interface{}),
	}

	// Load manifest
	manifestPath := filepath.Join(dir, "beta_manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	if err := json.Unmarshal(manifestData, &rb.Manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Load render_vocab (try both JSON and YAML)
	if data, err := w.loadFile(dir, "render_vocab"); err == nil {
		if err := yaml.Unmarshal(data, &rb.RenderVocab); err != nil {
			if err := json.Unmarshal(data, &rb.RenderVocab); err != nil {
				return nil, fmt.Errorf("failed to parse render_vocab: %w", err)
			}
		}
	}

	// Load style_rules
	if data, err := w.loadFile(dir, "style_rules"); err == nil {
		if err := yaml.Unmarshal(data, &rb.StyleRules); err != nil {
			if err := json.Unmarshal(data, &rb.StyleRules); err != nil {
				return nil, fmt.Errorf("failed to parse style_rules: %w", err)
			}
		}
	}

	return rb, nil
}

// loadFile tries to load a file with .json or .yaml extension
func (w *Workspace) loadFile(dir, baseName string) ([]byte, error) {
	for _, ext := range []string{".json", ".yaml", ".yml"} {
		path := filepath.Join(dir, baseName+ext)
		if data, err := os.ReadFile(path); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", baseName)
}

// Build compiles sources into a canonical JSON bundle
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
	path := filepath.Join(w.BaseDir, name, "beta_manifest.json")
	_, err := os.Stat(path)
	return err == nil
}
