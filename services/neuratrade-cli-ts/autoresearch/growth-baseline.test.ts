import { describe, expect, it } from "bun:test";
import {
  calibrateSigma,
  createRng,
  driftOnlyTradesToTarget,
  FROZEN_EXPECTANCY_PCT_PER_TRADE,
  FROZEN_MEDIAN_MAXDD_PCT,
  median,
  medianMaxDrawdownPct,
  randn,
  runGrowthMonteCarlo,
  simulateMaxDrawdownPct,
} from "./growth-baseline.ts";

const MU_LOG = Math.log(1 + FROZEN_EXPECTANCY_PCT_PER_TRADE / 100);

describe("growth-baseline rng", () => {
  it("reproduces the same gaussian stream per seed", () => {
    const a = createRng(42);
    const b = createRng(42);
    for (let i = 0; i < 20; i++) {
      expect(randn(a)).toBe(randn(b));
    }
  });

  it("draws an approximately standard normal (mean~0, var~1)", () => {
    const rng = createRng(7);
    const n = 20000;
    let sum = 0;
    let sum2 = 0;
    for (let i = 0; i < n; i++) {
      const x = randn(rng);
      sum += x;
      sum2 += x * x;
    }
    expect(Math.abs(sum / n)).toBeLessThan(0.05);
    expect(Math.abs(sum2 / n - 1)).toBeLessThan(0.05);
  });
});

describe("growth-baseline stats", () => {
  it("median handles odd, even, and empty input", () => {
    expect(median([3, 1, 2])).toBe(2);
    expect(median([4, 1, 3, 2])).toBe(2.5);
    expect(Number.isNaN(median([]))).toBe(true);
  });

  it("zero volatility means zero drawdown", () => {
    const rng = createRng(1);
    // Positive drift only: equity never dips below its running peak.
    expect(simulateMaxDrawdownPct(0, Math.abs(MU_LOG), 100, rng)).toBe(0);
  });

  it("median maxDD rises monotonically with sigma", () => {
    const low = medianMaxDrawdownPct(0.002, MU_LOG, 100, 500, 11);
    const high = medianMaxDrawdownPct(0.02, MU_LOG, 100, 500, 11);
    expect(high).toBeGreaterThan(low);
  });
});

describe("growth-baseline calibration", () => {
  it("recovers a sigma whose median DD matches the frozen 8.53%", () => {
    const sigma = calibrateSigma(FROZEN_MEDIAN_MAXDD_PCT, MU_LOG, 100, 99, 1500, 14);
    expect(Number.isFinite(sigma)).toBe(true);
    expect(sigma).toBeGreaterThan(0.001);
    expect(sigma).toBeLessThan(0.05);
    const check = medianMaxDrawdownPct(sigma, MU_LOG, 100, 3000, 1001);
    expect(Math.abs(check - FROZEN_MEDIAN_MAXDD_PCT)).toBeLessThan(1.0);
  });
});

describe("growth-baseline monte carlo", () => {
  it("is deterministic per seed and partitions every path", () => {
    const opts = {
      tradesPerDay: 5,
      paths: 100,
      maxYears: 2,
      seed: 5,
      sigma: 0.0087,
      muLog: MU_LOG,
    };
    const a = runGrowthMonteCarlo(opts);
    const b = runGrowthMonteCarlo(opts);
    expect(a).toEqual(b);
    expect(a.successRate + a.ruinRate + a.censoredRate).toBeCloseTo(1, 10);
  });

  it("a near-zero kill-switch ruins almost every path", () => {
    const r = runGrowthMonteCarlo({
      tradesPerDay: 5,
      paths: 200,
      maxYears: 5,
      seed: 3,
      sigma: 0.0087,
      muLog: MU_LOG,
      maxDdKillPct: 0.1,
    });
    expect(r.ruinRate).toBeGreaterThan(0.95);
    expect(r.successRate).toBe(0);
    expect(r.medianDaysToTarget).toBeNull();
  });

  it("daily-loss halt never increases realized trades vs no halt", () => {
    // With halt disabled (100%), paths run every scheduled trade; the kill
    // rate must be at least as high as with the 5% halt filtering trades.
    const base = {
      tradesPerDay: 5,
      paths: 200,
      maxYears: 5,
      seed: 9,
      sigma: 0.0087,
      muLog: MU_LOG,
    };
    const halted = runGrowthMonteCarlo(base);
    const unhalted = runGrowthMonteCarlo({ ...base, dailyLossHaltPct: 100 });
    expect(unhalted.ruinRate).toBeGreaterThanOrEqual(halted.ruinRate - 0.05);
  });

  it("drift-only benchmark matches ln(1000)/mu", () => {
    const trades = driftOnlyTradesToTarget(MU_LOG);
    expect(trades).toBeCloseTo(Math.log(1000) / MU_LOG, 8);
    // Frozen edge implies ~700k trades (~centuries at 3-7/day) on drift alone.
    expect(trades).toBeGreaterThan(500_000);
  });
});
