const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const compression = require('compression');
const rateLimit = require('express-rate-limit');
const ccxt = require('ccxt');
require('dotenv').config();

const app = express();
const PORT = process.env.PORT || 3001;

// Security middleware
app.use(helmet());
app.use(cors());
app.use(compression());
app.use(morgan('combined'));
app.use(express.json({ limit: '10mb' }));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 1000, // limit each IP to 1000 requests per windowMs
  message: 'Too many requests from this IP, please try again later.'
});
app.use('/api/', limiter);

// Initialize supported exchanges
const exchanges = {
  binance: new ccxt.binance({ enableRateLimit: true }),
  bybit: new ccxt.bybit({ enableRateLimit: true }),
  okx: new ccxt.okx({ enableRateLimit: true }),
  coinbase: new ccxt.coinbasepro({ enableRateLimit: true }),
  kraken: new ccxt.kraken({ enableRateLimit: true })
};

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'ccxt-service',
    version: '1.0.0'
  });
});

// Get supported exchanges
app.get('/api/exchanges', (req, res) => {
  try {
    const exchangeList = Object.keys(exchanges).map(id => ({
      id,
      name: exchanges[id].name,
      countries: exchanges[id].countries,
      urls: exchanges[id].urls
    }));
    res.json({ exchanges: exchangeList });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get ticker data
app.get('/api/ticker/:exchange/:symbol', async (req, res) => {
  try {
    const { exchange, symbol } = req.params;
    
    if (!exchanges[exchange]) {
      return res.status(400).json({ error: 'Exchange not supported' });
    }

    const ticker = await exchanges[exchange].fetchTicker(symbol);
    res.json({
      exchange,
      symbol,
      ticker,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get order book
app.get('/api/orderbook/:exchange/:symbol', async (req, res) => {
  try {
    const { exchange, symbol } = req.params;
    const limit = parseInt(req.query.limit) || 20;
    
    if (!exchanges[exchange]) {
      return res.status(400).json({ error: 'Exchange not supported' });
    }

    const orderbook = await exchanges[exchange].fetchOrderBook(symbol, limit);
    res.json({
      exchange,
      symbol,
      orderbook,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get OHLCV data
app.get('/api/ohlcv/:exchange/:symbol', async (req, res) => {
  try {
    const { exchange, symbol } = req.params;
    const timeframe = req.query.timeframe || '1h';
    const limit = parseInt(req.query.limit) || 100;
    
    if (!exchanges[exchange]) {
      return res.status(400).json({ error: 'Exchange not supported' });
    }

    const ohlcv = await exchanges[exchange].fetchOHLCV(symbol, timeframe, undefined, limit);
    res.json({
      exchange,
      symbol,
      timeframe,
      ohlcv,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get multiple tickers for arbitrage
app.post('/api/tickers', async (req, res) => {
  try {
    const { symbols, exchanges: requestedExchanges } = req.body;
    
    if (!symbols || !Array.isArray(symbols)) {
      return res.status(400).json({ error: 'Symbols array is required' });
    }

    const exchangesToQuery = requestedExchanges || Object.keys(exchanges);
    const results = {};

    for (const exchangeId of exchangesToQuery) {
      if (!exchanges[exchangeId]) continue;
      
      results[exchangeId] = {};
      
      for (const symbol of symbols) {
        try {
          const ticker = await exchanges[exchangeId].fetchTicker(symbol);
          results[exchangeId][symbol] = ticker;
        } catch (error) {
          results[exchangeId][symbol] = { error: error.message };
        }
      }
    }

    res.json({
      results,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get trading pairs for an exchange
app.get('/api/markets/:exchange', async (req, res) => {
  try {
    const { exchange } = req.params;
    
    if (!exchanges[exchange]) {
      return res.status(400).json({ error: 'Exchange not supported' });
    }

    const markets = await exchanges[exchange].loadMarkets();
    const symbols = Object.keys(markets);
    
    res.json({
      exchange,
      symbols,
      count: symbols.length,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Error handling middleware
app.use((error, req, res, next) => {
  console.error('Error:', error);
  res.status(500).json({
    error: 'Internal server error',
    message: error.message,
    timestamp: new Date().toISOString()
  });
});

// 404 handler
app.use('*', (req, res) => {
  res.status(404).json({
    error: 'Endpoint not found',
    path: req.originalUrl,
    timestamp: new Date().toISOString()
  });
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`CCXT Service running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`Supported exchanges: ${Object.keys(exchanges).join(', ')}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully');
  process.exit(0);
});