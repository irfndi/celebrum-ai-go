// Debug CCXT in test environment
process.env.ADMIN_API_KEY = 'test-admin-key-that-is-at-least-32-characters-long-for-security';
process.env.PORT = '3003';

console.log('=== Testing CCXT import in test environment ===');

try {
  const ccxt = require('ccxt');
  console.log('CCXT version:', ccxt.version);
  console.log('Total exchanges available:', ccxt.exchanges.length);
  console.log('First 10 exchanges:', ccxt.exchanges.slice(0, 10));
  
  // Test specific exchanges
  const testExchanges = ['binance', 'bybit', 'okx', 'kraken', 'coinbase'];
  
  for (const exchangeId of testExchanges) {
    try {
      const ExchangeClass = ccxt[exchangeId];
      if (ExchangeClass && typeof ExchangeClass === 'function') {
        const exchange = new ExchangeClass({ enableRateLimit: true });
        console.log(`✓ ${exchangeId}: id=${exchange.id}, name=${exchange.name}`);
        console.log(`  - fetchTicker: ${typeof exchange.fetchTicker}`);
        console.log(`  - fetchMarkets: ${typeof exchange.fetchMarkets}`);
      } else {
        console.log(`✗ ${exchangeId}: Class not found or not a function`);
      }
    } catch (error) {
      console.log(`✗ ${exchangeId}: Error -`, error.message);
    }
  }
  
} catch (error) {
  console.error('Failed to import CCXT:', error);
}

console.log('\n=== Now testing index.ts import ===');

try {
  // Import the service
  const mod = require('./index.ts');
  console.log('Service imported successfully');
  console.log('Service type:', typeof mod.default);
  
  // Wait a bit for initialization
  setTimeout(() => {
    console.log('Test completed');
    process.exit(0);
  }, 3000);
  
} catch (error) {
  console.error('Failed to import service:', error);
  process.exit(1);
}