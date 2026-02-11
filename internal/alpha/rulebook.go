// Package alpha provides the Alpha module for Partir Core.
// Alpha defines "law" - schemas, catalogs, rule tables, and constraints.
package alpha

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Manifest represents an Alpha rulebook bundle manifest
type Manifest struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Hash          string    `json:"hash"`
	CreatedAt     time.Time `json:"created_at"`
	ParentVersion string    `json:"parent_version,omitempty"`
	CoreVersion   string    `json:"core_version"`
}

// Compat represents compatibility constraints
type Compat struct {
	MinCoreVersion  string   `json:"min_core_version"`
	MaxCoreVersion  string   `json:"max_core_version,omitempty"`
	RequiredPlugins []string `json:"required_plugins,omitempty"`
}

// Rulebook represents a complete Alpha rulebook bundle
type Rulebook struct {
	Manifest Manifest               `json:"manifest"`
	Schemas  map[string]interface{} `json:"schemas"`  // JSON Schema artifacts
	Catalogs map[string][]Catalog   `json:"catalogs"` // Component catalogs
	Rules    Rules                  `json:"rules"`    // Combination rules
	Compat   Compat                 `json:"compat"`   // Compatibility constraints
}

// Catalog represents a component catalog
type Catalog struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]interface{} `json:"properties"`
}

// Rules represents combination rules, cost tables, and invariants
type Rules struct {
	Combinations []CombinationRule `json:"combinations,omitempty"` // Allowed/forbidden combinations
	CostTables   map[string][]Cost `json:"cost_tables,omitempty"`  // Cost/power tables
	Invariants   []Invariant       `json:"invariants,omitempty"`   // Must-hold constraints
}

// CombinationRule defines what can be combined
type CombinationRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Requires    []string `json:"requires,omitempty"`
	Forbids     []string `json:"forbids,omitempty"`
	AllowedWith []string `json:"allowed_with,omitempty"`
}

// Cost represents a cost table entry
type Cost struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	BaseCost  float64  `json:"base_cost"`
	PowerCost float64  `json:"power_cost,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Invariant represents a must-hold constraint
type Invariant struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"expression"` // Simple expression language
	ErrorMsg   string `json:"error_msg"`
}

// NewRulebook creates a new empty rulebook
func NewRulebook(name string) *Rulebook {
	return &Rulebook{
		Manifest: Manifest{
			ID:          uuid.New().String(),
			Name:        name,
			CreatedAt:   time.Now(),
			CoreVersion: "0.1.0",
		},
		Schemas:  make(map[string]interface{}),
		Catalogs: make(map[string][]Catalog),
		Rules:    Rules{},
		Compat: Compat{
			MinCoreVersion: "0.1.0",
		},
	}
}

// ComputeHash computes the stable content hash of the rulebook
func (r *Rulebook) ComputeHash() (string, error) {
	// Create canonical JSON (sorted keys)
	data, err := json.Marshal(struct {
		Schemas  map[string]interface{} `json:"schemas"`
		Catalogs map[string][]Catalog   `json:"catalogs"`
		Rules    Rules                  `json:"rules"`
		Compat   Compat                 `json:"compat"`
	}{
		Schemas:  r.Schemas,
		Catalogs: r.Catalogs,
		Rules:    r.Rules,
		Compat:   r.Compat,
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
