package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore manages idempotency keys for deduplication
type IdempotencyStore struct {
	pool *pgxpool.Pool

	// In-memory fallback for environments without Postgres
	mu       sync.RWMutex
	memStore map[string]time.Time
}

// NewIdempotencyStore creates a new idempotency store backed by Postgres
func NewIdempotencyStore(pool *pgxpool.Pool) *IdempotencyStore {
	return &IdempotencyStore{pool: pool}
}

// NewMemoryIdempotencyStore creates an in-memory idempotency store (for testing / dev)
func NewMemoryIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		memStore: make(map[string]time.Time),
	}
}

// Check returns true if the key has already been processed (i.e., is a duplicate)
func (s *IdempotencyStore) Check(ctx context.Context, key string) (bool, error) {
	if s.pool != nil {
		return s.checkDB(ctx, key)
	}
	return s.checkMem(key), nil
}

// Record marks a key as processed
func (s *IdempotencyStore) Record(ctx context.Context, key string) error {
	if s.pool != nil {
		return s.recordDB(ctx, key)
	}
	s.recordMem(key)
	return nil
}

// --- Postgres backend ---

func (s *IdempotencyStore) checkDB(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE key = $1)", key,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("idempotency check failed: %w", err)
	}
	return exists, nil
}

func (s *IdempotencyStore) recordDB(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO idempotency_keys (key, created_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING", key,
	)
	if err != nil {
		return fmt.Errorf("idempotency record failed: %w", err)
	}
	return nil
}

// --- In-memory backend ---

func (s *IdempotencyStore) checkMem(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.memStore[key]
	return ok
}

func (s *IdempotencyStore) recordMem(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memStore[key] = time.Now()
}
