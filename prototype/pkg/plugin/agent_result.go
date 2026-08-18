// Package plugin - Standardized Agent Result Contract
//
// All LLM plugins (Ollama, Helicone) MUST return this structure.
// Omega validates this output for schema correctness, policy compliance,
// guard gates, diff budgets, and determinism requirements.
package plugin

import (
	"encoding/json"
	"time"
)

// AgentResult is the standardized output from any LLM execution
type AgentResult struct {
	// Core identifiers
	TicketID string `json:"ticket_id"`
	RunID    string `json:"run_id"`

	// Optional structured outputs
	Plan  *AgentPlan  `json:"plan,omitempty"`  // plan.json content
	Patch *AgentPatch `json:"patch,omitempty"` // patch.diff content

	// Artifacts produced
	Artifacts []AgentArtifact `json:"artifacts"`

	// LLM telemetry (required)
	Telemetry LLMTelemetry `json:"llm_telemetry"`

	// Success/failure status
	Success bool          `json:"success"`
	Defects []DefectEntry `json:"defects,omitempty"` // Machine-readable failures
}

// AgentPlan represents a structured execution plan
type AgentPlan struct {
	Steps []PlanStep `json:"steps"`
	// Estimated resources
	EstimatedTokens  int     `json:"estimated_tokens,omitempty"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}

// PlanStep is a single step in the execution plan
type PlanStep struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Action      string   `json:"action"` // "create", "modify", "delete", "validate"
	Target      string   `json:"target"` // file path or resource
	DependsOn   []string `json:"depends_on,omitempty"`
}

// AgentPatch represents a patch/diff output
type AgentPatch struct {
	Format  string `json:"format"`  // "unified", "git"
	Content string `json:"content"` // The actual diff
	// Statistics
	FilesChanged int `json:"files_changed"`
	LinesAdded   int `json:"lines_added"`
	LinesRemoved int `json:"lines_removed"`
}

// AgentArtifact represents a produced artifact
type AgentArtifact struct {
	Path     string          `json:"path"`      // Relative path
	Type     string          `json:"type"`      // "file", "directory", "blob"
	MimeType string          `json:"mime_type"` // e.g., "application/json", "text/plain"
	Size     int64           `json:"size"`      // Bytes
	Hash     string          `json:"hash"`      // SHA256 for determinism checks
	Data     json.RawMessage `json:"data"`      // The actual content
}

// LLMTelemetry captures execution metrics from any provider
type LLMTelemetry struct {
	// Provider info
	Provider string `json:"provider"` // "ollama", "helicone", "openai", "anthropic"
	Model    string `json:"model"`    // e.g., "llama3.2", "gpt-4o", "claude-3-sonnet"

	// Token usage
	TokensIn  int `json:"tokens_in"`  // Prompt tokens
	TokensOut int `json:"tokens_out"` // Completion tokens

	// Cost tracking
	CostUSD float64 `json:"cost_usd"` // 0 for local Ollama

	// Timing
	LatencyMs     int64 `json:"latency_ms"`        // Total request time
	TimeToFirstMs int64 `json:"ttft_ms,omitempty"` // Time to first token (streaming)

	// Additional context
	RequestID    string    `json:"request_id,omitempty"` // Provider's request ID
	FinishReason string    `json:"finish_reason"`        // "stop", "length", "error"
	Timestamp    time.Time `json:"timestamp"`
}

// DefectEntry is a machine-readable failure report
type DefectEntry struct {
	// Classification
	Class    string `json:"class"`    // "schema", "policy", "guard", "budget", "determinism"
	Severity string `json:"severity"` // "error", "warning", "info"
	Gate     string `json:"gate"`     // Which gate caught this

	// Details
	Code    string `json:"code"`           // Machine-readable error code
	Message string `json:"message"`        // Human-readable description
	Path    string `json:"path,omitempty"` // File/field path where defect occurred

	// Remediation
	Suggestion string `json:"suggestion,omitempty"` // How to fix
	Retryable  bool   `json:"retryable"`            // Can this be retried?

	// Context
	Expected string `json:"expected,omitempty"` // What was expected
	Actual   string `json:"actual,omitempty"`   // What was received
}

// DefectClasses for classification
const (
	DefectClassSchema      = "schema"
	DefectClassPolicy      = "policy"
	DefectClassGuard       = "guard"
	DefectClassBudget      = "budget"
	DefectClassDeterminism = "determinism"
	DefectClassPrivacy     = "privacy"
)

// DefectSeverities
const (
	DefectSeverityError   = "error"
	DefectSeverityWarning = "warning"
	DefectSeverityInfo    = "info"
)

// Predefined defect codes
const (
	// Schema defects
	DefectCodeInvalidJSON  = "INVALID_JSON"
	DefectCodeMissingField = "MISSING_FIELD"
	DefectCodeTypeMismatch = "TYPE_MISMATCH"

	// Policy defects
	DefectCodePrivacyViolation = "PRIVACY_VIOLATION"
	DefectCodeForbiddenTerm    = "FORBIDDEN_TERM"

	// Guard defects
	DefectCodeGateAFailed = "GATE_A_PATH_ALLOWLIST"
	DefectCodeGateBFailed = "GATE_B_DOMAIN_DENYLIST"
	DefectCodeGateCFailed = "GATE_C_IMPORT_BOUNDARY"
	DefectCodeGateDFailed = "GATE_D_PARALLEL_IMPL"
	DefectCodeGateEFailed = "GATE_E_SHADOW_OTEL"
	DefectCodeGateFailed  = "GATE_F_DIFF_BUDGET"

	// Budget defects
	DefectCodeTokenBudgetExceeded = "TOKEN_BUDGET_EXCEEDED"
	DefectCodeCostBudgetExceeded  = "COST_BUDGET_EXCEEDED"
	DefectCodeFileBudgetExceeded  = "FILE_BUDGET_EXCEEDED"

	// Determinism defects
	DefectCodeNonDeterministic = "NON_DETERMINISTIC"
	DefectCodeHashMismatch     = "HASH_MISMATCH"
)
