#!/bin/bash

# Manual rollback script for quick recovery
set -e

SERVER="root@143.198.219.213"
REMOTE_DIR="/root/celebrum-ai-go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_status "Finding latest backup..."
LATEST_BACKUP=$(ssh $SERVER "cd $REMOTE_DIR && ls -t backups/*.tar.gz 2>/dev/null | head -1")

if [ -z "$LATEST_BACKUP" ]; then
    print_error "No backup found! Cannot rollback."
    exit 1
fi

print_status "Rolling back from backup: $LATEST_BACKUP"

# Stop current services
print_status "Stopping current services..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose down --remove-orphans || true"

# Restore from backup
print_status "Restoring from backup..."
ssh $SERVER "cd $REMOTE_DIR && 
    rm -rf *.yml Dockerfile .env ccxt-service/ configs/ scripts/ internal/ pkg/ cmd/ api/ database/ docs/ go.* && 
    tar -xzf $LATEST_BACKUP"

# Rebuild and restart
print_status "Rebuilding services..."
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml build"
ssh $SERVER "cd $REMOTE_DIR && docker-compose -f docker-compose.single-droplet.yml up -d"

print_status "Rollback completed!"
print_status "Check status with: ssh $SERVER 'cd $REMOTE_DIR && docker-compose ps'"