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

- **Backend**: Go 1.21+ with Gin web framework
- **Database**: PostgreSQL 15+ with Redis for caching
- **Market Data**: CCXT (Node.js service) for exchange integration
- **Deployment**: Docker containers on Digital Ocean
- **Monitoring**: Prometheus metrics and health checks

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for CCXT service)

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
make lint              # Run linter
make fmt               # Format code
make run               # Run the application
make dev               # Run with hot reload
make dev-setup         # Setup development environment
make dev-down          # Stop development environment
make install-tools     # Install development tools
make security          # Run security scan
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

### SSH Connection and Deployment

After pushing your changes to GitHub, connect to your server and deploy:

```bash
# Connect to your Digital Ocean droplet
ssh root@your-server-ip

# Navigate to your project directory
cd /path/to/celebrum-ai-go

# Pull latest changes
git pull origin main

# Run the deployment script
./scripts/deploy.sh production
```

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run
```

### Production Deployment Script

The deployment script (`scripts/deploy.sh`) handles:
- Pulling latest code from GitHub
- Building Docker containers
- Running database migrations
- Health checks
- Rollback on failure

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