-- Migration: 000013_resilience_tables (Down)

DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS tickets_dlq;
