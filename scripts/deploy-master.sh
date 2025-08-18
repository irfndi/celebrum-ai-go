#!/bin/bash

# Master Deployment Orchestrator for Celebrum AI
# Coordinates all deployment phases with comprehensive error handling and rollback capabilities
# Integrates with startup orchestrator, health monitoring, and migration systems

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${PROJECT_ROOT}/logs/deployment.log"
DEPLOYMENT_DIR="${PROJECT_ROOT}/deployments"
BACKUP_DIR="${PROJECT_ROOT}/backups"
CONFIG_FILE="${PROJECT_ROOT}/.env"
DOCKER_COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yml"
DOCKER_COMPOSE_CI_FILE="${PROJECT_ROOT}/docker-compose.ci.yml"

# Deployment configuration
DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-standard}"
ENVIRONMENT="${ENVIRONMENT:-production}"
BRANCH="${BRANCH:-main}"
TAG="${TAG:-latest}"
HEALTH_CHECK_TIMEOUT=300
ROLLBACK_TIMEOUT=180
MAX_RETRY_ATTEMPTS=3
RETRY_DELAY=10

# Service configuration
SERVICES=("postgres" "redis" "app" "ccxt" "nginx")
CRITICAL_SERVICES=("postgres" "redis" "app")
OPTIONAL_SERVICES=("ccxt" "nginx")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Deployment state tracking
DEPLOYMENT_ID="deploy_$(date +%Y%m%d_%H%M%S)"
DEPLOYMENT_STATE_FILE="${DEPLOYMENT_DIR}/${DEPLOYMENT_ID}/state.json"
CURRENT_PHASE="initialization"
ROLLBACK_POINT=""
BACKUP_CREATED=false
BACKUP_FILE=""

# Logging functions
log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[DEPLOY]${NC} $1" | tee -a "$LOG_FILE"
    update_deployment_state "log" "$1"
}

log_success() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
    update_deployment_state "success" "$1"
}

log_warn() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
    update_deployment_state "warning" "$1"
}

log_error() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
    update_deployment_state "error" "$1"
}

log_info() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${MAGENTA}[INFO]${NC} $1" | tee -a "$LOG_FILE"
    update_deployment_state "info" "$1"
}

log_phase() {
    local phase="$1"
    local message="$2"
    CURRENT_PHASE="$phase"
    echo -e "${CYAN}[PHASE: $phase]${NC} $message" | tee -a "$LOG_FILE"
    update_deployment_state "phase" "$phase: $message"
}

# Initialize deployment environment
init_deployment() {
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$DEPLOYMENT_DIR/$DEPLOYMENT_ID"
    mkdir -p "$BACKUP_DIR"
    
    log "Master deployment orchestrator started"
    log "Deployment ID: $DEPLOYMENT_ID"
    log "Environment: $ENVIRONMENT"
    log "Type: $DEPLOYMENT_TYPE"
    log "Branch/Tag: $BRANCH/$TAG"
    
    # Initialize deployment state
    create_deployment_state
}

# Create deployment state tracking
create_deployment_state() {
    local state_json="{
        \"deployment_id\": \"$DEPLOYMENT_ID\",
        \"environment\": \"$ENVIRONMENT\",
        \"type\": \"$DEPLOYMENT_TYPE\",
        \"branch\": \"$BRANCH\",
        \"tag\": \"$TAG\",
        \"started_at\": \"$(date -Iseconds)\",
        \"current_phase\": \"$CURRENT_PHASE\",
        \"status\": \"in_progress\",
        \"rollback_point\": \"$ROLLBACK_POINT\",
        \"backup_created\": $BACKUP_CREATED,
        \"backup_file\": \"$BACKUP_FILE\",
        \"logs\": []
    }"
    
    echo "$state_json" | jq '.' > "$DEPLOYMENT_STATE_FILE" 2>/dev/null || echo "$state_json" > "$DEPLOYMENT_STATE_FILE"
}

# Update deployment state
update_deployment_state() {
    local type="$1"
    local message="$2"
    local timestamp=$(date -Iseconds)
    
    if [ -f "$DEPLOYMENT_STATE_FILE" ]; then
        local temp_file=$(mktemp)
        
        # Update state using jq if available, otherwise use simple replacement
        if command -v jq &> /dev/null; then
            jq --arg type "$type" --arg message "$message" --arg timestamp "$timestamp" --arg phase "$CURRENT_PHASE" --arg rollback "$ROLLBACK_POINT" --arg backup_created "$BACKUP_CREATED" --arg backup_file "$BACKUP_FILE" '
                .current_phase = $phase |
                .rollback_point = $rollback |
                .backup_created = ($backup_created | test("true")) |
                .backup_file = $backup_file |
                .logs += [{"timestamp": $timestamp, "type": $type, "message": $message}]
            ' "$DEPLOYMENT_STATE_FILE" > "$temp_file" && mv "$temp_file" "$DEPLOYMENT_STATE_FILE"
        fi
    fi
}

# Finalize deployment state
finalize_deployment_state() {
    local status="$1"
    local end_time=$(date -Iseconds)
    
    if [ -f "$DEPLOYMENT_STATE_FILE" ] && command -v jq &> /dev/null; then
        local temp_file=$(mktemp)
        jq --arg status "$status" --arg end_time "$end_time" '
            .status = $status |
            .completed_at = $end_time
        ' "$DEPLOYMENT_STATE_FILE" > "$temp_file" && mv "$temp_file" "$DEPLOYMENT_STATE_FILE"
    fi
}

# Check prerequisites
check_prerequisites() {
    log_phase "prerequisites" "Checking deployment prerequisites"
    
    local missing_tools=()
    local missing_files=()
    
    # Check required tools
    for tool in docker docker-compose git curl jq; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        return 1
    fi
    
    # Check required files
    for file in "$DOCKER_COMPOSE_FILE" "$SCRIPT_DIR/startup-orchestrator.sh" "$SCRIPT_DIR/health-monitor-enhanced.sh" "$SCRIPT_DIR/migrate-optimized.sh"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
        fi
    done
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        log_error "Missing required files: ${missing_files[*]}"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        return 1
    fi
    
    # Check Git repository
    if [ ! -d "$PROJECT_ROOT/.git" ]; then
        log_error "Not a Git repository"
        return 1
    fi
    
    log_success "Prerequisites check passed"
    return 0
}

# Create comprehensive backup
create_comprehensive_backup() {
    log_phase "backup" "Creating comprehensive backup"
    
    local backup_timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_name="backup_${backup_timestamp}"
    local backup_path="$BACKUP_DIR/$backup_name"
    
    mkdir -p "$backup_path"
    
    # Database backup
    log "Creating database backup..."
    if bash "$SCRIPT_DIR/migrate-optimized.sh" backup > "$backup_path/database.sql"; then
        log_success "Database backup created"
    else
        log_error "Failed to create database backup"
        return 1
    fi
    
    # Code backup
    log "Creating code backup..."
    local current_commit=$(git rev-parse HEAD)
    echo "$current_commit" > "$backup_path/commit.txt"
    
    # Configuration backup
    log "Creating configuration backup..."
    if [ -f "$CONFIG_FILE" ]; then
        cp "$CONFIG_FILE" "$backup_path/config.env"
    fi
    
    # Docker state backup
    log "Creating Docker state backup..."
    docker-compose ps > "$backup_path/docker_state.txt" 2>/dev/null || true
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}" > "$backup_path/docker_images.txt" 2>/dev/null || true
    
    # Create backup manifest
    cat > "$backup_path/manifest.json" << EOF
{
    "backup_name": "$backup_name",
    "created_at": "$(date -Iseconds)",
    "environment": "$ENVIRONMENT",
    "commit": "$current_commit",
    "branch": "$(git branch --show-current)",
    "deployment_id": "$DEPLOYMENT_ID",
    "files": {
        "database": "database.sql",
        "commit": "commit.txt",
        "config": "config.env",
        "docker_state": "docker_state.txt",
        "docker_images": "docker_images.txt"
    }
}
EOF
    
    BACKUP_CREATED=true
    BACKUP_FILE="$backup_path"
    ROLLBACK_POINT="$backup_name"
    
    log_success "Comprehensive backup created: $backup_path"
    return 0
}

# Pull latest code
pull_code() {
    log_phase "code_update" "Pulling latest code"
    
    # Stash any local changes
    if ! git diff --quiet; then
        log_warn "Local changes detected, stashing..."
        git stash push -m "Auto-stash before deployment $DEPLOYMENT_ID"
    fi
    
    # Fetch latest changes
    log "Fetching latest changes..."
    if ! git fetch origin; then
        log_error "Failed to fetch from origin"
        return 1
    fi
    
    # Checkout target branch/tag
    if [ "$TAG" != "latest" ]; then
        log "Checking out tag: $TAG"
        if ! git checkout "$TAG"; then
            log_error "Failed to checkout tag: $TAG"
            return 1
        fi
    else
        log "Checking out branch: $BRANCH"
        if ! git checkout "$BRANCH"; then
            log_error "Failed to checkout branch: $BRANCH"
            return 1
        fi
        
        # Pull latest changes
        if ! git pull origin "$BRANCH"; then
            log_error "Failed to pull latest changes"
            return 1
        fi
    fi
    
    local new_commit=$(git rev-parse HEAD)
    log_success "Code updated to commit: $new_commit"
    return 0
}

# Run database migrations
run_migrations() {
    log_phase "migration" "Running database migrations"
    
    # Check if migrations are needed
    if bash "$SCRIPT_DIR/migrate-optimized.sh" status | grep -q "No pending migrations"; then
        log_info "No pending migrations"
        return 0
    fi
    
    # Run migrations
    log "Applying database migrations..."
    if bash "$SCRIPT_DIR/migrate-optimized.sh" migrate; then
        log_success "Database migrations completed"
        return 0
    else
        log_error "Database migrations failed"
        return 1
    fi
}

# Build application
build_application() {
    log_phase "build" "Building application"
    
    case "$DEPLOYMENT_TYPE" in
        "production")
            log "Building for production..."
            if docker-compose build --no-cache; then
                log_success "Production build completed"
            else
                log_error "Production build failed"
                return 1
            fi
            ;;
        "development")
            log "Building for development..."
            if docker-compose -f "$DOCKER_COMPOSE_CI_FILE" build; then
                log_success "Development build completed"
            else
                log_error "Development build failed"
                return 1
            fi
            ;;
        "standard")
            log "Building with standard configuration..."
            if docker-compose build; then
                log_success "Standard build completed"
            else
                log_error "Standard build failed"
                return 1
            fi
            ;;
        *)
            log_error "Unknown deployment type: $DEPLOYMENT_TYPE"
            return 1
            ;;
    esac
    
    return 0
}

# Deploy services
deploy_services() {
    log_phase "deployment" "Deploying services"
    
    case "$DEPLOYMENT_TYPE" in
        "zero-downtime")
            deploy_zero_downtime
            ;;
        "rolling")
            deploy_rolling
            ;;
        "standard")
            deploy_standard
            ;;
        *)
            log_error "Unknown deployment type: $DEPLOYMENT_TYPE"
            return 1
            ;;
    esac
}

# Standard deployment
deploy_standard() {
    log "Performing standard deployment..."
    
    # Stop services gracefully
    log "Stopping services..."
    docker-compose down --timeout 30
    
    # Start services with orchestrator
    log "Starting services with orchestrator..."
    if bash "$SCRIPT_DIR/startup-orchestrator.sh"; then
        log_success "Services started successfully"
        return 0
    else
        log_error "Failed to start services"
        return 1
    fi
}

# Zero-downtime deployment
deploy_zero_downtime() {
    log "Performing zero-downtime deployment..."
    
    # Create new deployment directory
    local new_deployment="${DEPLOYMENT_DIR}/${DEPLOYMENT_ID}_new"
    mkdir -p "$new_deployment"
    
    # Copy current configuration
    cp -r "$PROJECT_ROOT"/* "$new_deployment/" 2>/dev/null || true
    
    # Start new services on different ports
    cd "$new_deployment"
    
    # Modify docker-compose for new ports
    sed 's/8080:8080/8081:8080/g; s/3001:3001/3002:3001/g' "$DOCKER_COMPOSE_FILE" > "docker-compose-new.yml"
    
    # Start new services
    if docker-compose -f docker-compose-new.yml up -d; then
        log "New services started"
        
        # Wait for health check
        if wait_for_health "http://localhost:8081/health"; then
            log "New services are healthy"
            
            # Switch traffic (this would typically involve load balancer configuration)
            log "Switching traffic to new services..."
            
            # Stop old services
            cd "$PROJECT_ROOT"
            docker-compose down
            
            # Update main configuration
            sed 's/8081:8080/8080:8080/g; s/3002:3001/3001:3001/g' "$new_deployment/docker-compose-new.yml" > "$DOCKER_COMPOSE_FILE"
            
            # Start services with correct ports
            if bash "$SCRIPT_DIR/startup-orchestrator.sh"; then
                log_success "Zero-downtime deployment completed"
                
                # Cleanup
                docker-compose -f "$new_deployment/docker-compose-new.yml" down
                rm -rf "$new_deployment"
                
                return 0
            else
                log_error "Failed to start final services"
                return 1
            fi
        else
            log_error "New services failed health check"
            docker-compose -f "$new_deployment/docker-compose-new.yml" down
            return 1
        fi
    else
        log_error "Failed to start new services"
        return 1
    fi
}

# Rolling deployment
deploy_rolling() {
    log "Performing rolling deployment..."
    
    for service in "${SERVICES[@]}"; do
        log "Rolling update for service: $service"
        
        # Update service
        if docker-compose up -d --no-deps "$service"; then
            log "Service $service updated"
            
            # Wait for service to be healthy
            if wait_for_service_health "$service"; then
                log_success "Service $service is healthy"
            else
                log_error "Service $service failed health check"
                return 1
            fi
        else
            log_error "Failed to update service: $service"
            return 1
        fi
        
        # Brief pause between services
        sleep 5
    done
    
    log_success "Rolling deployment completed"
    return 0
}

# Wait for service health
wait_for_service_health() {
    local service="$1"
    local max_attempts=$((HEALTH_CHECK_TIMEOUT / 10))
    local attempt=1
    
    log "Waiting for service $service to be healthy..."
    
    while [ $attempt -le $max_attempts ]; do
        if bash "$SCRIPT_DIR/health-monitor-enhanced.sh" check "$service" &> /dev/null; then
            return 0
        fi
        
        log "Service $service not ready, waiting... (attempt $attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
    
    return 1
}

# Wait for HTTP health endpoint
wait_for_health() {
    local endpoint="$1"
    local max_attempts=$((HEALTH_CHECK_TIMEOUT / 10))
    local attempt=1
    
    log "Waiting for health endpoint: $endpoint"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$endpoint" > /dev/null 2>&1; then
            return 0
        fi
        
        log "Health endpoint not ready, waiting... (attempt $attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
    
    return 1
}

# Verify deployment
verify_deployment() {
    log_phase "verification" "Verifying deployment"
    
    # Health check all services
    log "Checking service health..."
    if bash "$SCRIPT_DIR/health-monitor-enhanced.sh" verify; then
        log_success "All services are healthy"
    else
        log_error "Some services are unhealthy"
        return 1
    fi
    
    # Verify database schema
    log "Verifying database schema..."
    if bash "$SCRIPT_DIR/migrate-optimized.sh" verify; then
        log_success "Database schema verified"
    else
        log_error "Database schema verification failed"
        return 1
    fi
    
    # Test critical endpoints
    log "Testing critical endpoints..."
    local endpoints=("http://localhost:8080/health" "http://localhost:3001/health")
    
    for endpoint in "${endpoints[@]}"; do
        if curl -sf "$endpoint" > /dev/null 2>&1; then
            log_success "Endpoint healthy: $endpoint"
        else
            log_warn "Endpoint not responding: $endpoint"
        fi
    done
    
    log_success "Deployment verification completed"
    return 0
}

# Rollback deployment
rollback_deployment() {
    local rollback_target="${1:-$ROLLBACK_POINT}"
    
    if [ -z "$rollback_target" ]; then
        log_error "No rollback target specified"
        return 1
    fi
    
    log_phase "rollback" "Rolling back deployment to: $rollback_target"
    
    local backup_path="$BACKUP_DIR/$rollback_target"
    
    if [ ! -d "$backup_path" ]; then
        log_error "Backup not found: $backup_path"
        return 1
    fi
    
    # Stop current services
    log "Stopping current services..."
    docker-compose down --timeout 30
    
    # Restore database
    if [ -f "$backup_path/database.sql" ]; then
        log "Restoring database..."
        if bash "$SCRIPT_DIR/migrate-optimized.sh" rollback "$backup_path/database.sql"; then
            log_success "Database restored"
        else
            log_error "Failed to restore database"
            return 1
        fi
    fi
    
    # Restore code
    if [ -f "$backup_path/commit.txt" ]; then
        local target_commit=$(cat "$backup_path/commit.txt")
        log "Restoring code to commit: $target_commit"
        
        if git checkout "$target_commit"; then
            log_success "Code restored"
        else
            log_error "Failed to restore code"
            return 1
        fi
    fi
    
    # Restore configuration
    if [ -f "$backup_path/config.env" ]; then
        log "Restoring configuration..."
        cp "$backup_path/config.env" "$CONFIG_FILE"
        log_success "Configuration restored"
    fi
    
    # Start services
    log "Starting services..."
    if bash "$SCRIPT_DIR/startup-orchestrator.sh"; then
        log_success "Services started after rollback"
        
        # Verify rollback
        if verify_deployment; then
            log_success "Rollback completed successfully"
            return 0
        else
            log_error "Rollback verification failed"
            return 1
        fi
    else
        log_error "Failed to start services after rollback"
        return 1
    fi
}

# Cleanup old deployments
cleanup_old_deployments() {
    log "Cleaning up old deployments..."
    
    # Keep last 10 deployment directories
    find "$DEPLOYMENT_DIR" -maxdepth 1 -type d -name "deploy_*" | sort -r | tail -n +11 | xargs rm -rf 2>/dev/null || true
    
    # Keep last 20 backups
    find "$BACKUP_DIR" -maxdepth 1 -type d -name "backup_*" | sort -r | tail -n +21 | xargs rm -rf 2>/dev/null || true
    
    # Clean up old Docker images
    docker image prune -f &> /dev/null || true
    
    log_success "Cleanup completed"
}

# Show deployment status
show_status() {
    echo "\n=== Celebrum AI Deployment Status ==="
    echo "Timestamp: $(date)"
    
    if [ -f "$DEPLOYMENT_STATE_FILE" ]; then
        echo "\n=== Current Deployment ==="
        if command -v jq &> /dev/null; then
            jq -r '
                "Deployment ID: " + .deployment_id,
                "Environment: " + .environment,
                "Type: " + .type,
                "Status: " + .status,
                "Current Phase: " + .current_phase,
                "Started: " + .started_at,
                (if .completed_at then "Completed: " + .completed_at else empty end),
                "Rollback Point: " + .rollback_point
            ' "$DEPLOYMENT_STATE_FILE"
        else
            cat "$DEPLOYMENT_STATE_FILE"
        fi
    fi
    
    echo "\n=== Service Status ==="
    bash "$SCRIPT_DIR/health-monitor-enhanced.sh" status
    
    echo "\n=== Recent Deployments ==="
    find "$DEPLOYMENT_DIR" -maxdepth 1 -type d -name "deploy_*" | sort -r | head -5 | while read -r dir; do
        local deployment_id=$(basename "$dir")
        echo "$deployment_id"
    done
    
    echo "\n=== Available Backups ==="
    find "$BACKUP_DIR" -maxdepth 1 -type d -name "backup_*" | sort -r | head -5 | while read -r dir; do
        local backup_name=$(basename "$dir")
        echo "$backup_name"
    done
}

# Main deployment function
main() {
    local command="${1:-deploy}"
    local arg1="${2:-}"
    local arg2="${3:-}"
    
    # Initialize deployment
    init_deployment
    
    # Change to project root
    cd "$PROJECT_ROOT"
    
    case "$command" in
        "deploy")
            # Full deployment process
            if check_prerequisites && \
               create_comprehensive_backup && \
               pull_code && \
               run_migrations && \
               build_application && \
               deploy_services && \
               verify_deployment; then
                
                finalize_deployment_state "success"
                cleanup_old_deployments
                log_success "Deployment completed successfully: $DEPLOYMENT_ID"
                
                # Show final status
                show_status
                
                exit 0
            else
                finalize_deployment_state "failed"
                log_error "Deployment failed: $DEPLOYMENT_ID"
                
                if [ "$BACKUP_CREATED" = true ]; then
                    log_warn "Backup available for rollback: $BACKUP_FILE"
                    log_warn "To rollback, run: $0 rollback $ROLLBACK_POINT"
                fi
                
                exit 1
            fi
            ;;
        "rollback")
            rollback_deployment "$arg1"
            ;;
        "status")
            show_status
            ;;
        "verify")
            verify_deployment
            ;;
        "backup")
            create_comprehensive_backup
            ;;
        "cleanup")
            cleanup_old_deployments
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
Master Deployment Orchestrator for Celebrum AI

Usage: $0 [COMMAND] [ARGS...]

Commands:
  deploy          - Run full deployment process (default)
  rollback [name] - Rollback to specified backup
  status          - Show deployment status
  verify          - Verify current deployment
  backup          - Create comprehensive backup
  cleanup         - Clean up old deployments and backups
  help            - Show this help message

Environment Variables:
  DEPLOYMENT_TYPE - Deployment type (standard|zero-downtime|rolling)
  ENVIRONMENT     - Target environment (production|staging|development)
  BRANCH          - Git branch to deploy (default: main)
  TAG             - Git tag to deploy (overrides branch)
  
Deployment Types:
  standard        - Stop all services, then start (default)
  zero-downtime   - Deploy without service interruption
  rolling         - Update services one by one

Examples:
  $0 deploy
  $0 status
  $0 rollback backup_20240115_143022
  DEPLOYMENT_TYPE=zero-downtime $0 deploy
  ENVIRONMENT=staging BRANCH=develop $0 deploy

Phases:
  1. Prerequisites - Check tools and requirements
  2. Backup        - Create comprehensive backup
  3. Code Update   - Pull latest code changes
  4. Migration     - Run database migrations
  5. Build         - Build application containers
  6. Deployment    - Deploy services using selected strategy
  7. Verification  - Verify deployment health and functionality
EOF
}

# Trap signals for cleanup
trap 'log_error "Deployment interrupted"; finalize_deployment_state "interrupted"; exit 1' INT TERM

# Execute main function with all arguments
main "$@"