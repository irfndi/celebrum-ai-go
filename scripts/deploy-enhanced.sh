#!/bin/bash

# Enhanced deployment script for Celebrum AI
# Supports both automated CI/CD and manual rsync deployment
# Provides zero-downtime deployment with rollback capability

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="celebrum-ai-go"
SERVER_USER="root"
SERVER_IP="143.198.219.213"
SERVER_PATH="/root/${PROJECT_NAME}"
BACKUP_PATH="/root/backups/${PROJECT_NAME}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BRANCH="main"

# Parse command line arguments
DEPLOYMENT_TYPE="production"
MANUAL_DEPLOY=false
ROLLBACK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --staging)
            DEPLOYMENT_TYPE="staging"
            shift
            ;;
        --manual)
            MANUAL_DEPLOY=true
            shift
            ;;
        --rollback)
            ROLLBACK=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --staging    Deploy to staging environment"
            echo "  --manual     Use manual rsync deployment instead of CI/CD"
            echo "  --rollback   Rollback to previous deployment"
            echo "  --help       Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if running as root for manual deployment
    if [[ $MANUAL_DEPLOY == true ]] && [[ $EUID -ne 0 ]]; then
        warning "Manual deployment may require root privileges for system operations"
    fi
    
    # Check required tools
    for tool in ssh rsync docker docker-compose curl; do
        if ! command -v $tool &> /dev/null; then
            error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Test SSH connection
    if ! ssh -o ConnectTimeout=5 ${SERVER_USER}@${SERVER_IP} "echo 'SSH connection successful'" &> /dev/null; then
        error "Cannot connect to server via SSH"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Create backup
create_backup() {
    log "Creating backup..."
    
    ssh ${SERVER_USER}@${SERVER_IP} "
        mkdir -p ${BACKUP_PATH}
        if [ -d ${SERVER_PATH} ]; then
            cp -r ${SERVER_PATH} ${BACKUP_PATH}/${TIMESTAMP}
            echo "Backup created: ${BACKUP_PATH}/${TIMESTAMP}"
        else
            echo "No existing deployment found, skipping backup"
        fi
    "
    
    success "Backup created"
}

# Manual rsync deployment
manual_deploy() {
    log "Starting manual rsync deployment..."
    
    # Build locally first
    log "Building project locally..."
    make build
    
    # Sync files to server
    log "Syncing files to server..."
    rsync -avz --exclude='.git' --exclude='node_modules' --exclude='tmp' \
          --exclude='*.log' --exclude='.env.local' \
          ./ ${SERVER_USER}@${SERVER_IP}:${SERVER_PATH}/
    
    # Restart services
    log "Restarting services..."
    ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}
        source .env
        docker-compose -f docker-compose.prod.yml down
        docker-compose -f docker-compose.prod.yml up -d --build
    "
    
    success "Manual deployment completed"
}

# Zero-downtime deployment
zero_downtime_deploy() {
    log "Starting zero-downtime deployment..."
    
    # Build and test locally
    log "Running local build and tests..."
    make ci-test
    make ci-build
    
    # Create deployment directory
    ssh ${SERVER_USER}@${SERVER_IP} "
        mkdir -p ${SERVER_PATH}/deployments/${TIMESTAMP}
    "
    
    # Sync new deployment
    log "Syncing new deployment..."
    rsync -avz --exclude='.git' --exclude='node_modules' --exclude='tmp' \
          --exclude='*.log' --exclude='.env.local' \
          ./ ${SERVER_USER}@${SERVER_IP}:${SERVER_PATH}/deployments/${TIMESTAMP}/
    
    # Switch to new deployment
    log "Switching to new deployment..."
    ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}
        ln -sfn deployments/${TIMESTAMP} current
        cd current
        source .env
        
        # Fix Redis system-level memory setting on server
        echo '🔧 Applying Redis system-level optimizations...'
        bash "$(dirname "$0")/fix-redis-sysctl.sh"
        
        docker-compose -f docker-compose.prod.yml up -d --build
    "
    
    # Verify deployment
    verify_deployment
    
    success "Zero-downtime deployment completed"
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment..."
    
    # Wait for services to start
    sleep 10
    
    # Check health endpoints
    for endpoint in health ready live; do
        if curl -f -s "http://${SERVER_IP}/${endpoint}" > /dev/null; then
            success "Health check ${endpoint} passed"
        else
            error "Health check ${endpoint} failed"
            return 1
        fi
    done
    
    # Check service logs
    ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}/current
        docker-compose -f docker-compose.prod.yml logs --tail=50 | grep -i error || echo 'No errors found'
    "
    
    success "Deployment verification passed"
}

# Rollback deployment
rollback_deployment() {
    log "Starting rollback..."
    
    # Find previous deployment
    PREVIOUS_DEPLOYMENT=$(ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}/deployments
        ls -t | head -2 | tail -1
    ")
    
    if [[ -z $PREVIOUS_DEPLOYMENT ]]; then
        error "No previous deployment found for rollback"
        exit 1
    fi
    
    # Switch back to previous deployment
    ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}
        ln -sfn deployments/${PREVIOUS_DEPLOYMENT} current
        cd current
        source .env
        docker-compose -f docker-compose.prod.yml up -d --build
    "
    
    # Verify rollback
    verify_deployment
    
    success "Rollback completed"
}

# Cleanup old deployments
cleanup_old_deployments() {
    log "Cleaning up old deployments..."
    
    ssh ${SERVER_USER}@${SERVER_IP} "
        cd ${SERVER_PATH}/deployments
        ls -t | tail -n +6 | xargs rm -rf || true
    "
    
    ssh ${SERVER_USER}@${SERVER_IP} "
        find ${BACKUP_PATH} -type d -mtime +7 -exec rm -rf {} + || true
    "
    
    success "Cleanup completed"
}

# Main deployment logic
main() {
    log "Starting enhanced deployment for ${PROJECT_NAME}..."
    log "Deployment type: ${DEPLOYMENT_TYPE}"
    log "Manual mode: ${MANUAL_DEPLOY}"
    
    check_prerequisites
    
    if [[ $ROLLBACK == true ]]; then
        rollback_deployment
        exit 0
    fi
    
    create_backup
    
    if [[ $MANUAL_DEPLOY == true ]]; then
        manual_deploy
    else
        zero_downtime_deploy
        cleanup_old_deployments
    fi
    
    log "Deployment completed successfully!"
    log "Server: https://${SERVER_IP}"
    log "Health: https://${SERVER_IP}/health"
    log "Ready: https://${SERVER_IP}/ready"
    log "Live: https://${SERVER_IP}/live"
}

# Execute main function
main "$@"