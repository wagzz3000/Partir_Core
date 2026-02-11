package resilience

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal operation
	StateOpen                         // Failing, reject fast
	StateHalfOpen                     // Testing recovery
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           // Failures before opening
	SuccessThreshold int           // Successes in half-open before closing
	Timeout          time.Duration // Time before transitioning open -> half-open
}

// DefaultCircuitConfig returns sensible defaults
func DefaultCircuitConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	name            string
	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	config          CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:   name,
		state:  StateClosed,
		config: config,
	}
}

// Allow checks if a request should be allowed through
func (cb *CircuitBreaker) Allow() error {
	cb.mu.RLock()
	state := cb.state
	lastFailure := cb.lastFailureTime
	cb.mu.RUnlock()

	switch state {
	case StateClosed:
		return nil
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(lastFailure) > cb.config.Timeout {
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.mu.Unlock()
			return nil
		}
		return fmt.Errorf("circuit breaker %q is open", cb.name)
	case StateHalfOpen:
		return nil
	}
	return nil
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	case StateClosed:
		cb.failures = 0 // Reset consecutive failures
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		// Any failure in half-open sends back to open
		cb.state = StateOpen
		cb.successes = 0
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
}
