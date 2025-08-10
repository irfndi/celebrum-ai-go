#!/bin/bash

# Manual deployment script using rsync and SSH
# This bypasses CI/CD and deploys directly to the server

set -euo pipefail

# --- Docker Compose Command Detection ---
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    echo "Error: Neither docker-compose nor docker compose found. Please install one of them." >&2
    exit 1
fi
# --------------------------------------

# Default configuration
SERVER_USER="root"
SERVER_IP="143.198.219.213"
REMOTE_DIR="/root/celebrum-ai-go"
LOCAL_DIR="$(pwd)"

# Parse command line arguments
while getopts "s:d:u:h" opt; do
    case $opt in
        s)
            SERVER_IP="$OPTARG"
            ;;
        d)
            REMOTE_DIR="$OPTARG"
            ;;
        u)
            SERVER_USER="$OPTARG"
            ;;
        h)
            echo "Usage: $0 [-s server_ip] [-d remote_dir] [-u server_user]"
            echo "  -s server_ip    Server IP address (default: 143.198.219.213)"
            echo "  -d remote_dir   Remote directory path (default: /root/celebrum-ai-go)"
            echo "  -u server_user  Server username (default: root)"
            exit 0
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            exit 1
            ;;
    esac
done

# Construct server connection string
SERVER="${SERVER_USER}@${SERVER_IP}"

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
if ! ssh -o ConnectTimeout=10 "$SERVER" "echo 'Server is reachable'" >/dev/null 2>&1; then
    print_error "Cannot connect to server $SERVER"
    exit 1
fi

# Create remote directory if it doesn't exist
print_status "Creating remote directory if needed..."
ssh "$SERVER" "mkdir -p \"$REMOTE_DIR\""

# Create backup of current deployment
print_status "Creating backup of current deployment..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && if [ -f docker-compose.single-droplet.yml ]; then 
    mkdir -p backups && 
    tar -czf backups/backup-\`date +%Y%m%d-%H%M%S\`.tar.gz --exclude=backups . || true; 
fi"

# Stop current services gracefully
print_status "Stopping current services..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && $COMPOSE_CMD -f docker-compose.single-droplet.yml down --remove-orphans || true"

# Check if .env file exists locally, skip setup if it does
if [ -f "$LOCAL_DIR/.env" ]; then
    print_status "Found local .env file - it will be synced to server"
else
    print_warning "No .env file found locally. Creating from environment variables..."
    ENV_CONTENT=""
    if [ -n "${JWT_SECRET:-}" ] && [ -n "${TELEGRAM_BOT_TOKEN:-}" ]; then
        # Using GitHub Secrets or environment variables
        print_status "Using environment variables for .env file"
        ENV_CONTENT=$(cat <<EOF
JWT_SECRET=${JWT_SECRET}
TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
TELEGRAM_WEBHOOK_URL=${TELEGRAM_WEBHOOK_URL:-}
TELEGRAM_WEBHOOK_SECRET=${TELEGRAM_WEBHOOK_SECRET:-}
FEATURE_TELEGRAM_BOT=${FEATURE_TELEGRAM_BOT:-true}
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:8081
EOF
)
    else
        print_warning "Environment variables not found. Please provide values interactively:"
        read -r -s -p "JWT_SECRET: " jwt_secret
        echo
        read -r -s -p "TELEGRAM_BOT_TOKEN: " telegram_bot_token
        echo
        read -r -p "TELEGRAM_WEBHOOK_URL: " telegram_webhook_url
        echo
        read -r -s -p "TELEGRAM_WEBHOOK_SECRET: " telegram_webhook_secret
        echo
        
        ENV_CONTENT=$(cat <<EOF
JWT_SECRET=${jwt_secret}
TELEGRAM_BOT_TOKEN=${telegram_bot_token}
TELEGRAM_WEBHOOK_URL=${telegram_webhook_url}
TELEGRAM_WEBHOOK_SECRET=${telegram_webhook_secret}
FEATURE_TELEGRAM_BOT=true
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:8081
EOF
)
    fi
    echo "$ENV_CONTENT" | ssh "$SERVER" "cat > \"$REMOTE_DIR/.env\""
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
    "$LOCAL_DIR/" "$SERVER:$REMOTE_DIR/"

# Build and start services
print_status "Building and starting services..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && $COMPOSE_CMD -f docker-compose.single-droplet.yml build --no-cache"
ssh "$SERVER" "cd \"$REMOTE_DIR\" && $COMPOSE_CMD -f docker-compose.single-droplet.yml up -d"

# Wait for services to start
print_status "Waiting for services to start..."
sleep 30

# Check service health
print_status "Checking service health..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && $COMPOSE_CMD -f docker-compose.single-droplet.yml ps"

# Show logs for debugging
print_status "Showing recent logs..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && $COMPOSE_CMD -f docker-compose.single-droplet.yml logs --tail=50"

# Test health endpoints
print_status "Testing health endpoints..."
ssh "$SERVER" "curl -f http://localhost:8080/health || echo 'Main app health check failed'"
ssh "$SERVER" "curl -f http://localhost:8081/health || echo 'CCXT service health check failed'"

print_status "Manual deployment completed!"
print_status "Services should now be running on the server."
print_status "Check status with: ssh $SERVER 'cd $REMOTE_DIR && $COMPOSE_CMD ps'"
print_status "Script executed with:"
print_status "  Server: $SERVER"
print_status "  Remote Directory: $REMOTE_DIR"