# Manual Deployment Guide

This guide provides instructions for deploying Celebrum AI using manual rsync and SSH instead of CI/CD pipeline.

## Quick Start

### Prerequisites
- SSH access to the server (`root@143.198.219.213`)
- Environment variables configured
- Local development environment ready

### 1. One-Command Manual Deployment

```bash
# Run the automated manual deployment script
./scripts/deploy-manual.sh
```

This script will:
- Check server connectivity
- Create a backup of the current deployment
- Stop existing services gracefully
- Sync all necessary files via rsync
- Build and start services with Docker Compose
- Verify deployment health

### 2. Manual Step-by-Step Deployment

If you prefer to run each step manually:

#### Step 1: Create Environment File
```bash
# SSH to server and create .env file
ssh root@143.198.219.213
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && cat > .env << EOF
JWT_SECRET=your-jwt-secret-here
TELEGRAM_BOT_TOKEN=your-bot-token-here
TELEGRAM_WEBHOOK_URL=https://your-domain.com/webhook
TELEGRAM_WEBHOOK_SECRET=your-webhook-secret
FEATURE_TELEGRAM_BOT=true
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:3001
EOF"
```

#### Step 2: Sync Files
```bash
# Sync all files to server (excluding unnecessary ones)
rsync -avz --delete \
    --exclude='.git' \
    --exclude='.github' \
    --exclude='.env' \
    --exclude='.env.local' \
    --exclude='node_modules' \
    --exclude='bun.lock' \
    --exclude='dist' \
    --exclude='*.log' \
    --exclude='backups' \
    --exclude='.trae' \
    --exclude='docs' \
    --exclude='tests' \
    ./ root@143.198.219.213:/root/celebrum-ai-go/
```

#### Step 3: Build and Deploy
```bash
# Build and start services
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml build --no-cache"
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml up -d"
```

#### Step 4: Verify Deployment
```bash
# Check service status
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose ps"

# Check logs
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs --tail=50"

# Test health endpoints
ssh root@143.198.219.213 "curl -f http://localhost:3000/health"
ssh root@143.198.219.213 "curl -f http://localhost:3001/health"
```

## Troubleshooting

### Rollback to Previous Version

If deployment fails, quickly rollback:

```bash
./scripts/rollback-manual.sh
```

This will:
- Find the latest backup
- Stop current services
- Restore from backup
- Restart services

### Common Issues and Solutions

#### 1. Service Health Check Failures
```bash
# Check individual service logs
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs ccxt-service"
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs app"

# Restart specific service
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose restart ccxt-service"
```

#### 2. Database Connection Issues
```bash
# Check database status
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose exec postgres pg_isready"

# Reset database if needed
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose restart postgres"
```

#### 3. Environment Variable Issues
```bash
# Verify .env file exists and has correct values
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && cat .env"

# Recreate .env file if corrupted
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && rm .env && # then recreate using steps above"
```

## Environment Variables Setup

### Required Variables
- `JWT_SECRET`: Random 32+ character string
- `TELEGRAM_BOT_TOKEN`: Bot token from @BotFather
- `TELEGRAM_WEBHOOK_URL`: Your webhook URL (https://your-domain.com/webhook)
- `TELEGRAM_WEBHOOK_SECRET`: Random 32+ character string

### Optional Variables
- `FEATURE_TELEGRAM_BOT`: Set to "true" to enable Telegram bot
- `ENVIRONMENT`: Set to "production" for production deployment

## Monitoring After Deployment

### Real-time Logs
```bash
# Watch live logs
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs -f"

# Monitor specific service
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs -f app"
```

### Health Checks
```bash
# Check all services
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose ps"

# Test external endpoints
curl -I https://143.198.219.213/health
curl -I https://143.198.219.213:3001/health
```

## Performance Tips

### 1. Optimize Build Time
```bash
# Use build cache
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml build"

# Pre-pull base images
ssh root@143.198.219.213 "docker pull postgres:15-alpine && docker pull redis:7-alpine"
```

### 2. Resource Management
```bash
# Monitor resource usage
ssh root@143.198.219.213 "docker stats"

# Clean up unused images
ssh root@143.198.219.213 "docker system prune -f"
```

## Security Notes

- Never commit `.env` file to version control
- Use strong, unique secrets for JWT and webhook
- Regularly rotate secrets
- Monitor server logs for security issues

## Quick Commands Reference

```bash
# Full manual deployment
./scripts/deploy-manual.sh

# Quick rollback
./scripts/rollback-manual.sh

# Check status
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose ps"

# View logs
ssh root@143.198.219.213 "cd /root/celebrum-ai-go && docker-compose logs --tail=20"
```