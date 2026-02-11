# Partir Core — Plugin API Reference

## Overview
Plugins extend Partir Core by implementing the `Plugin` interface from `github.com/partir/core/pkg/plugin`. Each plugin declares job types it handles, provides input/output schemas, executes work, and defines quality gates.

## Plugin Interface

```go
type Plugin interface {
    // Identity
    ID() string
    Version() string
    Manifest() *Manifest
    Capabilities() []string  // Job types this plugin handles

    // Schemas
    InputSchema(jobType string) json.RawMessage
    OutputSchema(jobType string) json.RawMessage

    // Execution
    Plan(workOrder WorkOrder) ([]Step, error)
    Execute(ctx context.Context, workOrder WorkOrder) (*ExecutionResult, error)

    // Gates
    Gates(jobType string) []string
    Validate(ctx context.Context, artifacts []Artifact) ([]Defect, error)
}
```

## Core Types

### WorkOrder
Passed to `Execute()` — contains everything the plugin needs.

| Field | Type | Description |
|---|---|---|
| `TicketID` | `string` | Unique ticket identifier |
| `JobType` | `string` | The job type being requested |
| `AlphaRef` | `string` | Reference to the Alpha rulebook |
| `BetaRef` | `string` | Reference to the Beta rulebook (optional) |
| `Inputs` | `json.RawMessage` | Job-specific input data |
| `Constraints` | `json.RawMessage` | Execution constraints |
| `Limits` | `Limits` | Max files, diff lines, attempts, cost |
| `Attempt` | `int` | Current attempt number (1-based) |
| `DeltaFields` | `[]string` | Fields to fix on retry |

### ExecutionResult
Returned from `Execute()`.

| Field | Type | Description |
|---|---|---|
| `Artifacts` | `[]Artifact` | Generated artifacts |
| `CostTokens` | `int` | Tokens consumed |
| `CostUSD` | `float64` | Cost in USD |
| `Logs` | `string` | Execution logs |

### Artifact
Each artifact produced by the plugin.

| Field | Type | Description |
|---|---|---|
| `ArtifactID` | `string` | Unique artifact ID |
| `ArtifactType` | `string` | Type identifier (e.g., `code`, `document`) |
| `Hash` | `string` | SHA-256 hash of content |
| `Data` | `json.RawMessage` | The artifact content |
| `Provenance` | `Provenance` | Origin tracking |

### Defect
Returned from `Validate()` when quality gates fail.

| Field | Type | Description |
|---|---|---|
| `GateID` | `string` | Which gate failed |
| `DefectClass` | `string` | Category (e.g., `schema_violation`) |
| `Message` | `string` | Human-readable description |
| `OffendingFields` | `json.RawMessage` | Fields that caused the failure |
| `SuggestedFix` | `string` | Recommended fix |

### Manifest (Security)
Required for signed plugin registration.

| Field | Type | Description |
|---|---|---|
| `ID` | `string` | Plugin identifier |
| `Version` | `string` | Semantic version |
| `Name` | `string` | Display name |
| `Author` | `string` | Author/org |
| `License` | `string` | License type |
| `Permissions` | `[]string` | Required permissions |
| `Signature` | `string` | Hex-encoded Ed25519 signature |

## Lifecycle

```
1. Registry.Register(plugin)  →  Manifest verified (if trust root set)
2. Dispatcher receives Ticket  →  Finds plugin by Capability
3. plugin.Execute(ctx, workOrder)  →  Returns ExecutionResult
4. Omega runs gates  →  plugin.Validate(ctx, artifacts)
5. Pass: artifacts stored  |  Fail: RetryController decides retry/escalate
```

## HTTP Plugin Protocol
Plugins can run as HTTP services. The Foundry dispatches work via:
- `POST /execute` — Send WorkOrder, receive ExecutionResult
- `GET /health` — Liveness check
- `GET /capabilities` — List supported job types

## Example Plugin

```go
type MyPlugin struct{}

func (p *MyPlugin) ID() string            { return "my-plugin" }
func (p *MyPlugin) Version() string       { return "1.0.0" }
func (p *MyPlugin) Manifest() *Manifest   { return &Manifest{ID: "my-plugin"} }
func (p *MyPlugin) Capabilities() []string { return []string{"generate-docs"} }

func (p *MyPlugin) Execute(ctx context.Context, wo WorkOrder) (*ExecutionResult, error) {
    // Your logic here
    return &ExecutionResult{
        Artifacts: []Artifact{{
            ArtifactID:   "doc-1",
            ArtifactType: "document",
            Hash:         "sha256:...",
            Data:         json.RawMessage(`{"content": "..."}`),
        }},
        CostTokens: 500,
    }, nil
}
```
