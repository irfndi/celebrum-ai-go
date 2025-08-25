// Check available symbols for Binance
process.env.ADMIN_API_KEY = 'test-admin-key-12345678901234567890123456789012';
// Use existing service on port 3001
try {
  console.log('=== Checking Binance Markets ===');
  const response = await fetch('http://localhost:3001/api/markets/binance');
  const data = await response.json();
  
  if (data.markets) {
    console.log('Total markets:', data.markets.length);
    
    // Find BTC markets
    const btcMarkets = data.markets.filter(m => m.symbol.includes('BTC')).slice(0, 10);
    console.log('\nFirst 10 BTC markets:');
    btcMarkets.forEach(m => {
      console.log(`- ${m.symbol} (base: ${m.base}, quote: ${m.quote})`);
    });
    
    // Test ticker with correct symbol
    if (btcMarkets.length > 0) {
      const testSymbol = btcMarkets[0].symbol;
      console.log(`\n=== Testing ticker with symbol: ${testSymbol} ===`);
      
      const tickerResponse = await fetch(`http://localhost:3001/api/ticker/binance/${testSymbol}`);
      console.log('Ticker status:', tickerResponse.status);
      
      if (tickerResponse.ok) {
        const tickerData = await tickerResponse.json();
        console.log('Ticker success! Price:', tickerData.ticker?.last);
      } else {
        const errorData = await tickerResponse.text();
        console.log('Ticker error:', errorData);
      }
    }
  }
} catch (error) {
  console.error('Error:', error.message);
}

process