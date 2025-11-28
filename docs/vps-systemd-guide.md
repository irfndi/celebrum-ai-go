# VPS Multi-Application Management Guide with systemd

A comprehensive guide for deploying and managing multiple isolated applications on a single VPS using systemd, with complete examples for both shared and isolated architectures.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Patterns](#architecture-patterns)
3. [Shared Resources Setup](#shared-resources-setup)
4. [Fully Isolated Setup](#fully-isolated-setup)
5. [Systemd Service Management](#systemd-service-management)
6. [Monitoring & Health Checks](#monitoring--health-checks)
7. [Backup & Recovery](#backup--recovery)
8. [Quick Reference](#quick-reference)

---

## Overview

This guide covers two deployment patterns for multiple applications on VPS:

- **Shared Resources**: Applications share a single database/cache instance (lower overhead)
- **Fully Isolated**: Each application has its own database/cache instances (maximum isolation)

### Prerequisites

- Linux VPS with systemd (Ubuntu 24+)
- PostgreSQL 17+ (or compatible database)
- Redis 6+ (or compatible cache layer)
- Basic Linux/systemd knowledge
- SSH access to VPS

### System Requirements

For reference, examples assume:
- **CPU**: 8 cores
- **RAM**: 12GB
- **Storage**: 50GB

Resource limits are adjustable per application needs.

---

## Architecture Patterns

### Pattern 1: Shared Resources (Lower Overhead)

All applications connect to a single database and cache instance.

```
┌─────────────────────────────────────┐
│     VPS (12GB RAM, 8 CPU)          │
├─────────────────────────────────────┤
│                                      │
│  ┌──────────────────────────────┐  │
│  │  PostgreSQL (port 5432)      │  │
│  │  Redis (port 6379)           │  │
│  └──────────────────────────────┘  │
│           ▲    ▲    ▲               │
│           │    │    │               │
│  ┌────────┴─┐  │  ┌─┴────────────┐ │
│  │ App 1    │  │  │ App 2        │ │
│  │ Port: -  │  │  │ Port: -      │ │
│  └──────────┘  │  └──────────────┘ │
│                │                    │
│          ┌─────┴──────┐             │
│          │ App 3      │             │
│          │ Port: -    │             │
│          └────────────┘             │
│                                      │
└─────────────────────────────────────┘
```

**Use when:**
- Applications trust each other
- Lower resource overhead is priority
- Simplified management needed
- Development/staging environments

---

### Pattern 2: Fully Isolated (Maximum Reliability)

Each application has dedicated database and cache instances on unique ports.

```
┌──────────────────────────────────────────────────┐
│         VPS (12GB RAM, 8 CPU)                   │
├──────────────────────────────────────────────────┤
│                                                   │
│  ┌─────────────────────────────────────────┐   │
│  │ App 1 Stack                              │   │
│  │  ├─ PostgreSQL (port 5433)              │   │
│  │  ├─ Redis (port 6380)                   │   │
│  │  └─ Application                         │   │
│  └─────────────────────────────────────────┘   │
│                                                   │
│  ┌─────────────────────────────────────────┐   │
│  │ App 2 Stack                              │   │
│  │  ├─ PostgreSQL (port 5434)              │   │
│  │  ├─ Redis (port 6381)                   │   │
│  │  └─ Application                         │   │
│  └─────────────────────────────────────────┘   │
│                                                   │
│  ┌─────────────────────────────────────────┐   │
│  │ App 3 Stack                              │   │
│  │  ├─ PostgreSQL (port 5435)              │   │
│  │  ├─ Redis (port 6382)                   │   │
│  │  └─ Application                         │   │
│  └─────────────────────────────────────────┘   │
│                                                   │
└──────────────────────────────────────────────────┘
```

**Use when:**
- Applications must be completely isolated
- Data security/separation is critical
- One application failure must not affect others
- Production environments
- Database corruption must be isolated

---

## Shared Resources Setup

### Directory Structure

```
/root/
├── apps/
│   ├── app1/
│   │   ├── main.py (or server.js, etc.)
│   │   ├── requirements.txt (or package.json)
│   │   ├── venv/ (or node_modules/)
│   │   └── data/              ← Application-specific data
│   │
│   ├── app2/
│   │   ├── main.py
│   │   ├── requirements.txt
│   │   ├── venv/
│   │   └── data/
│   │
│   ├── postgres/
│   │   ├── data/              ← Shared database
│   │   └── postgresql.conf
│   │
│   └── redis/
│       ├── data/              ← Shared cache
│       └── redis.conf
│
├── systemd/
│   ├── postgres.service
│   ├── redis.service
│   ├── app1.service
│   ├── app2.service
│   └── app-stack.target
│
└── backups/
    ├── app1/
    ├── app2/
    ├── postgres/
    └── redis/
```

### PostgreSQL Service (Shared)

**File**: `/etc/systemd/system/postgres.service`

```ini
[Unit]
Description=PostgreSQL Database (Shared)
After=network.target syslog.target
Before=app1.service app2.service

[Service]
Type=simple
User=postgres
WorkingDirectory=/root/apps/postgres
ExecStart=/usr/lib/postgresql/15/bin/postgres \
    -D /root/apps/postgres/data \
    -c config_file=/root/apps/postgres/postgresql.conf

Restart=always
RestartSec=5

MemoryLimit=2G
CPUQuota=100%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=postgres

[Install]
WantedBy=app-stack.target
```

### Redis Service (Shared)

**File**: `/etc/systemd/system/redis.service`

```ini
[Unit]
Description=Redis Cache (Shared)
After=network.target syslog.target
Before=app1.service app2.service

[Service]
Type=simple
User=redis
WorkingDirectory=/root/apps/redis
ExecStart=/usr/bin/redis-server /root/apps/redis/redis.conf

Restart=always
RestartSec=5

MemoryLimit=1G
CPUQuota=50%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=redis

[Install]
WantedBy=app-stack.target
```

### Application Service (App1)

**File**: `/etc/systemd/system/app1.service`

```ini
[Unit]
Description=Application 1
After=network.target postgres.service redis.service
Requires=postgres.service redis.service
Wants=postgres.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/apps/app1
Environment="APP_NAME=app1"
Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="REDIS_HOST=localhost"
Environment="REDIS_PORT=6379"

ExecStart=/root/apps/app1/venv/bin/python /root/apps/app1/main.py

Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=5

MemoryLimit=256M
CPUQuota=50%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=app1

[Install]
WantedBy=app-stack.target
```

### Application Stack Target

**File**: `/etc/systemd/system/app-stack.target`

```ini
[Unit]
Description=Application Stack (Shared Resources)
After=network.target

[Install]
WantedBy=multi-user.target
```

### Initialization & Startup

```bash
# Create directories
mkdir -p /root/apps/{postgres/data,redis/data,app1/data,app2/data}

# Initialize PostgreSQL
sudo -u postgres initdb -D /root/apps/postgres/data

# Configure PostgreSQL
cat > /root/apps/postgres/postgresql.conf << 'EOF'
port = 5432
data_directory = '/root/apps/postgres/data'
listen_addresses = 'localhost'
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
log_directory = '/root/apps/postgres'
EOF

# Configure Redis
cat > /root/apps/redis/redis.conf << 'EOF'
port 6379
dir /root/apps/redis/data
bind 127.0.0.1
maxmemory 1gb
save 900 1
appendonly yes
EOF

# Set permissions
chown -R postgres:postgres /root/apps/postgres
chown -R redis:redis /root/apps/redis

# Load services
sudo systemctl daemon-reload
sudo systemctl enable app-stack.target
sudo systemctl start app-stack.target

# Verify
sudo systemctl status postgres.service redis.service
```

---

## Fully Isolated Setup

### Directory Structure

```
/root/
├── apps/
│   ├── app1/
│   │   ├── main.py
│   │   ├── venv/
│   │   ├── data/              ← App-specific data
│   │   ├── db/
│   │   │   ├── data/          ← App1's PostgreSQL (port 5433)
│   │   │   └── postgresql.conf
│   │   └── cache/
│   │       ├── data/          ← App1's Redis (port 6380)
│   │       └── redis.conf
│   │
│   ├── app2/
│   │   ├── main.py
│   │   ├── venv/
│   │   ├── data/
│   │   ├── db/
│   │   │   ├── data/          ← App2's PostgreSQL (port 5434)
│   │   │   └── postgresql.conf
│   │   └── cache/
│   │       ├── data/          ← App2's Redis (port 6381)
│   │       └── redis.conf
│   │
│   └── app3/
│       ├── main.py
│       ├── venv/
│       ├── data/
│       ├── db/
│       │   ├── data/          ← App3's PostgreSQL (port 5435)
│       │   └── postgresql.conf
│       └── cache/
│           ├── data/          ← App3's Redis (port 6382)
│           └── redis.conf
│
├── systemd/
│   ├── app1-postgres.service
│   ├── app1-redis.service
│   ├── app1.service
│   ├── app1-stack.target
│   ├── app2-postgres.service
│   ├── app2-redis.service
│   ├── app2.service
│   ├── app2-stack.target
│   └── ...
│
└── backups/
    ├── app1-postgres/
    ├── app1-redis/
    ├── app2-postgres/
    ├── app2-redis/
    └── ...
```

### Port Assignment (Reference)

| Service | App1 | App2 | App3 | App4 | App5 |
|---------|------|------|------|------|------|
| PostgreSQL | 5433 | 5434 | 5435 | 5436 | 5437 |
| Redis | 6380 | 6381 | 6382 | 6383 | 6384 |

Pattern: PostgreSQL starts at 5433, Redis starts at 6380

### App1 PostgreSQL Service (Isolated)

**File**: `/etc/systemd/system/app1-postgres.service`

```ini
[Unit]
Description=PostgreSQL Database for App1 (Isolated)
After=network.target
Before=app1.service

[Service]
Type=simple
User=postgres
WorkingDirectory=/root/apps/app1/db
ExecStart=/usr/lib/postgresql/15/bin/postgres \
    -D /root/apps/app1/db/data \
    -c config_file=/root/apps/app1/db/postgresql.conf

Restart=always
RestartSec=5

MemoryLimit=512M
CPUQuota=50%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=app1-postgres

[Install]
WantedBy=app1-stack.target
```

### App1 Redis Service (Isolated)

**File**: `/etc/systemd/system/app1-redis.service`

```ini
[Unit]
Description=Redis Cache for App1 (Isolated)
After=network.target
Before=app1.service

[Service]
Type=simple
User=redis
WorkingDirectory=/root/apps/app1/cache
ExecStart=/usr/bin/redis-server /root/apps/app1/cache/redis.conf

Restart=always
RestartSec=5

MemoryLimit=256M
CPUQuota=50%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=app1-redis

[Install]
WantedBy=app1-stack.target
```

### App1 Application Service (Isolated)

**File**: `/etc/systemd/system/app1.service`

```ini
[Unit]
Description=Application 1 (Isolated Stack)
After=network.target app1-postgres.service app1-redis.service
Requires=app1-postgres.service app1-redis.service
Wants=app1-postgres.service app1-redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/apps/app1
Environment="APP_NAME=app1"
Environment="DB_HOST=localhost"
Environment="DB_PORT=5433"
Environment="REDIS_HOST=localhost"
Environment="REDIS_PORT=6380"

ExecStart=/root/apps/app1/venv/bin/python /root/apps/app1/main.py

Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=5

MemoryLimit=256M
CPUQuota=50%

StandardOutput=journal
StandardError=journal
SyslogIdentifier=app1-app

[Install]
WantedBy=app1-stack.target
```

### App1 Stack Target

**File**: `/etc/systemd/system/app1-stack.target`

```ini
[Unit]
Description=Application 1 Complete Stack (Isolated)
After=network.target

[Install]
WantedBy=multi-user.target
```

### Initialization & Startup (App1)

```bash
# Create directories
mkdir -p /root/apps/app1/{db/data,cache/data,data}

# Initialize PostgreSQL on port 5433
sudo -u postgres initdb -D /root/apps/app1/db/data

# Configure PostgreSQL
cat > /root/apps/app1/db/postgresql.conf << 'EOF'
port = 5433
data_directory = '/root/apps/app1/db/data'
listen_addresses = 'localhost'
max_connections = 20
shared_buffers = 128MB
effective_cache_size = 256MB
log_directory = '/root/apps/app1/db'
EOF

# Configure Redis on port 6380
cat > /root/apps/app1/cache/redis.conf << 'EOF'
port 6380
dir /root/apps/app1/cache/data
bind 127.0.0.1
maxmemory 256mb
save 900 1
appendonly yes
loglevel notice
logfile "/root/apps/app1/cache/redis.log"
EOF

# Set permissions
chown -R postgres:postgres /root/apps/app1/db
chown -R redis:redis /root/apps/app1/cache
chmod 700 /root/apps/app1/db/data

# Load and start
sudo systemctl daemon-reload
sudo systemctl enable app1-stack.target
sudo systemctl start app1-stack.target

# Verify
sudo systemctl status app1-postgres.service app1-redis.service app1.service
```

### Repeat for Additional Apps

For each additional app (app2, app3, etc.), repeat the steps above with:
- Incremented database port (5434, 5435, 5436, etc.)
- Incremented Redis port (6381, 6382, 6383, etc.)
- Different service names (app2-postgres.service, app2-redis.service, app2.service, app2-stack.target)
- Different working directories (/root/apps/app2/, etc.)

---

## Systemd Service Management

### Service Control Commands

```bash
# Start all services (respects dependencies)
sudo systemctl start app-stack.target

# Stop all services
sudo systemctl stop app-stack.target

# Restart all services
sudo systemctl restart app-stack.target

# Restart single service
sudo systemctl restart app1.service

# Enable auto-start on boot
sudo systemctl enable app1.service

# Check service status
sudo systemctl status app1.service

# View service logs
sudo journalctl -u app1.service -f

# View logs from specific time
sudo journalctl -u app1.service --since "1 hour ago"

# Filter only errors
sudo journalctl -u app1.service -p err

# Check service dependencies
systemctl list-dependencies app-stack.target --all

# List all active services
systemctl list-units --type=service --state=running
```

### Service Dependency Keywords

| Keyword | Behavior |
|---------|----------|
| `After=` | Wait for this unit to start first (order only) |
| `Before=` | This unit starts before the specified unit |
| `Requires=` | Fail if required unit fails; stop if required unit stops |
| `Wants=` | Try to start this unit; continue if it fails |
| `Binds to=` | Like Requires but stops this unit if binding fails |

### Service Restart Policies

```ini
# Never restart on failure
Restart=no

# Always restart
Restart=always

# Restart only on abnormal exit
Restart=on-failure

# Restart with backoff
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=5
```

---

## Monitoring & Health Checks

### Basic Status Monitor Script

**File**: `/root/monitor.sh`

```bash
#!/bin/bash

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              APPLICATION STACK MONITOR                        ║"
echo "║              $(date '+%Y-%m-%d %H:%M:%S')                      ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Configuration
SERVICES=("postgres" "redis" "app1" "app2")
FAILED=0
RUNNING=0

# Check each service
for service in "${SERVICES[@]}"; do
    FULL_SERVICE="$service.service"
    STATUS=$(systemctl is-active $FULL_SERVICE 2>/dev/null)
    
    if [ "$STATUS" = "active" ]; then
        PID=$(systemctl show -p MainPID --value $FULL_SERVICE)
        CPU=$(ps -p $PID -o %cpu= 2>/dev/null | xargs)
        MEM=$(ps -p $PID -o %mem= 2>/dev/null | xargs)
        
        echo "✅ $FULL_SERVICE"
        echo "   │ PID: $PID  |  CPU: ${CPU:-N/A}%  |  MEM: ${MEM:-N/A}%"
        ((RUNNING++))
    else
        echo "❌ $FULL_SERVICE"
        echo "   │ STATUS: INACTIVE"
        ((FAILED++))
    fi
    echo ""
done

# System resources
echo "SYSTEM RESOURCES:"
echo "─────────────────"
free -h | grep Mem | awk '{print "Memory: " $3 " / " $2}'
df -h / | tail -1 | awk '{print "Disk: " $3 " / " $2 " (" $5 ")"}'
uptime | awk -F'load average:' '{print "Load:" $2}'

echo ""
echo "SUMMARY: ✅ $RUNNING Running  |  ❌ $FAILED Failed"
```

**Make executable and schedule:**

```bash
chmod +x /root/monitor.sh

# Manual run
/root/monitor.sh

# Cron: every 5 minutes
*/5 * * * * /root/monitor.sh >> /var/log/app-monitor.log 2>&1
```

### Real-Time Monitoring Tools

```bash
# systemd cgroup monitor (shows resource usage)
systemd-cgtop

# Filter specific services
systemd-cgtop | grep "app\|postgres\|redis"

# View all service logs live
sudo journalctl -u app1.service -u app2.service -u postgres.service -u redis.service -f
```

### Health Check Script

**File**: `/root/health-check.sh`

```bash
#!/bin/bash

LOG_FILE="/var/log/health-check.log"
UNHEALTHY=0

check_port() {
    local port=$1
    local name=$2
    
    if ! nc -z localhost $port &>/dev/null; then
        echo "[$(date)] ❌ $name (port $port) unreachable" >> $LOG_FILE
        return 1
    fi
    return 0
}

# Check PostgreSQL
check_port 5432 "PostgreSQL" || ((UNHEALTHY++))

# Check Redis
check_port 6379 "Redis" || ((UNHEALTHY++))

# Check application health endpoint (if exists)
# curl -s http://localhost:3000/health > /dev/null || ((UNHEALTHY++))

if [ $UNHEALTHY -eq 0 ]; then
    echo "[$(date)] ✅ All services healthy" >> $LOG_FILE
else
    echo "[$(date)] ⚠️  $UNHEALTHY services unhealthy - attempting restart" >> $LOG_FILE
    # Optional: auto-restart
    # sudo systemctl restart app-stack.target
fi
```

**Schedule every 5 minutes:**

```cron
*/5 * * * * /root/health-check.sh > /dev/null 2>&1
```

---

## Backup & Recovery

### Backup Structure

```
/root/backups/
├── app1/
│   ├── app1-db-2025-11-27_120000.sql
│   ├── app1-data-2025-11-27_120000.tar.gz
│   └── ...
├── app2/
│   ├── app2-db-2025-11-27_120000.sql
│   └── ...
└── postgres/ (if shared)
    ├── postgres-2025-11-27_120000.sql
    └── ...
```

### Backup Script (Single App)

**File**: `/root/backup-app1.sh`

```bash
#!/bin/bash

APP_DIR="/root/apps/app1"
BACKUP_DIR="/root/backups/app1"
TIMESTAMP=$(date +"%Y-%m-%d_%H%M%S")
LOG_FILE="/var/log/backup-app1.log"

mkdir -p "$BACKUP_DIR"

# Backup application data directory
if [ -d "$APP_DIR/data" ]; then
    tar -czf "$BACKUP_DIR/app1-data-$TIMESTAMP.tar.gz" -C "$APP_DIR" data/ 2>/dev/null
    echo "[$(date)] Data backup: SUCCESS" >> $LOG_FILE
fi

# Backup database (if PostgreSQL)
if command -v pg_dump &> /dev/null; then
    pg_dump -h localhost -U postgres app1_db > "$BACKUP_DIR/app1-db-$TIMESTAMP.sql" 2>/dev/null
    echo "[$(date)] Database backup: SUCCESS" >> $LOG_FILE
fi

# Cleanup old backups (keep 7 days)
find "$BACKUP_DIR" -type f -mtime +7 -delete
echo "[$(date)] Cleanup: old backups removed" >> $LOG_FILE
```

### Backup Script (All Apps - Shared Resources)

**File**: `/root/backup-all.sh`

```bash
#!/bin/bash

BACKUP_DIR="/root/backups"
TIMESTAMP=$(date +"%Y-%m-%d_%H%M%S")
LOG_FILE="/var/log/backup-all.log"

echo "[$(date)] ====== BACKUP START ======" >> $LOG_FILE

# Backup each app
for app in app1 app2 app3; do
    APP_DIR="/root/apps/$app"
    
    if [ -d "$APP_DIR/data" ]; then
        mkdir -p "$BACKUP_DIR/$app"
        tar -czf "$BACKUP_DIR/$app/${app}-data-$TIMESTAMP.tar.gz" -C "$APP_DIR" data/ 2>/dev/null
        echo "[$(date)] $app data backup: SUCCESS" >> $LOG_FILE
    fi
done

# Backup shared PostgreSQL
if command -v pg_dump &> /dev/null; then
    mkdir -p "$BACKUP_DIR/postgres"
    pg_dump -U postgres --all-databases > "$BACKUP_DIR/postgres/postgres-$TIMESTAMP.sql" 2>/dev/null
    echo "[$(date)] PostgreSQL backup: SUCCESS" >> $LOG_FILE
fi

# Backup shared Redis
if [ -d "/root/apps/redis/data" ]; then
    mkdir -p "$BACKUP_DIR/redis"
    cp /root/apps/redis/data/dump.rdb "$BACKUP_DIR/redis/redis-$TIMESTAMP.rdb" 2>/dev/null
    echo "[$(date)] Redis backup: SUCCESS" >> $LOG_FILE
fi

# Cleanup old backups (keep 7 days)
find "$BACKUP_DIR" -type f -mtime +7 -delete

echo "[$(date)] ====== BACKUP END ======" >> $LOG_FILE
```

### Backup Schedule

**Add to crontab** (`sudo crontab -e`):

```cron
# Daily backup at 2 AM
0 2 * * * /root/backup-all.sh > /dev/null 2>&1

# Or every 6 hours
0 */6 * * * /root/backup-all.sh > /dev/null 2>&1

# Or every day at multiple times
0 2 * * * /root/backup-app1.sh > /dev/null 2>&1
0 2 * * * /root/backup-app2.sh > /dev/null 2>&1
0 14 * * * /root/backup-app1.sh > /dev/null 2>&1
```

### Restore from Backup

```bash
# Restore app1 data directory
cd /root/apps/app1
tar -xzf /root/backups/app1/app1-data-2025-11-27_120000.tar.gz

# Restore PostgreSQL database
sudo -u postgres psql < /root/backups/postgres/postgres-2025-11-27_120000.sql

# Restore Redis data
cp /root/backups/redis/redis-2025-11-27_120000.rdb /root/apps/redis/data/dump.rdb
sudo systemctl restart redis.service

# Restart application
sudo systemctl restart app1.service
```

---

## Resource Management

### Memory & CPU Limits

In service files, resource limits prevent runaway processes:

```ini
[Service]
# Limit memory to 512MB
MemoryLimit=512M

# Limit CPU to 100% (1 core)
CPUQuota=100%

# Limit CPU to 50% (half of 1 core)
CPUQuota=50%

# Limit CPU to 200% (2 cores)
CPUQuota=200%
```

### Recommended Resource Allocation

| Component | 12GB System | 16GB System | 32GB System |
|-----------|-------------|-------------|-------------|
| PostgreSQL (Shared) | 2GB | 3GB | 8GB |
| Redis (Shared) | 1GB | 2GB | 4GB |
| Per Application | 256-512MB | 512MB-1GB | 1-2GB |
| OS/System | 1GB | 1GB | 2GB |

---

## Quick Reference

### Common Tasks

```bash
# View status of all services
systemctl status app-stack.target

# Start everything
sudo systemctl start app-stack.target

# Stop everything
sudo systemctl stop app-stack.target

# Restart app1
sudo systemctl restart app1.service

# View app1 logs
sudo journalctl -u app1.service -f

# Monitor resource usage
systemd-cgtop

# Check if PostgreSQL is running
nc -z localhost 5432 && echo "PostgreSQL UP" || echo "PostgreSQL DOWN"

# Check if Redis is running
nc -z localhost 6379 && echo "Redis UP" || echo "Redis DOWN"

# View all open ports
netstat -tln | grep LISTEN

# Count running applications
systemctl list-units --type=service --state=running | grep -c app

# Reload systemd (after editing .service files)
sudo systemctl daemon-reload

# Enable service to auto-start on boot
sudo systemctl enable app1.service

# Disable auto-start
sudo systemctl disable app1.service

# Manually trigger backup
/root/backup-all.sh

# Check backup status
ls -lah /root/backups/
```

### Troubleshooting

```bash
# Service won't start - check logs
sudo journalctl -u app1.service -n 50

# Check if port is already in use
netstat -tln | grep 5433

# Kill process using port
sudo fuser -k 5433/tcp

# Check service dependencies
systemctl list-dependencies app1.service

# Test PostgreSQL connection
psql -h localhost -U postgres -d app1_db

# Test Redis connection
redis-cli -p 6379 ping

# Check disk space
df -h

# Monitor system in real-time
top

# Check systemd service syntax
systemd-analyze verify /etc/systemd/system/app1.service
```

---

## Best Practices

### Security

- Run each service with minimal required privileges
- Use unique ports for isolated instances
- Enable firewall rules to restrict access
- Rotate backup storage off-site
- Use strong database passwords
- Enable authentication for Redis (if exposed)

### Reliability

- Always set `Restart=always` for critical services
- Use `Requires=` for hard dependencies
- Set `After=` for soft ordering
- Monitor service health continuously
- Maintain automated backups (at least daily)
- Test restore procedures regularly
- Use resource limits to prevent cascading failures

### Performance

- Monitor CPU and memory usage regularly
- Adjust resource limits based on actual usage
- Use separate PostgreSQL instances only if necessary
- Cache frequently accessed data in Redis
- Keep logs rotated (systemd does this automatically)

### Maintenance

- Document environment variables for each service
- Keep systemd service files under version control
- Test changes in staging before production
- Keep backup logs for audit trails
- Regularly review and update backup retention policies

---

## Comparison Table

| Aspect | Shared Resources | Fully Isolated |
|--------|------------------|----------------|
| Database instances | 1 | Per app |
| Cache instances | 1 | Per app |
| Data separation | Logical (databases) | Physical (separate ports) |
| Resource overhead | Low | Medium-High |
| One failure impact | All apps affected | Only that app |
| Backup complexity | Simple | Per-app backups |
| Scaling | Centralized | Independent per app |
| Best for | Dev/staging | Production |
| Cost (resources) | Lower | Higher |
| Complexity | Lower | Higher |
| Recovery granularity | Full system | Single app |

---

## Conclusion

This guide provides two deployment patterns on systemd:

1. **Shared Resources**: Lower overhead, simpler management, suitable for trusted applications
2. **Fully Isolated**: Maximum reliability, better failure isolation, suitable for production

Both patterns use systemd for:
- Service lifecycle management
- Automatic restart on failure
- Dependency handling
- Resource limits
- Centralized logging

Choose the pattern that matches your reliability requirements and resource constraints.

For questions or issues, refer to systemd documentation or consult your system administrator.

---

**Last Updated**: November 27, 2025  
**Version**: 1.0  
**License**: Free to use and modify
