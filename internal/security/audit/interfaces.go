package audit

import (
	"context"
	"time"
)

// LogEntry represents a single audit event
type LogEntry struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	ActorID      string                 `json:"actor_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Auditor defines the interface for recording audit logs
type Auditor interface {
	// Log records an action
	Log(ctx context.Context, entry LogEntry) error

	// Query retrieves logs based on filters (implementation specific)
	// For now, simple list by resource
	ListByResource(ctx context.Context, tenantID, resourceType, resourceID string) ([]LogEntry, error)
}
