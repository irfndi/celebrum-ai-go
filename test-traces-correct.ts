#!/usr/bin/env bun

console.log("🚀 Testing OpenTelemetry with Correct Endpoints");
console.log("=============================================");

async function testEndpoints() {
  const endpoints = [
    "http://localhost:8081/api/v1/exchanges/supported",
    "http://localhost:3001/api/ticker/binance/BTCUSDT",
    "http://localhost:3001/api/ticker/kraken/ETHUSD",
    "http://localhost:3001/api/ticker/coinbase/BTCUSD"
  ];

  console.log("🔍 Testing endpoints to generate traces...");
  
  for (const endpoint of endpoints) {
    try {
      const start = Date.now();
      const response = await fetch(endpoint);
      const duration = Date.now() - start;
      
      console.log(`📡 ${endpoint}`);
      console.log(`   Status: ${response.status} ${response.statusText}`);
      console.log(`   Response Time: ${duration}ms`);
      
      if (response.ok) {
        console.log(`   ✅ Success`);
      } else {
        console.log(`   ❌ Request failed`);
      }
      console.log();
      
      // Small delay between requests
      await new Promise(resolve => setTimeout(resolve, 500));
      
    } catch (error) {
      console.log(`❌ Error: ${error.message}`);
    }
  }
}

testEndpoints().then(() => {
  console.log("✅ Test completed! Check SigNoz dashboard at http://localhost:3301");
  console.log("   - Navigate to Services tab");
  console.log("   - Look for 'celebrum-go-app' and 'ccxt-service'");
  console.log("   - Check Traces tab for recent traces");
});
