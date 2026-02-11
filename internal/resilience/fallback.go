package resilience

import (
	"context"
	"fmt"
	"sync"
)

// FallbackChain represents an ordered list of providers with automatic failover
type FallbackChain struct {
	mu        sync.RWMutex
	providers []NamedProvider
}

// NamedProvider wraps a provider function with a name
type NamedProvider struct {
	Name    string
	Execute func(ctx context.Context) (interface{}, error)
	Breaker *CircuitBreaker
}

// NewFallbackChain creates a new fallback chain
func NewFallbackChain() *FallbackChain {
	return &FallbackChain{
		providers: make([]NamedProvider, 0),
	}
}

// AddProvider adds a provider with its circuit breaker
func (fc *FallbackChain) AddProvider(name string, fn func(ctx context.Context) (interface{}, error), breaker *CircuitBreaker) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.providers = append(fc.providers, NamedProvider{
		Name:    name,
		Execute: fn,
		Breaker: breaker,
	})
}

// Execute tries each provider in order, falling back on failure
func (fc *FallbackChain) Execute(ctx context.Context) (interface{}, error) {
	fc.mu.RLock()
	providers := make([]NamedProvider, len(fc.providers))
	copy(providers, fc.providers)
	fc.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		// Check circuit breaker
		if p.Breaker != nil {
			if err := p.Breaker.Allow(); err != nil {
				lastErr = fmt.Errorf("provider %q circuit open: %w", p.Name, err)
				continue
			}
		}

		result, err := p.Execute(ctx)
		if err != nil {
			lastErr = fmt.Errorf("provider %q failed: %w", p.Name, err)
			if p.Breaker != nil {
				p.Breaker.RecordFailure()
			}
			continue
		}

		if p.Breaker != nil {
			p.Breaker.RecordSuccess()
		}
		return result, nil
	}

	return nil, fmt.Errorf("all providers exhausted, last error: %w", lastErr)
}
