# Partir Core Stack Policy

This document defines the approved technology stack for Partir Core.  
**All contributions MUST use these tools unless a ticket explicitly allows deviation.**

---

## Approved Stack

| Category | Tool | Version | Notes |
|----------|------|---------|-------|
| **Observability** | OpenTelemetry | SDK v1.40+ | Unified traces, metrics, logs |
| **Metrics Export** | OTLP → Prometheus | via OTel Collector | No standalone Prometheus client servers |
| **Logs** | Structured JSON → Loki | via OTel/Alloy | Use `slog` with trace correlation |
| **Database** | PostgreSQL | 16.4 (pinned) | Green tag enforced |
| **Blob Storage** | MinIO | RELEASE.2024-11-07 | S3-compatible |
| **Migrations** | golang-migrate | v4 | SQL files in `migrations/` |
| **CLI Framework** | Cobra | v1.8+ | All CLIs use this |
| **JSON Schema** | jsonschema/v5 | Santhosh-Tekuri | For Alpha/Beta validation |

---

## Approved Metrics Path (Enforceable)

> **`pkg/telemetry` is the ONLY authorized location for metrics infrastructure.**

| Component | Approved Location |
|-----------|-------------------|
| Meter Provider | `pkg/telemetry/provider.go` |
| Metric Definitions | `pkg/telemetry/metrics.go` |
| Attribute Labels | `pkg/telemetry/attributes.go` |
| OTLP Exporters | `pkg/telemetry/provider.go` |

**Core services export metrics via existing service HTTP endpoint (shared server), NOT a separate metrics-only server.**

---

## Forbidden Patterns

The following patterns are **automatically rejected by `cmd/guard`:**

| Pattern | Reason | Gate |
|---------|--------|------|
| `metrics_server.go` | Use OTel exporter | D |
| `telemetry_server.go` | Use OTel exporter | D |
| `prom_server.go` | Use OTel exporter | D |
| `custom_metrics.go` | Use `pkg/telemetry` | D |
| `prometheus.go` (outside pkg/telemetry) | Use OTel metrics | D |
| `promhttp.Handler()` | Use OTel exporter | D |
| `prometheus.NewRegistry()` | Use OTel metrics | D |
| `http.ListenAndServe` in metrics file | Use shared server | D |
| New logging framework | Use `slog` | Review |
| New config system | Use env vars | Review |
| Game/domain-specific code in Core | Use `examples/plugins/` | B |

---

## Adding New Dependencies

Any new dependency MUST include a **Justification Block** in the PR/commit:

```markdown
## New Dependency: <package-name>

**Why existing tools don't fit:**
<explanation>

**Why this library is necessary:**
<explanation>

**Why this library over alternatives:**
<explanation>
```

Dependencies without justification will be rejected by code review.

---

## Prometheus Scrape Configuration

The `prometheus.yml` scrapes Partir services' shared HTTP endpoints.

> **Important:** Services expose `/metrics` via their main HTTP server, NOT a dedicated metrics server.

```yaml
# Correct: Scrape shared service endpoint
- job_name: partir-foundry
  static_configs:
    - targets: ['foundry:8080']

# Wrong: Dedicated metrics server on different port
# - targets: ['foundry-metrics:9090']  # ← FORBIDDEN
```
