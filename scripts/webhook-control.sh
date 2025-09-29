#!/bin/bash

# Celebrum AI - Webhook Control Script
# This script manages external connections and webhook enablement

set -euo pipefail

# Configuration
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
ENV_FILE=".env.startup"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_info() {
    log "${BLUE}[INFO]${NC} $1"
}

log_success() {
    log "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    log "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    log "${RED}[ERROR]${NC} $1"
}

# Check if all services are healthy
check_all_services_healthy() {
    local services=("postgres" "redis" "ccxt" "app" "nginx")
    
    for service in "${services[@]}"; do
        local container_name="celebrum-${service}"
        if ! docker ps --filter "name=${container_name}" --filter "health=healthy" --format "table {{.Names}}" | grep -q "${container_name}"; then
            log_error "Service ${service} is not healthy"
            return 1
        fi
    done
    
    return 0
}

# Update environment file
update_env_file() {
    local key="$1"
    local value="$2"
    
    if [ -f "$ENV_FILE" ]; then
        if grep -q "^${key}=" "$ENV_FILE"; then
            sed -i.bak "s/^${key}=.*/${key}=${value}/" "$ENV_FILE"
        else
            echo "${key}=${value}" >> "$ENV_FILE"
        fi
    else
        echo "${key}=${value}" > "$ENV_FILE"
    fi
}

# Update running container environment
update_container_env() {
    local container_name="$1"
    local env_var="$2"
    local env_value="$3"
    
    if docker ps --filter "name=${container_name}" --format "table {{.Names}}" | grep -q "${container_name}"; then
        log_info "Updating ${container_name} environment: ${env_var}=${env_value}"
        docker exec "${container_name}" sh -c "echo '${env_var}=${env_value}' >> /tmp/runtime.env" || true
        
        # Signal the application to reload configuration if supported
        docker exec "${container_name}" sh -c "kill -USR1 1" 2>/dev/null || true
    fi
}

# Enable external connections
enable_external_connections() {
    log_info "Enabling external connections..."
    
    # Check if all services are healthy first
    if ! check_all_services_healthy; then
        log_error "Cannot enable external connections: some services are not healthy"
        return 1
    fi
    
    # Update environment file
    update_env_file "EXTERNAL_CONNECTIONS_ENABLED" "true"
    update_env_file "TELEGRAM_WEBHOOK_ENABLED" "true"
    
    # Update running containers
    update_container_env "celebrum-app" "EXTERNAL_CONNECTIONS_ENABLED" "true"
    update_container_env "celebrum-app" "TELEGRAM_WEBHOOK_ENABLED" "true"
    update_container_env "celebrum-nginx" "EXTERNAL_CONNECTIONS_ENABLED" "true"
    
    # Register Telegram webhook if URL is configured
    if [ -n "${TELEGRAM_WEBHOOK_URL:-}" ] && [ -n "${TELEGRAM_BOT_TOKEN:-}" ]; then
        log_info "Registering Telegram webhook..."
        register_telegram_webhook
    fi
    
    log_success "External connections enabled successfully"
}

# Disable external connections
disable_external_connections() {
    log_info "Disabling external connections..."
    
    # Update environment file
    update_env_file "EXTERNAL_CONNECTIONS_ENABLED" "false"
    update_env_file "TELEGRAM_WEBHOOK_ENABLED" "false"
    
    # Update running containers
    update_container_env "celebrum-app" "EXTERNAL_CONNECTIONS_ENABLED" "false"
    update_container_env "celebrum-app" "TELEGRAM_WEBHOOK_ENABLED" "false"
    update_container_env "celebrum-nginx" "EXTERNAL_CONNECTIONS_ENABLED" "false"
    
    # Unregister Telegram webhook
    if [ -n "${TELEGRAM_BOT_TOKEN:-}" ]; then
        log_info "Unregistering Telegram webhook..."
        unregister_telegram_webhook
    fi
    
    log_success "External connections disabled successfully"
}

# Register Telegram webhook
register_telegram_webhook() {
    if [ -z "${TELEGRAM_BOT_TOKEN:-}" ]; then
        log_error "TELEGRAM_BOT_TOKEN not set"
        return 1
    fi
    
    if [ -z "${TELEGRAM_WEBHOOK_URL:-}" ]; then
        log_error "TELEGRAM_WEBHOOK_URL not set"
        return 1
    fi
    
    local webhook_url="${TELEGRAM_WEBHOOK_URL}"
    local secret_token="${TELEGRAM_WEBHOOK_SECRET:-}"
    
    local curl_data="url=${webhook_url}"
    if [ -n "$secret_token" ]; then
        curl_data="${curl_data}&secret_token=${secret_token}"
    fi
    
    if curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
        -d "$curl_data" | grep -q '"ok":true'; then
        log_success "Telegram webhook registered: $webhook_url"
    else
        log_error "Failed to register Telegram webhook"
        return 1
    fi
}

# Unregister Telegram webhook
unregister_telegram_webhook() {
    if [ -z "${TELEGRAM_BOT_TOKEN:-}" ]; then
        log_error "TELEGRAM_BOT_TOKEN not set"
        return 1
    fi
    
    if curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/deleteWebhook" | grep -q '"ok":true'; then
        log_success "Telegram webhook unregistered"
    else
        log_error "Failed to unregister Telegram webhook"
        return 1
    fi
}

# Get current status
get_status() {
    log_info "Current external connection status:"
    
    if [ -f "$ENV_FILE" ]; then
        local external_enabled=$(grep "^EXTERNAL_CONNECTIONS_ENABLED=" "$ENV_FILE" | cut -d'=' -f2 || echo "false")
        local webhook_enabled=$(grep "^TELEGRAM_WEBHOOK_ENABLED=" "$ENV_FILE" | cut -d'=' -f2 || echo "false")
        
        echo "  External Connections: $external_enabled"
        echo "  Telegram Webhook: $webhook_enabled"
    else
        echo "  Configuration file not found: $ENV_FILE"
    fi
    
    log_info "Service health status:"
    docker-compose -f "$COMPOSE_FILE" ps
}

# Show usage
show_usage() {
    echo "Usage: $0 {enable|disable|status|webhook-register|webhook-unregister}"
    echo ""
    echo "Commands:"
    echo "  enable              Enable external connections and webhooks"
    echo "  disable             Disable external connections and webhooks"
    echo "  status              Show current status"
    echo "  webhook-register    Register Telegram webhook only"
    echo "  webhook-unregister  Unregister Telegram webhook only"
    echo ""
    echo "Environment variables:"
    echo "  TELEGRAM_BOT_TOKEN     Telegram bot token"
    echo "  TELEGRAM_WEBHOOK_URL   Webhook URL for Telegram"
    echo "  TELEGRAM_WEBHOOK_SECRET Optional webhook secret"
}

# Main function
main() {
    # Load environment variables
    if [ -f "$ENV_FILE" ]; then
        set -a
        source "$ENV_FILE"
        set +a
    fi
    
    case "${1:-}" in
        "enable")
            enable_external_connections
            ;;
        "disable")
            disable_external_connections
            ;;
        "status")
            get_status
            ;;
        "webhook-register")
            register_telegram_webhook
            ;;
        "webhook-unregister")
            unregister_telegram_webhook
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"