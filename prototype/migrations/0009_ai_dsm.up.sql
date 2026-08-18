-- AI DSM Taxonomy (Patent §9.6)
-- Diagnostic codes with severity levels and recommended intervention pathways.
-- Each code maps to a pathological pattern type that the diagnostic engine detects.
-- Severity drives triage routing: low → script, medium → WOT/CBT, high → surgery.

CREATE TABLE ai_dsm (
    diagnosis_code      TEXT PRIMARY KEY,
    category            TEXT NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT,
    severity_default    INT NOT NULL,
    recommended_pathway TEXT NOT NULL,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT valid_category CHECK (
        category IN ('obsessive','fragmentation','hallucination','refusal','drift')
    ),
    CONSTRAINT valid_pathway CHECK (
        recommended_pathway IN ('script','wot','cbt','surgery')
    ),
    CONSTRAINT valid_severity CHECK (
        severity_default BETWEEN 1 AND 10
    )
);

-- Seed common diagnostic codes
INSERT INTO ai_dsm (diagnosis_code, category, name, description, severity_default, recommended_pathway) VALUES
    ('LOOP_TYPE_1',      'obsessive',      'Obsessive Loop',                'Repetitive generation without convergence; output variance below threshold',                                 6, 'script'),
    ('LOOP_TYPE_2',      'obsessive',      'Runaway Recursion',             'Inability to terminate iterative refinement cycles; exceeds max recursion depth',                           8, 'wot'),
    ('HALLUC_CLUSTER_1', 'hallucination',  'Persistent Hallucination',      'Stable false beliefs encoded into graph with high centrality; retrieved across independent prompts',        7, 'surgery'),
    ('FRAG_TYPE_1',      'fragmentation',  'Context Fragmentation',         'Contradictions across sessions from desynchronization or drift between temporal and relational memory',     5, 'cbt'),
    ('REFUSAL_ATTR_1',   'refusal',        'Refusal Attractor',             'Disproportionate activation of refusal-related nodes across benign prompt classes',                         6, 'wot'),
    ('DRIFT_TYPE_1',     'drift',          'Behavioral Instability',        'High variance responses under similar inputs; reduced predictability',                                      4, 'script'),
    ('BRITTLE_TYPE_1',   'drift',          'Brittle Reasoning',             'Reduced adaptability due to attractor subgraphs or over-weighted associations trapping traversal',          7, 'surgery');
