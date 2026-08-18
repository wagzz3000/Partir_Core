package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Factory Physics metrics (Throughput Controller)
// See: factory_physics.md for mathematical derivation.
var (
	// throughput_mu tracks the effective processing rate per stage.
	// We use a Counter for total tickets processed, and rate() in PromQL gives us tickets/sec.
	// Labels: stage (alpha, beta, omega, foundry), hardware_tier (baseline, burst_light, burst_heavy)
	ThroughputMu = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_factory_throughput_mu_total",
		Help: "Total number of tickets processed by stage and hardware tier (Rate gives Mu)",
	}, []string{"stage", "hardware_tier"})

	// token_burn tracks total input/output tokens consumed per stage.
	// Labels: stage, model, operation_type (generation, classification)
	TokenBurn = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_factory_token_burn_total",
		Help: "Total tokens consumed (Input + Output) per stage",
	}, []string{"stage", "model", "operation_type"})

	// correction_loop_depth tracks the distribution of retries required to pass Omega.
	// Critical for detecting 'Death Spiral' (R > 1.2).
	// Labels: stage, defect_class
	CorrectionLoopDepth = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "partir_factory_correction_loop_depth",
		Help:    "Distribution of retry counts per ticket",
		Buckets: []float64{0, 1, 2, 3, 5, 8, 13, 21}, // Fibonacci buckets for retry depth
	}, []string{"stage", "defect_class"})

	// defect_rate tracks the number of defects found by Omega.
	// Labels: stage, gate_id, defect_class
	DefectDetection = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_factory_defect_detection_total",
		Help: "Total number of defects detected by Gate and Class",
	}, []string{"stage", "gate_id", "defect_class"})

	// latency_seconds tracks wall-clock time per ticket per stage.
	// Labels: stage, hardware_tier
	StageLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "partir_factory_stage_latency_seconds",
		Help:    "Wall-clock duration per stage",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17 mins (1024s)
	}, []string{"stage", "hardware_tier"})
)
