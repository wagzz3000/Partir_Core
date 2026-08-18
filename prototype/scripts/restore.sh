#!/usr/bin/env bash
# Partir Core Restore Script
# Usage: ./scripts/restore.sh <backup_file.tar.gz>

set -euo pipefail

BACKUP_FILE="${1:-}"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.tar.gz>"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "Restoring from $BACKUP_FILE..."

# Extract backup
tar -xzf "$BACKUP_FILE" -C "$TEMP_DIR"
BACKUP_DIR=$(find "$TEMP_DIR" -maxdepth 1 -type d | head -n 1)

# Source environment
source /etc/partir/partir.env 2>/dev/null || true
PG_USER="${PARTIR_DB_USER:-partir}"
PG_DB="${PARTIR_DB_NAME:-partir}"

# 1. Restore Postgres
if [ -f "$BACKUP_DIR/postgres.dump" ]; then
    echo "Restoring database..."
    # Drop and recreate database to ensure clean state
    dropdb -U "$PG_USER" "$PG_DB" --if-exists
    createdb -U "$PG_USER" "$PG_DB"
    pg_restore -U "$PG_USER" -d "$PG_DB" "$BACKUP_DIR/postgres.dump"
fi

# 2. Restore MinIO Data
if [ -f "$BACKUP_DIR/artifacts.tar.gz" ]; then
    echo "Restoring artifacts..."
    # Warning: This overwrites existing data
    tar -xzf "$BACKUP_DIR/artifacts.tar.gz" -C /var/lib/partir
fi

# 3. Restore Config
if [ -f "$BACKUP_DIR/partir.env" ]; then
    echo "Restoring config..."
    cp "$BACKUP_DIR/partir.env" /etc/partir/partir.env.restored
    echo "Config restored to /etc/partir/partir.env.restored (manual overwrite required)"
fi

echo "Restore complete."
