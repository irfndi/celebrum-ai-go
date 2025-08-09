#!/bin/bash

# Script to fix Redis vm.overcommit_memory warning on Linux servers
# This sets the system-level memory overcommit setting for Redis

set -e

echo "🔧 Fixing Redis vm.overcommit_memory system setting..."

# Check if we're running on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "⚠️  This script is intended for Linux systems. Current OS: $OSTYPE"
    echo "   For macOS/local development, this setting is not needed."
    exit 0
fi

# Check current setting
current_value=$(cat /proc/sys/vm/overcommit_memory 2>/dev/null || echo "unknown")
echo "📊 Current vm.overcommit_memory: $current_value"

# Set the recommended value for Redis
if [[ "$current_value" != "1" ]]; then
    echo "📝 Setting vm.overcommit_memory to 1..."
    echo 1 | sudo tee /proc/sys/vm/overcommit_memory > /dev/null
    
    # Make it permanent
    if ! grep -q "vm.overcommit_memory" /etc/sysctl.conf 2>/dev/null; then
        echo "vm.overcommit_memory = 1" | sudo tee -a /etc/sysctl.conf > /dev/null
        echo "✅ Added to /etc/sysctl.conf for persistence"
    else
        echo "✅ vm.overcommit_memory already configured in /etc/sysctl.conf"
    fi
    
    # Apply changes
    sudo sysctl -p > /dev/null 2>&1 || true
    echo "✅ System setting updated successfully"
else
    echo "✅ vm.overcommit_memory is already set correctly"
fi

echo "🎯 Redis system-level memory optimization complete!"