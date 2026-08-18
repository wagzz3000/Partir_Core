package tickets

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/partir/core/pkg/plugin"
)

// Ticket represents a canonical work order in the system
type Ticket struct {
	ID       string `json:"id"`
	TicketID string `json:"ticket_id"`
	Title    string `json:"title"`
	PluginID string `json:"plugin_id"`
	JobType  string `json:"job_type"`
	AlphaRef string `json:"alpha_ref"`
	BetaRef  string `json:"beta_ref,omitempty"`
	GreenTag string `json:"green_tag_ref,omitempty"`

	State    State `json:"state"`
	Priority int   `json:"priority"`

	Inputs          json.RawMessage `json:"inputs"`
	Constraints     json.RawMessage `json:"constraints"`
	AcceptanceGates []string        `json:"acceptance_gates"`
	Limits          plugin.Limits   `json:"limits"`

	OutputsExpected []string `json:"outputs_expected"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// State represents the lifecycle state of a ticket
type State string

const (
	StateDraft         State = "draft"
	StateSpecReady     State = "spec_ready"
	StateDispatched    State = "dispatched"
	StateLocalQCFailed State = "local_qc_failed"
	StateInExecution   State = "in_execution"
	StateReadyForOmega State = "ready_for_omega"
	StateOmegaFailed   State = "omega_failed"
	StateCompleted     State = "completed"
	StateCancelled     State = "cancelled"
)

// NewTicket creates a new ticket with a generated ID
func NewTicket(title, pluginID, jobType string) *Ticket {
	return &Ticket{
		ID:        uuid.New().String(),
		TicketID:  "TKT-" + uuid.New().String()[:8],
		Title:     title,
		PluginID:  pluginID,
		JobType:   jobType,
		State:     StateDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Inputs:    json.RawMessage("{}"),
	}
}
