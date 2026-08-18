// Package telemetry - Tracing spans for CLI operations
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer names for each module
const (
	TracerAlpha   = "partir-alpha"
	TracerBeta    = "partir-beta"
	TracerFoundry = "partir-foundry"
	TracerOmega   = "partir-omega"
)

// StartSpan starts a new span with common Partir attributes
func StartSpan(ctx context.Context, tracerName, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
}

// EndSpan ends a span with error handling
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// Alpha CLI spans
func StartAlphaInit(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerAlpha, "alpha.init",
		attribute.String("rulebook.name", name),
	)
}

func StartAlphaLint(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerAlpha, "alpha.lint",
		attribute.String("rulebook.name", name),
	)
}

func StartAlphaBuild(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerAlpha, "alpha.build",
		attribute.String("rulebook.name", name),
	)
}

func StartAlphaPromote(ctx context.Context, name, version string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerAlpha, "alpha.promote",
		attribute.String("rulebook.name", name),
		attribute.String("rulebook.version", version),
	)
}

// Beta CLI spans
func StartBetaInit(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerBeta, "beta.init",
		attribute.String("rulebook.name", name),
	)
}

func StartBetaLint(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerBeta, "beta.lint",
		attribute.String("rulebook.name", name),
	)
}

func StartBetaBuild(ctx context.Context, name string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerBeta, "beta.build",
		attribute.String("rulebook.name", name),
	)
}

func StartBetaPromote(ctx context.Context, name, version string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerBeta, "beta.promote",
		attribute.String("rulebook.name", name),
		attribute.String("rulebook.version", version),
	)
}

// Foundry CLI spans
func StartFoundrySubmit(ctx context.Context, pluginID, jobType string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerFoundry, "foundry.submit",
		attribute.String("plugin_id", pluginID),
		attribute.String("job_type", jobType),
	)
}

func StartFoundryRun(ctx context.Context, ticketID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerFoundry, "foundry.run",
		attribute.String("ticket_id", ticketID),
	)
}

func StartFoundryDispatch(ctx context.Context, ticketID, pluginID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerFoundry, "foundry.dispatch",
		attribute.String("ticket_id", ticketID),
		attribute.String("plugin_id", pluginID),
	)
}

func StartFoundryExecute(ctx context.Context, ticketID, runID, pluginID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerFoundry, "foundry.execute",
		attribute.String("ticket_id", ticketID),
		attribute.String("run_id", runID),
		attribute.String("plugin_id", pluginID),
	)
}

// Omega gate spans
func StartOmegaPreflight(ctx context.Context, ticketID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerOmega, "omega.preflight",
		attribute.String("ticket_id", ticketID),
	)
}

func StartOmegaPostExecution(ctx context.Context, ticketID, runID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerOmega, "omega.post_execution",
		attribute.String("ticket_id", ticketID),
		attribute.String("run_id", runID),
	)
}

func StartOmegaGate(ctx context.Context, gateID, ticketID string) (context.Context, trace.Span) {
	return StartSpan(ctx, TracerOmega, "omega.gate."+gateID,
		attribute.String("gate_id", gateID),
		attribute.String("ticket_id", ticketID),
	)
}
