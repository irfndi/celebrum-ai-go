#!/bin/bash
# setup-github-secrets.sh - Helper script to set up GitHub Secrets

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    print_error "GitHub CLI (gh) is not installed. Please install it first:"
    echo "  brew install gh  # macOS"
    echo "  apt install gh    # Ubuntu/Debian"
    echo "  See: https://cli.github.com/"
    exit 1
fi

# Check if user is logged in
if ! gh auth status &> /dev/null; then
    print_error "Please login to GitHub CLI first:"
    echo "  gh auth login"
    exit 1
fi

# Get repository information
REPO=$(gh repo view --json nameWithOwner -q '.nameWithOwner')
print_info "Setting up secrets for repository: $REPO"

# Function to set secret if not already set
set_secret() {
    local secret_name=$1
    local secret_value=$2
    
    if gh secret list | grep -q "$secret_name"; then
        print_warning "Secret '$secret_name' already exists. Skipping..."
    else
        echo "$secret_value" | gh secret set "$secret_name"
        print_info "Set secret: $secret_name"
    fi
}

# Generate random secrets
print_info "Generating secure secrets..."
JWT_SECRET=$(openssl rand -base64 64 | tr -d '\r\n')
TELEGRAM_WEBHOOK_SECRET=$(openssl rand -hex 32 | tr -d '\r\n')

# Interactive setup
print_info "Setting up GitHub Secrets..."
echo ""

# JWT Secret
set_secret "JWT_SECRET" "$JWT_SECRET"

# Telegram Bot Token (interactive)
read -p "Enter your Telegram Bot Token (from @BotFather): " TELEGRAM_BOT_TOKEN
if [[ -n "$TELEGRAM_BOT_TOKEN" ]]; then
    set_secret "TELEGRAM_BOT_TOKEN" "$TELEGRAM_BOT_TOKEN"
fi

# Telegram Webhook URL (interactive)
read -p "Enter your Telegram Webhook URL (e.g., https://yourdomain.com/webhook): " TELEGRAM_WEBHOOK_URL
if [[ -n "$TELEGRAM_WEBHOOK_URL" ]]; then
    set_secret "TELEGRAM_WEBHOOK_URL" "$TELEGRAM_WEBHOOK_URL"
fi

# Telegram Webhook Secret
set_secret "TELEGRAM_WEBHOOK_SECRET" "$TELEGRAM_WEBHOOK_SECRET"

# Feature flags
set_secret "FEATURE_TELEGRAM_BOT" "true"

# DigitalOcean (interactive)
read -p "Enter your DigitalOcean Access Token: " DIGITALOCEAN_TOKEN
if [[ -n "$DIGITALOCEAN_TOKEN" ]]; then
    set_secret "DIGITALOCEAN_ACCESS_TOKEN" "$DIGITALOCEAN_TOKEN"
fi

# Deployment SSH Key (interactive)
read -p "Enter your server hostname/IP: " DEPLOY_HOST
if [[ -n "$DEPLOY_HOST" ]]; then
    set_secret "DEPLOY_HOST" "$DEPLOY_HOST"
fi

read -p "Enter your SSH username: " DEPLOY_USER
if [[ -n "$DEPLOY_USER" ]]; then
    set_secret "DEPLOY_USER" "$DEPLOY_USER"
fi

echo ""
print_info "SSH Key Setup Instructions:"
echo "1. Generate SSH key pair on your server:"
echo "   ssh-keygen -t rsa -b 4096 -f ~/.ssh/celebrum_deploy"
echo ""
echo "2. Add public key to authorized_keys:"
echo "   cat ~/.ssh/celebrum_deploy.pub >> ~/.ssh/authorized_keys"
echo ""
echo "3. Copy private key and set as GitHub secret:"
echo "   cat ~/.ssh/celebrum_deploy | gh secret set DEPLOY_SSH_KEY"
echo ""

# Summary
echo ""
print_info "Setup Summary:"
print_info "Repository: $REPO"
print_info "Required secrets configured: JWT_SECRET, TELEGRAM_WEBHOOK_SECRET, FEATURE_TELEGRAM_BOT"
print_info "Optional secrets: Configure TELEGRAM_BOT_TOKEN, TELEGRAM_WEBHOOK_URL, DIGITALOCEAN_ACCESS_TOKEN"
print_info "SSH secrets: Configure DEPLOY_HOST, DEPLOY_USER, and DEPLOY_SSH_KEY"
print_info ""
print_info "To verify secrets:"
print_info "  gh secret list"
print_info ""
print_info "To update a secret:"
print_info "  echo 'new-value' | gh secret set SECRET_NAME"