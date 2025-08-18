#!/bin/bash

# Optimized Database Migration Script for Celebrum AI
# Integrates with deployment pipeline and provides enhanced functionality
# Supports transaction safety, rollback, and comprehensive validation

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="${PROJECT_ROOT}/migrations"
LOG_FILE="${PROJECT_ROOT}/logs/migration.log"
BACKUP_DIR="${PROJECT_ROOT}/backups/migrations"
MAX_WAIT_TIME=300
CONNECTION_TIMEOUT=30
MIGRATION_TIMEOUT=600
MAX_RETRY_ATTEMPTS=3
RETRY_DELAY=5

# Database configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-celebrum_ai}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_CONTAINER="${DB_CONTAINER:-postgres}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Logging functions
log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[MIGRATE]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_info() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${MAGENTA}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

# Initialize logging and directories
init_environment() {
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$MIGRATIONS_DIR"
    
    log "Migration script initialized at $(date)"
    log "Migrations directory: $MIGRATIONS_DIR"
    log "Backup directory: $BACKUP_DIR"
    log "Database: $DB_HOST:$DB_PORT/$DB_NAME"
}

# Check if required tools are available
check_prerequisites() {
    local missing_tools=()
    
    # Check for required commands
    for tool in docker docker-compose psql pg_dump pg_restore; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and try again"
        return 1
    fi
    
    # Check if migrations directory exists
    if [ ! -d "$MIGRATIONS_DIR" ]; then
        log_warn "Migrations directory does not exist, creating: $MIGRATIONS_DIR"
        mkdir -p "$MIGRATIONS_DIR"
    fi
    
    return 0
}

# Wait for database to be ready
wait_for_database() {
    local max_attempts=$((MAX_WAIT_TIME / 5))
    local attempt=1
    
    log "Waiting for database to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if check_database_connection; then
            log_success "Database is ready"
            return 0
        fi
        
        log "Database not ready, waiting... (attempt $attempt/$max_attempts)"
        sleep 5
        ((attempt++))
    done
    
    log_error "Database failed to become ready within $MAX_WAIT_TIME seconds"
    return 1
}

# Check database connection
check_database_connection() {
    local connection_string="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
    
    if [ -n "$DB_CONTAINER" ] && docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
        # Use docker exec if container is available
        if docker-compose exec -T "$DB_CONTAINER" pg_isready -h localhost -p 5432 -d "$DB_NAME" -U "$DB_USER" &> /dev/null; then
            return 0
        fi
    else
        # Use direct connection
        if timeout $CONNECTION_TIMEOUT psql "$connection_string" -c "SELECT 1;" &> /dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# Execute SQL command
execute_sql() {
    local sql="$1"
    local database="${2:-$DB_NAME}"
    local connection_string="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$database"
    
    if [ -n "$DB_CONTAINER" ] && docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
        # Use docker exec if container is available
        echo "$sql" | docker-compose exec -T "$DB_CONTAINER" psql -U "$DB_USER" -d "$database" -v ON_ERROR_STOP=1
    else
        # Use direct connection
        echo "$sql" | timeout $MIGRATION_TIMEOUT psql "$connection_string" -v ON_ERROR_STOP=1
    fi
}

# Execute SQL file
execute_sql_file() {
    local file="$1"
    local database="${2:-$DB_NAME}"
    local connection_string="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$database"
    
    if [ ! -f "$file" ]; then
        log_error "SQL file not found: $file"
        return 1
    fi
    
    if [ -n "$DB_CONTAINER" ] && docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
        # Use docker exec if container is available
        docker-compose exec -T "$DB_CONTAINER" psql -U "$DB_USER" -d "$database" -v ON_ERROR_STOP=1 -f "/docker-entrypoint-initdb.d/$(basename "$file")"
    else
        # Use direct connection
        timeout $MIGRATION_TIMEOUT psql "$connection_string" -v ON_ERROR_STOP=1 -f "$file"
    fi
}

# Create migrations table
create_migrations_table() {
    log "Creating migrations table if not exists..."
    
    local sql="
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL UNIQUE,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    rollback_sql TEXT
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_filename ON schema_migrations(filename);
CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);
"
    
    if execute_sql "$sql"; then
        log_success "Migrations table ready"
        return 0
    else
        log_error "Failed to create migrations table"
        return 1
    fi
}

# Get applied migrations
get_applied_migrations() {
    execute_sql "SELECT filename FROM schema_migrations ORDER BY applied_at;" | grep -v "filename" | grep -v "^-" | grep -v "^(" | grep -v "^$" || true
}

# Calculate file checksum
calculate_checksum() {
    local file="$1"
    
    if [ -f "$file" ]; then
        shasum -a 256 "$file" | cut -d' ' -f1
    else
        echo ""
    fi
}

# Validate migration file
validate_migration_file() {
    local file="$1"
    local filename=$(basename "$file")
    
    # Check if file exists and is readable
    if [ ! -f "$file" ]; then
        log_error "Migration file not found: $file"
        return 1
    fi
    
    if [ ! -r "$file" ]; then
        log_error "Migration file not readable: $file"
        return 1
    fi
    
    # Check if file is not empty
    if [ ! -s "$file" ]; then
        log_error "Migration file is empty: $file"
        return 1
    fi
    
    # Check filename format (should be timestamp_description.sql)
    if [[ ! "$filename" =~ ^[0-9]{14}_.*\.sql$ ]]; then
        log_warn "Migration file doesn't follow naming convention (YYYYMMDDHHMMSS_description.sql): $filename"
    fi
    
    # Check for dangerous SQL patterns
    local dangerous_patterns=("DROP DATABASE" "DROP SCHEMA" "TRUNCATE" "DELETE FROM.*WHERE.*1.*=.*1")
    
    for pattern in "${dangerous_patterns[@]}"; do
        if grep -qi "$pattern" "$file"; then
            log_error "Dangerous SQL pattern detected in $filename: $pattern"
            log_error "Please review the migration file carefully"
            return 1
        fi
    done
    
    log_info "Migration file validated: $filename"
    return 0
}

# Create database backup
create_backup() {
    local backup_name="backup_$(date +%Y%m%d_%H%M%S).sql"
    local backup_path="$BACKUP_DIR/$backup_name"
    local connection_string="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
    
    log "Creating database backup: $backup_name"
    
    if [ -n "$DB_CONTAINER" ] && docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
        # Use docker exec if container is available
        if docker-compose exec -T "$DB_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --no-owner --no-privileges > "$backup_path"; then
            log_success "Backup created: $backup_path"
            echo "$backup_path"
            return 0
        fi
    else
        # Use direct connection
        if pg_dump "$connection_string" --no-owner --no-privileges > "$backup_path"; then
            log_success "Backup created: $backup_path"
            echo "$backup_path"
            return 0
        fi
    fi
    
    log_error "Failed to create backup"
    return 1
}

# Restore from backup
restore_backup() {
    local backup_file="$1"
    local connection_string="postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    log "Restoring from backup: $backup_file"
    
    # Drop and recreate database
    execute_sql "DROP DATABASE IF EXISTS ${DB_NAME}_temp;" "postgres"
    execute_sql "CREATE DATABASE ${DB_NAME}_temp;" "postgres"
    
    if [ -n "$DB_CONTAINER" ] && docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
        # Use docker exec if container is available
        if docker-compose exec -T "$DB_CONTAINER" psql -U "$DB_USER" -d "${DB_NAME}_temp" < "$backup_file"; then
            # Swap databases
            execute_sql "ALTER DATABASE $DB_NAME RENAME TO ${DB_NAME}_old;" "postgres"
            execute_sql "ALTER DATABASE ${DB_NAME}_temp RENAME TO $DB_NAME;" "postgres"
            execute_sql "DROP DATABASE ${DB_NAME}_old;" "postgres"
            
            log_success "Database restored from backup"
            return 0
        fi
    else
        # Use direct connection
        if psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/${DB_NAME}_temp" < "$backup_file"; then
            # Swap databases
            execute_sql "ALTER DATABASE $DB_NAME RENAME TO ${DB_NAME}_old;" "postgres"
            execute_sql "ALTER DATABASE ${DB_NAME}_temp RENAME TO $DB_NAME;" "postgres"
            execute_sql "DROP DATABASE ${DB_NAME}_old;" "postgres"
            
            log_success "Database restored from backup"
            return 0
        fi
    fi
    
    log_error "Failed to restore from backup"
    return 1
}

# Apply single migration
apply_migration() {
    local file="$1"
    local filename=$(basename "$file")
    local checksum=$(calculate_checksum "$file")
    local start_time=$(date +%s%3N)
    
    log "Applying migration: $filename"
    
    # Check if migration was already applied
    local applied_checksum=$(execute_sql "SELECT checksum FROM schema_migrations WHERE filename = '$filename';" | grep -v "checksum" | grep -v "^-" | grep -v "^(" | grep -v "^$" | head -1 || true)
    
    if [ -n "$applied_checksum" ]; then
        if [ "$applied_checksum" = "$checksum" ]; then
            log_info "Migration already applied: $filename"
            return 0
        else
            log_error "Migration checksum mismatch for $filename"
            log_error "Expected: $checksum"
            log_error "Found: $applied_checksum"
            return 1
        fi
    fi
    
    # Escape variables to prevent SQL injection
    local escaped_filename=$(printf '%s\n' "$filename" | sed "s/'/'''/g")
    local escaped_checksum=$(printf '%s\n' "$checksum" | sed "s/'/'''/g")
    
    # Start transaction and apply migration
    local transaction_sql="
BEGIN;

-- Apply migration
\\i $file

-- Record migration
INSERT INTO schema_migrations (filename, checksum, execution_time_ms) 
VALUES ('$escaped_filename', '$escaped_checksum', 0);

COMMIT;
"
    
    if execute_sql "$transaction_sql"; then
        local end_time=$(date +%s%3N)
        local execution_time=$((end_time - start_time))
        
        # Update execution time with escaped filename
        execute_sql "UPDATE schema_migrations SET execution_time_ms = $execution_time WHERE filename = '$escaped_filename';"
        
        log_success "Migration applied successfully: $filename (${execution_time}ms)"
        return 0
    else
        log_error "Failed to apply migration: $filename"
        return 1
    fi
}

# Run all pending migrations
run_migrations() {
    local force="${1:-false}"
    local backup_created=false
    local backup_file=""
    
    log "Starting migration process..."
    
    # Check prerequisites
    if ! check_prerequisites; then
        return 1
    fi
    
    # Wait for database
    if ! wait_for_database; then
        return 1
    fi
    
    # Create migrations table
    if ! create_migrations_table; then
        return 1
    fi
    
    # Get list of migration files
    local migration_files=()
    if [ -d "$MIGRATIONS_DIR" ]; then
        while IFS= read -r -d '' file; do
            migration_files+=("$file")
        done < <(find "$MIGRATIONS_DIR" -name "*.sql" -type f -print0 | sort -z)
    fi
    
    if [ ${#migration_files[@]} -eq 0 ]; then
        log_info "No migration files found"
        return 0
    fi
    
    # Get applied migrations
    local applied_migrations=()
    while IFS= read -r migration; do
        if [ -n "$migration" ]; then
            applied_migrations+=("$migration")
        fi
    done <<< "$(get_applied_migrations)"
    
    # Find pending migrations
    local pending_migrations=()
    for file in "${migration_files[@]}"; do
        local filename=$(basename "$file")
        local is_applied=false
        
        for applied in "${applied_migrations[@]}"; do
            if [ "$filename" = "$applied" ]; then
                is_applied=true
                break
            fi
        done
        
        if [ "$is_applied" = false ]; then
            pending_migrations+=("$file")
        fi
    done
    
    if [ ${#pending_migrations[@]} -eq 0 ]; then
        log_success "All migrations are up to date"
        return 0
    fi
    
    log "Found ${#pending_migrations[@]} pending migrations"
    
    # Create backup before applying migrations
    if [ "$force" != "true" ]; then
        backup_file=$(create_backup)
        if [ $? -eq 0 ]; then
            backup_created=true
        else
            log_error "Failed to create backup, aborting migrations"
            return 1
        fi
    fi
    
    # Apply pending migrations
    local failed_migrations=()
    for file in "${pending_migrations[@]}"; do
        if ! validate_migration_file "$file"; then
            failed_migrations+=("$file")
            continue
        fi
        
        if ! apply_migration "$file"; then
            failed_migrations+=("$file")
            break  # Stop on first failure
        fi
    done
    
    # Report results
    if [ ${#failed_migrations[@]} -eq 0 ]; then
        log_success "All migrations applied successfully"
        
        # Clean up old backups (keep last 10)
        find "$BACKUP_DIR" -name "backup_*.sql" -type f | sort -r | tail -n +11 | xargs rm -f 2>/dev/null || true
        
        return 0
    else
        log_error "Failed to apply ${#failed_migrations[@]} migrations"
        
        if [ "$backup_created" = true ] && [ -n "$backup_file" ]; then
            log_warn "Backup available for rollback: $backup_file"
            log_warn "To rollback, run: $0 rollback $backup_file"
        fi
        
        return 1
    fi
}

# Show migration status
show_status() {
    log "Checking migration status..."
    
    # Check prerequisites
    if ! check_prerequisites; then
        return 1
    fi
    
    # Check database connection
    if ! check_database_connection; then
        log_error "Cannot connect to database"
        return 1
    fi
    
    # Create migrations table if needed
    create_migrations_table
    
    echo "\n=== Migration Status ==="
    echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
    echo "Migrations directory: $MIGRATIONS_DIR"
    echo "\n=== Applied Migrations ==="
    
    local applied_migrations=$(execute_sql "SELECT filename, applied_at, execution_time_ms FROM schema_migrations ORDER BY applied_at;" | grep -v "filename" | grep -v "^-" | grep -v "^(" | grep -v "^$" || true)
    
    if [ -n "$applied_migrations" ]; then
        echo "$applied_migrations"
    else
        echo "No migrations applied yet"
    fi
    
    echo "\n=== Pending Migrations ==="
    
    # Get list of migration files
    local migration_files=()
    if [ -d "$MIGRATIONS_DIR" ]; then
        while IFS= read -r -d '' file; do
            migration_files+=("$file")
        done < <(find "$MIGRATIONS_DIR" -name "*.sql" -type f -print0 | sort -z)
    fi
    
    # Get applied migration filenames
    local applied_filenames=()
    while IFS= read -r migration; do
        if [ -n "$migration" ]; then
            applied_filenames+=("$migration")
        fi
    done <<< "$(get_applied_migrations)"
    
    # Find pending migrations
    local pending_count=0
    for file in "${migration_files[@]}"; do
        local filename=$(basename "$file")
        local is_applied=false
        
        for applied in "${applied_filenames[@]}"; do
            if [ "$filename" = "$applied" ]; then
                is_applied=true
                break
            fi
        done
        
        if [ "$is_applied" = false ]; then
            echo "$filename"
            ((pending_count++))
        fi
    done
    
    if [ $pending_count -eq 0 ]; then
        echo "No pending migrations"
    fi
    
    echo "\n=== Summary ==="
    echo "Total migration files: ${#migration_files[@]}"
    echo "Applied migrations: ${#applied_filenames[@]}"
    echo "Pending migrations: $pending_count"
}

# Verify database schema
verify_schema() {
    log "Verifying database schema..."
    
    # Check prerequisites
    if ! check_prerequisites; then
        return 1
    fi
    
    # Check database connection
    if ! check_database_connection; then
        log_error "Cannot connect to database"
        return 1
    fi
    
    # Check critical tables
    local critical_tables=("trading_pairs" "exchanges" "market_data" "schema_migrations")
    local missing_tables=()
    
    for table in "${critical_tables[@]}"; do
        local exists=$(execute_sql "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = :'table_name')" -v table_name="$table" | grep -v "exists" | grep -v "^-" | grep -v "^(" | grep -v "^$" | head -1 || echo "f")
        
        if [ "$exists" != "t" ]; then
            missing_tables+=("$table")
        fi
    done
    
    if [ ${#missing_tables[@]} -gt 0 ]; then
        log_error "Missing critical tables: ${missing_tables[*]}"
        return 1
    fi
    
    # Test problematic query
    local test_query="SELECT tp.symbol, tp.base_asset, tp.quote_asset, e.name as exchange_name FROM trading_pairs tp JOIN exchanges e ON tp.exchange_id = e.id LIMIT 1;"
    
    if execute_sql "$test_query" > /dev/null 2>&1; then
        log_success "Database schema verification passed"
        return 0
    else
        log_error "Database schema verification failed"
        return 1
    fi
}

# Rollback to backup
rollback() {
    local backup_file="$1"
    
    if [ -z "$backup_file" ]; then
        log_error "Backup file not specified"
        return 1
    fi
    
    log "Rolling back to backup: $backup_file"
    
    if restore_backup "$backup_file"; then
        log_success "Rollback completed successfully"
        return 0
    else
        log_error "Rollback failed"
        return 1
    fi
}

# Create new migration file
create_migration() {
    local description="$1"
    
    if [ -z "$description" ]; then
        log_error "Migration description not provided"
        return 1
    fi
    
    # Generate filename
    local timestamp=$(date +%Y%m%d%H%M%S)
    local filename="${timestamp}_${description}.sql"
    local filepath="$MIGRATIONS_DIR/$filename"
    
    # Create migration file
    cat > "$filepath" << EOF
-- Migration: $description
-- Created: $(date)
-- Author: $(whoami)

-- Add your SQL statements here
-- Example:
-- CREATE TABLE example (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- Remember to:
-- 1. Test your migration on a copy of production data
-- 2. Consider the impact on running applications
-- 3. Add appropriate indexes for performance
-- 4. Include rollback instructions in comments if needed
EOF
    
    log_success "Migration file created: $filepath"
    log "Please edit the file to add your SQL statements"
    
    return 0
}

# Main function
main() {
    local command="${1:-status}"
    local arg1="${2:-}"
    local arg2="${3:-}"
    
    # Initialize environment
    init_environment
    
    # Change to project root
    cd "$PROJECT_ROOT"
    
    case "$command" in
        "migrate")
            run_migrations "$arg1"
            ;;
        "status")
            show_status
            ;;
        "verify")
            verify_schema
            ;;
        "rollback")
            rollback "$arg1"
            ;;
        "backup")
            create_backup
            ;;
        "create")
            create_migration "$arg1"
            ;;
        "force-migrate")
            run_migrations "true"
            ;;
        "help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show help
show_help() {
    cat << EOF
Optimized Database Migration Script for Celebrum AI

Usage: $0 [COMMAND] [ARGS...]

Commands:
  migrate         - Run all pending migrations (default: with backup)
  force-migrate   - Run migrations without creating backup
  status          - Show migration status
  verify          - Verify database schema
  rollback <file> - Rollback to specified backup file
  backup          - Create database backup
  create <desc>   - Create new migration file
  help            - Show this help message

Examples:
  $0 status
  $0 migrate
  $0 verify
  $0 create "add_user_preferences_table"
  $0 rollback /path/to/backup.sql

Environment Variables:
  DB_HOST         - Database host (default: localhost)
  DB_PORT         - Database port (default: 5432)
  DB_NAME         - Database name (default: celebrum_ai)
  DB_USER         - Database user (default: postgres)
  DB_PASSWORD     - Database password
  DB_CONTAINER    - Docker container name (default: postgres)

Configuration:
  MAX_WAIT_TIME        - Maximum wait time for DB (default: 300s)
  CONNECTION_TIMEOUT   - Connection timeout (default: 30s)
  MIGRATION_TIMEOUT    - Migration timeout (default: 600s)
  MAX_RETRY_ATTEMPTS   - Maximum retry attempts (default: 3)
EOF
}

# Execute main function with all arguments
main "$@"