// Simple test to check exchange initialization in test environment
process.env.ADMIN_API_KEY =
  "test-admin-key-that-is-at-least-32-characters-long-for-security";
process.env.PORT = "3003";

console.log("=== Testing Exchange Initialization ===");

// Import the service
const mod = await import("./index.ts");
const service = mod.default;

console.log("Service imported successfully");

// Wait for service to be ready with active readiness probe
async function waitForServiceReady(timeoutMs = 10000, intervalMs = 300) {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    try {
      const healthRes = await service.fetch(
        new Request("http://localhost/health"),
      );
      if (healthRes.status === 200) {
        console.log("Service is ready!");
        return;
      }
    // eslint-disable-next-line no-unused-vars
    } catch (error) {
      // Network errors are expected during startup, continue polling
    }

    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  throw new Error(`Service failed to become ready within ${timeoutMs}ms`);
}

await waitForServiceReady();

// Test ticker endpoint with error details
try {
  const res = await service.fetch(
    new Request("http://localhost/api/ticker/binance/BTC/USDT"),
  );
  console.log("Ticker endpoint status:", res.status);
  const body = await res.json();
  if (res.status !== 200) {
    console.log("Ticker error details:", JSON.stringify(body, null, 2));
  } else {
    console.log(
      "Ticker success - exchange:",
      body.exchange,
      "symbol:",
      body.symbol,
    );
  }
} catch (error) {
  console.error("Error testing ticker endpoint:", error);
}

// Test funding rates endpoint with error details
try {
  const res = await service.fetch(
    new Request("http://localhost/api/funding-rates/binance"),
  );
  console.log("Funding rates endpoint status:", res.status);
  const body = await res.json();
  if (res.status !== 200) {
    console.log("Funding rates error details:", JSON.stringify(body, null, 2));
  } else {
    console.log("Funding rates success - count:", body.count);
  }
} catch (error) {
  console.error("Error testing funding rates endpoint:", error);
}

console.log("=== Test Complete ===");
