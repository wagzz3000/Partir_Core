# Partir Core — User Guide

> **The complete reference for installing, configuring, and running Partir Core.**
> Hand this to any developer, AI agent (Cursor, Warp, factory.ai), or CI system and they can get up and running.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Quick Start (5 minutes)](#2-quick-start)
3. [Configuration](#3-configuration)
4. [Database Migrations](#4-database-migrations)
5. [Architecture Overview](#5-architecture-overview)
6. [The Round Table (Planning)](#6-the-round-table)
7. [Running the Foundry](#7-running-the-foundry)
8. [Plugins](#8-plugins)
9. [Session Persistence](#9-session-persistence)
10. [Security & Authentication](#10-security--authentication)
11. [Resilience & Reliability](#11-resilience--reliability)
12. [ChatOps Integration](#12-chatops-integration)
13. [Remote Provisioning](#13-remote-provisioning)
14. [Monitoring & Observability](#14-monitoring--observability)
15. [Maintenance](#15-maintenance)
16. [Diagnostics (`partir doctor`)](#16-diagnostics)
17. [CLI Reference](#17-cli-reference)
18. [Troubleshooting](#18-troubleshooting)
19. [Project Structure](#19-project-structure)

---

## 1. Prerequisites

### 1.1 Host Tools
These must be installed on your local machine to build and manage the project.

| Requirement | Min Version | Purpose |
|---|---|---|
| **Go** | 1.21+ | Build Partir Core binaries |
| **Docker + Docker Compose** | 24.x+ | **Recommended** - Runs all infra (DB, S3, NATS, etc.) |
| **Git** | 2.x+ | Repository and rulebook versioning |

### 1.2 Infrastructure Services
Partir Core requires the following services. You can either use the included `docker-compose.yml` (easiest) or point the application to existing instances.

| Service | Min Version | Description |
|---|---|---|
| **PostgreSQL** | 15+ | Core registry, ledger, audit logs |
| **MinIO** | Latest | Artifact storage (S3-compatible) |
| **NATS JetStream** | 2.10+ | Inter-worker messaging |
| **FalkorDB** | 4.x+ | Graph memory for memory pairs |
| **Ollama** | Latest | Local LLM inference engine |

| **NVIDIA GPU** *(optional)* | — | Local LLM inference via Ollama |

> **For AI agents**: Using **Docker** is the recommended way to set up the infrastructure. If Docker is not available, you will need to manually provision the services listed in Section 1.2 and configure the `.env` file accordingly.

---

## 2. Quick Start

```bash
# 1. Clone
git clone https://github.com/partir/core.git
cd core

# 2. Configure
cp .env.example .env
# Edit .env — see Section 3 for all variables

# 3. Start infrastructure
docker compose up -d

# 4. Build all binaries
go build -o bin/ ./cmd/alpha ./cmd/beta ./cmd/foundry ./cmd/partir ./cmd/migrate

# 5. Run migrations
./bin/migrate up

# 6. Pull an LLM model
docker exec -it partir-core-ollama-1 ollama pull llama3:8b

# 7. Register the executor plugin
./bin/partir executors add ollama http://localhost:8081

# 8. Verify everything works
./bin/partir doctor
./bin/partir health
```

After step 8, you should see all checks green. You're ready to use Partir Core.

### Alternative: `go install`

```bash
go install ./cmd/alpha ./cmd/beta ./cmd/foundry ./cmd/partir ./cmd/migrate
```

This puts `alpha`, `beta`, `foundry`, `partir`, `migrate` on your `$PATH`. All examples below use the short names.

---

## 3. Configuration

Partir Core uses [Viper](https://github.com/spf13/viper) for configuration. Sources are merged in priority order:

```
1. Environment variables     (highest priority)
2. Config file (partir.yaml) (medium)
3. Built-in defaults         (lowest)
```

### 3.1 Environment Variables

Create a `.env` file from the example:

```bash
cp .env.example .env
```

**Required variables:**

| Variable | Default | Description |
|---|---|---|
| `PARTIR_DB_URL` | `postgres://partir:partir@localhost:5432/partir?sslmode=disable` | PostgreSQL connection string |
| `PARTIR_MINIO_ENDPOINT` | `localhost:9000` | MinIO/S3 endpoint |
| `PARTIR_MINIO_ACCESS_KEY` | `partir` | MinIO access key |
| `PARTIR_MINIO_SECRET_KEY` | *(none)* | MinIO secret key |
| `PARTIR_JWT_SECRET` | *(none)* | Secret for signing JWT tokens |

**Optional variables:**

| Variable | Default | Description |
|---|---|---|
| `NATS_URL` | `nats://localhost:4222` | NATS JetStream URL |
| `PARTIR_MINIO_BUCKET` | `partir-artifacts` | S3 bucket name |
| `PARTIR_METRICS_PORT` | `9090` | Prometheus metrics port |
| `PARTIR_SLACK_WEBHOOK_URL` | *(none)* | Slack alerting webhook |
| `PARTIR_PAGERDUTY_KEY` | *(none)* | PagerDuty routing key |
| `PARTIR_MAX_CONCURRENT_TICKETS` | `10` | Max concurrent tickets per tenant |
| `PARTIR_MAX_TICKETS_PER_HOUR` | `100` | Max tickets per tenant per hour |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | OpenTelemetry collector endpoint |
| `GRAFANA_ADMIN_PASSWORD` | `partir` | Grafana admin password |

### 3.2 Config File (Optional)

Instead of (or in addition to) env vars, create a `partir.yaml`:

```yaml
# partir.yaml — place in ./, /etc/partir/, or ~/.partir/
database_url: postgres://partir:secure@db-host:5432/partir?sslmode=require
nats_url: nats://nats-host:4222
minio_endpoint: minio-host:9000
minio_bucket: partir-artifacts
metrics_port: 9090
jwt_secret: your-256-bit-secret
max_concurrent_tickets: 20
max_tickets_per_hour: 500
```

Viper searches for `partir.yaml`, `partir.json`, or `partir.toml` in:
1. Current directory (`./`)
2. `/etc/partir/`
3. `~/.partir/`

### 3.3 Infrastructure Services

`docker compose up -d` starts these services:

| Service | Port(s) | Purpose |
|---|---|---|
| **PostgreSQL** | 5432 | Core data store |
| **MinIO** | 9000 / 9001 | Artifact storage (S3-compatible) |
| **NATS JetStream** | 4222 / 8222 | Inter-worker messaging |
| **FalkorDB** | 6379 | Graph memory (per pair) |
| **Ollama** | 11434 | Local LLM inference |
| **Prometheus** | 9090 | Metrics collection |
| **Grafana** | 3000 | Dashboards |
| **Loki** | 3100 | Log aggregation |

> **GPU support**: Use `docker compose -f docker-compose.gpu.yml up -d` for NVIDIA GPU passthrough to Ollama.

---

## 4. Database Migrations

```bash
# Apply all migrations
migrate up

# Check current state
migrate version

# Roll back one migration
migrate down 1
```

**Migration inventory (13 migrations):**

| Migration | Purpose |
|---|---|
| 001–003 | Core schema: tickets, artifacts, runs, green tags |
| 004–005 | Factory: workers, stations |
| 006 | Memory pairs (A1–A4, B1–B4, O1–O4) |
| 007 | Factory ledger (append-only audit) |
| 008 | Maintenance sub-states |
| 009 | AI DSM diagnostic taxonomy |
| 010 | Script registry |
| 011 | Multi-tenancy (`tenant_id` columns) |
| 012 | Audit logs (immutable — triggers block UPDATE/DELETE) |
| 013 | Resilience tables (DLQ, idempotency keys with TTL cleanup) |

---

## 5. Architecture Overview

Partir Core is a **manufacturing-style AI factory** with four modules:

```
┌───────────────────────────────────────────────────────┐
│                   Control Surface                      │
│       CLI / API / ChatOps (Slack, Discord)             │
└───────────┬───────────────────────────┬───────────────┘
            │                           │
    ┌───────▼───────┐          ┌───────▼───────┐
    │     Alpha     │          │     Beta      │
    │  "The Rules"  │          │  "The Design" │
    │  Schemas      │          │  Render Vocab  │
    │  Catalogs     │          │  Palettes      │
    │  Constraints  │          │  Effects       │
    └───────┬───────┘          └───────┬───────┘
            │      alpha_ref           │  beta_ref
            └──────────┬───────────────┘
                       │
               ┌───────▼───────┐
               │    Foundry    │
               │  "Production" │
               │  Tickets →    │
               │  Plugins →    │
               │  Artifacts    │
               └───────┬───────┘
                       │
               ┌───────▼───────┐
               │     Omega     │
               │  "Quality"    │
               │  Gates →      │
               │  Pass / Fail  │
               └───────────────┘
```

- **Alpha** = Engineering spec — *what* can be built and *what the constraints are*
- **Beta** = Design language — *how things look and feel*
- **Foundry** = Production line — tickets → LLM plugins → artifacts
- **Omega** = Quality control — every artifact passes gates before shipping

### Memory Pairs

Each project gets a **12-pair assembly**:

```
Alpha:  A1 (active), A2, A3, A4  (hot standby)
Beta:   B1 (active), B2, B3, B4  (hot standby)
Omega:  O1 (active), O2, O3, O4  (hot standby)
```

Each pair bonds a **chrono-schema** to a **FalkorDB graph**. Data isolation is strict — pairs in project A cannot access graphs from project B.

---

## 6. The Round Table

### 6.1 Start a Planning Session

```bash
foundry plan my_project
foundry plan my_project --upload ./gdd.pdf --upload ./brand_guidelines.pdf
foundry plan my_project --alpha my_rulebook --beta my_design
```

The Round Table is an interactive conversation between you, **Alpha** (Architect), and **Beta** (Designer). Describe what you need in plain English — they'll ask clarifying questions and converge on a plan.

### 6.2 Auto-Scoping & Approval

When Alpha and Beta have enough context, the Planner generates a project scope:

```
━━━ CONVERGENCE REACHED ━━━
╔════════════════════════════════════════════════════════╗
║          PROJECT SCOPE                                ║
╚════════════════════════════════════════════════════════╝
Summary: Mobile game backend with Clan/Quest systems.
Tickets: 42  |  Groups: 4  |  Est. Cost: $0.12

[A]pprove, [E]dit, [O]bject, [Q]uit:
```

- **Approve** → Submits all tickets to the Foundry
- **Object** → Explain your concern; the scope is regenerated
- **Edit** → Modify individual tickets

### 6.3 Saved Scopes

```bash
foundry scope load scope.json     # View saved scope
foundry scope approve scope.json  # Submit to Foundry
```

---

## 7. Running the Foundry

### 7.1 Manual Ticket Submission

```json
{
  "title": "Generate User Schema",
  "plugin_id": "ollama",
  "tenant_id": "default",
  "priority": 5,
  "inputs": { "prompt": "Generate a PostgreSQL schema for users..." }
}
```

```bash
foundry submit ticket.json       # Submit ticket
foundry run <ticket_id>          # Execute it
foundry status <ticket_id>       # Check progress
foundry artifacts <ticket_id>    # List output artifacts
```

### 7.2 Execution Pipeline

```
1. Load ticket          → Validate it exists and is pending
2. Resolve Alpha ref    → Load the exact versioned rulebook
3. Resolve Beta ref     → Load the design vocabulary
4. Acquire worker       → Claim an available worker
5. Pre-execution gates  → Omega checks inputs
6. Execute via plugin   → Send to LLM executor
7. Post-execution gates → Omega validates output
   ├─ Pass → Store artifacts → Complete ✓
   └─ Fail → Andon Cord → Triage → Fix or Alert
```

### 7.3 The Andon Cord

When Omega gates fail:
1. Halt signal published to `collab.andon`
2. Collaboration API triages:
   - **Beta/Foundry error** → Auto-retry via `collab.fix`
   - **Alpha spec error** → Escalated to human via `collab.alert`
3. All events recorded in the Factory Ledger

---

## 8. Plugins

### 8.1 Registering Executor Plugins

```bash
partir executors add ollama http://localhost:8081
partir executors add helicone http://localhost:8082
partir executors list
partir executors health
partir executors remove ollama
```

### 8.2 Plugin Manager (Registry)

Install, update, and discover plugins from a remote registry:

```bash
# Search available plugins
partir plugin list --remote

# Install a plugin (downloads binary, verifies SHA-256 hash)
partir plugin install code-gen
partir plugin install https://registry.partir.dev/plugins/code-gen/v1.2.0

# Update to latest version
partir plugin update code-gen

# List installed plugins
partir plugin list
```

Installed plugins are stored in `~/.partir/plugins/` with manifests in `~/.partir/plugins/.manifests/`.

### 8.3 Building a Custom Plugin

Plugins implement the `plugin.Executor` interface:

```go
package plugin

// Executor is the interface every LLM plugin must implement
type Executor interface {
    Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)
    Name() string
    HealthCheck(ctx context.Context) error
}

type ExecutionRequest struct {
    Prompt    string            `json:"prompt"`
    Model     string            `json:"model"`
    Params    map[string]any    `json:"params"`
    MaxTokens int               `json:"max_tokens"`
}

type ExecutionResult struct {
    Output    string   `json:"output"`
    Artifacts []string `json:"artifacts"`
    TokensIn  int      `json:"tokens_in"`
    TokensOut int      `json:"tokens_out"`
}
```

Build your plugin, place the binary in `~/.partir/plugins/`, and register it:

```bash
partir executors add my-plugin http://localhost:8083
```

---

## 9. Session Persistence

Round Table sessions are automatically saved so you can resume interrupted work.

```bash
# Start a session (auto-saved to .partir/session.json)
foundry plan my_project

# If interrupted, resume later:
foundry resume
```

Session states:
- `active` — Currently running
- `paused` — Interrupted or errored (resumable)
- `completed` — Finished normally

Session file location: `.partir/session.json` in your working directory.

---

## 10. Security & Authentication

### 10.1 JWT Authentication

Set `PARTIR_JWT_SECRET` to a strong 256-bit secret. The auth system issues and validates HMAC-SHA256 JWTs.

```go
// Included middleware for HTTP APIs:
// internal/security/middleware/auth.go   — JWT validation
// internal/security/middleware/rbac.go   — Role-based access control
```

### 10.2 Roles & RBAC

| Role | Permissions |
|---|---|
| `admin` | Full access |
| `operator` | Factory management, ticket submission |
| `viewer` | Read-only access to status, artifacts |

### 10.3 Multi-Tenancy

All resources are scoped by `tenant_id`. Migration 011 added tenant columns to all tables. Quotas are enforced per-tenant:

```bash
# Resource limits are configured via env vars or partir.yaml:
PARTIR_MAX_CONCURRENT_TICKETS=10
PARTIR_MAX_TICKETS_PER_HOUR=100
```

### 10.4 Secrets Management

```go
// internal/security/secrets/ provides:
// - EnvProvider  — reads from environment (default)
// - VaultProvider — HashiCorp Vault integration (production)
```

### 10.5 Audit Logging

All actions are recorded in an **immutable** `audit_logs` table:
- Database triggers prevent UPDATE/DELETE
- REVOKE applied on the `partir` DB role
- Export with filters via `internal/security/audit/export.go` (CSV/JSON)

---

## 11. Resilience & Reliability

Partir Core includes production-grade resilience patterns:

| Pattern | Package | Purpose |
|---|---|---|
| **Circuit Breaker** | `internal/resilience` | Stops calling failing services after threshold |
| **Rate Limiter** | `internal/resilience` | Sliding window with binary-search pruning |
| **Dead Letter Queue** | `internal/resilience` | Failed tickets stored in `tickets_dlq` for later analysis |
| **Exponential Backoff** | `internal/resilience` | Retry with jitter and context-aware cancellation |
| **Idempotency** | `internal/resilience` | Prevents duplicate processing via `idempotency_keys` table |
| **Fallback Chain** | `internal/resilience` | Per-provider circuit breakers for graceful degradation |

### DLQ Inspection

Failed tickets that exhaust retries are moved to the dead letter queue:

```sql
SELECT ticket_id, reason, last_error, attempts
FROM tickets_dlq
WHERE tenant_id = 'default' AND resolved_at IS NULL;
```

### Idempotency Key Cleanup

Keys expire automatically via `cleanup_idempotency_keys()`:

```sql
-- Manual cleanup (remove keys older than 24 hours)
SELECT cleanup_idempotency_keys(24);

-- Or schedule via pg_cron:
SELECT cron.schedule('cleanup-idem', '0 * * * *', $$SELECT cleanup_idempotency_keys(24)$$);
```

---

## 12. ChatOps Integration

### 12.1 Slack

```go
bot := chatops.NewSlackBot(handler, signingSecret)
http.HandleFunc("/slack/commands", bot.SlashCommandHandler())
```

Slack commands:
- `/partir status` — Live factory status
- `/partir approve <ticket-id>` — Approve a ticket
- `/partir help` — Command reference

### 12.2 Discord

```go
bot, _ := chatops.NewDiscordBot(handler, token, publicKeyHex)
http.HandleFunc("/discord/interactions", bot.InteractionHandler())
```

Discord interactions use Ed25519 signature verification (required by Discord API). The bot handles PING/PONG automatically.

### 12.3 Live Status

Connect a `StatusProvider` to get real-time data in status responses:

```go
type StatusProvider interface {
    GetQueueDepth(ctx context.Context) (int, error)
    GetActiveTickets(ctx context.Context) (int, error)
    GetLastCompletedAt(ctx context.Context) (*time.Time, error)
}

handler := chatops.NewDefaultHandler(myStatusProvider)
```

---

## 13. Remote Provisioning

Deploy workers and plugins to remote nodes via SSH.

### 13.1 Worker Installation

```bash
partir remote-install worker deploy@192.168.1.100
partir remote-install worker root@gpu-node.example.com:2222
```

This SSHs into the target and:
1. Installs Ollama
2. Pulls the configured model (e.g., `llama3:8b`)
3. Creates a `partir` user
4. Creates plugin directory at `/opt/partir/plugins/`

### 13.2 Plugin Deployment

```bash
partir remote-install plugin code-gen deploy@192.168.1.100
```

### 13.3 Systemd Services

Auto-generated systemd units with security hardening:

```ini
# Generated for partir-worker.service
[Service]
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/partir
PrivateTmp=yes
```

Pre-built unit templates:
- `partir-worker.service` — Foundry worker daemon
- `ollama.service` — Ollama LLM server

---

## 14. Monitoring & Observability

### 14.1 Health Endpoints

```bash
partir health              # Full JSON health report
partir healthz             # Liveness probe (ok/error)
partir doctor              # Full environment diagnostics
```

HTTP health server:
```bash
partir health serve-health --port 8080
# GET /health  — full status (JSON)
# GET /healthz — liveness probe
# GET /ready   — readiness probe
```

### 14.2 Grafana Dashboards

```
URL:      http://localhost:3000
Username: admin
Password: (your GRAFANA_ADMIN_PASSWORD)
```

**Factory Dashboard:**

| Panel | Metric |
|---|---|
| Workers Online | Active/idle/crashed count |
| Station Utilization | % capacity per station |
| Ticket Throughput | Completed per minute |
| Run Duration | P50/P95/P99 |
| Defect Rate | Gate failures per run |
| Token Usage | Consumed per plugin |
| Cost Tracking | USD per executor |

### 14.3 Prometheus Metrics

```
partir_ticket_total{status}
partir_ticket_completed_total
partir_run_duration_seconds{plugin,job_type}
partir_run_failed_total{reason}
partir_ai_tokens_total{model,plugin}
partir_ai_cost_usd_total{model}
partir_worker_heartbeat_age_seconds{worker_id}
partir_station_utilization{station}
partir_gate_duration_seconds{gate}
partir_defect_total{class,gate}
```

### 14.4 Alerting

Configure alert destinations:

```env
PARTIR_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx
PARTIR_PAGERDUTY_KEY=your-routing-key
```

Alerts fire on: worker crashes, high defect rates, queue backpressure, and quota exhaustion.

### 14.5 Log Aggregation

```
Grafana → Explore → Loki
```

```logql
{container="partir-core"} |= "andon"
{container="partir-core"} |= "defect"
```

---

## 15. Maintenance

### 15.1 Pair Inspection

```bash
partir maintenance inspect A3
```

### 15.2 Maintenance Workflow

```bash
partir maintenance phase A3 diagnose
partir maintenance phase A3 triage
partir maintenance phase A3 script
partir maintenance phase A3 wot
partir maintenance phase A3 test
partir maintenance release A3     # Release back to service
```

State machine: `Diagnose → Triage → Script/WOT/CBT/Surgery → Rehab → Test → Release`

---

## 16. Diagnostics

```bash
partir doctor
```

Output:
```
🩺 Partir Doctor — Environment Diagnostics
   Go: go1.23.0  |  OS: linux/amd64

  ✅  PARTIR_DB_URL           post****
  ✅  PARTIR_JWT_SECRET       sk-c****
  ✅  NATS_URL                nats****
  ✅  PARTIR_MINIO_ENDPOINT   loca****
  ✅  PostgreSQL              reachable at localhost:5432
  ✅  NATS                    reachable at localhost:4222
  ✅  docker                  Docker version 27.2.0
  ⚠️   mc                      not found in PATH
  ✅  Disk Write              current directory is writable
```

Checks performed:
- Required environment variables
- PostgreSQL TCP connectivity
- NATS TCP connectivity
- External tools (`docker`, `mc`)
- Disk write permissions

---

## 17. CLI Reference

### `partir` — Factory Control

| Command | Description |
|---|---|
| `partir version` | Print version |
| `partir health` | Full system health (JSON) |
| `partir healthz` | Liveness probe |
| `partir doctor` | Full environment diagnostics |
| `partir health serve-health` | HTTP health endpoints |
| `partir factory status` | Factory overview |
| `partir factory workers` | List all workers |
| `partir factory bindings` | Station assignments |
| `partir factory assign <station> <worker>` | Assign worker |
| `partir factory unassign <worker>` | Unassign worker |
| `partir factory metrics <station>` | Station metrics |
| `partir factory init` | Create default stations |
| `partir maintenance inspect <pair>` | Inspect pair |
| `partir maintenance phase <pair> <phase>` | Set maintenance state |
| `partir maintenance release <pair>` | Release pair |
| `partir executors add <name> <url>` | Register executor plugin |
| `partir executors list` | List executors |
| `partir executors health` | Check executor health |
| `partir executors remove <name>` | Remove executor |
| `partir plugin install <slug|url>` | Install plugin from registry |
| `partir plugin update <slug>` | Update plugin to latest |
| `partir plugin list` | List installed plugins |
| `partir plugin list --remote` | Browse plugin registry |
| `partir remote-install worker <user@host>` | Remote worker setup via SSH |
| `partir remote-install plugin <slug> <user@host>` | Remote plugin install |

### `foundry` — Production & Planning

| Command | Description |
|---|---|
| `foundry plan <project>` | Start Round Table session |
| `foundry plan <project> --upload <file>` | Plan with context files |
| `foundry plan <project> --alpha <ref> --beta <ref>` | Plan with rulebook refs |
| `foundry resume` | Resume saved session |
| `foundry scope load <file>` | View saved scope |
| `foundry scope approve <file>` | Submit planned tickets |
| `foundry submit <ticket.json>` | Submit a manual ticket |
| `foundry run <ticket_id>` | Execute a ticket |
| `foundry status <ticket_id>` | Check ticket status |
| `foundry artifacts <ticket_id>` | List artifacts |

### `alpha` — Architecture Architect

| Command | Description |
|---|---|
| `alpha start <name>` | Interactive architecture session |
| `alpha start <name> --upload <file>` | Start with file upload |
| `alpha resume <name>` | Resume from graph memory |
| `alpha import db <name> --dsn <url>` | Import from database |
| `alpha import openapi <name> --spec <file>` | Import from OpenAPI |
| `alpha import catalog <name> --csv <file>` | Import catalog from CSV |
| `alpha init <name>` | Create rulebook manually |
| `alpha lint <name>` | Validate rulebook |
| `alpha build <name> [-o file]` | Compile to JSON |
| `alpha promote <name> -v <ver>` | Promote to version |
| `alpha list` | List all rulebooks |

### `beta` — Design Architect

| Command | Description |
|---|---|
| `beta start <name>` | Interactive design session |
| `beta start <name> --upload <file>` | Start with file upload |
| `beta start <name> --alpha-ref <ref>` | Start with Alpha context |
| `beta resume <name>` | Resume design session |
| `beta export figma <name>` | Export to Figma |
| `beta import figma <name> --file-key <key>` | Import from Figma |
| `beta init <name>` | Create design rulebook |
| `beta lint <name>` | Validate design rules |
| `beta build <name> [-o file]` | Compile to JSON |
| `beta promote <name> -v <ver>` | Promote to version |
| `beta list` | List all design rulebooks |

### `migrate` — Database Migrations

| Command | Description |
|---|---|
| `migrate up` | Apply all pending migrations |
| `migrate down <N>` | Roll back N migrations |
| `migrate version` | Show current version |

---

## 18. Troubleshooting

### Common Issues

| Symptom | Cause | Fix |
|---|---|---|
| `failed to connect to database` | PostgreSQL not running | `docker compose up -d postgres` |
| `no executor plugin configured` | No LLM registered | `partir executors add ollama http://localhost:8081` |
| `rate limit exceeded` | Tenant quota hit | Increase `PARTIR_MAX_TICKETS_PER_HOUR` |
| `omega gate failed` | Output failed quality | Check defect messages, adjust inputs |
| `worker heartbeat timeout` | Worker crashed | Check worker logs, restart |
| `invalid signature` (Discord) | Wrong public key | Verify `publicKeyHex` matches Discord app |
| `alpha start` hangs | Executor not responding | `partir executors health` |
| Grafana shows no data | Telemetry disabled | Set `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `No saved session found` | No prior `foundry plan` | Run `foundry plan <project>` first |

### Docker Service Health

```bash
docker compose ps                  # All services
docker compose logs -f postgres    # Database logs
docker compose logs -f ollama      # LLM logs
docker compose restart grafana     # Restart a service
```

### Reset Everything

```bash
# ⚠️ Warning: destroys all data
docker compose down -v
docker compose up -d
migrate up
```

### Run Full Test Suite

```bash
go test ./...
go test -v ./internal/dispatch/...
go test -v ./internal/resilience/...
go test -v ./internal/remote/...
go test -v ./internal/security/auth/...
```

---

## 19. Project Structure

```
Partir_Core/
├── cmd/                          # CLI entry points
│   ├── alpha/                    #   Alpha architect CLI
│   ├── beta/                     #   Beta designer CLI
│   ├── foundry/                  #   Foundry production CLI
│   ├── partir/                   #   Factory control CLI
│   └── migrate/                  #   Database migration tool
│
├── internal/                     # Private packages
│   ├── alpha/                    #   Alpha architect engine
│   ├── beta/                     #   Beta designer engine
│   ├── foundry/                  #   Dispatcher, planner, round table
│   ├── omega/                    #   Quality gates and defect tracking
│   ├── factory/                  #   Worker & station management
│   ├── storage/                  #   PostgreSQL repos + MinIO blob store
│   ├── config/                   #   Viper-based configuration
│   ├── security/                 #   Auth (JWT), RBAC, secrets, audit
│   ├── resilience/               #   Circuit breaker, rate limiter, DLQ, backoff
│   ├── dispatch/                 #   Priority queue + tenant quotas
│   ├── alerting/                 #   Alert manager (Slack, PagerDuty, webhooks)
│   ├── chatops/                  #   Slack & Discord bot handlers
│   ├── session/                  #   Round Table session persistence
│   ├── doctor/                   #   Environment diagnostics
│   ├── pluginmgr/                #   Plugin install/update/list from registry
│   ├── remote/                   #   SSH provisioning + systemd unit generation
│   ├── collaboration/            #   NATS-based collab API
│   ├── interview/                #   Memory pair assembly
│   ├── maintenance/              #   Pair maintenance workflow
│   ├── sync/                     #   External sync (Jira, Linear stubs)
│   └── ux/                       #   CLI UX toolkit (colors, spinners, progress)
│
├── pkg/                          # Public packages
│   ├── plugin/                   #   Plugin SDK (Executor interface, LLM client)
│   ├── metrics/                  #   Prometheus metrics + health server
│   └── telemetry/                #   OpenTelemetry tracing
│
├── migrations/                   # SQL migrations (001–013)
├── deploy/                       # Deployment configs
│   └── helm/partir-core/         #   Helm chart (Chart.yaml, values.yaml, templates/)
├── scripts/                      # Operational scripts
│   └── backup.sh                 #   PostgreSQL + MinIO backup
├── docs/                         # Documentation
│   ├── plugin_api_reference.md   #   Plugin SDK reference
│   ├── architecture.md           #   Architecture diagrams
│   └── operator_runbook.md       #   Production operations guide
├── docker-compose.yml            # Dev infrastructure
├── docker-compose.gpu.yml        # GPU variant
├── docker-compose.prod.yml       # Production variant
├── Dockerfile                    # Multi-stage production build
├── Makefile                      # Build targets
├── go.mod / go.sum               # Go dependencies
└── .env.example                  # Environment template
```
