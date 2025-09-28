const ccxt = require("ccxt");

console.log("CCXT version:", ccxt.version);
console.log("Total exchanges available:", ccxt.exchanges.length);
console.log("First 10 exchanges:", ccxt.exchanges.slice(0, 10));

// Test if binance class exists
console.log("\nTesting binance:");
const BinanceClass = ccxt.binance;
console.log("Binance class exists:", !!BinanceClass);
const binance = new BinanceClass();
console.log("Binance instance created:", !!binance);
console.log("Binance has fetchTicker:", !!binance.fetchTicker);
console.log("Binance has fetchMarkets:", !!binance.fetchMarkets);
console.log("Binance id:", binance.id);
console.log("Binance name:", binance.name);

// Test a few more exchanges
const testExchanges = ["bybit", "okx", "coinbase", "kraken"];
for (const exchangeId of testExchanges) {
  console.log(`\nTesting ${exchangeId}:`);
  const ExchangeClass = ccxt[exchangeId];
  console.log(`${exchangeId} class exists:`, !!ExchangeClass);
  if (ExchangeClass) {
    try {
      const exchange = new ExchangeClass();
      console.log(`${exchangeId} instance created:`, !!exchange);
      console.log(`${exchangeId} has fetchTicker:`, !!exchange.fetchTicker);
      console.log(`${exchangeId} has fetchMarkets:`, !!exchange.fetchMarkets);
    } catch (error) {
      console.log(`${exchangeId} initialization error:`, error.message);
    }
  }
}
