# CCXT Service

A Node.js microservice that provides cryptocurrency market data using the CCXT library. This service acts as a bridge between the main Go application and various cryptocurrency exchanges.

## Features

- **Multi-Exchange Support**: Binance, Bybit, OKX, Coinbase Pro, Kraken
- **Market Data**: Tickers, order books, OHLCV data
- **Rate Limiting**: Built-in protection against API abuse
- **Health Checks**: Monitoring and status endpoints
- **Error Handling**: Comprehensive error responses
- **Security**: Helmet, CORS, and input validation

## API Endpoints

### Health Check
```
GET /health
```

### Exchange Information
```
GET /api/exchanges
```

### Market Data
```
GET /api/ticker/:exchange/:symbol
GET /api/orderbook/:exchange/:symbol?limit=20
GET /api/ohlcv/:exchange/:symbol?timeframe=1h&limit=100
GET /api/markets/:exchange
```

### Bulk Operations
```
POST /api/tickers
Body: {
  "symbols": ["BTC/USDT", "ETH/USDT"],
  "exchanges": ["binance", "bybit"]
}
```

## Supported Exchanges

- **binance**: Binance
- **bybit**: Bybit
- **okx**: OKX
- **coinbase**: Coinbase Pro
- **kraken**: Kraken

## Environment Variables

```bash
PORT=3001
NODE_ENV=production
RATE_LIMIT_WINDOW_MS=900000
RATE_LIMIT_MAX_REQUESTS=1000
LOG_LEVEL=info
CORS_ORIGIN=*
REQUEST_TIMEOUT=30000
```

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Start production server
npm start

# Run tests
npm test
```

## Docker

```bash
# Build image
docker build -t ccxt-service .

# Run container
docker run -p 3001:3001 ccxt-service
```

## Usage Examples

### Get Bitcoin ticker from Binance
```bash
curl http://localhost:3001/api/ticker/binance/BTC/USDT
```

### Get order book
```bash
curl http://localhost:3001/api/orderbook/binance/BTC/USDT?limit=10
```

### Get multiple tickers for arbitrage
```bash
curl -X POST http://localhost:3001/api/tickers \
  -H "Content-Type: application/json" \
  -d '{
    "symbols": ["BTC/USDT", "ETH/USDT"],
    "exchanges": ["binance", "bybit"]
  }'
```

## Error Handling

The service returns structured error responses:

```json
{
  "error": "Exchange not supported",
  "timestamp": "2024-01-15T10:30:00.000Z"
}
```

## Rate Limiting

- Default: 1000 requests per 15 minutes per IP
- Configurable via environment variables
- Returns 429 status code when exceeded

## Health Monitoring

The `/health` endpoint provides service status:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "service": "ccxt-service",
  "version": "1.0.0"
}
```