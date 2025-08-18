import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { logger } from 'hono/logger';
// import { compress } from 'hono/compress'; // Removed due to CompressionStream not available in Bun
import { secureHeaders } from 'hono/secure-headers';
import { validator } from 'hono/validator';
import ccxt from 'ccxt';
import { readFileSync, writeFileSync, existsSync } from 'fs';
import { join } from 'path';
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
const ADMIN_API_KEY = process.env.ADMIN_API_KEY;

// SECURITY: Validate ADMIN_API_KEY is set and not using default insecure value
if (!ADMIN_API_KEY) {
  console.error('ADMIN_API_KEY environment variable must be set');
  process.exit(1);
}

if (ADMIN_API_KEY === 'admin-secret-key-change-me' || ADMIN_API_KEY === 'admin-dev-key-change-in-production') {
  console.error('ADMIN_API_KEY cannot use default/example values. Please set a secure API key.');
  process.exit(1);
}

if (ADMIN_API_KEY.length < 32) {
  console.error('ADMIN_API_KEY must be at least 32 characters long for security');
  process.exit(1);
}

if (false) { // This condition will never be true, keeping original structure
  console.error('ERROR: ADMIN_API_KEY environment variable must be set with a secure value');
  console.error('Do not use the default value "admin-secret-key-change-me" in production');
  process.exit(1);
}

// Initialize Hono app
const app = new Hono();

// Middleware
app.use('*', secureHeaders());
app.use('*', cors());
// app.use('*', compress()); // Removed due to CompressionStream not available in Bun
app.use('*', logger());

// Authentication middleware for admin endpoints
const adminAuth = async (c: any, next: any) => {
  const authHeader = c.req.header('Authorization');
  const apiKey = c.req.header('X-API-Key');
  
  // Check for API key in Authorization header (Bearer token) or X-API-Key header
  const providedKey = authHeader?.replace('Bearer ', '') || apiKey;
  
  if (!providedKey || providedKey !== ADMIN_API_KEY) {
    return c.json({
      error: 'Unauthorized',
      message: 'Valid API key required for admin endpoints',
      timestamp: new Date().toISOString()
    }, 401);
  }
  
  await next();
};

// Simple rate limiting can be implemented later if needed
// For now, we rely on exchange-level rate limiting via CCXT

// Exchange configuration for different types of exchanges
const exchangeConfigs: Record<string, any> = {
  binance: {
    enableRateLimit: true,
    timeout: 30000,
    rateLimit: 1200,
    options: { 'defaultType': 'future' }
  },
  bybit: {
    enableRateLimit: true,
    options: { 'defaultType': 'future' }
  },
  okx: {
    enableRateLimit: true,
    options: { 'defaultType': 'future' }
  },
  coinbaseexchange: { enableRateLimit: true },
  kraken: { enableRateLimit: true },
  // Default config for other exchanges
  default: {
    enableRateLimit: true,
    timeout: 30000,
    rateLimit: 2000
  }
};

// Configuration file path
const CONFIG_FILE_PATH = join(process.cwd(), 'exchange-config.json');

// Default exchange configuration
const defaultExchangeConfig = {
  blacklistedExchanges: [
    'test', 'mock', 'sandbox', 'demo', 'testnet',
    'coinbaseprime', // Use coinbaseexchange instead
    'ftx', 'ftxus', // Defunct exchanges
    'liquid', 'quoine', // Defunct exchanges
    'idex', 'ethfinex', // Deprecated exchanges
    'yobit', 'livecoin', 'coinfloor', // Problematic exchanges
    'southxchange', 'coinmate', 'lakebtc', // Often unreliable
  ]
};

// Load configuration from file or use defaults
function loadExchangeConfig() {
  try {
    if (existsSync(CONFIG_FILE_PATH)) {
      const configData = readFileSync(CONFIG_FILE_PATH, 'utf8');
      const config = JSON.parse(configData);
      console.log('Loaded exchange configuration from file');
      return config;
    }
  } catch (error) {
    console.warn('Failed to load exchange configuration from file, using defaults:', error);
  }
  return { ...defaultExchangeConfig };
}

// Save configuration to file
function saveExchangeConfig(config: any) {
  try {
    writeFileSync(CONFIG_FILE_PATH, JSON.stringify(config, null, 2), 'utf8');
    console.log('Saved exchange configuration to file');
    return true;
  } catch (error) {
    console.error('Failed to save exchange configuration:', error);
    return false;
  }
}

// Load exchange configuration
const exchangeConfig = loadExchangeConfig();

// Convert to Set for faster lookups during initialization
const blacklistedExchanges = new Set(exchangeConfig.blacklistedExchanges);

// Priority exchanges (will be initialized first)
const priorityExchanges = [
  'binance', 'bybit', 'okx', 'coinbasepro', 'kraken',
  'kucoin', 'huobi', 'gateio', 'mexc', 'bitget',
  'coinbaseexchange', 'bingx', 'cryptocom', 'htx'
];

// Initialize supported exchanges dynamically
const exchanges: ExchangeManager = {};

// Function to initialize an exchange
function initializeExchange(exchangeId: string): boolean {
  try {
    if (blacklistedExchanges.has(exchangeId)) {
      console.log(`Skipping blacklisted exchange: ${exchangeId}`);
      return false;
    }

    // Check if exchange class exists in CCXT
    const ExchangeClass = (ccxt as any)[exchangeId];
    if (!ExchangeClass || typeof ExchangeClass !== 'function') {
      console.warn(`Exchange class not found for: ${exchangeId}`);
      return false;
    }

    // Get configuration for this exchange
    const config = exchangeConfigs[exchangeId] || exchangeConfigs.default;
    
    // Initialize the exchange
    const exchange = new ExchangeClass(config);
    
    // Basic validation - check if exchange has required methods
    if (!exchange.fetchTicker) {
      console.warn(`Exchange ${exchangeId} missing fetchTicker method`);
      return false;
    }
    
    if (!exchange.fetchMarkets) {
      console.warn(`Exchange ${exchangeId} missing fetchMarkets method`);
      return false;
    }

    // Additional validation - check if exchange has basic properties
    if (!exchange.id || !exchange.name) {
      console.warn(`Exchange ${exchangeId} missing basic properties`);
      return false;
    }

    exchanges[exchangeId] = exchange;
    console.log(`✓ Successfully initialized exchange: ${exchangeId} (${exchange.name})`);
    return true;
  } catch (error) {
    console.warn(`✗ Failed to initialize exchange ${exchangeId}:`, error instanceof Error ? error.message : error);
    return false;
  }
}

// Get all available exchanges from CCXT
const allExchanges = ccxt.exchanges; // ccxt.exchanges is an array of exchange names
console.log(`Total CCXT exchanges available: ${allExchanges.length}`);
console.log(`Blacklisted exchanges: ${Array.from(blacklistedExchanges).join(', ')}`);

// Initialize priority exchanges first
let initializedCount = 0;
let failedCount = 0;
const failedExchanges: string[] = [];

for (const exchangeId of priorityExchanges) {
  if (initializeExchange(exchangeId)) {
    initializedCount++;
  } else {
    failedCount++;
    failedExchanges.push(exchangeId);
  }
}

console.log(`Priority exchanges initialized: ${initializedCount}/${priorityExchanges.length}`);

// Initialize remaining exchanges
for (const exchangeId of allExchanges) {
  if (!exchanges[exchangeId] && !priorityExchanges.includes(exchangeId)) {
    if (initializeExchange(exchangeId)) {
      initializedCount++;
    } else {
      failedCount++;
      failedExchanges.push(exchangeId);
    }
  }
}

console.log(`Successfully initialized ${Object.keys(exchanges).length} out of ${allExchanges.length} total exchanges`);
console.log(`Failed to initialize ${failedCount} exchanges:`, failedExchanges.slice(0, 10).join(', '), failedCount > 10 ? `... and ${failedCount - 10} more` : '');
console.log(`Active exchanges:`, Object.keys(exchanges).sort().join(', '));

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
  validator('query', (value, c) => {
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



// Exchange management endpoints

// Add exchange to blacklist
app.post('/api/admin/exchanges/blacklist/:exchange', adminAuth, async (c) => {
  try {
    const exchange = c.req.param('exchange');
    
    if (!exchangeConfig.blacklistedExchanges.includes(exchange)) {
      exchangeConfig.blacklistedExchanges.push(exchange);
      blacklistedExchanges.add(exchange);
      
      // Save configuration to file
      const saved = saveExchangeConfig(exchangeConfig);
      if (!saved) {
        console.warn('Failed to persist blacklist changes to file');
      }
      
      // Remove from active exchanges if it exists
      if (exchanges[exchange]) {
        delete exchanges[exchange];
      }
      
      console.log(`Exchange ${exchange} added to blacklist`);
    }
    
    return c.json({
      message: `Exchange ${exchange} blacklisted successfully`,
      blacklistedExchanges: exchangeConfig.blacklistedExchanges,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Remove exchange from blacklist
app.delete('/api/admin/exchanges/blacklist/:exchange', adminAuth, async (c) => {
  try {
    const exchange = c.req.param('exchange');
    
    const index = exchangeConfig.blacklistedExchanges.indexOf(exchange);
    if (index > -1) {
      exchangeConfig.blacklistedExchanges.splice(index, 1);
      blacklistedExchanges.delete(exchange);
      
      // Save configuration to file
      const saved = saveExchangeConfig(exchangeConfig);
      if (!saved) {
        console.warn('Failed to persist blacklist changes to file');
      }
      
      // Try to initialize the exchange if it's available
      try {
        initializeExchange(exchange);
        console.log(`Exchange ${exchange} removed from blacklist and initialized`);
      } catch (error) {
        console.warn(`Failed to initialize ${exchange} after removing from blacklist:`, error);
      }
    }
    
    return c.json({
      message: `Exchange ${exchange} removed from blacklist`,
      blacklistedExchanges: exchangeConfig.blacklistedExchanges,
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Get exchange configuration
app.get('/api/admin/exchanges/config', adminAuth, async (c) => {
  return c.json({
    config: exchangeConfig,
    activeExchanges: Object.keys(exchanges),
    availableExchanges: ccxt.exchanges,
    timestamp: new Date().toISOString()
  });
});

// Refresh exchanges (re-initialize all non-blacklisted exchanges)
app.post('/api/admin/exchanges/refresh', adminAuth, async (c) => {
  try {
    // Clear current exchanges
    Object.keys(exchanges).forEach(key => delete exchanges[key]);
    
    // Re-initialize priority exchanges first
    let initializedCount = 0;
    for (const exchangeId of priorityExchanges) {
      if (initializeExchange(exchangeId)) {
        initializedCount++;
      }
    }
    
    // Re-initialize remaining exchanges
    const allExchanges = ccxt.exchanges;
    for (const exchangeId of allExchanges) {
      if (!exchanges[exchangeId] && !priorityExchanges.includes(exchangeId)) {
        if (initializeExchange(exchangeId)) {
          initializedCount++;
        }
      }
    }
    
    console.log(`Refreshed and initialized ${initializedCount} exchanges`);
    
    return c.json({
      message: 'Exchanges refreshed successfully',
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    const errorResponse: ErrorResponse = {
      error: error instanceof Error ? error.message : 'Unknown error',
      timestamp: new Date().toISOString()
    };
    return c.json(errorResponse, 500);
  }
});

// Add new exchange dynamically
app.post('/api/admin/exchanges/add/:exchange', adminAuth, async (c) => {
  try {
    const exchange = c.req.param('exchange');
    
    // Check if exchange is available in CCXT
    if (!ccxt.exchanges.includes(exchange)) {
      const errorResponse: ErrorResponse = {
        error: `Exchange ${exchange} is not available in CCXT library`,
        availableExchanges: ccxt.exchanges,
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 400);
    }
    
    // Check if already blacklisted
    if (exchangeConfig.blacklistedExchanges.includes(exchange)) {
      const errorResponse: ErrorResponse = {
        error: `Exchange ${exchange} is blacklisted. Remove from blacklist first.`,
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 400);
    }
    
    // Try to initialize the exchange
    const success = initializeExchange(exchange);
    if (!success) {
      const errorResponse: ErrorResponse = {
        error: `Failed to initialize exchange ${exchange}`,
        timestamp: new Date().toISOString()
      };
      return c.json(errorResponse, 500);
    }
    
    return c.json({
      message: `Exchange ${exchange} added successfully`,
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
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