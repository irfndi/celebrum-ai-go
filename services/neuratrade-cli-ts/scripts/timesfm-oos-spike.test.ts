import { describe, expect, it } from "bun:test";
import { latencyBudget, planFrozenHoldout } from "./timesfm-oos-spike.js";

describe("timesfm-oos-spike (throwaway)", () => {
  it("reuses the frozen last-20% holdout rule", () => {
    const plan = planFrozenHoldout(69120, 256, 12, 12, 0);
    expect(plan.oosStart).toBe(Math.floor(69120 * 0.8));
    expect(plan.oosBars).toBe(69120 - Math.floor(69120 * 0.8));
    expect(plan.origins).toBeGreaterThan(1000);
    expect(plan.coveredBars).toBe(plan.origins * 12);
  });

  it("caps origins when maxOrigins is set", () => {
    const full = planFrozenHoldout(69120, 256, 12, 12, 0);
    const capped = planFrozenHoldout(69120, 256, 12, 12, 96);
    expect(capped.origins).toBe(96);
    expect(full.origins).toBeGreaterThan(96);
  });

  it("sizes the latency budget against the 900s bar loop", () => {
    const budget = latencyBudget(3);
    expect(budget.barLoopMs).toBe(900_000);
    expect(budget.mustP95Ms).toBe(120_000);
    expect(budget.shouldTotalMs).toBe(60_000);
    expect(budget.mustP95Ms).toBeLessThan(budget.barLoopMs);
  });

  it("rejects non-positive inputs", () => {
    expect(() => planFrozenHoldout(100, 0, 12, 12, 0)).toThrow();
    expect(() => latencyBudget(0)).toThrow();
  });
});
