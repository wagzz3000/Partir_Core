// Package foundry - Mock executor for testing
package foundry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/partir/core/pkg/plugin"
)

// MockExecutor is a deterministic executor for testing the pipeline
type MockExecutor struct {
	// CallLog records all calls for verification
	CallLog []MockExecutorCall

	// Behavior configuration
	SimulateError     bool
	SimulateLatencyMs int64
	FixedOutput       string
}

// MockExecutorCall records a single call to Execute
type MockExecutorCall struct {
	Request   *plugin.ExecuteRequest
	Response  *plugin.ExecuteResponse
	Timestamp time.Time
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		CallLog:           []MockExecutorCall{},
		SimulateLatencyMs: 100,
	}
}

// ID implements plugin.Executor
func (m *MockExecutor) ID() string {
	return "mock"
}

// Name implements plugin.Executor
func (m *MockExecutor) Name() string {
	return "Mock Executor (Testing)"
}

// Execute implements plugin.Executor
func (m *MockExecutor) Execute(ctx context.Context, req *plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	start := time.Now()

	// Simulate latency
	if m.SimulateLatencyMs > 0 {
		time.Sleep(time.Duration(m.SimulateLatencyMs) * time.Millisecond)
	}

	// Check for simulated error
	if m.SimulateError {
		return &plugin.ExecuteResponse{
			Output:       "",
			FinishReason: "error",
			Error:        "simulated error for testing",
			Model:        "mock-v1",
			Provider:     "mock",
			LatencyMs:    time.Since(start).Milliseconds(),
		}, fmt.Errorf("simulated error")
	}

	// Generate deterministic output
	output := m.FixedOutput
	if output == "" {
		// Hash the prompt to create deterministic but varied output
		hash := sha256.Sum256([]byte(req.Prompt))
		hashStr := hex.EncodeToString(hash[:8])
		output = fmt.Sprintf("Mock output for ticket %s, run %s. Hash: %s\n\nThis is a deterministic response based on your prompt.", req.TicketID, req.RunID, hashStr)
	}

	// Calculate fake token counts (rough approximation)
	promptTokens := len(req.Prompt) / 4
	completionTokens := len(output) / 4

	response := &plugin.ExecuteResponse{
		Output:           output,
		FinishReason:     "stop",
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		CostUSD:          float64(promptTokens+completionTokens) * 0.000001, // Fake cost
		Model:            "mock-v1",
		Provider:         "mock",
		LatencyMs:        time.Since(start).Milliseconds(),
	}

	// Log the call
	m.CallLog = append(m.CallLog, MockExecutorCall{
		Request:   req,
		Response:  response,
		Timestamp: time.Now(),
	})

	return response, nil
}

// Capabilities implements plugin.Executor
func (m *MockExecutor) Capabilities() plugin.ExecutorCapabilities {
	return plugin.ExecutorCapabilities{
		Models:                 []string{"mock-v1"},
		DefaultModel:           "mock-v1",
		MaxContextLen:          4096,
		SupportsStreaming:      false,
		SupportsToolCalling:    false,
		SupportsVision:         false,
		CostPerPromptToken:     0.0000005,
		CostPerCompletionToken: 0.0000015,
		RequestsPerMinute:      1000,
		TokensPerMinute:        100000,
	}
}

// HealthCheck implements plugin.Executor
func (m *MockExecutor) HealthCheck(ctx context.Context) error {
	return nil // Mock is always healthy
}

// Reset clears the call log
func (m *MockExecutor) Reset() {
	m.CallLog = []MockExecutorCall{}
}

// GetLastCall returns the most recent call, or nil if none
func (m *MockExecutor) GetLastCall() *MockExecutorCall {
	if len(m.CallLog) == 0 {
		return nil
	}
	return &m.CallLog[len(m.CallLog)-1]
}
