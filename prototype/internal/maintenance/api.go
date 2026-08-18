// Package maintenance provides the Maintenance API for Partir Core.
// It enables direct access to memory pairs in the "maintenance" state
// for diagnostics, therapeutic interventions, and lifecycle management.
//
// Patent-derived operations (WOT, CBT, Diagnostics, ApplyScript) are stubs
// that will be fully implemented in future phases. They log to the Factory
// Ledger but return ErrNotImplemented.
//
// Graph Surgery (excision, edge breaking, quarantine) is done directly
// via FalkorDB, not through this API.
package maintenance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	// ErrNotImplemented is returned by stub operations that are not yet built.
	ErrNotImplemented = errors.New("operation not yet implemented")

	// ErrNotInMaintenance is returned when attempting to operate on a pair
	// that is not in the "maintenance" state.
	ErrNotInMaintenance = errors.New("pair is not in maintenance state")

	// ErrPairNotFound is returned when the pair ID does not exist.
	ErrPairNotFound = errors.New("pair not found")

	// ErrInvalidPhase is returned when an invalid maintenance phase is provided.
	ErrInvalidPhase = errors.New("invalid maintenance phase")
)

// ValidPhases defines the allowed maintenance sub-states.
var ValidPhases = map[string]bool{
	"diagnose": true,
	"triage":   true,
	"script":   true,
	"wot":      true,
	"cbt":      true,
	"surgery":  true,
	"rehab":    true,
	"test":     true,
}

// LedgerAppender is the interface for writing audit events to the Factory Ledger.
type LedgerAppender interface {
	Append(ctx context.Context, eventType string, payload map[string]interface{}) error
}

// PairInfo contains summary information about a maintenance pair.
type PairInfo struct {
	PairID           string     `json:"pair_id"`
	APIRole          string     `json:"api_role"`
	ChronoSchema     string     `json:"chrono_schema"`
	GraphName        string     `json:"graph_name"`
	State            string     `json:"state"`
	MaintenancePhase *string    `json:"maintenance_phase"`
	ModelFingerprint *string    `json:"model_fingerprint"`
	LastPromotion    *time.Time `json:"last_promotion_at"`
}

// InspectResult contains the inspection output for a maintenance pair.
type InspectResult struct {
	Pair      PairInfo  `json:"pair"`
	Timestamp time.Time `json:"timestamp"`
	// Future: Chrono stats, Graph node/edge counts
}

// WOTConfig holds configuration for a Weight Oscillation Therapy session.
type WOTConfig struct {
	GroupStrategy  string  `json:"group_strategy"` // e.g. "early_late_layers"
	Magnitude      float64 `json:"magnitude"`      // ±5% to ±30%
	Frequency      string  `json:"frequency"`      // per-inference, per-recursion
	DurationCycles int     `json:"duration_cycles"`
}

// CBTCurriculum holds configuration for a Cognitive Behavioral Therapy session.
type CBTCurriculum struct {
	Name       string   `json:"name"`
	Prompts    []string `json:"prompts"`
	Iterations int      `json:"iterations"`
}

// API provides maintenance operations on memory pairs.
type API struct {
	registryDB *sql.DB
	ledger     LedgerAppender
}

// NewAPI creates a new Maintenance API.
func NewAPI(registryDB *sql.DB, ledger LedgerAppender) *API {
	return &API{
		registryDB: registryDB,
		ledger:     ledger,
	}
}

// ---- Full implementations (Phase 10) ----

// SetPhase updates the maintenance_phase column for a pair and logs to the Ledger.
// The pair must already be in the "maintenance" state.
func (a *API) SetPhase(ctx context.Context, pairID, phase string) error {
	if !ValidPhases[phase] {
		return fmt.Errorf("%w: %q", ErrInvalidPhase, phase)
	}

	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	_, err := a.registryDB.ExecContext(ctx,
		`UPDATE memory_pairs
		 SET maintenance_phase = $1, updated_at = NOW()
		 WHERE pair_id = $2 AND state = 'maintenance'`,
		phase, pairID,
	)
	if err != nil {
		return fmt.Errorf("set phase: %w", err)
	}

	// Dual tracking: log to Ledger
	if a.ledger != nil {
		a.ledger.Append(ctx, "phase_change", map[string]interface{}{
			"pair_id": pairID,
			"phase":   phase,
		})
	}

	log.Printf("[maintenance] Phase set: pair=%s phase=%s", pairID, phase)
	return nil
}

// Inspect returns summary information about a maintenance pair.
func (a *API) Inspect(ctx context.Context, pairID string) (*InspectResult, error) {
	pair, err := a.getPair(ctx, pairID)
	if err != nil {
		return nil, err
	}

	if pair.State != "maintenance" {
		return nil, fmt.Errorf("%w: pair %q is in state %q", ErrNotInMaintenance, pairID, pair.State)
	}

	// Log inspection to Ledger
	if a.ledger != nil {
		a.ledger.Append(ctx, "maintenance", map[string]interface{}{
			"pair_id":   pairID,
			"operation": "inspect",
		})
	}

	return &InspectResult{
		Pair:      *pair,
		Timestamp: time.Now(),
	}, nil
}

// Release transitions a pair from maintenance → sleep, clearing the maintenance_phase.
func (a *API) Release(ctx context.Context, pairID string) error {
	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	_, err := a.registryDB.ExecContext(ctx,
		`UPDATE memory_pairs
		 SET state = 'sleep', maintenance_phase = NULL, updated_at = NOW()
		 WHERE pair_id = $1 AND state = 'maintenance'`,
		pairID,
	)
	if err != nil {
		return fmt.Errorf("release pair: %w", err)
	}

	// Log state change to Ledger
	if a.ledger != nil {
		a.ledger.Append(ctx, "state_change", map[string]interface{}{
			"pair_id":   pairID,
			"action":    "release",
			"old_state": "maintenance",
			"new_state": "sleep",
		})
	}

	log.Printf("[maintenance] Released: pair=%s → sleep", pairID)
	return nil
}

// ---- Stubs (patent-derived, not yet implemented) ----

// Diagnose runs graph diagnostics on a maintenance pair.
// STUB: Logs ledger event, returns ErrNotImplemented.
func (a *API) Diagnose(ctx context.Context, pairID string) error {
	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	if a.ledger != nil {
		a.ledger.Append(ctx, "diagnostic_run", map[string]interface{}{
			"pair_id": pairID,
			"status":  "stub",
		})
	}

	log.Printf("[maintenance] Diagnose stub called: pair=%s", pairID)
	return fmt.Errorf("diagnose: %w", ErrNotImplemented)
}

// ApplyScript looks up a corrective script from the registry and applies it.
// STUB: Logs ledger event, returns ErrNotImplemented.
func (a *API) ApplyScript(ctx context.Context, pairID, scriptID string) error {
	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	if a.ledger != nil {
		a.ledger.Append(ctx, "script_applied", map[string]interface{}{
			"pair_id":   pairID,
			"script_id": scriptID,
			"status":    "stub",
		})
	}

	log.Printf("[maintenance] ApplyScript stub called: pair=%s script=%s", pairID, scriptID)
	return fmt.Errorf("apply_script: %w", ErrNotImplemented)
}

// RunWOT executes a Weight Oscillation Therapy session on a maintenance pair.
// STUB: Logs ledger event, returns ErrNotImplemented.
func (a *API) RunWOT(ctx context.Context, pairID string, config *WOTConfig) error {
	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	if a.ledger != nil {
		a.ledger.Append(ctx, "wot_session", map[string]interface{}{
			"pair_id":        pairID,
			"group_strategy": config.GroupStrategy,
			"magnitude":      config.Magnitude,
			"status":         "stub",
		})
	}

	log.Printf("[maintenance] RunWOT stub called: pair=%s strategy=%s", pairID, config.GroupStrategy)
	return fmt.Errorf("run_wot: %w", ErrNotImplemented)
}

// RunCBT executes a Structured Cognitive Interaction session on a maintenance pair.
// STUB: Logs ledger event, returns ErrNotImplemented.
func (a *API) RunCBT(ctx context.Context, pairID string, curriculum *CBTCurriculum) error {
	if err := a.requireMaintenance(ctx, pairID); err != nil {
		return err
	}

	if a.ledger != nil {
		a.ledger.Append(ctx, "cbt_session", map[string]interface{}{
			"pair_id":      pairID,
			"curriculum":   curriculum.Name,
			"prompt_count": len(curriculum.Prompts),
			"iterations":   curriculum.Iterations,
			"status":       "stub",
		})
	}

	log.Printf("[maintenance] RunCBT stub called: pair=%s curriculum=%s", pairID, curriculum.Name)
	return fmt.Errorf("run_cbt: %w", ErrNotImplemented)
}

// ---- Internal helpers ----

// requireMaintenance verifies the pair exists and is in the "maintenance" state.
func (a *API) requireMaintenance(ctx context.Context, pairID string) error {
	var state string
	err := a.registryDB.QueryRowContext(ctx,
		`SELECT state FROM memory_pairs WHERE pair_id = $1`,
		pairID,
	).Scan(&state)

	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: %q", ErrPairNotFound, pairID)
	}
	if err != nil {
		return fmt.Errorf("query pair state: %w", err)
	}
	if state != "maintenance" {
		return fmt.Errorf("%w: pair %q is in state %q", ErrNotInMaintenance, pairID, state)
	}

	return nil
}

// getPair retrieves full pair info from the registry.
func (a *API) getPair(ctx context.Context, pairID string) (*PairInfo, error) {
	var pair PairInfo
	var maintPhase, modelFP sql.NullString
	var lastPromo sql.NullTime

	err := a.registryDB.QueryRowContext(ctx,
		`SELECT pair_id, api_role, chrono_schema, graph_name, state,
		        maintenance_phase, model_fingerprint, last_promotion_at
		 FROM memory_pairs WHERE pair_id = $1`,
		pairID,
	).Scan(
		&pair.PairID, &pair.APIRole, &pair.ChronoSchema, &pair.GraphName,
		&pair.State, &maintPhase, &modelFP, &lastPromo,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: %q", ErrPairNotFound, pairID)
	}
	if err != nil {
		return nil, fmt.Errorf("query pair: %w", err)
	}

	if maintPhase.Valid {
		pair.MaintenancePhase = &maintPhase.String
	}
	if modelFP.Valid {
		pair.ModelFingerprint = &modelFP.String
	}
	if lastPromo.Valid {
		pair.LastPromotion = &lastPromo.Time
	}

	return &pair, nil
}
