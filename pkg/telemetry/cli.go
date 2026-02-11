// Package telemetry - CLI tracing helpers
package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// WithSpan wraps a function with a tracing span
func WithSpan(ctx context.Context, tracer trace.Tracer, spanName string, attrs []attribute.KeyValue, fn func(ctx context.Context) error) error {
	ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// InitCLI initializes telemetry for a CLI command and returns the provider
// Call provider.Shutdown(ctx) in defer when done
func InitCLI(serviceName string) (*Provider, context.Context, error) {
	ctx := context.Background()

	// Check if telemetry is disabled
	if os.Getenv("PARTIR_TELEMETRY_DISABLED") == "true" {
		return nil, ctx, nil
	}

	cfg := DefaultConfig(serviceName)
	provider, err := Init(ctx, cfg)
	if err != nil {
		// Non-fatal: just log and continue without telemetry
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
		return nil, ctx, nil
	}

	return provider, ctx, nil
}

// ShutdownCLI gracefully shuts down telemetry provider if it exists
func ShutdownCLI(provider *Provider, ctx context.Context) {
	if provider != nil {
		provider.Shutdown(ctx)
	}
}
