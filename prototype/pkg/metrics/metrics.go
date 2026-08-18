// Package metrics provides Prometheus metrics for Partir Core.
// All metrics use the "partir_" prefix for clear product identity.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ticket lifecycle metrics
var (
	// TicketTotal counts tickets by state
	TicketTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_ticket_total",
		Help: "Total number of tickets created, by state",
	}, []string{"state"})

	// TicketStateDuration tracks time spent in each state
	TicketStateDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "partir_ticket_state_duration_seconds",
		Help:    "Time spent in each ticket state",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
	}, []string{"state"})

	// TicketCompleted counts successfully completed tickets
	TicketCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "partir_ticket_completed_total",
		Help: "Total number of successfully completed tickets",
	})

	// TicketLineStop counts tickets that caused a line stop (hard failure)
	TicketLineStop = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_ticket_line_stop_total",
		Help: "Total number of tickets that caused a line stop",
	}, []string{"reason"})
)

// Foundry execution metrics
var (
	// RunTotal counts runs by plugin and job type
	RunTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_run_total",
		Help: "Total number of execution runs",
	}, []string{"plugin", "job_type"})

	// RunAttempt counts run attempts
	RunAttempt = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_run_attempt_total",
		Help: "Total run attempts by attempt number",
	}, []string{"attempt"})

	// RunFailed counts failed runs by reason
	RunFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_run_failed_total",
		Help: "Total failed runs by failure reason",
	}, []string{"reason"})

	// RunDuration tracks run execution time
	RunDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "partir_run_duration_seconds",
		Help:    "Execution duration by plugin and job type",
		Buckets: prometheus.ExponentialBuckets(0.5, 2, 12), // 0.5s to ~34 min
	}, []string{"plugin", "job_type"})
)

// Omega validation metrics (critical for DMAIC)
var (
	// GateTotal counts gate executions by gate and result
	GateTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_omega_gate_total",
		Help: "Total gate executions by gate ID and result",
	}, []string{"gate", "result"})

	// DefectTotal counts defects by class and gate (Pareto source)
	DefectTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_omega_defect_total",
		Help: "Total defects by defect class and gate",
	}, []string{"defect_class", "gate"})

	// PassRate tracks the rolling pass rate
	PassRate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "partir_omega_pass_rate",
		Help: "Current rolling pass rate (0.0-1.0)",
	})
)

// Token and cost control metrics
var (
	// AITokensTotal counts tokens consumed by model and plugin
	AITokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_ai_tokens_total",
		Help: "Total AI tokens consumed",
	}, []string{"model", "plugin"})

	// AICostTotal tracks cost in USD
	AICostTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_ai_cost_usd_total",
		Help: "Total AI cost in USD",
	}, []string{"model"})

	// AIRetryTotal counts retries triggered by defect class
	AIRetryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_ai_retry_total",
		Help: "Total AI retries by defect class",
	}, []string{"defect_class"})
)

// Factory-level metrics
var (
	// FactoryLineStopTotal counts factory-wide line stops
	FactoryLineStopTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_factory_line_stop_total",
		Help: "Total factory line stops by cause",
	}, []string{"cause"})

	// FactoryThroughput tracks tickets per minute
	FactoryThroughput = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "partir_factory_throughput_per_minute",
		Help: "Current factory throughput (tickets/min)",
	})
)
