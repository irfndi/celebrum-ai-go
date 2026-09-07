/**
 * Growth E2 timeframe downshift — ADDITIVE-ONLY tests (no DB writes, no
 * champion writes, no CLAIM-gate edits). Pure helpers + synthetic panels.
 */
import { describe, expect, it } from "bun:test";
import {
  E2_CHAMPION_KNOBS,
  E2_EXCHANGE,
  E2_FEE_PCT,
  E2_SLIPPAGE_BPS,
  E2_TAKER_EXIT_FEE_PCT,
  evaluateE2Holdout,
  evaluateE2Selection,
  geometryForTf,
  holdoutBarsForTf,
  median,
  type E2Panel,
} from "./timeframe-downshift-e2.ts";
import type { Candle } from "../src/market-data/types.ts";

function osc(
  n: number,
  mid: number,
  amp: number,
  symbol: string,
  tfMin: number,
): Candle[] {
  const out: Candle[] = [];
  for (let i = 0; i < n; i++) {
    const side = i % 2 === 0 ? 1 : -1;
    out.push({
      exchange: E2_EXCHANGE,
      symbol,
      timeframe: "15m",
      open: mid - (amp / 2) * side,
      high: mid + amp + 0.05,
      low: mid - amp - 0.05,
      close: mid + (amp / 2) * side,
      volume: 1,
      timestamp: new Date(i * tfMin * 60_000),
    });
  }
  return out;
}
interface SynthPanelResult {
  readonly panel: E2Panel;
  readonly spec: ReturnType<typeof geometryForTf>;
}

function synthPanel(panelTf: string, n: number): SynthPanelResult {
  const src = panelTf === "15m" ? "5m" : panelTf;
  const spec = geometryForTf(panelTf, src);
  const tfMin = spec.tfMinutes;
  const aligned = new Map<string, Candle[]>([
    ["A", osc(n, 100, 0.6, "A", tfMin)],
    ["B", osc(n, 200, 0.9, "B", tfMin)],
    ["C", osc(n, 300, 0.5, "C", tfMin)],
    ["D", osc(n, 400, 0.7, "D", tfMin)],
  ]);
  return {
    spec,
    panel: {
      symbols: ["A", "B", "C", "D"],
      aligned,
      refLen: n,
      exchange: E2_EXCHANGE,
      sourceTimeframe: src,
      panelTimeframe: panelTf,
      holdoutBars: spec.holdoutBars,
      reason: "ok",
    },
  };
}

describe("timeframe-downshift-e2 (additive probe)", () => {
  it("uses the honest taker-exit fee schedule", () => {
    expect(E2_FEE_PCT).toBe(0.02);
    expect(E2_TAKER_EXIT_FEE_PCT).toBe(0.06);
    expect(E2_SLIPPAGE_BPS).toBe(2);
  });

  it("scales the frozen 30d holdout per timeframe", () => {
    expect(holdoutBarsForTf(15)).toBe(2880);
    expect(holdoutBarsForTf(5)).toBe(8640);
    expect(holdoutBarsForTf(1)).toBe(43200);
    expect(geometryForTf("15m", "5m").forwardBars).toBe(2880);
    expect(geometryForTf("5m", "5m").forwardBars).toBe(8640);
    expect(geometryForTf("1m", "1m").forwardBars).toBe(43200);
  });

  it("keeps the champion copy frozen (shape only, never imports live knobs)", () => {
    expect(E2_CHAMPION_KNOBS.rungs).toBe(2);
    expect(E2_CHAMPION_KNOBS.gridStepPct).toBeCloseTo(1.3, 6);
    expect(E2_CHAMPION_KNOBS.positionFraction).toBe(1);
  });

  it("median helper handles empty/NaN", () => {
    expect(median([3, 1, 2])).toBe(2);
    expect(Number.isNaN(median([]))).toBe(true);
    expect(Number.isNaN(median([Number.NaN]))).toBe(true);
  });

  it("selection never touches the frozen holdout tail", () => {
    const { panel, spec } = synthPanel("15m", specLenFor("15m"));
    const sel = evaluateE2Selection(E2_CHAMPION_KNOBS, panel, spec, {
      maxSteps: 2,
    });
    expect(sel.windows).toBeGreaterThan(0);
    // Poison the holdout tail: selection metrics must be invariant.
    const poisoned = poisonTail(panel);
    const sel2 = evaluateE2Selection(E2_CHAMPION_KNOBS, poisoned, spec, {
      maxSteps: 2,
    });
    expect(sel2.score).toBe(sel.score);
    expect(sel2.tradesPerSymMonth).toBe(sel.tradesPerSymMonth);
  });

  it("holdout evaluates exactly the trailing 30d window per symbol", () => {
    const { panel, spec } = synthPanel("15m", specLenFor("15m"));
    const h = evaluateE2Holdout(E2_CHAMPION_KNOBS, panel, spec);
    expect(h.windows).toBe(panel.symbols.length);
    // Poison the selection prefix: holdout metrics must be invariant.
    const poisoned = poisonHead(panel);
    const h2 = evaluateE2Holdout(E2_CHAMPION_KNOBS, poisoned, spec);
    expect(h2.score).toBe(h.score);
    expect(h2.tradesPerSymMonth).toBe(h.tradesPerSymMonth);
  });

  it("reports missing 1m venue data honestly instead of mixing venues", () => {
    const spec = geometryForTf("1m", "1m");
    const empty: E2Panel = {
      symbols: [],
      aligned: new Map(),
      refLen: 0,
      exchange: E2_EXCHANGE,
      sourceTimeframe: "1m",
      panelTimeframe: "1m",
      holdoutBars: spec.holdoutBars,
      reason: "insufficient_data_no_1m_venue_candles",
    };
    const sel = evaluateE2Selection(E2_CHAMPION_KNOBS, empty, spec);
    const h = evaluateE2Holdout(E2_CHAMPION_KNOBS, empty, spec);
    expect(sel.reason).toContain("insufficient_data_no_1m_venue_candles");
    expect(h.reason).toContain("insufficient_data_no_1m_venue_candles");
    expect(empty.exchange).toBe("bybit-futures");
  });
});

function specLenFor(panelTf: string): number {
  const spec = geometryForTf(panelTf, panelTf === "15m" ? "5m" : panelTf);
  return (
    spec.holdoutBars + spec.forwardBars + spec.firstBar + spec.stepBars * 3
  );
}

function poisonTail(panel: E2Panel): E2Panel {
  const aligned = new Map<string, Candle[]>();
  for (const [s, cs] of panel.aligned) {
    const cp = cs.map((c) => ({ ...c }));
    let px = cp[cp.length - panel.holdoutBars - 1]!.close;
    for (let i = cp.length - panel.holdoutBars; i < cp.length; i++) {
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

function poisonHead(panel: E2Panel): E2Panel {
  const aligned = new Map<string, Candle[]>();
  for (const [s, cs] of panel.aligned) {
    const cp = cs.map((c) => ({ ...c }));
    let px = 1;
    for (let i = 0; i < cp.length - panel.holdoutBars; i++) {
      px *= 1.05;
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
