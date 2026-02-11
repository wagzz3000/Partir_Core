// Package beta provides the Beta module for Partir Core.
// Beta defines "expression" - visualization/style rulebooks, render vocab, UX constraints.
package beta

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Manifest represents a Beta rulebook bundle manifest
type Manifest struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Hash          string    `json:"hash"`
	CreatedAt     time.Time `json:"created_at"`
	ParentVersion string    `json:"parent_version,omitempty"`
	CoreVersion   string    `json:"core_version"`
}

// Rulebook represents a complete Beta rulebook bundle
type Rulebook struct {
	Manifest    Manifest               `json:"manifest"`
	RenderVocab RenderVocab            `json:"render_vocab"` // Effect types
	StyleRules  StyleRules             `json:"style_rules"`  // Bounds + propagation
	Schemas     map[string]interface{} `json:"schemas,omitempty"`
}

// RenderVocab defines the atomic effect vocabulary
type RenderVocab struct {
	Effects    []Effect    `json:"effects"`    // pulse, wave, flame, smoke, shadow, etc.
	Palettes   []Palette   `json:"palettes"`   // Color palettes
	Animations []Animation `json:"animations"` // Animation envelopes
}

// Effect represents a render effect type
type Effect struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // glow, particle, overlay, etc.
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]EffectParam `json:"parameters"`
	Composable  bool                   `json:"composable"` // Can be combined with others
}

// EffectParam defines a parameter for an effect
type EffectParam struct {
	Type    string      `json:"type"` // float, int, color, enum
	Default interface{} `json:"default"`
	Min     interface{} `json:"min,omitempty"`
	Max     interface{} `json:"max,omitempty"`
	Options []string    `json:"options,omitempty"` // For enum type
}

// Palette represents a color palette
type Palette struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Colors      []string `json:"colors"` // Hex colors
	Description string   `json:"description,omitempty"`
}

// Animation represents an animation envelope
type Animation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // ease, bounce, elastic, etc.
	DurationMs  int    `json:"duration_ms"`
	Description string `json:"description,omitempty"`
}

// StyleRules defines bounds and propagation rules
type StyleRules struct {
	Bounds      []StyleBound      `json:"bounds,omitempty"`
	Propagation []PropagationRule `json:"propagation,omitempty"`
}

// StyleBound defines min/max bounds for style properties
type StyleBound struct {
	Property string      `json:"property"`
	Min      interface{} `json:"min,omitempty"`
	Max      interface{} `json:"max,omitempty"`
	Unit     string      `json:"unit,omitempty"`
}

// PropagationRule defines how styles propagate
type PropagationRule struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	From     string   `json:"from"`     // Source property
	To       string   `json:"to"`       // Target property
	Factor   float64  `json:"factor"`   // Multiplier
	Contexts []string `json:"contexts"` // When to apply
}

// NewRulebook creates a new empty Beta rulebook
func NewRulebook(name string) *Rulebook {
	return &Rulebook{
		Manifest: Manifest{
			ID:          uuid.New().String(),
			Name:        name,
			CreatedAt:   time.Now(),
			CoreVersion: "0.1.0",
		},
		RenderVocab: RenderVocab{
			Effects:    []Effect{},
			Palettes:   []Palette{},
			Animations: []Animation{},
		},
		StyleRules: StyleRules{},
		Schemas:    make(map[string]interface{}),
	}
}

// ComputeHash computes the stable content hash of the rulebook
func (r *Rulebook) ComputeHash() (string, error) {
	data, err := json.Marshal(struct {
		RenderVocab RenderVocab            `json:"render_vocab"`
		StyleRules  StyleRules             `json:"style_rules"`
		Schemas     map[string]interface{} `json:"schemas"`
	}{
		RenderVocab: r.RenderVocab,
		StyleRules:  r.StyleRules,
		Schemas:     r.Schemas,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal rulebook: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// Promote promotes the rulebook to a new version
func (r *Rulebook) Promote(version string) error {
	hash, err := r.ComputeHash()
	if err != nil {
		return err
	}

	r.Manifest.ParentVersion = r.Manifest.Version
	r.Manifest.Version = version
	r.Manifest.Hash = hash
	r.Manifest.CreatedAt = time.Now()

	return nil
}

// ToJSON serializes the rulebook to canonical JSON
func (r *Rulebook) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON deserializes a rulebook from JSON
func FromJSON(data []byte) (*Rulebook, error) {
	var r Rulebook
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("failed to parse rulebook: %w", err)
	}
	return &r, nil
}
