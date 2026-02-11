-- Factory floor tables for worker and station management
-- These enable tracking of throughput, defects, and worker health

-- Workers table - tracks all worker instances
CREATE TABLE IF NOT EXISTS workers (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('alpha', 'beta', 'omega')),
    endpoint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'sleeping' CHECK (status IN ('active', 'sleeping', 'crashed', 'starting')),
    assigned_station TEXT,
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Stations table - tracks factory stations
CREATE TABLE IF NOT EXISTS stations (
    id TEXT PRIMARY KEY CHECK (id IN ('alpha', 'beta', 'omega')),
    worker_id TEXT REFERENCES workers(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'idle' CHECK (status IN ('active', 'idle', 'maintenance')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Station metrics - time-series data for factory analytics
CREATE TABLE IF NOT EXISTS station_metrics (
    id SERIAL PRIMARY KEY,
    station_id TEXT NOT NULL REFERENCES stations(id),
    throughput INT DEFAULT 0,          -- Tickets completed in window
    defects INT DEFAULT 0,             -- Gate failures in window
    avg_latency_ms INT DEFAULT 0,      -- Average execution latency
    tokens_used BIGINT DEFAULT 0,      -- LLM tokens consumed
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_workers_type ON workers(type);
CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
CREATE INDEX IF NOT EXISTS idx_workers_assigned_station ON workers(assigned_station);
CREATE INDEX IF NOT EXISTS idx_station_metrics_station_id ON station_metrics(station_id);
CREATE INDEX IF NOT EXISTS idx_station_metrics_recorded_at ON station_metrics(recorded_at);

-- Seed initial stations
INSERT INTO stations (id, status, created_at, updated_at)
VALUES 
    ('alpha', 'idle', NOW(), NOW()),
    ('beta', 'idle', NOW(), NOW()),
    ('omega', 'idle', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
