package omega

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/partir/core/pkg/metrics"
	"github.com/partir/core/pkg/telemetry"
)

// Engine orchestrates gate validation
type Engine struct {
	gates           []Gate
	retryController *RetryController
	infraGate       *InfraGate
}

// NewEngine creates a new Omega engine with ordered gates
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		gates:           []Gate{},
		retryController: NewRetryController(),
		infraGate:       NewInfraGate(pool),
	}
}

// AddGate adds a gate to the pipeline (order matters - cheap first!)
func (e *Engine) AddGate(gate Gate) {
	e.gates = append(e.gates, gate)
}

// DefaultGates returns the standard gate chain in optimal order
func DefaultGates(coreVersion string, existsFunc func(context.Context, string) (bool, error)) []Gate {
	return []Gate{
		NewSchemaGate(),                   // 1. instant
		NewVersionCompatGate(coreVersion), // 2. instant
		NewAllowedCombosGate(),            // 3. fast lookup
		NewBudgetGate(),                   // 4. arithmetic
		NewUniquenessGate(existsFunc),     // 5. DB query
		NewDeterminismGate(),              // 6. hash compare
	}
}

// RunPreflight validates ticket before execution
func (e *Engine) RunPreflight(ctx context.Context, req *GateRequest) *Result {
	ctx, span := telemetry.StartOmegaPreflight(ctx, req.TicketID)
	defer span.End()

	start := time.Now()
	result := &Result{
		Pass: true,
		Metrics: ResultMetrics{
			GatesRun: []string{},
		},
	}

	// 1. Infrastructure Check (Factory 0 - Enforce Green Tag)
	if e.infraGate != nil {
		result.Metrics.GatesRun = append(result.Metrics.GatesRun, e.infraGate.ID())
		defects := e.infraGate.Run(ctx, req)
		if len(defects) > 0 {
			result.Pass = false
			result.Defects = defects
			result.Metrics.GatesFailed = append(result.Metrics.GatesFailed, e.infraGate.ID())
			result.ShouldRetry = false // Infra fail = hard stop
			metrics.GateTotal.WithLabelValues(e.infraGate.ID(), "fail").Inc()
			for _, d := range defects {
				metrics.DefectTotal.WithLabelValues(d.DefectClass, e.infraGate.ID()).Inc()
			}
			return result
		}
		metrics.GateTotal.WithLabelValues(e.infraGate.ID(), "pass").Inc()
	}

	// 2. Logic Check
	preflight := NewPreflightGate()
	defects := preflight.Run(ctx, req)
	result.Metrics.GatesRun = append(result.Metrics.GatesRun, preflight.ID())

	if len(defects) > 0 {
		result.Pass = false
		result.Defects = defects
		result.Metrics.GatesFailed = append(result.Metrics.GatesFailed, preflight.ID())
		result.ShouldRetry = false // Don't retry preflight failures
		metrics.GateTotal.WithLabelValues(preflight.ID(), "fail").Inc()
		for _, d := range defects {
			metrics.DefectTotal.WithLabelValues(d.DefectClass, preflight.ID()).Inc()
		}
	} else {
		metrics.GateTotal.WithLabelValues(preflight.ID(), "pass").Inc()
	}

	result.Metrics.DurationMs = int(time.Since(start).Milliseconds())
	return result
}

// RunPostExecution validates artifacts after execution
func (e *Engine) RunPostExecution(ctx context.Context, req *GateRequest) *Result {
	ctx, span := telemetry.StartOmegaPostExecution(ctx, req.TicketID, req.RunID)
	defer span.End()

	start := time.Now()
	result := &Result{
		Pass: true,
		Metrics: ResultMetrics{
			GatesRun: []string{},
		},
	}

	// Run gates in order (fail-fast)
	for _, gate := range e.gates {
		result.Metrics.GatesRun = append(result.Metrics.GatesRun, gate.ID())

		defects := gate.Run(ctx, req)
		if len(defects) > 0 {
			result.Pass = false
			result.Defects = append(result.Defects, defects...)
			result.Metrics.GatesFailed = append(result.Metrics.GatesFailed, gate.ID())

			// Fail-fast: stop on first gate failure
			break
		}
	}

	// Determine retry strategy if failed
	if !result.Pass {
		result.ShouldRetry, result.DeltaFields = e.retryController.ShouldRetry(req.TicketID, result.Defects)
	}

	result.Metrics.DurationMs = int(time.Since(start).Milliseconds())
	return result
}

// Run is a convenience method that runs both preflight and post-execution
func (e *Engine) Run(ctx context.Context, req *GateRequest) *Result {
	// First run preflight
	preflightResult := e.RunPreflight(ctx, req)
	if !preflightResult.Pass {
		return preflightResult
	}

	// Then run post-execution gates
	return e.RunPostExecution(ctx, req)
}

// Reset clears the retry controller state
func (e *Engine) Reset() {
	e.retryController = NewRetryController()
}
