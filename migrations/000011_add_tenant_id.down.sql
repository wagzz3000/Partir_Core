-- Migration: 000011_add_tenant_id (Down)

-- Drop indexes
DROP INDEX IF EXISTS idx_scripts_tenant;
DROP INDEX IF EXISTS idx_ledger_tenant;
DROP INDEX IF EXISTS idx_defects_tenant;
DROP INDEX IF EXISTS idx_artifacts_tenant;
DROP INDEX IF EXISTS idx_runs_tenant;
DROP INDEX IF EXISTS idx_tickets_tenant;
DROP INDEX IF EXISTS idx_rulebooks_tenant;

-- Drop columns
ALTER TABLE script_registry DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE factory_ledger DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE defects DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE artifacts DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE runs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE tickets DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE rulebooks DROP COLUMN IF EXISTS tenant_id;
