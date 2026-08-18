#!/bin/bash
# Partir Core Backup Script
# Usage: ./backup.sh [--db-only | --minio-only | --full]

set -euo pipefail

BACKUP_DIR="${PARTIR_BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_URL="${PARTIR_DB_URL:?PARTIR_DB_URL is required}"
MINIO_ALIAS="${PARTIR_MINIO_ALIAS:-partir}"
MINIO_BUCKET="${PARTIR_MINIO_BUCKET:-partir-artifacts}"

mkdir -p "$BACKUP_DIR"

backup_db() {
    echo "📦 Backing up PostgreSQL..."
    local file="$BACKUP_DIR/db_$TIMESTAMP.sql.gz"
    pg_dump "$DB_URL" | gzip > "$file"
    echo "✅ Database backup: $file ($(du -h "$file" | cut -f1))"
}

backup_minio() {
    echo "📦 Backing up MinIO artifacts..."
    local dir="$BACKUP_DIR/minio_$TIMESTAMP"
    mc mirror "$MINIO_ALIAS/$MINIO_BUCKET" "$dir" --quiet
    echo "✅ MinIO backup: $dir"
}

case "${1:---full}" in
    --db-only)
        backup_db
        ;;
    --minio-only)
        backup_minio
        ;;
    --full)
        backup_db
        backup_minio
        echo ""
        echo "🎉 Full backup complete: $BACKUP_DIR/*_$TIMESTAMP*"
        ;;
    *)
        echo "Usage: $0 [--db-only | --minio-only | --full]"
        exit 1
        ;;
esac
