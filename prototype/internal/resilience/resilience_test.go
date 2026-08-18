package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================
// Circuit Breaker Tests
// ============================================================

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	assert.Equal(t, StateClosed, cb.State())

	// 3 failures should open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Should reject
	assert.Error(t, cb.Allow())
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open on Allow
	assert.NoError(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow() // transition to half-open

	cb.RecordSuccess()
	assert.Equal(t, StateHalfOpen, cb.State())
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenBackToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	// Failure in half-open goes back to open
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

// ============================================================
// Rate Limiter Tests
// ============================================================

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		MaxRequests: 3,
		Window:      time.Second,
	})

	assert.NoError(t, rl.Allow())
	assert.NoError(t, rl.Allow())
	assert.NoError(t, rl.Allow())
	assert.Error(t, rl.Allow()) // 4th should fail
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		MaxRequests: 2,
		Window:      50 * time.Millisecond,
	})

	assert.NoError(t, rl.Allow())
	assert.NoError(t, rl.Allow())
	assert.Error(t, rl.Allow())

	time.Sleep(60 * time.Millisecond)
	assert.NoError(t, rl.Allow()) // Window expired, should work
}

// ============================================================
// Backoff Tests
// ============================================================

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(context.Background(), BackoffConfig{
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2,
		MaxRetries:   3,
	}, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("not yet")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryWithBackoff_Exhausted(t *testing.T) {
	err := RetryWithBackoff(context.Background(), BackoffConfig{
		InitialDelay: time.Millisecond,
		MaxDelay:     5 * time.Millisecond,
		Multiplier:   2,
		MaxRetries:   2,
	}, func(ctx context.Context) error {
		return errors.New("always fails")
	})

	assert.Error(t, err)
	assert.Equal(t, "always fails", err.Error())
}

func TestRetryWithBackoff_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, BackoffConfig{
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2,
		MaxRetries:   10,
	}, func(ctx context.Context) error {
		return errors.New("fail")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// ============================================================
// Idempotency Tests (in-memory)
// ============================================================

func TestIdempotencyStore_Memory(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	ctx := context.Background()

	exists, err := store.Check(ctx, "key-1")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = store.Record(ctx, "key-1")
	assert.NoError(t, err)

	exists, err = store.Check(ctx, "key-1")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Different key
	exists, err = store.Check(ctx, "key-2")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// ============================================================
// Fallback Chain Tests
// ============================================================

func TestFallbackChain_FirstProviderSucceeds(t *testing.T) {
	fc := NewFallbackChain()
	fc.AddProvider("primary", func(ctx context.Context) (interface{}, error) {
		return "primary-result", nil
	}, nil)
	fc.AddProvider("fallback", func(ctx context.Context) (interface{}, error) {
		return "fallback-result", nil
	}, nil)

	result, err := fc.Execute(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "primary-result", result)
}

func TestFallbackChain_FallsBack(t *testing.T) {
	fc := NewFallbackChain()
	fc.AddProvider("primary", func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("primary down")
	}, nil)
	fc.AddProvider("fallback", func(ctx context.Context) (interface{}, error) {
		return "fallback-result", nil
	}, nil)

	result, err := fc.Execute(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "fallback-result", result)
}

func TestFallbackChain_AllExhausted(t *testing.T) {
	fc := NewFallbackChain()
	fc.AddProvider("a", func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("a failed")
	}, nil)
	fc.AddProvider("b", func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("b failed")
	}, nil)

	_, err := fc.Execute(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all providers exhausted")
}

func TestFallbackChain_WithCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("primary", CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          time.Minute,
	})
	// Trip the breaker
	cb.RecordFailure()

	fc := NewFallbackChain()
	fc.AddProvider("primary", func(ctx context.Context) (interface{}, error) {
		return "primary-result", nil
	}, cb) // Circuit is open
	fc.AddProvider("fallback", func(ctx context.Context) (interface{}, error) {
		return "fallback-result", nil
	}, nil)

	result, err := fc.Execute(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "fallback-result", result) // Should skip primary
}
