# Comprehensive Deployment Strategy for Celebrum AI

## 1. Overview

This document provides a step-by-step deployment strategy to ensure reliable production deployments, addressing the specific issues encountered with remote server deployments including service connectivity, migration failures, and server stability.

## 2. Pre-Deployment Checklist

### 2.1 Local Environment Verification
- [ ] All services running locally without errors
- [ ] Database migrations applied successfully
- [ ] CCXT service connectivity verified
- [ ] All tests passing
- [ ] Environment variables properly configured

### 2.2 Environment Synchronization
- [ ] `.env` file synchronized using rsync
- [ ] Environment variables validated on remote server
- [ ] Database credentials verified
- [ ] API keys and secrets confirmed

## 3. Sequential Deployment Steps

### Step 1: Server Preparation
```bash
# Connect to server
ssh root@<server-ip>

# Navigate to project directory
cd /opt/celebrum-ai

# Stop existing services
docker-compose -f docker-compose.single-droplet.yml down

# Clean up containers and volumes if needed
docker system prune -f
docker volume prune -f
```

### Step 2: Environment Synchronization
```bash
# From local machine, sync environment file
rsync -avz .env root@<server-ip>:/opt/celebrum-ai/

# Verify environment file on server
ssh root@<server-ip> "cd /opt/celebrum-ai && cat .env | grep -E 'POSTGRES_|REDIS_|CCXT_'"
```

### Step 3: Database Preparation
```bash
# On server, start only database and redis first
docker-compose -f docker-compose.single-droplet.yml up -d postgres redis

# Wait for database to be ready
docker-compose -f docker-compose.single-droplet.yml logs postgres

# Verify database connectivity
docker-compose -f docker-compose.single-droplet.yml exec postgres psql -U celebrum_user -d celebrum_db -c "SELECT version();"
```

### Step 4: Migration Execution
```bash
# Run migrations with explicit dependency
docker-compose -f docker-compose.single-droplet.yml up migrate

# Verify migration completion
docker-compose -f docker-compose.single-droplet.yml logs migrate

# Check migration status in database
docker-compose -f docker-compose.single-droplet.yml exec postgres psql -U celebrum_user -d celebrum_db -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;"
```

### Step 5: Service Startup
```bash
# Start CCXT service first
docker-compose -f docker-compose.single-droplet.yml up -d ccxt-service

# Wait and verify CCXT service health (retry loop)
for i in {1..30}; do
  if curl -fsS http://localhost:3000/api/health >/dev/null; then
    echo "CCXT service is healthy"; break
  fi
  echo "Waiting for CCXT service... ($i)"; sleep 2
done

# Start main application
docker-compose -f docker-compose.single-droplet.yml up -d app

# Start nginx
docker-compose -f docker-compose.single-droplet.yml up -d nginx
```

### Step 6: Verification
```bash
# Check all container status
docker-compose -f docker-compose.single-droplet.yml ps

# Verify application logs
docker-compose -f docker-compose.single-droplet.yml logs app --tail=50

# Test API endpoints
curl -f http://localhost/api/health
curl -f http://localhost/api/exchanges
```

## 4. Service Dependencies and Health Checks

### 4.1 Startup Order
1. **postgres** - Database must be ready first
2. **redis** - Cache service
3. **migrate** - Database migrations (runs once)
4. **ccxt-service** - External service dependency
5. **app** - Main application (depends on all above)
6. **nginx** - Reverse proxy (last)

### 4.2 Health Check Commands
```bash
# Database health
docker-compose exec postgres pg_isready -U celebrum_user

# Redis health
docker-compose exec redis redis-cli ping

# CCXT service health
curl -f http://localhost:3000/api/health

# Application health
curl -f http://localhost:8080/api/health

# Nginx health
curl -f http://localhost/api/health
```

## 5. Migration Strategy

### 5.1 Migration Best Practices
- Always backup database before migrations
- Run migrations in a separate container with `restart: "no"`
- Verify migration completion before starting application
- Use idempotent migration scripts

### 5.2 Migration Troubleshooting
```bash
# Check migration container logs
docker-compose logs migrate

# Manually run migration script
docker-compose exec postgres psql -U celebrum_user -d celebrum_db -f /docker-entrypoint-initdb.d/migrations/latest.sql

# Reset migrations if needed (DANGER - only in development)
docker-compose exec postgres psql -U celebrum_user -d celebrum_db -c "DROP TABLE IF EXISTS schema_migrations;"
```

## 6. Environment Configuration

### 6.1 Critical Environment Variables
```bash
# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=celebrum_db
POSTGRES_USER=celebrum_user
POSTGRES_PASSWORD=<secure-password>

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<secure-password>

# CCXT Service
CCXT_SERVICE_URL=http://ccxt-service:3000

# Application
SERVER_PORT=8080
ENVIRONMENT=production
```

### 6.2 Environment Validation Script
```bash
#!/bin/bash
# validate-env.sh

required_vars=(
    "POSTGRES_HOST"
    "POSTGRES_DB"
    "POSTGRES_USER"
    "POSTGRES_PASSWORD"
    "REDIS_HOST"
    "CCXT_SERVICE_URL"
)

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "ERROR: $var is not set"
        exit 1
    fi
done

echo "All required environment variables are set"
```

## 7. Troubleshooting Guide

### 7.1 Common Issues and Solutions

#### Issue: CCXT Service Connection Failed
**Symptoms:** `dial tcp: lookup ccxt-service: no such host`
**Solution:**
1. Verify CCXT service is running: `docker-compose ps ccxt-service`
2. Check service logs: `docker-compose logs ccxt-service`
3. Test connectivity: `docker-compose exec app ping ccxt-service`
4. Restart services in order: ccxt-service first, then app

#### Issue: Database Migration Failures
**Symptoms:** Migration container exits with code 1
**Solution:**
1. Check migration logs: `docker-compose logs migrate`
2. Verify database connectivity: `docker-compose exec postgres pg_isready`
3. Check for conflicting migrations
4. Run migrations manually if needed

#### Issue: Server Freezing
**Symptoms:** SSH timeouts, unresponsive services
**Solution:**
1. Check system resources: `htop`, `df -h`
2. Monitor Docker resource usage: `docker stats`
3. Restart server if necessary
4. Implement resource limits in docker-compose

### 7.2 Monitoring Commands
```bash
# System resources
htop
df -h
free -h

# Docker resources
docker stats
docker system df

# Service logs
docker-compose logs --tail=100 -f

# Network connectivity
docker network ls
docker network inspect celebrum-ai_default
```

## 8. Rollback Strategy

### 8.1 Quick Rollback
```bash
# Stop current deployment
docker-compose -f docker-compose.single-droplet.yml down

# Restore from backup (if available)
docker-compose -f docker-compose.single-droplet.yml.backup up -d

# Or revert to previous image tags
docker-compose -f docker-compose.single-droplet.yml up -d --force-recreate
```

### 8.2 Database Rollback
```bash
# Restore database from backup
docker-compose exec postgres pg_restore -U celebrum_user -d celebrum_db /backups/latest.sql

# Or run rollback migrations
docker-compose exec postgres psql -U celebrum_user -d celebrum_db -f /migrations/rollback.sql
```

## 9. Verification Checklist

### 9.1 Post-Deployment Verification
- [ ] All containers running: `docker-compose ps`
- [ ] Database accessible: `psql` connection test
- [ ] Redis accessible: `redis-cli ping`
- [ ] CCXT service responding: `curl http://localhost:3000/api/health`
- [ ] Main app responding: `curl http://localhost:8080/api/health`
- [ ] Nginx serving requests: `curl http://localhost/api/health`
- [ ] API endpoints functional: Test key endpoints
- [ ] Logs clean: No error messages in recent logs

### 9.2 Performance Verification
- [ ] Response times acceptable
- [ ] Memory usage within limits
- [ ] CPU usage stable
- [ ] Disk space sufficient
- [ ] Network connectivity stable

## 10. Automation Scripts

### 10.1 Complete Deployment Script
```bash
#!/bin/bash
# deploy.sh - Complete deployment automation

set -e

SERVER_IP="$1"
if [ -z "$SERVER_IP" ]; then
    echo "Usage: $0 <server-ip>"
    exit 1
fi

echo "Starting deployment to $SERVER_IP..."

# Step 1: Sync environment
echo "Syncing environment..."
rsync -avz .env root@$SERVER_IP:/opt/celebrum-ai/

# Step 2: Deploy services
echo "Deploying services..."
ssh root@$SERVER_IP << 'EOF'
cd /opt/celebrum-ai

# Stop existing services
docker-compose -f docker-compose.single-droplet.yml down

# Start database and redis
docker-compose -f docker-compose.single-droplet.yml up -d postgres redis
sleep 30

# Run migrations
docker-compose -f docker-compose.single-droplet.yml up migrate

# Start services in order
docker-compose -f docker-compose.single-droplet.yml up -d ccxt-service
sleep 30
docker-compose -f docker-compose.single-droplet.yml up -d app
docker-compose -f docker-compose.single-droplet.yml up -d nginx

# Verify deployment
sleep 30
docker-compose -f docker-compose.single-droplet.yml ps
curl -f http://localhost/api/health || echo "Health check failed"
EOF

echo "Deployment completed!"
```

### 10.2 Health Check Script
```bash
#!/bin/bash
# health-check.sh

SERVER_IP="$1"

ssh root@$SERVER_IP << 'EOF'
cd /opt/celebrum-ai

echo "=== Container Status ==="
docker-compose -f docker-compose.single-droplet.yml ps

echo "\n=== Health Checks ==="
echo "Database: $(docker-compose exec -T postgres pg_isready -U celebrum_user)"
echo "Redis: $(docker-compose exec -T redis redis-cli ping)"
echo "CCXT: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/health)"
echo "App: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/health)"
echo "Nginx: $(curl -s -o /dev/null -w "%{http_code}" http://localhost/api/health)"

echo "\n=== Recent Logs ==="
docker-compose -f docker-compose.single-droplet.yml logs --tail=10 app
EOF
```

## 11. Best Practices Summary

1. **Always follow the sequential order**: Database → Redis → Migrations → CCXT → App → Nginx
2. **Verify each step**: Don't proceed until the previous step is confirmed working
3. **Use health checks**: Implement and use health check endpoints
4. **Monitor resources**: Keep an eye on system resources during deployment
5. **Have a rollback plan**: Always be prepared to rollback if something goes wrong
6. **Test locally first**: Ensure everything works locally before deploying
7. **Sync environments**: Keep local and remote environments synchronized
8. **Log everything**: Maintain comprehensive logs for troubleshooting
9. **Automate when possible**: Use scripts to reduce human error
10. **Document issues**: Keep track of problems and solutions for future reference

This strategy ensures reliable, repeatable deployments while minimizing downtime and deployment failures.
