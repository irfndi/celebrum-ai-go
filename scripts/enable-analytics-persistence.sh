# Redis configuration for Celebrum AI
# Memory optimization settings

# Enable memory overcommit
vm.overcommit_memory 1

# Set max memory to 512MB (adjust based on your server)
maxmemory 512mb

# Use allkeys-lru eviction policy when memory limit is reached
maxmemory-policy allkeys-lru

# Disable saving to disk for performance (can be enabled if persistence needed)
save ""

# Enable keyspace notifications for cache invalidation
notify-keyspace-events Ex

# Set TCP keepalive
tcp-keepalive 300

# Set timeout for idle connections
timeout 300

# Enable slow log for debugging
slowlog-log-slower-than 10000
slowlog-max-len 128

# Set client output buffer limits
client-output-buffer-limit normal 0 0 0
client-output-buffer-limit replica 512mb 128mb 60
client-output-buffer-limit pubsub 32mb 8mb 60