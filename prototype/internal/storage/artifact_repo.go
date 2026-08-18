package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Artifact represents a generated artifact
type Artifact struct {
	ID           string          `json:"id"`
	ArtifactID   string          `json:"artifact_id"`
	TicketID     string          `json:"ticket_id"`
	RunID        string          `json:"run_id,omitempty"`
	ArtifactType string          `json:"artifact_type"`
	Hash         string          `json:"hash"`
	SchemaRef    string          `json:"schema_ref,omitempty"`
	Provenance   json.RawMessage `json:"provenance"`
	StorageURI   string          `json:"storage_uri"`
	SizeBytes    int64           `json:"size_bytes"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Provenance tracks the origin of an artifact
type Provenance struct {
	TicketID     string   `json:"ticket_id"`
	PluginID     string   `json:"plugin_id"`
	InputsHash   string   `json:"inputs_hash"`
	AlphaRef     string   `json:"alpha_ref"`
	BetaRef      string   `json:"beta_ref,omitempty"`
	RulebookHash string   `json:"rulebook_hash"`
	GatesRun     []string `json:"gates_run"`
}

// ArtifactRepo handles artifact persistence
type ArtifactRepo struct {
	pool *pgxpool.Pool
	blob BlobRepository
}

// NewArtifactRepo creates a new artifact repository
func NewArtifactRepo(pool *pgxpool.Pool, blob BlobRepository) *ArtifactRepo {
	return &ArtifactRepo{pool: pool, blob: blob}
}

// Create inserts a new artifact and stores its blob
func (r *ArtifactRepo) Create(ctx context.Context, a *Artifact, data []byte, contentType string) error {
	// Store blob first
	storageURI, err := r.blob.PutBlob(ctx, a.Hash, data, contentType)
	if err != nil {
		return err
	}
	a.StorageURI = storageURI
	a.SizeBytes = int64(len(data))

	// Insert metadata
	query := `
		INSERT INTO artifacts (id, artifact_id, ticket_id, run_id, artifact_type, hash, schema_ref, provenance, storage_uri, size_bytes, created_at)
		VALUES ($1, $2, (SELECT id FROM tickets WHERE ticket_id = $3), (SELECT id FROM runs WHERE run_id = $4), $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = r.pool.Exec(ctx, query,
		a.ID, a.ArtifactID, a.TicketID, a.RunID, a.ArtifactType,
		a.Hash, a.SchemaRef, a.Provenance, a.StorageURI, a.SizeBytes, a.CreatedAt,
	)
	return err
}

// GetByHash retrieves an artifact by its content hash
func (r *ArtifactRepo) GetByHash(ctx context.Context, hash string) (*Artifact, error) {
	query := `
		SELECT a.id, a.artifact_id, t.ticket_id, COALESCE(r.run_id, ''), a.artifact_type, 
			   a.hash, a.schema_ref, a.provenance, a.storage_uri, a.size_bytes, a.created_at
		FROM artifacts a
		JOIN tickets t ON a.ticket_id = t.id
		LEFT JOIN runs r ON a.run_id = r.id
		WHERE a.hash = $1
	`
	row := r.pool.QueryRow(ctx, query, hash)

	var a Artifact
	err := row.Scan(
		&a.ID, &a.ArtifactID, &a.TicketID, &a.RunID, &a.ArtifactType,
		&a.Hash, &a.SchemaRef, &a.Provenance, &a.StorageURI, &a.SizeBytes, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetBlob retrieves the artifact blob data
func (r *ArtifactRepo) GetBlob(ctx context.Context, hash string) ([]byte, error) {
	return r.blob.GetBlob(ctx, hash)
}

// ListByTicket retrieves all artifacts for a ticket
func (r *ArtifactRepo) ListByTicket(ctx context.Context, ticketID string) ([]Artifact, error) {
	query := `
		SELECT a.id, a.artifact_id, t.ticket_id, COALESCE(r.run_id, ''), a.artifact_type,
			   a.hash, a.schema_ref, a.provenance, a.storage_uri, a.size_bytes, a.created_at
		FROM artifacts a
		JOIN tickets t ON a.ticket_id = t.id
		LEFT JOIN runs r ON a.run_id = r.id
		WHERE t.ticket_id = $1
		ORDER BY a.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(
			&a.ID, &a.ArtifactID, &a.TicketID, &a.RunID, &a.ArtifactType,
			&a.Hash, &a.SchemaRef, &a.Provenance, &a.StorageURI, &a.SizeBytes, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// ExistsByHash checks if an artifact with the given hash exists (for uniqueness gate)
func (r *ArtifactRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM artifacts WHERE hash = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, hash).Scan(&exists)
	return exists, err
}
