#!/bin/bash

# Celebrum AI Environment Setup Script
# This script sets up the environment for different deployment scenarios

set -euo pipefail

trap 'error_handler $? $LINENO' ERR

function error_handler() {
  echo "Error: command failed with exit code $1 at line $2"
  exit 1
}

# --- Docker Compose Command Detection ---
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    echo "Error: Neither docker-compose nor docker compose found. Please install one of them." >&2
    exit 1
fi
# --------------------------------------

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="development"
FORCE=false
SKIP_BUILD=false

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -e, --environment   Environment to set up (development, staging, production)"
    echo "  -f, --force         Force overwrite existing configuration"
    echo "  -s, --skip-build    Skip building Docker images"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -e development"
    echo "  $0 -e production -f"
    echo "  $0 --environment staging --skip-build"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -s|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
    print_error "Invalid environment: $ENVIRONMENT"
    print_error "Valid environments: development, staging, production"
    exit 1
fi

print_status "Setting up Celebrum AI for environment: $ENVIRONMENT"

# Create necessary directories
print_status "Creating necessary directories..."
mkdir -p logs
mkdir -p configs/ssl
mkdir -p configs/nginx

# Copy environment-specific configuration
print_status "Setting up configuration files..."

# Copy .env template if it doesn't exist
if [[ ! -f ".env" ]] || [[ "$FORCE" == "true" ]]; then
    if [[ -f ".env.template" ]]; then
        cp .env.template .env
        print_status "Created .env file from template"
        print_warning "Please edit .env file with your actual values"
    else
        print_error ".env.template not found"
        exit 1
    fi
else
    print_status ".env file already exists, skipping"
fi

# Copy configuration files
CONFIG_FILE="configs/config.$ENVIRONMENT.yml"
    if [[ "$ENVIRONMENT" == "development" ]]; then
        CONFIG_FILE="configs/config.local.yml"
    fi
if [[ -f "$CONFIG_FILE" ]]; then
    if [[ ! -f "configs/config.yml" ]] || [[ "$FORCE" == "true" ]]; then
        cp "$CONFIG_FILE" configs/config.yml
        print_status "Copied $CONFIG_FILE to configs/config.yml"
    else
        print_status "configs/config.yml already exists, skipping"
    fi
else
    print_error "Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# Set up Docker Compose files based on environment
case $ENVIRONMENT in
    development)
        print_status "Setting up for development..."
        print_status "Use: $COMPOSE_CMD up -d"
        print_status "Optional development tools: $COMPOSE_CMD --profile dev-tools up -d"
        ;;
    staging)
        print_status "Setting up for staging..."
        print_status "Use: $COMPOSE_CMD -f docker-compose.yml -f docker-compose.staging.yml up -d"
        ;;
    production)
        print_status "Setting up for production..."
        print_status "Use: $COMPOSE_CMD -f docker-compose.yml -f docker-compose.prod.yml up -d"
        ;;
esac

# Build Docker images if not skipped
if [[ "$SKIP_BUILD" != "true" ]]; then
    print_status "Building Docker images..."
    case $ENVIRONMENT in
        development)
            $COMPOSE_CMD build
            ;;
        staging)
            $COMPOSE_CMD -f docker-compose.yml -f docker-compose.staging.yml build
            ;;
        production)
            $COMPOSE_CMD -f docker-compose.yml -f docker-compose.prod.yml build
            ;;
    esac
fi

# Create SSL certificates for development (self-signed)
if [[ "$ENVIRONMENT" == "development" ]]; then
    if ! command -v openssl &> /dev/null; then
        print_warning "openssl command not found. Skipping SSL certificate generation."
        print_warning "Please install openssl to generate certificates for development."
    else
        print_status "Creating self-signed SSL certificates for development..."
        if [[ ! -f "configs/ssl/server.crt" ]] || [[ "$FORCE" == "true" ]]; then
            openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
                -keyout configs/ssl/server.key \
                -out configs/ssl/server.crt \
                -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
            print_status "Created self-signed SSL certificates"
        else
            print_status "SSL certificates already exist, skipping"
        fi
    fi
fi

# Create nginx configuration
print_status "Setting up nginx configuration..."
if [[ ! -f "configs/nginx.conf" ]] || [[ "$FORCE" == "true" ]]; then
    case $ENVIRONMENT in
        development)
            cp configs/nginx.dev.conf configs/nginx.conf 2>/dev/null || print_warning "nginx.dev.conf not found, using default"
            ;;
        staging)
            cp configs/nginx.staging.conf configs/nginx.conf 2>/dev/null || print_warning "nginx.staging.conf not found, using default"
            ;;
        production)
            cp configs/nginx.prod.conf configs/nginx.conf 2>/dev/null || print_warning "nginx.prod.conf not found, using default"
            ;;
    esac
fi

# Final instructions
echo ""
print_status "Environment setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env file with your actual configuration values"
echo "2. For development: $COMPOSE_CMD up -d"
echo "3. For production: $COMPOSE_CMD -f docker-compose.yml -f docker-compose.prod.yml up -d"
echo ""
echo "Available services:"
echo "- PostgreSQL: localhost:5432"
echo "- Redis: localhost:6379"
echo "- CCXT Service: localhost:3001"
echo "- Main App: localhost:8080"
echo ""
echo "For development tools (Adminer, Redis Commander):"
echo "$COMPOSE_CMD --profile dev-tools up -d"
echo ""
echo "Adminer: http://localhost:8081"
echo "Redis Commander: http://localhost:8082"