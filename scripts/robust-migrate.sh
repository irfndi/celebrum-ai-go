#!/bin/bash

# Enhanced Robust Migration Script for Docker Environment
# Handles database initialization and migration with comprehensive error handling
# Includes fixes for all issues discovered during local debugging

set -e

# Configuration from environment variables
# SECURITY: All database credentials must be provided via environment variables
# No default passwords are provided for security reasons
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-celebrum_ai}"
DB_USER="${DB_USER:-postgres}"
# SECURITY: DB_PASSWORD must be set via environment variable - no default provided
DB_PASSWORD="${DB_PASSWORD:?Error: DB_PASSWORD environment variable must be set}"

MIGRATIONS_DIR="/database/migrations"
MAX_RETRIES="${MAX_RETRIES:-3}"
RETRY_DELAY="${RETRY_DELAY:-5}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

# Function to execute SQL with retry logic
execute_sql_with_retry() {
    local sql="$1"
    local description="$2"
    local attempt=1
    
    while [ $attempt -le $MAX_RETRIES ]; do
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$sql" >/dev/null 2>&1; then
            return 0
        fi
        
        log_warn "$description failed (attempt $attempt/$MAX_RETRIES)"
        if [ $attempt -lt $MAX_RETRIES ]; then
            sleep $RETRY_DELAY
        fi
        attempt=$((attempt + 1))
    done
    
    log_error "$description failed after $MAX_RETRIES attempts"
    return 1
}

# Function to wait for database to be ready
wait_for_db() {
    log "Waiting for database to be ready..."
    local max_attempts=60
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" >/dev/null 2>&1; then
            log "Database is ready!"
            return 0
        fi
        
        if [ $((attempt % 10)) -eq 0 ]; then
            log "Database not ready yet (attempt $attempt/$max_attempts), still waiting..."
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    
    log_error "Database failed to become ready after $max_attempts attempts"
    exit 1
}

# Function to check database connectivity
check_db_connectivity() {
    log_info "Testing database connectivity..."
    
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" >/dev/null 2>&1; then
        log_error "Cannot connect to database. Please check connection parameters."
        exit 1
    fi
    
    log "✓ Database connectivity verified"
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
    
    local create_table_sql="
        CREATE TABLE IF NOT EXISTS migrations (
            id SERIAL PRIMARY KEY,
            filename VARCHAR(255) UNIQUE NOT NULL,
            applied BOOLEAN DEFAULT false,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )
    "
    
    if execute_sql_with_retry "$create_table_sql" "Creating migrations table"; then
        log "✓ Migrations table ready"
    else
        log_error "Failed to create migrations table"
        exit 1
    fi
}

# Function to calculate file checksum
calculate_checksum() {
    local file="$1"
    sha256sum "$file" | cut -d' ' -f1
}

# Function to validate migration file
validate_migration_file() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file")
    
    # Check if file exists and is readable
    if [ ! -r "$migration_file" ]; then
        log_error "Migration file not readable: $migration_name"
        return 1
    fi
    
    # Check if file is not empty
    if [ ! -s "$migration_file" ]; then
        log_error "Migration file is empty: $migration_name"
        return 1
    fi
    
    # Basic SQL syntax check (look for obvious issues)
    if grep -q "DROP DATABASE\|DROP SCHEMA" "$migration_file"; then
        log_error "Dangerous SQL detected in migration: $migration_name"
        return 1
    fi
    
    return 0
}

# Function to apply migration with transaction safety
apply_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file")
    local start_time=$(date +%s%3N)
    
    if migration_applied "$migration_name"; then
        log_warn "Migration $migration_name already applied, skipping"
        return 0
    fi
    
    # Validate migration file
    if ! validate_migration_file "$migration_file"; then
        return 1
    fi
    
    log "Applying migration: $migration_name"
    
    # Calculate checksum
    local checksum=$(calculate_checksum "$migration_file")
    
    # Apply the migration in a transaction
    local migration_sql="
        BEGIN;
        $(cat "$migration_file")
        COMMIT;
    "
    
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$migration_sql"; then
        local end_time=$(date +%s%3N)
        local execution_time=$((end_time - start_time))
        
        log "Successfully applied migration: $migration_name (${execution_time}ms)"
        
        # Record migration in migrations table
        local record_sql="
            INSERT INTO migrations (filename, applied) 
            VALUES ('$migration_name', true)
            ON CONFLICT (filename) DO UPDATE SET 
                applied = true, 
                applied_at = NOW()
        "
        
        if ! execute_sql_with_retry "$record_sql" "Recording migration $migration_name"; then
            log_error "Failed to record migration: $migration_name"
            return 1
        fi
    else
        log_error "Failed to apply migration: $migration_name"
        return 1
    fi
}

# Function to verify critical schema components
verify_schema() {
    log "Verifying database schema..."
    
    # Check trading_pairs table schema (critical for cache_warming.go fix)
    local trading_pairs_columns=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'trading_pairs'" -t -A 2>/dev/null | tr '\n' ' ')
    
    if [[ "$trading_pairs_columns" == *"base_currency"* && "$trading_pairs_columns" == *"quote_currency"* && "$trading_pairs_columns" == *"exchange_id"* ]]; then
        log "✓ Trading pairs table schema verified (base_currency, quote_currency, exchange_id present)"
    else
        log_error "Missing expected columns in trading_pairs table. Found: $trading_pairs_columns"
        exit 1
    fi
    
    # Check exchanges table
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
    
    # Test the problematic query that was fixed in cache_warming.go
    local test_query="SELECT id, exchange_id, symbol, base_currency, quote_currency FROM trading_pairs LIMIT 1"
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$test_query" >/dev/null 2>&1; then
        log "✓ Cache warming query compatibility verified"
    else
        log_error "Cache warming query test failed - schema mismatch detected"
        exit 1
    fi
}

# Function to run all pending migrations
run_all_migrations() {
    log "Starting enhanced database migration process..."
    log_info "Configuration: $DB_HOST:$DB_PORT/$DB_NAME (user: $DB_USER)"
    
    # Wait for database to be ready
    wait_for_db
    
    # Check connectivity
    check_db_connectivity
    
    # Create migrations table
    create_migrations_table
    
    # Apply all migrations in order
    if [ -d "$MIGRATIONS_DIR" ]; then
        local migration_count=0
        local applied_count=0
        local skipped_count=0
        
        for migration_file in $(ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sort -V); do
            migration_count=$((migration_count + 1))
            
            if apply_migration "$migration_file"; then
                if migration_applied "$(basename "$migration_file")"; then
                    applied_count=$((applied_count + 1))
                else
                    skipped_count=$((skipped_count + 1))
                fi
            else
                log_error "Migration failed: $(basename "$migration_file")"
                exit 1
            fi
        done
        
        if [ $migration_count -eq 0 ]; then
            log_warn "No migration files found in $MIGRATIONS_DIR"
        else
            log "Migration summary: $migration_count total, $applied_count applied, $skipped_count skipped"
        fi
    else
        log_error "Migrations directory not found: $MIGRATIONS_DIR"
        exit 1
    fi
    
    # Verify schema
    verify_schema
    
    log "All migrations completed successfully. System ready."
}

# Function to show migration status
show_migration_status() {
    log_info "Current migration status:"
    
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" 2>/dev/null | grep -q migrations; then
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT filename, applied, applied_at, execution_time_ms FROM migrations ORDER BY applied_at;"
    else
        log_info "No migrations table found - database not initialized"
    fi
}

# Main execution
case "${1:-migrate}" in
    "migrate")
        log "Starting robust migration script..."
        run_all_migrations
        log "Migration process completed successfully!"
        ;;
    "status")
        show_migration_status
        ;;
    "verify")
        wait_for_db
        check_db_connectivity
        verify_schema
        log "Schema verification completed successfully!"
        ;;
    *)
        echo "Usage: $0 [migrate|status|verify]"
        echo "  migrate: Run all pending migrations (default)"
        echo "  status:  Show current migration status"
        echo "  verify:  Verify database schema without running migrations"
        exit 1
        ;;
esac