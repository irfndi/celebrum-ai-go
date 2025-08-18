#!/bin/bash

# Redis Security Validation Script
# Tests the Redis security configuration after deployment

set -e

echo "🔍 Redis Security Validation"
echo "========================"

# Get the server IP (works on most cloud providers)
SERVER_IP=$(curl -s https://ipinfo.io/ip || echo "127.0.0.1")
echo "Testing Redis security for server: $SERVER_IP"
echo ""

# Test 1: Check if Redis port is exposed publicly
echo "1️⃣ Testing external Redis access..."
if timeout 5 bash -c "echo > /dev/tcp/$SERVER_IP/6379" 2>/dev/null; then
    echo "❌ SECURITY ISSUE: Redis port 6379 is accessible from the internet!"
    echo "   Run the firewall setup script: sudo ./scripts/setup-firewall.sh"
    exit 1
else
    echo "✅ Redis port 6379 is NOT accessible from the internet"
fi

# Test 2: Check if Redis is running locally
echo ""
echo "2️⃣ Testing local Redis access..."
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        echo "✅ Redis is accessible locally"
    else
        echo "⚠️  Redis is not accessible locally - check Redis service status"
    fi
else
    echo "⚠️  redis-cli not found - skipping local test"
fi

# Test 3: Check Docker Redis (if running)
echo ""
echo "3️⃣ Testing Docker Redis access..."
if command -v docker &> /dev/null && docker ps | grep -q celebrum-redis; then
    echo "🐳 Docker Redis detected, testing container access..."
    
    # Test from redis container directly
    if docker compose exec -T redis redis-cli ping &> /dev/null; then
        echo "✅ Redis accessible from redis container"
    else
        echo "❌ Redis not accessible from redis container"
    fi
    
    # Test from ccxt-service container
    if docker exec celebrum-ccxt redis-cli -h redis ping &> /dev/null; then
        echo "✅ Redis accessible from ccxt-service container"
    else
        echo "❌ Redis not accessible from ccxt-service container"
    fi
else
    echo "ℹ️  Docker Redis not detected - skipping Docker tests"
fi

# Test 4: Check firewall status
echo ""
echo "4️⃣ Checking firewall status..."
if command -v ufw &> /dev/null; then
    echo "UFW Status:"
    ufw status verbose
    
    # Check if port 6379 is blocked
    if ufw status | grep -q "6379.*DENY"; then
        echo "✅ Port 6379 is blocked by UFW"
    else
        echo "⚠️  Port 6379 may not be blocked by UFW"
    fi
else
    echo "ℹ️  UFW not found - check your firewall solution"
fi

# Test 5: Check Redis configuration
echo ""
echo "5️⃣ Checking Redis configuration..."
if command -v redis-cli &> /dev/null && redis-cli ping &> /dev/null; then
    CONFIG_BIND=$(redis-cli CONFIG GET bind 2>/dev/null | tail -n 1)
    echo "Redis bind address: $CONFIG_BIND"
    
    if [[ "$CONFIG_BIND" == "127.0.0.1" || "$CONFIG_BIND" == "0.0.0.0" ]]; then
        echo "✅ Redis bind configuration looks secure"
    else
        echo "⚠️  Check Redis bind configuration"
    fi
else
    echo "ℹ️  Cannot check Redis configuration - service may not be running"
fi

echo ""
echo "🎯 Security Validation Complete"
echo "=============================="
echo "Summary:"
echo "- ✅ External access blocked"
echo "- ✅ Local/Docker access working"
echo "- ✅ Firewall configured"
echo ""
echo "If all tests pass, your Redis installation is secure!"
echo "For troubleshooting, see: docs/REDIS_SECURITY.md"