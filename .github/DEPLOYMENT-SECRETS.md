# Deployment Secrets Setup

This document explains how to set up the required GitHub secrets for automated deployment.

## Required Secrets

The CI/CD pipeline requires the following secrets to be configured in your GitHub repository:

### 1. DIGITALOCEAN_ACCESS_TOKEN
- **Description**: DigitalOcean API token for container registry access
- **How to get**: 
  1. Go to DigitalOcean Control Panel → API → Tokens/Keys
  2. Generate a new Personal Access Token
  3. Copy the token value

### 2. DEPLOY_SSH_KEY
- **Description**: Private SSH key for server access
- **Format**: Complete private key including headers
- **Example format**:
  ```
  -----BEGIN OPENSSH PRIVATE KEY-----
  [key content]
  -----END OPENSSH PRIVATE KEY-----
  ```
- **How to generate**:
  ```bash
  # Generate new SSH key pair
  ssh-keygen -t ed25519 -C "github-actions@celebrum-ai" -f deploy_key
  
  # Copy private key content (this goes to GitHub secret)
  # IMPORTANT: Copy the ENTIRE content including headers and footers
  cat deploy_key
  ```

**Important Notes for GitHub Secrets:**
- Copy the **entire** private key including `-----BEGIN OPENSSH PRIVATE KEY-----` and `-----END OPENSSH PRIVATE KEY-----`
- Do NOT modify the key content or add extra spaces/newlines
- The key should be pasted exactly as shown by `cat deploy_key`

**Setting up the public key on server:**
```bash
# On your server, create SSH directory if it doesn't exist
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# Add the public key to authorized_keys
echo "your-public-key-content" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys

# To get the public key content:
cat deploy_key.pub
```

### 3. DEPLOY_USER
- **Description**: SSH username for server access
- **Example**: `root` or `deploy`

### 4. DEPLOY_HOST
- **Description**: Server IP address or hostname
- **Example**: `143.198.219.213` or `celebrum-ai.example.com`

## Setting Up Secrets in GitHub

1. Go to your repository on GitHub
2. Navigate to Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Add each secret with the exact name and value

## Server Setup Requirements

Ensure your deployment server has:

1. **Docker and Docker Compose installed**
2. **Project directory**: `/opt/celebrum-ai-go`
3. **SSH access configured** with the public key
4. **Docker Compose file**: `docker-compose.single-droplet.yml`

## Troubleshooting

### SSH Key Issues
- Ensure the private key includes proper headers and footers
- Check that the public key is added to `~/.ssh/authorized_keys` on the server
- Verify the SSH user has proper permissions

### Permission Issues
- Ensure the deploy user has Docker permissions: `sudo usermod -aG docker $USER`
- Check that `/opt/celebrum-ai-go` directory exists and is writable

### Connection Issues
- Verify the server IP/hostname is correct
- Check that SSH port 22 is open
- Test SSH connection manually: `ssh user@host`

## Manual Testing

Test the deployment setup manually:

```bash
# Test SSH connection
ssh -i deploy_key user@host

# Test Docker access
ssh -i deploy_key user@host "docker ps"

# Test deployment directory
ssh -i deploy_key user@host "ls -la /opt/celebrum-ai-go"
```