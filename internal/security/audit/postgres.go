package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAuditor implements Auditor using Postgres
type PostgresAuditor struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditor creates a new Postgres auditor
func NewPostgresAuditor(pool *pgxpool.Pool) *PostgresAuditor {
	return &PostgresAuditor{pool: pool}
}

func (a *PostgresAuditor) Log(ctx context.Context, entry LogEntry) error {
	query := `
		INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, resource_id, changes, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	// Ensure defaults if missing
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	_, err := a.pool.Exec(ctx, query,
		entry.TenantID,
		entry.ActorID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		entry.Changes,
		entry.Metadata,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}
	return nil
}

func (a *PostgresAuditor) ListByResource(ctx context.Context, tenantID, resourceType, resourceID string) ([]LogEntry, error) {
	query := `
		SELECT id, tenant_id, actor_id, action, resource_type, resource_id, changes, metadata, created_at
		FROM audit_logs
		WHERE tenant_id = $1 AND resource_type = $2 AND resource_id = $3
		ORDER BY created_at DESC
	`

	rows, err := a.pool.Query(ctx, query, tenantID, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		// changes/metadata are JSONB, likely need careful scanning or intermediate map
		// pgx handles map[string]interface{} for JSONB automatically usually
		err := rows.Scan(
			&l.ID, &l.TenantID, &l.ActorID, &l.Action,
			&l.ResourceType, &l.ResourceID, &l.Changes, &l.Metadata, &l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
