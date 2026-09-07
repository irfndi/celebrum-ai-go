import { describe, expect, test } from "bun:test";
import { Database } from "bun:sqlite";
import {
  bucketStartMs,
  filterClosedCandles,
  insCandleSQL,
  isClosedCandle,
  TIMEFRAME_MS,
} from "./backfill-bybit-15m.ts";

describe("backfill-bybit-15m open-candle guard", () => {
  test("bucketStartMs floors to the 15m boundary", () => {
    // 12:07 -> 12:00 bucket start
    const now = Date.UTC(2026, 7, 13, 12, 7, 30);
    expect(bucketStartMs(now)).toBe(Date.UTC(2026, 7, 13, 12, 0, 0));
    expect(TIMEFRAME_MS).toBe(15 * 60 * 1000);
  });

  test("open (forming) candle is excluded", () => {
    const now = Date.UTC(2026, 7, 13, 12, 7, 30);
    const cutoff = bucketStartMs(now); // 12:00
    const closed = { timestamp: new Date(cutoff - 15 * 60 * 1000) };
    const justClosed = { timestamp: new Date(cutoff - 1) };
    const open = { timestamp: new Date(cutoff) };
    const future = { timestamp: new Date(cutoff + 15 * 60 * 1000) };

    expect(isClosedCandle(closed.timestamp.getTime(), now)).toBe(true);
    expect(isClosedCandle(justClosed.timestamp.getTime(), now)).toBe(true);
    expect(isClosedCandle(open.timestamp.getTime(), now)).toBe(false);
    expect(isClosedCandle(future.timestamp.getTime(), now)).toBe(false);

    const kept = filterClosedCandles(
      [closed, justClosed, open, future] as any,
      now,
    );
    expect(kept.map((c: any) => c.timestamp.getTime())).toEqual([
      closed.timestamp.getTime(),
      justClosed.timestamp.getTime(),
    ]);
  });

  test("upsert heals a previously frozen unfinished candle", () => {
    // Regression: INSERT OR IGNORE kept the first (unfinished) write forever.
    expect(insCandleSQL).not.toContain("INSERT OR IGNORE");
    expect(insCandleSQL).toContain("ON CONFLICT");
    expect(insCandleSQL).toContain("DO UPDATE SET");

    const db = new Database(":memory:");
    db.exec(`CREATE TABLE ohlcv_data (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      exchange_id INTEGER NOT NULL,
      trading_pair_id INTEGER NOT NULL,
      timeframe TEXT NOT NULL,
      open_price REAL NOT NULL,
      high_price REAL NOT NULL,
      low_price REAL NOT NULL,
      close_price REAL NOT NULL,
      volume REAL NOT NULL,
      timestamp DATETIME NOT NULL,
      UNIQUE(exchange_id, trading_pair_id, timeframe, timestamp)
    )`);
    const stmt = db.prepare(insCandleSQL);
    const ts = "2026-08-13T12:00:00.000Z";
    // First write: unfinished candle (narrow range, partial volume).
    stmt.run(1, 1, "15m", 100, 101, 99, 100.5, 10, ts);
    // Second write: finalized candle for the same bucket must overwrite.
    const res = stmt.run(1, 1, "15m", 100, 105, 98, 104, 50, ts) as any;
    const row = db
      .query(
        "SELECT open_price, high_price, low_price, close_price, volume FROM ohlcv_data",
      )
      .get() as any;
    expect(row.close_price).toBe(104);
    expect(row.volume).toBe(50);
    expect(row.high_price).toBe(105);
    // With the old INSERT OR IGNORE the second write would change 0 rows.
    expect(Number(res.changes)).toBeGreaterThan(0);
    db.close();
  });
});
