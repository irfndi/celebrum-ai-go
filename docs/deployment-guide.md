# Deployment Guide

## Sequential Deployment Steps

This guide outlines the correct sequential steps for deploying the Celebrum AI Go application to ensure all services start properly and in the correct order.

### 1. Environment Setup

#### Local Development
```bash
# Ensure .env file is properly configured
cp .env.example .env
# Edit .env with your configuration

# Start all services
make dev-up
```

#### Remote Deployment
```bash
# Sync environment variables (using rsync as mentioned)
rsync -av .env user@remote-server:/path/to/deployment/

# Deploy services
docker-compose up -d
```

### 2. Service Dependencies and Startup Order

The services must start in this order:

1. **PostgreSQL Database** (`postgres`)
2. **Redis Cache** (`redis`)
3. **CCXT Service** (`ccxt-service`)
4. **Main Application** (`app`)

### 3. Critical Fix Applied

**Issue**: The `PerformBackfillIfNeeded()` method was blocking the HTTP server startup, causing the application to appear unresponsive even though other services were running.

**Solution**: Moved the backfill operation to run asynchronously in a goroutine:

```go
// Before (blocking)
if err := collectorService.PerformBackfillIfNeeded(); err != nil {
    appLogger.WithError(err).Warn("Backfill failed")
}

// After (non-blocking)
go func() {
    if err := collectorService.PerformBackfillIfNeeded(); err != nil {
        appLogger.WithError(err).Warn("Backfill failed")
    }
}()
```

### 4. Health Check Verification

After deployment, verify all services are healthy:

```bash
# Check container status
docker ps

# Check PostgreSQL
docker-compose ps postgres

# Check Redis
docker-compose ps redis

# Check CCXT Service
docker-compose ps ccxt-service

# Check Main App
docker-compose ps app

# Test main application health
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2025-08-15T15:14:15.431097216Z",
  "services": {
    "ccxt": "healthy",
    "database": "healthy",
    "redis": "healthy",
    "telegram": "healthy"
  },
  "version": "",
  "uptime": "36.581544475s"
}
```

### 5. Common Issues and Solutions

#### Issue: Database foreign key constraint violations
- **Cause**: Missing exchange records in database
- **Solution**: Ensure proper database migrations and seed data
**Impact**: FK violations indicate data integrity issues that can break application flows.
**Action**: Ensure all migrations have run and required seed/reference data exists before starting dependent services.
### 6. Monitoring and Logs

```bash
# Monitor application logs
docker logs -f app

# Check for startup messages
docker logs app | grep -E '(Service starting|Starting|event: startup)'

# Monitor all services
docker-compose logs -f
```

### 7. Environment Variable Synchronization

Ensure the following critical environment variables are synchronized between local and remote:

- `DATABASE_URL`
- `REDIS_URL`
- `CCXT_SERVICE_URL`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

### 8. Deployment Verification Checklist

- [ ] All containers are running (`docker ps`)
- [ ] Health endpoint responds (`curl http://localhost:8080/health`)
- [ ] Server is listening on port 8080 (`netstat -tlnp | grep 8080`)
- [ ] No critical errors in logs (`docker logs app`)
- [ ] CCXT service is accessible from main app
- [ ] Database connections are working
- [ ] Redis connections are working

## Troubleshooting

If deployment fails:

1. Check container logs: `docker logs <container-name>`
2. Verify network connectivity between services
3. Ensure environment variables are correctly set
4. Check that all required ports are available
5. Verify database migrations have been applied

## Common Issues

### CCXT Service Connection Issues
- **Problem**: Main app cannot connect to ccxt-service
- **Solution**: Ensure ccxt-service is healthy before starting main app
- **Check**: `docker logs ccxt-service`

### Database Connection Issues
- **Problem**: App fails to connect to PostgreSQL
- **Solution**: Verify PostgreSQL is healthy and environment variables are set
- **Check**: `docker logs postgres`

### Redis Connection Issues
- **Problem**: App fails to connect to Redis
- **Solution**: Verify Redis is running and accessible
- **Check**: `docker logs redis`

For persistent issues, restart services in dependency order:

### Phase 1: Database Services
1. **Start PostgreSQL**:
   ```bash
   docker-compose up -d postgres
   ```

2. **Start Redis**:
   ```bash
   docker-compose up -d redis
   ```

### Phase 2: CCXT Service
3. **Start CCXT Service**:
   ```bash
   docker-compose up -d ccxt-service
   ```

### Phase 3: Main Application
4. **Start Main Go Application**:
   ```bash
   docker-compose up -d app
   ```