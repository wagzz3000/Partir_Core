#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════"
echo "  Partir Core — Production Boot"
echo "═══════════════════════════════════════════════════"

# ── 0. Ensure we are in the project root ─────────────────
cd /root/partir-core

# ── 1. Strip Windows CRLF from all text files ────────────
#    Files SCP'd from Windows carry \r that breaks bash.
echo "[1/7] Stripping CRLF line endings..."
find . -name '*.sql' -o -name '*.env*' -o -name '*.yml' -o -name '*.yaml' \
       -o -name '*.go' -o -name '*.sh' -o -name '*.mod' -o -name '*.sum' \
  | xargs -r sed -i 's/\r$//'

# ── 2. Generate .env from template if missing ────────────
echo "[2/7] Configuring environment..."
if [ ! -f .env ]; then
    cp .env.example .env

    # Generate random secrets
    DB_PASS=$(openssl rand -hex 16)
    MINIO_PASS=$(openssl rand -hex 16)
    GRAFANA_PASS=$(openssl rand -hex 12)
    JWT_SECRET=$(openssl rand -base64 32 | tr -d '/+=' | head -c 40)

    # Replace all placeholders
    sed -i "s|changeme_db_password|${DB_PASS}|g" .env
    sed -i "s|changeme_minio_password|${MINIO_PASS}|g" .env
    sed -i "s|changeme_grafana_password|${GRAFANA_PASS}|g" .env
    sed -i "s|changeme_jwt_secret|${JWT_SECRET}|g" .env

    echo "  → Generated .env with random secrets"
else
    echo "  → .env already exists, keeping existing config"
fi

# ── 3. Source environment variables ──────────────────────
set -a
source .env
set +a
echo "  → Environment loaded (DB_USER=${PARTIR_DB_USER})"

# ── 4. Start Docker services ────────────────────────────
echo "[3/7] Starting Docker services..."
docker compose up -d

# ── 5. Wait for Postgres to accept authenticated connections
echo "[4/7] Waiting for Postgres..."
RETRIES=0
MAX_RETRIES=30
until docker exec partir-core-postgres-1 \
        psql -U "${PARTIR_DB_USER}" -d "${PARTIR_DB_NAME}" \
        -c "SELECT 1" > /dev/null 2>&1; do
    RETRIES=$((RETRIES + 1))
    if [ $RETRIES -ge $MAX_RETRIES ]; then
        echo "  ✗ Postgres failed to become ready after ${MAX_RETRIES} attempts"
        docker logs partir-core-postgres-1 --tail 20
        exit 1
    fi
    echo "  - Attempt ${RETRIES}/${MAX_RETRIES} — sleeping 3s..."
    sleep 3
done
echo "  ✓ Postgres is ready"

# ── 6. Build CLI binaries ───────────────────────────────
echo "[5/7] Building CLI binaries..."
/usr/local/go/bin/go build -o /usr/local/bin/migrate ./cmd/migrate
/usr/local/go/bin/go build -o /usr/local/bin/partir ./cmd/partir
/usr/local/go/bin/go build -o /usr/local/bin/foundry ./cmd/foundry
echo "  ✓ migrate, partir, foundry installed"

# ── 7. Run database migrations ──────────────────────────
echo "[6/7] Running migrations..."

# ── 7. Run database migrations ──────────────────────────
echo "[6/7] Running migrations..."
docker pull migrate/migrate:v4.17.0

# Use a temporary container to run migrate, attached to the same network
# This avoids host networking issues and ensures we can reach 'postgres' service
docker run --rm --network partir-core_default \
  -v $(pwd)/migrations:/migrations \
  -e DATABASE_URL="postgres://${PARTIR_DB_USER}:${PARTIR_DB_PASSWORD}@postgres:5432/${PARTIR_DB_NAME}?sslmode=disable" \
  migrate/migrate:v4.17.0 \
  -path=/migrations/ \
  -database "postgres://${PARTIR_DB_USER}:${PARTIR_DB_PASSWORD}@postgres:5432/${PARTIR_DB_NAME}?sslmode=disable" \
  up

echo "  ✓ Migrations applied"

# ── 8. Health check & status ────────────────────────────
echo "[7/7] Running diagnostics..."
echo ""
/usr/local/bin/partir doctor
echo ""
docker compose ps
echo ""
echo "═══════════════════════════════════════════════════"
echo "  ✓ Deployment complete!"
echo "═══════════════════════════════════════════════════"
