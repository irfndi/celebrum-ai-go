#!/bin/bash

# Script to fix Redis vm.overcommit_memory warning on Linux servers
# This sets the system-level memory overcommit setting for Redis

set -euo pipefail

echo "🔧 Fixing Redis vm.overcommit_memory system setting..."

# Determine if we need sudo
SUDO=""
if [[ $EUID -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo "❌ Error: This script requires root privileges or sudo to run."
        exit 1
    fi
fi

# Check if we're running on Linux
if [[ "$(uname -s)" != "Linux" ]]; then
    echo "⚠️  This script is intended for Linux systems. Current OS: $(uname -s)"
    echo "   For macOS/local development, this setting is not needed."
    exit 0
fi

# Check current setting
current_value=$(cat /proc/sys/vm/overcommit_memory 2>/dev/null || echo "unknown")
echo "📊 Current vm.overcommit_memory: $current_value"

# Set the recommended value for Redis
if [[ "$current_value" != "1" ]]; then
    echo "📝 Setting vm.overcommit_memory to 1..."
    echo 1 | $SUDO tee /proc/sys/vm/overcommit_memory > /dev/null
    
    # Create sysctl drop-in file for idempotency
    $SUDO mkdir -p /etc/sysctl.d
    echo "vm.overcommit_memory = 1" | $SUDO tee /etc/sysctl.d/99-redis.conf > /dev/null
    echo "✅ Created /etc/sysctl.d/99-redis.conf for persistence"
    
    # Apply changes
    $SUDO sysctl --system > /dev/null 2>&1 || $SUDO sysctl -p > /dev/null 2>&1 || true
    echo "✅ System setting updated successfully"
else
    echo "✅ vm.overcommit_memory is already set correctly"
fi

echo "🎯 Redis system-level memory optimization complete!"