#!/bin/bash

# Celebrum AI - 4-Phase Sequential Startup Orchestrator
# This script manages the sequential startup of Docker services
# and controls when external connections (like Telegram webhooks) are enabled

set -euo pipefail

# Configuration
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
LOG_FILE="/tmp/celebrum-startup.log"
MAX_WAIT_TIME=600  # 10 minutes max wait
HEALTH_CHECK_INTERVAL=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_info() {
    log "${BLUE}[INFO]${NC} $1"
}

log_success() {
    log "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    log "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    log "${RED}[ERROR]${NC} $1"
}

# Check if service is healthy
check_service_health() {
    local service_name="$1"
    local container_name="celebrum-${service_name}"
    
    if docker ps --filter "name=${container_name}" --filter "health=healthy" --format "table {{.Names}}" | grep -q "${container_name}"; then
        return 0
    else
        return 1
    fi
}

# Wait for service to be healthy
wait_for_service() {
    local service_name="$1"
    local max_wait="${2:-$MAX_WAIT_TIME}"
    local waited=0
    
    log_info "Waiting for ${service_name} to be healthy..."
    
    while [ $waited -lt $max_wait ]; do
        if check_service_health "$service_name"; then
            log_success "${service_name} is healthy"
            return 0
        fi
        
        sleep $HEALTH_CHECK_INTERVAL
        waited=$((waited + HEALTH_CHECK_INTERVAL))
        
        if [ $((waited % 60)) -eq 0 ]; then
            log_info "Still waiting for ${service_name}... (${waited}s elapsed)"
        fi
    done
    
    log_error "${service_name} failed to become healthy within ${max_wait} seconds"
    return 1
}

# Check if migration completed successfully
check_migration_completed() {
    if docker ps -a --filter "name=celebrum-migrate" --filter "exited=0" --format "table {{.Names}}" | grep -q "celebrum-migrate"; then
        return 0
    else
        return 1
    fi
}

# Wait for migration to complete
wait_for_migration() {
    local max_wait="${1:-300}"  # 5 minutes for migration
    local waited=0
    
    log_info "Waiting for database migration to complete..."
    
    while [ $waited -lt $max_wait ]; do
        if check_migration_completed; then
            log_success "Database migration completed successfully"
            return 0
        fi
        
        sleep $HEALTH_CHECK_INTERVAL
        waited=$((waited + HEALTH_CHECK_INTERVAL))
        
        if [ $((waited % 30)) -eq 0 ]; then
            log_info "Still waiting for migration... (${waited}s elapsed)"
        fi
    done
    
    log_error "Migration failed to complete within ${max_wait} seconds"
    return 1
}

# Enable external connections
enable_external_connections() {
    log_info "Enabling external connections..."
    
    # Set environment variables to enable external connections
    export EXTERNAL_CONNECTIONS_ENABLED=true
    export TELEGRAM_WEBHOOK_ENABLED=true
    
    # Update running containers with new environment variables
    if docker ps --filter "name=celebrum-app" --format "table {{.Names}}" | grep -q "celebrum-app"; then
        log_info "Updating app container environment for external connections..."
        docker exec celebrum-app sh -c 'echo "EXTERNAL_CONNECTIONS_ENABLED=true" >> /tmp/runtime.env'
        docker exec celebrum-app sh -c 'echo "TELEGRAM_WEBHOOK_ENABLED=true" >> /tmp/runtime.env'
    fi
    
    if docker ps --filter "name=celebrum-nginx" --format "table {{.Names}}" | grep -q "celebrum-nginx"; then
        log_info "Updating nginx container environment for external connections..."
        docker exec celebrum-nginx sh -c 'echo "EXTERNAL_CONNECTIONS_ENABLED=true" >> /tmp/runtime.env'
    fi
    
    log_success "External connections enabled"
}

# Main startup orchestration
main() {
    log_info "Starting Celebrum AI 4-Phase Sequential Startup"
    log_info "Using compose file: $COMPOSE_FILE"
    
    # Ensure external connections are disabled initially
    export EXTERNAL_CONNECTIONS_ENABLED=false
    export TELEGRAM_WEBHOOK_ENABLED=false
    
    # PHASE 1: Start database services
    log_info "=== PHASE 1: Starting Database Services ==="
    
    log_info "Starting PostgreSQL..."
    docker-compose -f "$COMPOSE_FILE" up -d postgres
    wait_for_service "postgres" 180
    
    log_info "Starting Redis..."
    docker-compose -f "$COMPOSE_FILE" up -d redis
    wait_for_service "redis" 120
    
    log_info "Running database migrations..."
    docker-compose -f "$COMPOSE_FILE" up migrate
    wait_for_migration 300
    
    log_success "Phase 1 completed: Database services are ready"
    
    # PHASE 2: Start gateway (Nginx will wait for app services)
    log_info "=== PHASE 2: Preparing Gateway ==="
    log_info "Gateway (Nginx) will start automatically after app services are ready"
    
    # PHASE 3: Start application services for warm-up
    log_info "=== PHASE 3: Starting Application Services (Warm-up) ==="
    
    log_info "Starting CCXT service..."
    docker-compose -f "$COMPOSE_FILE" up -d ccxt-service
    wait_for_service "ccxt" 180
    
    log_info "Starting main application..."
    docker-compose -f "$COMPOSE_FILE" up -d app
    wait_for_service "app" 240
    
    log_info "Starting Nginx gateway..."
    docker-compose -f "$COMPOSE_FILE" up -d nginx
    wait_for_service "nginx" 120
    
    log_success "Phase 3 completed: Application services are warmed up"
    
    # PHASE 4: Enable external connections
    log_info "=== PHASE 4: Enabling External Connections ==="
    
    # Wait a bit more for services to fully stabilize
    log_info "Allowing services to stabilize for 30 seconds..."
    sleep 30
    
    # Final health check
    log_info "Performing final health checks..."
    if check_service_health "postgres" && check_service_health "redis" && \
       check_service_health "ccxt" && check_service_health "app" && \
       check_service_health "nginx"; then
        
        enable_external_connections
        
        log_success "=== ALL PHASES COMPLETED SUCCESSFULLY ==="
        log_success "Celebrum AI is ready to accept external connections"
        log_info "Services status:"
        docker-compose -f "$COMPOSE_FILE" ps
        
    else
        log_error "Some services are not healthy. External connections will remain disabled."
        log_info "Current service status:"
        docker-compose -f "$COMPOSE_FILE" ps
        exit 1
    fi
}

# Cleanup function
cleanup() {
    log_info "Startup orchestrator interrupted. Current status:"
    docker-compose -f "$COMPOSE_FILE" ps
}

# Set trap for cleanup
trap cleanup INT TERM

# Run main function
main "$@"