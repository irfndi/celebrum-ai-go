#!/bin/bash

# Manual deployment script using rsync and SSH
# This bypasses CI/CD and deploys directly to the server

set -e

# Configuration
SERVER="root@143.198.219.213"
REMOTE_DIR="/root/celebrum-ai-go"
LOCAL_DIR="$(pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if server is reachable
print_status "Checking server connectivity..."
if ! ssh -o ConnectTimeout=10 $SERVER "echo 'Server is reachable'" >/dev/null 2>&1; then
    print_error "Cannot connect to server $SERVER"
    exit 1
fi

# Create backup of current deployment
print_status "Creating backup of current deployment..."
ssh $SERVER "cd $REMOTE_DIR && if [ -f docker-compose.single-droplet.yml ]; then 
    mkdir -p backups && 
    tar -czf backups/backup-$(date +%Y%m%d-%H%M%S).tar.gz --exclude=backups . || true; 
fi"

# Stop current services gracefully
print_status "Stopping current services..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml down --remove-orphans || true"

# Check if .env file exists locally, skip setup if it does
if [ -f "$LOCAL_DIR/.env" ]; then
    print_status "Found local .env file - it will be synced to server"
else
    print_warning "No .env file found locally. Creating from environment variables..."
    if [ -n "$JWT_SECRET" ] && [ -n "$TELEGRAM_BOT_TOKEN" ]; then
        # Using GitHub Secrets or environment variables
        ssh $SERVER "cd $REMOTE_DIR && cat > .env << EOF
JWT_SECRET=$JWT_SECRET
TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN
TELEGRAM_WEBHOOK_URL=$TELEGRAM_WEBHOOK_URL
TELEGRAM_WEBHOOK_SECRET=$TELEGRAM_WEBHOOK_SECRET
FEATURE_TELEGRAM_BOT=$FEATURE_TELEGRAM_BOT
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:3001
EOF"
    else
        print_warning "GitHub Secrets not found. Please provide environment variables:"
        read -p "JWT_SECRET: " jwt_secret
        read -p "TELEGRAM_BOT_TOKEN: " telegram_bot_token
        read -p "TELEGRAM_WEBHOOK_URL: " telegram_webhook_url
        read -p "TELEGRAM_WEBHOOK_SECRET: " telegram_webhook_secret
        
        ssh $SERVER "cd $REMOTE_DIR && cat > .env << EOF
JWT_SECRET=$jwt_secret
TELEGRAM_BOT_TOKEN=$telegram_bot_token
TELEGRAM_WEBHOOK_URL=$telegram_webhook_url
TELEGRAM_WEBHOOK_SECRET=$telegram_webhook_secret
FEATURE_TELEGRAM_BOT=true
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:3001
EOF"
    fi
fi

# Sync files to server using rsync
print_status "Syncing files to server..."
rsync -avz --delete \
    --exclude='.git' \
    --exclude='.github' \
    --exclude='.env.local' \
    --exclude='node_modules' \
    --exclude='dist' \
    --exclude='*.log' \
    --exclude='backups' \
    --exclude='.trae' \
    --exclude='docs' \
    --exclude='tests' \
    $LOCAL_DIR/ $SERVER:$REMOTE_DIR/

# Build and start services
print_status "Building and starting services..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml build --no-cache"
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml up -d"

# Wait for services to start
print_status "Waiting for services to start..."
sleep 30

# Check service health
print_status "Checking service health..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml ps"

# Show logs for debugging
print_status "Showing recent logs..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml logs --tail=50"

# Test health endpoints
print_status "Testing health endpoints..."
ssh $SERVER "curl -f http://localhost:3000/health || echo 'Main app health check failed'"
ssh $SERVER "curl -f http://localhost:3001/health || echo 'CCXT service health check failed'"

print_status "Manual deployment completed!"
print_status "Services should now be running on the server."
print_status "Check status with: ssh $SERVER 'cd $REMOTE_DIR && docker-compose ps'"