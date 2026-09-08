import { describe, expect, it } from "bun:test";
import {
  looseDirection,
  scoreDirections,
  type LooseObservation,
} from "./timesfm-oos-loose.js";

function obs(
  partial: Partial<LooseObservation> & { actualReturnPct: number },
): LooseObservation {
  return {
    originIndex: 0,
    pointForecastReturnPct: 0,
    q10ReturnPct: null,
    q90ReturnPct: null,
    baselineDirection: "flat",
    baselineNetReturnPct: 0,
    ...partial,
  };
}

describe("looseDirection (quantile band excludes zero)", () => {
  it("goes long when q10 is above the threshold", () => {
    expect(looseDirection(0.01, 0.5, 0)).toBe("long");
  });
  it("goes short when q90 is below the negated threshold", () => {
    expect(looseDirection(-0.5, -0.01, 0)).toBe("short");
  });
  it("stays flat when the band straddles the threshold", () => {
    expect(looseDirection(-0.5, 0.5, 0)).toBe("flat");
  });
  it("fails closed on missing quantiles", () => {
    expect(looseDirection(null, null, 0)).toBe("flat");
  });
  it("tight threshold is stricter than the loose one", () => {
    expect(looseDirection(0.05, 0.5, 0.16)).toBe("flat");
    expect(looseDirection(0.05, 0.5, 0)).toBe("long");
  });
});

describe("scoreDirections", () => {
  it("scores a tiny trade list with friction", () => {
    const observations = [
      obs({ actualReturnPct: 1, q10ReturnPct: 0.1, q90ReturnPct: 0.5 }),
      obs({ actualReturnPct: -1, q10ReturnPct: -0.5, q90ReturnPct: -0.1 }),
      obs({ actualReturnPct: 1, q10ReturnPct: -0.5, q90ReturnPct: 0.5 }),
    ];
    const metrics = scoreDirections(
      observations,
      (o) => looseDirection(o.q10ReturnPct, o.q90ReturnPct, 0),
      (o) => {
        const direction = looseDirection(o.q10ReturnPct, o.q90ReturnPct, 0);
        if (direction === "long") return o.actualReturnPct - 0.16;
        if (direction === "short") return -o.actualReturnPct - 0.16;
        return 0;
      },
      (o) => o.pointForecastReturnPct,
    );
    expect(metrics.trades).toBe(2);
    expect(metrics.directionAccuracyPct).toBe(100);
    expect(metrics.winRatePct).toBe(100);
    expect(metrics.netReturnPct).toBeCloseTo(0.84 + 0.84, 10);
  });
});
