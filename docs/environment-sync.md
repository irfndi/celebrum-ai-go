# Environment Variable Synchronization Guide

## Overview

This guide ensures proper synchronization of environment variables between local development and remote production environments for the Celebrum AI Go application.

## Critical Environment Variables for Remote Deployment

### Database Configuration
```bash
# For production, use DATABASE_URL instead of individual settings
DATABASE_URL=postgresql://user:password@host:port/database?sslmode=require

# Or individual settings for development
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=your-secure-password
DATABASE_DBNAME=celebrum_ai
DATABASE_SSLMODE=require  # Use 'require' for production
```

### Service URLs (Critical for Docker Networking)
```bash
# Local development
CCXT_SERVICE_URL=http://localhost:3001
REDIS_HOST=localhost

# Remote deployment (Docker Compose)
CCXT_SERVICE_URL=http://ccxt-service:3001
REDIS_HOST=redis
```

### Security Configuration
```bash
# Generate strong secrets for production
JWT_SECRET=your-super-secret-jwt-key-minimum-32-chars
ADMIN_API_KEY=your-secure-admin-api-key
TELEGRAM_WEBHOOK_SECRET=your-webhook-secret
```

### External API Keys
```bash
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
COINMARKETCAP_API_KEY=your_coinmarketcap_api_key
```

## Synchronization Process

### 1. Using rsync (Recommended)
```bash
# Sync .env file to remote server
rsync -av .env user@remote-server:/path/to/deployment/

# Verify sync
ssh user@remote-server 'cat /path/to/deployment/.env | grep -E "(DATABASE_|CCXT_|TELEGRAM_)"'
```

### 2. Manual Verification
```bash
# Run the verification script locally
./scripts/verify-env-sync.sh

# Run on remote server
ssh user@remote-server 'cd /path/to/deployment && ./scripts/verify-env-sync.sh'
```

### 3. Environment-Specific Adjustments

#### Local Development (.env)
```bash
ENVIRONMENT=development
DATABASE_HOST=localhost
CCXT_SERVICE_URL=http://localhost:3001
REDIS_HOST=localhost
DATABASE_SSLMODE=disable
```

#### Remote Production (.env)
```bash
ENVIRONMENT=production
DATABASE_URL=postgresql://user:pass@host:port/db?sslmode=require
CCXT_SERVICE_URL=http://ccxt-service:3001
REDIS_HOST=redis
DATABASE_SSLMODE=require
```

## Key Differences Between Environments

| Variable | Local | Remote |
|----------|-------|--------|
| `CCXT_SERVICE_URL` | `http://localhost:3001` | `http://ccxt-service:3001` |
| `REDIS_HOST` | `localhost` | `redis` |
| `DATABASE_HOST` | `localhost` | Use `DATABASE_URL` instead |
| `DATABASE_SSLMODE` | `disable` | `require` |
| `ENVIRONMENT` | `development` | `production` |

## Verification Checklist

- [ ] `.env` file exists on remote server
- [ ] All required environment variables are set
- [ ] Service URLs use correct Docker service names
- [ ] Database SSL mode is appropriate for environment
- [ ] Security secrets are properly configured
- [ ] External API keys are valid
- [ ] Feature flags are set correctly

## Troubleshooting

### Common Issues

1. **"no such host" errors**
   - Check `CCXT_SERVICE_URL` uses service name, not localhost
   - Verify Docker Compose service names match

2. **Database connection failures**
   - Ensure `DATABASE_URL` is properly formatted
   - Check SSL mode settings
   - Verify database credentials

3. **Redis connection issues**
   - Use `redis` as hostname in Docker environment
   - Check Redis password if set

4. **Telegram webhook failures**
   - Verify `TELEGRAM_WEBHOOK_URL` points to correct domain
   - Check webhook secret matches

### Validation Commands

```bash
# Test database connection
if [ -n "$DATABASE_URL" ]; then
  docker compose exec -T postgres pg_isready -d "$DATABASE_URL"
else
  docker compose exec -T postgres pg_isready -h "$DATABASE_HOST" -p "$DATABASE_PORT"
fi

# Test Redis connection
if [ -n "$REDIS_PASSWORD" ]; then
  docker compose exec -T redis redis-cli -h "$REDIS_HOST" -a "$REDIS_PASSWORD" ping
else
  docker compose exec -T redis redis-cli -h "$REDIS_HOST" ping
fi

# Test CCXT service
docker exec celebrum-app curl "$CCXT_SERVICE_URL/health"

# Check application health
curl http://localhost:8080/health
```

## Security Notes

- Never commit `.env` files to version control
- Use strong, unique passwords and secrets in production
- Rotate API keys and secrets regularly
- Use SSL/TLS for all external connections in production
- Limit access to environment files on remote servers

## Automation

For automated deployments, consider:

1. Using environment variable injection in CI/CD pipelines
2. Secret management services (AWS Secrets Manager, HashiCorp Vault)
3. Configuration management tools (Ansible, Terraform)
4. Container orchestration secrets (Kubernetes secrets, Docker secrets)