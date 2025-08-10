#!/bin/bash

# Manual rollback script for quick recovery
set -euo pipefail

# Default values
DEPLOY_USER="${DEPLOY_USER:-deploy}"
SERVER_IP="${SERVER_IP:-localhost}"
DEPLOY_PATH="${DEPLOY_PATH:-/home/${DEPLOY_USER}/celebrum-ai-go}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--user)
            DEPLOY_USER="$2"
            shift 2
            ;;
        -s|--server)
            SERVER_IP="$2"
            shift 2
            ;;
        -p|--path)
            DEPLOY_PATH="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -u, --user USER        SSH username (default: deploy or $DEPLOY_USER)"
            echo "  -s, --server SERVER    Server IP or hostname (default: localhost or $SERVER_IP)"
            echo "  -p, --path PATH        Remote deployment path (default: /home/USER/celebrum-ai-go)"
            echo "  -h, --help             Show this help message"
            echo ""
            echo "Environment variables can also be used:"
            echo "  DEPLOY_USER, SERVER_IP, DEPLOY_PATH"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

SERVER="${DEPLOY_USER}@${SERVER_IP}"
REMOTE_DIR="${DEPLOY_PATH}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_status "Finding latest backup..."
LATEST_BACKUP=$(ssh "$SERVER" "cd \"$REMOTE_DIR\" && ls -t backups/*.tar.gz 2>/dev/null | head -1")

if [ -z "$LATEST_BACKUP" ]; then
    print_error "No backup found! Cannot rollback."
    exit 1
fi

print_status "Rolling back from backup: $LATEST_BACKUP"

# Stop current services
print_status "Stopping current services..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && docker compose -f docker-compose.single-droplet.yml down --remove-orphans || true"

# Restore from backup
print_status "Restoring from backup..."
ssh "$SERVER" "bash -c 'set -euo pipefail; cd \"$REMOTE_DIR\"; if [ -z \"$LATEST_BACKUP\" ] || [[ ! \"$LATEST_BACKUP\" =~ ^backups/ ]]; then echo \"ERROR: Invalid backup path\" >&2; exit 1; fi; rm -rf ./*.yml ./Dockerfile ./.env ./ccxt-service/ ./configs/ ./scripts/ ./internal/ ./pkg/ ./cmd/ ./api/ ./database/ ./docs/ ./go.*; tar -xzf \"$LATEST_BACKUP\" --strip-components=1'"

# Rebuild and restart
print_status "Rebuilding services..."
ssh "$SERVER" "cd \"$REMOTE_DIR\" && docker compose -f docker-compose.single-droplet.yml build --pull"
ssh "$SERVER" "cd \"$REMOTE_DIR\" && docker compose -f docker-compose.single-droplet.yml up -d"

print_status "Rollback completed!"
print_status "Check status with: ssh \"$SERVER\" 'cd \"$REMOTE_DIR\" && docker compose ps'"