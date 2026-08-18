#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════"
echo "  Partir Core — Graceful Shutdown"
echo "═══════════════════════════════════════════════════"

cd /root/partir-core

echo "[1/3] Stopping services with SIGTERM (graceful)..."
# 'stop' allows services to finish in-flight requests / save state
docker compose stop

echo "[2/3] Removing containers and networks..."
# 'down' removes the containers but PRESERVES volumes (data is safe)
docker compose down

echo "[3/3] Verifying shutdown..."
if [ -z "$(docker compose ps -q)" ]; then
    echo "  ✓ All services stopped."
    echo ""
    echo "You may now safely power off the server."
else
    echo "  ! Some services may still be running:"
    docker compose ps
fi
echo "═══════════════════════════════════════════════════"
