package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// BackoffConfig controls exponential backoff behavior
type BackoffConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool // Add random jitter to prevent thundering herd
	MaxRetries   int
}

// DefaultBackoffConfig returns sensible defaults
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   5,
	}
}

// RetryWithBackoff executes fn with exponential backoff
// It returns the result of the first successful call, or the last error after exhausting retries.
func RetryWithBackoff(ctx context.Context, config BackoffConfig, fn func(ctx context.Context) error) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if err := fn(ctx); err != nil {
			lastErr = err

			// Don't sleep after the last attempt
			if attempt == config.MaxRetries {
				break
			}

			// Calculate next delay
			actualDelay := delay
			if config.Jitter {
				jitter := time.Duration(rand.Float64() * float64(delay) * 0.5)
				actualDelay = delay + jitter
			}

			// Wait or abort on context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(actualDelay):
			}

			// Increase delay for next attempt
			delay = time.Duration(math.Min(
				float64(delay)*config.Multiplier,
				float64(config.MaxDelay),
			))
		} else {
			return nil // Success
		}
	}

	return lastErr
}
