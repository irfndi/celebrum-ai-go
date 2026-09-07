import { describe, expect, it } from "bun:test";
import { Database } from "bun:sqlite";
import { mkdtempSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  DEFAULT_EXCHANGE,
  DEFAULT_TIMEFRAME,
  HOLDOUT_BARS,
  PHASE_GEOM,
  computePanelHash,
  datasetsMatch,
  evaluateHoldoutOnPanel,
  evaluateKnobsOnPanel,
  isProvenanceCompatible,
  loadAlignedPanel,
  selectionRefLen,
  toDatasetProvenance,
  type AlignedPanel,
} from "./prepare.ts";
import { CLAIM_BARS, meetsClaimBars } from "./goals.ts";
import { knobs as seedKnobs } from "./knobs.ts";
import type { Candle } from "../src/market-data/types.ts";

// ---------- synthetic panel helpers ----------

function oscCandles(
  n: number,
  mid: number,
  amp: number,
  symbol: string,
): Candle[] {
  const out: Candle[] = [];
  for (let i = 0; i < n; i++) {
    const side = i % 2 === 0 ? 1 : -1;
    out.push({
      exchange: "bybit-futures",
      symbol,
      timeframe: "5m",
      open: mid - (amp / 2) * side,
      high: mid + amp * side + 0.05,
      low: mid - amp * side - 0.05,
      close: mid + (amp / 2) * side,
      volume: 1,
      timestamp: new Date(i * 15 * 60 * 1000),
    });
  }
  return out;
}

function synthPanel(n: number, holdoutBars: number): AlignedPanel {
  const aligned = new Map<string, Candle[]>([
    ["A", oscCandles(n, 100, 0.6, "A")],
    ["B", oscCandles(n, 200, 0.9, "B")],
    ["C", oscCandles(n, 300, 0.5, "C")],
    ["D", oscCandles(n, 400, 0.7, "D")],
  ]);
  return {
    symbols: ["A", "B", "C", "D"],
    aligned,
    refLen: n,
    loadedMs: 1,
    exchange: "bybit-futures",
    timeframe: "5m",
    panelTimeframe: "15m",
    panelHash: "synth",
    holdoutBars: holdoutBars,
  };
}

/** Rewrite the reserved tail as a steep uptrend; selection must not notice. */
function poisonTail(panel: AlignedPanel): AlignedPanel {
  const holdout = panel.holdoutBars;
  const aligned = new Map<string, Candle[]>();
  for (const [s, cs] of panel.aligned) {
    const cp = cs.map((c) => ({ ...c }));
    let px = cp[cp.length - holdout - 1]!.close;
    for (let i = cp.length - holdout; i < cp.length; i++) {
      px *= 1.02;
      cp[i] = {
        ...cp[i]!,
        open: px / 1.01,
        high: px * 1.005,
        low: px / 1.015,
        close: px,
      };
    }
    aligned.set(s, cp);
  }
  return { ...panel, aligned };
}

// ---------- venue DB helpers ----------

function seedVenueDb(path: string): void {
  const db = new Database(path);
  db.exec(`CREATE TABLE exchanges (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
    CREATE TABLE trading_pairs (id INTEGER PRIMARY KEY AUTOINCREMENT, exchange_id INTEGER NOT NULL,
      symbol TEXT NOT NULL, base_currency TEXT NOT NULL, quote_currency TEXT NOT NULL,
      UNIQUE(exchange_id, symbol));
    CREATE TABLE ohlcv_data (id INTEGER PRIMARY KEY AUTOINCREMENT, exchange_id INTEGER NOT NULL,
      trading_pair_id INTEGER NOT NULL, timeframe TEXT NOT NULL, open_price NUMERIC NOT NULL,
      high_price NUMERIC NOT NULL, low_price NUMERIC NOT NULL, close_price NUMERIC NOT NULL,
      volume NUMERIC NOT NULL, timestamp DATETIME NOT NULL,
      UNIQUE(exchange_id, trading_pair_id, timeframe, timestamp));`);
  db.exec(
    `INSERT INTO exchanges (id, name) VALUES (1, 'bybit-futures'), (2, 'binance');`,
  );
  db.exec(`INSERT INTO trading_pairs (id, exchange_id, symbol, base_currency, quote_currency)
    VALUES (1, 1, 'BTC/USDT:USDT', 'BTC', 'USDT'),
           (2, 2, 'BTC/USDT:USDT', 'BTC', 'USDT'),
           (3, 1, 'BTC/USDT', 'BTC', 'USDT');`);
  // Same wire text, different venues/prices; plus an alias text on bybit.
  const specs = [
    { pair: 1, venue: 1, price: 100 },
    { pair: 2, venue: 2, price: 10000 },
    { pair: 3, venue: 1, price: 200 },
  ];
  const base = Date.UTC(2026, 0, 1);
  const ins = db.prepare(
    `INSERT INTO ohlcv_data (exchange_id, trading_pair_id, timeframe, open_price, high_price,
      low_price, close_price, volume, timestamp) VALUES (?, ?, '5m', ?, ?, ?, ?, 1, ?)`,
  );
  const txn = db.transaction(() => {
    for (const s of specs) {
      for (let i = 0; i < 30; i++) {
        const ts = new Date(base + i * 5 * 60 * 1000).toISOString();
        ins.run(
          s.venue,
          s.pair,
          s.price,
          s.price * 1.001,
          s.price * 0.999,
          s.price,
          ts,
        );
      }
    }
  });
  txn();
  db.close();
}

function median(xs: number[]): number {
  const s = [...xs].sort((a, b) => a - b);
  return s[Math.floor(s.length / 2)]!;
}

// ---------- tests ----------

describe("venue provenance", () => {
  it("defaults to the live venue", () => {
    expect(DEFAULT_EXCHANGE).toBe("bybit-futures");
    expect(DEFAULT_TIMEFRAME).toBe("5m");
    expect(HOLDOUT_BARS).toBe(2880);
  });

  it("loads one venue only: no cross-exchange merge, no alias merge", () => {
    const dir = mkdtempSync(join(tmpdir(), "ar-venue-"));
    try {
      const dbPath = join(dir, "t.db");
      seedVenueDb(dbPath);
      const panel = loadAlignedPanel({
        symbols: 8,
        dbPath,
        minCandles: 4,
        exchange: "bybit-futures",
        timeframe: "5m",
      });
      expect(panel.exchange).toBe("bybit-futures");
      expect(panel.timeframe).toBe("5m");
      expect(panel.holdoutBars).toBe(HOLDOUT_BARS);
      // Alias texts stay distinct symbols.
      expect(panel.symbols).toContain("BTC/USDT:USDT");
      expect(panel.symbols).toContain("BTC/USDT");
      // Bybit wire prices survive unmixed (binance ~10000 must not leak in).
      const perps = panel.aligned.get("BTC/USDT:USDT")!;
      expect(median(perps.map((c) => c.close))).toBeCloseTo(100, 0);
      const spotAlias = panel.aligned.get("BTC/USDT")!;
      expect(median(spotAlias.map((c) => c.close))).toBeCloseTo(200, 0);
      for (const cs of panel.aligned.values()) {
        for (const c of cs) expect(c.exchange).toBe("bybit-futures");
      }
      // Binance loads its own dataset with its own hash.
      const binance = loadAlignedPanel({
        symbols: 8,
        dbPath,
        minCandles: 4,
        exchange: "binance",
        timeframe: "5m",
      });
      expect(
        median(binance.aligned.get("BTC/USDT:USDT")!.map((c) => c.close)),
      ).toBeCloseTo(10000, -2);
      expect(binance.panelHash).not.toBe(panel.panelHash);
      expect(
        datasetsMatch(toDatasetProvenance(panel), toDatasetProvenance(binance)),
      ).toBe(false);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  it("panel hash is stable per dataset and binds the champion", () => {
    const dir = mkdtempSync(join(tmpdir(), "ar-hash-"));
    try {
      const dbPath = join(dir, "t.db");
      seedVenueDb(dbPath);
      const opts = {
        symbols: 8,
        dbPath,
        minCandles: 4,
        exchange: "bybit-futures",
        timeframe: "5m",
      };
      const a = loadAlignedPanel(opts);
      const b = loadAlignedPanel(opts);
      expect(a.panelHash).toBe(b.panelHash);
      expect(
        datasetsMatch(toDatasetProvenance(a), toDatasetProvenance(b)),
      ).toBe(true);
      expect(isProvenanceCompatible(toDatasetProvenance(a), b)).toBe(true);
      expect(isProvenanceCompatible(null, b)).toBe(false);
      const moved = { ...toDatasetProvenance(a), panelHash: "deadbeef" };
      expect(isProvenanceCompatible(moved, b)).toBe(false);
      expect(
        computePanelHash({
          exchange: "bybit-futures",
          timeframe: "5m",
          panelTimeframe: "15m",
          symbols: ["X"],
          refLen: 10,
          t0Ms: 1,
          t1Ms: 2,
        }),
      ).not.toBe(
        computePanelHash({
          exchange: "binance",
          timeframe: "5m",
          panelTimeframe: "15m",
          symbols: ["X"],
          refLen: 10,
          t0Ms: 1,
          t1Ms: 2,
        }),
      );
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe("frozen holdout", () => {
  it("selection windows end before the reserved tail", () => {
    const panel = synthPanel(1600, 120);
    expect(selectionRefLen(panel)).toBe(1480);
    for (const phase of ["screen", "confirm"] as const) {
      const geom = PHASE_GEOM[phase];
      const lastStart = selectionRefLen(panel) - geom.forwardBars - 1;
      // Last selection window ends strictly before the holdout tail.
      expect(lastStart + geom.forwardBars).toBeLessThanOrEqual(
        panel.refLen - panel.holdoutBars,
      );
    }
  });

  it("screen/confirm ignore a poisoned tail while the holdout sees it", () => {
    const clean = synthPanel(1600, 120);
    const poisoned = poisonTail(clean);
    const screenClean = evaluateKnobsOnPanel(seedKnobs, clean, {
      phase: "screen",
      maxSteps: 50,
      budgetSec: 60,
    });
    const screenPoisoned = evaluateKnobsOnPanel(seedKnobs, poisoned, {
      phase: "screen",
      maxSteps: 50,
      budgetSec: 60,
    });
    expect(screenClean.windows).toBeGreaterThan(0);
    expect(screenPoisoned.score).toBe(screenClean.score);
    expect(screenPoisoned.windows).toBe(screenClean.windows);
    const holdoutClean = evaluateHoldoutOnPanel(seedKnobs, clean, {
      budgetSec: 60,
    });
    const holdoutPoisoned = evaluateHoldoutOnPanel(seedKnobs, poisoned, {
      budgetSec: 60,
    });
    expect(holdoutClean.phase).toBe("holdout");
    expect(holdoutClean.windows).toBe(clean.symbols.length);
    expect(holdoutPoisoned.score).not.toBe(holdoutClean.score);
  });

  it("tiny panels fail closed on the holdout instead of claiming", () => {
    const dir = mkdtempSync(join(tmpdir(), "ar-small-"));
    try {
      const dbPath = join(dir, "t.db");
      seedVenueDb(dbPath);
      const panel = loadAlignedPanel({
        symbols: 8,
        dbPath,
        minCandles: 4,
        exchange: "bybit-futures",
        timeframe: "5m",
      });
      const holdout = evaluateHoldoutOnPanel(seedKnobs, panel, {
        budgetSec: 10,
      });
      expect(holdout.phase).toBe("holdout");
      expect(holdout.guardsOk).toBe(false);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe("claim gates unchanged", () => {
  it("keeps v2 bars (0.08 / 0.001)", () => {
    expect(CLAIM_BARS.minMedianLogReturn).toBe(0.08);
    expect(CLAIM_BARS.minExpectancyPct).toBe(0.001);
  });

  it("meetsClaimBars accepts a passing holdout-shaped result", () => {
    const passing = {
      medianLogReturn: 0.09,
      winRatePct: 56,
      medianDrawdownPct: 9,
      tradesPerSymMonth: 5,
      expectancyPct: 0.002,
    };
    expect(meetsClaimBars(passing)).toBe(true);
    expect(meetsClaimBars({ ...passing, expectancyPct: 0.0005 })).toBe(false);
    expect(meetsClaimBars({ ...passing, medianLogReturn: 0.079 })).toBe(false);
  });
});
