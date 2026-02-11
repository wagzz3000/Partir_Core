package mocks

import (
	"context"
	"testing"

	"github.com/partir/core/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestMockTicketRepository(t *testing.T) {
	m := new(MockTicketRepository)
	ctx := context.Background()
	ticket := &storage.Ticket{ID: "t1"}

	m.On("Create", ctx, ticket).Return(nil)

	err := m.Create(ctx, ticket)
	assert.NoError(t, err)

	m.AssertExpectations(t)
}

func TestMockBlobRepository(t *testing.T) {
	m := new(MockBlobRepository)
	ctx := context.Background()
	hash := "abc"
	data := []byte("data")

	m.On("PutBlob", ctx, hash, data, "text/plain").Return("s3://bucket/abc", nil)

	uri, err := m.PutBlob(ctx, hash, data, "text/plain")
	assert.NoError(t, err)
	assert.Equal(t, "s3://bucket/abc", uri)

	m.AssertExpectations(t)
}
