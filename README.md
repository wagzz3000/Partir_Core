# Partir Core

A modular, plugin-based production system for generating and validating versioned artifacts, governed by immutable rulebooks.

## Philosophy

> **Tokens are spent only on entropy. Everything else must be deterministic.**

## Quick Start

```bash
# Boot the factory starter kit
docker-compose up -d

# Build all CLIs
make build

# Initialize an Alpha rulebook
./bin/alpha init my_game_rules

# Initialize a Beta rulebook
./bin/beta init my_render_rules

# Submit and run a ticket
./bin/foundry submit ticket.json
./bin/foundry run test-001
```

## Architecture

```
Intent → Alpha → Beta → Omega (Gate) → Foundry → Omega (Gate) → Artifacts
```

### Modules

| Module | Purpose |
|--------|---------|
| **Alpha** | Define *law* — schemas, catalogs, rule tables, constraints |
| **Beta** | Define *expression* — render vocab, style rules, animations |
| **Omega** | Authority — validates artifacts, enforces gates, classifies failures |
| **Foundry** | Production system — routes tickets, runs jobs, stores artifacts |
| **Factory 0** | Infrastructure Gate — ensures environment matches `green_tags` registry |

### Storage

- **Postgres 16.4** — Registry of record (locked version)
- **MinIO** — Artifact blob store
- **Green Tags** — Environment version registry (Factory 0)

### Observability

- **OpenTelemetry** — Traces + metrics
- **Prometheus** — Metrics scraping
- **Grafana** — Dashboards
- **Loki** — Log aggregation

## Project Structure

```
partir_core/
├── cmd/
│   ├── alpha/          # Alpha CLI
│   ├── beta/           # Beta CLI
│   ├── foundry/        # Foundry CLI
│   └── omega/          # Omega CLI
├── internal/
│   ├── alpha/          # Alpha module
│   ├── beta/           # Beta module
│   ├── omega/          # Omega validation engine
│   ├── foundry/        # Foundry orchestration
│   └── storage/        # Postgres + S3
├── pkg/
│   └── plugin/         # Plugin SDK
├── migrations/         # DB migrations
└── docker-compose.yml  # Factory starter kit
```

## CLI Reference

### Alpha

```bash
alpha init <name>              # Create new rulebook
alpha lint <name>              # Validate rulebook
alpha build <name>             # Compile to JSON bundle
alpha promote <name> -v 0.1.0  # Promote to version
alpha list                     # List all rulebooks
alpha get <name>               # Get rulebook details
```

### Beta

Same commands as Alpha.

### Foundry

```bash
foundry submit <ticket.json>   # Submit work order
foundry run <ticket_id>        # Execute ticket
foundry status <ticket_id>     # Check status
foundry artifacts <ticket_id>  # List artifacts
foundry export <ticket_id>     # Export as zip
```

## Plugin Architecture

Partir Core is **LLM-free by design**. LLM execution is delegated to external plugins via HTTP.

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

### Managing Executors

```bash
# Register an executor
partir executors add --name agent_ollama --url http://localhost:8081

# List registered executors
partir executors list

# Test connectivity
partir executors ping --name agent_ollama

# Remove an executor
partir executors remove --name agent_ollama
```

### Executor Config

Executors are stored in `~/.partir/config.yaml`:

```yaml
executors:
  agent_ollama:
    type: http
    url: http://localhost:8081
  agent_helicone:
    type: http
    url: http://localhost:8082
```

### HTTP Plugin Protocol

Executor plugins expose these endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/capabilities` | GET | Models, limits, features |
| `/execute` | POST | WorkOrder → RunResult |
| `/estimate` | POST | Cost/latency estimate (optional) |

### Writing Plugins

Plugins implement the HTTP protocol and return standardized `AgentResult`:

```go
type AgentResult struct {
    TicketID   string           `json:"ticket_id"`
    RunID      string           `json:"run_id"`
    Plan       *AgentPlan       `json:"plan,omitempty"`
    Patch      *AgentPatch      `json:"patch,omitempty"`
    Artifacts  []AgentArtifact  `json:"artifacts"`
    Telemetry  LLMTelemetry     `json:"llm_telemetry"`
    Success    bool             `json:"success"`
    Defects    []DefectEntry    `json:"defects,omitempty"`
}

type LLMTelemetry struct {
    Provider   string  `json:"provider"`   // "ollama" or "helicone"
    Model      string  `json:"model"`
    TokensIn   int     `json:"tokens_in"`
    TokensOut  int     `json:"tokens_out"`
    CostUSD    float64 `json:"cost_usd"`   // 0 for local
    LatencyMs  int64   `json:"latency_ms"`
}
```

See `References/plugin_interface.md` for full documentation.

## Observability

All telemetry flows through OpenTelemetry:

- **Traces**: Spans for all CLI commands and gate executions
- **Metrics**: Ticket counts, gate pass/fail rates, token usage, costs
- **Logs**: Structured JSON with `trace_id`/`span_id` correlation

```bash
# Enable telemetry
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318

# Disable telemetry
export PARTIR_TELEMETRY_DISABLED=true
```

## License

MIT
