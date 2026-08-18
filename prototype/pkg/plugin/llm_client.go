package plugin

import (
	"context"
	"fmt"
)

// LLMClient adapts an HTTPClient to the Executor interface for LLM operations.
// It maps the internal ExecuteRequest to the HTTP protocol's HTTPWorkOrder.
type LLMClient struct {
	client *HTTPClient
}

// NewLLMClient creates a new LLM client from configuration.
// If name is empty, it uses the first available executor from ~/.partir/config.yaml.
func NewLLMClient(name string) (*LLMClient, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Executors) == 0 {
		return nil, fmt.Errorf("no executors configured in %s. Please run 'partir executors add' first", DefaultConfigPath())
	}

	var execName string
	var execCfg ExecutorConfig
	var found bool

	if name != "" {
		execCfg, found = cfg.GetExecutor(name)
		if !found {
			return nil, fmt.Errorf("executor %q not found", name)
		}
		execName = name
	} else {
		// Pick first available executor
		for n, c := range cfg.Executors {
			execName = n
			execCfg = c
			break
		}
	}

	client, err := NewHTTPClientFromConfig(execName, execCfg)
	if err != nil {
		return nil, err
	}

	return &LLMClient{client: client}, nil
}

// NewLLMClientWithHTTP creates a new LLM client using an existing HTTP client.
// Useful for testing or custom client configuration.
func NewLLMClientWithHTTP(client *HTTPClient) *LLMClient {
	return &LLMClient{client: client}
}

// ID returns the executor ID (name)
func (c *LLMClient) ID() string {
	return c.client.Name()
}

// Name returns the executor name
func (c *LLMClient) Name() string {
	return c.client.Name()
}

// Execute runs the LLM generation via HTTP
func (c *LLMClient) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	// Map ExecuteRequest to HTTPWorkOrder
	order := &HTTPWorkOrder{
		TicketID:        req.TicketID,
		RunID:           req.RunID,
		TaskType:        "generate", // Signal this is a generation task
		Prompt:          req.Prompt,
		SystemPrompt:    req.SystemPrompt,
		MaxTokens:       req.MaxTokens,
		Temperature:     req.Temperature,
		StopTokens:      req.StopTokens,
		AlphaRulebookID: req.AlphaRulebookID,
		BetaRulebookID:  req.BetaRulebookID,
		Metadata:        req.Metadata,
		MaxCostUSD:      req.MaxCostUSD,
	}

	res, err := c.client.Execute(ctx, order)
	if err != nil {
		return nil, err
	}

	// Map RunResult to ExecuteResponse
	return &ExecuteResponse{
		Output:           res.Output,
		FinishReason:     res.Telemetry.FinishReason,
		PromptTokens:     res.Telemetry.TokensIn,
		CompletionTokens: res.Telemetry.TokensOut,
		TotalTokens:      res.Telemetry.TokensIn + res.Telemetry.TokensOut,
		CostUSD:          res.Telemetry.CostUSD,
		Model:            res.Telemetry.Model,
		Provider:         res.Telemetry.Provider,
		LatencyMs:        res.Telemetry.LatencyMs,
	}, nil
}

// Capabilities returns the capabilities (stub for now as it requires async fetch)
func (c *LLMClient) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{
		Models: []string{"default"},
	}
}

// HealthCheck checks if the backend is reachable
func (c *LLMClient) HealthCheck(ctx context.Context) error {
	_, err := c.client.Health(ctx)
	return err
}
