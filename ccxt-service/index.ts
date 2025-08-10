import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { logger } from 'hono/logger';
// import { compress } from 'hono/compress'; // Removed due to CompressionStream not available in Bun
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
  ExchangeManager,
  FundingRate,
  FundingRateResponse
} from './types';

// Load environment variables
const PORT = process.env.PORT || 3001;

// Initialize Hono app
const app = new Hono();

// Middleware
app.use('*', secureHeaders());
app.use('*', cors());
// app.use('*', compress()); // Removed due to CompressionStream not available in Bun
app.use('*', logger());

// Simple rate limiting can be implemented later if needed
// For now, we rely on exchange-level rate limiting via CCXT

// Initialize supported exchanges
const exchanges: ExchangeManager = {
  binance: new ccxt.binance({ 
    enableRateLimit: true,
    timeout: 30000,
    rateLimit: 1200, // Binance rate limit
    options: {
      'defaultType': 'future' // Enable futures trading for funding rates
    }
  }),
  bybit: new ccxt.bybit({ 
    enableRateLimit: true,
    options: {
      'defaultType': 'future' // Enable futures trading for funding rates
    }
  }),
  okx: new ccxt.okx({ 
    enableRateLimit: true,
    options: {
      'defaultType': 'future' // Enable futures trading for funding rates
    }
  }),
  coinbasepro: new ccxt.coinbaseexchange({ enableRateLimit: true }),
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
      name: exchanges[id].name || id,
      countries: (exchanges[id].countries || []).filter((country): country is string => country !== undefined),
      urls: exchanges[id].urls || {}
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

    // Add retry logic for Binance
    let retries = exchange === 'binance' ? 3 : 1;
    let lastError;
    
    for (let i = 0; i < retries; i++) {
      try {
        const ticker = await exchanges[exchange].fetchTicker(symbol);
        const response: TickerResponse = {
          exchange,
          symbol,
          ticker,
          timestamp: new Date().toISOString()
        };
        
        return c.json(response);
      } catch (error) {
        lastError = error;
        if (i < retries - 1) {
          await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1))); // Exponential backoff
        }
      }
    }
    
    throw lastError;
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

// Get funding rates for an exchange
app.get('/api/funding-rates/:exchange',
  validator('query', (value, _c) => {
    const symbols = value.symbols ? (value.symbols as string).split(',') : undefined;
    return { symbols };
  }),
  async (c) => {
    try {
      const exchange = c.req.param('exchange');
      const { symbols } = c.req.valid('query');
      
      if (!exchanges[exchange]) {
        const errorResponse: ErrorResponse = {
          error: 'Exchange not supported',
          timestamp: new Date().toISOString()
        };
        return c.json(errorResponse, 400);
      }

      // Check if exchange supports funding rates
      if (!exchanges[exchange].has['fetchFundingRates'] && !exchanges[exchange].has['fetchFundingRate']) {
        const errorResponse: ErrorResponse = {
          error: 'Exchange does not support funding rates',
          timestamp: new Date().toISOString()
        };
        return c.json(errorResponse, 400);
      }

      let fundingRates: FundingRate[] = [];

      if (symbols && symbols.length > 0) {
        // Fetch funding rates for specific symbols
        for (const symbol of symbols) {
          try {
            let fundingRate;
            if (exchanges[exchange].has['fetchFundingRate']) {
              fundingRate = await exchanges[exchange].fetchFundingRate(symbol);
            } else {
              // Fallback to fetchFundingRates with single symbol
              const rates = await exchanges[exchange].fetchFundingRates([symbol]);
              fundingRate = rates[symbol];
            }
            
            if (fundingRate) {
               fundingRates.push({
                 symbol: fundingRate.symbol || symbol,
                 fundingRate: fundingRate.fundingRate || 0,
                 fundingTimestamp: fundingRate.fundingTimestamp || Date.now(),
                 nextFundingTime: fundingRate.nextFundingDatetime ? new Date(fundingRate.nextFundingDatetime).getTime() : 0,
                 markPrice: fundingRate.markPrice || 0,
                 indexPrice: fundingRate.indexPrice || 0,
                 timestamp: fundingRate.timestamp || Date.now()
               });
            }
          } catch (error) {
            console.warn(`Failed to fetch funding rate for ${symbol} on ${exchange}:`, error);
          }
        }
      } else {
        // Fetch all funding rates
        try {
          if (exchanges[exchange].has['fetchFundingRates']) {
            const rates = await exchanges[exchange].fetchFundingRates();
            fundingRates = Object.values(rates).map((rate: any) => ({
               symbol: rate.symbol,
               fundingRate: rate.fundingRate || 0,
               fundingTimestamp: rate.fundingTimestamp || Date.now(),
               nextFundingTime: rate.nextFundingDatetime ? new Date(rate.nextFundingDatetime).getTime() : 0,
               markPrice: rate.markPrice || 0,
               indexPrice: rate.indexPrice || 0,
               timestamp: rate.timestamp || Date.now()
             }));
          }
        } catch (error) {
          console.warn(`Failed to fetch all funding rates for ${exchange}:`, error);
        }
      }

      const response: FundingRateResponse = {
        exchange,
        fundingRates,
        count: fundingRates.length,
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
  idleTimeout: 30, // 30 seconds timeout to prevent Bun.serve timeout issues
};