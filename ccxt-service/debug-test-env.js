const ccxt = require("ccxt");

// Test binance funding rates support
const binance = new ccxt.binance();
console.log("Binance has fetchFundingRates:", binance.has["fetchFundingRates"]);
console.log("Binance has fetchFundingRate:", binance.has["fetchFundingRate"]);
console.log("Binance has fetchTicker:", binance.has["fetchTicker"]);
console.log("Binance has fetchMarkets:", binance.has["fetchMarkets"]);

// Test if binance has the required methods
console.log(
  "Binance fetchTicker method exists:",
  typeof binance.fetchTicker === "function",
);
console.log(
  "Binance fetchMarkets method exists:",
  typeof binance.fetchMarkets === "function",
);

// Check available exchanges
console.log("Total CCXT exchanges:", ccxt.exchanges.length);
console.log("First 10 exchanges:", ccxt.exchanges.slice(0, 10));
console.log("Binance in exchanges list:", ccxt.exchanges.includes("binance"));
