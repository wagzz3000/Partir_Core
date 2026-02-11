// Package storage provides the storage layer for Partir Core.
// It includes Postgres repositories and S3/MinIO blob storage.
package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds storage configuration
type Config struct {
	PostgresURL string
	MinioURL    string
	MinioBucket string
	MinioUser   string
	MinioPass   string
}

// Storage is the main storage layer combining Postgres and S3
type Storage struct {
	Pool *pgxpool.Pool
	Blob *BlobStore

	Rulebooks *RulebookRepo
	Tickets   *TicketRepo
	Runs      *RunRepo
	Artifacts *ArtifactRepo
	Defects   *DefectRepo
}

// New creates a new Storage instance
func New(ctx context.Context, cfg Config) (*Storage, error) {
	// Connect to Postgres
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Create blob store
	blob, err := NewBlobStore(cfg.MinioURL, cfg.MinioBucket, cfg.MinioUser, cfg.MinioPass)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob store: %w", err)
	}

	s := &Storage{
		Pool: pool,
		Blob: blob,
	}

	// Initialize repositories
	s.Rulebooks = NewRulebookRepo(pool)
	s.Tickets = NewTicketRepo(pool)
	s.Runs = NewRunRepo(pool)
	s.Artifacts = NewArtifactRepo(pool, blob)
	s.Defects = NewDefectRepo(pool)

	return s, nil
}

// Close closes all storage connections
func (s *Storage) Close() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}

// Ping checks storage connectivity
func (s *Storage) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}

// UUIDv7 generates a time-ordered UUID v7
func UUIDv7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 if V7 generation fails
		return uuid.New().String()
	}
	return id.String()
}
