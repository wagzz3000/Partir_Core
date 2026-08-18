package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Rulebook represents an Alpha or Beta rulebook in the registry
type Rulebook struct {
	ID            string          `json:"id"`
	RulebookType  string          `json:"rulebook_type"` // "alpha" or "beta"
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Hash          string          `json:"hash"`
	Manifest      json.RawMessage `json:"manifest"`
	Compat        json.RawMessage `json:"compat,omitempty"`
	ParentVersion string          `json:"parent_version,omitempty"`
	StorageURI    string          `json:"storage_uri,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// RulebookRepo handles rulebook persistence
type RulebookRepo struct {
	pool *pgxpool.Pool
}

// NewRulebookRepo creates a new rulebook repository
func NewRulebookRepo(pool *pgxpool.Pool) *RulebookRepo {
	return &RulebookRepo{pool: pool}
}

// Create inserts a new rulebook
func (r *RulebookRepo) Create(ctx context.Context, rb *Rulebook) error {
	query := `
		INSERT INTO rulebooks (id, rulebook_type, name, version, hash, manifest, compat, parent_version, storage_uri, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		rb.ID, rb.RulebookType, rb.Name, rb.Version, rb.Hash,
		rb.Manifest, rb.Compat, rb.ParentVersion, rb.StorageURI, rb.CreatedAt,
	)
	return err
}

// GetByRef retrieves a rulebook by name@version reference
func (r *RulebookRepo) GetByRef(ctx context.Context, rulebookType, name, version string) (*Rulebook, error) {
	query := `
		SELECT id, rulebook_type, name, version, hash, manifest, compat, parent_version, storage_uri, created_at
		FROM rulebooks
		WHERE rulebook_type = $1 AND name = $2 AND version = $3
	`
	row := r.pool.QueryRow(ctx, query, rulebookType, name, version)

	var rb Rulebook
	err := row.Scan(
		&rb.ID, &rb.RulebookType, &rb.Name, &rb.Version, &rb.Hash,
		&rb.Manifest, &rb.Compat, &rb.ParentVersion, &rb.StorageURI, &rb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rb, nil
}

// GetByHash retrieves a rulebook by its content hash
func (r *RulebookRepo) GetByHash(ctx context.Context, hash string) (*Rulebook, error) {
	query := `
		SELECT id, rulebook_type, name, version, hash, manifest, compat, parent_version, storage_uri, created_at
		FROM rulebooks
		WHERE hash = $1
	`
	row := r.pool.QueryRow(ctx, query, hash)

	var rb Rulebook
	err := row.Scan(
		&rb.ID, &rb.RulebookType, &rb.Name, &rb.Version, &rb.Hash,
		&rb.Manifest, &rb.Compat, &rb.ParentVersion, &rb.StorageURI, &rb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rb, nil
}

// List retrieves all rulebooks of a given type
func (r *RulebookRepo) List(ctx context.Context, rulebookType string) ([]Rulebook, error) {
	query := `
		SELECT id, rulebook_type, name, version, hash, manifest, compat, parent_version, storage_uri, created_at
		FROM rulebooks
		WHERE rulebook_type = $1
		ORDER BY name, version DESC
	`
	rows, err := r.pool.Query(ctx, query, rulebookType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Rulebook
	for rows.Next() {
		var rb Rulebook
		if err := rows.Scan(
			&rb.ID, &rb.RulebookType, &rb.Name, &rb.Version, &rb.Hash,
			&rb.Manifest, &rb.Compat, &rb.ParentVersion, &rb.StorageURI, &rb.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, rb)
	}
	return result, rows.Err()
}

// ListVersions retrieves all versions of a rulebook by name
func (r *RulebookRepo) ListVersions(ctx context.Context, rulebookType, name string) ([]Rulebook, error) {
	query := `
		SELECT id, rulebook_type, name, version, hash, manifest, compat, parent_version, storage_uri, created_at
		FROM rulebooks
		WHERE rulebook_type = $1 AND name = $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, rulebookType, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Rulebook
	for rows.Next() {
		var rb Rulebook
		if err := rows.Scan(
			&rb.ID, &rb.RulebookType, &rb.Name, &rb.Version, &rb.Hash,
			&rb.Manifest, &rb.Compat, &rb.ParentVersion, &rb.StorageURI, &rb.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, rb)
	}
	return result, rows.Err()
}

// Exists checks if a rulebook with the given name and version exists
func (r *RulebookRepo) Exists(ctx context.Context, rulebookType, name, version string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM rulebooks WHERE rulebook_type = $1 AND name = $2 AND version = $3)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, rulebookType, name, version).Scan(&exists)
	return exists, err
}
