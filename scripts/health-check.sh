#!/bin/bash
# Health check and auto-recovery script for Celebrum AI services

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
HEALTH_CHECK_URL="http://localhost:8080/api/v1/health"
LOG_FILE="/var/log/celebrum-health.log"
MAX_RETRIES=3
RETRY_DELAY=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check if service is running
check_service() {
    local service_name=$1
    if docker-compose ps | grep -q "$service_name.*Up"; then
        return 0
    else
        return 1
    fi
}

# Check service health via HTTP
check_health_endpoint() {
    local url=$1
    local service_name=$2
    
    if curl -f -s "$url" > /dev/null; then
        echo -e "${GREEN}✅ $service_name health check passed${NC}"
        return 0
    else
        echo -e "${RED}❌ $service_name health check failed${NC}"
        return 1
    fi
}

# Get detailed health information
get_health_details() {
    local url=$1
    curl -s "$url" | jq -r '.services | to_entries[] | "\(.key): \(.value)"' 2>/dev/null || echo "Could not parse health response"
}

# Restart a specific service
restart_service() {
    local service_name=$1
    log "Restarting $service_name..."
    
    cd "$PROJECT_DIR"
    docker-compose restart "$service_name"
    
    # Wait for service to come back up
    sleep 10
    
    if check_service "$service_name"; then
        log "✅ $service_name restarted successfully"
        return 0
    else
        log "❌ Failed to restart $service_name"
        return 1
    fi
}

# Full service restart
restart_all_services() {
    log "Restarting all services..."
    
    cd "$PROJECT_DIR"
    docker-compose down
    docker-compose -f docker-compose.single-droplet.yml up -d
    
    # Wait for all services to be ready
    sleep 30
    
    log "✅ All services restarted"
}

# Check all services
check_all_services() {
    echo -e "${YELLOW}🔍 Checking service status...${NC}"
    
    local services=("celebrum-app" "celebrum-ccxt" "celebrum-redis" "celebrum-postgres" "celebrum-nginx")
    local failed_services=()
    
    for service in "${services[@]}"; do
        if check_service "$service"; then
            echo -e "${GREEN}✅ $service is running${NC}"
        else
            echo -e "${RED}❌ $service is not running${NC}"
            failed_services+=("$service")
        fi
    done
    
    # Check application health
    echo -e "${YELLOW}🔍 Checking application health...${NC}"
    if check_health_endpoint "$HEALTH_CHECK_URL" "Application"; then
        echo -e "${GREEN}✅ Application is healthy${NC}"
        get_health_details "$HEALTH_CHECK_URL"
    else
        echo -e "${RED}❌ Application health check failed${NC}"
        failed_services+=("application")
    fi
    
    return ${#failed_services[@]}
}

# Auto-recovery function
auto_recover() {
    local retry_count=0
    
    while [ $retry_count -lt $MAX_RETRIES ]; do
        if check_all_services; then
            log "All services are healthy"
            return 0
        fi
        
        log "Attempting recovery (attempt $((retry_count + 1))/$MAX_RETRIES)"
        restart_all_services
        
        ((retry_count++))
        if [ $retry_count -lt $MAX_RETRIES ]; then
            log "Waiting $RETRY_DELAY seconds before retry..."
            sleep $RETRY_DELAY
        fi
    done
    
    log "❌ Recovery failed after $MAX_RETRIES attempts"
    return 1
}

# Monitor mode
monitor_mode() {
    echo -e "${YELLOW}📊 Starting monitor mode (Press Ctrl+C to exit)...${NC}"
    
    while true; do
        clear
        echo "=== Celebrum AI Service Monitor ==="
        echo "$(date)"
        echo
        
        # Service status
        echo "🐳 Docker Services:"
        docker-compose ps
        echo
        
        # Health check
        echo "❤️  Health Status:"
        if check_health_endpoint "$HEALTH_CHECK_URL" "Application" 2>/dev/null; then
            get_health_details "$HEALTH_CHECK_URL"
        else
            echo "Could not reach health endpoint"
        fi
        echo
        
        # Recent logs
        echo "📝 Recent Logs:"
        docker-compose logs --tail=5 2>/dev/null || echo "No logs available"
        
        sleep 30
    done
}

# Telegram webhook verification
check_telegram_webhook() {
    echo -e "${YELLOW}🤖 Checking Telegram webhook...${NC}"
    
    if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
        source "$PROJECT_DIR/.env" 2>/dev/null || {
            echo -e "${RED}❌ Could not load .env file${NC}"
            return 1
        }
    fi
    
    if [ -n "$TELEGRAM_BOT_TOKEN" ]; then
        response=$(curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getWebhookInfo")
        if echo "$response" | jq -e '.ok' > /dev/null; then
            url=$(echo "$response" | jq -r '.result.url')
            echo -e "${GREEN}✅ Telegram webhook configured: $url${NC}"
            
            # Test webhook
            if curl -s -X POST "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/setWebhook" \
                -d 'url=https://localhost/api/v1/telegram/webhook' > /dev/null; then
                echo -e "${GREEN}✅ Webhook test successful${NC}"
            else
                echo -e "${YELLOW}⚠️  Webhook test failed - may need manual verification${NC}"
            fi
        else
            echo -e "${RED}❌ Could not verify Telegram webhook${NC}"
        fi
    else
        echo -e "${RED}❌ TELEGRAM_BOT_TOKEN not found${NC}"
    fi
}

# Main execution
main() {
    case "${1:-check}" in
        "check")
            check_all_services
            ;;
        "monitor")
            monitor_mode
            ;;
        "recover")
            auto_recover
            ;;
        "telegram")
            check_telegram_webhook
            ;;
        "restart")
            restart_all_services
            ;;
        *)
            echo "Usage: $0 [check|monitor|recover|telegram|restart]"
            echo
            echo "Commands:"
            echo "  check    - Check all service health (default)"
            echo "  monitor  - Continuous monitoring mode"
            echo "  recover  - Auto-recover failed services"
            echo "  telegram - Check Telegram webhook status"
            echo "  restart  - Restart all services"
            exit 1
            ;;
    esac
}

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$LOG_FILE")"

# Run main function
main "$@"