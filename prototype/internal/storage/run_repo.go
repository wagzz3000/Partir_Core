package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run represents an execution attempt for a ticket
type Run struct {
	ID         string          `json:"id"`
	RunID      string          `json:"run_id"`
	TicketID   string          `json:"ticket_id"`
	Attempt    int             `json:"attempt"`
	Executor   string          `json:"executor"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	Status     string          `json:"status"` // running, completed, failed
	CostTokens int             `json:"cost_tokens"`
	CostUSD    float64         `json:"cost_usd"`
	LogsURI    string          `json:"logs_uri,omitempty"`
	Metadata   json.RawMessage `json:"metadata"`
}

// RunRepo handles run persistence
type RunRepo struct {
	pool *pgxpool.Pool
}

// NewRunRepo creates a new run repository
func NewRunRepo(pool *pgxpool.Pool) *RunRepo {
	return &RunRepo{pool: pool}
}

// Create inserts a new run
func (r *RunRepo) Create(ctx context.Context, run *Run) error {
	query := `
		INSERT INTO runs (id, run_id, ticket_id, attempt, executor, started_at, status, cost_tokens, cost_usd, logs_uri, metadata)
		VALUES ($1, $2, (SELECT id FROM tickets WHERE ticket_id = $3), $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		run.ID, run.RunID, run.TicketID, run.Attempt, run.Executor,
		run.StartedAt, run.Status, run.CostTokens, run.CostUSD, run.LogsURI, run.Metadata,
	)
	return err
}

// Complete marks a run as completed
func (r *RunRepo) Complete(ctx context.Context, runID string, status string, costTokens int, costUSD float64, logsURI string) error {
	query := `
		UPDATE runs 
		SET finished_at = $1, status = $2, cost_tokens = $3, cost_usd = $4, logs_uri = $5
		WHERE run_id = $6
	`
	now := time.Now()
	_, err := r.pool.Exec(ctx, query, now, status, costTokens, costUSD, logsURI, runID)
	return err
}

// GetByRunID retrieves a run by its run_id
func (r *RunRepo) GetByRunID(ctx context.Context, runID string) (*Run, error) {
	query := `
		SELECT r.id, r.run_id, t.ticket_id, r.attempt, r.executor, r.started_at, r.finished_at, 
			   r.status, r.cost_tokens, r.cost_usd, r.logs_uri, r.metadata
		FROM runs r
		JOIN tickets t ON r.ticket_id = t.id
		WHERE r.run_id = $1
	`
	row := r.pool.QueryRow(ctx, query, runID)

	var run Run
	err := row.Scan(
		&run.ID, &run.RunID, &run.TicketID, &run.Attempt, &run.Executor,
		&run.StartedAt, &run.FinishedAt, &run.Status, &run.CostTokens,
		&run.CostUSD, &run.LogsURI, &run.Metadata,
	)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// ListByTicket retrieves all runs for a ticket
func (r *RunRepo) ListByTicket(ctx context.Context, ticketID string) ([]Run, error) {
	query := `
		SELECT r.id, r.run_id, t.ticket_id, r.attempt, r.executor, r.started_at, r.finished_at,
			   r.status, r.cost_tokens, r.cost_usd, r.logs_uri, r.metadata
		FROM runs r
		JOIN tickets t ON r.ticket_id = t.id
		WHERE t.ticket_id = $1
		ORDER BY r.attempt ASC
	`
	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(
			&run.ID, &run.RunID, &run.TicketID, &run.Attempt, &run.Executor,
			&run.StartedAt, &run.FinishedAt, &run.Status, &run.CostTokens,
			&run.CostUSD, &run.LogsURI, &run.Metadata,
		); err != nil {
			return nil, err
		}
		result = append(result, run)
	}
	return result, rows.Err()
}

// CountAttempts returns the number of attempts for a ticket
func (r *RunRepo) CountAttempts(ctx context.Context, ticketID string) (int, error) {
	query := `
		SELECT COUNT(*) FROM runs r
		JOIN tickets t ON r.ticket_id = t.id
		WHERE t.ticket_id = $1
	`
	var count int
	err := r.pool.QueryRow(ctx, query, ticketID).Scan(&count)
	return count, err
}
