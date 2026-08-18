// Package telemetry - Partir-specific metrics using OpenTelemetry
package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all Partir-specific OTel metrics
type Metrics struct {
	// Ticket lifecycle
	TicketTotal         metric.Int64Counter
	TicketStateDuration metric.Float64Histogram
	TicketCompleted     metric.Int64Counter
	TicketLineStop      metric.Int64Counter

	// Foundry execution
	RunTotal    metric.Int64Counter
	RunAttempt  metric.Int64Counter
	RunFailed   metric.Int64Counter
	RunDuration metric.Float64Histogram

	// Omega validation
	GateTotal   metric.Int64Counter
	DefectTotal metric.Int64Counter

	// Token & cost
	AITokensTotal metric.Int64Counter
	AICostTotal   metric.Float64Counter
	AIRetryTotal  metric.Int64Counter

	// Factory-level
	FactoryLineStop   metric.Int64Counter
	FactoryThroughput metric.Float64Gauge
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// GetMetrics returns the singleton Metrics instance
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = initMetrics()
	})
	return globalMetrics
}

func initMetrics() *Metrics {
	meter := otel.Meter("partir-core")
	m := &Metrics{}

	// Ticket metrics
	m.TicketTotal, _ = meter.Int64Counter("partir.ticket.total",
		metric.WithDescription("Total tickets by state"))
	m.TicketStateDuration, _ = meter.Float64Histogram("partir.ticket.state_duration_seconds",
		metric.WithDescription("Time spent in each ticket state"))
	m.TicketCompleted, _ = meter.Int64Counter("partir.ticket.completed_total",
		metric.WithDescription("Total successfully completed tickets"))
	m.TicketLineStop, _ = meter.Int64Counter("partir.ticket.line_stop_total",
		metric.WithDescription("Tickets causing line stops"))

	// Run metrics
	m.RunTotal, _ = meter.Int64Counter("partir.run.total",
		metric.WithDescription("Total execution runs"))
	m.RunAttempt, _ = meter.Int64Counter("partir.run.attempt_total",
		metric.WithDescription("Run attempts by attempt number"))
	m.RunFailed, _ = meter.Int64Counter("partir.run.failed_total",
		metric.WithDescription("Failed runs by reason"))
	m.RunDuration, _ = meter.Float64Histogram("partir.run.duration_seconds",
		metric.WithDescription("Run duration"))

	// Omega metrics
	m.GateTotal, _ = meter.Int64Counter("partir.omega.gate_total",
		metric.WithDescription("Gate executions by result"))
	m.DefectTotal, _ = meter.Int64Counter("partir.omega.defect_total",
		metric.WithDescription("Defects by class and gate"))

	// AI/Cost metrics
	m.AITokensTotal, _ = meter.Int64Counter("partir.ai.tokens_total",
		metric.WithDescription("AI tokens consumed"))
	m.AICostTotal, _ = meter.Float64Counter("partir.ai.cost_usd_total",
		metric.WithDescription("AI cost in USD"))
	m.AIRetryTotal, _ = meter.Int64Counter("partir.ai.retry_total",
		metric.WithDescription("AI retries by defect class"))

	// Factory metrics
	m.FactoryLineStop, _ = meter.Int64Counter("partir.factory.line_stop_total",
		metric.WithDescription("Factory line stops"))
	m.FactoryThroughput, _ = meter.Float64Gauge("partir.factory.throughput_per_minute",
		metric.WithDescription("Factory throughput"))

	return m
}

// RecordTicketState records a ticket state transition
func (m *Metrics) RecordTicketState(ctx context.Context, state string) {
	m.TicketTotal.Add(ctx, 1, metric.WithAttributes(
		LabelState(state),
	))
}

// RecordRun records a run execution
func (m *Metrics) RecordRun(ctx context.Context, plugin, jobType string, attempt int) {
	m.RunTotal.Add(ctx, 1, metric.WithAttributes(
		LabelPlugin(plugin),
		LabelJobType(jobType),
	))
	m.RunAttempt.Add(ctx, 1, metric.WithAttributes(
		LabelAttempt(attempt),
	))
}

// RecordGate records a gate execution result
func (m *Metrics) RecordGate(ctx context.Context, gate string, passed bool) {
	result := "pass"
	if !passed {
		result = "fail"
	}
	m.GateTotal.Add(ctx, 1, metric.WithAttributes(
		LabelGate(gate),
		LabelResult(result),
	))
}

// RecordDefect records a defect
func (m *Metrics) RecordDefect(ctx context.Context, defectClass, gate string) {
	m.DefectTotal.Add(ctx, 1, metric.WithAttributes(
		LabelDefectClass(defectClass),
		LabelGate(gate),
	))
}
