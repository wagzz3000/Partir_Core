# Partir Plugin Interface

This document describes how to implement LLM executor plugins for Partir Core.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Partir Core                            │
├─────────────────────────────────────────────────────────────┤
│  Alpha/Beta     │     Foundry      │       Omega            │
│  (LLM-free)     │  (routes tickets)│  (validates output)    │
│                 │                  │                        │
│  Emits tickets  │  → Calls Plugin  │  ← Validates Result    │
│  like:          │    via Executor  │    via Gates           │
│  ai_draft       │    interface     │                        │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    External Plugins                          │
├─────────────────────────────────────────────────────────────┤
│  partir-plugin-ollama     │     partir-plugin-helicone      │
│  (local SLM)              │     (cloud LLM proxy)           │
│  - GPU or CPU             │     - OpenAI, Anthropic, etc.   │
│  - cost_usd = 0           │     - cost tracking via Helicone│
└─────────────────────────────────────────────────────────────┘
```

## Implementing an Executor Plugin

External plugins must implement the `Executor` interface from `pkg/plugin/executor.go`:

```go
type Executor interface {
    ID() string
    Name() string
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
    Capabilities() ExecutorCapabilities
    HealthCheck(ctx context.Context) error
}
```

## Standardized Output Contract

**All plugins MUST return `AgentResult`** from `pkg/plugin/agent_result.go`:

```go
type AgentResult struct {
    TicketID    string           `json:"ticket_id"`
    RunID       string           `json:"run_id"`
    Plan        *AgentPlan       `json:"plan,omitempty"`       // Optional
    Patch       *AgentPatch      `json:"patch,omitempty"`      // Optional
    Artifacts   []AgentArtifact  `json:"artifacts"`            // Required
    Telemetry   LLMTelemetry     `json:"llm_telemetry"`        // Required
    Success     bool             `json:"success"`
    Defects     []DefectEntry    `json:"defects,omitempty"`    // On failure
}
```

### LLMTelemetry (Required)

```go
type LLMTelemetry struct {
    Provider    string    `json:"provider"`     // "ollama" or "helicone"
    Model       string    `json:"model"`        // e.g., "llama3.2", "gpt-4o"
    TokensIn    int       `json:"tokens_in"`    // Prompt tokens
    TokensOut   int       `json:"tokens_out"`   // Completion tokens
    CostUSD     float64   `json:"cost_usd"`     // 0 for local Ollama
    LatencyMs   int64     `json:"latency_ms"`
    FinishReason string   `json:"finish_reason"` // "stop", "length", "error"
}
```

### DefectEntry (On Failure)

```go
type DefectEntry struct {
    Class      string `json:"class"`      // "schema", "policy", "guard", "budget"
    Severity   string `json:"severity"`   // "error", "warning", "info"
    Gate       string `json:"gate"`       // Which gate caught this
    Code       string `json:"code"`       // Machine-readable error code
    Message    string `json:"message"`    // Human-readable description
    Retryable  bool   `json:"retryable"`  // Can this be retried?
}
```

## Omega Validation

Omega runs these gates on every `AgentResult`:

| Gate | Validates |
|------|-----------|
| `agent_output` | Schema correctness of AgentResult |
| `schema` | Artifacts match their JSON schemas |
| `budget` | Token/cost limits not exceeded |
| `determinism` | Artifact hashes match (where applicable) |
| `guard` | No forbidden terms, patterns, or imports |

## Plugin Repos

### partir-plugin-ollama

Local SLM execution via Ollama.

```go
// Configuration
type OllamaConfig struct {
    Endpoint string // Default: "http://localhost:11434"
    Model    string // Default: "llama3.2"
    GPU      bool   // Enable GPU acceleration
}
```

### partir-plugin-helicone

Cloud LLM proxy via Helicone for cost tracking.

```go
// Configuration
type HeliconeConfig struct {
    APIKey      string // Helicone API key
    Provider    string // "openai", "anthropic", "together"
    Model       string // e.g., "gpt-4o", "claude-3-sonnet"
    MaxCostUSD  float64 // Budget cap per request
}
```

## Ticket Types

Alpha/Beta emit tickets that Foundry routes to plugins:

| Ticket Type | Description |
|-------------|-------------|
| `alpha.ai_draft_rulebook` | Generate Alpha rulebook draft |
| `beta.ai_draft_rulebook` | Generate Beta rulebook draft |
| `foundry.generate` | Generic content generation |
| `foundry.transform` | Transform existing artifacts |
