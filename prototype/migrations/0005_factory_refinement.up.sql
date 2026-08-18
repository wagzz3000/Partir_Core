-- Add new columns to workers table
ALTER TABLE workers ADD COLUMN IF NOT EXISTS model_fingerprint TEXT NOT NULL DEFAULT '';
ALTER TABLE workers ADD COLUMN IF NOT EXISTS memory_binding_id TEXT;
ALTER TABLE workers ADD COLUMN IF NOT EXISTS memory_namespace TEXT;
ALTER TABLE workers ADD COLUMN IF NOT EXISTS assigned_slot TEXT;

-- Update status constraint to include new factory lifecycle states
ALTER TABLE workers DROP CONSTRAINT IF EXISTS workers_status_check;
ALTER TABLE workers ADD CONSTRAINT workers_status_check CHECK (status IN ('active', 'sleeping', 'crashed', 'starting', 'hot_standby', 'testing', 'quarantined', 'maintenance'));

-- Remove worker_id from stations table as stations now have multiple workers (slots)
ALTER TABLE stations DROP COLUMN IF EXISTS worker_id;
