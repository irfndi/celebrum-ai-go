#!/bin/bash

# Script to optimize rate limits for high-frequency data collection
# This script updates configurations to maximize data collection rates

set -e

echo "🚀 Optimizing rate limits for high-frequency data collection..."

# Backup existing nginx configuration
echo "📋 Backing up existing nginx configuration..."
sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup.$(date +%Y%m%d_%H%M%S)

# Update nginx configuration with new rate limits
echo "⚙️  Updating nginx rate limits..."
sudo cp configs/nginx.single-droplet.conf /etc/nginx/sites-available/default

# Test nginx configuration
echo "🔍 Testing nginx configuration..."
sudo nginx -t

# Restart nginx to apply changes
echo "🔄 Restarting nginx..."
sudo systemctl restart nginx

echo "✅ Nginx rate limits optimized!"

# Update CCXT service configuration
echo "📊 Updating CCXT service rate limits..."

# Create environment variables for CCXT service
cat > .env.ccxt-optimized << EOF
# CCXT Service High-Frequency Configuration
CCXT_RATE_LIMIT_BINANCE=50
CCXT_RATE_LIMIT_BYBIT=8
CCXT_RATE_LIMIT_OKX=3
CCXT_RATE_LIMIT_KRAKEN=1000
CCXT_RATE_LIMIT_COINBASE=100

# Collection intervals
COLLECTION_INTERVAL_MARKET_DATA=30s
COLLECTION_INTERVAL_FUNDING_RATES=60s
COLLECTION_INTERVAL_ORDERBOOK=15s
COLLECTION_INTERVAL_TICKER=10s
COLLECTION_INTERVAL_OHLCV=60s

# Concurrent settings
MAX_CONCURRENT_REQUESTS=50
BATCH_SIZE=100
WORKERS_PER_EXCHANGE=5
EOF

echo "✅ CCXT service configuration updated!"

# Restart services to apply new configurations
echo "🔄 Restarting services..."
docker-compose restart ccxt-service
sleep 5

# Verify services are running
echo "🔍 Verifying services..."
docker-compose ps

echo "🎉 Rate limit optimization complete!"
echo ""
echo "📈 New rate limits:"
echo "- API endpoints: 100 req/sec (was 10 req/sec)"
echo "- Burst capacity: 100 requests (was 20)"
echo "- Collection intervals reduced by 10x"
echo ""
echo "💾 Expected storage increase: 10-20x current volume"
echo "🔄 Run 'docker-compose logs -f ccxt-service' to monitor collection rates"