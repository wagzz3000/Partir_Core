# Partir Core — Prototype

An early-stage implementation of a code factory built on the Partir architecture. This is a node mesh that dispatches work orders to external AI executor plugins and validates returned output through quality gates. The prototype itself does not generate code — it's the router and validator.

**Status:** Incomplete. The dispatcher doesn't retry on its own, factory worker scaling isn't functional, and many CLI commands are stubbed. This was never run as a production service. It should be understood as an architectural reference — not a turnkey system.

---

## What It Does

The prototype implements the **dispatch and quality-gate layer** of the Partir framework:

- Accept work orders (tickets) with defined schemas and constraints
- Route tickets to external AI executor plugins via HTTP (Ollama, cloud LLMs)
- Validate returned output through Omega quality gates (schema, determinism, budget, uniqueness, agent output structure)
- Classify gate failures as typed defects (schema_violation, budget_exceeded, nondeterministic_output, etc.) and store them
- Signal whether a failure is retryable based on defect class history (same class fails twice → stop retrying)
- Store validated artifacts in MinIO with provenance metadata

**Important limitations:**

- The prototype does not generate code itself — it routes prompts to plugins and validates what comes back
- The dispatcher signals `ShouldRetry` but does not actually re-execute in a loop
- Factory worker scaling is partially wired but not functional
- Many CLI commands are stubbed or incomplete

It does **not** implement the diagnostics, therapeutic intervention, maintenance rotation, graph surgery, or drift-management concepts described in the parent project's README.

---

## Architecture

```
Intent → Alpha → Beta → Omega (Gate) → Foundry → Omega (Gate) → Artifacts
```

### Modules

| Module | Purpose | Status |
|--------|---------|--------|
| **Alpha** | Define schemas, catalogs, rules, constraints | Partial — workspace init/lint/build works, promotion stubbed |
| **Beta** | Define render vocab, style rules | Partial — mirrors Alpha structure |
| **Omega** | Validate artifacts, enforce gates, classify failures | Core gates implemented |
| **Foundry** | Route tickets, run jobs, manage retries | Dispatcher and executor routing work |
| **Factory** | Worker registration, node topology | Partial — registration works, scaling not wired |
| **Guard** | Code quality enforcement for agent contributions | Implemented |

### Infrastructure

- **PostgreSQL 16.4** — Registry of record
- **MinIO** — Artifact blob store
- **NATS** — Inter-worker messaging
- **FalkorDB** — Graph database (present in compose, minimally integrated)

### Observability

- **OpenTelemetry** — Traces + metrics via `pkg/telemetry`
- **Prometheus** — Metrics scraping via OTel exporter
- **Grafana** — Dashboards
- **Loki** — Log aggregation

---

## Project Structure

```
prototype/
├── cmd/
│   ├── alpha/          # Alpha CLI (interactive + commands)
│   ├── beta/           # Beta CLI
│   ├── foundry/        # Foundry CLI (dispatch, calibrate)
│   ├── partir/         # Main CLI (server, health, factory, executors, doctor)
│   ├── guard/          # Code quality gate enforcement
│   ├── migrate/        # Database migration runner
│   └── verify_stack/   # Stack verification utility
├── internal/
│   ├── alpha/          # Alpha rulebook workspace
│   ├── beta/           # Beta rulebook workspace
│   ├── omega/          # Quality gate engine
│   ├── foundry/        # Dispatcher + HTTP executor
│   ├── factory/        # Worker registration + topology
│   ├── dispatch/       # Priority queue + tenant quotas
│   ├── resilience/     # Circuit breaker, backoff, DLQ, idempotency
│   ├── security/       # JWT auth, RBAC, audit, secrets
│   ├── storage/        # PostgreSQL + MinIO abstraction
│   ├── worker/         # Worker compute controller
│   ├── server/         # HTTP server
│   └── ...             # Additional packages (maintenance, sync, etc.)
├── pkg/
│   ├── plugin/         # Plugin SDK (executor interface, manifest, registry)
│   ├── metrics/        # Prometheus metric definitions
│   └── telemetry/      # OpenTelemetry provider + metrics
├── migrations/         # PostgreSQL schema migrations
├── deploy/             # Docker Compose, Helm, systemd, observability configs
├── scripts/            # Deployment and maintenance scripts
├── docker-compose.yml  # Development stack
├── docker-compose.prod.yml  # Production stack
├── Dockerfile
└── Makefile
```

---

## Plugin Architecture

The core is **LLM-free by design**. AI execution is delegated to external plugins via HTTP.

```
┌─────────────────────────────────────────────────────────────┐
│                       Partir Core                            │
├─────────────────────────────────────────────────────────────┤
│  Alpha/Beta     │     Foundry      │       Omega            │
│  (emit tickets) │  (routes work)   │  (validates output)    │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼ HTTP
┌─────────────────────────────────────────────────────────────┐
│                    Executor Plugins                          │
├─────────────────────────────────────────────────────────────┤
│  partir-plugin-ollama     │     partir-plugin-helicone      │
│  (local SLM, cost=0)      │     (cloud LLM, usage-based)    │
└─────────────────────────────────────────────────────────────┘
```

Plugins implement the HTTP protocol and return a standardized `AgentResult`:

```go
type AgentResult struct {
    TicketID   string           `json:"ticket_id"`
    RunID      string           `json:"run_id"`
    Artifacts  []AgentArtifact  `json:"artifacts"`
    Telemetry  LLMTelemetry     `json:"llm_telemetry"`
    Success    bool             `json:"success"`
    Defects    []DefectEntry    `json:"defects,omitempty"`
}
```

See `docs/design/plugin-interface.md` in the parent directory for full plugin documentation.

---

## Setup

See `INSTALL.md` for detailed setup instructions.

Quick start:

```bash
# Copy environment config
cp .env.example .env
# Edit .env with your passwords

# Start infrastructure
docker-compose up -d

# Build CLIs
make build

# Run migrations
./bin/migrate up

# Check health
./bin/partir health
```

---

## License

Apache License 2.0 — see `LICENSE` in the repository root.
