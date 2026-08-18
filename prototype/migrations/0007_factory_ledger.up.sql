-- Factory Ledger
-- Append-only event log for cross-pair awareness.
-- All roles (including Foundry) write events.
-- Inactive pairs tail the ledger to stay aware of production.
-- Full audit history — no auto-trim.

CREATE TABLE factory_ledger (
    seq_id         BIGSERIAL PRIMARY KEY,
    api_role       TEXT NOT NULL,
    source_pair_id TEXT REFERENCES memory_pairs(pair_id),  -- NULL for Foundry
    event_type     TEXT NOT NULL,
    event_payload  JSONB NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);

-- Fast tail queries: WHERE api_role = ? AND seq_id > ?
CREATE INDEX idx_ledger_role_seq ON factory_ledger (api_role, seq_id);

-- Fast cross-role queries
CREATE INDEX idx_ledger_seq ON factory_ledger (seq_id);
