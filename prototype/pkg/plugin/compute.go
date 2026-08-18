package plugin

import (
	"context"
	"time"
)

// TicketVolume represents a quantity of work to be processed.
type TicketVolume int

// ClusterCapacity represents the current state of the compute infrastructure.
type ClusterCapacity struct {
	// BaselineReady is the number of always-on workers available.
	BaselineReady int
	// BurstReady is the number of ephemeral workers currently provisioned.
	BurstReady int
	// MaxBurst is the theoretical limit of burst capacity.
	MaxBurst int
}

// AllocationID uniquely identifies a burst allocation request.
type AllocationID string

// ComputeProvider defines the interface for orchestrating GPU resources.
// This interface abstracts the underlying infrastructure (Hetzner, Beam, Modal).
type ComputeProvider interface {
	// Name returns the provider name (e.g., "modal", "beam").
	Name() string

	// GetCapacity returns the current state of the compute cluster.
	GetCapacity(ctx context.Context) (ClusterCapacity, error)

	// Burst provisions ephemeral resources to meet the specified demand.
	// It returns an AllocationID that can be used to track or release the resources.
	// The implementation should handle the specifics of spinning up GPU containers.
	Burst(ctx context.Context, demand TicketVolume) (AllocationID, error)

	// Release de-provisions resources associated with the allocation.
	// This is called when demand subsides (drain time < target).
	Release(ctx context.Context, allocationID AllocationID) error

	// HealthCheck returns the provider's status.
	HealthCheck(ctx context.Context) error
}

// ComputeConfig holds configuration for the compute plugin.
type ComputeConfig struct {
	Provider      string        `json:"provider" yaml:"provider"`               // "modal", "beam", "mock"
	MaxBurst      int           `json:"max_burst" yaml:"max_burst"`             // Max concurrent GPUs
	ScaleUpWindow time.Duration `json:"scale_up_window" yaml:"scale_up_window"` // Cooldown between scale-ups
	DrainTimeout  time.Duration `json:"drain_timeout" yaml:"drain_timeout"`     // Time to wait before scaling down
}
