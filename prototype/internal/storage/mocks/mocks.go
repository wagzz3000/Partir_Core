package mocks

import (
	"context"

	"github.com/partir/core/internal/omega"
	"github.com/partir/core/internal/storage"
	"github.com/stretchr/testify/mock"
)

// MockTicketRepository is a mock implementation of storage.TicketRepository
type MockTicketRepository struct {
	mock.Mock
}

func (m *MockTicketRepository) Create(ctx context.Context, t *storage.Ticket) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTicketRepository) GetByTicketID(ctx context.Context, ticketID string) (*storage.Ticket, error) {
	args := m.Called(ctx, ticketID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Ticket), args.Error(1)
}

func (m *MockTicketRepository) UpdateState(ctx context.Context, ticketID string, state storage.TicketState) error {
	args := m.Called(ctx, ticketID, state)
	return args.Error(0)
}

// MockRunRepository is a mock implementation of storage.RunRepository
type MockRunRepository struct {
	mock.Mock
}

func (m *MockRunRepository) Create(ctx context.Context, r *storage.Run) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockRunRepository) Complete(ctx context.Context, runID, status string, tokens int, cost float64, errorMsg string) error {
	args := m.Called(ctx, runID, status, tokens, cost, errorMsg)
	return args.Error(0)
}

// MockArtifactRepository is a mock implementation of storage.ArtifactRepository
type MockArtifactRepository struct {
	mock.Mock
}

func (m *MockArtifactRepository) Create(ctx context.Context, a *storage.Artifact, data []byte, contentType string) error {
	args := m.Called(ctx, a, data, contentType)
	return args.Error(0)
}

// MockDefectRepository is a mock implementation of storage.DefectRepository
type MockDefectRepository struct {
	mock.Mock
}

func (m *MockDefectRepository) Create(ctx context.Context, d *storage.Defect) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

// MockBlobRepository is a mock implementation of storage.BlobRepository
type MockBlobRepository struct {
	mock.Mock
}

func (m *MockBlobRepository) PutBlob(ctx context.Context, hash string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, hash, data, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockBlobRepository) GetBlob(ctx context.Context, hash string) ([]byte, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// MockOmegaEngine is a mock implementation of foundry.OmegaEngine
// We place it here for convenience, importing omega package
// Note: Ensure no circular dependency between storage/mocks and omega
type MockOmegaEngine struct {
	mock.Mock
}

func (m *MockOmegaEngine) RunPreflight(ctx context.Context, req *omega.GateRequest) *omega.Result {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*omega.Result)
}

func (m *MockOmegaEngine) RunPostExecution(ctx context.Context, req *omega.GateRequest) *omega.Result {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*omega.Result)
}

func (m *MockOmegaEngine) Reset() {
	m.Called()
}
