import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { logger } from 'hono/logger';
import { compress } from 'hono/compress';
import { secureHeaders } from 'hono/secure-headers';
import { validator } from 'hono/validator';
import ccxt from 'ccxt';
import type {
  HealthResponse,
  ExchangesResponse,
  TickerResponse,
  OrderBookResponse,
  OHLCVResponse,
  MarketsResponse,
  MultiTickerRequest,
  MultiTickerResponse,
  ErrorResponse,
  ExchangeManager
} from './types';

// Load environment variables
const PORT = process.env.PORT || 3001;

// Initialize Hono app
const app = new Hono();

// Middleware
app.use('*', secureHeaders());
app.use('*', cors());
app.use('*', compress());
app.use('*', logger());

// Simple rate limiting can be implemented later if needed
// For now, we rely on exchange-level rate limiting via CCXT

// Initialize supported exchanges
const exchanges: ExchangeManager = {
  binance: new ccxt.binance({ enableRateLimit: true }),
  bybit: new ccxt.bybit({ enableRateLimit: true }),
  okx: new ccxt.okx({ enableRateLimit: true }),
  coinbase: new ccxt.coinbase({ enableRateLimit: true }),
  kraken: new ccxt.kraken({ enableRateLimit: true })
};

// Health check endpoint
app.get('/health', (c) => {
  const response: HealthResponse = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'ccxt-service',
    version: '1.0.0'
  };
  return c.json(response);
});

// Get supported exchanges
app.get('/api/exchanges', (c) => {
  try {
    const exchangeList = Object.keys(exchanges).map(id => ({
      id,
      name: exchanges[id].name,
      countries: exchanges[id].countries,
      urls: exchanges[id].urls
    }));
    
    const response: ExchangesResponse = { exchanges: exchangeList };
    return c.json(response);
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Get ticker data
app.get('/api/ticker/:exchange/:symbol', async (c) => {
  try {
    const exchange = c.req.param('exchange');
    const symbol = c.req.param('symbol');
    
    if (!exchanges[exchange]) {
      const errorResponse: ErrorResponse = {
        error: 'Exchange not supported',
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 400);
    }

    const ticker = await exchanges[exchange].fetchTicker(symbol);
    const response: TickerResponse = {
      exchange,
      symbol,
      ticker,
      timestamp: new Date().toISOString()
    };
    
    return c.json(response);
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Get order book
app.get('/api/orderbook/:exchange/:symbol', 
  validator('query', (value, c) => {
    const limit = value.limit ? parseInt(value.limit as string) : 20;
    if (isNaN(limit) || limit <= 0) {
      return c.text('Invalid limit parameter', 400);
    }
    return { limit };
  }),
  async (c) => {
    try {
      const exchange = c.req.param('exchange');
      const symbol = c.req.param('symbol');
      const { limit } = c.req.valid('query');
      
      if (!exchanges[exchange]) {
        const errorResponse: ErrorResponse = {
          error: 'Exchange not supported',
          timestamp: new Date().toISOString()
        };
        return c.json(errorResponse, 400);
      }

      const orderbook = await exchanges[exchange].fetchOrderBook(symbol, limit);
      const response: OrderBookResponse = {
        exchange,
        symbol,
        orderbook,
        timestamp: new Date().toISOString()
      };
      
      return c.json(response);
    } catch (error) {
      const errorResponse: ErrorResponse = {
        error: error instanceof Error ? error.message : 'Unknown error',
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 500);
    }
  }
);

// Get OHLCV data
app.get('/api/ohlcv/:exchange/:symbol',
  validator('query', (value, c) => {
    const timeframe = (value.timeframe as string) || '1h';
    const limit = value.limit ? parseInt(value.limit as string) : 100;
    if (isNaN(limit) || limit <= 0) {
      return c.text('Invalid limit parameter', 400);
    }
    return { timeframe, limit };
  }),
  async (c) => {
    try {
      const exchange = c.req.param('exchange');
      const symbol = c.req.param('symbol');
      const { timeframe, limit } = c.req.valid('query');
      
      if (!exchanges[exchange]) {
        const errorResponse: ErrorResponse = {
          error: 'Exchange not supported',
          timestamp: new Date().toISOString()
        };
        return c.json(errorResponse, 400);
      }

      const ohlcv = await exchanges[exchange].fetchOHLCV(symbol, timeframe, undefined, limit);
      const response: OHLCVResponse = {
        exchange,
        symbol,
        timeframe,
        ohlcv,
        timestamp: new Date().toISOString()
      };
      
      return c.json(response);
    } catch (error) {
      const errorResponse: ErrorResponse = {
        error: error instanceof Error ? error.message : 'Unknown error',
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 500);
    }
  }
);

// Get multiple tickers for arbitrage
app.post('/api/tickers',
  validator('json', (value, c) => {
    const { symbols, exchanges: requestedExchanges } = value as MultiTickerRequest;
    
    if (!symbols || !Array.isArray(symbols)) {
      return c.text('Symbols array is required', 400);
    }
    
    return { symbols, exchanges: requestedExchanges };
  }),
  async (c) => {
    try {
      const { symbols, exchanges: requestedExchanges } = c.req.valid('json');
      
      const exchangesToQuery = requestedExchanges || Object.keys(exchanges);
      const results: Record<string, Record<string, any>> = {};

      for (const exchangeId of exchangesToQuery) {
        if (!exchanges[exchangeId]) continue;
        
        results[exchangeId] = {};
        
        for (const symbol of symbols) {
          try {
            const ticker = await exchanges[exchangeId].fetchTicker(symbol);
            results[exchangeId][symbol] = ticker;
          } catch (error) {
            results[exchangeId][symbol] = { 
              error: error instanceof Error ? error.message : 'Unknown error' 
            };
          }
        }
      }

      const response: MultiTickerResponse = {
        results,
        timestamp: new Date().toISOString()
      };
      
      return c.json(response);
    } catch (error) {
      const errorResponse: ErrorResponse = {
        error: error instanceof Error ? error.message : 'Unknown error',
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 500);
    }
  }
);

// Get trading pairs for an exchange
app.get('/api/markets/:exchange', async (c) => {
  try {
    const exchange = c.req.param('exchange');
    
    if (!exchanges[exchange]) {
      const errorResponse: ErrorResponse = {
        error: 'Exchange not supported',
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 400);
    }

    const markets = await exchanges[exchange].loadMarkets();
    const symbols = Object.keys(markets);
    
    const response: MarketsResponse = {
      exchange,
      symbols,
      count: symbols.length,
      timestamp: new Date().toISOString()
    };
    
    return c.json(response);
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Global error handler
app.onError((error, c) => {
  console.error('Error:', error);
  const errorResponse: ErrorResponse = {
    error: 'Internal server error',
    message: error.message,
    timestamp: new Date().toISOString()
  };
  return c.json(errorResponse, 500);
});

// 404 handler
app.notFound((c) => {
  const errorResponse: ErrorResponse = {
    error: 'Not Found',
    message: `Route ${c.req.path} not found`,
    timestamp: new Date().toISOString()
  };
  return c.json(errorResponse, 404);
});

// Start server
console.log(`🚀 CCXT Service starting on port ${PORT}`);
console.log(`📊 Supported exchanges: ${Object.keys(exchanges).join(', ')}`);

export default {
  port: PORT,
  fetch: app.fetch,
};