/**
 * Growth E1 sizing sweep — additive-only analysis tests (no DB, no CLAIM).
 *
 * Covers: holdout-tail-only scoring (poisoned prefix is invisible),
 * honest taker-exit fees, DD-budget -> account-kill mapping, ruin/growth
 * math, frozen-base immutability, and markdown reporting.
 */
import { describe, expect, it } from "bun:test";
import {
  HONEST_MAKER_FEE_PCT,
  HONEST_TAKER_EXIT_FEE_PCT,
  buildSweepBacktestOptions,
  holdoutSlice,
  isRuined,
  logReturn,
  median,
  renderMarkdownTable,
  runGrowthSizingSweep,
} from "./growth-sizing-sweep.ts";
import type { AlignedPanel } from "./prepare.ts";
import { runLadderGridBacktest } from "../src/scalping/ladder-grid.ts";
import type { AutoresearchKnobs } from "./knobs.ts";
import type { Candle } from "../src/market-data/types.ts";

const BASE: AutoresearchKnobs = {
  rungs: 2,
  gridStepPct: 1.3,
  gridMaxGrids: 2,
  gridPauseAfterLossBars: 2,
  stopRatio: 1.58,
  targetRatio: 1.95,
  maxHoldBars: 39,
  trendFilterPeriod: 0,
  chopGateAdxThreshold: 0,
  positionFraction: 1,
};

function oscCandles(n: number, mid: number, symbol: string): Candle[] {
  const out: Candle[] = [];
  for (let i = 0; i < n; i++) {
    const side = i % 2 === 0 ? 1 : -1;
    out.push({
      exchange: "bybit-futures",
      symbol,
      timeframe: "15m",
      open: mid - 0.3 * side,
      high: mid + 0.6 + 0.05,
      low: mid - 0.6 - 0.05,
      close: mid + 0.3 * side,
      volume: 1,
      timestamp: new Date(i * 15 * 60 * 1000),
    });
  }
  return out;
}

function synthPanel(n: number, holdoutBars: number): AlignedPanel {
  const aligned = new Map<string, Candle[]>([
    ["A", oscCandles(n, 100, "A")],
    ["B", oscCandles(n, 200, "B")],
    ["C", oscCandles(n, 300, "C")],
  ]);
  return {
    symbols: ["A", "B", "C"],
    aligned,
    refLen: n,
    loadedMs: 1,
    exchange: "bybit-futures",
    timeframe: "5m",
    panelTimeframe: "15m",
    panelHash: "synth-sizing",
    holdoutBars,
  };
}

/** Rewrite every selection-prefix bar as a steep trend; tail untouched. */
function poisonPrefix(panel: AlignedPanel): AlignedPanel {
  const keep = panel.holdoutBars;
  const aligned = new Map<string, Candle[]>();
  for (const [s, cs] of panel.aligned) {
    const cp = cs.map((c) => ({ ...c }));
    let px = 10;
    for (let i = 0; i < cp.length - keep; i++) {
      px *= 1.05;
      cp[i] = { ...cp[i]!, open: px / 1.01, high: px * 1.005, low: px / 1.015, close: px };
    }
    aligned.set(s, cp);
  }
  return { ...panel, aligned };
}

describe("sizing sweep honoring frozen-holdout convention", () => {
  it("uses honest venue fees and maps DD budget to the account kill", () => {
    expect(HONEST_MAKER_FEE_PCT).toBe(0.02);
    expect(HONEST_TAKER_EXIT_FEE_PCT).toBe(0.06);
    const opts = buildSweepBacktestOptions(BASE, {
      positionFraction: 0.5,
      leverage: 2,
      maxDrawdownPct: 30,
    });
    expect(opts.feePct).toBe(0.02);
    expect(opts.takerExitFeePct).toBe(0.06);
    expect(opts.maxDrawdownPct).toBe(30);
    expect(opts.positionFraction).toBe(0.5);
    expect(opts.leverage).toBe(2);
    // Champion geometry survives untouched.
    expect(opts.rungs).toBe(BASE.rungs);
    expect(opts.stopRatio).toBe(BASE.stopRatio);
    expect(opts.targetRatio).toBe(BASE.targetRatio);
    expect(opts.conservativeIntrabar).toBe(true);
  });

  it("scores only the frozen tail: a poisoned prefix changes nothing", () => {
    const clean = synthPanel(400, 120);
    const poisoned = poisonPrefix(clean);
    const a = runGrowthSizingSweep(clean, BASE);
    const b = runGrowthSizingSweep(poisoned, BASE);
    expect(a.rows.length).toBe(b.rows.length);
    expect(a.rows.length).toBe(3 * 3 * 3);
    for (let i = 0; i < a.rows.length; i++) {
      expect(b.rows[i]!.growthLogRet).toBe(a.rows[i]!.growthLogRet);
      expect(b.rows[i]!.ruinRate).toBe(a.rows[i]!.ruinRate);
      expect(b.rows[i]!.medianMaxDrawdownPct).toBe(
        a.rows[i]!.medianMaxDrawdownPct,
      );
    }
  });

  it("holdoutSlice is the reserved trailing tail", () => {
    const panel = synthPanel(400, 120);
    const slice = holdoutSlice(panel, "A");
    expect(slice.length).toBe(120);
    expect(slice[0]!.timestamp.getTime()).toBe(
      panel.aligned.get("A")![280]!.timestamp.getTime(),
    );
  });

  it("taker-exit fee never flatters a stop-heavy run", () => {
    const panel = synthPanel(400, 120);
    const slice = holdoutSlice(panel, "A");
    const cell = { positionFraction: 1, leverage: 2, maxDrawdownPct: 50 };
    const honest = runLadderGridBacktest(
      slice,
      buildSweepBacktestOptions(BASE, cell),
    );
    const makerOnly = runLadderGridBacktest(slice, {
      ...buildSweepBacktestOptions(BASE, cell),
      takerExitFeePct: HONEST_MAKER_FEE_PCT,
    });
    expect(honest.totalReturnPct).toBeLessThanOrEqual(
      makerOnly.totalReturnPct + 1e-9,
    );
  });

  it("ruin, log-return, and median math", () => {
    expect(isRuined(-16, 0, 15)).toBe(true);
    expect(isRuined(-14.9, 0, 15)).toBe(false);
    expect(isRuined(5, 1, 50)).toBe(true);
    expect(isRuined(-100, 0, 50)).toBe(true);
    expect(isRuined(Number.NaN, 0, 15)).toBe(true);
    expect(logReturn(0)).toBeCloseTo(0, 12);
    expect(logReturn(-100)).toBeCloseTo(Math.log(0.05), 12);
    expect(median([3, 1, 2])).toBe(2);
    expect(median([4, 1, 3, 2])).toBe(2.5);
    expect(median([])).toBeNaN();
  });

  it("never mutates the frozen base knobs", () => {
    const panel = synthPanel(400, 120);
    const snapshot = JSON.stringify(BASE);
    runGrowthSizingSweep(panel, BASE);
    expect(JSON.stringify(BASE)).toBe(snapshot);
  });

  it("reports growth vs maxDD vs ruin per budget with a within-budget pick", () => {
    const panel = synthPanel(400, 120);
    const report = runGrowthSizingSweep(panel, BASE);
    expect(report.budgets).toEqual([15, 30, 50]);
    expect(report.honestFees.takerExitFeePct).toBe(0.06);
    expect(report.dataset.panelHash).toBe("synth-sizing");
    for (const budget of [15, 30, 50]) {
      const rows = report.rows.filter((r) => r.budgetPct === budget);
      expect(rows.length).toBe(9);
      for (const r of rows) {
        expect(r.windows).toBe(3);
        expect(Number.isFinite(r.growthLogRet)).toBe(true);
        expect(r.ruinRate).toBeGreaterThanOrEqual(0);
        expect(r.ruinRate).toBeLessThanOrEqual(1);
      }
      const rec = report.recommended[budget];
      if (rec) {
        expect(rec.budgetPct).toBe(budget);
        expect(rec.medianMaxDrawdownPct).toBeLessThanOrEqual(budget);
      }
    }
    const md = renderMarkdownTable(report);
    for (const budget of [15, 30, 50]) {
      expect(md).toContain(`DD budget ${budget}%`);
    }
    expect(md).toContain("growthLogRet");
    expect(md).toContain("ruinRate");
  });
});
