package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exporter handles compliance data export
type Exporter struct {
	pool *pgxpool.Pool
}

// NewExporter creates a new compliance exporter
func NewExporter(pool *pgxpool.Pool) *Exporter {
	return &Exporter{pool: pool}
}

// ExportFilter defines criteria for filtering audit logs
type ExportFilter struct {
	TenantID     string
	ActorID      string
	Action       string
	ResourceType string
	StartDate    string // ISO 8601 format
	EndDate      string // ISO 8601 format
}

// ExportJSON writes filtered audit logs as JSON to the writer
func (e *Exporter) ExportJSON(ctx context.Context, w io.Writer, filter ExportFilter) error {
	logs, err := e.queryLogs(ctx, filter)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(logs)
}

// ExportCSV writes filtered audit logs as CSV to the writer
func (e *Exporter) ExportCSV(ctx context.Context, w io.Writer, filter ExportFilter) error {
	logs, err := e.queryLogs(ctx, filter)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Header
	if err := csvWriter.Write([]string{
		"id", "tenant_id", "actor_id", "action",
		"resource_type", "resource_id", "created_at",
	}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, l := range logs {
		if err := csvWriter.Write([]string{
			l.ID, l.TenantID, l.ActorID, l.Action,
			l.ResourceType, l.ResourceID, l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (e *Exporter) queryLogs(ctx context.Context, filter ExportFilter) ([]LogEntry, error) {
	query := `
		SELECT id, tenant_id, actor_id, action, resource_type, resource_id, changes, metadata, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, filter.ActorID)
		argIdx++
	}
	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filter.Action)
		argIdx++
	}
	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filter.ResourceType)
		argIdx++
	}
	if filter.StartDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, filter.EndDate)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := e.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
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
