-- Migration: 000013_resilience_tables
-- Dead Letter Queue and Idempotency Keys for reliability

-- Dead Letter Queue for permanently failed tickets
CREATE TABLE tickets_dlq (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    reason VARCHAR(500) NOT NULL,
    last_error TEXT,
    attempts INT NOT NULL DEFAULT 0,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_dlq_tenant ON tickets_dlq(tenant_id);
CREATE INDEX idx_dlq_unresolved ON tickets_dlq(tenant_id) WHERE resolved_at IS NULL;
CREATE INDEX idx_dlq_created ON tickets_dlq(created_at);

-- Idempotency Keys for deduplication
CREATE TABLE idempotency_keys (
    key VARCHAR(512) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cleanup function for expired idempotency keys (call via pg_cron or application-level scheduler)
-- Deletes keys older than the specified retention period (default 24 hours)
CREATE OR REPLACE FUNCTION cleanup_idempotency_keys(retention_hours INT DEFAULT 24)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM idempotency_keys
    WHERE created_at < NOW() - (retention_hours || ' hours')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Index to efficiently find expired keys
CREATE INDEX idx_idempotency_created ON idempotency_keys(created_at);

-- Schedule cleanup every hour (requires pg_cron extension)
-- Uncomment after enabling pg_cron:
-- SELECT cron.schedule('cleanup-idempotency-keys', '0 * * * *', $$SELECT cleanup_idempotency_keys(24)$$);

