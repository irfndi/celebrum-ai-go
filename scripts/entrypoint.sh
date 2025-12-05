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
        export DATABASE_URL="${DATABASE_HOST}"
    fi

    # Coolify/Docker DNS Fix:
    # If DATABASE_URL uses generic 'postgres' hostname but we have a specific DATABASE_HOST,
    # we patch the URL to use the correct hostname.
    if [ -n "$DATABASE_URL" ] && [ -n "$DATABASE_HOST" ]; then
        if [[ "$DATABASE_URL" == *"@postgres:"* ]] && [ "$DATABASE_HOST" != "postgres" ]; then
            log "Patching DATABASE_URL: Replacing generic 'postgres' host with '${DATABASE_HOST}'..."
            export DATABASE_URL="${DATABASE_URL/@postgres:/@$DATABASE_HOST:}"
        fi
    fi

    # Auto-recovery: If DATABASE_URL is set but unreachable, and DATABASE_HOST works, prefer DATABASE_HOST
    # This fixes common issues where Coolify/Platform provides a broken DATABASE_URL but correct component vars
    if [ -n "$DATABASE_URL" ]; then
        # Try a quick check on DATABASE_URL (timeout 2s)
        if ! pg_isready -d "$DATABASE_URL" -t 2 >/dev/null 2>&1; then
            log "DATABASE_URL check failed. Checking fallback to DATABASE_HOST=${DATABASE_HOST}..."
            
            # If DATABASE_URL fails, check if we have a valid DATABASE_HOST component
            if [ -n "$DATABASE_HOST" ] && ! echo "${DATABASE_HOST}" | grep -qE "^postgres(ql)?://"; then
                 export CHECK_DB_PORT="${DATABASE_PORT:-5432}"
                 export CHECK_DB_USER="${DATABASE_USER:-postgres}"
                 export CHECK_DB_PASSWORD="${DATABASE_PASSWORD:-postgres}"
                 
                 # Check connectivity to DATABASE_HOST for logging purposes
                 log "Testing connection to DATABASE_HOST ($DATABASE_HOST)..."
                 # We run this just to print the error to logs (remove -q/silence)
                 PGPASSWORD="${CHECK_DB_PASSWORD}" pg_isready -h "${DATABASE_HOST}" -p "${CHECK_DB_PORT}" -U "${CHECK_DB_USER}" -t 2 || true
                 
                 log "WARNING: DATABASE_URL is unreachable. Forcing switch to DATABASE_HOST configuration to retry connection loop there."
                 unset DATABASE_URL
            fi
        fi
    fi

    if [ -n "$DATABASE_URL" ]; then
        log "Using DATABASE_URL for connection..."
        export DATABASE_URL
        # Wait for database to be ready using URL
        until pg_isready -d "$DATABASE_URL"; do
            log "Database not ready, waiting..."
            # Try to print the actual error message using psql
            psql "$DATABASE_URL" -c "SELECT 1" 2>&1 | head -n 1 || true
            sleep 2
        done
    else
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
