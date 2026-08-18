package foundry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/partir/core/pkg/plugin"
)

// HTTPExecutor routes work orders to HTTP executor plugins
type HTTPExecutor struct {
	clients map[string]*plugin.HTTPClient
	config  *plugin.Config
}

// NewHTTPExecutor creates an executor that routes to HTTP plugins
func NewHTTPExecutor() (*HTTPExecutor, error) {
	cfg, err := plugin.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading executor config: %w", err)
	}

	clients := make(map[string]*plugin.HTTPClient)
	for name, execCfg := range cfg.Executors {
		client, err := plugin.NewHTTPClientFromConfig(name, execCfg)
		if err != nil {
			return nil, fmt.Errorf("creating client for %s: %w", name, err)
		}
		clients[name] = client
	}

	return &HTTPExecutor{
		clients: clients,
		config:  cfg,
	}, nil
}

// Execute routes the work order to the appropriate HTTP plugin
func (e *HTTPExecutor) Execute(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error) {
	// Determine which HTTP executor to use based on plugin ID or routing hints
	// For now, use the first available executor or one matching the plugin ID
	var client *plugin.HTTPClient

	// Try to find executor matching plugin ID
	if c, ok := e.clients[plug.ID()]; ok {
		client = c
	} else if len(e.clients) > 0 {
		// Use first available executor
		for _, c := range e.clients {
			client = c
			break
		}
	}

	if client == nil {
		return nil, fmt.Errorf("no HTTP executor available for plugin %s", plug.ID())
	}

	// Convert WorkOrder to HTTPWorkOrder
	httpOrder := &plugin.HTTPWorkOrder{
		TicketID:        workOrder.TicketID,
		RunID:           workOrder.RunID,
		TaskType:        workOrder.JobType,
		AlphaRulebookID: workOrder.AlphaRef,
		BetaRulebookID:  workOrder.BetaRef,
		MaxCostUSD:      workOrder.Limits.MaxCostBudget,
		Metadata: map[string]string{
			"attempt": fmt.Sprintf("%d", workOrder.Attempt),
		},
	}

	// Extract Inputs (Prompt, Model, etc.)
	if len(workOrder.Inputs) > 0 {
		var inputs map[string]interface{}
		if err := json.Unmarshal(workOrder.Inputs, &inputs); err == nil {
			if s, ok := inputs["prompt"].(string); ok {
				httpOrder.Prompt = s
			}
			if s, ok := inputs["model"].(string); ok {
				httpOrder.Model = s
			}
			// Future: extract other params like temperature if needed
		}
	}

	// Execute via HTTP
	result, err := client.Execute(ctx, httpOrder)
	if err != nil {
		return nil, fmt.Errorf("HTTP execution failed: %w", err)
	}

	// Convert RunResult back to ExecutionResult
	execResult := &plugin.ExecutionResult{
		CostTokens: result.Telemetry.TokensIn + result.Telemetry.TokensOut,
		CostUSD:    result.Telemetry.CostUSD,
	}

	// Convert artifacts
	for _, a := range result.Artifacts {
		execResult.Artifacts = append(execResult.Artifacts, plugin.Artifact{
			ArtifactID:   a.Path, // Use path as ID
			ArtifactType: a.Type,
			Hash:         a.Hash,
			Data:         a.Data,
		})
	}

	return execResult, nil
}

// AddClient adds an HTTP client for an executor
func (e *HTTPExecutor) AddClient(name string, client *plugin.HTTPClient) {
	e.clients[name] = client
}

// GetClient returns an HTTP client by name
func (e *HTTPExecutor) GetClient(name string) (*plugin.HTTPClient, bool) {
	c, ok := e.clients[name]
	return c, ok
}

// ListClients returns all registered executor names
func (e *HTTPExecutor) ListClients() []string {
	names := make([]string, 0, len(e.clients))
	for name := range e.clients {
		names = append(names, name)
	}
	return names
}

// HealthCheckAll checks health of all registered executors
func (e *HTTPExecutor) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for name, client := range e.clients {
		_, err := client.Health(ctx)
		results[name] = err
	}
	return results
}
