-- Seed green_tags with validated Postgres version
INSERT INTO green_tags (name, postgres_version, postgres_major, postgres_minor, active, created_at)
VALUES ('v16.4', '16.4', 16, 4, true, NOW())
ON CONFLICT DO NOTHING;
