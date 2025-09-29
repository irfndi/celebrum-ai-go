#!/usr/bin/env bun
/**
 * Quick helper script to inspect markets and tickers from the local CCXT service.
 *
 * Usage:
 *   ADMIN_API_KEY=your-admin-key bun check-symbols.js
 *
 * Optionally override the service URL by setting CCXT_SERVICE_URL.
 */

const baseUrl = process.env.CCXT_SERVICE_URL ?? "http://localhost:3000";
const adminApiKey = process.env.ADMIN_API_KEY;

if (!adminApiKey) {
  console.error(
    "ADMIN_API_KEY is required. Set it in the environment before running this script.",
  );
  process.exit(1);
}

const defaultHeaders = {
  "x-api-key": adminApiKey,
};

/**
 * Orchestrates fetching Binance markets and optionally tests a ticker from the configured local CCXT service.
 *
 * Fetches markets from `${baseUrl}/api/markets/binance`, logs the total count, and prints up to the first 10 markets whose symbol includes "BTC" with their base and quote. If any BTC markets are found, requests a ticker for the first BTC symbol and logs the HTTP status and either the last price (`ticker.last`) on success or the error response text on failure. Uses the module's `baseUrl` and `defaultHeaders`. On any unexpected error the process exits with code 1.
 */
async function main() {
  try {
    console.log("=== Checking Binance Markets ===");
    const response = await fetch(`${baseUrl}/api/markets/binance`, {
      headers: defaultHeaders,
    });

    if (!response.ok) {
      throw new Error(
        `Failed to fetch markets: ${response.status} ${response.statusText}`,
      );
    }

    const data = await response.json();

    if (Array.isArray(data.markets)) {
      console.log("Total markets:", data.markets.length);

      const btcMarkets = data.markets
        .filter((m) => m.symbol.includes("BTC"))
        .slice(0, 10);
      console.log("\nFirst 10 BTC markets:");
      btcMarkets.forEach((m) => {
        console.log(`- ${m.symbol} (base: ${m.base}, quote: ${m.quote})`);
      });

      if (btcMarkets.length > 0) {
        const testSymbol = btcMarkets[0].symbol;
        const encodedSymbol = encodeURIComponent(testSymbol);
        console.log(`\n=== Testing ticker with symbol: ${testSymbol} ===`);

        const tickerResponse = await fetch(
          `${baseUrl}/api/ticker/binance/${encodedSymbol}`,
          {
            headers: defaultHeaders,
          },
        );
        console.log("Ticker status:", tickerResponse.status);

        if (tickerResponse.ok) {
          const tickerData = await tickerResponse.json();
          console.log("Ticker success! Price:", tickerData.ticker?.last);
        } else {
          const errorData = await tickerResponse.text();
          console.log("Ticker error:", errorData);
        }
      }
    } else {
      console.log("Unexpected response format:", data);
    }
  } catch (error) {
    console.error("Error:", error instanceof Error ? error.message : error);
    process.exit(1);
  }
}

await main();
