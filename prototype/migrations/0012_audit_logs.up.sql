-- Migration: 000012_audit_logs
-- Purpose: Create immutable audit log for compliance.

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(255) NOT NULL,
    actor_id VARCHAR(255) NOT NULL,      -- User ID or Service Account ID
    action VARCHAR(255) NOT NULL,        -- e.g., "ticket.submit", "scope.approve"
    resource_type VARCHAR(255) NOT NULL, -- e.g., "ticket", "rulebook"
    resource_id VARCHAR(255) NOT NULL,   -- UUID or ID of the resource
    changes JSONB,                       -- Diff or snapshot of changes
    metadata JSONB DEFAULT '{}',         -- IP, UserAgent, RequestID
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for compliance queries
CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_actor ON audit_logs(actor_id);
CREATE INDEX idx_audit_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at);

-- Enforce immutability: prevent UPDATE and DELETE on audit_logs
CREATE OR REPLACE FUNCTION audit_logs_immutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs table is immutable: % operations are not allowed', TG_OP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_audit_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_immutable();

CREATE TRIGGER prevent_audit_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_immutable();

-- Revoke destructive permissions from the application role
-- (Assumes app connects as 'partir' role)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'partir') THEN
        REVOKE UPDATE, DELETE ON audit_logs FROM partir;
    END IF;
END $$;

