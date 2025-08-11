#!/bin/bash

# Robust Migration Script for Docker Environment
# Handles database initialization and migration with proper error handling

set -e

# Configuration from environment variables
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-celebrum_ai}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
MIGRATIONS_DIR="/database/migrations"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to log messages
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

# Function to wait for database to be ready
wait_for_db() {
    log "Waiting for database to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" >/dev/null 2>&1; then
            log "Database is ready!"
            return 0
        fi
        
        log "Database not ready yet (attempt $attempt/$max_attempts), waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_error "Database failed to become ready after $max_attempts attempts"
    exit 1
}

# Function to check if migration has been applied
migration_applied() {
    local migration_name="$1"
    
    # Check if migrations table exists
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" 2>/dev/null | grep -q migrations; then
        return 1
    fi
    
    # Check if migration has been applied
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM migrations WHERE filename = '$migration_name' AND applied = true" -t -A 2>/dev/null | grep -q 1
}

# Function to create migrations table if it doesn't exist
create_migrations_table() {
    log "Creating migrations table if it doesn't exist..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        CREATE TABLE IF NOT EXISTS migrations (
            id SERIAL PRIMARY KEY,
            filename VARCHAR(255) UNIQUE NOT NULL,
            applied BOOLEAN DEFAULT false,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )
    "
}

# Function to apply migration
apply_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file")
    
    if migration_applied "$migration_name"; then
        log_warn "Migration $migration_name already applied, skipping"
        return 0
    fi
    
    log "Applying migration: $migration_name"
    
    # Apply the migration
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration_file"; then
        log "Successfully applied migration: $migration_name"
        
        # Record migration in migrations table
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
            INSERT INTO migrations (filename, applied) VALUES ('$migration_name', true)
            ON CONFLICT (filename) DO UPDATE SET applied = true, applied_at = NOW()
        "
    else
        log_error "Failed to apply migration: $migration_name"
        exit 1
    fi
}

# Function to verify schema after migrations
verify_schema() {
    log "Verifying database schema..."
    
    # Check if exchanges table has required columns
    local exchanges_columns=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'exchanges'" -t -A 2>/dev/null | tr '\n' ' ')
    
    if [[ "$exchanges_columns" == *"display_name"* && "$exchanges_columns" == *"ccxt_id"* ]]; then
        log "✓ Exchanges table schema verified"
    else
        log_error "Missing expected columns in exchanges table: $exchanges_columns"
        exit 1
    fi
    
    # Check if market_data table exists
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" 2>/dev/null | grep -q market_data; then
        log "✓ Market data table exists"
    else
        log_error "Market data table not found"
        exit 1
    fi
}

# Function to run all pending migrations
run_all_migrations() {
    log "Starting database migration process..."
    
    # Wait for database to be ready
    wait_for_db
    
    # Create migrations table
    create_migrations_table
    
    # Apply all migrations in order
    if [ -d "$MIGRATIONS_DIR" ]; then
        local migration_count=0
        for migration_file in $(ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sort -V); do
            apply_migration "$migration_file"
            migration_count=$((migration_count + 1))
        done
        
        if [ $migration_count -eq 0 ]; then
            log_warn "No migration files found in $MIGRATIONS_DIR"
        else
            log "Applied $migration_count migrations"
        fi
    else
        log_error "Migrations directory not found: $MIGRATIONS_DIR"
        exit 1
    fi
    
    # Verify schema
    verify_schema
    
    log "All migrations completed. System ready."
}

# Main execution
log "Starting robust migration script..."
log "Database: $DB_HOST:$DB_PORT/$DB_NAME"
log "User: $DB_USER"

run_all_migrations

log "Migration process completed successfully!"