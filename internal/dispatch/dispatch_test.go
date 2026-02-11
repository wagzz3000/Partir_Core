package dispatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue()

	pq.Push(PriorityItem{TicketID: "low", Priority: 1})
	pq.Push(PriorityItem{TicketID: "high", Priority: 10})
	pq.Push(PriorityItem{TicketID: "medium", Priority: 5})

	assert.Equal(t, 3, pq.Len())

	// Should pop in priority order (highest first)
	item := pq.Pop()
	assert.Equal(t, "high", item.TicketID)

	item = pq.Pop()
	assert.Equal(t, "medium", item.TicketID)

	item = pq.Pop()
	assert.Equal(t, "low", item.TicketID)

	// Empty queue
	assert.Nil(t, pq.Pop())
}

func TestQuotaManager(t *testing.T) {
	qm := NewQuotaManager(nil)
	ctx := context.Background()

	// Set tight quota
	qm.SetQuota("tenant-a", QuotaConfig{
		MaxConcurrentTickets: 2,
		MaxTicketsPerHour:    5,
		MaxTokensPerDay:      1000,
	})

	// Should allow initial tickets
	assert.NoError(t, qm.CheckQuota(ctx, "tenant-a"))

	// Record usage
	qm.RecordTicketStart("tenant-a")
	qm.RecordTicketStart("tenant-a")

	// Should now exceed concurrent limit
	assert.Error(t, qm.CheckQuota(ctx, "tenant-a"))

	// End one ticket
	qm.RecordTicketEnd("tenant-a")
	assert.NoError(t, qm.CheckQuota(ctx, "tenant-a"))
}

func TestQuotaManager_Tokens(t *testing.T) {
	qm := NewQuotaManager(nil)
	ctx := context.Background()

	qm.SetQuota("tenant-b", QuotaConfig{
		MaxConcurrentTickets: 100,
		MaxTicketsPerHour:    100,
		MaxTokensPerDay:      500,
	})

	qm.RecordTokens("tenant-b", 400)
	assert.NoError(t, qm.CheckQuota(ctx, "tenant-b"))

	qm.RecordTokens("tenant-b", 200)
	assert.Error(t, qm.CheckQuota(ctx, "tenant-b"))
}

func TestQuotaManager_DefaultQuota(t *testing.T) {
	qm := NewQuotaManager(nil)
	ctx := context.Background()

	// Unknown tenant gets default quota
	assert.NoError(t, qm.CheckQuota(ctx, "unknown-tenant"))
}
