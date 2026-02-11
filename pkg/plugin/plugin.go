// Package plugin provides the Plugin SDK for Partir Core.
// Plugins declare job types, schemas, execution logic, and custom gates.
package plugin

import (
	"context"
	"encoding/json"
)

// Plugin is the interface that all plugins must implement
type Plugin interface {
	// Identity
	ID() string
	Version() string
	Manifest() *Manifest    // Security metadata
	Capabilities() []string // Job types this plugin handles

	// Schemas
	InputSchema(jobType string) json.RawMessage
	OutputSchema(jobType string) json.RawMessage

	// Execution
	Plan(workOrder WorkOrder) ([]Step, error) // Optional: break work into steps
	Execute(ctx context.Context, workOrder WorkOrder) (*ExecutionResult, error)

	// Gates
	Gates(jobType string) []string                                        // Gate IDs required for this job type
	Validate(ctx context.Context, artifacts []Artifact) ([]Defect, error) // Plugin-side validation
}

// WorkOrder contains everything needed for execution
type WorkOrder struct {
	TicketID    string          `json:"ticket_id"`
	JobType     string          `json:"job_type"`
	AlphaRef    string          `json:"alpha_ref"`
	BetaRef     string          `json:"beta_ref,omitempty"`
	Inputs      json.RawMessage `json:"inputs"`
	Constraints json.RawMessage `json:"constraints,omitempty"`
	Limits      Limits          `json:"limits"`
	Attempt     int             `json:"attempt"`
	DeltaFields []string        `json:"delta_fields,omitempty"` // For targeted retries
}

// Limits defines execution constraints
type Limits struct {
	MaxFilesChanged int     `json:"max_files_changed,omitempty"`
	MaxDiffLines    int     `json:"max_diff_lines,omitempty"`
	MaxAttempts     int     `json:"max_attempts,omitempty"`
	MaxCostBudget   float64 `json:"max_cost_budget,omitempty"`
}

// Step represents a planned execution step
type Step struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Inputs      json.RawMessage `json:"inputs,omitempty"`
}

// ExecutionResult contains the output of plugin execution
type ExecutionResult struct {
	Artifacts  []Artifact `json:"artifacts"`
	CostTokens int        `json:"cost_tokens"`
	CostUSD    float64    `json:"cost_usd"`
	Logs       string     `json:"logs,omitempty"`
}

// Artifact represents a generated artifact
type Artifact struct {
	ArtifactID   string          `json:"artifact_id"`
	ArtifactType string          `json:"artifact_type"`
	Hash         string          `json:"hash"`
	SchemaRef    string          `json:"schema_ref,omitempty"`
	Data         json.RawMessage `json:"data"`
	Provenance   Provenance      `json:"provenance"`
}

// Provenance tracks artifact origin
type Provenance struct {
	TicketID   string `json:"ticket_id"`
	PluginID   string `json:"plugin_id"`
	InputsHash string `json:"inputs_hash"`
	AlphaRef   string `json:"alpha_ref"`
	BetaRef    string `json:"beta_ref,omitempty"`
}

// Defect represents a validation failure from plugin
type Defect struct {
	GateID          string          `json:"gate_id"`
	DefectClass     string          `json:"defect_class"`
	Message         string          `json:"message"`
	OffendingFields json.RawMessage `json:"offending_fields,omitempty"`
	SuggestedFix    string          `json:"suggested_fix,omitempty"`
}
