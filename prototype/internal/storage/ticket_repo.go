package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TicketState represents the lifecycle state of a ticket
type TicketState string

const (
	StateSpecReady         TicketState = "SPEC_READY"
	StateDispatched        TicketState = "DISPATCHED"
	StateInExecution       TicketState = "IN_EXECUTION"
	StateLocalQCFailed     TicketState = "LOCAL_QC_FAILED"
	StateReadyForOmega     TicketState = "READY_FOR_OMEGA"
	StateOmegaFailed       TicketState = "OMEGA_FAILED"
	StateCompleted         TicketState = "COMPLETED"
	StateLineStopEscalated TicketState = "LINE_STOP_ESCALATED"
)

// Ticket represents a spec-based work order
type Ticket struct {
	ID              string          `json:"id"`
	TicketID        string          `json:"ticket_id"`
	Title           string          `json:"title"`
	PluginID        string          `json:"plugin_id"`
	JobType         string          `json:"job_type"`
	AlphaRef        string          `json:"alpha_ref"`
	BetaRef         string          `json:"beta_ref,omitempty"`
	GreenTagRef     string          `json:"green_tag_ref,omitempty"`
	Inputs          json.RawMessage `json:"inputs"`
	Constraints     json.RawMessage `json:"constraints"`
	AcceptanceGates json.RawMessage `json:"acceptance_gates"`
	Limits          json.RawMessage `json:"limits"`
	RoutingHints    json.RawMessage `json:"routing_hints"`
	OutputsExpected json.RawMessage `json:"outputs_expected"`
	State           TicketState     `json:"state"`
	Priority        int             `json:"priority"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TicketLimits represents the limits on a ticket
type TicketLimits struct {
	MaxFilesChanged int     `json:"max_files_changed,omitempty"`
	MaxDiffLines    int     `json:"max_diff_lines,omitempty"`
	MaxAttempts     int     `json:"max_attempts,omitempty"`
	MaxCostBudget   float64 `json:"max_cost_budget,omitempty"` // tokens or $
}

// TicketRepo handles ticket persistence
type TicketRepo struct {
	pool *pgxpool.Pool
}

// NewTicketRepo creates a new ticket repository
func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{pool: pool}
}

// Create inserts a new ticket
func (r *TicketRepo) Create(ctx context.Context, t *Ticket) error {
	query := `
		INSERT INTO tickets (id, ticket_id, title, plugin_id, job_type, alpha_ref, beta_ref, green_tag_ref,
			inputs, constraints, acceptance_gates, limits, routing_hints, outputs_expected, state, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.pool.Exec(ctx, query,
		t.ID, t.TicketID, t.Title, t.PluginID, t.JobType, t.AlphaRef, t.BetaRef, t.GreenTagRef,
		t.Inputs, t.Constraints, t.AcceptanceGates, t.Limits, t.RoutingHints, t.OutputsExpected,
		t.State, t.Priority, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// GetByTicketID retrieves a ticket by its ticket_id
func (r *TicketRepo) GetByTicketID(ctx context.Context, ticketID string) (*Ticket, error) {
	query := `
		SELECT id, ticket_id, title, plugin_id, job_type, alpha_ref, beta_ref, green_tag_ref,
			inputs, constraints, acceptance_gates, limits, routing_hints, outputs_expected, state, priority, created_at, updated_at
		FROM tickets
		WHERE ticket_id = $1
	`
	row := r.pool.QueryRow(ctx, query, ticketID)

	var t Ticket
	err := row.Scan(
		&t.ID, &t.TicketID, &t.Title, &t.PluginID, &t.JobType, &t.AlphaRef, &t.BetaRef, &t.GreenTagRef,
		&t.Inputs, &t.Constraints, &t.AcceptanceGates, &t.Limits, &t.RoutingHints, &t.OutputsExpected,
		&t.State, &t.Priority, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateState transitions a ticket to a new state
func (r *TicketRepo) UpdateState(ctx context.Context, ticketID string, state TicketState) error {
	query := `UPDATE tickets SET state = $1, updated_at = $2 WHERE ticket_id = $3`
	_, err := r.pool.Exec(ctx, query, state, time.Now(), ticketID)
	return err
}

// ListByState retrieves all tickets in a given state
func (r *TicketRepo) ListByState(ctx context.Context, state TicketState) ([]Ticket, error) {
	query := `
		SELECT id, ticket_id, title, plugin_id, job_type, alpha_ref, beta_ref, green_tag_ref,
			inputs, constraints, acceptance_gates, limits, routing_hints, outputs_expected, state, priority, created_at, updated_at
		FROM tickets
		WHERE state = $1
		ORDER BY priority DESC, created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(
			&t.ID, &t.TicketID, &t.Title, &t.PluginID, &t.JobType, &t.AlphaRef, &t.BetaRef, &t.GreenTagRef,
			&t.Inputs, &t.Constraints, &t.AcceptanceGates, &t.Limits, &t.RoutingHints, &t.OutputsExpected,
			&t.State, &t.Priority, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}
