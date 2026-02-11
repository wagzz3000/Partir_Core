-- Memory Pairs Registry
-- Each pair bonds a StarRocks (Chrono) schema to a FalkorDB (Graph) permanently.
-- Invariant 1: 1 Chrono ↔ 1 Graph, always.
-- Invariant 2: Alpha owns A1-A4, Beta owns B1-B4, Omega owns O1-O4, Foundry owns none.
-- Invariant 3: One active pair per role, one writer per pair.

CREATE TABLE memory_pairs (
    pair_id           TEXT PRIMARY KEY,
    api_role          TEXT NOT NULL,
    chrono_schema     TEXT NOT NULL,
    graph_name        TEXT NOT NULL,
    state             TEXT NOT NULL DEFAULT 'sleep',
    writer_worker_id  TEXT,
    model_fingerprint TEXT,
    last_promotion_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_state CHECK (state IN ('active','hot','sleep','maintenance')),
    CONSTRAINT valid_role  CHECK (api_role IN ('alpha','beta','omega'))
);

-- Enforces: exactly one active pair per role
CREATE UNIQUE INDEX idx_one_active_per_role
    ON memory_pairs (api_role) WHERE state = 'active';

-- Enforces: no two pairs share a writer
CREATE UNIQUE INDEX idx_single_writer
    ON memory_pairs (writer_worker_id) WHERE writer_worker_id IS NOT NULL;

-- Seed 12 pairs (4 per role)
INSERT INTO memory_pairs (pair_id, api_role, chrono_schema, graph_name, state) VALUES
    ('A1', 'alpha', 'chrono_a1', 'graph_a1', 'active'),
    ('A2', 'alpha', 'chrono_a2', 'graph_a2', 'sleep'),
    ('A3', 'alpha', 'chrono_a3', 'graph_a3', 'maintenance'),
    ('A4', 'alpha', 'chrono_a4', 'graph_a4', 'hot'),
    ('B1', 'beta',  'chrono_b1', 'graph_b1', 'active'),
    ('B2', 'beta',  'chrono_b2', 'graph_b2', 'sleep'),
    ('B3', 'beta',  'chrono_b3', 'graph_b3', 'maintenance'),
    ('B4', 'beta',  'chrono_b4', 'graph_b4', 'hot'),
    ('O1', 'omega', 'chrono_o1', 'graph_o1', 'active'),
    ('O2', 'omega', 'chrono_o2', 'graph_o2', 'sleep'),
    ('O3', 'omega', 'chrono_o3', 'graph_o3', 'maintenance'),
    ('O4', 'omega', 'chrono_o4', 'graph_o4', 'hot');
