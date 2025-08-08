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

### API Endpoints

#### Health Check
- `GET /health` - Application health status

#### Market Data
- `GET /api/v1/market-data/ticker/:exchange/:symbol` - Get ticker data
- `GET /api/v1/market-data/orderbook/:exchange/:symbol` - Get order book
- `GET /api/v1/market-data/exchanges` - List supported exchanges

#### Arbitrage
- `GET /api/v1/arbitrage/opportunities` - Get current arbitrage opportunities
- `GET /api/v1/arbitrage/opportunities/:pair` - Get opportunities for specific pair

#### Technical Analysis
- `GET /api/v1/technical-analysis/indicators/:exchange/:symbol` - Get technical indicators
- `POST /api/v1/technical-analysis/calculate` - Calculate custom indicators

#### User Management
- `POST /api/v1/users/register` - Register new user
- `POST /api/v1/users/login` - User login
- `GET /api/v1/users/profile` - Get user profile

#### Alerts
- `GET /api/v1/alerts` - Get user alerts
- `POST /api/v1/alerts` - Create new alert
- `DELETE /api/v1/alerts/:id` - Delete alert

#### Telegram Bot
- `POST /api/v1/telegram/webhook` - Telegram webhook endpoint

## Configuration

The application uses a hierarchical configuration system:

1. Default values in `configs/config.yaml`
2. Environment-specific overrides
3. Environment variables (highest priority)

Key configuration sections:

- **Server**: Port, timeouts, CORS settings
- **Database**: PostgreSQL connection settings
- **Redis**: Cache configuration
- **CCXT**: Market data service settings
- **Telegram**: Bot configuration
- **Security**: JWT and rate limiting
- **Features**: Feature flags for different components

## Testing

The project maintains a minimum of 60% test coverage:

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run specific test package
go test -v ./internal/services/...
```

## Deployment

### Docker

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run
```

### Production Deployment

1. **Build for production**
   ```bash
   CGO_ENABLED=0 GOOS=linux go build -o celebrum-ai cmd/server/main.go
   ```

2. **Deploy to Digital Ocean**
   ```bash
   # Use the provided deployment scripts
   ./scripts/deploy.sh production
   ```

## Monitoring

- **Health Checks**: `/health` endpoint for service monitoring
- **Metrics**: Prometheus metrics available at `/metrics`
- **Logging**: Structured JSON logging with configurable levels

## Security

- JWT-based authentication
- Rate limiting on all endpoints
- Input validation and sanitization
- CORS protection
- Security headers

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Contact the development team
- Check the documentation in the `docs/` directory

## Roadmap

- [ ] Advanced arbitrage strategies
- [ ] Machine learning price predictions
- [ ] Mobile application
- [ ] Advanced portfolio management
- [ ] Social trading features
- [ ] Advanced risk management tools