#!/bin/bash

# Disable Analytics Persistence Script
# This script reverts the system to standard cleanup configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔄 Disabling Analytics Persistence...${NC}"

# Function to restore original configuration
restore_original_config() {
    echo -e "${YELLOW}📝 Restoring original cleanup configuration...${NC}"
    
    # Check for backup files
    backup_file=$(ls -t config.yaml.backup.* 2>/dev/null | head -1)
    
    if [ -n "$backup_file" ]; then
        cp "$backup_file" config.yaml
        echo -e "${GREEN}✅ Restored from backup: $backup_file${NC}"
    else
        # Create standard configuration
        cat > config.yaml << 'EOF'
# Standard Configuration - Original cleanup settings
cleanup:
  market_data_retention_hours: 24
  funding_rate_retention_hours: 24
  arbitrage_retention_hours: 72
  funding_arbitrage_retention_hours: 72
  cleanup_interval_minutes: 60
EOF
        echo -e "${GREEN}✅ Created standard configuration${NC}"
    fi
    
    # Remove analytics environment variables
    if [ -f .env ]; then
        sed -i '/^# Analytics Persistence Settings/,/^ENABLE_HISTORICAL_ANALYSIS=true/d' .env
        echo -e "${GREEN}✅ Removed analytics environment variables${NC}"
    fi
}

# Function to restart services
restart_services() {
    echo -e "${YELLOW}🔄 Restarting services with standard configuration...${NC}"
    
    if command -v docker-compose &> /dev/null; then
        docker-compose down
        docker-compose up -d
        
        sleep 10
        
        if docker-compose ps | grep -q "Up (healthy)"; then
            echo -e "${GREEN}✅ Services restarted with standard configuration${NC}"
        else
            echo -e "${YELLOW}⚠️  Services starting, please check with 'docker-compose ps'${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Docker-compose not available, manual restart required${NC}"
    fi
}

# Function to verify standard configuration
verify_standard_config() {
    echo -e "${YELLOW}🔍 Verifying standard cleanup configuration...${NC}"
    
    echo -e "${GREEN}✅ Standard configuration restored:${NC}"
    echo "  - Market Data: 24 hours retention"
    echo "  - Funding Rates: 24 hours retention"
    echo "  - Arbitrage Opportunities: 72 hours retention"
    echo "  - Funding Arbitrage: 72 hours retention"
    echo "  - Cleanup Interval: 60 minutes"
}

# Main execution
echo -e "${GREEN}🎯 Reverting to Standard Cleanup Configuration${NC}"

restore_original_config
restart_services
verify_standard_config

echo -e "${GREEN}🎉 Analytics persistence disabled - standard cleanup restored${NC}"