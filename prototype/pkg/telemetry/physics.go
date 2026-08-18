package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RecordSaturation logs a high-severity event when the factory is saturated.
func RecordSaturation(ctx context.Context, deficit float64, drainTimeMins float64) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("factory.saturation", trace.WithAttributes(
		attribute.Float64("velocity_deficit", deficit),
		attribute.Float64("drain_time_mins", drainTimeMins),
		attribute.String("status", "critical"),
	))
}

// RecordBurst logs the details of a burst allocation.
func RecordBurst(ctx context.Context, provider string, demand int, allocationID string) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("factory.burst", trace.WithAttributes(
		attribute.String("provider", provider),
		attribute.Int("demand_tickets", demand),
		attribute.String("allocation_id", allocationID),
	))
}

// RecordCooldown logs when a scaling action was suppressed due to cooldown.
func RecordCooldown(ctx context.Context, reason string) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("factory.cooldown", trace.WithAttributes(
		attribute.String("reason", reason),
	))
}
