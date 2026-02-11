# Partir Core - Installation Guide

This guide covers setting up Partir Core, its Orchestrator Workers, and optional GPU acceleration.

## Prerequisites

- **Go 1.23+**
- **Docker & Docker Compose v2+**
- **Git**
- **NVIDIA Container Toolkit** (Optional, for GPU support)

## 1. Quick Start (Production Stack)

The simplest way to run Partir is using the production Docker stack, which includes:
- **Core**: API, Registry, and Dispatcher.
- **Workers**: Alpha, Beta, and Omega orchestrators (pre-configured with OWSLM).
- **Observability**: Prometheus, Grafana, Loki, Tempo.
- **Infrastructure**: Postgres, MinIO, NATS, FalkorDB, Ollama.

### Deployment
```bash
# 1. Clone Repository
git clone https://github.com/partir/core.git
cd core

# 2. Configure Environment
cp .env.example .env
# Edit .env to set database passwords and secrets

# 3. Start Stack (CPU Only)
docker-compose -f docker-compose.prod.yml up -d
```

### Enable GPU Acceleration
If you have an NVIDIA GPU and the Container Toolkit installed:

```bash
docker-compose -f docker-compose.prod.yml -f docker-compose.gpu.yml up -d
```

Valid `GPU_STRATEGY` values in `.env` (or default to auto):
- `auto`: Detects local GPU via `nvidia-smi`.
- `cpu`: Forces CPU-only mode.
- `remote`: Uses `GPU_REMOTE_ENDPOINT` for inference.

## 2. CLI Setup

Install the Partir CLI to manage the factory.

```bash
# Build from source
make build

# Verify Health
./bin/partir health
```

## 3. Factory Management

### Checking Status
View the status of connected workers and their slots.
```bash
./bin/partir factory status
```

### Auditing Bindings
Check worker identity, memory bindings, and hardware capabilities.
```bash
./bin/partir factory bindings
```

### Scaling Workers
You can scale workers using Docker Compose:
```bash
docker-compose -f docker-compose.prod.yml up -d --scale alpha-worker=3
```
*Note: New workers will automatically register with the factory registry.*

## 4. Manual Plugin Setup (Development)

If you are developing plugins or running them outside Docker:

### Worker Plugin
```bash
cd ../partir-plugin-worker
export NATS_URL=nats://localhost:4222
export OLLAMA_URL=http://localhost:11434
export WORKER_TYPE=alpha
go run . --port 8090
```

### External Executors
You can still register external executors (like cloud LLMs) via the CLI:
```bash
./bin/partir executors add --name cloud-gpt4 --url http://localhost:8082
```

## Troubleshooting

**GPU Not Detected?**
- Ensure `nvidia-smi` works on the host.
- Ensure `GPU_STRATEGY=auto` (default).
- Check worker logs: `docker-compose logs alpha-worker | grep "Hardware Profile"`

**OOM Errors?**
- Workers automatically retry with lower context on OOM.
- Check metrics in Grafana: `partir_worker_oom_retries_total`.
- If persistent, the worker will self-quarantine.

**Observability**
- Grafana: `http://localhost:3000` (User/Pass in .env)
- Prometheus: `http://localhost:9090`
