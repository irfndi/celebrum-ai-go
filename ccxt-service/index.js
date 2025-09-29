// Initialize OpenTelemetry tracing first
require('./tracing');

const { Hono } = require('hono');
const { cors } = require('hono/cors');
const { secureHeaders } = require('hono/secure-headers');
const ccxt = require('ccxt');
const crypto = require('crypto');
const { trace, context, SpanStatusCode, SpanKind } = require('@opentelemetry/api');
const { SEMATTRS_HTTP_METHOD, SEMATTRS_HTTP_ROUTE, SEMATTRS_HTTP_STATUS_CODE } = require('@opentelemetry/semantic-conventions');

// Get tracer instance
const tracer = trace.getTracer('ccxt-service', '1.0.0');

// Startup guards: Ensure both API_KEY and ADMIN_API_KEY are configured
if (!process.env.API_KEY) {
  console.error('ERROR: API_KEY environment variable is required but not set');
  console.error('Please set API_KEY environment variable before starting the service');
  process.exit(1);
}

if (!process.env.ADMIN_API_KEY) {
  console.error('ERROR: ADMIN_API_KEY environment variable is required but not set');
  console.error('Please set ADMIN_API_KEY environment variable before starting the service');
  process.exit(1);
}

const app = new Hono();

// Security middleware
app.use('*', secureHeaders());
app.use('*', cors({
  origin: ['http://localhost:3000', 'http://localhost:8080'],
  allowMethods: ['GET', 'POST', 'PUT', 'DELETE'],
  allowHeaders: ['Content-Type', 'Authorization', 'X-API-Key'],
}));

// Admin API Key validation middleware (only for admin endpoints)
const adminAuth = async (c, next) => {
  const span = tracer.startSpan('ccxt.auth.validate_admin_api_key', {
    kind: SpanKind.INTERNAL,
    attributes: {
      'operation.type': 'admin_authentication',
      'auth.method': 'api_key',
    },
  });

  try {
    const apiKey = c.req.header('X-API-Key');
    const expectedKey = process.env.ADMIN_API_KEY;
    
    span.setAttributes({
      'auth.api_key.present': !!apiKey,
      'auth.expected_key.present': !!expectedKey,
    });
    
    if (!apiKey || !expectedKey) {
      span.recordException(new Error('Missing API key or expected key'));
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Authentication failed' });
      return c.json({ error: 'Unauthorized' }, 401);
    }
    
    // Use timing-safe comparison to prevent timing attacks
    const providedKeyBuffer = Buffer.from(apiKey, 'utf8');
    const expectedKeyBuffer = Buffer.from(expectedKey, 'utf8');
    
    if (providedKeyBuffer.length !== expectedKeyBuffer.length || 
        !crypto.timingSafeEqual(providedKeyBuffer, expectedKeyBuffer)) {
      span.recordException(new Error('Invalid API key'));
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Authentication failed' });
      return c.json({ error: 'Unauthorized' }, 401);
    }
    
    span.setAttributes({
      'auth.result': 'success',
    });
    span.setStatus({ code: SpanStatusCode.OK });
    
    await next();
  } catch (error) {
    span.recordException(error);
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    throw error;
  } finally {
    span.end();
  }
};

// Exchange configuration
const exchangeConfigs = {
  binance: { sandbox: false, enableRateLimit: true },
  coinbase: { sandbox: false, enableRateLimit: true },
  kraken: { sandbox: false, enableRateLimit: true },
  okx: { sandbox: false, enableRateLimit: true },
  bybit: { sandbox: false, enableRateLimit: true },
};

const blacklistedExchanges = ['bitfinex2', 'bitmex'];

// Initialize exchanges with tracing
const exchanges = {};

/**
 * Initialize and populate the in-memory exchanges map from configured exchange definitions while respecting the blacklist.
 *
 * Attempts to instantiate each configured exchange (skipping blacklisted ones), stores successful instances in the `exchanges` object, and records initialization metrics on an OpenTelemetry span. Logs initialization summary and individual instantiation failures.
 *
 * @throws {Error} Re-throws any unexpected error that occurs during the initialization process.
 */
function initializeExchanges() {
  const span = tracer.startSpan('ccxt.exchanges.initialize', {
    kind: SpanKind.INTERNAL,
    attributes: {
      'operation.type': 'initialization',
      'exchanges.total_configured': Object.keys(exchangeConfigs).length,
      'exchanges.blacklisted_count': blacklistedExchanges.length,
    },
  });

  try {
    let successCount = 0;
    let errorCount = 0;

    for (const [exchangeId, config] of Object.entries(exchangeConfigs)) {
      if (blacklistedExchanges.includes(exchangeId)) {
        continue;
      }

      try {
        const ExchangeClass = ccxt[exchangeId];
        if (ExchangeClass) {
          exchanges[exchangeId] = new ExchangeClass(config);
          successCount++;
        }
      } catch (error) {
        console.error(`Failed to initialize exchange ${exchangeId}:`, error);
        errorCount++;
      }
    }

    span.setAttributes({
      'exchanges.initialized_count': successCount,
      'exchanges.failed_count': errorCount,
      'initialization.result': errorCount === 0 ? 'success' : 'partial_success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    console.log(`Initialized ${successCount} exchanges successfully`);
  } catch (error) {
    span.recordException(error);
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    throw error;
  } finally {
    span.end();
  }
}

// Initialize exchanges on startup
initializeExchanges();

// Health check endpoint
app.get('/health', (c) => {
  const span = tracer.startSpan('ccxt.health.check', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'health_check',
      'http.method': 'GET',
      'http.route': '/health',
    },
  });

  try {
    const exchangeCount = Object.keys(exchanges).length;
    const status = exchangeCount > 0 ? 'healthy' : 'unhealthy';
    
    span.setAttributes({
      'health.status': status,
      'health.exchanges_count': exchangeCount,
      'health.timestamp': new Date().toISOString(),
    });

    span.setStatus({ code: SpanStatusCode.OK });
    
    return c.json({
      status,
      timestamp: new Date().toISOString(),
      exchanges: Object.keys(exchanges),
      count: exchangeCount,
    });
  } catch (error) {
    span.recordException(error);
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    throw error;
  } finally {
    span.end();
  }
});

// Get available exchanges
app.get('/api/exchanges', (c) => {
  const span = tracer.startSpan('ccxt.exchanges.list', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'list_exchanges',
      'http.method': 'GET',
      'http.route': '/api/exchanges',
    },
  });

  try {
    const exchangeIds = Object.keys(exchanges);

    span.setAttributes({
      'exchanges.count': exchangeIds.length,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ exchanges: exchangeIds });
  } catch (error) {
    span.recordException(error);
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    throw error;
  } finally {
    span.end();
  }
});

// Get ticker data
app.get('/api/ticker/:exchange/:symbol', async (c) => {
  const exchangeId = c.req.param('exchange');
  const symbol = c.req.param('symbol');
  
  const span = tracer.startSpan('ccxt.ticker.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_ticker',
      'http.method': 'GET',
      'http.route': '/api/ticker/:exchange/:symbol',
      'exchange.id': exchangeId,
      'trading.symbol': symbol,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    if (!exchange.has.fetchTicker) {
      span.setAttributes({
        'error.type': 'feature_not_supported',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Ticker not supported' });
      return c.json({ error: 'Ticker not supported by this exchange' }, 400);
    }

    const startTime = Date.now();
    const ticker = await exchange.fetchTicker(symbol);
    const duration = Date.now() - startTime;

    span.setAttributes({
      'ticker.symbol': ticker.symbol,
      'ticker.last_price': ticker.last,
      'ticker.bid': ticker.bid,
      'ticker.ask': ticker.ask,
      'ticker.volume': ticker.baseVolume,
      'ticker.timestamp': ticker.timestamp,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ ticker, responseTime: duration });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Get order book data
app.get('/api/orderbook/:exchange/:symbol', async (c) => {
  const exchangeId = c.req.param('exchange');
  const symbol = c.req.param('symbol');
  const limit = parseInt(c.req.query('limit') || '20');
  
  const span = tracer.startSpan('ccxt.orderbook.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_orderbook',
      'http.method': 'GET',
      'http.route': '/api/orderbook/:exchange/:symbol',
      'exchange.id': exchangeId,
      'trading.symbol': symbol,
      'orderbook.limit': limit,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    if (!exchange.has.fetchOrderBook) {
      span.setAttributes({
        'error.type': 'feature_not_supported',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Order book not supported' });
      return c.json({ error: 'Order book not supported by this exchange' }, 400);
    }

    const startTime = Date.now();
    const orderbook = await exchange.fetchOrderBook(symbol, limit);
    const duration = Date.now() - startTime;

    span.setAttributes({
      'orderbook.symbol': orderbook.symbol,
      'orderbook.bids_count': orderbook.bids.length,
      'orderbook.asks_count': orderbook.asks.length,
      'orderbook.timestamp': orderbook.timestamp,
      'orderbook.best_bid': orderbook.bids[0] ? orderbook.bids[0][0] : null,
      'orderbook.best_ask': orderbook.asks[0] ? orderbook.asks[0][0] : null,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ orderbook, responseTime: duration });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Get recent trades
app.get('/api/trades/:exchange/:symbol', async (c) => {
  const exchangeId = c.req.param('exchange');
  const symbol = c.req.param('symbol');
  const limit = parseInt(c.req.query('limit') || '50');
  
  const span = tracer.startSpan('ccxt.trades.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_trades',
      'http.method': 'GET',
      'http.route': '/api/trades/:exchange/:symbol',
      'exchange.id': exchangeId,
      'trading.symbol': symbol,
      'trades.limit': limit,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    if (!exchange.has.fetchTrades) {
      span.setAttributes({
        'error.type': 'feature_not_supported',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Trades not supported' });
      return c.json({ error: 'Trades not supported by this exchange' }, 400);
    }

    const startTime = Date.now();
    const trades = await exchange.fetchTrades(symbol, undefined, limit);
    const duration = Date.now() - startTime;

    span.setAttributes({
      'trades.symbol': symbol,
      'trades.count': trades.length,
      'trades.total_volume': trades.reduce((sum, trade) => sum + trade.amount, 0),
      'trades.latest_timestamp': trades.length > 0 ? trades[trades.length - 1].timestamp : null,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ trades, responseTime: duration });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Get OHLCV data
app.get('/api/ohlcv/:exchange/:symbol', async (c) => {
  const exchangeId = c.req.param('exchange');
  const symbol = c.req.param('symbol');
  const timeframe = c.req.query('timeframe') || '1h';
  const limit = parseInt(c.req.query('limit') || '100');
  
  const span = tracer.startSpan('ccxt.ohlcv.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_ohlcv',
      'http.method': 'GET',
      'http.route': '/api/ohlcv/:exchange/:symbol',
      'exchange.id': exchangeId,
      'trading.symbol': symbol,
      'ohlcv.timeframe': timeframe,
      'ohlcv.limit': limit,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    if (!exchange.has.fetchOHLCV) {
      span.setAttributes({
        'error.type': 'feature_not_supported',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'OHLCV not supported' });
      return c.json({ error: 'OHLCV not supported by this exchange' }, 400);
    }

    const startTime = Date.now();
    const ohlcv = await exchange.fetchOHLCV(symbol, timeframe, undefined, limit);
    const duration = Date.now() - startTime;

    span.setAttributes({
      'ohlcv.symbol': symbol,
      'ohlcv.timeframe': timeframe,
      'ohlcv.count': ohlcv.length,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      exchange: exchangeId,
      symbol,
      timeframe,
      ohlcv,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Get trading pairs for an exchange
app.get('/api/markets/:exchange', async (c) => {
  const exchangeId = c.req.param('exchange');
  
  const span = tracer.startSpan('ccxt.markets.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_markets',
      'http.method': 'GET',
      'http.route': '/api/markets/:exchange',
      'exchange.id': exchangeId,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    const startTime = Date.now();
    const markets = await exchange.loadMarkets();
    const symbols = Object.keys(markets);
    const duration = Date.now() - startTime;

    span.setAttributes({
      'markets.count': symbols.length,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      exchange: exchangeId,
      symbols,
      count: symbols.length,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Get funding rates for an exchange
app.get('/api/funding-rates/:exchange', async (c) => {
  const exchangeId = c.req.param('exchange');
  const symbolsParam = c.req.query('symbols');
  const symbols = symbolsParam ? symbolsParam.split(',') : undefined;
  
  const span = tracer.startSpan('ccxt.funding_rates.fetch', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'fetch_funding_rates',
      'http.method': 'GET',
      'http.route': '/api/funding-rates/:exchange',
      'exchange.id': exchangeId,
      'funding_rates.symbols_count': symbols ? symbols.length : 0,
    },
  });

  try {
    const exchange = exchanges[exchangeId];
    if (!exchange) {
      span.setAttributes({
        'error.type': 'exchange_not_found',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Exchange not found' });
      return c.json({ error: 'Exchange not found' }, 404);
    }

    // Check if exchange supports funding rates
    if (!exchange.has.fetchFundingRates && !exchange.has.fetchFundingRate) {
      span.setAttributes({
        'error.type': 'feature_not_supported',
        'operation.result': 'error',
      });
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Funding rates not supported' });
      return c.json({ error: 'Exchange does not support funding rates' }, 400);
    }

    const startTime = Date.now();
    let fundingRates = [];

    if (symbols && symbols.length > 0) {
      // Fetch funding rates for specific symbols
      for (const symbol of symbols) {
        try {
          let fundingRate;
          if (exchange.has.fetchFundingRate) {
            fundingRate = await exchange.fetchFundingRate(symbol);
          } else if (exchange.has.fetchFundingRates) {
            // Fallback to fetchFundingRates with single symbol - verify support first
            const rates = await exchange.fetchFundingRates([symbol]);
            fundingRate = rates[symbol];
          } else {
            console.warn(`Exchange ${exchangeId} does not support funding rates for symbol ${symbol}`);
            continue;
          }
          
          if (fundingRate) {
            // Normalize the funding rate data
            const normalizedRate = {
              symbol: fundingRate.symbol || symbol,
              fundingRate: typeof fundingRate.fundingRate === 'number' ? fundingRate.fundingRate : 0,
              fundingTimestamp: fundingRate.fundingTimestamp || fundingRate.timestamp || Date.now(),
              nextFundingTime: fundingRate.nextFundingDatetime ? new Date(fundingRate.nextFundingDatetime).getTime() : 0,
              markPrice: typeof fundingRate.markPrice === 'number' ? fundingRate.markPrice : 0,
              indexPrice: typeof fundingRate.indexPrice === 'number' ? fundingRate.indexPrice : 0,
              timestamp: fundingRate.timestamp || Date.now()
            };
            fundingRates.push(normalizedRate);
          }
        } catch (error) {
          console.warn(`Failed to fetch funding rate for ${symbol} on ${exchangeId}:`, error);
        }
      }
    } else {
      // Fetch all funding rates
      try {
        if (exchange.has.fetchFundingRates) {
          const rates = await exchange.fetchFundingRates();
          // Normalize rates to array - handle both array and object responses, guard against null/undefined
          const normalizedRates = rates == null ? [] : (Array.isArray(rates) ? rates : Object.values(rates));
          fundingRates = normalizedRates.map((rate) => ({
            symbol: rate.symbol,
            fundingRate: typeof rate.fundingRate === 'number' ? rate.fundingRate : 0,
            fundingTimestamp: rate.fundingTimestamp || rate.timestamp || Date.now(),
            nextFundingTime: rate.nextFundingDatetime ? new Date(rate.nextFundingDatetime).getTime() : 0,
            markPrice: typeof rate.markPrice === 'number' ? rate.markPrice : 0,
            indexPrice: typeof rate.indexPrice === 'number' ? rate.indexPrice : 0,
            timestamp: rate.timestamp || Date.now()
          }));
        }
      } catch (error) {
        console.warn(`Failed to fetch all funding rates for ${exchangeId}:`, error);
      }
    }

    const duration = Date.now() - startTime;

    span.setAttributes({
      'funding_rates.count': fundingRates.length,
      'api.response_time_ms': duration,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      exchange: exchangeId,
      fundingRates,
      count: fundingRates.length,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Admin endpoints (require API key authentication)
app.post('/api/admin/exchanges/blacklist/:exchange', adminAuth, async (c) => {
  const exchangeId = c.req.param('exchange');
  
  const span = tracer.startSpan('ccxt.admin.blacklist_exchange', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'admin_blacklist_exchange',
      'http.method': 'POST',
      'http.route': '/api/admin/exchanges/blacklist/:exchange',
      'exchange.id': exchangeId,
    },
  });

  try {
    if (!blacklistedExchanges.includes(exchangeId)) {
      blacklistedExchanges.push(exchangeId);
      
      // Remove from active exchanges if it exists
      if (exchanges[exchangeId]) {
        delete exchanges[exchangeId];
      }
    }

    span.setAttributes({
      'blacklist.exchange_id': exchangeId,
      'blacklist.total_count': blacklistedExchanges.length,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      message: `Exchange ${exchangeId} blacklisted`,
      blacklistedExchanges,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

app.get('/api/admin/exchanges/config', adminAuth, async (c) => {
  const span = tracer.startSpan('ccxt.admin.get_config', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'admin_get_config',
      'http.method': 'GET',
      'http.route': '/api/admin/exchanges/config',
    },
  });

  try {
    span.setAttributes({
      'config.exchanges_count': Object.keys(exchangeConfigs).length,
      'config.blacklisted_count': blacklistedExchanges.length,
      'operation.result': 'success',
    });

    // Redact sensitive information from exchangeConfigs before returning
    const redactedExchangeConfigs = {};
    for (const [exchangeId, config] of Object.entries(exchangeConfigs)) {
      redactedExchangeConfigs[exchangeId] = { ...config };
      if (redactedExchangeConfigs[exchangeId].apiKey) redactedExchangeConfigs[exchangeId].apiKey = '[REDACTED]';
      if (redactedExchangeConfigs[exchangeId].secret) redactedExchangeConfigs[exchangeId].secret = '[REDACTED]';
      if (redactedExchangeConfigs[exchangeId].password) redactedExchangeConfigs[exchangeId].password = '[REDACTED]';
      if (redactedExchangeConfigs[exchangeId].uid) redactedExchangeConfigs[exchangeId].uid = '[REDACTED]';
      if (redactedExchangeConfigs[exchangeId].privateKey) redactedExchangeConfigs[exchangeId].privateKey = '[REDACTED]';
    }

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      exchangeConfigs: redactedExchangeConfigs,
      blacklistedExchanges,
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

app.post('/api/admin/exchanges/refresh', adminAuth, async (c) => {
  const span = tracer.startSpan('ccxt.admin.refresh_exchanges', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'admin_refresh_exchanges',
      'http.method': 'POST',
      'http.route': '/api/admin/exchanges/refresh',
    },
  });

  try {
    const previousCount = Object.keys(exchanges).length;
    
    // Clear existing exchanges
    for (const exchangeId in exchanges) {
      delete exchanges[exchangeId];
    }
    
    // Reinitialize exchanges
    initializeExchanges();
    
    const newCount = Object.keys(exchanges).length;

    span.setAttributes({
      'refresh.previous_count': previousCount,
      'refresh.new_count': newCount,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      message: 'Exchanges refreshed successfully',
      previousCount,
      newCount,
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

app.post('/api/admin/exchanges/add/:exchange', adminAuth, async (c) => {
  const exchangeId = c.req.param('exchange');
  const body = await c.req.json().catch(() => ({}));
  const config = body.config || { sandbox: false, enableRateLimit: true };
  
  const span = tracer.startSpan('ccxt.admin.add_exchange', {
    kind: SpanKind.SERVER,
    attributes: {
      'operation.type': 'admin_add_exchange',
      'http.method': 'POST',
      'http.route': '/api/admin/exchanges/add/:exchange',
      'exchange.id': exchangeId,
    },
  });

  try {
    // Remove from blacklist if present
    const blacklistIndex = blacklistedExchanges.indexOf(exchangeId);
    if (blacklistIndex > -1) {
      blacklistedExchanges.splice(blacklistIndex, 1);
    }
    
    // Add to config
    exchangeConfigs[exchangeId] = config;
    
    // Initialize the exchange
    try {
      const ExchangeClass = ccxt[exchangeId];
      if (ExchangeClass) {
        exchanges[exchangeId] = new ExchangeClass(config);
      } else {
        throw new Error(`Exchange class not found for ${exchangeId}`);
      }
    } catch (initError) {
      console.error(`Failed to initialize exchange ${exchangeId}:`, initError);
      throw initError;
    }

    // Redact sensitive information from config before logging
    const redactedConfig = { ...config };
    if (redactedConfig.apiKey) redactedConfig.apiKey = '[REDACTED]';
    if (redactedConfig.secret) redactedConfig.secret = '[REDACTED]';
    if (redactedConfig.password) redactedConfig.password = '[REDACTED]';
    if (redactedConfig.uid) redactedConfig.uid = '[REDACTED]';
    if (redactedConfig.privateKey) redactedConfig.privateKey = '[REDACTED]';
    
    span.setAttributes({
      'add.exchange_id': exchangeId,
      'add.config': JSON.stringify(redactedConfig),
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ 
      message: `Exchange ${exchangeId} added successfully`,
      exchangeId,
      config: redactedConfig,
      activeExchanges: Object.keys(exchanges),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    span.recordException(error);
    span.setAttributes({
      'error.type': error.constructor.name,
      'error.message': error.message,
      'operation.result': 'error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
    return c.json({ error: error.message }, 500);
  } finally {
    span.end();
  }
});

// Error handling middleware
app.onError((err, c) => {
  const span = tracer.startSpan('ccxt.error.handler', {
    kind: SpanKind.INTERNAL,
    attributes: {
      'operation.type': 'error_handling',
      'error.type': err.constructor.name,
      'error.message': err.message,
      'http.status_code': 500,
    },
  });

  span.recordException(err);
  span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
  span.end();

  console.error('Unhandled error:', err);
  return c.json({ error: 'Internal server error' }, 500);
});

const port = parseInt(process.env.PORT || '3001');

console.log(`Starting CCXT service on port ${port}`);
console.log(`Available exchanges: ${Object.keys(exchanges).join(', ')}`);

// Start the server
const serve = {
  port,
  fetch: app.fetch,
};

// For Bun compatibility
if (typeof Bun !== 'undefined') {
  module.exports = serve;
} else {
  // For Node.js
  const { serve: nodeServe } = require('@hono/node-server');
  
  nodeServe({
    fetch: app.fetch,
    port,
  });
  
  console.log(`CCXT service is running on http://localhost:${port}`);
}