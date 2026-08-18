-- Maintenance Sub-States
-- Tracks internal phase of a pair while in maintenance state.
-- Enables dual tracking: DB column (current phase) + Ledger events (aggregation/audit).
-- maintenance_phase is only set when state = 'maintenance'; NULL otherwise.

ALTER TABLE memory_pairs
    ADD COLUMN maintenance_phase TEXT DEFAULT NULL;

-- CHECK constraint: only valid maintenance phases
ALTER TABLE memory_pairs
    ADD CONSTRAINT valid_maint_phase CHECK (
        maintenance_phase IS NULL OR
        maintenance_phase IN (
            'diagnose',
            'triage',
            'script',
            'wot',
            'cbt',
            'surgery',
            'rehab',
            'test'
        )
    );
