#!/usr/bin/env bun

/**
 * Bun-based test script to verify OpenTelemetry traces
 * Tests both Go service and CCXT service endpoints
 */

interface TestResult {
  endpoint: string;
  status: number;
  success: boolean;
  responseTime: number;
  error?: string;
}

class TraceTestRunner {
  private goServiceUrl = 'http://localhost:8081';
  private ccxtServiceUrl = 'http://localhost:3001';
  private results: TestResult[] = [];

  async testGoService(): Promise<void> {
    console.log('\n🔍 Testing Go Service Endpoints...');
    
    const endpoints = [
      '/health',
      '/ready', 
      '/live'
    ];

    for (const endpoint of endpoints) {
      await this.makeRequest('Go Service', `${this.goServiceUrl}${endpoint}`);
    }
  }

  async testCCXTService(): Promise<void> {
    console.log('\n🔍 Testing CCXT Service Endpoints...');
    
    const endpoints = [
      '/health',
      '/exchanges',
      '/exchanges/binance/ticker/BTC/USDT'
    ];

    for (const endpoint of endpoints) {
      await this.makeRequest('CCXT Service', `${this.ccxtServiceUrl}${endpoint}`);
    }
  }

  private async makeRequest(serviceName: string, url: string): Promise<void> {
    const startTime = Date.now();
    
    try {
      console.log(`\n📡 ${serviceName}: ${url}`);
      
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': 'Bun-TraceTest/1.0.0'
        }
      });
      
      const responseTime = Date.now() - startTime;
      const success = response.ok;
      
      console.log(`   Status: ${response.status} ${response.statusText}`);
      console.log(`   Response Time: ${responseTime}ms`);
      
      if (success) {
        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
          const data = await response.json();
          console.log(`   Response: ${JSON.stringify(data).substring(0, 200)}...`);
        } else {
          const text = await response.text();
          console.log(`   Response: ${text.substring(0, 200)}...`);
        }
      } else {
        console.log(`   ❌ Request failed`);
      }
      
      this.results.push({
        endpoint: url,
        status: response.status,
        success,
        responseTime
      });
      
    } catch (error) {
      const responseTime = Date.now() - startTime;
      console.log(`   ❌ Error: ${error}`);
      
      this.results.push({
        endpoint: url,
        status: 0,
        success: false,
        responseTime,
        error: String(error)
      });
    }
    
    // Small delay between requests
    await new Promise(resolve => setTimeout(resolve, 500));
  }

  printSummary(): void {
    console.log('\n📊 Test Summary:');
    console.log('================');
    
    const successful = this.results.filter(r => r.success).length;
    const total = this.results.length;
    
    console.log(`Total Requests: ${total}`);
    console.log(`Successful: ${successful}`);
    console.log(`Failed: ${total - successful}`);
    console.log(`Success Rate: ${((successful / total) * 100).toFixed(1)}%`);
    
    console.log('\nDetailed Results:');
    this.results.forEach(result => {
      const status = result.success ? '✅' : '❌';
      console.log(`${status} ${result.endpoint} - ${result.status} (${result.responseTime}ms)`);
      if (result.error) {
        console.log(`   Error: ${result.error}`);
      }
    });
  }

  printTraceInstructions(): void {
    console.log('\n🔍 OpenTelemetry Trace Verification:');
    console.log('====================================');
    console.log('1. Check Docker logs for console trace output:');
    console.log('   docker-compose logs --tail=20 app');
    console.log('   docker-compose logs --tail=20 ccxt-service');
    console.log('');
    console.log('2. Check SigNoz dashboard at: http://localhost:3301');
    console.log('   - Navigate to Services tab');
    console.log('   - Look for "celebrum-go-app" and "ccxt-service"');
    console.log('   - Check Traces tab for recent traces');
    console.log('');
    console.log('3. If no traces appear:');
    console.log('   - Verify OTLP collector is running: docker-compose logs signoz-otel-collector');
    console.log('   - Check service configurations in docker-compose.yml');
    console.log('   - Ensure OTEL_EXPORTER_OTLP_ENDPOINT is correctly set');
  }
}

async function main(): Promise<void> {
  console.log('🚀 Starting OpenTelemetry Trace Test with Bun');
  console.log('==============================================');
  
  const testRunner = new TraceTestRunner();
  
  try {
    // Test both services
    await testRunner.testGoService();
    await testRunner.testCCXTService();
    
    // Print results
    testRunner.printSummary();
    testRunner.printTraceInstructions();
    
    console.log('\n✅ Test completed! Check the instructions above to verify traces.');
    
  } catch (error) {
    console.error('❌ Test failed:', error);
    process.exit(1);
  }
}

// Run the test
if (import.meta.main) {
  main();
}