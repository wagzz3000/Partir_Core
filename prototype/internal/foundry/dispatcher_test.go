package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/partir/core/internal/omega"
	"github.com/partir/core/internal/storage"
	"github.com/partir/core/pkg/plugin"
)

// Manual Mocks

type MockTicketRepo struct {
	GetByTicketIDFunc func(ctx context.Context, ticketID string) (*storage.Ticket, error)
	UpdateStateFunc   func(ctx context.Context, ticketID string, state storage.TicketState) error
	CreateFunc        func(ctx context.Context, t *storage.Ticket) error
	ListByStateFunc   func(ctx context.Context, state storage.TicketState) ([]storage.Ticket, error)
}

func (m *MockTicketRepo) Create(ctx context.Context, t *storage.Ticket) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	return nil
}
func (m *MockTicketRepo) GetByTicketID(ctx context.Context, ticketID string) (*storage.Ticket, error) {
	if m.GetByTicketIDFunc != nil {
		return m.GetByTicketIDFunc(ctx, ticketID)
	}
	return nil, fmt.Errorf("ticket not found")
}
func (m *MockTicketRepo) UpdateState(ctx context.Context, ticketID string, state storage.TicketState) error {
	if m.UpdateStateFunc != nil {
		return m.UpdateStateFunc(ctx, ticketID, state)
	}
	return nil
}
func (m *MockTicketRepo) ListByState(ctx context.Context, state storage.TicketState) ([]storage.Ticket, error) {
	if m.ListByStateFunc != nil {
		return m.ListByStateFunc(ctx, state)
	}
	return nil, nil
}

type MockRunRepo struct {
	CreateFunc   func(ctx context.Context, run *storage.Run) error
	CompleteFunc func(ctx context.Context, runID, status string, tokens int, cost float64, errorMsg string) error
}

func (m *MockRunRepo) Create(ctx context.Context, run *storage.Run) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, run)
	}
	return nil
}
func (m *MockRunRepo) Complete(ctx context.Context, runID, status string, tokens int, cost float64, errorMsg string) error {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, runID, status, tokens, cost, errorMsg)
	}
	return nil
}

type MockArtifactRepo struct {
	CreateFunc func(ctx context.Context, artifact *storage.Artifact, data []byte, contentType string) error
}

func (m *MockArtifactRepo) Create(ctx context.Context, artifact *storage.Artifact, data []byte, contentType string) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, artifact, data, contentType)
	}
	return nil
}

type MockDefectRepo struct {
	CreateFunc func(ctx context.Context, defect *storage.Defect) error
}

func (m *MockDefectRepo) Create(ctx context.Context, defect *storage.Defect) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, defect)
	}
	return nil
}

type CheckMockPlugin struct {
	id string
}

func (m *CheckMockPlugin) ID() string                                             { return m.id }
func (m *CheckMockPlugin) Version() string                                        { return "1.0" }
func (m *CheckMockPlugin) Manifest() *plugin.Manifest                             { return &plugin.Manifest{ID: m.id} }
func (m *CheckMockPlugin) Capabilities() []string                                 { return []string{"default"} }
func (m *CheckMockPlugin) InputSchema(jobType string) json.RawMessage             { return []byte("{}") }
func (m *CheckMockPlugin) OutputSchema(jobType string) json.RawMessage            { return []byte("{}") }
func (m *CheckMockPlugin) Plan(workOrder plugin.WorkOrder) ([]plugin.Step, error) { return nil, nil }
func (m *CheckMockPlugin) Execute(ctx context.Context, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error) {
	return nil, nil // Not used via Executor
}
func (m *CheckMockPlugin) Gates(jobType string) []string { return nil }
func (m *CheckMockPlugin) Validate(ctx context.Context, artifacts []plugin.Artifact) ([]plugin.Defect, error) {
	return nil, nil
}

// TestExecutor implements foundrys internal Executor interface
type TestExecutor struct {
	ExecuteFunc func(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error)
}

func (m *TestExecutor) Execute(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, plug, workOrder)
	}
	return nil, fmt.Errorf("mock executor execute not implemented")
}

// MockOmega implements OmegaEngine interface
type MockOmega struct {
	RunPreflightFunc     func(ctx context.Context, req *omega.GateRequest) *omega.Result
	RunPostExecutionFunc func(ctx context.Context, req *omega.GateRequest) *omega.Result
	ResetFunc            func()
}

func (m *MockOmega) RunPreflight(ctx context.Context, req *omega.GateRequest) *omega.Result {
	if m.RunPreflightFunc != nil {
		return m.RunPreflightFunc(ctx, req)
	}
	return &omega.Result{Pass: true}
}

func (m *MockOmega) RunPostExecution(ctx context.Context, req *omega.GateRequest) *omega.Result {
	if m.RunPostExecutionFunc != nil {
		return m.RunPostExecutionFunc(ctx, req)
	}
	return &omega.Result{Pass: true}
}

func (m *MockOmega) Reset() {
	if m.ResetFunc != nil {
		m.ResetFunc()
	}
}

func NewTestDispatcher(
	tickets storage.TicketRepository,
	runs storage.RunRepository,
	artifacts storage.ArtifactRepository,
	defects storage.DefectRepository,
	plugins *plugin.Registry,
	omega OmegaEngine,
	executor Executor,
) *Dispatcher {
	return &Dispatcher{
		tickets:   tickets,
		runs:      runs,
		artifacts: artifacts,
		defects:   defects,
		plugins:   plugins,
		omega:     omega,
		executor:  executor,
	}
}

func TestDispatcher_Run_Success(t *testing.T) {
	ctx := context.Background()
	ticketID := "ticket-1"

	// Track calls
	calls := make(map[string]int)

	// Setup Mocks
	mockTickets := &MockTicketRepo{
		GetByTicketIDFunc: func(ctx context.Context, id string) (*storage.Ticket, error) {
			calls["GetByTicketID"]++
			if id != ticketID {
				return nil, fmt.Errorf("not found")
			}
			return &storage.Ticket{
				TicketID: ticketID,
				PluginID: "mock-plugin",
				Inputs:   []byte("{}"),
			}, nil
		},
		UpdateStateFunc: func(ctx context.Context, id string, state storage.TicketState) error {
			calls["UpdateState:"+string(state)]++
			return nil
		},
	}

	mockRuns := &MockRunRepo{
		CreateFunc: func(ctx context.Context, run *storage.Run) error {
			calls["RunCreate"]++
			return nil
		},
		CompleteFunc: func(ctx context.Context, runID, status string, tokens int, cost float64, errorMsg string) error {
			calls["RunComplete:"+status]++
			return nil
		},
	}

	mockArtifacts := &MockArtifactRepo{
		CreateFunc: func(ctx context.Context, artifact *storage.Artifact, data []byte, contentType string) error {
			calls["ArtifactCreate"]++
			return nil
		},
	}

	mockDefects := &MockDefectRepo{}

	mockEx := &TestExecutor{
		ExecuteFunc: func(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error) {
			calls["Execute"]++
			return &plugin.ExecutionResult{
				Artifacts: []plugin.Artifact{
					{ArtifactID: "art-1", Data: []byte("foo"), ArtifactType: "file"},
				},
				CostTokens: 100,
				CostUSD:    0.01,
			}, nil
		},
	}

	mockOmega := &MockOmega{
		RunPreflightFunc: func(ctx context.Context, req *omega.GateRequest) *omega.Result {
			calls["OmegaPreflight"]++
			return &omega.Result{Pass: true}
		},
		RunPostExecutionFunc: func(ctx context.Context, req *omega.GateRequest) *omega.Result {
			calls["OmegaPost"]++
			return &omega.Result{Pass: true}
		},
	}

	pluginReg := plugin.NewRegistry()
	mockPlug := &CheckMockPlugin{id: "mock-plugin"}
	pluginReg.Register(mockPlug)

	d := NewTestDispatcher(mockTickets, mockRuns, mockArtifacts, mockDefects, pluginReg, mockOmega, mockEx)

	// Run
	res, err := d.Run(ctx, ticketID)

	// Assertions
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("RunResult.Success should be true")
	}
	if len(res.Artifacts) != 1 {
		t.Errorf("Expected 1 artifact, got %d", len(res.Artifacts))
	}

	// Verify interactions
	if calls["GetByTicketID"] != 1 {
		t.Errorf("Expected GetByTicketID call")
	}
	if calls["UpdateState:DISPATCHED"] != 1 {
		t.Errorf("Expected UpdateState DISPATCHED")
	}
	if calls["OmegaPreflight"] != 1 {
		t.Errorf("Expected OmegaPreflight")
	}
	if calls["RunCreate"] != 1 {
		t.Errorf("Expected RunCreate")
	}
	if calls["UpdateState:IN_EXECUTION"] != 1 {
		t.Errorf("Expected UpdateState IN_EXECUTION")
	}
	if calls["Execute"] != 1 {
		t.Errorf("Expected Execute")
	}
	if calls["UpdateState:READY_FOR_OMEGA"] != 1 {
		t.Errorf("Expected UpdateState READY_FOR_OMEGA")
	}
	if calls["OmegaPost"] != 1 {
		t.Errorf("Expected OmegaPost")
	}
	if calls["ArtifactCreate"] != 1 {
		t.Errorf("Expected ArtifactCreate")
	}
	if calls["UpdateState:COMPLETED"] != 1 {
		t.Errorf("Expected UpdateState COMPLETED")
	}
	if calls["RunComplete:completed"] != 1 {
		t.Errorf("Expected RunComplete completed")
	}
}

func TestDispatcher_Run_OmegaPreflightFailure(t *testing.T) {
	ctx := context.Background()
	ticketID := "ticket-fail"

	calls := make(map[string]int)

	mockTickets := &MockTicketRepo{
		GetByTicketIDFunc: func(ctx context.Context, id string) (*storage.Ticket, error) {
			calls["GetByTicketID"]++
			return &storage.Ticket{
				TicketID: ticketID,
				PluginID: "mock-plugin",
				Inputs:   []byte("{}"),
			}, nil
		},
		UpdateStateFunc: func(ctx context.Context, id string, state storage.TicketState) error {
			calls["UpdateState:"+string(state)]++
			return nil
		},
	}

	mockRuns := &MockRunRepo{}

	mockOmega := &MockOmega{
		RunPreflightFunc: func(ctx context.Context, req *omega.GateRequest) *omega.Result {
			calls["OmegaPreflight"]++
			return &omega.Result{
				Pass: false,
				Defects: []omega.Defect{
					{Message: "preflight failed"},
				},
			}
		},
	}

	pluginReg := plugin.NewRegistry()
	mockPlug := &CheckMockPlugin{id: "mock-plugin"}
	pluginReg.Register(mockPlug)

	d := NewTestDispatcher(mockTickets, mockRuns, nil, nil, pluginReg, mockOmega, nil)

	res, err := d.Run(ctx, ticketID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if res.Success {
		t.Errorf("Expected Success=false")
	}
	if res.OmegaResult == nil {
		t.Errorf("Expected OmegaResult")
	} else if res.OmegaResult.Pass {
		t.Errorf("Expected OmegaResult.Pass=false")
	}

	if calls["OmegaPreflight"] != 1 {
		t.Errorf("Expected OmegaPreflight")
	}
	if calls["UpdateState:LOCAL_QC_FAILED"] != 1 {
		t.Errorf("Expected UpdateState LOCAL_QC_FAILED")
	}
	// Verify we did NOT run execution
	if calls["RunCreate"] > 0 {
		t.Errorf("Should not have created run")
	}
}
