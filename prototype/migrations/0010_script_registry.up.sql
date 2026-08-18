-- Script Registry (Patent §12.6)
-- Stores validated corrective scripts with compatibility constraints and side-effect profiles.
-- Each script is tied to a diagnosis code from the AI DSM taxonomy.
-- Scripts go through lifecycle: draft → validated → active → expired.

CREATE TABLE script_registry (
    script_id           TEXT PRIMARY KEY,
    diagnosis_code      TEXT NOT NULL REFERENCES ai_dsm(diagnosis_code),
    name                TEXT NOT NULL,
    version             INT NOT NULL DEFAULT 1,
    script_type         TEXT NOT NULL,
    target_components   JSONB,
    dosage              NUMERIC,
    duration_cycles     INT,
    side_effects        JSONB,
    compatibility       JSONB,
    status              TEXT NOT NULL DEFAULT 'draft',
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_script_type CHECK (
        script_type IN ('stabilization','oscillation','cognitive')
    ),
    CONSTRAINT valid_script_status CHECK (
        status IN ('draft','validated','active','expired')
    )
);

-- Fast lookup by diagnosis code
CREATE INDEX idx_script_diagnosis ON script_registry (diagnosis_code);

-- Fast lookup of active scripts
CREATE INDEX idx_script_active ON script_registry (status) WHERE status = 'active';
