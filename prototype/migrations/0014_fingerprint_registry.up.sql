CREATE TABLE IF NOT EXISTS model_fingerprints (
    model_name TEXT NOT NULL,
    allowed_sha TEXT NOT NULL,
    added_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (model_name, allowed_sha)
);

CREATE INDEX idx_model_fingerprints_name ON model_fingerprints(model_name);
