#!/usr/bin/env node

/**
 * Test script to verify OpenTelemetry traces are being generated
 * This script makes requests to both Go and Node.js services
 * and should show console output from both services' trace exporters
 */

const http = require('http');
const https = require('https');

// Service endpoints
const GO_SERVICE_URL = 'http://localhost:8081';
const CCXT_SERVICE_URL = 'http://localhost:3001';

// Helper function to make HTTP requests
function makeRequest(url, path = '') {
  return new Promise((resolve, reject) => {
    const fullUrl = `${url}${path}`;
    console.log(`\n🔄 Making request to: ${fullUrl}`);
    
    const client = url.startsWith('https') ? https : http;
    
    const req = client.get(fullUrl, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        console.log(`✅ Response from ${fullUrl}: ${res.statusCode}`);
        if (res.statusCode === 200) {
          try {
            const jsonData = JSON.parse(data);
            console.log(`📊 Response data:`, JSON.stringify(jsonData, null, 2));
          } catch (e) {
            console.log(`📊 Response data:`, data.substring(0, 200));
          }
        }
        resolve({ statusCode: res.statusCode, data });
      });
    });
    
    req.on('error', (err) => {
      console.log(`❌ Error making request to ${fullUrl}:`, err.message);
      reject(err);
    });
    
    req.setTimeout(5000, () => {
      console.log(`⏰ Request timeout for ${fullUrl}`);
      req.destroy();
      reject(new Error('Request timeout'));
    });
  });
}

// Test functions
async function testGoService() {
  console.log('\n🚀 Testing Go Service (celebrum-ai-go)');
  console.log('=' .repeat(50));
  
  try {
    // Test health endpoint
    await makeRequest(GO_SERVICE_URL, '/health');
    
    // Test readiness endpoint
    await makeRequest(GO_SERVICE_URL, '/ready');
    
    // Test liveness endpoint  
    await makeRequest(GO_SERVICE_URL, '/live');
    
  } catch (error) {
    console.log(`❌ Go service test failed:`, error.message);
  }
}

async function testCCXTService() {
  console.log('\n🚀 Testing CCXT Service (Node.js)');
  console.log('=' .repeat(50));
  
  try {
    // Test health endpoint
    await makeRequest(CCXT_SERVICE_URL, '/health');
    
    // Test exchanges endpoint
    await makeRequest(CCXT_SERVICE_URL, '/exchanges');
    
    // Test specific exchange ticker
    await makeRequest(CCXT_SERVICE_URL, '/exchanges/binance/ticker/BTC/USDT');
    
  } catch (error) {
    console.log(`❌ CCXT service test failed:`, error.message);
  }
}

// Main test function
async function runTests() {
  console.log('🧪 OpenTelemetry Trace Generation Test');
  console.log('=' .repeat(60));
  console.log('This script will make requests to both services.');
  console.log('Watch the console output for trace spans from both services.');
  console.log('\n📝 Expected behavior:');
  console.log('- Go service should output JSON trace spans to console');
  console.log('- Node.js service should output trace spans to console');
  console.log('- Both services should send traces to SigNoz at signoz-otel-collector:4318');
  
  // Wait a moment for user to read
  await new Promise(resolve => setTimeout(resolve, 3000));
  
  try {
    // Test both services
    await testGoService();
    await testCCXTService();
    
    console.log('\n✅ Test completed!');
    console.log('\n📋 Next steps:');
    console.log('1. Check the console output above for trace spans');
    console.log('2. Check SigNoz dashboard at http://localhost:8080');
    console.log('3. Look for traces from "celebrum-ai-go" and "ccxt-service"');
    console.log('4. If no traces appear, check Docker logs for both services');
    
  } catch (error) {
    console.log('\n❌ Test failed:', error.message);
  }
}

// Handle script termination
process.on('SIGINT', () => {
  console.log('\n👋 Test interrupted by user');
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log('\n👋 Test terminated');
  process.exit(0);
});

// Run the tests
if (require.main === module) {
  runTests().catch(console.error);
}

module.exports = { runTests, testGoService, testCCXTService };