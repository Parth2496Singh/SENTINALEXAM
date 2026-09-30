#!/bin/sh
# migrate.sh — runs all SQL migration files in order inside the backend container.
# Usage: docker compose exec backend /app/scripts/migrate.sh
set -e

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/app/migrations}"
DB_HOST="${DATABASE_HOST:-postgres}"
DB_PORT="${DATABASE_PORT:-5432}"
DB_USER="${DATABASE_USER:-sentinel}"
DB_PASSWORD="${DATABASE_PASSWORD:-sentinel_dev_password}"
DB_NAME="${DATABASE_NAME:-sentinelexam}"

export PGPASSWORD="$DB_PASSWORD"

echo "[migrate] Running migrations from $MIGRATIONS_DIR against $DB_HOST:$DB_PORT/$DB_NAME"

for file in "$MIGRATIONS_DIR"/*.sql; do
  echo "[migrate] Applying: $file"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file" --single-transaction
  echo "[migrate] Done: $file"
done

echo "[migrate] All migrations applied successfully."
