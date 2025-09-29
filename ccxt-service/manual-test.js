// Manual test without Bun test framework
process.env.ADMIN_API_KEY = 'test-admin-key-that-is-at-least-32-characters-long-for-security';
process.env.PORT = '3003';

/**
 * Run a manual end-to-end test suite against the local service and log results.
 *
 * Performs checks of the /health, /api/exchanges, /api/markets/:exchange,
 * /api/ticker/:exchange/:base/:quote and /api/funding-rates/:exchange endpoints
 * on http://localhost, logging HTTP statuses and response bodies; any errors
 * encountered are caught and logged.
 */
async function runTests() {
  console.log('=== Manual Test Suite ===');
  
  try {
    // Import the service
    const mod = await import('./index.ts');
    const service = mod.default;
    
    console.log('Service imported successfully');
    
    // Wait for initialization
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    // Test 1: Health endpoint
    console.log('\n--- Test 1: Health endpoint ---');
    const healthRes = await service.fetch(new Request('http://localhost/health'));
    console.log('Health status:', healthRes.status);
    const healthBody = await healthRes.json();
    console.log('Health response:', healthBody);
    
    // Test 2: Exchanges endpoint
    console.log('\n--- Test 2: Exchanges endpoint ---');
    const exchangesRes = await service.fetch(new Request('http://localhost/api/exchanges'));
    console.log('Exchanges status:', exchangesRes.status);
    const exchangesBody = await exchangesRes.json();
    
    // Handle both string[] and object[] exchange formats
    let count = 0;
    let hasBinance = false;
    
    if (exchangesBody && exchangesBody.exchanges && Array.isArray(exchangesBody.exchanges)) {
      count = exchangesBody.exchanges.length;
      
      if (count > 0) {
        const firstElement = exchangesBody.exchanges[0];
        if (typeof firstElement === 'string') {
          // Handle string[] format
          hasBinance = exchangesBody.exchanges.includes('binance');
        } else if (typeof firstElement === 'object' && firstElement !== null) {
          // Handle object[] format
          hasBinance = exchangesBody.exchanges.some(item => 
            item && (item.id === 'binance' || item.name === 'binance')
          );
        }
      }
    }
    
    console.log('Exchanges count:', count);
    console.log('Has binance:', hasBinance);
    
    // Test 3: Markets endpoint
    console.log('\n--- Test 3: Markets endpoint ---');
    const marketsRes = await service.fetch(new Request('http://localhost/api/markets/binance'));
    console.log('Markets status:', marketsRes.status);
    const marketsBody = await marketsRes.json();
    console.log('Markets response:', marketsBody.exchange, marketsBody.count);
    
    // Test 4: Ticker endpoint (valid)
    console.log('\n--- Test 4: Ticker endpoint (valid) ---');
    const tickerRes = await service.fetch(new Request('http://localhost/api/ticker/binance/BTC/USDT'));
    console.log('Ticker status:', tickerRes.status);
    const tickerBody = await tickerRes.json();
    console.log('Ticker response:', tickerBody.exchange, tickerBody.symbol);
    
    // Test 5: Ticker endpoint (invalid exchange)
    console.log('\n--- Test 5: Ticker endpoint (invalid exchange) ---');
    const invalidTickerRes = await service.fetch(new Request('http://localhost/api/ticker/unknown/BTC/USDT'));
    console.log('Invalid ticker status:', invalidTickerRes.status);
    const invalidTickerBody = await invalidTickerRes.json();
    console.log('Invalid ticker response:', invalidTickerBody);
    
    // Test 6: Funding rates endpoint
    console.log('\n--- Test 6: Funding rates endpoint ---');
    const fundingRes = await service.fetch(new Request('http://localhost/api/funding-rates/binance'));
    console.log('Funding rates status:', fundingRes.status);
    const fundingBody = await fundingRes.json();
    console.log('Funding rates response:', fundingBody.count || 'error', fundingBody.error || 'success');
    
    console.log('\n=== All tests completed ===');
    
  } catch (error) {
    console.error('Test failed:', error);
  }
}

runTests();