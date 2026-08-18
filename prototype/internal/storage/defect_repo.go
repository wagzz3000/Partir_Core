package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Defect represents a QC failure for DMAIC analysis
type Defect struct {
	ID              string          `json:"id"`
	DefectID        string          `json:"defect_id"`
	RunID           string          `json:"run_id"`
	GateID          string          `json:"gate_id"`
	DefectClass     string          `json:"defect_class"`
	PluginID        string          `json:"plugin_id,omitempty"`
	Message         string          `json:"message"`
	OffendingFields json.RawMessage `json:"offending_fields"`
	SuggestedFix    string          `json:"suggested_fix,omitempty"`
	RulebookVersion string          `json:"rulebook_version,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// DefectRepo handles defect persistence for DMAIC analysis
type DefectRepo struct {
	pool *pgxpool.Pool
}

// NewDefectRepo creates a new defect repository
func NewDefectRepo(pool *pgxpool.Pool) *DefectRepo {
	return &DefectRepo{pool: pool}
}

// Create inserts a new defect
func (r *DefectRepo) Create(ctx context.Context, d *Defect) error {
	query := `
		INSERT INTO defects (id, defect_id, run_id, gate_id, defect_class, plugin_id, message, offending_fields, suggested_fix, rulebook_version, created_at)
		VALUES ($1, $2, (SELECT id FROM runs WHERE run_id = $3), $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		d.ID, d.DefectID, d.RunID, d.GateID, d.DefectClass, d.PluginID,
		d.Message, d.OffendingFields, d.SuggestedFix, d.RulebookVersion, d.CreatedAt,
	)
	return err
}

// CreateBatch inserts multiple defects
func (r *DefectRepo) CreateBatch(ctx context.Context, defects []Defect) error {
	for _, d := range defects {
		if err := r.Create(ctx, &d); err != nil {
			return err
		}
	}
	return nil
}

// ListByRun retrieves all defects for a run
func (r *DefectRepo) ListByRun(ctx context.Context, runID string) ([]Defect, error) {
	query := `
		SELECT d.id, d.defect_id, r.run_id, d.gate_id, d.defect_class, d.plugin_id,
			   d.message, d.offending_fields, d.suggested_fix, d.rulebook_version, d.created_at
		FROM defects d
		JOIN runs r ON d.run_id = r.id
		WHERE r.run_id = $1
		ORDER BY d.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Defect
	for rows.Next() {
		var d Defect
		if err := rows.Scan(
			&d.ID, &d.DefectID, &d.RunID, &d.GateID, &d.DefectClass, &d.PluginID,
			&d.Message, &d.OffendingFields, &d.SuggestedFix, &d.RulebookVersion, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// CountByClass returns defect counts grouped by class (for DMAIC trending)
func (r *DefectRepo) CountByClass(ctx context.Context, since time.Time) (map[string]int, error) {
	query := `
		SELECT defect_class, COUNT(*) as count
		FROM defects
		WHERE created_at >= $1
		GROUP BY defect_class
		ORDER BY count DESC
	`
	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var class string
		var count int
		if err := rows.Scan(&class, &count); err != nil {
			return nil, err
		}
		result[class] = count
	}
	return result, rows.Err()
}

// CountByGate returns defect counts grouped by gate (for DMAIC trending)
func (r *DefectRepo) CountByGate(ctx context.Context, since time.Time) (map[string]int, error) {
	query := `
		SELECT gate_id, COUNT(*) as count
		FROM defects
		WHERE created_at >= $1
		GROUP BY gate_id
		ORDER BY count DESC
	`
	rows, err := r.pool.Query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var gate string
		var count int
		if err := rows.Scan(&gate, &count); err != nil {
			return nil, err
		}
		result[gate] = count
	}
	return result, rows.Err()
}

// HasRepeatedDefectClass checks if a defect class has occurred before for a ticket
// Used for the bounded retry strategy: same defect twice = stop
func (r *DefectRepo) HasRepeatedDefectClass(ctx context.Context, ticketID, defectClass string) (bool, error) {
	query := `
		SELECT COUNT(*) > 1
		FROM defects d
		JOIN runs r ON d.run_id = r.id
		JOIN tickets t ON r.ticket_id = t.id
		WHERE t.ticket_id = $1 AND d.defect_class = $2
	`
	var repeated bool
	err := r.pool.QueryRow(ctx, query, ticketID, defectClass).Scan(&repeated)
	return repeated, err
}
