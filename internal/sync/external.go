package sync

import (
	"context"
	"fmt"
	"time"
)

// TicketState represents possible external ticket states
type TicketState string

const (
	StateOpen       TicketState = "open"
	StateInProgress TicketState = "in_progress"
	StateDone       TicketState = "done"
	StateCancelled  TicketState = "cancelled"
)

// ExternalTicket represents a ticket in an external system
type ExternalTicket struct {
	ExternalID  string            `json:"external_id"`
	PartirID    string            `json:"partir_id"`
	State       TicketState       `json:"state"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ExternalSyncer defines the interface for bi-directional ticket sync
type ExternalSyncer interface {
	// Push sends Partir ticket state to the external system
	Push(ctx context.Context, ticket ExternalTicket) error

	// Pull fetches the latest state from the external system
	Pull(ctx context.Context, externalID string) (*ExternalTicket, error)

	// Name returns the integration name
	Name() string
}

// JiraSyncer implements ExternalSyncer for Jira
type JiraSyncer struct {
	BaseURL   string
	APIToken  string
	ProjectID string
}

func NewJiraSyncer(baseURL, apiToken, projectID string) *JiraSyncer {
	return &JiraSyncer{
		BaseURL:   baseURL,
		APIToken:  apiToken,
		ProjectID: projectID,
	}
}

func (j *JiraSyncer) Name() string { return "jira" }

func (j *JiraSyncer) Push(ctx context.Context, ticket ExternalTicket) error {
	// TODO: Implement Jira REST API call
	// POST /rest/api/3/issue or PUT /rest/api/3/issue/{issueIdOrKey}/transitions
	return fmt.Errorf("jira push not yet implemented")
}

func (j *JiraSyncer) Pull(ctx context.Context, externalID string) (*ExternalTicket, error) {
	// TODO: Implement Jira REST API call
	// GET /rest/api/3/issue/{issueIdOrKey}
	return nil, fmt.Errorf("jira pull not yet implemented")
}

// LinearSyncer implements ExternalSyncer for Linear
type LinearSyncer struct {
	APIKey string
	TeamID string
}

func NewLinearSyncer(apiKey, teamID string) *LinearSyncer {
	return &LinearSyncer{
		APIKey: apiKey,
		TeamID: teamID,
	}
}

func (l *LinearSyncer) Name() string { return "linear" }

func (l *LinearSyncer) Push(ctx context.Context, ticket ExternalTicket) error {
	// TODO: Implement Linear GraphQL mutation
	return fmt.Errorf("linear push not yet implemented")
}

func (l *LinearSyncer) Pull(ctx context.Context, externalID string) (*ExternalTicket, error) {
	// TODO: Implement Linear GraphQL query
	return nil, fmt.Errorf("linear pull not yet implemented")
}
