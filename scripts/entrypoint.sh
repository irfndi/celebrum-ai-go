#!/bin/bash
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
# Use absolute path for directory change to ensure we know where we are
cd /app/ccxt
bun run dist/index.js &
PID_BUN=$!

# Wait for CCXT to be ready (optional, but good practice)
# We can just wait a few seconds or use a health check loop
sleep 2

# Auto-Run Migrations if enabled
if [ "${RUN_MIGRATIONS}" = "true" ]; then
    log "Running database migrations..."
    
    # Use absolute path for migrate script
    MIGRATE_SCRIPT="/app/database/migrate.sh"
    
    # Ensure migrate.sh is executable
    if [ -f "$MIGRATE_SCRIPT" ]; then
        chmod +x "$MIGRATE_SCRIPT"
    else
        log "Error: Migration script not found at $MIGRATE_SCRIPT"
        exit 1
    fi

    # Check for connection string in DATABASE_HOST
    if echo "${DATABASE_HOST}" | grep -qE "^postgres(ql)?://"; then
        log "DATABASE_HOST contains a full URL, using it as DATABASE_URL"
        export DATABASE_URL="${DATABASE_HOST}"
    fi
    
    # Coolify Fix: Replace generic hostname in DATABASE_URL with actual DATABASE_HOST
    # Coolify provides DATABASE_URL with generic hostname (e.g., @postgres:5432)
    # but DATABASE_HOST contains the actual Docker service name (e.g., postgresql-develop)
    if [ -n "$DATABASE_URL" ] && [ -n "$DATABASE_HOST" ]; then
        # Only patch if DATABASE_HOST is not a full URL
        if ! echo "${DATABASE_HOST}" | grep -qE "^postgres(ql)?://"; then
            # Simple replacement: @anything:port -> @DATABASE_HOST:port
            # This handles @postgres:5432, @localhost:5432, @h4sggs8coswc4csg440k4s4o:5432, etc.
            if echo "$DATABASE_URL" | grep -q "@.*:[0-9]\+"; then
                PATCHED_URL=$(echo "$DATABASE_URL" | sed "s|@[^:/@]*:|@${DATABASE_HOST}:|")
                if [ "$PATCHED_URL" != "$DATABASE_URL" ]; then
                    log "Patching DATABASE_URL: replacing hostname with ${DATABASE_HOST}"
                    export DATABASE_URL="$PATCHED_URL"
                fi
            fi
        fi
    fi
    
    # Debug: Log ALL environment variables to diagnose the issue
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] === Database Environment Variables ==="
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_URL=${DATABASE_URL:-<NOT SET>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_HOST=${DATABASE_HOST:-<NOT SET>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_PORT=${DATABASE_PORT:-<NOT SET>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_USER=${DATABASE_USER:-<NOT SET>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_DBNAME=${DATABASE_DBNAME:-<NOT SET>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DATABASE_PASSWORD=${DATABASE_PASSWORD:+<REDACTED>}"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ======================================"

    # Simple approach: If DATABASE_URL is set, use it directly
    if [ -n "$DATABASE_URL" ]; then
        log "Using DATABASE_URL for connection..."
        export DATABASE_URL
        # Wait for database to be ready using URL
        until pg_isready -d "$DATABASE_URL" -t 3; do
            log "Database not ready, waiting..."
            sleep 2
        done
    else
        # Fall back to component-based configuration
        export DB_HOST="${DATABASE_HOST:-localhost}"
        export DB_PORT="${DATABASE_PORT:-5432}"
        export DB_NAME="${DATABASE_DBNAME:-celebrum_ai}"
        export DB_USER="${DATABASE_USER:-postgres}"
        export DB_PASSWORD="${DATABASE_PASSWORD:-postgres}"

        # Wait for database to be ready
        log "Waiting for database at ${DB_HOST}:${DB_PORT} (database: ${DB_NAME})..."
        until PGPASSWORD="${DB_PASSWORD}" pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -t 3; do
            log "Database not ready, waiting..."
            sleep 2
        done
    fi
    log "Database is ready."

    if "$MIGRATE_SCRIPT"; then
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
