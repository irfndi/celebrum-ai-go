#!/bin/bash

# Celebrum AI Redis Security Setup Script
# This script configures firewall rules to secure Redis on DigitalOcean Droplets

set -e

echo "🔒 Celebrum AI Redis Security Setup"
echo "=================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root (use sudo)"
   exit 1
fi

# Detect OS
if ! command -v ufw &> /dev/null; then
    echo "❌ UFW not found. This script requires UFW (Ubuntu/Debian)"
    exit 1
fi

echo "📋 Current UFW status:"
ufw status verbose

echo ""
echo "🔧 Configuring UFW rules for Redis security..."

# Reset UFW to defaults (optional - uncomment if needed)
# ufw --force reset

# Default policies
ufw default deny incoming
ufw default allow outgoing

# Allow SSH (modify port if needed)
ufw allow 22/tcp

# Allow HTTP and HTTPS
ufw allow 80/tcp
ufw allow 443/tcp

# Explicitly deny Redis port from external access
ufw deny 6379/tcp

# Allow Docker internal communication (if using Docker)
if command -v docker &> /dev/null; then
    echo "🐳 Docker detected - configuring Docker-specific rules..."
    
    # Allow Docker containers to communicate with Redis
    # This allows internal Docker network traffic
    ufw allow in on docker0 to any port 6379
fi

echo ""
echo "⚠️  Review the following rules before enabling:"
echo "Rules to be applied:"
ufw show added

echo ""
read -p "Do you want to enable these firewall rules? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔥 Enabling UFW..."
    ufw --force enable
    
    echo "✅ UFW enabled successfully!"
    echo "📊 New UFW status:"
    ufw status verbose
else
    echo "⚡ Firewall rules configured but not enabled."
    echo "Run 'sudo ufw enable' when ready to apply changes."
fi

echo ""
echo "🧪 Testing Redis security..."

# Test if Redis is accessible locally
if command -v redis-cli &> /dev/null; then
    echo "Testing local Redis connection..."
    if redis-cli ping &> /dev/null; then
        echo "✅ Redis accessible locally"
    else
        echo "⚠️  Redis not accessible locally (check Redis service)"
    fi
else
    echo "⚠️  redis-cli not found - skipping local test"
fi

echo ""
echo "🔍 Security verification steps:"
echo "1. Test external access: telnet YOUR_IP 6379 (should fail)"
echo "2. Test local access: redis-cli ping (should succeed)"
echo "3. Check UFW logs: sudo tail -f /var/log/ufw.log"
echo ""
echo "📖 For more details, see: docs/REDIS_SECURITY.md"