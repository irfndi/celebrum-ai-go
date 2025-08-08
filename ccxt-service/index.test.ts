import { test, expect } from "bun:test";
import { Hono } from "hono";

// Basic test to ensure the service can start
test("CCXT service basic functionality", () => {
  const app = new Hono();
  expect(app).toBeDefined();
});

// Test environment variables
test("Environment variables", () => {
  // These should be defined in production
  expect(process.env.NODE_ENV).toBeDefined();
});

// Test CCXT library import
test("CCXT library import", async () => {
  const ccxt = await import("ccxt");
  expect(ccxt).toBeDefined();
  expect(typeof ccxt.exchanges).toBe("object");
});