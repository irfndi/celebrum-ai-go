#!/bin/bash

# Local Binary Deployment Script for Celebrum AI
# This script deploys pre-built binaries locally on the server

set -euo pipefail

# Configuration
ENVIRONMENT="${1:-staging}"
PROJECT_ROOT="/opt/celebrum-ai"
BACKUP_DIR="/opt/celebrum-ai/backups"
SERVICE_USER="celebrum"
LOG_FILE="/var/log/celebrum-deploy.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" | tee -a "$LOG_FILE"
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1" | tee -a "$LOG_FILE"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   error "This script must be run as root"
   exit 1
fi

# Validate environment
case "$ENVIRONMENT" in
    "staging"|"production"|"test")
        log "Deploying to $ENVIRONMENT environment"
        ;;
    *)
        error "Invalid environment: $ENVIRONMENT. Use staging, production, or test"
        exit 1
        ;;
esac

# Create deployment directory structure
log "Creating deployment directory structure..."
mkdir -p "$PROJECT_ROOT/bin"
mkdir -p "$PROJECT_ROOT/configs"
mkdir -p "$PROJECT_ROOT/logs"
mkdir -p "$BACKUP_DIR"

# Setup service user if not exists
if ! id "$SERVICE_USER" &>/dev/null; then
    log "Creating service user: $SERVICE_USER"
    useradd -r -s /bin/false -d "$PROJECT_ROOT" "$SERVICE_USER"
fi

# Backup current deployment
log "Creating backup of current deployment..."
if [[ -f "$PROJECT_ROOT/bin/main" ]]; then
    BACKUP_NAME="celebrum-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$BACKUP_DIR/$BACKUP_NAME"
    cp -r "$PROJECT_ROOT/bin" "$BACKUP_DIR/$BACKUP_NAME/"
    cp -r "$PROJECT_ROOT/configs" "$BACKUP_DIR/$BACKUP_NAME/"
    log "Backup created: $BACKUP_DIR/$BACKUP_NAME"
fi

# Stop services
log "Stopping services..."
systemctl stop celebrum-ai 2>/dev/null || true
systemctl stop celebrum-ccxt 2>/dev/null || true

# Deploy Go backend binary
log "Deploying Go backend binary..."
if [[ -f "bin/celebrum-ai" ]]; then
    # Copy binary
    cp "bin/celebrum-ai" "$PROJECT_ROOT/bin/main"
    chmod +x "$PROJECT_ROOT/bin/main"
    
    # Copy configs
    cp -r "configs" "$PROJECT_ROOT/"
    cp "config.yaml" "$PROJECT_ROOT/"
    
    log "Go backend deployed successfully"
else
    error "Go backend binary not found"
    exit 1
fi

# Copy existing CCXT service if it exists
if [[ -d "/opt/celebrum-ai/ccxt-service" ]]; then
    log "CCXT service already exists, keeping current version"
else
    warn "CCXT service not found, skipping"
fi

# Copy environment file
log "Copying environment configuration..."
if [[ -f ".env" ]]; then
    cp ".env" "$PROJECT_ROOT/.env"
    chmod 600 "$PROJECT_ROOT/.env"
else
    # Create basic environment file
    cat > "$PROJECT_ROOT/.env" << EOF
# Environment: $ENVIRONMENT
DATABASE_URL=postgres://celebrum_user:celebrum_password_123@postgres:5432/celebrum_ai?sslmode=disable
REDIS_URL=redis://redis:6379
SECRET_KEY=local-secret-key
ENVIRONMENT=$ENVIRONMENT
NODE_ENV=$([ "$ENVIRONMENT" == "production" ] && echo "production" || echo "development")
PORT=8080
CCXT_PORT=3001
EOF
fi

# Set proper permissions
log "Setting proper permissions..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$PROJECT_ROOT"
chmod 755 "$PROJECT_ROOT/bin"
chmod 644 "$PROJECT_ROOT/config.yaml"
chmod 600 "$PROJECT_ROOT/.env"

# Create systemd services
log "Creating systemd services..."

# Main service
cat > /etc/systemd/system/celebrum-ai.service << EOF
[Unit]
Description=Celebrum AI Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=$SERVICE_USER
WorkingDirectory=$PROJECT_ROOT
ExecStart=$PROJECT_ROOT/bin/main
Restart=always
RestartSec=10
EnvironmentFile=$PROJECT_ROOT/.env
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=$PROJECT_ROOT/logs
ProtectHome=true
RemoveIPC=true

[Install]
WantedBy=multi-user.target
EOF

# CCXT service (if it exists)
if [[ -d "$PROJECT_ROOT/ccxt-service" ]]; then
    cat > /etc/systemd/system/celebrum-ccxt.service << EOF
[Unit]
Description=Celebrum AI CCXT Service
After=network.target celebrum-ai.service

[Service]
Type=simple
User=$SERVICE_USER
WorkingDirectory=$PROJECT_ROOT/ccxt-service
ExecStart=/root/.bun/bin/bun run start:bun
Restart=always
RestartSec=10
EnvironmentFile=$PROJECT_ROOT/.env
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=$PROJECT_ROOT/logs
ProtectHome=true
RemoveIPC=true

[Install]
WantedBy=multi-user.target
EOF
fi

# Reload systemd
log "Reloading systemd..."
systemctl daemon-reload

# Enable and start services
log "Starting services..."
systemctl enable celebrum-ai
systemctl start celebrum-ai

if [[ -d "$PROJECT_ROOT/ccxt-service" ]]; then
    systemctl enable celebrum-ccxt
    systemctl start celebrum-ccxt
fi

# Health check
log "Performing health check..."
sleep 10

if systemctl is-active --quiet celebrum-ai; then
    log "✓ Celebrum AI service is running"
else
    error "✗ Celebrum AI service failed to start"
    systemctl status celebrum-ai
    exit 1
fi

if systemctl is-active --quiet celebrum-ccxt; then
    log "✓ CCXT service is running"
elif [[ -d "$PROJECT_ROOT/ccxt-service" ]]; then
    warn "⚠ CCXT service failed to start"
fi

# Clean up old backups (keep last 5)
log "Cleaning up old backups..."
cd "$BACKUP_DIR"
ls -t | tail -n +6 | xargs rm -rf 2>/dev/null || true

# Deployment summary
log "=== Deployment Summary ==="
log "Environment: $ENVIRONMENT"
log "Project Root: $PROJECT_ROOT"
log "Services: celebrum-ai$( [[ -d "$PROJECT_ROOT/ccxt-service" ]] && echo ", celebrum-ccxt" || echo "")"
log "Backup: $BACKUP_NAME"
log "Timestamp: $(date)"

log "🎉 Deployment completed successfully!"

# Show service status
echo
log "Service Status:"
systemctl status celebrum-ai --no-pager -l
echo
if [[ -d "$PROJECT_ROOT/ccxt-service" ]]; then
    systemctl status celebrum-ccxt --no-pager -l
fi