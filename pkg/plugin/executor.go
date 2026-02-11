// Package plugin - Executor interface for LLM plugins
//
// This interface is implemented by external LLM plugins:
// - partir-plugin-ollama: Local SLM via Ollama
// - partir-plugin-helicone: Cloud LLM via Helicone proxy
package plugin

import "context"

// Executor defines the interface for LLM execution plugins
type Executor interface {
	// ID returns a unique identifier for this executor
	ID() string

	// Name returns a human-readable name
	Name() string

	// Execute runs the LLM with the given request and returns output
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

	// Capabilities returns what this executor supports
	Capabilities() ExecutorCapabilities

	// HealthCheck verifies the executor is ready
	HealthCheck(ctx context.Context) error
}

// ExecuteRequest contains all inputs for an LLM execution
type ExecuteRequest struct {
	// Core identifiers
	TicketID string `json:"ticket_id"`
	RunID    string `json:"run_id"`

	// LLM parameters
	Prompt       string   `json:"prompt"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens"`
	Temperature  float64  `json:"temperature"`
	StopTokens   []string `json:"stop_tokens,omitempty"`

	// Context
	AlphaRulebookID string            `json:"alpha_rulebook_id,omitempty"`
	BetaRulebookID  string            `json:"beta_rulebook_id,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	// Budget (optional hard limits)
	MaxCostUSD float64 `json:"max_cost_usd,omitempty"`
}

// ExecuteResponse contains the LLM output and metadata
type ExecuteResponse struct {
	// Output content
	Output       string `json:"output"`
	FinishReason string `json:"finish_reason"` // "stop", "length", "error"

	// Token usage
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// Cost tracking
	CostUSD float64 `json:"cost_usd"`

	// Model info
	Model    string `json:"model"`
	Provider string `json:"provider"` // "ollama", "openai", "anthropic", etc.

	// Timing
	LatencyMs int64 `json:"latency_ms"`

	// Error (if any)
	Error string `json:"error,omitempty"`
}

// ExecutorCapabilities describes what an executor supports
type ExecutorCapabilities struct {
	// Model info
	Models        []string `json:"models"`
	DefaultModel  string   `json:"default_model"`
	MaxContextLen int      `json:"max_context_len"`

	// Features
	SupportsStreaming   bool `json:"supports_streaming"`
	SupportsToolCalling bool `json:"supports_tool_calling"`
	SupportsVision      bool `json:"supports_vision"`

	// Cost info
	CostPerPromptToken     float64 `json:"cost_per_prompt_token"`
	CostPerCompletionToken float64 `json:"cost_per_completion_token"`

	// Rate limits
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
}

// ExecutorRegistry manages registered executors
type ExecutorRegistry struct {
	executors map[string]Executor
}

// NewExecutorRegistry creates a new executor registry
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[string]Executor),
	}
}

// Register adds an executor to the registry
func (r *ExecutorRegistry) Register(e Executor) {
	r.executors[e.ID()] = e
}

// Get retrieves an executor by ID
func (r *ExecutorRegistry) Get(id string) (Executor, bool) {
	e, ok := r.executors[id]
	return e, ok
}

// List returns all registered executor IDs
func (r *ExecutorRegistry) List() []string {
	ids := make([]string, 0, len(r.executors))
	for id := range r.executors {
		ids = append(ids, id)
	}
	return ids
}
