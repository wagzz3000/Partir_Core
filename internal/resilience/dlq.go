package resilience

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DLQEntry represents a dead letter queue entry
type DLQEntry struct {
	ID         string          `json:"id"`
	TicketID   string          `json:"ticket_id"`
	TenantID   string          `json:"tenant_id"`
	Reason     string          `json:"reason"`
	LastError  string          `json:"last_error"`
	Attempts   int             `json:"attempts"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
	ResolvedAt *time.Time      `json:"resolved_at,omitempty"`
}

// DeadLetterQueue stores permanently failed tickets for manual inspection
type DeadLetterQueue struct {
	pool *pgxpool.Pool
}

// NewDeadLetterQueue creates a new DLQ
func NewDeadLetterQueue(pool *pgxpool.Pool) *DeadLetterQueue {
	return &DeadLetterQueue{pool: pool}
}

// Enqueue adds a failed ticket to the dead letter queue
func (dlq *DeadLetterQueue) Enqueue(ctx context.Context, entry DLQEntry) error {
	query := `
		INSERT INTO tickets_dlq (ticket_id, tenant_id, reason, last_error, attempts, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	_, err := dlq.pool.Exec(ctx, query,
		entry.TicketID, entry.TenantID, entry.Reason, entry.LastError,
		entry.Attempts, entry.Payload, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue to DLQ: %w", err)
	}
	return nil
}

// List returns unresolved DLQ entries for a tenant
func (dlq *DeadLetterQueue) List(ctx context.Context, tenantID string) ([]DLQEntry, error) {
	query := `
		SELECT id, ticket_id, tenant_id, reason, last_error, attempts, payload, created_at, resolved_at
		FROM tickets_dlq
		WHERE tenant_id = $1 AND resolved_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := dlq.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ: %w", err)
	}
	defer rows.Close()

	var entries []DLQEntry
	for rows.Next() {
		var e DLQEntry
		if err := rows.Scan(&e.ID, &e.TicketID, &e.TenantID, &e.Reason,
			&e.LastError, &e.Attempts, &e.Payload, &e.CreatedAt, &e.ResolvedAt); err != nil {
			return nil, fmt.Errorf("failed to scan DLQ entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Resolve marks a DLQ entry as resolved
func (dlq *DeadLetterQueue) Resolve(ctx context.Context, id string) error {
	query := `UPDATE tickets_dlq SET resolved_at = NOW() WHERE id = $1`
	_, err := dlq.pool.Exec(ctx, query, id)
	return err
}
