#!/bin/bash

# Celebrum AI Database Migration Script
# Usage: ./migrate.sh [migration_number]
# If no migration_number provided, runs all pending migrations

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-celebrum_ai}"
DB_USER="${DB_USER:-celebrum_ai_user}"
DB_PASSWORD="${DB_PASSWORD:-}"
MIGRATIONS_DIR="$(dirname "$0")/migrations"

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

# Function to check if migration has been applied
migration_applied() {
    local migration_name="$1"
    
    # Check if schema_migrations table exists
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" | grep -q schema_migrations; then
        return 1
    fi
    
    # Check if migration has been applied
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -c "SELECT 1 FROM schema_migrations WHERE filename = \$1 AND applied = true" \
        -v migration_name="$migration_name" -t -A 2>/dev/null | grep -q 1
}

# Function to apply migration
apply_migration() {
    local migration_file="$1"
    local migration_name
    migration_name=$(basename "$migration_file")
    
    if migration_applied "$migration_name"; then
        log_warn "Migration $migration_name already applied, skipping"
        return 0
    fi
    
    log "Applying migration: $migration_name"
    
    # Apply the migration
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration_file"; then
        log "Successfully applied migration: $migration_name"
        
        # Record migration in schema_migrations table
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            -v migration_name="$migration_name" \
            -c "INSERT INTO schema_migrations (filename, applied) VALUES (:'migration_name', true) ON CONFLICT (filename) DO UPDATE SET applied = true, applied_at = NOW()"
    else
        log_error "Failed to apply migration: $migration_name"
        exit 1
    fi
}

# Function to create schema_migrations table if it doesn't exist
create_migrations_table() {
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        CREATE TABLE IF NOT EXISTS schema_migrations (
            id SERIAL PRIMARY KEY,
            filename VARCHAR(255) UNIQUE NOT NULL,
            applied BOOLEAN DEFAULT false,
            applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        )
    "
}

# Function to list available migrations
list_migrations() {
    log "Available migrations:"
    ls -1 "$MIGRATIONS_DIR"/*.sql | sort -V | while read -r file; do
        local filename
        filename=$(basename "$file")
        if migration_applied "$filename"; then
            echo "  ✓ $filename (applied)"
        else
            echo "  ⏳ $filename (pending)"
        fi
    done
}

# Function to show migration status
show_status() {
    log "Migration status:"
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" | grep -q schema_migrations; then
        log_warn "Schema migrations table does not exist"
        list_migrations
        return
    fi
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT filename, applied, applied_at 
        FROM schema_migrations 
        ORDER BY applied_at DESC, filename
    "
}

# Function to run specific migration
run_specific_migration() {
    local migration_number="$1"
    local migration_files=("$MIGRATIONS_DIR"/${migration_number}_*.sql)
    
    if [ ${#migration_files[@]} -eq 0 ] || [ ! -f "${migration_files[0]}" ]; then
        log_error "Migration file not found for number: $migration_number"
        exit 1
    fi
    
    if [ ${#migration_files[@]} -gt 1 ]; then
        log_error "Multiple migration files found for number: $migration_number"
        exit 1
    fi
    
    local migration_file="${migration_files[0]}"
    
    create_migrations_table
    apply_migration "$migration_file"
}

# Function to run all pending migrations
run_all_migrations() {
    log "Running all pending migrations..."
    create_migrations_table
    
    ls -1 "$MIGRATIONS_DIR"/*.sql | sort -V | while read -r file; do
        apply_migration "$file"
    done
    
    log "All migrations completed successfully!"
}

# Function to rollback migration
rollback_migration() {
    local migration_number="$1"
    local migration_files=("$MIGRATIONS_DIR"/${migration_number}_*.sql)
    
    if [ ${#migration_files[@]} -eq 0 ] || [ ! -f "${migration_files[0]}" ]; then
        log_error "Migration file not found for number: $migration_number"
        exit 1
    fi
    
    if [ ${#migration_files[@]} -gt 1 ]; then
        log_error "Multiple migration files found for number: $migration_number"
        exit 1
    fi
    
    local migration_file="${migration_files[0]}"
    local migration_name
    migration_name=$(basename "$migration_file")
    
    log "Rolling back migration: $migration_name"
    
    # This is a simplified rollback - in practice, you'd need rollback scripts
    log_warn "Rollback functionality requires specific rollback scripts"
    log "Consider creating rollback scripts for complex migrations"
    
    # Remove migration record (using parameterized query to prevent SQL injection)
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -c "DELETE FROM schema_migrations WHERE filename = \$1" \
        "$migration_name"
    
    log "Migration record removed: $migration_name"
}

# Main script logic
case "${1:-run}" in
    "status")
        show_status
        ;;
    "list")
        list_migrations
        ;;
    "run")
        run_all_migrations
        ;;
    "rollback")
        if [ -z "$2" ]; then
            log_error "Usage: $0 rollback <migration_number>"
            exit 1
        fi
        rollback_migration "$2"
        ;;
    *)
        if [[ "$1" =~ ^[0-9]+$ ]]; then
            run_specific_migration "$1"
        else
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  status          - Show migration status"
            echo "  list            - List available migrations"
            echo "  run             - Run all pending migrations (default)"
            echo "  rollback <num>  - Rollback specific migration"
            echo "  <number>        - Run specific migration"
            echo ""
            echo "Environment variables:"
            echo "  DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD"
            exit 1
        fi
        ;;
esac