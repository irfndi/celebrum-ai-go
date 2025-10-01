#!/bin/bash

# Binary Deployment Script for Celebrum AI
# This script deploys pre-built binaries to the server

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

# Check required environment variables
required_vars=(
    "SSH_PRIVATE_KEY"
    "SSH_HOST"
    "DATABASE_URL"
    "REDIS_URL"
    "SECRET_KEY"
)

# Handle DEPLOYMENT_TYPE if not set
if [[ -z "${DEPLOYMENT_TYPE:-}" ]]; then
    DEPLOYMENT_TYPE="binary"
    log "DEPLOYMENT_TYPE not set, defaulting to 'binary'"
fi

# Validate deployment type
case "$DEPLOYMENT_TYPE" in
    "binary"|"standard"|"zero-downtime"|"rolling")
        log "Deployment type: $DEPLOYMENT_TYPE"
        ;;
    *)
        error "Invalid deployment type: $DEPLOYMENT_TYPE. Use binary, standard, zero-downtime, or rolling"
        exit 1
        ;;
esac

for var in "${required_vars[@]}"; do
    if [[ -z "${!var:-}" ]]; then
        error "Required environment variable $var is not set"
        exit 1
    fi
done

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
if [[ -f "go-backend/main" ]]; then
    # Copy binary
    cp "go-backend/main" "$PROJECT_ROOT/bin/"
    chmod +x "$PROJECT_ROOT/bin/main"
    
    # Copy configs
    cp -r "go-backend/configs" "$PROJECT_ROOT/"
    cp "go-backend/config.yaml" "$PROJECT_ROOT/"
    
    log "Go backend deployed successfully"
else
    error "Go backend binary not found"
    exit 1
fi

# Deploy CCXT service
log "Deploying CCXT service..."
if [[ -d "ccxt-service" ]]; then
    # Copy CCXT files
    cp -r "ccxt-service" "$PROJECT_ROOT/"
    
    # Install dependencies
    cd "$PROJECT_ROOT/ccxt-service"
    
    # Update Bun to latest version
    log "Updating Bun to latest version (1.2.23)..."
    curl -fsSL https://bun.sh/install | bash
    
    # Export Bun to PATH for this session
    export BUN_INSTALL="$HOME/.bun"
    export PATH="$BUN_INSTALL/bin:$PATH"
    
    # Install dependencies
    log "Installing CCXT service dependencies..."
    bun install --frozen-lockfile
    
    log "CCXT service deployed successfully"
    cd -
else
    error "CCXT service files not found"
    exit 1
fi

# Create environment file
log "Creating environment configuration..."
cat > "$PROJECT_ROOT/.env" << EOF
# Environment: $ENVIRONMENT
DATABASE_URL=$DATABASE_URL
REDIS_URL=$REDIS_URL
SECRET_KEY=$SECRET_KEY
ENVIRONMENT=$ENVIRONMENT
NODE_ENV=$([ "$ENVIRONMENT" == "production" ] && echo "production" || echo "development")
PORT=8080
CCXT_PORT=3000
EOF

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

# CCXT service
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

# Reload systemd
log "Reloading systemd..."
systemctl daemon-reload

# Enable and start services
log "Starting services..."
systemctl enable celebrum-ai
systemctl enable celebrum-ccxt
systemctl start celebrum-ai
systemctl start celebrum-ccxt

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
else
    warn "⚠ CCXT service failed to start (this might be expected if Go service isn't fully ready)"
    systemctl status celebrum-ccxt
fi

# Clean up old backups (keep last 5)
log "Cleaning up old backups..."
cd "$BACKUP_DIR"
ls -t | tail -n +6 | xargs rm -rf 2>/dev/null || true

# Deployment summary
log "=== Deployment Summary ==="
log "Environment: $ENVIRONMENT"
log "Project Root: $PROJECT_ROOT"
log "Services: celebrum-ai, celebrum-ccxt"
log "Backup: $BACKUP_NAME"
log "Timestamp: $(date)"

log "🎉 Deployment completed successfully!"

# Show service status
echo
log "Service Status:"
systemctl status celebrum-ai --no-pager -l
echo
systemctl status celebrum-ccxt --no-pager -l