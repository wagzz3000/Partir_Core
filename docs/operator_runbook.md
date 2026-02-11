# Partir Core — Operator Runbook

## Prerequisites
- Docker / Podman
- PostgreSQL 15+
- MinIO (or S3-compatible storage)
- NATS Server
- Go 1.25+ (for building from source)

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PARTIR_DB_URL` | Yes | `postgres://partir:partir@localhost:5432/partir?sslmode=disable` | PostgreSQL connection string |
| `NATS_URL` | No | `nats://localhost:4222` | NATS server URL |
| `PARTIR_MINIO_ENDPOINT` | Yes | `localhost:9000` | MinIO endpoint |
| `PARTIR_MINIO_ACCESS_KEY` | Yes | — | MinIO access key |
| `PARTIR_MINIO_SECRET_KEY` | Yes | — | MinIO secret key |
| `PARTIR_MINIO_BUCKET` | No | `partir-artifacts` | MinIO bucket name |
| `PARTIR_JWT_SECRET` | Yes | — | JWT signing secret |
| `PARTIR_METRICS_PORT` | No | `9090` | Prometheus metrics port |
| `PARTIR_SLACK_WEBHOOK_URL` | No | — | Slack alerting webhook |
| `PARTIR_PAGERDUTY_KEY` | No | — | PagerDuty routing key |

---

## Deploy

### Build from Source
```bash
go build -o foundry ./cmd/foundry
go build -o partir ./cmd/partir
```

### Docker
```bash
docker build -t partir-core:latest .
docker run -d --name partir \
  -e PARTIR_DB_URL="postgres://..." \
  -e PARTIR_MINIO_ENDPOINT="minio:9000" \
  -p 9090:9090 \
  partir-core:latest
```

---

## Database Migrations

### Run Migrations
```bash
# Using golang-migrate
migrate -path ./migrations -database "$PARTIR_DB_URL" up
```

### Rollback
```bash
# Roll back one migration
migrate -path ./migrations -database "$PARTIR_DB_URL" down 1

# Roll back all
migrate -path ./migrations -database "$PARTIR_DB_URL" down
```

### Migration History
| # | Name | Purpose |
|---|---|---|
| 001 | `initial` | Core tables (tickets, runs, artifacts, defects) |
| 002 | `add_green_tags` | Green tag quality markers |
| 003 | `seed_green_tags` | Seed initial green tags |
| 004 | `factory_tables` | Workers, stations, station metrics |
| 005 | `factory_refinement` | Factory refinements |
| 006 | `memory_pairs` | Memory pair registry |
| 007 | `factory_ledger` | Append-only event ledger |
| 008 | `maintenance_substates` | Maintenance state tracking |
| 009 | `ai_dsm` | AI diagnostic taxonomy |
| 010 | `script_registry` | Corrective script registry |
| 011 | `add_tenant_id` | Multi-tenancy columns |
| 012 | `audit_logs` | Compliance audit logging |
| 013 | `resilience_tables` | DLQ + idempotency keys |

---

## Backup

### PostgreSQL
```bash
# Full backup
pg_dump "$PARTIR_DB_URL" > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore
psql "$PARTIR_DB_URL" < backup_20260210.sql
```

### MinIO
```bash
# Using mc (MinIO Client)
mc alias set partir http://localhost:9000 $ACCESS_KEY $SECRET_KEY
mc mirror partir/partir-artifacts ./backup/minio/
```

### Restore
```bash
mc mirror ./backup/minio/ partir/partir-artifacts/
```

---

## Scale

### Horizontal Scaling
- **Foundry Workers**: Run multiple `foundry` instances. NATS handles work distribution.
- **Plugins**: Deploy as independent HTTP services. Register via `plugin.Registry`.
- **Database**: Use PostgreSQL read replicas for query-heavy workloads.

### Vertical Scaling
- Increase `PARTIR_MAX_CONCURRENT_TICKETS` for higher throughput.
- Increase `PARTIR_MAX_TICKETS_PER_HOUR` for burst capacity.

---

## Health Checks

```bash
# Liveness (process is running)
curl http://localhost:9090/health
# {"status":"ok","uptime":"2h30m"}

# Readiness (all dependencies connected)
curl http://localhost:9090/ready
# {"ready":true,"checks":{"postgres":"ok","minio":"ok"}}

# Prometheus metrics
curl http://localhost:9090/metrics
```

---

## Troubleshooting

### Common Issues

| Symptom | Cause | Fix |
|---|---|---|
| `circuit breaker "X" is open` | Too many failures to provider X | Check provider health; breaker auto-recovers after timeout |
| `rate limit exceeded` | Burst of requests | Wait for window to expire or increase limits |
| `tenant exceeded quota` | Tenant at resource limit | Increase quota via `QuotaManager.SetQuota()` |
| `signature verification failed` | Plugin binary tampered | Re-download plugin; verify with `manifest.json` |
| `secret not found` | Missing env var | Set required env vars (see table above) |

### Dead Letter Queue
```sql
-- View unresolved DLQ entries
SELECT * FROM tickets_dlq WHERE resolved_at IS NULL ORDER BY created_at DESC;

-- Resolve an entry
UPDATE tickets_dlq SET resolved_at = NOW() WHERE id = '<uuid>';
```

### Audit Trail
```sql
-- Recent actions by a user
SELECT * FROM audit_logs WHERE actor_id = 'user-1' ORDER BY created_at DESC LIMIT 20;
```

---

## Rollback Playbook

1. **Stop** all Foundry instances
2. **Rollback** migrations: `migrate -path ./migrations -database "$DB_URL" down N`
3. **Restore** database from backup: `psql "$DB_URL" < backup.sql`
4. **Restore** MinIO artifacts: `mc mirror ./backup/minio/ partir/partir-artifacts/`
5. **Deploy** previous application version
6. **Verify** health: `curl http://localhost:9090/ready`
