# Celebrum AI - Crypto Arbitrage & Technical Analysis Platform

A comprehensive cryptocurrency arbitrage detection and technical analysis platform built with Go, featuring real-time market data collection, arbitrage opportunity identification, and technical indicator calculations.

## Features

- **Real-time Market Data**: Integration with 100+ cryptocurrency exchanges via CCXT
- **Arbitrage Detection**: Automated identification of profitable arbitrage opportunities
- **Technical Analysis**: Advanced technical indicators (RSI, MACD, SMA, EMA, Bollinger Bands)
- **Multi-Interface Support**: REST API, Telegram Bot, and Web Interface
- **High Performance**: Built with Go for optimal performance and concurrency
- **Scalable Architecture**: Microservices design with Redis caching and PostgreSQL storage

## Tech Stack

- **Backend**: Go 1.25+ with Gin web framework
- **Database**: PostgreSQL 15+ with Redis for caching
- **Market Data**: CCXT (Bun service) for exchange integration
- **Deployment**: Docker containers on Digital Ocean
- **Monitoring**: Prometheus metrics and health checks

## Quick Start

### Prerequisites

- Go 1.25 or higher
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Bun 1.0+ (for CCXT service)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/irfndi/celebrum-ai-go.git
   cd celebrum-ai-go
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start development environment**
   ```bash
   make dev-setup
   ```

4. **Install dependencies**
   ```bash
   go mod download
   ```

5. **Run the application**
   ```bash
   make run
   ```

## Development

### Available Make Commands

```bash
make help              # Show all available commands
make build             # Build the application
make test              # Run tests
make test-coverage     # Run tests with coverage report
make coverage-check    # Compute coverage across core packages (warn if <80%)
make lint              # Run linter
make fmt               # Format code
make run               # Run the application
make dev               # Run with hot reload
make dev-setup         # Setup development environment
make dev-down          # Stop development environment
make install-tools     # Install development tools
make security          # Run security scan
make ci-check          # Run all CI checks locally
make ci-lint           # Run CI linter
make ci-test           # Run CI tests
make ci-build          # Build for CI
```

### Development Workflow

#### Pre-commit Hooks (Recommended)

Install pre-commit hooks to catch issues before committing:

```bash
# Install pre-commit (requires Python)
pip install pre-commit

# Install the git hook scripts
pre-commit install

# Run against all files (optional)
pre-commit run --all-files
```

The pre-commit hooks will automatically:
- Format Go code
- Run linting
- Run tests
- Check for secrets
- Validate YAML/JSON files
- Check Dockerfile syntax

#### Code Quality

```bash
# Before committing, run:
make fmt               # Format code
make lint              # Check for issues
make test              # Run tests
make ci-check          # Run full CI suite
```

### Project Structure

```
.
├── cmd/
│   ├── server/          # Main application entry point
│   └── worker/          # Background workers
├── internal/
│   ├── api/             # HTTP handlers and routes
│   ├── config/          # Configuration management
│   ├── database/        # Database connections and operations
│   ├── models/          # Data models
│   └── services/        # Business logic
├── pkg/
│   ├── ccxt/            # CCXT service client
│   └── utils/           # Utility functions
├── api/                 # API documentation
├── configs/             # Configuration files
├── scripts/             # Build and deployment scripts
├── docs/                # Documentation
└── tests/               # Test files
```

## Deployment

### 🚀 Enhanced Deployment & Monitoring

The project now includes comprehensive deployment automation and monitoring:

- **Zero-downtime deployments** with automatic rollback
- **Health monitoring** for all services (PostgreSQL, Redis, CCXT, Telegram)
- **Automated recovery** for failed services
- **Comprehensive logging** and alerting
- **Backup automation** before deployments

### 📚 New Documentation

- **[Deployment Improvements Summary](docs/DEPLOYMENT_IMPROVEMENTS_SUMMARY.md)** - Complete overview of all fixes and improvements
- **[Telegram Bot Troubleshooting](docs/TELEGRAM_BOT_TROUBLESHOOTING.md)** - Common issues and solutions
- **[Deployment Best Practices](docs/DEPLOYMENT_BEST_PRACTICES.md)** - Comprehensive deployment guide

### Automated CI/CD Pipeline

The project includes a complete CI/CD pipeline using GitHub Actions that automatically:

- **Runs tests and linting** on every push and pull request
- **Builds and pushes Docker images** to GitHub Container Registry
- **Deploys to production** when code is pushed to the main branch
- **Performs health checks** and automatic rollback on failure

#### Setup CI/CD

1. **Configure GitHub Secrets** (see [.github/DEPLOYMENT.md](.github/DEPLOYMENT.md)):
   - `DIGITALOCEAN_ACCESS_TOKEN`
   - `DEPLOY_SSH_KEY`
   - `DEPLOY_USER`
   - `DEPLOY_HOST`

2. **Push to main branch** - deployment happens automatically!

#### Local CI Checks

```bash
# Run the same checks as CI
make ci-check

# Individual CI commands
make ci-lint      # Run linter
make ci-test      # Run tests with race detection
make ci-build     # Build with version info
```

### Quick Deployment Commands

```bash
# Deploy to production (zero-downtime)
make deploy

# Deploy to staging
make deploy-staging

# Check production health
make health-prod

# Check service status
make status-prod

# Monitor all services
./scripts/health-check.sh

# Auto-recover failed services
./scripts/health-check.sh --recover
```

### Manual Deployment

#### SSH Connection and Deployment

```bash
# Connect to your Digital Ocean droplet
ssh root@your-server-ip

# Navigate to your project directory
cd /opt/celebrum-ai-go

# Pull latest changes
git pull origin main

# Run the deployment script
./scripts/deploy.sh production
```

#### Docker Deployment (via Makefile)

```bash
# Build and run with Compose using Make targets
make docker-build     # build images
make docker-run       # start stack (Compose)

# Check service status
docker compose ps

# View logs
docker compose logs -f app

# Stop services
make dev-down         # or: docker compose down
```

### Production Deployment Script

The deployment script (`scripts/deploy.sh`) handles:
- Pulling latest code from GitHub
- Building Docker containers
- Running database migrations
- Health checks
- Automatic rollback on failure
- Backup creation before deployment

## Configuration

The application uses a hierarchical configuration system:

1. Default values in `configs/config.yaml`
2. Environment-specific overrides
3. Environment variables (highest priority)

## Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run specific test package
go test -v ./internal/services/...
```

## Monitoring

- **Health Checks**: `/health` endpoint for service monitoring
- **Metrics**: Prometheus metrics available at `/metrics`
- **Logging**: Structured JSON logging with configurable levels

## Security

- JWT-based authentication
- Rate limiting on API endpoints
- SSL/TLS encryption
- Environment variable management

## License

MIT License - see LICENSE file for details
# Automated deployment sync: Fri Oct  3 13:48:00 WIB 2025
