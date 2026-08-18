package resilience

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	MaxRequests int           // Maximum requests per window
	Window      time.Duration // Time window
}

// DefaultRateLimiterConfig returns sensible defaults
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		MaxRequests: 100,
		Window:      time.Minute,
	}
}

// RateLimiter implements a simple sliding window rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	config   RateLimiterConfig
	requests []time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		config:   config,
		requests: make([]time.Time, 0),
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.config.Window)

	// Binary search for the first request still in the window.
	// Since requests are appended in order, the slice is sorted.
	lo, hi := 0, len(rl.requests)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if rl.requests[mid].Before(windowStart) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// Prune expired entries in-place (shift the underlying array)
	if lo > 0 {
		copy(rl.requests, rl.requests[lo:])
		rl.requests = rl.requests[:len(rl.requests)-lo]
	}

	if len(rl.requests) >= rl.config.MaxRequests {
		return fmt.Errorf("rate limit exceeded: %d/%d requests in %v",
			len(rl.requests), rl.config.MaxRequests, rl.config.Window)
	}

	rl.requests = append(rl.requests, now)
	return nil
}

// CurrentRate returns the current request count in the window
func (rl *RateLimiter) CurrentRate() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.config.Window)

	count := 0
	for _, t := range rl.requests {
		if t.After(windowStart) {
			count++
		}
	}
	return count
}
