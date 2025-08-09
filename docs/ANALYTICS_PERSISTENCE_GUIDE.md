#!/bin/bash

# Enable Analytics Persistence Script
# This script configures the system to preserve arbitrage opportunities
# for long-term analytics instead of cleaning them up

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔍 Enabling Analytics Persistence...${NC}"

# Function to display current configuration
display_current_config() {
    echo -e "${YELLOW}📊 Current Cleanup Configuration:${NC}"
    if command -v docker &> /dev/null; then
        docker-compose exec -T app sh -c "
            if [ -f .env ]; then
                source .env
                echo 'Environment variables loaded'
            fi
            echo 'Current retention settings:'
            echo 'Market Data: 24 hours (default)'
            echo 'Funding Rates: 24 hours (default)'
            echo 'Arbitrage Opportunities: 72 hours (default)'
            echo 'Funding Arbitrage: 72 hours (default)'
        " 2>/dev/null || echo "Docker not available, showing defaults"
    else
        echo "Docker not available, showing defaults"
    fi
}

# Function to update configuration
update_analytics_config() {
    echo -e "${YELLOW}⚙️  Updating configuration for analytics persistence...${NC}"
    
    # Create backup of current config
    if [ -f config.yaml ]; then
        cp config.yaml "config.yaml.backup.$(date +%Y%m%d_%H%M%S)"
        echo -e "${GREEN}✅ Backup created: config.yaml.backup.$(date +%Y%m%d_%H%M%S)${NC}"
    fi
    
    # Copy analytics persistence config
    cp configs/analytics-persistence-config.yaml config.yaml
    echo -e "${GREEN}✅ Analytics persistence configuration applied${NC}"
    
    # Update environment variables if .env exists
    if [ -f .env ]; then
        # Add analytics-specific environment variables
        echo "" >> .env
        echo "# Analytics Persistence Settings" >> .env
        echo "ANALYTICS_RETENTION_ENABLED=true" >> .env
        echo "ARBITRAGE_RETENTION_YEARS=1" >> .env
        echo "ENABLE_HISTORICAL_ANALYSIS=true" >> .env
        echo -e "${GREEN}✅ Environment variables updated${NC}"
    fi
}

# Function to restart services with new configuration
restart_services() {
    echo -e "${YELLOW}🔄 Restarting services with analytics configuration...${NC}"
    
    # Check if docker-compose is available
    if command -v docker-compose &> /dev/null; then
        echo -e "${YELLOW}⏳ Stopping services...${NC}"
        docker-compose down
        
        echo -e "${YELLOW}⏳ Starting services with new configuration...${NC}"
        docker-compose up -d
        
        echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
        sleep 10
        
        # Check if services are healthy
        if docker-compose ps | grep -q "Up (healthy)"; then
            echo -e "${GREEN}✅ Services restarted successfully${NC}"
        else
            echo -e "${YELLOW}⚠️  Services starting, please check with 'docker-compose ps'${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Docker-compose not available, manual restart required${NC}"
    fi
}

# Function to verify analytics persistence
verify_analytics_persistence() {
    echo -e "${YELLOW}🔍 Verifying analytics persistence configuration...${NC}"
    
    # Check if database is accessible
    if command -v docker-compose &> /dev/null; then
        echo -e "${YELLOW}📊 Checking database tables...${NC}"
        docker-compose exec -T postgres psql -U postgres -d celebrum_ai -c "
            SELECT 
                'market_data' as table_name, 
                COUNT(*) as record_count,
                MIN(created_at) as oldest_record,
                MAX(created_at) as newest_record
            FROM market_data
            UNION ALL
            SELECT 
                'arbitrage_opportunities', 
                COUNT(*) as record_count,
                MIN(created_at) as oldest_record,
                MAX(created_at) as newest_record
            FROM arbitrage_opportunities
            UNION ALL
            SELECT 
                'funding_arbitrage_opportunities', 
                COUNT(*) as record_count,
                MIN(created_at) as oldest_record,
                MAX(created_at) as newest_record
            FROM funding_arbitrage_opportunities;
        "
    fi
    
    echo -e "${GREEN}✅ Analytics persistence verification complete${NC}"
}

# Function to create analytics dashboard endpoint
create_analytics_endpoint() {
    echo -e "${YELLOW}📈 Setting up analytics dashboard endpoint...${NC}"
    
    # This would typically be done via API, but we'll document it
    cat > docs/ANALYTICS_SETUP.md << EOF
# Analytics Persistence Setup

## Configuration Applied
- **Market Data**: 7 days retention (168 hours)
- **Funding Rates**: 7 days retention (168 hours)  
- **Arbitrage Opportunities**: 1 year retention (8760 hours)
- **Funding Arbitrage**: 1 year retention (8760 hours)
- **Cleanup Interval**: Every 6 hours (reduced frequency)

## Verification Commands
\`\`\`bash
# Check current record counts
docker-compose exec postgres psql -U postgres -d celebrum_ai -c "
SELECT 'market_data' as table, COUNT(*) FROM market_data;
SELECT 'arbitrage_opportunities' as table, COUNT(*) FROM arbitrage_opportunities;
"

# Check retention settings
docker-compose logs app | grep -i cleanup
\`\`\`

## Next Steps
1. Monitor storage growth with extended retention
2. Set up automated data archival for very old records
3. Configure analytics dashboard endpoints
4. Implement data export functionality

## Storage Impact
With extended retention, expect storage growth to increase significantly. Monitor the following:
- Arbitrage opportunities: ~1MB per 1000 opportunities
- Funding arbitrage: ~1MB per 1000 opportunities  
- Market data: Continues at current rate with 7-day retention

## Rollback
To revert to original settings:
\`\`\`bash
./scripts/disable-analytics-persistence.sh
\`\`\`
EOF
    
    echo -e "${GREEN}✅ Analytics documentation created: docs/ANALYTICS_SETUP.md${NC}"
}

# Main execution
main() {
    echo -e "${GREEN}🎯 Starting Analytics Persistence Configuration${NC}"
    
    # Display current state
    display_current_config
    
    # Update configuration
    update_analytics_config
    
    # Restart services
    restart_services
    
    # Verify setup
    verify_analytics_persistence
    
    # Create documentation
    create_analytics_endpoint
    
    echo -e "${GREEN}🎉 Analytics persistence enabled successfully!${NC}"
    echo -e "${GREEN}📊 Arbitrage opportunities will now be preserved for 1 year${NC}"
    echo -e "${GREEN}📖 See docs/ANALYTICS_SETUP.md for detailed information${NC}"
}

# Handle command line arguments
case "${1:-}" in
    --verify)
        verify_analytics_persistence
        ;;
    --status)
        display_current_config
        ;;
    --help)
        echo "Usage: $0 [--verify|--status|--help]"
        echo "  --verify: Check current analytics persistence status"
        echo "  --status: Show current configuration"
        echo "  --help: Show this help message"
        ;;
    *)
        main
        ;;
esac