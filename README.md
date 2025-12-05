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

3. **Start development environment (Docker)**
   ```bash
   # Starts App, Postgres, and Redis in Docker
   make dev-up-orchestrated
   ```

4. **Alternative: Run locally (Go binary)**
   ```bash
   # Start only dependencies (Postgres/Redis)
   make dev-setup
   
   # Run migrations
   make db-migrate
   
   # Run app
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
├── scripts/             # Build and deployment scripts
├── docs/                # Documentation
└── tests/               # Test files
```

## Deployment

### 🚀 Coolify Deployment (Recommended)

This project is optimized for deployment via [Coolify](https://coolify.io/).

- **Architecture**: Single Docker container (Go + Bun/CCXT)
- **Database**: Managed PostgreSQL (via Coolify)
- **Cache**: Managed Redis (via Coolify)
- **Routing**: Automated via Coolify's Traefik

For detailed deployment instructions, please refer to **[DEPLOYMENT.md](docs/deployment/DEPLOYMENT.md)**.

### Automated CI/CD

The project includes a CI/CD pipeline using GitHub Actions that automatically:
- Runs tests, linting, and formatting on every push
- Builds the application to ensure integrity

Deployment is triggered via Coolify webhooks.
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

1. Default values in `config.yml`
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
- **Sentry**: Enable error and performance reporting by setting `sentry.*` in `config.yml` (Go API) and `SENTRY_DSN` for the Bun CCXT service

## Security

- JWT-based authentication
- Rate limiting on API endpoints
- SSL/TLS encryption
- Environment variable management

## License

MIT License - see LICENSE file for details
# Automated deployment sync: Fri Oct  3 13:48:00 WIB 2025
