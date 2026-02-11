-- Migration: 000011_add_tenant_id
-- Purpose: Add multi-tenancy support by isolating data per tenant.
-- Infrastructure tables (workers, stations, memory_pairs) remain global/shared for now.

-- Core Data Tables
ALTER TABLE rulebooks ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE tickets ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE runs ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE artifacts ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE defects ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

-- Factory Ledger (Audit/Event Log needs isolation)
ALTER TABLE factory_ledger ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

-- Script Registry (Scripts might be tenant-scoped in future)
ALTER TABLE script_registry ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

-- Indexes for performance and isolation enforcement
CREATE INDEX idx_rulebooks_tenant ON rulebooks(tenant_id);
CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);
CREATE INDEX idx_runs_tenant ON runs(tenant_id);
CREATE INDEX idx_artifacts_tenant ON artifacts(tenant_id);
CREATE INDEX idx_defects_tenant ON defects(tenant_id);
CREATE INDEX idx_ledger_tenant ON factory_ledger(tenant_id);
CREATE INDEX idx_scripts_tenant ON script_registry(tenant_id);
