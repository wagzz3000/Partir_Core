// Package omega provides the Omega module for Partir Core.
// Omega is the authority - validates artifacts, enforces gates, classifies failures.
package omega

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/partir/core/pkg/plugin"
)

// Defect represents a QC failure
type Defect struct {
	ID              string          `json:"id"`
	DefectID        string          `json:"defect_id"`
	GateID          string          `json:"gate_id"`
	DefectClass     string          `json:"defect_class"`
	PluginID        string          `json:"plugin_id,omitempty"`
	Message         string          `json:"message"`
	OffendingFields json.RawMessage `json:"offending_fields"`
	SuggestedFix    string          `json:"suggested_fix,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	// New fields from plugin contract
	Class    string `json:"class,omitempty"`    // Defect classification
	Severity string `json:"severity,omitempty"` // error, warning, info
	Field    string `json:"field,omitempty"`    // Field/path where defect occurred
}

// NewDefect creates a new defect
func NewDefect(gateID, defectClass, message string) *Defect {
	return &Defect{
		ID:          uuid.New().String(),
		DefectID:    uuid.New().String(),
		GateID:      gateID,
		DefectClass: defectClass,
		Message:     message,
		CreatedAt:   time.Now(),
	}
}

// WithOffendingFields adds offending fields to the defect
func (d *Defect) WithOffendingFields(fields interface{}) *Defect {
	data, _ := json.Marshal(fields)
	d.OffendingFields = data
	return d
}

// WithSuggestedFix adds a suggested fix
func (d *Defect) WithSuggestedFix(fix string) *Defect {
	d.SuggestedFix = fix
	return d
}

// Result represents the outcome of Omega validation
type Result struct {
	Pass        bool          `json:"pass"`
	Defects     []Defect      `json:"defects"`
	ShouldRetry bool          `json:"should_retry"`
	DeltaFields []string      `json:"delta_fields,omitempty"` // Fields to retry
	Metrics     ResultMetrics `json:"metrics"`
}

// ResultMetrics contains timing and gate info
type ResultMetrics struct {
	DurationMs  int      `json:"duration_ms"`
	GatesRun    []string `json:"gates_run"`
	GatesFailed []string `json:"gates_failed"`
}

// Gate is the interface for validation gates
type Gate interface {
	ID() string
	Run(ctx context.Context, req *GateRequest) []Defect
}

// GateRequest contains all context needed for gate validation
type GateRequest struct {
	TicketID    string
	RunID       string
	PluginID    string
	AlphaRef    string
	BetaRef     string
	Inputs      json.RawMessage
	Artifacts   []ArtifactData
	AlphaHash   string
	BetaHash    string
	CoreVersion string
	// AgentResult from LLM plugin execution (for post-execution validation)
	AgentResult *plugin.AgentResult `json:"agent_result,omitempty"`
}

// ArtifactData represents an artifact being validated
type ArtifactData struct {
	ArtifactID   string          `json:"artifact_id"`
	ArtifactType string          `json:"artifact_type"`
	Hash         string          `json:"hash"`
	SchemaRef    string          `json:"schema_ref,omitempty"`
	Data         json.RawMessage `json:"data"`
}

// DefectClass constants for categorization
const (
	DefectClassSchema      = "schema_violation"
	DefectClassVersion     = "version_mismatch"
	DefectClassCompat      = "compatibility_error"
	DefectClassCombination = "forbidden_combination"
	DefectClassBudget      = "budget_exceeded"
	DefectClassUniqueness  = "duplicate_artifact"
	DefectClassDeterminism = "nondeterministic_output"
	DefectClassPreflight   = "preflight_failure"
)
