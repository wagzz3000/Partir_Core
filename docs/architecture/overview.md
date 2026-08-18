# Partir Core — Architecture

## System Overview

```mermaid
graph TB
    subgraph CLI["CLI Layer"]
        FC["foundry CLI"]
        PC["partir CLI"]
    end

    subgraph Core["Core Engine"]
        DISP["Dispatcher"]
        ALPHA["Alpha Architect"]
        BETA["Beta Designer"]
        OMEGA["Omega Inspector"]
        RT["Round Table"]
        PLAN["Planner"]
    end

    subgraph Plugins["Plugin Layer"]
        REG["Plugin Registry"]
        P1["Plugin A"]
        P2["Plugin B"]
        HTTP["HTTP Plugin Protocol"]
    end

    subgraph Security["Security Layer"]
        AUTH["JWT Authenticator"]
        RBAC["RBAC Middleware"]
        AUDIT["Audit Logger"]
        SECRETS["Secret Provider"]
    end

    subgraph Resilience["Resilience Layer"]
        CB["Circuit Breaker"]
        RL["Rate Limiter"]
        DLQ["Dead Letter Queue"]
        BACKOFF["Exp Backoff"]
        IDEMP["Idempotency Store"]
        FALL["Fallback Chain"]
    end

    subgraph Data["Data Layer"]
        PG["PostgreSQL"]
        MINIO["MinIO"]
        NATS["NATS"]
    end

    subgraph Observability["Observability"]
        PROM["Prometheus /metrics"]
        OTEL["OTel Tracing"]
        ALERT["Alert Manager"]
        HEALTH["/health + /ready"]
    end

    FC --> DISP
    PC --> ALPHA & BETA & OMEGA & RT

    DISP --> REG
    REG --> P1 & P2 & HTTP
    DISP --> OMEGA

    OMEGA --> CB
    DISP --> RL
    DISP --> DLQ
    DISP --> IDEMP

    AUTH --> RBAC
    RBAC --> DISP

    DISP --> PG & MINIO
    PC --> PG & NATS

    DISP --> PROM & OTEL
    ALERT --> DISP
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI as Foundry CLI
    participant Dispatcher
    participant Plugin
    participant Omega
    participant Storage as PostgreSQL + MinIO

    User->>CLI: foundry submit ticket.yaml
    CLI->>Dispatcher: Submit(Ticket)
    Dispatcher->>Dispatcher: Idempotency Check
    Dispatcher->>Dispatcher: Quota Check
    Dispatcher->>Plugin: Execute(WorkOrder)
    Plugin-->>Dispatcher: ExecutionResult + Artifacts
    Dispatcher->>Omega: RunPostExecution(artifacts)
    Omega->>Omega: Run Gates (Schema, Version, Determinism)

    alt All Gates Pass
        Omega-->>Dispatcher: PASS
        Dispatcher->>Storage: Store Artifacts
        Dispatcher-->>CLI: ✅ Completed
    else Gate Failure
        Omega-->>Dispatcher: FAIL + Defects
        Dispatcher->>Dispatcher: RetryController.ShouldRetry?
        alt Retry
            Dispatcher->>Plugin: Execute(WorkOrder, attempt=2)
        else Escalate
            Dispatcher->>Storage: Dead Letter Queue
            Dispatcher-->>CLI: ❌ Escalated
        end
    end
```

## Package Structure

```
partir-core/
├── cmd/
│   ├── foundry/          # Foundry CLI (dispatch, status, plugins)
│   └── partir/           # Partir CLI (alpha, beta, omega, round-table)
├── internal/
│   ├── alerting/         # Webhook alert manager
│   ├── alpha/            # Alpha Architect workspace
│   ├── beta/             # Beta Designer workspace
│   ├── config/           # Centralized configuration
│   ├── dispatch/         # Priority queue + tenant quotas
│   ├── factory/          # Factory registration + topology
│   ├── foundry/          # Dispatcher + execution engine
│   ├── omega/            # Quality gates + retry controller
│   ├── resilience/       # Circuit breaker, backoff, DLQ, idempotency
│   ├── security/
│   │   ├── auth/         # JWT authenticator
│   │   ├── audit/        # Audit logging + compliance export
│   │   ├── middleware/   # Auth + RBAC middleware
│   │   └── secrets/      # SecretProvider interface
│   ├── storage/          # PostgreSQL + MinIO abstraction
│   └── sync/             # External sync (Jira, Linear)
├── pkg/
│   ├── metrics/          # Prometheus metrics + health server
│   ├── plugin/           # Plugin SDK (interface, manifest, registry)
│   └── telemetry/        # OpenTelemetry tracing
├── migrations/           # Database migrations (001-013)
└── deploy/               # Docker + Helm charts
```
