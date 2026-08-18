package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/partir/core/pkg/plugin"
)

// ThroughputController manages the factory scaling logic based on physics metrics.
// It implements the Manager logic described in factory_physics.md.
type ThroughputController struct {
	provider    plugin.ComputeProvider
	config      plugin.ComputeConfig
	lastBurst   time.Time
	targetDrain time.Duration // Maximum allowed drain time (SLA)

	// Internal state
	currentDeficit float64
	currentDrain   time.Duration
}

// ScalingAction represents the decision made by the controller.
type ScalingAction string

const (
	ActionMaintain ScalingAction = "MAINTAIN"
	ActionBurst    ScalingAction = "BURST"
	ActionRelease  ScalingAction = "RELEASE"
)

var (
	// ControllerDecisions counts scaling actions
	ControllerDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "partir_controller_decisions_total",
		Help: "Total scaling decisions made by the throughput controller",
	}, []string{"action", "reason"})
)

// NewThroughputController initializes the controller with a provider.
func NewThroughputController(provider plugin.ComputeProvider, cfg plugin.ComputeConfig, slaTarget time.Duration) *ThroughputController {
	return &ThroughputController{
		provider:    provider,
		config:      cfg,
		targetDrain: slaTarget,
	}
}

// Evaluate performs the physics calculation and executes scaling actions.
// This should be called periodically (e.g., every 30s) by the main loop.
func (c *ThroughputController) Evaluate(ctx context.Context) error {
	// 1. Gather Physics Snapshot
	lambda := getIncomingRate()        // events/min
	mu := getEffectiveProcessingRate() // events/min
	backlog := getBacklogSize()        // events count

	// Prevent division by zero
	if mu <= 0.1 {
		return nil // Waiting for baseline data
	}

	// 2. Calculate Invariants
	velocityDeficit := lambda > mu
	drainTime := time.Duration(float64(backlog)/mu) * time.Minute

	c.currentDeficit = lambda - mu
	c.currentDrain = drainTime

	// 3. Decision Logic
	action := ActionMaintain
	reason := "Stable"

	if velocityDeficit {
		action = ActionBurst
		reason = "VelocityDeficit"
	} else if drainTime > c.targetDrain {
		action = ActionBurst
		reason = fmt.Sprintf("SLABreach_DrainTime_%.1fm", drainTime.Minutes())
	}

	// 4. Cooldown Check
	if action == ActionBurst && time.Since(c.lastBurst) < c.config.ScaleUpWindow {
		action = ActionMaintain
		reason = "Cooldown"
	}

	// 5. Execute
	ControllerDecisions.WithLabelValues(string(action), reason).Inc()

	switch action {
	case ActionBurst:
		return c.executeBurst(ctx, backlog)
	case ActionMaintain:
		// Potential release logic could go here if drain time is very low
		return nil
	}

	return nil
}

func (c *ThroughputController) executeBurst(ctx context.Context, backlog float64) error {
	// Simple burst strategy: request capacity for 50% of backlog to clear it fast
	demand := plugin.TicketVolume(backlog / 2)
	if demand < 10 {
		return nil // Don't burst for tiny queues
	}

	_, err := c.provider.Burst(ctx, demand)
	if err != nil {
		return fmt.Errorf("failed to burst: %w", err)
	}

	c.lastBurst = time.Now()
	return nil
}

// Helper stubs that will hook into the real Prometheus metrics
// In a real implementation, these would query the prometheus client API or internal accumulators.
func getIncomingRate() float64 {
	// Implementation would read rate(metrics.TicketTotal[1m])
	// For now returning 0 to allow compilation
	return 0
}

func getEffectiveProcessingRate() float64 {
	// Implementation would read rate(metrics.ThroughputMu[5m])
	return 1.0
}

func getBacklogSize() float64 {
	// Implementation would read metrics.TicketTotal{state="queued"}
	return 0
}
