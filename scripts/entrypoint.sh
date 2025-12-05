#!/bin/sh
set -e

# Logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Trap signals
cleanup() {
    log "Received termination signal, shutting down..."
    kill -TERM "$PID_GO" 2>/dev/null
    kill -TERM "$PID_BUN" 2>/dev/null
    wait "$PID_GO"
    wait "$PID_BUN"
    exit 0
}

trap cleanup INT TERM

# Start CCXT Service (Bun) in background
log "Starting CCXT Service..."
cd /app/ccxt
bun run dist/index.js &
PID_BUN=$!

# Wait for CCXT to be ready (optional, but good practice)
# We can just wait a few seconds or use a health check loop
sleep 2

# Auto-Run Migrations if enabled
if [ "${RUN_MIGRATIONS}" = "true" ]; then
    log "Running database migrations..."
    # Ensure migrate.sh is executable
    chmod +x database/migrate.sh

    # Export DB_ vars from DATABASE_ vars for migrate.sh
    export DB_HOST="${DATABASE_HOST:-localhost}"
    export DB_PORT="${DATABASE_PORT:-5432}"
    export DB_NAME="${DATABASE_DBNAME:-celebrum_ai}"
    export DB_USER="${DATABASE_USER:-postgres}"
    export DB_PASSWORD="${DATABASE_PASSWORD:-postgres}"

    # Wait for database to be ready
    log "Waiting for database at ${DB_HOST}:${DB_PORT}..."
    until PGPASSWORD="${DB_PASSWORD}" pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}"; do
        log "Database not ready, waiting..."
        sleep 2
    done
    log "Database is ready."

    if ./database/migrate.sh; then
        log "Migrations completed successfully."
    else
        log "Migration failed. Exiting."
        exit 1
    fi
fi

# Start Go App in background
log "Starting Main Application..."
cd /app
./main &
PID_GO=$!

# Monitor processes
log "All services started. Monitoring..."
wait "$PID_GO" "$PID_BUN"
