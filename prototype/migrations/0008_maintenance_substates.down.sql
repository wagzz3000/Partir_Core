ALTER TABLE memory_pairs DROP CONSTRAINT IF EXISTS valid_maint_phase;
ALTER TABLE memory_pairs DROP COLUMN IF EXISTS maintenance_phase;
