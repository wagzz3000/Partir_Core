-- Partir Core v0.1 Initial Schema
-- Migration: 001_initial

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Rulebooks table (Alpha and Beta)
CREATE TABLE rulebooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rulebook_type VARCHAR(10) NOT NULL CHECK (rulebook_type IN ('alpha', 'beta')),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    manifest JSONB NOT NULL,
    compat JSONB,
    parent_version VARCHAR(50),
    storage_uri TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (name, version, rulebook_type)
);

CREATE INDEX idx_rulebooks_type ON rulebooks(rulebook_type);
CREATE INDEX idx_rulebooks_name ON rulebooks(name);
CREATE INDEX idx_rulebooks_hash ON rulebooks(hash);

-- Tickets table (Spec-Based Work Orders)
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id VARCHAR(255) NOT NULL UNIQUE,
    title VARCHAR(500) NOT NULL,
    plugin_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(255) NOT NULL,
    alpha_ref VARCHAR(255) NOT NULL,
    beta_ref VARCHAR(255),
    green_tag_ref VARCHAR(255),
    inputs JSONB NOT NULL DEFAULT '{}',
    constraints JSONB NOT NULL DEFAULT '{}',
    acceptance_gates JSONB NOT NULL DEFAULT '[]',
    limits JSONB NOT NULL DEFAULT '{}',
    routing_hints JSONB NOT NULL DEFAULT '{}',
    outputs_expected JSONB NOT NULL DEFAULT '[]',
    state VARCHAR(50) NOT NULL DEFAULT 'SPEC_READY',
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickets_state ON tickets(state);
CREATE INDEX idx_tickets_plugin ON tickets(plugin_id);
CREATE INDEX idx_tickets_created ON tickets(created_at);

-- Runs table (Execution attempts)
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id VARCHAR(255) NOT NULL UNIQUE,
    ticket_id UUID NOT NULL REFERENCES tickets(id),
    attempt INTEGER NOT NULL DEFAULT 1,
    executor VARCHAR(255) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    cost_tokens INTEGER DEFAULT 0,
    cost_usd DECIMAL(10, 6) DEFAULT 0,
    logs_uri TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_runs_ticket ON runs(ticket_id);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_started ON runs(started_at);

-- Artifacts table
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    artifact_id VARCHAR(255) NOT NULL UNIQUE,
    ticket_id UUID NOT NULL REFERENCES tickets(id),
    run_id UUID REFERENCES runs(id),
    artifact_type VARCHAR(255) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    schema_ref VARCHAR(255),
    provenance JSONB NOT NULL DEFAULT '{}',
    storage_uri TEXT NOT NULL,
    size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_artifacts_ticket ON artifacts(ticket_id);
CREATE INDEX idx_artifacts_type ON artifacts(artifact_type);
CREATE INDEX idx_artifacts_hash ON artifacts(hash);

-- Defects table (for DMAIC analysis)
CREATE TABLE defects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    defect_id VARCHAR(255) NOT NULL UNIQUE,
    run_id UUID NOT NULL REFERENCES runs(id),
    gate_id VARCHAR(255) NOT NULL,
    defect_class VARCHAR(255) NOT NULL,
    plugin_id VARCHAR(255),
    message TEXT NOT NULL,
    offending_fields JSONB NOT NULL DEFAULT '[]',
    suggested_fix TEXT,
    rulebook_version VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- DMAIC indexes for trending and analysis
CREATE INDEX idx_defects_class ON defects(defect_class);
CREATE INDEX idx_defects_gate ON defects(gate_id);
CREATE INDEX idx_defects_plugin ON defects(plugin_id);
CREATE INDEX idx_defects_created ON defects(created_at);
CREATE INDEX idx_defects_run ON defects(run_id);
