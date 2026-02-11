package foundry

import (
	"context"

	"github.com/partir/core/pkg/plugin"
)

// Executor interface for different execution backends
type Executor interface {
	Execute(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error)
}

// LocalExecutor runs plugins in-process
type LocalExecutor struct {
	plugins *plugin.Registry
}

// NewLocalExecutor creates a new local executor
func NewLocalExecutor(plugins *plugin.Registry) *LocalExecutor {
	return &LocalExecutor{plugins: plugins}
}

// Execute runs the plugin's Execute method directly
func (e *LocalExecutor) Execute(ctx context.Context, plug plugin.Plugin, workOrder plugin.WorkOrder) (*plugin.ExecutionResult, error) {
	return plug.Execute(ctx, workOrder)
}
