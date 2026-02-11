package factory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWorkerNotFound  = errors.New("worker not found")
	ErrStationNotFound = errors.New("station not found")
	ErrNoWorkerAvail   = errors.New("no available worker for station type")
	ErrWorkerAssigned  = errors.New("worker already assigned to a station")
)

// Registry manages workers and their station assignments
type Registry struct {
	db *pgxpool.Pool
	mu sync.RWMutex
}

// NewRegistry creates a new factory registry
func NewRegistry(db *pgxpool.Pool) *Registry {
	return &Registry{db: db}
}

// RegisterWorker adds a new worker to the pool
func (r *Registry) RegisterWorker(ctx context.Context, worker *Worker) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	query := `
		INSERT INTO workers (id, type, endpoint, status, assigned_station, assigned_slot, model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			endpoint = EXCLUDED.endpoint,
			status = EXCLUDED.status,
			model_fingerprint = EXCLUDED.model_fingerprint,
			memory_binding_id = EXCLUDED.memory_binding_id,
			memory_namespace = EXCLUDED.memory_namespace,
			last_heartbeat = EXCLUDED.last_heartbeat
		RETURNING assigned_station, assigned_slot
	`

	var station, slot sql.NullString
	err := r.db.QueryRow(ctx, query,
		worker.ID,
		worker.Type,
		worker.Endpoint,
		worker.Status,
		worker.AssignedStation,
		worker.AssignedSlot,
		worker.ModelFingerprint,
		worker.MemoryBindingID,
		worker.MemoryNamespace,
		worker.LastHeartbeat,
		time.Now(),
	).Scan(&station, &slot)

	if err != nil {
		return err
	}

	if station.Valid {
		worker.AssignedStation = station.String
	}
	if slot.Valid {
		worker.AssignedSlot = slot.String
	}

	return nil
}

// UpdateHeartbeat updates a worker's heartbeat timestamp
func (r *Registry) UpdateHeartbeat(ctx context.Context, workerID string) error {
	query := `UPDATE workers SET last_heartbeat = $1, status = 'active' WHERE id = $2`
	result, err := r.db.Exec(ctx, query, time.Now(), workerID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWorkerNotFound
	}
	return nil
}

// GetWorker retrieves a worker by ID
func (r *Registry) GetWorker(ctx context.Context, workerID string) (*Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
		       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
		FROM workers WHERE id = $1`
	row := r.db.QueryRow(ctx, query, workerID)

	var w Worker
	var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
	var lastHeartbeat sql.NullTime

	err := row.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
		&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrWorkerNotFound
	}
	if err != nil {
		return nil, err
	}

	if assignedStation.Valid {
		w.AssignedStation = assignedStation.String
	}
	if assignedSlot.Valid {
		w.AssignedSlot = assignedSlot.String
	}
	if modelFingerprint.Valid {
		w.ModelFingerprint = modelFingerprint.String
	}
	if memoryBindingID.Valid {
		w.MemoryBindingID = &memoryBindingID.String
	}
	if memoryNamespace.Valid {
		w.MemoryNamespace = &memoryNamespace.String
	}
	if lastHeartbeat.Valid {
		w.LastHeartbeat = &lastHeartbeat.Time
	}

	return &w, nil
}

// ListWorkers returns all workers
func (r *Registry) ListWorkers(ctx context.Context) ([]Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
		       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
		FROM workers ORDER BY type, id`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		var w Worker
		var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
		var lastHeartbeat sql.NullTime

		err := rows.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
			&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
		if err != nil {
			return nil, err
		}

		if assignedStation.Valid {
			w.AssignedStation = assignedStation.String
		}
		if assignedSlot.Valid {
			w.AssignedSlot = assignedSlot.String
		}
		if modelFingerprint.Valid {
			w.ModelFingerprint = modelFingerprint.String
		}
		if memoryBindingID.Valid {
			w.MemoryBindingID = &memoryBindingID.String
		}
		if memoryNamespace.Valid {
			w.MemoryNamespace = &memoryNamespace.String
		}
		if lastHeartbeat.Valid {
			w.LastHeartbeat = &lastHeartbeat.Time
		}

		workers = append(workers, w)
	}

	return workers, nil
}

// GetAvailableWorker finds an unassigned worker of the given type
func (r *Registry) GetAvailableWorker(ctx context.Context, workerType string) (*Worker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
		       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
		FROM workers 
		WHERE type = $1 AND (assigned_station IS NULL OR assigned_station = '') AND status != 'crashed'
		ORDER BY last_heartbeat DESC NULLS LAST
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, query, workerType)

	var w Worker
	var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
	var lastHeartbeat sql.NullTime

	err := row.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
		&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNoWorkerAvail
	}
	if err != nil {
		return nil, err
	}

	if lastHeartbeat.Valid {
		w.LastHeartbeat = &lastHeartbeat.Time
	}

	return &w, nil
}

// AssignWorkerToStation assigns a worker to a station slot
func (r *Registry) AssignWorkerToStation(ctx context.Context, stationID StationID, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Determine next available slot for station
	// Determine prefix based on station (A, B, O)
	prefix := "A"
	switch stationID {
	case StationBeta:
		prefix = "B"
	case StationOmega:
		prefix = "O"
	}

	// Find used slots
	rows, err := tx.Query(ctx, `SELECT assigned_slot FROM workers WHERE assigned_station = $1 AND assigned_slot IS NOT NULL`, string(stationID))
	if err != nil {
		return err
	}
	defer rows.Close()

	usedSlots := make(map[string]bool)
	for rows.Next() {
		var slot string
		if err := rows.Scan(&slot); err == nil {
			usedSlots[slot] = true
		}
	}

	// assign first available
	var slotID string
	for i := 1; i <= 5; i++ {
		trySlot := fmt.Sprintf("%s%d", prefix, i)
		if !usedSlots[trySlot] {
			slotID = trySlot
			break
		}
	}

	if slotID == "" {
		return fmt.Errorf("no slots available in station %s", stationID)
	}

	// 2. Update worker
	workerQuery := `UPDATE workers SET assigned_station = $1, assigned_slot = $2, status = 'active' WHERE id = $3`
	_, err = tx.Exec(ctx, workerQuery, string(stationID), slotID, workerID)
	if err != nil {
		return err
	}

	// 3. Update station status (Stations table no longer has worker_id)
	stationQuery := `
		INSERT INTO stations (id, status, created_at, updated_at)
		VALUES ($1, 'active', NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET status = 'active', updated_at = NOW()
	`
	_, err = tx.Exec(ctx, stationQuery, string(stationID))
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UnassignWorkerFromStation removes a worker from their station
// NOTE: This signature previously took stationID. It should arguably take workerID or handle one worker.
// To maintain compatibility but support slots, we'll unassign specific workerID if passed?
// The current interface uses stationID.
// But now a station has multiple workers.
// If we call this, do we unassign ALL workers? That seems dangerous.
// Let's modify it to unassign the specific worker associated with the context or assume a generic "unassign last"?
// Better: UnassignWorker(workerID).
// BUT existing callers might use UnassignWorkerFromStation(stationID).
// I will change the signature if I can.
// Checking callers... likely mostly CLI and tests.
// Refactoring to UnassignWorker(workerID) is cleaner.
func (r *Registry) UnassignWorker(ctx context.Context, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(ctx, `UPDATE workers SET assigned_station = NULL, assigned_slot = NULL, status = 'sleeping' WHERE id = $1`, workerID)
	return err
}

// GetStation retrieves a station by ID with its assigned workers
func (r *Registry) GetStation(ctx context.Context, stationID StationID) (*Station, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `SELECT id, status, created_at, updated_at FROM stations WHERE id = $1`
	row := r.db.QueryRow(ctx, query, string(stationID))

	var s Station
	err := row.Scan(&s.ID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrStationNotFound
	}
	if err != nil {
		return nil, err
	}

	// Fetch assigned workers
	workersQuery := `
		SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
		       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
		FROM workers WHERE assigned_station = $1 ORDER BY assigned_slot`
	rows, err := r.db.Query(ctx, workersQuery, string(stationID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var w Worker
		var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
		var lastHeartbeat sql.NullTime

		err := rows.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
			&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
		if err != nil {
			continue
		}

		if assignedStation.Valid {
			w.AssignedStation = assignedStation.String
		}
		if assignedSlot.Valid {
			w.AssignedSlot = assignedSlot.String
		}
		if modelFingerprint.Valid {
			w.ModelFingerprint = modelFingerprint.String
		}
		if memoryBindingID.Valid {
			w.MemoryBindingID = &memoryBindingID.String
		}
		if memoryNamespace.Valid {
			w.MemoryNamespace = &memoryNamespace.String
		}
		if lastHeartbeat.Valid {
			w.LastHeartbeat = &lastHeartbeat.Time
		}

		s.Workers = append(s.Workers, w)
	}

	return &s, nil
}

// ListStations returns all stations with their workers
func (r *Registry) ListStations(ctx context.Context) ([]Station, error) {
	// First get all stations
	stationsQuery := `SELECT id, status, created_at, updated_at FROM stations ORDER BY id`
	rows, err := r.db.Query(ctx, stationsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stations []Station
	for rows.Next() {
		var s Station
		err := rows.Scan(&s.ID, &s.Status, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		stations = append(stations, s)
	}
	rows.Close() // Ensure closed before next query

	// Then populate workers for each
	for i := range stations {
		s := &stations[i]
		workersQuery := `
			SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
			       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
			FROM workers WHERE assigned_station = $1 ORDER BY assigned_slot`
		wRows, err := r.db.Query(ctx, workersQuery, string(s.ID))
		if err != nil {
			continue
		}

		for wRows.Next() {
			var w Worker
			var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
			var lastHeartbeat sql.NullTime

			err := wRows.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
				&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
			if err == nil {
				if assignedStation.Valid {
					w.AssignedStation = assignedStation.String
				}
				if assignedSlot.Valid {
					w.AssignedSlot = assignedSlot.String
				}
				if modelFingerprint.Valid {
					w.ModelFingerprint = modelFingerprint.String
				}
				if memoryBindingID.Valid {
					w.MemoryBindingID = &memoryBindingID.String
				}
				if memoryNamespace.Valid {
					w.MemoryNamespace = &memoryNamespace.String
				}
				if lastHeartbeat.Valid {
					w.LastHeartbeat = &lastHeartbeat.Time
				}

				s.Workers = append(s.Workers, w)
			}
		}
		wRows.Close()
	}

	return stations, nil
}

// EnsureStationsExist creates default stations if they don't exist
func (r *Registry) EnsureStationsExist(ctx context.Context) error {
	for _, stationID := range AllStations() {
		query := `
			INSERT INTO stations (id, status, created_at, updated_at)
			VALUES ($1, 'idle', NOW(), NOW())
			ON CONFLICT (id) DO NOTHING
		`
		_, err := r.db.Exec(ctx, query, string(stationID))
		if err != nil {
			return fmt.Errorf("failed to create station %s: %w", stationID, err)
		}
	}
	return nil
}

// DetectCrashedWorkers marks workers as crashed if no heartbeat within timeout
func (r *Registry) DetectCrashedWorkers(ctx context.Context, timeout time.Duration) ([]Worker, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-timeout)

	// Find crashed workers
	query := `
		SELECT id, type, endpoint, status, assigned_station, assigned_slot, 
		       model_fingerprint, memory_binding_id, memory_namespace, last_heartbeat, created_at 
		FROM workers 
		WHERE status IN ('active', 'hot_standby') AND last_heartbeat < $1
	`
	rows, err := r.db.Query(ctx, query, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crashed []Worker
	for rows.Next() {
		var w Worker
		var assignedStation, assignedSlot, modelFingerprint, memoryBindingID, memoryNamespace sql.NullString
		var lastHeartbeat sql.NullTime

		err := rows.Scan(&w.ID, &w.Type, &w.Endpoint, &w.Status, &assignedStation, &assignedSlot,
			&modelFingerprint, &memoryBindingID, &memoryNamespace, &lastHeartbeat, &w.CreatedAt)
		if err != nil {
			return nil, err
		}
		// Populate struct... skipping for brevity as we just need ID for update
		crashed = append(crashed, w)
	}
	rows.Close()

	// Mark them as crashed
	if len(crashed) > 0 {
		_, err = r.db.Exec(ctx, `UPDATE workers SET status = 'crashed' WHERE status IN ('active', 'hot_standby') AND last_heartbeat < $1`, cutoff)
		if err != nil {
			return nil, err
		}
	}

	return crashed, nil
}

// GetFactoryStatus returns the overall factory status
func (r *Registry) GetFactoryStatus(ctx context.Context) (*FactoryStatus, error) {
	stations, err := r.ListStations(ctx)
	if err != nil {
		return nil, err
	}

	workers, err := r.ListWorkers(ctx)
	if err != nil {
		return nil, err
	}

	status := &FactoryStatus{
		Stations:     stations,
		Workers:      workers,
		TotalWorkers: len(workers),
	}

	for _, w := range workers {
		switch w.Status {
		case WorkerStatusActive:
			status.ActiveWorkers++
		case WorkerStatusCrashed:
			status.CrashedWorkers++
		}
	}

	return status, nil
}
