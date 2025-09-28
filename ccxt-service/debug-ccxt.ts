import ccxt from "ccxt";

console.log("CCXT version:", ccxt.version);
console.log("Total exchanges available:", ccxt.exchanges.length);
console.log("First 10 exchanges:", ccxt.exchanges.slice(0, 10));

// Test if binance class exists
console.log("\nTesting binance:");
const BinanceClass = (ccxt as any).binance;
console.log("Binance class exists:", !!BinanceClass);
if (BinanceClass) {
  const binance = new BinanceClass();
  console.log("Binance instance created:", !!binance);
  console.log("Binance has fetchTicker:", !!binance.fetchTicker);
  console.log("Binance has fetchMarkets:", !!binance.fetchMarkets);
  console.log("Binance id:", binance.id);
  console.log("Binance name:", binance.name);
}

// Test a few more exchanges
const testExchanges = ["bybit", "okx", "coinbase", "kraken"];
for (const exchangeId of testExchanges) {
  console.log(`\nTesting ${exchangeId}:`);
  const ExchangeClass = (ccxt as any)[exchangeId];
  console.log(`${exchangeId} class exists:`, !!ExchangeClass);
  if (ExchangeClass) {
    try {
      const exchange = new ExchangeClass();
      console.log(`${exchangeId} instance created:`, !!exchange);
      console.log(`${exchangeId} has fetchTicker:`, !!exchange.fetchTicker);
      console.log(`${exchangeId} has fetchMarkets:`, !!exchange.fetchMarkets);
    } catch (error) {
      console.log(
        `${exchangeId} initialization error:`,
        (error as Error).message,
      );
    }
  }
}

// Test the exact same way as in index.ts
console.log("\n=== Testing exact same logic as index.ts ===");
const allExchanges = ccxt.exchanges;
console.log(`Total CCXT exchanges available: ${allExchanges.length}`);

const priorityExchanges = [
  "binance",
  "bybit",
  "okx",
  "coinbasepro",
  "kraken",
  "kucoin",
  "huobi",
  "gateio",
  "mexc",
  "bitget",
  "coinbase",
  "bingx",
  "cryptocom",
  "htx",
];

for (const exchangeId of priorityExchanges.slice(0, 5)) {
  // Test first 5
  console.log(`\nTesting ${exchangeId} with exact index.ts logic:`);

  // Check if exchange class exists in CCXT
  const ExchangeClass = (ccxt as any)[exchangeId];
  if (!ExchangeClass || typeof ExchangeClass !== "function") {
    console.warn(`Exchange class not found for: ${exchangeId}`);
    continue;
  }

  console.log(`✓ Exchange class found for: ${exchangeId}`);

  try {
    // Initialize the exchange
    const exchange = new ExchangeClass({
      enableRateLimit: true,
      timeout: 30000,
      rateLimit: 2000,
    });

    // Basic validation - check if exchange has required methods
    if (!exchange.fetchTicker) {
      console.warn(`Exchange ${exchangeId} missing fetchTicker method`);
      continue;
    }

    if (!exchange.fetchMarkets) {
      console.warn(`Exchange ${exchangeId} missing fetchMarkets method`);
      continue;
    }

    // Additional validation - check if exchange has basic properties
    if (!exchange.id || !exchange.name) {
      console.warn(`Exchange ${exchangeId} missing basic properties`);
      continue;
    }

    console.log(
      `✓ Successfully initialized exchange: ${exchangeId} (${exchange.name})`,
    );
  } catch (error) {
    console.warn(
      `✗ Failed to initialize exchange ${exchangeId}:`,
      error instanceof Error ? error.message : error,
    );
  }
}
