// Package collaboration provides the Collaboration API for Partir Core.
// It is a stateless NATS-based message router that handles Andon Cord signals,
// violation triage, fix-it routing, and repo commit confirmations.
// All audit is done via Factory Ledger — the Collaboration API stores no state.
package collaboration

import (
	"encoding/json"
	"fmt"
	"time"
)

// NATS topic constants for the Collaboration API.
const (
	TopicSubmit  = "collab.submit"  // New ticket from Control Surface
	TopicAndon   = "collab.andon"   // Andon Cord halt signal from Foundry
	TopicFix     = "collab.fix"     // Fix-it ticket routed to API workers
	TopicConfirm = "collab.confirm" // Repo commit confirmation
	TopicAlert   = "collab.alert"   // Human escalation
)

// ViolationType classifies the source of a production failure.
type ViolationType string

const (
	// ViolationBetaOutput indicates Beta API produced bad output (spec/schema mismatch).
	ViolationBetaOutput ViolationType = "beta_output"

	// ViolationFoundryRuntime indicates the Bounded Agent Runtime failed (timeout, crash, etc).
	ViolationFoundryRuntime ViolationType = "foundry_runtime"

	// ViolationAlphaSpec indicates the Alpha spec itself is invalid or impossible.
	ViolationAlphaSpec ViolationType = "alpha_spec"

	// ViolationUnknown is the fallback when classification fails.
	ViolationUnknown ViolationType = "unknown"
)

// AndonSignal is published to collab.andon when the Foundry Orchestrator
// detects a post-execution failure via the Omega engine gates.
type AndonSignal struct {
	SignalID  string            `json:"signal_id"`
	TicketID  string            `json:"ticket_id"`
	RunID     string            `json:"run_id"`
	PluginID  string            `json:"plugin_id"`
	GateID    string            `json:"gate_id"`
	Defects   []DefectSummary   `json:"defects"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// DefectSummary is a compact representation of a defect for triage.
type DefectSummary struct {
	DefectID    string `json:"defect_id"`
	DefectClass string `json:"defect_class"`
	GateID      string `json:"gate_id"`
	Message     string `json:"message"`
}

// FixItTicket is published to collab.fix when the triage logic determines
// a violation is auto-fixable (Beta output or Foundry runtime issue).
type FixItTicket struct {
	FixID         string          `json:"fix_id"`
	OriginalRunID string          `json:"original_run_id"`
	TicketID      string          `json:"ticket_id"`
	TargetAPI     string          `json:"target_api"` // "beta" or "foundry"
	Violation     ViolationType   `json:"violation"`
	Defects       []DefectSummary `json:"defects"`
	Timestamp     time.Time       `json:"timestamp"`
}

// HumanAlert is published to collab.alert when the triage logic detects
// an Alpha spec error that requires human intervention.
type HumanAlert struct {
	AlertID   string          `json:"alert_id"`
	TicketID  string          `json:"ticket_id"`
	RunID     string          `json:"run_id"`
	Violation ViolationType   `json:"violation"`
	Defects   []DefectSummary `json:"defects"`
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"timestamp"`
}

// RepoConfirmation is published to collab.confirm when an artifact
// is successfully committed to the repo/artifact store.
type RepoConfirmation struct {
	TicketID   string    `json:"ticket_id"`
	RunID      string    `json:"run_id"`
	ArtifactID string    `json:"artifact_id"`
	RepoRef    string    `json:"repo_ref"`
	Timestamp  time.Time `json:"timestamp"`
}

// ToPayload converts an AndonSignal to a generic map for Ledger storage.
func (s *AndonSignal) ToPayload() map[string]interface{} {
	return map[string]interface{}{
		"signal_id": s.SignalID,
		"ticket_id": s.TicketID,
		"run_id":    s.RunID,
		"plugin_id": s.PluginID,
		"gate_id":   s.GateID,
		"defects":   s.Defects,
		"timestamp": s.Timestamp,
		"metadata":  s.Metadata,
	}
}

// Marshal encodes the signal as JSON bytes for NATS publishing.
func (s *AndonSignal) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

// Marshal encodes a fix-it ticket as JSON bytes for NATS publishing.
func (f *FixItTicket) Marshal() ([]byte, error) {
	return json.Marshal(f)
}

// Marshal encodes a human alert as JSON bytes for NATS publishing.
func (a *HumanAlert) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

// Marshal encodes a repo confirmation as JSON bytes for NATS publishing.
func (r *RepoConfirmation) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// ClassifyViolation determines the ViolationType from a set of defect summaries.
// The classification logic inspects defect classes and gate IDs to determine
// whether the failure is auto-fixable (Beta/Foundry) or requires human escalation (Alpha).
func ClassifyViolation(defects []DefectSummary) ViolationType {
	for _, d := range defects {
		// Alpha spec violations are identified by schema_gate failures
		// or defect classes that indicate the spec itself is wrong.
		switch d.DefectClass {
		case "schema_violation", "spec_invalid", "constraint_impossible":
			return ViolationAlphaSpec
		case "runtime_error", "timeout", "crash":
			return ViolationFoundryRuntime
		}
	}

	// If no Alpha/Foundry issues, assume Beta output issue (most common)
	if len(defects) > 0 {
		return ViolationBetaOutput
	}

	return ViolationUnknown
}

// ViolationDescription returns a human-readable description.
func ViolationDescription(v ViolationType) string {
	switch v {
	case ViolationBetaOutput:
		return "Beta API produced output that failed Omega gates"
	case ViolationFoundryRuntime:
		return "Bounded Agent Runtime failed during execution"
	case ViolationAlphaSpec:
		return "Alpha specification is invalid or impossible to fulfill"
	default:
		return fmt.Sprintf("Unknown violation type: %s", v)
	}
}
