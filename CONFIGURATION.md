# Celebrum AI Configuration Guide

This guide explains how to configure Celebrum AI for different environments using our standardized configuration system.

## 🏗️ Configuration Architecture

Our configuration system uses a hierarchical approach:

1. **Environment Variables** (highest priority)
2. **Configuration Files** (`configs/config.{env}.yml`)
3. **Defaults** (lowest priority)

## 📁 Configuration Files

### Core Files

- **`.env.template`** - Template for environment variables
- **`.env`** - Your actual environment variables (git-ignored)
- **`configs/config.template.yml`** - Template for application configuration
- **`configs/config.{env}.yml`** - Environment-specific configurations
- **`docker-compose.yml`** - Base Docker configuration
- **`docker-compose.{env}.yml`** - Environment-specific Docker overrides

### Environment Files

| Environment | Config File | Docker Compose |
|-------------|-------------|----------------|
| Development | `config.local.yml` | `docker-compose.override.yml` |
| Staging | `config.ci.yml` | `docker-compose.staging.yml` |
| Production | `config.prod.yml` | `docker-compose.prod.yml` |

## 🚀 Quick Start

### 1. Initial Setup

```bash
# Run the setup script
./scripts/setup-environment.sh -e development

# Or for production
./scripts/setup-environment.sh -e production
```

### 2. Configure Environment Variables

```bash
# Copy the template
cp .env.template .env

# Edit with your values
nano .env
```

### 3. Start Services

```bash
# Development
docker-compose up -d

# Production
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 🔧 Environment Variables

### Required Variables

| Variable | Description | Development | Production |
|----------|-------------|-------------|------------|
| `ENVIRONMENT` | Environment name | `development` | `production` |
| `POSTGRES_USER` | Database user | `postgres` | Set secret |
| `POSTGRES_PASSWORD` | Database password | `postgres` | Set secret |
| `POSTGRES_DB` | Database name | `celebrum_ai` | Set secret |
| `JWT_SECRET` | JWT signing key | `dev-secret` | Set secret |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Application port |
| `CCXT_SERVICE_URL` | `http://localhost:3001` | CCXT service URL |
| `TELEGRAM_BOT_TOKEN` | - | Telegram bot token |
| `REDIS_PASSWORD` | - | Redis password |

## 🐳 Docker Configuration

### Development

```bash
# Start all services
docker-compose up -d

# With development tools
docker-compose --profile dev-tools up -d

# Available tools:
# - Adminer: http://localhost:8081 (Database admin)
# - Redis Commander: http://localhost:8082 (Redis admin)
```

### Staging

```bash
# Start staging environment
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d
```

### Production

```bash
# Start production environment
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 🔄 CI/CD Integration

### GitHub Actions

The CI/CD pipeline uses environment-specific configurations:

1. **Testing**: Uses `config.ci.yml` with test services
2. **Staging**: Uses `config.staging.yml` with staging overrides
3. **Production**: Uses `config.prod.yml` with production overrides

### Required GitHub Secrets

| Secret | Description | Required |
|--------|-------------|----------|
| `POSTGRES_USER` | Database user | ✅ |
| `POSTGRES_PASSWORD` | Database password | ✅ |
| `POSTGRES_DB` | Database name | ✅ |
| `JWT_SECRET` | JWT signing key | ✅ |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | ❌ |
| `TELEGRAM_WEBHOOK_URL` | Telegram webhook URL | ❌ |
| `TELEGRAM_WEBHOOK_SECRET` | Telegram webhook secret | ❌ |

## 🛠️ Configuration Validation

### Validate Configuration

```bash
# Check configuration syntax
docker-compose config

# Validate environment variables
docker-compose config --quiet

# Check service health
docker-compose ps
```

### Test Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Metrics
curl http://localhost:9090/metrics

# CCXT service health
curl http://localhost:3001/health
```

## 🔍 Troubleshooting

### Common Issues

#### Port Conflicts

```bash
# Check port usage
lsof -i :8080
lsof -i :3001
lsof -i :5432

# Change ports in .env
SERVER_PORT=8081
CCXT_SERVICE_PORT=3002
POSTGRES_PORT=5433
```

#### Database Connection Issues

```bash
# Check database health
docker-compose exec postgres pg_isready -U postgres

# Check connection from app
docker-compose exec app nc -zv postgres 5432
```

#### Redis Connection Issues

```bash
# Check Redis health
docker-compose exec redis redis-cli ping

# Check connection from app
docker-compose exec app nc -zv redis 6379
```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Start with debug output
docker-compose up
```

## 📊 Monitoring

### Health Checks

All services include health checks:

- **PostgreSQL**: `pg_isready`
- **Redis**: `redis-cli ping`
- **App**: `curl /health`
- **CCXT Service**: `curl /health`

### Logs

```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f app
docker-compose logs -f postgres
docker-compose logs -f ccxt-service
```

## 🔄 Configuration Updates

### Updating Configuration

1. **Environment Variables**: Edit `.env` and restart services
2. **Configuration Files**: Edit `configs/config.{env}.yml` and restart services
3. **Docker Configuration**: Edit `docker-compose.{env}.yml` and restart services

### Environment Migration

```bash
# Switch from development to production
./scripts/setup-environment.sh -e production -f
```

## 📝 Best Practices

1. **Never commit sensitive data** to git
2. **Use environment variables** for secrets
3. **Test configurations** in staging before production
4. **Monitor resource usage** in production
5. **Use health checks** for service dependencies
6. **Keep configurations** in version control (except secrets)

## 🎯 Next Steps

1. **Set up monitoring** (Prometheus/Grafana)
2. **Configure alerting** (PagerDuty/Slack)
3. **Set up backup** strategies
4. **Configure SSL/TLS** certificates
5. **Set up log aggregation** (ELK stack)

## 🤝 Contributing

When adding new configuration:

1. Update `config.template.yml`
2. Add to `.env.template`
3. Update relevant `docker-compose.{env}.yml`
4. Update this documentation
5. Test in all environments