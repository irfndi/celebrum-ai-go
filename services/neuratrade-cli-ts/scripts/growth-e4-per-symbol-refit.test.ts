// Growth E4 per-symbol refit — unit tests for the pure helpers.
// No DB, no live workers, no champion files: synthetic candles only.
import { describe, expect, test } from "bun:test";
import {
  CANDIDATE_COUNT,
  HOLDOUT_FRACTION,
  MIN_CANDLES,
  REFIT_ADX,
  REFIT_GRIDS,
  REFIT_PAUSES,
  REFIT_STEPS,
  REFIT_SYMBOLS,
  REFIT_TARGETS,
  TIMEFRAME,
  VENUE,
  baseGrid,
  buildCandidates,
  oosStartFor,
  passesSelectionGate,
  rankSelection,
  selectionWalkForward,
  type RankedCandidate,
  type SelectionMetrics,
} from "./growth-e4-per-symbol-refit.js";
import type { CandleLike } from "../src/scalping/types.js";

function flatCandles(n: number): CandleLike[] {
  const t0 = Date.UTC(2025, 0, 1);
  return Array.from({ length: n }, (_, i) => ({
    open: 100,
    high: 100,
    low: 100,
    close: 100,
    volume: 1,
    timestamp: new Date(t0 + i * 15 * 60 * 1000),
  }));
}

function sel(over: Partial<SelectionMetrics> = {}): SelectionMetrics {
  return {
    windowCount: 10,
    profitableWindowPct: 60,
    compoundedReturnPct: 5,
    maxDrawdownPct: 8,
    totalTrades: 100,
    valid: true,
    ...over,
  };
}

describe("refit venue + symbol pins", () => {
  test("targets E3 closest-but-failing on bybit-futures 15m", () => {
    expect(VENUE).toBe("bybit-futures");
    expect(TIMEFRAME).toBe("15m");
    expect(MIN_CANDLES).toBe(55000);
    expect(HOLDOUT_FRACTION).toBe(0.2);
    expect([...REFIT_SYMBOLS]).toEqual([
      "LTC/USDT:USDT",
      "XRP/USDT:USDT",
      "ETC/USDT:USDT",
    ]);
  });
});

describe("candidate space", () => {
  test("162 structural configs with honest fees frozen", () => {
    expect(CANDIDATE_COUNT).toBe(162);
    expect(
      REFIT_STEPS.length *
        REFIT_GRIDS.length *
        REFIT_PAUSES.length *
        REFIT_TARGETS.length *
        REFIT_ADX.length,
    ).toBe(162);
    const cands = buildCandidates();
    expect(cands).toHaveLength(162);
    for (const c of cands) {
      expect(c.feePct).toBe(0.02);
      expect(c.takerExitFeePct).toBe(0.06);
      expect(c.slippageBps).toBe(1);
      expect(c.leverage).toBe(1);
      expect(c.positionFraction).toBe(1);
      expect(c.initialCapital).toBe(100);
      expect(c.trendFilterPeriod).toBe(0);
    }
  });
  test("base grid matches E3 frozen champion knobs", () => {
    const b = baseGrid();
    expect(b.gridStepPct).toBe(1.3);
    expect(b.gridMaxGrids).toBe(2);
    expect(b.gridPauseAfterLossBars).toBe(2);
    expect(b.targetRatio).toBe(1.95);
    expect(b.chopGateAdxThreshold).toBe(0);
  });
  test("candidate order is deterministic", () => {
    const a = buildCandidates();
    const b = buildCandidates();
    expect(a.map((c) => c.gridStepPct)).toEqual(b.map((c) => c.gridStepPct));
    expect(a[0]?.gridStepPct).toBe(REFIT_STEPS[0]);
  });
});

describe("oosStartFor", () => {
  test("69120 bars split into 55296 selection + 13824 tail", () => {
    expect(oosStartFor(69120)).toBe(55296);
  });
  test("selection holds exactly 10 walk-forward windows", () => {
    // start + 11520 + 4320 <= 55296 -> 10 starts (0..38880 step 4320)
    let windows = 0;
    for (let s = 0; s + 11520 + 4320 <= oosStartFor(69120); s += 4320) {
      windows += 1;
    }
    expect(windows).toBe(10);
  });
});

describe("selectionWalkForward", () => {
  test("flat candles run without NaN (no trades, zero return)", () => {
    const m = selectionWalkForward(flatCandles(200), baseGrid(), 20, 10);
    expect(m.valid).toBe(true);
    expect(m.windowCount).toBeGreaterThan(0);
    expect(Number.isFinite(m.compoundedReturnPct)).toBe(true);
    expect(Number.isFinite(m.maxDrawdownPct)).toBe(true);
  });
  test("empty selection is invalid, not a crash", () => {
    const m = selectionWalkForward([], baseGrid());
    expect(m.valid).toBe(false);
    expect(m.windowCount).toBe(0);
  });
});

describe("passesSelectionGate", () => {
  test("good selection passes", () => {
    expect(passesSelectionGate(sel())).toBe(true);
  });
  test("each breach fails", () => {
    expect(passesSelectionGate(sel({ windowCount: 9 }))).toBe(false);
    expect(passesSelectionGate(sel({ profitableWindowPct: 40 }))).toBe(false);
    expect(passesSelectionGate(sel({ compoundedReturnPct: -1 }))).toBe(false);
    expect(passesSelectionGate(sel({ maxDrawdownPct: 20 }))).toBe(false);
    expect(passesSelectionGate(sel({ valid: false }))).toBe(false);
  });
});

describe("rankSelection", () => {
  const grid = baseGrid();
  const row = (
    index: number,
    metrics: SelectionMetrics,
    eligible: boolean,
  ): RankedCandidate => ({ index, grid, metrics, eligible });
  test("eligible outranks higher-return ineligible", () => {
    const winner = rankSelection([
      row(0, sel({ compoundedReturnPct: 99 }), false),
      row(1, sel({ compoundedReturnPct: 1 }), true),
    ]);
    expect(winner.index).toBe(1);
  });
  test("compounded desc, then dd asc, then index asc", () => {
    const winner = rankSelection([
      row(2, sel({ compoundedReturnPct: 5, maxDrawdownPct: 9 }), true),
      row(1, sel({ compoundedReturnPct: 5, maxDrawdownPct: 9 }), true),
      row(0, sel({ compoundedReturnPct: 5, maxDrawdownPct: 3 }), true),
    ]);
    expect(winner.index).toBe(0);
    const byRet = rankSelection([
      row(0, sel({ compoundedReturnPct: 2 }), true),
      row(1, sel({ compoundedReturnPct: 7 }), true),
    ]);
    expect(byRet.index).toBe(1);
  });
  test("empty list throws instead of returning undefined", () => {
    expect(() => rankSelection([])).toThrow();
  });
});
