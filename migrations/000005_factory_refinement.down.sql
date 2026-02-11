-- Restore worker_id to stations table
ALTER TABLE stations ADD COLUMN IF NOT EXISTS worker_id TEXT REFERENCES workers(id);

-- Revert status constraint
ALTER TABLE workers DROP CONSTRAINT IF EXISTS workers_status_check;
ALTER TABLE workers ADD CONSTRAINT workers_status_check CHECK (status IN ('active', 'sleeping', 'crashed', 'starting'));

-- Drop new columns
ALTER TABLE workers DROP COLUMN IF EXISTS assigned_slot;
ALTER TABLE workers DROP COLUMN IF EXISTS memory_namespace;
ALTER TABLE workers DROP COLUMN IF EXISTS memory_binding_id;
ALTER TABLE workers DROP COLUMN IF EXISTS model_fingerprint;
