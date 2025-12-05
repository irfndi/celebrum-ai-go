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
        export DATABASE_URL="${DATABASE_HOST}"
    fi

    # Coolify/Docker DNS Fix:
    # Coolify may provide DATABASE_URL with internal Docker hostname that's not resolvable,
    # but DATABASE_HOST contains the correct service name. We need to replace the hostname.
    if [ -n "$DATABASE_URL" ] && [ -n "$DATABASE_HOST" ]; then
        # Check if DATABASE_HOST is just a hostname (not a full URL)
        if ! echo "${DATABASE_HOST}" | grep -qE "^postgres(ql)?://"; then
            # Extract current hostname from DATABASE_URL
            # Handle IPv6 addresses first (enclosed in brackets)
            if echo "$DATABASE_URL" | grep -q '@\['; then
                CURRENT_HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\(\[[^]]*\]\).*|\1|p')
            else
                # Handle regular hostnames - extract up to : or /
                CURRENT_HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\([^:/@]*\)[:/].*|\1|p')
            fi
            
            # If the hostnames differ, patch DATABASE_URL with DATABASE_HOST
            if [ -n "$CURRENT_HOST" ] && [ "$CURRENT_HOST" != "$DATABASE_HOST" ]; then
                log "Patching DATABASE_URL: Replacing host '$CURRENT_HOST' with '${DATABASE_HOST}'..."
                # Escape special regex characters in CURRENT_HOST for sed
                ESCAPED_HOST=$(echo "$CURRENT_HOST" | sed 's/[.[\*^$()+?{|]/\\&/g')
                # Replace hostname - handle both :port and /dbname cases
                DATABASE_URL=$(echo "$DATABASE_URL" | sed "s|@${ESCAPED_HOST}\([:/]\)|@${DATABASE_HOST}\1|")
                export DATABASE_URL
                log "Updated DATABASE_URL host to: ${DATABASE_HOST}"
            fi
        fi
    fi

    # Auto-recovery: If DATABASE_URL is set but unreachable, and DATABASE_HOST works, prefer DATABASE_HOST
    # This fixes common issues where Coolify/Platform provides a broken DATABASE_URL but correct component vars
    if [ -n "$DATABASE_URL" ]; then
        log "Testing DATABASE_URL connectivity..."
        # Try a quick check on DATABASE_URL (timeout 3s)
        if ! pg_isready -d "$DATABASE_URL" -t 3 >/dev/null 2>&1; then
            log "DATABASE_URL check failed. Checking fallback options..."
            
            # If DATABASE_URL fails, check if we have a valid DATABASE_HOST component
            if [ -n "$DATABASE_HOST" ] && ! echo "${DATABASE_HOST}" | grep -qE "^postgres(ql)?://"; then
                 export CHECK_DB_PORT="${DATABASE_PORT:-5432}"
                 export CHECK_DB_USER="${DATABASE_USER:-postgres}"
                 export CHECK_DB_PASSWORD="${DATABASE_PASSWORD:-postgres}"
                 export CHECK_DB_NAME="${DATABASE_DBNAME:-celebrum_ai}"
                 
                 # Check connectivity to DATABASE_HOST for logging purposes
                 log "Testing connection to DATABASE_HOST ($DATABASE_HOST) with database '${CHECK_DB_NAME}'..."
                 # We run this just to print the error to logs (remove -q/silence)
                 PGPASSWORD="${CHECK_DB_PASSWORD}" pg_isready -h "${DATABASE_HOST}" -p "${CHECK_DB_PORT}" -U "${CHECK_DB_USER}" -d "${CHECK_DB_NAME}" -t 3 || true
                 
                 log "WARNING: DATABASE_URL is unreachable. Forcing switch to DATABASE_HOST configuration to retry connection loop there."
                 unset DATABASE_URL
            else
                 # DATABASE_HOST is not set or empty - extract components from DATABASE_URL instead
                 log "DATABASE_HOST not provided. Extracting connection details from DATABASE_URL..."
                 
                 # Extract hostname (handle IPv6)
                 if echo "$DATABASE_URL" | grep -q '@\['; then
                     EXTRACTED_HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\(\[[^]]*\]\).*|\1|p')
                 else
                     EXTRACTED_HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\([^:/@]*\)[:/].*|\1|p')
                 fi
                 
                 # Extract port (defaults to 5432 if not found)
                 # Match :port/ or :port$ patterns
                 EXTRACTED_PORT=$(echo "$DATABASE_URL" | sed -n 's|.*:\([0-9]\+\)\(/\|$\).*|\1|p')
                 EXTRACTED_PORT="${EXTRACTED_PORT:-5432}"
                 
                 # Extract database name (may be empty)
                 # Only extract if there's a / after the port/host
                 if echo "$DATABASE_URL" | grep -q ':[0-9]\+/'; then
                     EXTRACTED_DBNAME=$(echo "$DATABASE_URL" | sed -n 's|.*:[0-9]\+/\([^?]*\).*|\1|p')
                 elif echo "$DATABASE_URL" | grep -q '@[^:@]\+:[0-9]\+$'; then
                     EXTRACTED_DBNAME=""
                 else
                     EXTRACTED_DBNAME=$(echo "$DATABASE_URL" | sed -n 's|.*/\([^/?]\+\)$|\1|p')
                 fi
                 EXTRACTED_DBNAME="${EXTRACTED_DBNAME:-postgres}"
                 
                 # Extract username (required)
                 EXTRACTED_USER=$(echo "$DATABASE_URL" | sed -n 's|.*://\([^:@]*\)[:@].*|\1|p')
                 EXTRACTED_USER="${EXTRACTED_USER:-postgres}"
                 
                 # Extract password (may be empty for trust auth)
                 EXTRACTED_PASSWORD=$(echo "$DATABASE_URL" | sed -n 's|.*://[^:]*:\([^@]*\)@.*|\1|p')
                 EXTRACTED_PASSWORD="${EXTRACTED_PASSWORD:-}"
                 
                 log "Extracted from URL: host=${EXTRACTED_HOST}, port=${EXTRACTED_PORT}, db=${EXTRACTED_DBNAME}"
                 
                 # Override environment variables with extracted values
                 export DATABASE_HOST="${EXTRACTED_HOST}"
                 export DATABASE_PORT="${EXTRACTED_PORT}"
                 export DATABASE_DBNAME="${EXTRACTED_DBNAME}"
                 export DATABASE_USER="${EXTRACTED_USER}"
                 export DATABASE_PASSWORD="${EXTRACTED_PASSWORD}"
                 
                 log "WARNING: DATABASE_URL hostname may not be resolvable. Will use extracted components for connection."
                 unset DATABASE_URL
            fi
        else
            log "DATABASE_URL is reachable."
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
        log "Waiting for database at ${DB_HOST}:${DB_PORT} (database: ${DB_NAME})..."
        until PGPASSWORD="${DB_PASSWORD}" pg_isready -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}"; do
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
