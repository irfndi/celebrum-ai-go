# Manual Deployment Guide

This guide provides instructions for deploying Celebrum AI using manual rsync and SSH instead of a CI/CD pipeline.

## Quick Start

### Prerequisites

- SSH access to the server (`${DEPLOY_USER}@${SERVER_IP}`)
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
ssh ${DEPLOY_USER}@${SERVER_IP} << \'EOF'
cd /home/${DEPLOY_USER}/celebrum-ai-go
cat > .env << \'ENV_EOF'
JWT_SECRET=your-jwt-secret-here
TELEGRAM_BOT_TOKEN=your-bot-token-here
TELEGRAM_WEBHOOK_URL=<https://your-domain.com/webhook>
TELEGRAM_WEBHOOK_SECRET=your-webhook-secret
FEATURE_TELEGRAM_BOT=true
ENVIRONMENT=production
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/celebrum_ai
REDIS_URL=redis://redis:6379
CCXT_SERVICE_URL=http://ccxt-service:8081
ENV_EOF
chmod 600 .env
EOF
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
    --exclude='docs' \
    --exclude='tests' \
    ./ ${DEPLOY_USER}@${SERVER_IP}:/home/${DEPLOY_USER}/celebrum-ai-go/
```

#### Step 3: Build and Deploy

```bash
# Build and start services
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
# Example with Compose V2: ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker compose -f docker-compose.single-droplet.yml build"
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml build"
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml up -d"
```

#### Step 4: Verify Deployment

```bash
# Check service status
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose ps"

# Check logs
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose logs --tail=50"

# Test health endpoints
ssh ${DEPLOY_USER}@${SERVER_IP} "curl -f http://localhost:8080/health"
ssh ${DEPLOY_USER}@${SERVER_IP} "curl -f http://localhost:8081/health"
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
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose logs ccxt-service"
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose logs app"

# Restart specific service
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose restart ccxt-service"
```

#### 2. Database Connection Issues

```bash
# Check database status
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose exec postgres pg_isready"

# Reset database if needed
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose restart postgres"
```

#### 3. Environment Variable Issues

```bash
# Verify .env file exists and has correct values
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && awk -F= '{print $1 \"=<redacted>\"}' .env"

# Recreate .env file if corrupted
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && rm .env && # then recreate using steps above"
```

## Environment Variables Setup

### Required Variables

- `JWT_SECRET`: Random 32+ character string
- `TELEGRAM_BOT_TOKEN`: Bot token from @BotFather
- `TELEGRAM_WEBHOOK_URL`: Your webhook URL (<https://your-domain.com/webhook>)
- `TELEGRAM_WEBHOOK_SECRET`: Random 32+ character string

### Optional Variables

- `FEATURE_TELEGRAM_BOT`: Set to "true" to enable Telegram bot
- `ENVIRONMENT`: Set to "production" for production deployment

## Monitoring After Deployment

### Real-time Logs

```bash
# Watch live logs
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose logs -f"

# Monitor specific service
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose logs -f app"
```

### Health Checks

```bash
# Check all services
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose ps"

# Test external endpoints
curl -I https://${SERVER_DOMAIN}/health
curl -I https://${SERVER_DOMAIN}:8081/health
```

## Security Best Practices

### User Management

- **Never use root user for deployment** - create a dedicated deploy user with limited privileges
- Use SSH key authentication instead of passwords
- Configure proper sudo access for the deploy user if needed

### Create Deploy User (if needed)

```bash
# On the server, create a deploy user
sudo adduser deploy
sudo usermod -aG docker deploy
sudo usermod -aG sudo deploy
```

### Update Deployment Commands

Replace all instances of:

- `root@143.198.219.213` → `${DEPLOY_USER}@${SERVER_IP}`
- `/root/celebrum-ai-go` → `/home/${DEPLOY_USER}/celebrum-ai-go`

## Performance Tips

### 1. Optimize Build Time

```bash
# Use build cache
# Note: Use 'docker compose' (Compose V2) instead of 'docker-compose' (Compose V1) if your Docker version uses the newer CLI.
ssh ${DEPLOY_USER}@${SERVER_IP} "cd /home/${DEPLOY_USER}/celebrum-ai-go && docker-compose -f docker-compose.single-droplet.yml build"

# Pre-pull base images
ssh ${DEPLOY_USER}@${SERVER_IP} "docker pull postgres:15-alpine && docker pull redis:7-alpine"
```

### 2. Resource Management

```bash
# Monitor resource usage
ssh ${DEPLOY_USER}@${SERVER_IP} "docker stats"

# Clean up unused images
ssh ${DEPLOY_USER}@${SERVER_IP} "docker system prune -f"
```

## Security Notes

### .env File Security

- Never commit `.env` file to version control
- Always set secure file permissions: `chmod 600 .env`
- **Never include secrets in shell commands** - use file transfer methods instead
- Use secure file transfer methods like `scp` or `rsync` to move .env files
- Consider using encrypted file transfer or secrets management systems

### Best Practices for Environment Variables

- Use strong, unique secrets for JWT and webhook (32+ characters)
- Regularly rotate secrets (quarterly recommended)
- Monitor server logs for security issues
- Use a non-root user for deployment (see Security Best Practices section)
- Avoid exposing secrets in shell command history

### Secure .env File Transfer Example

```bash
# Instead of creating .env via SSH commands, use secure transfer:
scp .env ${DEPLOY_USER}@${SERVER_IP}:/home/${DEPLOY_USER}/celebrum-ai-go/.env
ssh ${DEPLOY_USER}@${SERVER_IP} "chmod 600 /home/${DEPLOY_USER}/celebrum-ai-go/.env"

# Or use rsync for encrypted transfer:
rsync -avz --chmod=600 .env ${DEPLOY_USER}@${SERVER_IP}:/home/${DEPLOY_USER}/celebrum-ai-go/.env
```

## Quick Commands Reference

```bash
# Full manual deployment
./scripts/deploy-manual.sh

# Quick rollback
./scripts/rollback-manual.sh
```