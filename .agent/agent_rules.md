# Partir Core Agent Rules

These rules govern agent (AI/automation) behavior when contributing to Partir Core.  
**Violation of these rules will cause automatic rejection by `cmd/guard` and code review.**

---

## Rule 1: No New Subsystem If Package Exists

> **If an approved internal package exists, the agent MUST use it.**

Before creating any new subsystem (server, client, wrapper, framework, "manager"):

1. **Search the repo first** (required command):
   ```bash
   rg -n "otel|OpenTelemetry|prometheus|/metrics|observability|telemetry" .
   ```

2. **If existing package found** → Use it. Creating a parallel implementation is **disallowed**.

3. **If no existing package** → Add a **Justification Block** in the PR:
   ```markdown
   ## New Subsystem Justification
   
   **Existing tools searched:** <list search terms used>
   **Why existing tools don't fit:** <explanation>
   **Why new implementation is necessary:** <explanation>
   **Why this library over alternatives:** <explanation>
   ```

### Examples

| Scenario | Correct Action |
|----------|----------------|
| Need metrics | Use `pkg/telemetry` |
| Need new metric type | Extend `pkg/telemetry/metrics.go` |
| Need HTTP endpoint | Extend existing service's HTTP server |
| Need new logging | Use `slog` with existing patterns |

---

## Rule 2: No New Servers/Endpoints Without Justification

> **If you add a new server/endpoint, you must justify why existing endpoints can't be extended.**

**Approved patterns:**
- Core services export metrics via **existing service HTTP endpoint** (shared server)
- No separate metrics-only servers allowed
- All `/metrics` endpoints must be served by the single shared core server

**Forbidden:**
- ❌ New `http.ListenAndServe` or `http.Server{}` solely for metrics
- ❌ Standalone Prometheus HTTP server when OTel exists
- ❌ New port bindings for observability endpoints

---

## Rule 3: Approved Metrics Emission Path (Enforceable)

> **`pkg/telemetry` is the ONLY place for meter providers/exporters.**

| Component | Approved Location |
|-----------|-------------------|
| Meter Provider | `pkg/telemetry/provider.go` |
| Metric Definitions | `pkg/telemetry/metrics.go` |
| Attribute Labels | `pkg/telemetry/attributes.go` |
| OTLP Exporters | `pkg/telemetry/provider.go` |

**Any other location for metrics infrastructure is automatically rejected.**

---

## Rule 4: Plan-Required New Files

> **Agent may ONLY create files listed in the ticket plan or `allowed_new_files`.**

When creating new files:

1. File must be explicitly listed in the approved implementation plan
2. Or file must be in a pre-approved directory for the ticket type
3. Any unlisted file creation = **automatic reject**

**Pre-approved file patterns:**
- `*_test.go` in existing package directories
- Files explicitly listed in `implementation_plan.md`

**Forbidden:**
- ❌ Creating `game_items.go` because "I saw game items mentioned"
- ❌ Creating new packages not in the plan
- ❌ Creating utility files "for convenience"

---

## Rule 5: No Parallel Stacks

> **Do NOT introduce alternative stacks for the same capability.**

| If This Exists | Do NOT Add |
|----------------|------------|
| OpenTelemetry (`pkg/telemetry`) | Standalone Prometheus client server |
| `slog` | logrus, zap, zerolog |
| Cobra CLI | New CLI framework |
| `golang-migrate` | New migration framework |

---

## Rule 6: No New Dependencies Without Justification

Any new dependency MUST include in the PR:

```json
{
  "new_deps": [
    {
      "package": "github.com/example/pkg",
      "reason": "Required for X because existing Y doesn't support Z"
    }
  ]
}
```

---

## Rule 7: No Refactors Unless Asked

- Do NOT refactor code unless the ticket explicitly says "refactor"
- If integration requires refactor, **propose a decomposition plan first**
- Small, surgical changes > sweeping rewrites

---

## Enforcement

These rules are enforced by:

| Gate | Checks |
|------|--------|
| `cmd/guard` Gate D | Detects `metrics_server`, `telemetry_server`, `prom_server`, `custom_metrics` in filenames |
| `cmd/guard` Gate D | Detects `http.ListenAndServe` in new metrics-only packages |
| `cmd/guard` Gate D | Detects `promhttp.Handler()` when OTel exists |
| Code Review | Justification blocks for new dependencies |
| Code Review | Plan-required new files check |

### Denylist Patterns (Auto-Fail)

Files/packages containing:
- `metrics_server`
- `telemetry_server`
- `prom_server`
- `custom_metrics`
- `prometheus.go` (outside `pkg/telemetry`)

Code patterns:
- `http.ListenAndServe` in new package whose only purpose is metrics
- `promhttp.Handler()` when OTel already in repo
- `prometheus.NewRegistry()` outside approved locations
