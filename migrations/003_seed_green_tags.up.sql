-- Seed green_tags with validated Postgres version
INSERT INTO green_tags (postgres_major, postgres_minor, active, verified_at)
VALUES (16, 4, true, NOW())
ON CONFLICT DO NOTHING;
