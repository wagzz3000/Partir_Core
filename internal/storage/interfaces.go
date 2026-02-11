package storage

import (
	"context"
)

// TicketRepository defines ticket persistence operations
type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) error
	GetByTicketID(ctx context.Context, ticketID string) (*Ticket, error)
	UpdateState(ctx context.Context, ticketID string, state TicketState) error
	// ListByState is not strictly used by Dispatcher.Run but might be useful?
	// Dispatcher doesn't call it. But foundry/main.go might.
	// For Dispatcher test, we only need what Dispatcher uses.
}

// RunRepository defines run persistence operations
type RunRepository interface {
	Create(ctx context.Context, run *Run) error
	Complete(ctx context.Context, runID, status string, tokens int, cost float64, errorMsg string) error
}

// ArtifactRepository defines artifact persistence operations
type ArtifactRepository interface {
	Create(ctx context.Context, artifact *Artifact, data []byte, contentType string) error
}

// DefectRepository defines defect persistence operations
type DefectRepository interface {
	Create(ctx context.Context, defect *Defect) error
}

// BlobRepository defines blob storage operations
type BlobRepository interface {
	PutBlob(ctx context.Context, hash string, data []byte, contentType string) (string, error)
	GetBlob(ctx context.Context, hash string) ([]byte, error)
}
