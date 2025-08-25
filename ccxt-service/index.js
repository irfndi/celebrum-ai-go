// Initialize OpenTelemetry tracing first
require('./tracing');

const { Hono } = require('hono');
const { cors } = require('hono/cors');
const { secureHeaders } = require('hono/secure-headers');
const ccxt = require('ccxt');
const { trace, context, SpanStatusCode, SpanKind } = require('@opentelemetry/api');
const { SEMATTRS_HTTP_METHOD, SEMATTRS_HTTP_ROUTE, SEMATTRS_HTTP_STATUS_CODE } = require('@opentelemetry/semantic-conventions');

// Get tracer instance
const tracer = trace.getTracer('ccxt-service', '1.0.0');

const app = new Hono();

// Security middleware
app.use('*', secureHeaders());
app.use('*', cors({
  origin: ['http://localhost:3000', 'http://localhost:8080'],
  allowMethods: ['GET', 'POST', 'PUT', 'DELETE'],
  allowHeaders: ['Content-Type', 'Authorization', 'X-API-Key'],
}));

// API Key validation middleware
app.use('/api/*', async (c, next) => {
  const span = tracer.startSpan('ccxt.auth.validate_api_key', {
    kind: SpanKind.INTERNAL,
    attributes: {
      'operation.type': 'authentication',
      'auth.method': 'api_key',
    },
  });

  try {
    const apiKey = c.req.header('X-API-Key');
    const expectedKey = process.env.API_KEY || 'your-secret-api-key';
    
    span.setAttributes({
      'auth.api_key.present': !!apiKey,
      'auth.expected_key.present': !!expectedKey,
    });
    
    if (!apiKey || apiKey !== expectedKey) {
      span.recordException(new Error('Invalid or missing API key'));
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
});

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
    const exchangeList = Object.keys(exchanges).map(id => ({
      id,
      name: exchanges[id].name,
      countries: exchanges[id].countries,
      has: {
        fetchTicker: exchanges[id].has.fetchTicker,
        fetchOrderBook: exchanges[id].has.fetchOrderBook,
        fetchTrades: exchanges[id].has.fetchTrades,
      },
    }));

    span.setAttributes({
      'exchanges.count': exchangeList.length,
      'operation.result': 'success',
    });

    span.setStatus({ code: SpanStatusCode.OK });
    return c.json({ exchanges: exchangeList });
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