#!/bin/bash

# Staging Environment Setup Script
# Sets up staging environment for testing hybrid deployment approach

set -euo pipefail

# Configuration
STAGING_HOST="194.233.73.250"
STAGING_USER="root"
PROJECT_ROOT="/opt/celebrum-ai-staging"
STAGING_DOMAIN="staging.celebrum-ai.com"
STAGING_BINARY_DOMAIN="staging-binary.celebrum-ai.com"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

# Check if we have SSH access
check_ssh_access() {
    log "Checking SSH access to $STAGING_HOST..."
    if ssh -o ConnectTimeout=10 -o BatchMode=yes "$STAGING_USER@$STAGING_HOST" "echo 'SSH access successful'"; then
        log "✓ SSH access confirmed"
        return 0
    else
        error "✗ SSH access failed to $STAGING_HOST"
        return 1
    fi
}

# Update system packages
update_system() {
    log "Updating system packages on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    # Update package list
    apt-get update
    
    # Install essential packages
    apt-get install -y \
        curl \
        wget \
        git \
        htop \
        vim \
        nano \
        tree \
        jq \
        nginx \
        postgresql-client \
        redis-tools \
        supervisor \
        logrotate \
        fail2ban \
        ufw
    
    # Clean up
    apt-get autoremove -y
    apt-get autoclean
EOF
    
    log "✓ System packages updated"
}

# Setup firewall
setup_firewall() {
    log "Configuring firewall on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    # Reset firewall
    ufw --force reset
    
    # Default policies
    ufw default deny incoming
    ufw default allow outgoing
    
    # Allow essential ports
    ufw allow 22/tcp       # SSH
    ufw allow 80/tcp       # HTTP
    ufw allow 443/tcp      # HTTPS
    ufw allow 8080/tcp     # Go backend (staging)
    ufw allow 3000/tcp     # CCXT service (staging)
    ufw allow 8081/tcp     # Go backend (staging-binary)
    ufw allow 3002/tcp     # CCXT service (staging-binary)
    
    # Enable firewall
    ufw --force enable
    
    # Show status
    ufw status numbered
EOF
    
    log "✓ Firewall configured"
}

# Setup monitoring
setup_monitoring() {
    log "Setting up monitoring on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    # Create monitoring directory
    mkdir -p /opt/monitoring
    
    # Install Node Exporter for system metrics
    cd /opt/monitoring
    wget https://github.com/prometheus/node_exporter/releases/download/v1.7.0/node_exporter-1.7.0.linux-amd64.tar.gz
    tar -xzf node_exporter-1.7.0.linux-amd64.tar.gz
    mv node_exporter-1.7.0.linux-amd64/node_exporter /usr/local/bin/
    rm -rf node_exporter-1.7.0.linux-amd64*
    
    # Create Node Exporter service
    cat > /etc/systemd/system/node-exporter.service << 'EOFF'
[Unit]
Description=Node Exporter
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/node_exporter
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOFF
    
    systemctl daemon-reload
    systemctl enable node-exporter
    systemctl start node-exporter
    
    # Create basic monitoring dashboard
    mkdir -p /opt/monitoring/grafana/dashboards
EOF
    
    log "✓ Monitoring setup completed"
}

# Setup Nginx reverse proxy
setup_nginx() {
    log "Setting up Nginx reverse proxy on $STAGING_HOST..."
    
    # Create Nginx configuration for container deployment
    ssh "$STAGING_USER@$STAGING_HOST" << EOF
    # Create Nginx config for container deployment
    cat > /etc/nginx/sites-available/staging.celebrum-ai.com << 'EOFF'
server {
    listen 80;
    server_name $STAGING_DOMAIN;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Rate limiting
    limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;
    
    # Main application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # Buffer settings
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }
    
    # CCXT service
    location /ccxt/ {
        proxy_pass http://127.0.0.1:3000/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # Health check
    location /health {
        access_log off;
        proxy_pass http://127.0.0.1:8080/health;
    }
}
EOFF

    # Create Nginx config for binary deployment
    cat > /etc/nginx/sites-available/staging-binary.celebrum-ai.com << 'EOFF'
server {
    listen 80;
    server_name $STAGING_BINARY_DOMAIN;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Rate limiting
    limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;
    
    # Main application
    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # Buffer settings
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }
    
    # CCXT service
    location /ccxt/ {
        proxy_pass http://127.0.0.1:3002/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # Health check
    location /health {
        access_log off;
        proxy_pass http://127.0.0.1:8081/health;
    }
}
EOFF

    # Enable sites
    ln -sf /etc/nginx/sites-available/staging.celebrum-ai.com /etc/nginx/sites-enabled/
    ln -sf /etc/nginx/sites-available/staging-binary.celebrum-ai.com /etc/nginx/sites-enabled/
    
    # Remove default site
    rm -f /etc/nginx/sites-enabled/default
    
    # Test and restart Nginx
    nginx -t
    systemctl restart nginx
    systemctl enable nginx
EOF
    
    log "✓ Nginx reverse proxy configured"
}

# Setup SSL certificates
setup_ssl() {
    log "Setting up SSL certificates on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << EOF
    # Install Certbot
    apt-get install -y certbot python3-certbot-nginx
    
    # Request certificates (this will fail initially but shows the command)
    echo "To obtain SSL certificates, run:"
    echo "certbot --nginx -d $STAGING_DOMAIN -d $STAGING_BINARY_DOMAIN --email admin@$STAGING_DOMAIN --agree-tos --no-eff-email"
    
    # Create temporary self-signed cert for testing
    mkdir -p /etc/nginx/ssl
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /etc/nginx/ssl/staging.key -out /etc/nginx/ssl/staging.crt -subj "/C=US/ST=State/L=City/O=Organization/CN=$STAGING_DOMAIN"
    
    # Update Nginx config for SSL
    cat > /etc/nginx/snippets/ssl.conf << 'EOFF'
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers on;
EOFF

    # Update server configs to include SSL
    sed -i '/listen 80;/a\\    listen 443 ssl http2;' /etc/nginx/sites-available/staging.celebrum-ai.com
    sed -i '/listen 80;/a\\    listen 443 ssl http2;' /etc/nginx/sites-available/staging-binary.celebrum-ai.com
    
    sed -i '/server_name/a\\    ssl_certificate /etc/nginx/ssl/staging.crt;' /etc/nginx/sites-available/staging.celebrum-ai.com
    sed -i '/server_name/a\\    ssl_certificate /etc/nginx/ssl/staging.crt;' /etc/nginx/sites-available/staging-binary.celebrum-ai.com
    
    sed -i '/ssl_certificate/a\\    ssl_certificate_key /etc/nginx/ssl/staging.key;' /etc/nginx/sites-available/staging.celebrum-ai.com
    sed -i '/ssl_certificate/a\\    ssl_certificate_key /etc/nginx/ssl/staging.key;' /etc/nginx/sites-available/staging-binary.celebrum-ai.com
    
    sed -i '/ssl_certificate_key/a\\    include /etc/nginx/snippets/ssl.conf;' /etc/nginx/sites-available/staging.celebrum-ai.com
    sed -i '/ssl_certificate_key/a\\    include /etc/nginx/snippets/ssl.conf;' /etc/nginx/sites-available/staging-binary.celebrum-ai.com
    
    nginx -t
    systemctl restart nginx
EOF
    
    log "✓ SSL certificates setup completed"
}

# Setup logging and monitoring
setup_logging() {
    log "Setting up logging on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    # Create log directories
    mkdir -p /var/log/celebrum-ai
    chown -R syslog:adm /var/log/celebrum-ai
    
    # Configure log rotation
    cat > /etc/logrotate.d/celebrum-ai << 'EOFF'
/var/log/celebrum-ai/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 syslog adm
    postrotate
        systemctl reload rsyslog
    endscript
}
EOFF
    
    # Configure rsyslog for application logs
    cat > /etc/rsyslog.d/celebrum-ai.conf << 'EOFF'
template(name="CelebrumAIFileLog" type="string" string="/var/log/celebrum-ai/%programname%.log")
if \$programname == 'celebrum-ai' then ?CelebrumAIFileLog
if \$programname == 'celebrum-ccxt' then ?CelebrumAIFileLog
& stop
EOFF
    
    systemctl restart rsyslog
EOF
    
    log "✓ Logging setup completed"
}

# Create deployment user
create_deployment_user() {
    log "Creating deployment user on $STAGING_HOST..."
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    # Create deployment user
    useradd -r -s /bin/bash -d /opt/celebrum-ai celebrum || true
    
    # Add to sudo group for deployment
    usermod -aG sudo celebrum || true
    
    # Create sudoers file for passwordless deployment
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart celebrum-ai" > /etc/sudoers.d/celebrum-deploy
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart celebrum-ccxt" >> /etc/sudoers.d/celebrum-deploy
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl start celebrum-ai" >> /etc/sudoers.d/celebrum-deploy
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl start celebrum-ccxt" >> /etc/sudoers.d/celebrum-deploy
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop celebrum-ai" >> /etc/sudoers.d/celebrum-deploy
    echo "celebrum ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop celebrum-ccxt" >> /etc/sudoers.d/celebrum-deploy
    
    # Set proper permissions
    chmod 440 /etc/sudoers.d/celebrum-deploy
EOF
    
    log "✓ Deployment user created"
}

# Main setup process
main() {
    log "🚀 Setting up staging environment for hybrid deployment approach"
    log "Target: $STAGING_HOST"
    log "Domains: $STAGING_DOMAIN, $STAGING_BINARY_DOMAIN"
    
    # Check SSH access first
    if ! check_ssh_access; then
        error "Cannot proceed without SSH access"
        exit 1
    fi
    
    # Run setup steps
    update_system
    setup_firewall
    setup_monitoring
    setup_nginx
    setup_ssl
    setup_logging
    create_deployment_user
    
    # Summary
    log "🎉 Staging environment setup completed!"
    log ""
    log "=== Setup Summary ==="
    log "Server: $STAGING_HOST"
    log "Container deployment: http://$STAGING_DOMAIN"
    log "Binary deployment: http://$STAGING_BINARY_DOMAIN"
    log "SSH access: ssh $STAGING_USER@$STAGING_HOST"
    log ""
    log "=== Next Steps ==="
    log "1. Configure DNS records for both domains"
    log "2. Update GitHub secrets for SSH_PRIVATE_KEY and STAGING_SSH_HOST"
    log "3. Test deployment with both container and binary approaches"
    log "4. Set up proper SSL certificates with Let's Encrypt"
    log ""
    log "=== Service Status ==="
    ssh "$STAGING_USER@$STAGING_HOST" << 'EOF'
    echo "System status:"
    systemctl status nginx --no-pager -l
    echo
    echo "Node Exporter status:"
    systemctl status node-exporter --no-pager -l
    echo
    echo "Firewall status:"
    ufw status
EOF
}

# Run main function
main "$@"