// Package telemetry - Structured logging with trace correlation
package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// Logger returns a structured logger with trace correlation
func Logger(ctx context.Context) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Add trace context if available
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger = logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	return logger
}

// LoggerWith returns a logger with additional attributes
func LoggerWith(ctx context.Context, attrs ...any) *slog.Logger {
	return Logger(ctx).With(attrs...)
}

// TicketLogger returns a logger preconfigured for ticket operations
func TicketLogger(ctx context.Context, ticketID string) *slog.Logger {
	return Logger(ctx).With(
		slog.String("ticket_id", ticketID),
	)
}

// RunLogger returns a logger preconfigured for run operations
func RunLogger(ctx context.Context, ticketID, runID string) *slog.Logger {
	return Logger(ctx).With(
		slog.String("ticket_id", ticketID),
		slog.String("run_id", runID),
	)
}

// GateLogger returns a logger preconfigured for gate operations
func GateLogger(ctx context.Context, gateID string) *slog.Logger {
	return Logger(ctx).With(
		slog.String("gate_id", gateID),
	)
}
