package dispatch

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuotaConfig defines resource limits for a tenant
type QuotaConfig struct {
	MaxConcurrentTickets int `json:"max_concurrent_tickets"`
	MaxTicketsPerHour    int `json:"max_tickets_per_hour"`
	MaxTokensPerDay      int `json:"max_tokens_per_day"`
}

// DefaultQuotaConfig returns defaults for new tenants
func DefaultQuotaConfig() QuotaConfig {
	return QuotaConfig{
		MaxConcurrentTickets: 10,
		MaxTicketsPerHour:    100,
		MaxTokensPerDay:      1_000_000,
	}
}

// QuotaStore abstracts persistent storage for quota usage counters.
// Implement with Redis or PostgreSQL for production use.
type QuotaStore interface {
	// GetUsage loads counters for a tenant from the persistent store
	GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)
	// SaveUsage persists counters for a tenant
	SaveUsage(ctx context.Context, tenantID string, usage *QuotaUsage) error
}

// QuotaManager enforces per-tenant resource limits
type QuotaManager struct {
	mu     sync.RWMutex
	quotas map[string]QuotaConfig // tenantID -> config
	usage  map[string]*QuotaUsage // tenantID -> current usage (hot cache)
	store  QuotaStore             // persistent backend (nil = in-memory only)
}

// QuotaUsage tracks current resource consumption
type QuotaUsage struct {
	ConcurrentTickets int       `json:"concurrent_tickets"`
	TicketsThisHour   int       `json:"tickets_this_hour"`
	TokensToday       int       `json:"tokens_today"`
	HourResetAt       time.Time `json:"hour_reset_at"`
	DayResetAt        time.Time `json:"day_reset_at"`
}

// NewQuotaManager creates a new quota manager with optional persistent store
func NewQuotaManager(store QuotaStore) *QuotaManager {
	return &QuotaManager{
		quotas: make(map[string]QuotaConfig),
		usage:  make(map[string]*QuotaUsage),
		store:  store,
	}
}

// SetQuota sets the quota for a tenant
func (qm *QuotaManager) SetQuota(tenantID string, config QuotaConfig) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.quotas[tenantID] = config
}

// CheckQuota returns an error if the tenant has exceeded their limits
func (qm *QuotaManager) CheckQuota(ctx context.Context, tenantID string) error {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	quota, ok := qm.quotas[tenantID]
	if !ok {
		quota = DefaultQuotaConfig()
	}

	usage := qm.getUsage(tenantID)
	usage.resetIfExpired()

	if usage.ConcurrentTickets >= quota.MaxConcurrentTickets {
		return fmt.Errorf("tenant %q exceeded concurrent ticket limit (%d/%d)",
			tenantID, usage.ConcurrentTickets, quota.MaxConcurrentTickets)
	}
	if usage.TicketsThisHour >= quota.MaxTicketsPerHour {
		return fmt.Errorf("tenant %q exceeded hourly ticket limit (%d/%d)",
			tenantID, usage.TicketsThisHour, quota.MaxTicketsPerHour)
	}
	if usage.TokensToday >= quota.MaxTokensPerDay {
		return fmt.Errorf("tenant %q exceeded daily token limit (%d/%d)",
			tenantID, usage.TokensToday, quota.MaxTokensPerDay)
	}

	return nil
}

// RecordTicketStart increments concurrent ticket count
func (qm *QuotaManager) RecordTicketStart(tenantID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	usage := qm.getOrCreateUsage(tenantID)
	usage.resetIfExpired()
	usage.ConcurrentTickets++
	usage.TicketsThisHour++
	qm.persistUsage(tenantID, usage)
}

// RecordTicketEnd decrements concurrent ticket count
func (qm *QuotaManager) RecordTicketEnd(tenantID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	usage := qm.getOrCreateUsage(tenantID)
	if usage.ConcurrentTickets > 0 {
		usage.ConcurrentTickets--
	}
	qm.persistUsage(tenantID, usage)
}

// RecordTokens adds to the daily token count
func (qm *QuotaManager) RecordTokens(tenantID string, tokens int) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	usage := qm.getOrCreateUsage(tenantID)
	usage.resetIfExpired()
	usage.TokensToday += tokens
	qm.persistUsage(tenantID, usage)
}

func (qm *QuotaManager) getUsage(tenantID string) *QuotaUsage {
	if u, ok := qm.usage[tenantID]; ok {
		return u
	}
	return &QuotaUsage{
		HourResetAt: time.Now().Add(time.Hour),
		DayResetAt:  time.Now().Add(24 * time.Hour),
	}
}

func (qm *QuotaManager) getOrCreateUsage(tenantID string) *QuotaUsage {
	if u, ok := qm.usage[tenantID]; ok {
		return u
	}

	// Try loading from persistent store
	if qm.store != nil {
		if u, err := qm.store.GetUsage(context.Background(), tenantID); err == nil && u != nil {
			qm.usage[tenantID] = u
			return u
		}
	}

	u := &QuotaUsage{
		HourResetAt: time.Now().Add(time.Hour),
		DayResetAt:  time.Now().Add(24 * time.Hour),
	}
	qm.usage[tenantID] = u
	return u
}

func (qm *QuotaManager) persistUsage(tenantID string, usage *QuotaUsage) {
	if qm.store != nil {
		// Fire-and-forget persist — errors are non-fatal
		_ = qm.store.SaveUsage(context.Background(), tenantID, usage)
	}
}

// resetIfExpired resets hourly/daily counters when their windows expire
func (u *QuotaUsage) resetIfExpired() {
	now := time.Now()
	if now.After(u.HourResetAt) {
		u.TicketsThisHour = 0
		u.HourResetAt = now.Add(time.Hour)
	}
	if now.After(u.DayResetAt) {
		u.TokensToday = 0
		u.DayResetAt = now.Add(24 * time.Hour)
	}
}

// InMemoryQuotaStore is a no-op store for development/testing
type InMemoryQuotaStore struct{}

func (s *InMemoryQuotaStore) GetUsage(_ context.Context, _ string) (*QuotaUsage, error) {
	return nil, nil
}
func (s *InMemoryQuotaStore) SaveUsage(_ context.Context, _ string, _ *QuotaUsage) error {
	return nil
}
