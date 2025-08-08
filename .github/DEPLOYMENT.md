# Deployment Setup Guide

This guide explains how to set up CI/CD for the Celebrum AI Go project using GitHub Actions.

## Required GitHub Secrets

To enable automatic deployment, you need to configure the following secrets in your GitHub repository:

### Repository Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions → New repository secret

#### Required Secrets:

1. **`DIGITALOCEAN_ACCESS_TOKEN`**
   - Your DigitalOcean API token
   - Used for deployment operations
   - Get from: DigitalOcean Control Panel → API → Tokens/Keys

2. **`DEPLOY_SSH_KEY`**
   - Private SSH key for accessing your deployment server
   - Should be the private key corresponding to the public key added to your server
   - Generate with: `ssh-keygen -t ed25519 -C "github-actions@celebrum-ai"`

3. **`DEPLOY_USER`**
   - Username for SSH access to your deployment server
   - Example: `root` or `deploy`

4. **`DEPLOY_HOST`**
   - IP address or hostname of your deployment server
   - Example: `your-server-ip` or `celebrum-ai.com`

#### Optional Secrets:

5. **`SLACK_WEBHOOK_URL`** (Optional)
   - Slack webhook URL for deployment notifications
   - Get from: Slack → Apps → Incoming Webhooks

6. **`DISCORD_WEBHOOK_URL`** (Optional)
   - Discord webhook URL for deployment notifications

### Repository Variables

Go to your GitHub repository → Settings → Secrets and variables → Actions → Variables tab

1. **`PRODUCTION_URL`**
   - Your production application URL
   - Example: `https://celebrum-ai.com`

## Server Setup

### 1. Prepare Your Server

Ensure your deployment server has:

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Create deployment directory
sudo mkdir -p /opt/celebrum-ai-go
sudo chown $USER:$USER /opt/celebrum-ai-go
```

### 2. Add SSH Key

Add the public key to your server:

```bash
# On your server
mkdir -p ~/.ssh
echo "your-public-key-here" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
chmod 700 ~/.ssh
```

### 3. Clone Repository

```bash
cd /opt/celebrum-ai-go
git clone https://github.com/your-username/celebrum-ai-go.git .
```

### 4. Setup Environment

Create your production environment file:

```bash
cp .env.example .env
# Edit .env with your production values
nano .env
```

## CI/CD Pipeline Overview

The GitHub Actions workflow (`.github/workflows/ci-cd.yml`) includes:

### 1. Test Job
- Runs on every push and pull request
- Sets up Go environment
- Runs linting with golangci-lint
- Executes tests with coverage
- Builds the application
- Uses PostgreSQL and Redis services for integration tests

### 2. Build and Push Job
- Runs only on pushes to main branch
- Builds Docker image
- Pushes to GitHub Container Registry (ghcr.io)
- Tags with branch name and latest

### 3. Deploy Job
- Runs only on pushes to main branch after successful build
- Connects to your server via SSH
- Pulls latest Docker images
- Updates the deployment using docker-compose
- Performs health checks

### 4. Notify Job
- Sends deployment status notifications
- Runs regardless of success/failure

## Manual Deployment

You can also deploy manually using the Makefile:

```bash
# Deploy to production
make deploy

# Deploy to staging
make deploy-staging

# Run CI checks locally
make ci-check
```

## Monitoring and Logs

### View Deployment Logs
```bash
# On your server
tail -f /var/log/celebrum-ai-deploy.log
```

### View Application Logs
```bash
# On your server
cd /opt/celebrum-ai-go
docker-compose logs -f
```

### Health Check
```bash
# Check if application is running
curl -f http://your-server/health
```

## Rollback Procedure

If deployment fails, the system automatically attempts rollback. For manual rollback:

```bash
# On your server
cd /opt/celebrum-ai-go

# Stop current deployment
docker-compose down

# Restore from backup
sudo cp -r /var/backups/celebrum-ai/backup-YYYYMMDD-HHMMSS/* .

# Start services
docker-compose up -d
```

## Troubleshooting

### Common Issues

1. **SSH Connection Failed**
   - Verify SSH key is correctly added to server
   - Check DEPLOY_USER and DEPLOY_HOST secrets
   - Ensure server allows SSH connections

2. **Docker Build Failed**
   - Check Dockerfile syntax
   - Verify all dependencies are available
   - Check GitHub Container Registry permissions

3. **Health Check Failed**
   - Verify application starts correctly
   - Check database connectivity
   - Review application logs

4. **Permission Denied**
   - Ensure deploy user has Docker permissions
   - Add user to docker group: `sudo usermod -aG docker $USER`

### Debug Commands

```bash
# Check GitHub Actions logs
# Go to: GitHub Repository → Actions → Select workflow run

# Check server status
ssh user@server 'docker ps'
ssh user@server 'docker-compose ps'

# Check application logs
ssh user@server 'cd /opt/celebrum-ai-go && docker-compose logs app'
```

## Security Considerations

1. **SSH Keys**: Use dedicated SSH keys for deployment, not your personal keys
2. **Secrets**: Never commit secrets to the repository
3. **Server Access**: Limit SSH access to deployment user only
4. **Firewall**: Configure firewall to allow only necessary ports
5. **Updates**: Keep server and Docker updated regularly

## Environment-Specific Configuration

### Staging Environment
- Uses `docker-compose.yml`
- Deploys on pushes to `develop` branch
- Uses staging database and services

### Production Environment
- Uses `docker-compose.single-droplet.yml`
- Deploys on pushes to `main` branch
- Uses production database and services
- Includes additional monitoring and backup

## Support

For deployment issues:
1. Check GitHub Actions logs
2. Review server logs
3. Verify all secrets are correctly configured
4. Ensure server meets requirements