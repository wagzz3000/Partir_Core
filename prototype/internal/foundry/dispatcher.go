// Package foundry provides the Foundry module for Partir Core.
// Foundry is the production system - routes tickets, runs jobs, stores artifacts.
package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/partir/core/internal/collaboration"
	"github.com/partir/core/internal/factory"
	"github.com/partir/core/internal/omega"
	"github.com/partir/core/internal/storage"
	"github.com/partir/core/pkg/metrics"
	"github.com/partir/core/pkg/plugin"
)

// OmegaEngine abstracts the Omega validation engine
type OmegaEngine interface {
	RunPreflight(ctx context.Context, req *omega.GateRequest) *omega.Result
	RunPostExecution(ctx context.Context, req *omega.GateRequest) *omega.Result
	Reset()
}

// Dispatcher routes tickets to plugins and executors
type Dispatcher struct {
	// Storage interfaces for easier mocking
	tickets   storage.TicketRepository
	runs      storage.RunRepository
	artifacts storage.ArtifactRepository
	defects   storage.DefectRepository

	// Keep original reference if needed, or just use interfaces
	storage  *storage.Storage
	plugins  *plugin.Registry
	factory  *factory.Registry
	omega    OmegaEngine
	executor Executor
	andonNC  *nats.Conn // NATS connection for Andon Cord signals
}

// NewDispatcher creates a new Foundry dispatcher
func NewDispatcher(store *storage.Storage, plugins *plugin.Registry, factory *factory.Registry, om OmegaEngine) *Dispatcher {
	return &Dispatcher{
		storage:   store,
		tickets:   store.Tickets,
		runs:      store.Runs,
		artifacts: store.Artifacts,
		defects:   store.Defects,
		plugins:   plugins,
		factory:   factory,
		omega:     om,
		executor:  &LocalExecutor{plugins: plugins},
	}
}

// SetExecutor sets the executor to use
func (d *Dispatcher) SetExecutor(executor Executor) {
	d.executor = executor
}

// SetAndonNC sets the NATS connection used for Andon Cord signals.
// If set, the dispatcher will publish halt signals on Omega gate failures.
func (d *Dispatcher) SetAndonNC(nc *nats.Conn) {
	d.andonNC = nc
}

// Submit creates a new ticket in the system
func (d *Dispatcher) Submit(ctx context.Context, ticket *Ticket) error {
	// Convert to storage ticket
	t := &storage.Ticket{
		ID:        uuid.New().String(),
		TicketID:  ticket.TicketID,
		Title:     ticket.Title,
		PluginID:  ticket.PluginID,
		JobType:   ticket.JobType,
		AlphaRef:  ticket.AlphaRef,
		BetaRef:   ticket.BetaRef,
		State:     storage.StateSpecReady,
		Priority:  ticket.Priority,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Marshal JSON fields
	if ticket.Inputs != nil {
		t.Inputs, _ = json.Marshal(ticket.Inputs)
	} else {
		t.Inputs = []byte("{}")
	}
	if ticket.Constraints != nil {
		t.Constraints, _ = json.Marshal(ticket.Constraints)
	} else {
		t.Constraints = []byte("{}")
	}
	if len(ticket.AcceptanceGates) > 0 {
		t.AcceptanceGates, _ = json.Marshal(ticket.AcceptanceGates)
	} else {
		t.AcceptanceGates = []byte("[]")
	}
	if ticket.Limits != nil {
		t.Limits, _ = json.Marshal(ticket.Limits)
	} else {
		t.Limits = []byte("{}")
	}
	t.RoutingHints = []byte("{}")
	t.OutputsExpected = []byte("[]")

	return d.tickets.Create(ctx, t)
}

// Run executes a ticket through the full pipeline
func (d *Dispatcher) Run(ctx context.Context, ticketID string) (*RunResult, error) {
	runStart := time.Now()

	// 1. Load ticket
	ticket, err := d.tickets.GetByTicketID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ticket: %w", err)
	}

	// 2. Get plugin
	plug, err := d.plugins.Get(ticket.PluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %w", err)
	}

	// 3. Update state
	if err := d.tickets.UpdateState(ctx, ticketID, storage.StateDispatched); err != nil {
		return nil, err
	}
	metrics.TicketTotal.WithLabelValues(string(storage.StateDispatched)).Inc()

	// 4. Run preflight Omega check
	preflightReq := &omega.GateRequest{
		TicketID: ticketID,
		PluginID: ticket.PluginID,
		AlphaRef: ticket.AlphaRef,
		BetaRef:  ticket.BetaRef,
		Inputs:   ticket.Inputs,
	}
	preflightResult := d.omega.RunPreflight(ctx, preflightReq)
	if !preflightResult.Pass {
		d.tickets.UpdateState(ctx, ticketID, storage.StateLocalQCFailed)
		metrics.TicketTotal.WithLabelValues(string(storage.StateLocalQCFailed)).Inc()
		metrics.RunFailed.WithLabelValues("preflight").Inc()
		return &RunResult{
			TicketID:    ticketID,
			Success:     false,
			OmegaResult: preflightResult,
		}, nil
	}

	// 5. Create run record
	run := &storage.Run{
		ID:        uuid.New().String(),
		RunID:     uuid.New().String(),
		TicketID:  ticketID,
		Attempt:   1,
		Executor:  "local",
		StartedAt: time.Now(),
		Status:    "running",
		Metadata:  []byte("{}"),
	}
	if err := d.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	metrics.RunTotal.WithLabelValues(ticket.PluginID, ticket.JobType).Inc()
	metrics.RunAttempt.WithLabelValues(strconv.Itoa(run.Attempt)).Inc()

	// 6. Update state
	d.tickets.UpdateState(ctx, ticketID, storage.StateInExecution)
	metrics.TicketTotal.WithLabelValues(string(storage.StateInExecution)).Inc()

	// 7. Assign worker from Factory (if enabled)
	var worker *factory.Worker
	if d.factory != nil {
		// Prefer a specific station based on job type? For now just pick by type
		workerType := "alpha" // Default
		if ticket.JobType != "" {
			workerType = ticket.JobType // e.g., alpha, beta, omega
		}

		// Try to get a specific assigned worker first?
		// For now simple routing: find available worker of type
		worker, err = d.factory.GetAvailableWorker(ctx, workerType)
		if err == nil && worker != nil {
			// Temporarily assign to a dynamic station or just use worker endpoint
			// In full factory, we'd assign to a station. Here just use the worker.
			// d.factory.AssignWorkerToStation(ctx, factory.StationAlpha, worker.ID)
		} else {
			// Log warning but proceed with local/random if no worker found?
			// Or fail if factory mode is strict?
			// For now, fall back to local executor or proceed without specific worker assignment
			fmt.Printf("Warning: No factory worker available for %s: %v\n", workerType, err)
		}
	}

	// 8. Execute via plugin
	workOrder := plugin.WorkOrder{
		TicketID: ticketID,
		RunID:    run.RunID,
		JobType:  ticket.JobType,
		AlphaRef: ticket.AlphaRef,
		BetaRef:  ticket.BetaRef,
		Inputs:   ticket.Inputs,
		Attempt:  1,
	}

	execResult, err := d.executor.Execute(ctx, plug, workOrder)
	if err != nil {
		d.tickets.UpdateState(ctx, ticketID, storage.StateLocalQCFailed)
		d.runs.Complete(ctx, run.RunID, "failed", 0, 0, "")
		metrics.RunFailed.WithLabelValues("execution").Inc()
		return &RunResult{
			TicketID: ticketID,
			Success:  false,
			Error:    err.Error(),
		}, nil
	}

	// 8. Run post-execution Omega check
	d.tickets.UpdateState(ctx, ticketID, storage.StateReadyForOmega)

	omegaReq := &omega.GateRequest{
		TicketID: ticketID,
		RunID:    run.RunID,
		PluginID: ticket.PluginID,
		AlphaRef: ticket.AlphaRef,
		BetaRef:  ticket.BetaRef,
		Inputs:   ticket.Inputs,
	}

	// Convert artifacts
	for _, a := range execResult.Artifacts {
		omegaReq.Artifacts = append(omegaReq.Artifacts, omega.ArtifactData{
			ArtifactID:   a.ArtifactID,
			ArtifactType: a.ArtifactType,
			Hash:         a.Hash,
			SchemaRef:    a.SchemaRef,
			Data:         a.Data,
		})
	}

	omegaResult := d.omega.RunPostExecution(ctx, omegaReq)

	if !omegaResult.Pass {
		d.tickets.UpdateState(ctx, ticketID, storage.StateOmegaFailed)
		d.runs.Complete(ctx, run.RunID, "failed", execResult.CostTokens, execResult.CostUSD, "")
		metrics.TicketTotal.WithLabelValues(string(storage.StateOmegaFailed)).Inc()
		metrics.RunFailed.WithLabelValues("omega").Inc()

		// Pull the Andon Cord — halt signal to Collaboration API
		d.pullAndonCord(ticketID, run.RunID, ticket.PluginID, omegaResult)

		// Store defects
		for _, defect := range omegaResult.Defects {
			d.defects.Create(ctx, &storage.Defect{
				ID:              uuid.New().String(),
				DefectID:        defect.DefectID,
				RunID:           run.RunID,
				GateID:          defect.GateID,
				DefectClass:     defect.DefectClass,
				PluginID:        ticket.PluginID,
				Message:         defect.Message,
				OffendingFields: defect.OffendingFields,
				SuggestedFix:    defect.SuggestedFix,
				CreatedAt:       time.Now(),
			})
		}

		return &RunResult{
			TicketID:    ticketID,
			Success:     false,
			OmegaResult: omegaResult,
		}, nil
	}

	// 9. Store artifacts
	for _, a := range execResult.Artifacts {
		artifact := &storage.Artifact{
			ID:           uuid.New().String(),
			ArtifactID:   a.ArtifactID,
			TicketID:     ticketID,
			RunID:        run.RunID,
			ArtifactType: a.ArtifactType,
			Hash:         a.Hash,
			SchemaRef:    a.SchemaRef,
			CreatedAt:    time.Now(),
		}
		prov, _ := json.Marshal(a.Provenance)
		artifact.Provenance = prov

		if err := d.artifacts.Create(ctx, artifact, a.Data, "application/json"); err != nil {
			return nil, fmt.Errorf("failed to store artifact: %w", err)
		}
	}

	// 10. Complete
	d.tickets.UpdateState(ctx, ticketID, storage.StateCompleted)
	d.runs.Complete(ctx, run.RunID, "completed", execResult.CostTokens, execResult.CostUSD, "")
	d.omega.Reset() // Clear retry history on success

	// Record success metrics
	metrics.TicketCompleted.Inc()
	metrics.TicketTotal.WithLabelValues(string(storage.StateCompleted)).Inc()
	metrics.RunDuration.WithLabelValues(ticket.PluginID, ticket.JobType).Observe(time.Since(runStart).Seconds())
	metrics.AITokensTotal.WithLabelValues("default", ticket.PluginID).Add(float64(execResult.CostTokens))
	metrics.AICostTotal.WithLabelValues("default").Add(execResult.CostUSD)

	return &RunResult{
		TicketID:  ticketID,
		Success:   true,
		Artifacts: execResult.Artifacts,
	}, nil
}

// pullAndonCord publishes a halt signal to the Collaboration API
// when the Omega post-execution gates fail. This triggers the triage
// logic which routes fix-it tickets (Beta/Foundry) or human alerts (Alpha).
func (d *Dispatcher) pullAndonCord(ticketID, runID, pluginID string, omegaResult *omega.Result) {
	if d.andonNC == nil {
		return
	}

	// Build defect summaries from Omega defects
	var defects []collaboration.DefectSummary
	for _, defect := range omegaResult.Defects {
		defects = append(defects, collaboration.DefectSummary{
			DefectID:    defect.DefectID,
			DefectClass: defect.DefectClass,
			GateID:      defect.GateID,
			Message:     defect.Message,
		})
	}

	// Derive the first failed gate ID for the signal
	failedGate := ""
	if len(omegaResult.Metrics.GatesFailed) > 0 {
		failedGate = omegaResult.Metrics.GatesFailed[0]
	}

	signal := &collaboration.AndonSignal{
		SignalID:  uuid.New().String(),
		TicketID:  ticketID,
		RunID:     runID,
		PluginID:  pluginID,
		GateID:    failedGate,
		Defects:   defects,
		Timestamp: time.Now(),
	}

	data, err := signal.Marshal()
	if err != nil {
		log.Printf("[andon] Failed to marshal signal: %v", err)
		return
	}

	if err := d.andonNC.Publish(collaboration.TopicAndon, data); err != nil {
		log.Printf("[andon] Failed to publish halt signal: %v", err)
		return
	}

	log.Printf("[andon] 🔴 HALT — ticket=%s run=%s gate=%s defects=%d",
		ticketID, runID, failedGate, len(defects))
}

// Ticket represents a work order for the dispatcher
type Ticket struct {
	TicketID        string         `json:"ticket_id"`
	Title           string         `json:"title"`
	PluginID        string         `json:"plugin_id"`
	JobType         string         `json:"job_type"`
	AlphaRef        string         `json:"alpha_ref"`
	BetaRef         string         `json:"beta_ref,omitempty"`
	Inputs          interface{}    `json:"inputs"`
	Constraints     interface{}    `json:"constraints,omitempty"`
	AcceptanceGates []string       `json:"acceptance_gates,omitempty"`
	Limits          *plugin.Limits `json:"limits,omitempty"`
	Priority        int            `json:"priority,omitempty"`
}

// RunResult contains the outcome of a ticket execution
type RunResult struct {
	TicketID    string            `json:"ticket_id"`
	Success     bool              `json:"success"`
	Artifacts   []plugin.Artifact `json:"artifacts,omitempty"`
	OmegaResult *omega.Result     `json:"omega_result,omitempty"`
	Error       string            `json:"error,omitempty"`
}
