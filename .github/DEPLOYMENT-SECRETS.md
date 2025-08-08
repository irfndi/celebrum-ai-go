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
  [key content lines]
  -----END OPENSSH PRIVATE KEY-----
  ```

## Setting up SSH Key for Deployment

### 1. Generate SSH Key Pair

```bash
# Generate a new SSH key pair specifically for deployment
ssh-keygen -t ed25519 -C "github-actions@celebrum-ai" -f ~/.ssh/deploy_key

# This creates two files:
# ~/.ssh/deploy_key (private key - for GitHub secrets)
# ~/.ssh/deploy_key.pub (public key - for server)
```

### 2. Add Public Key to Server

Copy the public key to your server:

```bash
# Copy public key content
cat ~/.ssh/deploy_key.pub

# On your server, add it to authorized_keys
echo "your-public-key-content" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
chmod 700 ~/.ssh
```

**Important**: Make sure you're adding the **public key** (`.pub` file) to the server, not the private key.

### 3. Add Private Key to GitHub Secrets

1. Copy the **entire private key** including headers and footers:
   ```bash
   cat ~/.ssh/deploy_key
   ```

2. In GitHub repository settings → Secrets and variables → Actions
3. Add new secret named `DEPLOY_SSH_KEY`
4. Paste the complete private key content exactly as shown:
   ```
   -----BEGIN OPENSSH PRIVATE KEY-----
   [key content lines]
   -----END OPENSSH PRIVATE KEY-----
   ```

**Critical Requirements**:
- Copy the **entire private key** including the header and footer lines
- Do NOT modify or remove any characters
- Do NOT add extra spaces or line breaks
- The key must start with `-----BEGIN OPENSSH PRIVATE KEY-----`
- The key must end with `-----END OPENSSH PRIVATE KEY-----`

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

### SSH Connection Issues

1. **"SSH key does not start with proper header" Error**:
   - This means the private key in GitHub secrets is malformed
   - **Solution**: Re-copy the private key ensuring you include the complete content:
     ```bash
     cat ~/.ssh/deploy_key
     ```
   - The key MUST start with `-----BEGIN OPENSSH PRIVATE KEY-----`
   - The key MUST end with `-----END OPENSSH PRIVATE KEY-----`
   - Do NOT copy only the middle content without headers/footers

2. **Permission denied (publickey)**:
   - Verify the **public key** (not private) is in `~/.ssh/authorized_keys` on the server
   - Check file permissions: `chmod 600 ~/.ssh/authorized_keys` and `chmod 700 ~/.ssh`
   - Ensure the private key in GitHub secrets is complete and unmodified
   - Test locally: `ssh -i ~/.ssh/deploy_key user@server`

3. **"SSH key validation failed" Error**:
   - The private key format is corrupted or incomplete
   - **Solution**: Generate a new key pair and update both server and GitHub secrets
   - Verify key integrity: `ssh-keygen -l -f ~/.ssh/deploy_key`

4. **Host key verification failed**:
   - The CI/CD pipeline automatically handles host key verification
   - If issues persist, check if the server's SSH configuration allows key-based authentication

5. **Connection timeout**:
   - Verify the server is accessible and SSH service is running
   - Check if the `DEPLOY_HOST` secret contains the correct server IP/hostname
   - Test connectivity: `ping your-server-ip`

### Key Format Verification

To verify your SSH key is properly formatted:

```bash
# Check private key format
head -1 ~/.ssh/deploy_key  # Should show: -----BEGIN OPENSSH PRIVATE KEY-----
tail -1 ~/.ssh/deploy_key  # Should show: -----END OPENSSH PRIVATE KEY-----

# Validate key can be loaded
ssh-keygen -l -f ~/.ssh/deploy_key

# Test connection
ssh -i ~/.ssh/deploy_key -o ConnectTimeout=10 user@server "echo 'Connection successful'"
```

### Docker Issues

1. **Docker not found**:
   - Ensure Docker and Docker Compose are installed on the server
   - The deployment script includes automatic Docker installation

2. **Permission denied for Docker**:
   - Add the deploy user to the docker group: `sudo usermod -aG docker $USER`
   - Restart the SSH session after adding to the group

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