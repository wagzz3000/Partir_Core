package factory

import (
	"time"
)

// StationID represents a station in the factory
type StationID string

const (
	StationAlpha StationID = "alpha"
	StationBeta  StationID = "beta"
	StationOmega StationID = "omega"
)

// AllStations returns all station IDs
func AllStations() []StationID {
	return []StationID{StationAlpha, StationBeta, StationOmega}
}

// WorkerStatus represents the current state of a worker
type WorkerStatus string

const (
	WorkerStatusActive      WorkerStatus = "active"
	WorkerStatusSleeping    WorkerStatus = "sleeping"
	WorkerStatusCrashed     WorkerStatus = "crashed"
	WorkerStatusStarting    WorkerStatus = "starting"
	WorkerStatusHotStandby  WorkerStatus = "hot_standby"
	WorkerStatusTesting     WorkerStatus = "testing"
	WorkerStatusQuarantined WorkerStatus = "quarantined"
	WorkerStatusMaintenance WorkerStatus = "maintenance"
)

// StationStatus represents the current state of a station
type StationStatus string

const (
	StationStatusActive      StationStatus = "active"
	StationStatusIdle        StationStatus = "idle"
	StationStatusMaintenance StationStatus = "maintenance"
)

// Worker represents a worker instance in the factory
type Worker struct {
	ID              string       `json:"id" db:"id"`
	Type            string       `json:"type" db:"type"`                         // alpha, beta, omega
	Endpoint        string       `json:"endpoint" db:"endpoint"`                 // http://alpha-worker:8090
	Status          WorkerStatus `json:"status" db:"status"`                     // active, sleeping, crashed...
	AssignedStation string       `json:"assigned_station" db:"assigned_station"` // Station ID or empty
	AssignedSlot    string       `json:"assigned_slot" db:"assigned_slot"`       // Slot ID (e.g., A1) or empty

	// Identity & Binding
	ModelFingerprint string  `json:"model_fingerprint" db:"model_fingerprint"`
	MemoryBindingID  *string `json:"memory_binding_id" db:"memory_binding_id"`
	MemoryNamespace  *string `json:"memory_namespace" db:"memory_namespace"`

	LastHeartbeat *time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`

	// Runtime fields (not persisted)
	Uptime         time.Duration `json:"uptime,omitempty"`
	ExecutionCount int64         `json:"execution_count,omitempty"`
	CrashCount     int64         `json:"crash_count,omitempty"`
	RestartCount   int64         `json:"restart_count,omitempty"`
}

// IsHealthy returns true if the worker has recent heartbeat
func (w *Worker) IsHealthy(timeout time.Duration) bool {
	if w.LastHeartbeat == nil {
		return false
	}
	return time.Since(*w.LastHeartbeat) < timeout
}

// Station represents a station in the factory
type Station struct {
	ID        StationID     `json:"id" db:"id"`
	Status    StationStatus `json:"status" db:"status"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`

	// Joined data (not persisted separately)
	Workers []Worker `json:"workers,omitempty"`
}

// HasWorkers returns true if any workers are assigned
func (s *Station) HasWorkers() bool {
	return len(s.Workers) > 0
}

// StationMetrics represents time-series metrics for a station
type StationMetrics struct {
	ID           int64     `json:"id" db:"id"`
	StationID    string    `json:"station_id" db:"station_id"`
	Throughput   int       `json:"throughput" db:"throughput"`         // Tickets completed in window
	Defects      int       `json:"defects" db:"defects"`               // Gate failures in window
	AvgLatencyMs int       `json:"avg_latency_ms" db:"avg_latency_ms"` // Average execution latency
	TokensUsed   int64     `json:"tokens_used" db:"tokens_used"`       // LLM tokens consumed
	RecordedAt   time.Time `json:"recorded_at" db:"recorded_at"`
}

// FactoryStatus represents the overall factory state
type FactoryStatus struct {
	Stations       []Station        `json:"stations"`
	Workers        []Worker         `json:"workers"`
	TotalWorkers   int              `json:"total_workers"`
	ActiveWorkers  int              `json:"active_workers"`
	CrashedWorkers int              `json:"crashed_workers"`
	Metrics        []StationMetrics `json:"metrics,omitempty"`

	// Factory Physics KPIs
	QueueDepth  int     `json:"queue_depth"`  // B (Backlog)
	Throughput  float64 `json:"throughput"`   // mu (Tickets/min)
	ArrivalRate float64 `json:"arrival_rate"` // lambda (Tickets/min)
	DrainTime   float64 `json:"drain_time"`   // T (Minutes)
	TokenBurn   float64 `json:"token_burn"`   // Avg tokens/ticket
	Latency     float64 `json:"latency"`      // Avg ms/ticket
	DefectRate  float64 `json:"defect_rate"`  // % Rejected
	RetryDepth  float64 `json:"retry_depth"`  // Avg retries
	RetryMult   float64 `json:"retry_mult"`   // R (1 + %reject)
}

// WorkerHeartbeat is the message workers send to report health
type WorkerHeartbeat struct {
	WorkerID         string       `json:"worker_id"`
	WorkerType       string       `json:"worker_type"`
	Status           WorkerStatus `json:"status"`
	Endpoint         string       `json:"endpoint"`
	AssignedStation  string       `json:"assigned_station"`
	AssignedSlot     string       `json:"assigned_slot"`
	ModelFingerprint string       `json:"model_fingerprint"`
	MemoryBindingID  string       `json:"memory_binding_id"`
	MemoryNamespace  string       `json:"memory_namespace"`
	Timestamp        time.Time    `json:"timestamp"`
	Metrics          struct {
		ExecutionCount int64 `json:"execution_count"`
		TokensUsed     int64 `json:"tokens_used"`
		AvgLatencyMs   int64 `json:"avg_latency_ms"`
	} `json:"metrics"`
}
