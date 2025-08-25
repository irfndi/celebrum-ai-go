#!/bin/bash

# Deploy Observability Stack to Production Server
# Usage: ./scripts/deploy-observability.sh

set -e

# Configuration
SERVER="root@194.233.73.250"
REMOTE_PATH="/opt/celebrum-ai-go"
LOCAL_PATH="/Users/irfandi/Coding/2025/celebrum-ai-go"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if server is reachable
check_server_connectivity() {
    log_info "Checking server connectivity..."
    if ssh -o ConnectTimeout=10 "$SERVER" "echo 'Server is reachable'" >/dev/null 2>&1; then
        log_success "Server is reachable"
    else
        log_error "Cannot connect to server $SERVER"
        exit 1
    fi
}

# Function to sync files to server
sync_files() {
    log_info "Syncing files to production server..."
    
    # Create remote directory if it doesn't exist
    ssh "$SERVER" "mkdir -p $REMOTE_PATH"
    
    # Rsync with exclusions
    rsync -avz --delete \
        --exclude='.git/' \
        --exclude='node_modules/' \
        --exclude='*.log' \
        --exclude='dump.rdb' \
        --exclude='.env' \
        --exclude='celebrum-ai-go' \
        --exclude='main' \
        --exclude='server' \
        --exclude='ci-artifacts/' \
        --exclude='deployments/' \
        --exclude='backups/' \
        --progress \
        "$LOCAL_PATH/" "$SERVER:$REMOTE_PATH/"
    
    log_success "Files synced successfully"
}

# Function to rebuild Docker containers
rebuild_containers() {
    log_info "Rebuilding Docker containers on production server..."
    
    ssh "$SERVER" "cd $REMOTE_PATH && docker-compose down --remove-orphans"
    ssh "$SERVER" "cd $REMOTE_PATH && docker system prune -f"
    ssh "$SERVER" "cd $REMOTE_PATH && docker-compose build --no-cache"
    
    log_success "Docker containers rebuilt successfully"
}

# Function to deploy observability stack
deploy_observability() {
    log_info "Deploying SigNoz observability stack..."
    
    # Copy environment file
    ssh "$SERVER" "cd $REMOTE_PATH/observability && cp .env.observability .env"
    
    # Create data directories with proper permissions
    ssh "$SERVER" "cd $REMOTE_PATH/observability && mkdir -p data/{clickhouse,signoz,zookeeper,alertmanager,prometheus}"
    ssh "$SERVER" "cd $REMOTE_PATH/observability && chmod -R 755 data/"
    
    # Deploy observability stack
    ssh "$SERVER" "cd $REMOTE_PATH/observability && docker-compose up -d"
    
    log_success "Observability stack deployed"
}

# Function to verify services
verify_services() {
    log_info "Verifying observability services..."
    
    # Wait for services to start
    sleep 30
    
    # Check service status
    ssh "$SERVER" "cd $REMOTE_PATH/observability && docker-compose ps"
    
    # Check if SigNoz frontend is accessible
    if ssh "$SERVER" "curl -f http://localhost:3301/api/v1/health" >/dev/null 2>&1; then
        log_success "SigNoz frontend is accessible"
    else
        log_warning "SigNoz frontend may not be ready yet"
    fi
    
    # Check if OpenTelemetry collector is running
    if ssh "$SERVER" "curl -f http://localhost:4318/v1/traces" >/dev/null 2>&1; then
        log_success "OpenTelemetry collector is accessible"
    else
        log_warning "OpenTelemetry collector may not be ready yet"
    fi
}

# Function to start main application with observability
start_application() {
    log_info "Starting main application with observability..."
    
    # Start main application stack
    ssh "$SERVER" "cd $REMOTE_PATH && docker-compose up -d"
    
    log_success "Main application started with observability enabled"
}

# Function to display access information
display_access_info() {
    log_info "Observability Stack Access Information:"
    echo ""
    echo "SigNoz Dashboard: http://194.233.73.250:3301"
    echo "OpenTelemetry Collector: http://194.233.73.250:4318"
    echo "ClickHouse HTTP: http://194.233.73.250:8123"
    echo "Alertmanager: http://194.233.73.250:9093"
    echo ""
    echo "To view logs:"
    echo "  docker-compose -f observability/docker-compose.yml logs -f"
    echo ""
    echo "To check service status:"
    echo "  docker-compose -f observability/docker-compose.yml ps"
}

# Main deployment process
main() {
    log_info "Starting observability stack deployment to $SERVER"
    echo ""
    
    check_server_connectivity
    sync_files
    rebuild_containers
    deploy_observability
    verify_services
    start_application
    
    echo ""
    log_success "Observability stack deployment completed!"
    echo ""
    display_access_info
}

# Run main function
main "$@"