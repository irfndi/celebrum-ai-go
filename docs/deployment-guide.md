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

1. **PostgreSQL Database** (`celebrum-postgres`)
2. **Redis Cache** (`celebrum-redis`)
3. **CCXT Service** (`celebrum-ccxt`)
4. **Main Application** (`celebrum-app`)

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

#### Issue: "no such host" errors for ccxt-service
- **Cause**: Service networking misconfiguration
- **Solution**: Ensure `CCXT_SERVICE_URL=http://ccxt-service:3001` in environment variables
- **Verification**: `docker exec celebrum-app curl http://ccxt-service:3001/health`

#### Issue: Application not responding on port 8080
- **Cause**: Blocking operations preventing HTTP server startup
- **Solution**: Move long-running operations to background goroutines
- **Verification**: `netstat -tlnp | grep 8080` should show the server listening

#### Issue: Database foreign key constraint violations
- **Cause**: Missing exchange records in database
- **Solution**: Ensure proper database migrations and seed data
- **Note**: These are warnings and don't prevent the application from functioning

### 6. Monitoring and Logs

```bash
# Monitor application logs
docker logs -f celebrum-app

# Check for startup messages
docker logs celebrum-app | grep -E '(Service starting|Starting|event: startup)'

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
- [ ] No critical errors in logs (`docker logs celebrum-app`)
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

For persistent issues, restart services in dependency order:

```bash
docker-compose down
docker-compose up -d postgres redis
# Wait for databases to be ready
docker-compose up -d ccxt-service
# Wait for CCXT service to be healthy
docker-compose up -d app
```