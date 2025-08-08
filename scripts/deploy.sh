#!/bin/bash
# deploy.sh - Production deployment script for Celebrum AI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ENVIRONMENT=${1:-production}
COMPOSE_FILE="docker-compose.single-droplet.yml"
BACKUP_DIR="/var/backups/celebrum-ai"
LOG_FILE="/var/log/celebrum-ai-deploy.log"

# Functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
    exit 1
}

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
   error "This script must be run as root or with sudo"
fi

# Create necessary directories
mkdir -p "$BACKUP_DIR"
mkdir -p "$(dirname "$LOG_FILE")"

log "Starting deployment for environment: $ENVIRONMENT"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    error "Docker is not running. Please start Docker and try again."
fi

# Check if docker-compose file exists
if [[ ! -f "$COMPOSE_FILE" ]]; then
    error "Docker compose file $COMPOSE_FILE not found"
fi

# Backup current deployment
log "Creating backup of current deployment..."
BACKUP_NAME="backup-$(date +'%Y%m%d-%H%M%S')"
mkdir -p "$BACKUP_DIR/$BACKUP_NAME"

# Backup database if running
if docker-compose -f "$COMPOSE_FILE" ps postgres | grep -q "Up"; then
    log "Backing up database..."
    docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_dump -U postgres celebrum_ai > "$BACKUP_DIR/$BACKUP_NAME/database.sql" || warn "Database backup failed"
fi

# Backup environment file
if [[ -f ".env" ]]; then
    cp .env "$BACKUP_DIR/$BACKUP_NAME/" || warn "Environment file backup failed"
fi

# Pull latest code
log "Pulling latest code from repository..."
if [[ -d ".git" ]]; then
    git fetch origin
    git reset --hard origin/main
    log "Code updated successfully"
else
    warn "Not a git repository. Skipping code update."
fi

# Stop current services
log "Stopping current services..."
docker-compose -f "$COMPOSE_FILE" down || warn "Failed to stop some services"

# Pull latest images
log "Pulling latest Docker images..."
docker-compose -f "$COMPOSE_FILE" pull || warn "Failed to pull some images"

# Build and start services
log "Building and starting services..."
docker-compose -f "$COMPOSE_FILE" up -d --build

# Wait for services to be ready
log "Waiting for services to be ready..."
sleep 30

# Run database migrations
log "Running database migrations..."
if docker-compose -f "$COMPOSE_FILE" ps app | grep -q "Up"; then
    docker-compose -f "$COMPOSE_FILE" exec -T app ./celebrum-ai migrate || warn "Database migration failed"
else
    warn "App service not running, skipping migrations"
fi

# Health check
log "Performing health check..."
HEALTH_CHECK_RETRIES=5
HEALTH_CHECK_DELAY=10

for i in $(seq 1 $HEALTH_CHECK_RETRIES); do
    if curl -f http://localhost/health > /dev/null 2>&1; then
        log "Health check passed!"
        break
    elif [[ $i -eq $HEALTH_CHECK_RETRIES ]]; then
        error "Health check failed after $HEALTH_CHECK_RETRIES attempts. Rolling back..."
        
        # Rollback
        log "Rolling back deployment..."
        docker-compose -f "$COMPOSE_FILE" down
        
        # Restore database if backup exists
        if [[ -f "$BACKUP_DIR/$BACKUP_NAME/database.sql" ]]; then
            log "Restoring database backup..."
            docker-compose -f "$COMPOSE_FILE" up -d postgres
            sleep 10
            docker-compose -f "$COMPOSE_FILE" exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS celebrum_ai;"
            docker-compose -f "$COMPOSE_FILE" exec -T postgres psql -U postgres -c "CREATE DATABASE celebrum_ai;"
            docker-compose -f "$COMPOSE_FILE" exec -T postgres psql -U postgres celebrum_ai < "$BACKUP_DIR/$BACKUP_NAME/database.sql"
        fi
        
        exit 1
    else
        warn "Health check attempt $i failed, retrying in ${HEALTH_CHECK_DELAY}s..."
        sleep $HEALTH_CHECK_DELAY
    fi
done

# Clean up old Docker images
log "Cleaning up old Docker images..."
docker image prune -f || warn "Failed to clean up Docker images"

# Clean up old backups (keep last 5)
log "Cleaning up old backups..."
cd "$BACKUP_DIR"
ls -t | tail -n +6 | xargs -r rm -rf

# Show service status
log "Deployment completed successfully!"
log "Service status:"
docker-compose -f "$COMPOSE_FILE" ps

log "Deployment logs saved to: $LOG_FILE"
log "Backup created at: $BACKUP_DIR/$BACKUP_NAME"

# Send notification (optional)
if command -v curl > /dev/null 2>&1 && [[ -n "$WEBHOOK_URL" ]]; then
    curl -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\":\"🚀 Celebrum AI deployment completed successfully on $(hostname) at $(date)\"}" \
        > /dev/null 2>&1 || warn "Failed to send notification"
fi

log "🎉 Deployment completed successfully!"