// Package plugin - HTTP Protocol for Executor Plugins
//
// Plugins expose these endpoints:
//
//	GET  /health        - Health check
//	GET  /capabilities  - Models, context limits, features
//	POST /execute       - WorkOrder → RunResult
//	POST /estimate      - Optional cost/latency estimate
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPWorkOrder is sent to the plugin for execution via HTTP
type HTTPWorkOrder struct {
	// Identifiers
	TicketID string `json:"ticket_id"`
	RunID    string `json:"run_id"`

	// What to do
	TaskType string `json:"task_type"` // "generate", "transform", "validate"

	// LLM parameters
	Prompt       string   `json:"prompt"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	MaxTokens    int      `json:"max_tokens"`
	Temperature  float64  `json:"temperature"`
	StopTokens   []string `json:"stop_tokens,omitempty"`
	Model        string   `json:"model,omitempty"` // Override default model

	// Context
	AlphaRulebookID string            `json:"alpha_rulebook_id,omitempty"`
	BetaRulebookID  string            `json:"beta_rulebook_id,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`

	// Budget constraints
	MaxCostUSD   float64 `json:"max_cost_usd,omitempty"`
	MaxLatencyMs int64   `json:"max_latency_ms,omitempty"`

	// Features requested
	WantPlan      bool `json:"want_plan,omitempty"`
	WantPatch     bool `json:"want_patch,omitempty"`
	WantArtifacts bool `json:"want_artifacts,omitempty"`
}

// RunResult is returned from the plugin after execution
type RunResult struct {
	// Maps directly to AgentResult for Omega validation
	AgentResult

	// Additional HTTP-specific fields
	RequestID string `json:"request_id,omitempty"` // Plugin's request ID
	Output    string `json:"output,omitempty"`     // Raw text output (for LLM generation)
}

// ExecutorCapabilitiesResponse from GET /capabilities
type ExecutorCapabilitiesResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Models       []string `json:"models"`
	DefaultModel string   `json:"default_model"`

	// Context and limits
	MaxContextLen int `json:"max_context_len"`
	MaxOutputLen  int `json:"max_output_len"`

	// Features
	SupportsPlan      bool `json:"supports_plan"`
	SupportsPatch     bool `json:"supports_patch"`
	SupportsArtifacts bool `json:"supports_artifacts"`
	SupportsStreaming bool `json:"supports_streaming"`

	// Cost info (for routing decisions)
	CostPerPromptToken     float64 `json:"cost_per_prompt_token"`
	CostPerCompletionToken float64 `json:"cost_per_completion_token"`

	// Rate limits
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
}

// HealthResponse from GET /health
type HealthResponse struct {
	Status    string `json:"status"` // "ok", "degraded", "error"
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// EstimateRequest for POST /estimate
type EstimateRequest struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
	Model     string `json:"model,omitempty"`
}

// EstimateResponse from POST /estimate
type EstimateResponse struct {
	EstimatedTokensIn  int     `json:"estimated_tokens_in"`
	EstimatedTokensOut int     `json:"estimated_tokens_out"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
	EstimatedLatencyMs int64   `json:"estimated_latency_ms"`
}

// HTTPClient communicates with executor plugins over HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	name       string
}

// NewHTTPClient creates a new HTTP client for an executor plugin
func NewHTTPClient(name, baseURL string) *HTTPClient {
	return &HTTPClient{
		name:    name,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for LLM execution
		},
	}
}

// Health checks if the plugin is healthy
func (c *HTTPClient) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("creating health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health check returned %d: %s", resp.StatusCode, string(body))
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("decoding health response: %w", err)
	}

	return &health, nil
}

// Capabilities returns the plugin's capabilities
func (c *HTTPClient) Capabilities(ctx context.Context) (*ExecutorCapabilitiesResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/capabilities", nil)
	if err != nil {
		return nil, fmt.Errorf("creating capabilities request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("capabilities request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("capabilities returned %d: %s", resp.StatusCode, string(body))
	}

	var caps ExecutorCapabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&caps); err != nil {
		return nil, fmt.Errorf("decoding capabilities response: %w", err)
	}

	return &caps, nil
}

// Execute sends a work order and waits for the result
func (c *HTTPClient) Execute(ctx context.Context, order *HTTPWorkOrder) (*RunResult, error) {
	body, err := json.Marshal(order)
	if err != nil {
		return nil, fmt.Errorf("marshaling work order: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/execute", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating execute request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("execute returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result RunResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding run result: %w", err)
	}

	return &result, nil
}

// Estimate gets a cost/latency estimate (optional endpoint)
func (c *HTTPClient) Estimate(ctx context.Context, est *EstimateRequest) (*EstimateResponse, error) {
	body, err := json.Marshal(est)
	if err != nil {
		return nil, fmt.Errorf("marshaling estimate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/estimate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating estimate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Estimate is optional, return nil on failure
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Estimate is optional
		return nil, nil
	}

	var result EstimateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil // Optional endpoint, ignore errors
	}

	return &result, nil
}

// Name returns the executor name
func (c *HTTPClient) Name() string {
	return c.name
}

// URL returns the base URL
func (c *HTTPClient) URL() string {
	return c.baseURL
}
