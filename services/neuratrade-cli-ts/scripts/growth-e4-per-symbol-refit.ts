// Growth E4 per-symbol grid refit (clever-cabin-6a8) — ADDITIVE-ONLY screen.
//
// E3 (scripts/growth-e3-cohort-expansion.ts) found LTC/XRP/ETC closest-but-
// failing on bybit-futures 15m with one-size-fits-all frozen champion grid.
// This script refits the structural grid knobs PER SYMBOL on SELECTION
// windows only (first 80% of candles per symbol — never the tail), then
// evaluates each fitted config on the SAME frozen last-20% tail + 5-seed
// stress used in E3, via the SAME validateGridEvidence + passesCohortGate
// imported from the E3 module (no reimplementation, no drift).
//
// Frozen honest fees (program.md: maker 0.02, taker-exit 0.06):
//   feePct 0.02, takerExitFeePct 0.06, slippageBps 1, leverage 1,
//   positionFraction 1, initialCapital 100, trendFilterPeriod 0,
//   onlyWithTrend false. Refit varies ONLY: gridStepPct, gridMaxGrids,
//   gridPauseAfterLossBars, targetRatio, chopGateAdxThreshold.
//
// Selection protocol (no tail peeking):
//   oosStart = floor(n * 0.8); selection = candles[0:oosStart].
//   Each candidate runs walk-forward inside the selection slice with the
//   SAME trainBars 11520 / testBars 4320 stepping as validateGridEvidence.
//   Rank: selection-gate-eligible first (windows>=10, win>=50%, ret>0,
//   dd<=15), then by compounded return desc, dd asc, candidate index asc
//   (deterministic tiebreak). 162 candidates per symbol.
//
// Final verdict per symbol: validateGridEvidence(full candles, fitted grid)
// + passesCohortGate (E3 imports) -> PASS/FAIL vs all 8 cohort gates.
// A frozen-grid re-eval on the same tail is included for the refit delta.
//
// ADDITIVE ONLY: opens the SQLite DB read-only, writes ONLY the two new
// result files below, never touches live workers, DBs, champion files,
// CLAIM gates (goals.ts), or ecosystem configs.
//
// Usage:
//   bun run scripts/growth-e4-per-symbol-refit.ts [--symbols=LTC/XRP/ETC]
//   bun run scripts/growth-e4-per-symbol-refit.ts --symbols=LTC/USDT:USDT
//
// Output (new files, additive):
//   autoresearch/results/growth-e4-per-symbol-refit.json
//   autoresearch/results/growth-e4-per-symbol-refit.md
import { Database } from "bun:sqlite";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { runGridBacktest, type GridOptions } from "../src/scalping/grid.js";
import type { CandleLike } from "../src/scalping/types.js";
import { validateGridEvidence } from "../src/scalping/grid-validation.js";
import {
  FROZEN_CHAMPION_GRID,
  meanTradeExpectancyPct,
  passesCohortGate,
  tradesPerMonth,
  type CohortGateMetrics,
} from "./growth-e3-cohort-expansion.js";

export const VENUE = "bybit-futures";
export const TIMEFRAME = "15m";
export const TIMEFRAME_MINUTES = 15;
export const MIN_CANDLES = 55000;
export const HOLDOUT_FRACTION = 0.2;
export const TRAIN_BARS = 11520;
export const TEST_BARS = 4320;

/** Closest-but-failing E3 symbols selected for per-symbol refit. */
export const REFIT_SYMBOLS = [
  "LTC/USDT:USDT",
  "XRP/USDT:USDT",
  "ETC/USDT:USDT",
] as const;

/** Honest-fee floor shared by every candidate (frozen, never refit). */
export function baseGrid(): GridOptions {
  return {
    gridStepPct: 1.3,
    gridMaxGrids: 2,
    gridPauseAfterLossBars: 2,
    feePct: 0.02,
    takerExitFeePct: 0.06,
    slippageBps: 1,
    initialCapital: 100,
    trendFilterPeriod: 0,
    leverage: 1,
    positionFraction: 1,
    onlyWithTrend: false,
    targetRatio: 1.95,
    chopGateAdxThreshold: 0,
  };
}

export const REFIT_STEPS = [0.9, 1.3, 1.7] as const;
export const REFIT_GRIDS = [2, 3] as const;
export const REFIT_PAUSES = [0, 2, 4] as const;
export const REFIT_TARGETS = [1.5, 1.95, 2.5] as const;
export const REFIT_ADX = [0, 20, 25] as const;
export const CANDIDATE_COUNT =
  REFIT_STEPS.length *
  REFIT_GRIDS.length *
  REFIT_PAUSES.length *
  REFIT_TARGETS.length *
  REFIT_ADX.length;

/** Deterministic candidate space: 162 structural configs, fees frozen. */
export function buildCandidates(): GridOptions[] {
  const out: GridOptions[] = [];
  for (const gridStepPct of REFIT_STEPS) {
    for (const gridMaxGrids of REFIT_GRIDS) {
      for (const gridPauseAfterLossBars of REFIT_PAUSES) {
        for (const targetRatio of REFIT_TARGETS) {
          for (const chopGateAdxThreshold of REFIT_ADX) {
            out.push({
              ...baseGrid(),
              gridStepPct,
              gridMaxGrids,
              gridPauseAfterLossBars,
              targetRatio,
              chopGateAdxThreshold,
            });
          }
        }
      }
    }
  }
  return out;
}

/** Frozen tail split: selection = first 80%, tail = last 20% (E3 tail). */
export function oosStartFor(candleCount: number): number {
  return Math.floor(candleCount * (1 - HOLDOUT_FRACTION));
}

export interface SelectionMetrics {
  readonly windowCount: number;
  readonly profitableWindowPct: number;
  readonly compoundedReturnPct: number;
  readonly maxDrawdownPct: number;
  readonly totalTrades: number;
  readonly valid: boolean;
}

/**
 * Walk-forward inside the SELECTION slice only, mirroring the compounding
 * math of validateGridEvidence (capital *= 1+ret/100, peak-to-trough dd).
 * Any window at/below -100% invalidates the candidate (same as the gate).
 */
export function selectionWalkForward(
  selection: readonly CandleLike[],
  grid: GridOptions,
  trainBars = TRAIN_BARS,
  testBars = TEST_BARS,
): SelectionMetrics {
  let capital = grid.initialCapital;
  let peak = capital;
  let maxDd = 0;
  let wins = 0;
  let windows = 0;
  let trades = 0;
  for (
    let start = 0;
    start + trainBars + testBars <= selection.length;
    start += testBars
  ) {
    const r = runGridBacktest(
      selection.slice(start + trainBars, start + trainBars + testBars),
      { ...grid, initialCapital: grid.initialCapital },
    );
    if (r.totalReturnPct <= -100) {
      return {
        windowCount: 0,
        profitableWindowPct: 0,
        compoundedReturnPct: Number.NEGATIVE_INFINITY,
        maxDrawdownPct: Number.POSITIVE_INFINITY,
        totalTrades: 0,
        valid: false,
      };
    }
    capital *= 1 + r.totalReturnPct / 100;
    peak = Math.max(peak, capital);
    if (peak > 0) maxDd = Math.max(maxDd, ((peak - capital) / peak) * 100);
    if (r.totalReturnPct > 0) wins += 1;
    windows += 1;
    trades += r.totalTrades;
  }
  if (windows === 0) {
    return {
      windowCount: 0,
      profitableWindowPct: 0,
      compoundedReturnPct: 0,
      maxDrawdownPct: 0,
      totalTrades: 0,
      valid: false,
    };
  }
  return {
    windowCount: windows,
    profitableWindowPct: (wins / windows) * 100,
    compoundedReturnPct:
      ((capital - grid.initialCapital) / grid.initialCapital) * 100,
    maxDrawdownPct: maxDd,
    totalTrades: trades,
    valid: true,
  };
}

/** Selection gate mirrors the walk-forward half of the cohort gate. */
export function passesSelectionGate(m: SelectionMetrics): boolean {
  return (
    m.valid &&
    m.windowCount >= 10 &&
    m.profitableWindowPct >= 50 &&
    m.compoundedReturnPct > 0 &&
    m.maxDrawdownPct <= 15
  );
}

export interface RankedCandidate {
  readonly index: number;
  readonly grid: GridOptions;
  readonly metrics: SelectionMetrics;
  readonly eligible: boolean;
}

/**
 * Deterministic rank: eligible first, then compounded desc, dd asc,
 * candidate index asc. Returns the winner (index 0 after sort).
 */
export function rankSelection(
  rows: readonly RankedCandidate[],
): RankedCandidate {
  const sorted = [...rows].sort((a, b) => {
    if (a.eligible !== b.eligible) return a.eligible ? -1 : 1;
    if (a.metrics.compoundedReturnPct !== b.metrics.compoundedReturnPct) {
      return b.metrics.compoundedReturnPct - a.metrics.compoundedReturnPct;
    }
    if (a.metrics.maxDrawdownPct !== b.metrics.maxDrawdownPct) {
      return a.metrics.maxDrawdownPct - b.metrics.maxDrawdownPct;
    }
    return a.index - b.index;
  });
  const winner = sorted[0];
  if (!winner) throw new Error("rankSelection: empty candidate list");
  return winner;
}

export interface RefitRow {
  readonly symbol: string;
  readonly status: "ok" | "invalid" | "venue-filtered";
  readonly candles?: number;
  readonly detail?: string;
  readonly fitted?: {
    readonly gridStepPct: number;
    readonly gridMaxGrids: number;
    readonly gridPauseAfterLossBars: number;
    readonly targetRatio: number;
    readonly chopGateAdxThreshold: number;
    readonly candidateIndex: number;
  };
  readonly selection?: SelectionMetrics & { readonly pass: boolean };
  readonly expectancyPct?: number;
  readonly tradesPerMonth?: number;
  readonly oosWinRatePct?: number;
  readonly oosReturnPct?: number;
  readonly profitFactor?: number | string;
  readonly gates?: CohortGateMetrics;
  readonly pass?: boolean;
  readonly failures?: string[];
  readonly frozen?: {
    readonly gates: CohortGateMetrics;
    readonly pass: boolean;
    readonly failures: string[];
  };
  readonly deltaVsFrozenPp?: number;
}

function fmt(n: number, digits = 2): string {
  return Number.isFinite(n) ? n.toFixed(digits) : "n/a";
}

function knobs(r: RefitRow): string {
  if (!r.fitted) return "n/a";
  const f = r.fitted;
  return `step=${f.gridStepPct} grids=${f.gridMaxGrids} pause=${f.gridPauseAfterLossBars} target=${f.targetRatio} adx=${f.chopGateAdxThreshold}`;
}

function frozenVerdictOf(r: RefitRow): string {
  if (!r.frozen) return "n/a";
  return r.frozen.pass ? "PASS" : `FAIL(${r.frozen.failures.join(",")})`;
}

function selCells(r: RefitRow): string {
  const s = r.selection;
  if (!s) return "n/a | n/a | n/a | n/a";
  return `${fmt(s.profitableWindowPct, 1)} | ${fmt(s.compoundedReturnPct)} | ${fmt(s.maxDrawdownPct)} | ${s.pass ? "SEL-PASS" : "SEL-FAIL"}`;
}

function nonOkLine(r: RefitRow): string {
  return `| ${r.symbol} | ${knobs(r)} | ${selCells(r)} | — | — | — | — | — | — | — | — | — | — | — | — | ${r.status}${r.detail ? `: ${r.detail}` : ""} | ${frozenVerdictOf(r)} | — |`;
}

function gateCell(r: RefitRow): string {
  if (r.pass) return "PASS";
  return `FAIL(${(r.failures ?? []).join(",")})`;
}

function okLine(r: RefitRow): string {
  const g = r.gates;
  if (!g) return nonOkLine(r);
  return `| ${r.symbol} | ${knobs(r)} | ${selCells(r)} | ${fmt(g.profitableWindowPct, 1)} | ${fmt(g.compoundedReturnPct)} | ${fmt(g.maxDrawdownPct)} | ${g.fixedOosTrades} | ${fmt(r.expectancyPct ?? 0, 5)} | ${fmt(r.tradesPerMonth ?? 0, 1)} | ${fmt(r.oosWinRatePct ?? 0, 1)} | ${fmt(r.oosReturnPct ?? 0)} | ${fmt(g.confidenceLowerBoundPct, 5)} | ${fmt(g.stressWorstReturnPct)} | ${fmt(g.stressLowerBoundPct, 5)} | ${gateCell(r)} | ${frozenVerdictOf(r)} | ${r.deltaVsFrozenPp !== undefined ? fmt(r.deltaVsFrozenPp) : "n/a"} |`;
}

function toMarkdown(generatedAt: string, rows: RefitRow[]): string {
  const ok = rows.filter((r) => r.status === "ok");
  const passCount = ok.filter((r) => r.pass).length;
  const lines: string[] = [];
  lines.push(`# Growth E4 per-symbol grid refit — ${VENUE} 15m`);
  lines.push(``);
  lines.push(`Generated: ${generatedAt}`);
  lines.push(
    `Task: clever-cabin-6a8 (follow-up to E3 clever-cabin-lnj, 0/24 PASS).`,
  );
  lines.push(
    `Refit symbols: ${REFIT_SYMBOLS.join(", ")} (E3 closest-but-failing).`,
  );
  lines.push(
    `Selection (no tail peek): first 80% of candles per symbol, walk-forward train=${TRAIN_BARS} test=${TEST_BARS}; ${CANDIDATE_COUNT} structural candidates per symbol, fees frozen (maker 0.02 / takerExit 0.06 / slip 1bps / lev 1 / posFrac 1).`,
  );
  lines.push(
    `Final eval (frozen, SAME as E3): validateGridEvidence last-20% tail + 5-seed stress + passesCohortGate (imported from scripts/growth-e3-cohort-expansion.ts).`,
  );
  lines.push(
    `Refit: ${rows.length} symbols | valid: ${ok.length} | PASS: ${passCount}`,
  );
  lines.push(``);
  lines.push(
    `| Symbol | Fitted knobs | SelWin% | SelRet% | SelDD% | Sel | WinWin% | HistRet% | MaxDD% | OOS n | Expect%/tr | Trades/mo | OOS win% | OOS ret% | ConfLB | StressWorst | StressLB | Gate | Frozen | Δpp |`,
  );
  lines.push(
    `|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|`,
  );
  for (const r of rows) {
    lines.push(r.status === "ok" ? okLine(r) : nonOkLine(r));
  }
  lines.push(``);
  lines.push(
    `Notes: CLAIM bars (goals.ts 0.08/0.001) untouched — no promotion implied. Expectancy>=0.001 column (Expect%/tr) is informational vs the expectancy claim bar. DB opened read-only; only these two result files written.`,
  );
  lines.push(``);
  return lines.join("\n");
}
type EvidenceEval = ReturnType<typeof validateGridEvidence>;
type OkEvidence = Extract<EvidenceEval, { kind: "ok" }>;

function frozenGatesFor(v: EvidenceEval): CohortGateMetrics | null {
  if (v.kind !== "ok") return null;
  return {
    profitableWindowPct: v.historical.profitableWindowPct,
    compoundedReturnPct: v.historical.compoundedReturnPct,
    maxDrawdownPct: v.historical.maximumDrawdownPct,
    fixedOosTrades: v.fixedOos.totalTrades,
    confidenceLowerBoundPct: v.confidence.lowerBoundPct,
    stressWorstReturnPct: v.stress.worstReturnPct,
    stressLowerBoundPct: v.stress.pooledLowerBoundPct,
    windowCount: v.historical.windows.length,
  };
}

function fittedKnobsOf(
  winner: RankedCandidate,
): NonNullable<RefitRow["fitted"]> {
  return {
    gridStepPct: winner.grid.gridStepPct,
    gridMaxGrids: winner.grid.gridMaxGrids,
    gridPauseAfterLossBars: winner.grid.gridPauseAfterLossBars,
    targetRatio: winner.grid.targetRatio ?? 1.95,
    chopGateAdxThreshold: winner.grid.chopGateAdxThreshold ?? 0,
    candidateIndex: winner.index,
  };
}

function frozenSliceOf(frozenEval: EvidenceEval): RefitRow["frozen"] {
  const fg = frozenGatesFor(frozenEval);
  if (!fg) return undefined;
  const fv = passesCohortGate(fg);
  return { gates: fg, pass: fv.pass, failures: fv.failures };
}

function countBars(db: Database, symbol: string): number {
  return (
    db
      .query(
        `SELECT COUNT(*) AS n FROM ohlcv_data o
         JOIN exchanges e ON e.id = o.exchange_id
         JOIN trading_pairs tp ON tp.id = o.trading_pair_id
         WHERE e.name = ? AND tp.symbol = ? AND o.timeframe = ?`,
      )
      .get(VENUE, symbol, TIMEFRAME) as { n: number }
  ).n;
}

function loadCandles(db: Database, symbol: string): CandleLike[] {
  const raw = db
    .query(
      `SELECT o.open_price AS open, o.high_price AS high, o.low_price AS low,
              o.close_price AS close, o.volume AS volume, o.timestamp AS ts
       FROM ohlcv_data o JOIN exchanges e ON e.id = o.exchange_id
       JOIN trading_pairs tp ON tp.id = o.trading_pair_id
       WHERE e.name = ? AND tp.symbol = ? AND o.timeframe = ?
       ORDER BY o.timestamp ASC`,
    )
    .all(VENUE, symbol, TIMEFRAME) as Array<{
    open: number;
    high: number;
    low: number;
    close: number;
    volume: number;
    ts: string;
  }>;
  return raw.map((r) => ({
    open: r.open,
    high: r.high,
    low: r.low,
    close: r.close,
    volume: r.volume,
    timestamp: new Date(Date.parse(r.ts)),
  }));
}

export interface SelectionWinner {
  readonly winner: RankedCandidate;
  readonly selPass: boolean;
}

function selectWinner(
  candles: readonly CandleLike[],
  candidates: readonly GridOptions[],
): SelectionWinner {
  const selection = candles.slice(0, oosStartFor(candles.length));
  const ranked: RankedCandidate[] = candidates.map((grid, index) => {
    const metrics = selectionWalkForward(selection, grid);
    return { index, grid, metrics, eligible: passesSelectionGate(metrics) };
  });
  const winner = rankSelection(ranked);
  return { winner, selPass: passesSelectionGate(winner.metrics) };
}

function invalidRow(
  symbol: string,
  candleCount: number,
  detail: string,
  winner: RankedCandidate,
  selPass: boolean,
  frozenEval: EvidenceEval,
): RefitRow {
  return {
    symbol,
    status: "invalid",
    candles: candleCount,
    detail,
    fitted: fittedKnobsOf(winner),
    selection: { ...winner.metrics, pass: selPass },
    frozen: frozenSliceOf(frozenEval),
  };
}

function okRow(
  symbol: string,
  candleCount: number,
  winner: RankedCandidate,
  selPass: boolean,
  fittedEval: OkEvidence,
  frozenEval: EvidenceEval,
): RefitRow {
  const gates: CohortGateMetrics = {
    profitableWindowPct: fittedEval.historical.profitableWindowPct,
    compoundedReturnPct: fittedEval.historical.compoundedReturnPct,
    maxDrawdownPct: fittedEval.historical.maximumDrawdownPct,
    fixedOosTrades: fittedEval.fixedOos.totalTrades,
    confidenceLowerBoundPct: fittedEval.confidence.lowerBoundPct,
    stressWorstReturnPct: fittedEval.stress.worstReturnPct,
    stressLowerBoundPct: fittedEval.stress.pooledLowerBoundPct,
    windowCount: fittedEval.historical.windows.length,
  };
  const { pass, failures } = passesCohortGate(gates);
  const frozen = frozenSliceOf(frozenEval);
  return {
    symbol,
    status: "ok",
    candles: candleCount,
    fitted: fittedKnobsOf(winner),
    selection: { ...winner.metrics, pass: selPass },
    expectancyPct: meanTradeExpectancyPct(fittedEval.fixedOos),
    tradesPerMonth: tradesPerMonth(
      fittedEval.fixedOos.totalTrades,
      candleCount,
    ),
    oosWinRatePct: fittedEval.fixedOos.winRate,
    oosReturnPct: fittedEval.fixedOos.totalReturnPct,
    profitFactor: Number.isFinite(fittedEval.fixedOos.profitFactor)
      ? fittedEval.fixedOos.profitFactor
      : "Infinity",
    gates,
    pass,
    failures,
    frozen,
    deltaVsFrozenPp: frozen
      ? gates.compoundedReturnPct - frozen.gates.compoundedReturnPct
      : undefined,
  };
}

function processSymbol(
  db: Database,
  symbol: string,
  candidates: readonly GridOptions[],
): RefitRow {
  const count = countBars(db, symbol);
  if (!(count >= MIN_CANDLES)) {
    console.log(`${symbol}: VENUE-FILTERED (${count} bars)`);
    return {
      symbol,
      status: "venue-filtered",
      candles: count,
      detail: `below min-candles ${MIN_CANDLES}`,
    };
  }
  const candles = loadCandles(db, symbol);
  const { winner, selPass } = selectWinner(candles, candidates);
  console.log(
    `${symbol}: selection winner #${winner.index} step=${winner.grid.gridStepPct} grids=${winner.grid.gridMaxGrids} pause=${winner.grid.gridPauseAfterLossBars} target=${winner.grid.targetRatio} adx=${winner.grid.chopGateAdxThreshold} | sel win=${winner.metrics.profitableWindowPct.toFixed(1)}% ret=${winner.metrics.compoundedReturnPct.toFixed(2)}% dd=${winner.metrics.maxDrawdownPct.toFixed(2)}% n=${winner.metrics.windowCount} ${selPass ? "SEL-PASS" : "SEL-FAIL"}`,
  );
  const now = new Date(
    candles.at(-1)!.timestamp.getTime() + TIMEFRAME_MINUTES * 60 * 1000,
  );
  const evalGrid = (grid: GridOptions): EvidenceEval =>
    validateGridEvidence(candles, {
      now,
      timeframeMinutes: TIMEFRAME_MINUTES,
      grid,
      executionParityPassed: true,
    });
  const fittedEval = evalGrid(winner.grid);
  const frozenEval = evalGrid(FROZEN_CHAMPION_GRID);
  if (fittedEval.kind !== "ok") {
    console.log(`${symbol}: INVALID -> ${fittedEval.failures.join("; ")}`);
    return invalidRow(
      symbol,
      candles.length,
      `fitted: ${fittedEval.failures.join("; ")}`,
      winner,
      selPass,
      frozenEval,
    );
  }
  const row = okRow(
    symbol,
    candles.length,
    winner,
    selPass,
    fittedEval,
    frozenEval,
  );
  console.log(
    `${symbol}: ${row.pass ? "PASS" : "FAIL(" + (row.failures ?? []).join(",") + ")"} | win=${row.gates?.profitableWindowPct.toFixed(1)}% ret=${row.gates?.compoundedReturnPct.toFixed(2)}% dd=${row.gates?.maxDrawdownPct.toFixed(2)}% oos=${row.gates?.fixedOosTrades} exp=${(row.expectancyPct ?? 0).toFixed(5)}%/tr`,
  );
  return row;
}

async function main(): Promise<void> {
  const flag = "--symbols=";
  const onlyRaw =
    process.argv.find((a) => a.startsWith(flag))?.slice(flag.length) ?? "";
  const only = onlyRaw
    ? onlyRaw.split(",").map((s) => s.trim())
    : [...REFIT_SYMBOLS];
  const home = process.env.NEURATRADE_HOME ?? `${process.env.HOME}/.neuratrade`;
  const db = new Database(`${home}/data/neuratrade.db`, { readonly: true });
  const candidates = buildCandidates();
  const t0 = Date.now();
  const rows = only.map((symbol) => processSymbol(db, symbol, candidates));
  db.close();
  const generatedAt = new Date().toISOString();
  const payload = {
    generatedAt,
    task: "growth-E4-per-symbol-refit (clever-cabin-6a8)",
    parent: "growth-E3-cohort-expansion (clever-cabin-lnj), 0/24 PASS",
    venue: VENUE,
    timeframe: TIMEFRAME,
    minCandles: MIN_CANDLES,
    refitSymbols: [...REFIT_SYMBOLS],
    candidateCount: CANDIDATE_COUNT,
    selection:
      "first 80% of candles per symbol, walk-forward train=11520 test=4320; eligible-first rank (win>=50, ret>0, dd<=15) then compounded desc, dd asc, index asc",
    frozenFees: {
      feePct: 0.02,
      takerExitFeePct: 0.06,
      slippageBps: 1,
      leverage: 1,
      positionFraction: 1,
    },
    frozenGrid: { ...FROZEN_CHAMPION_GRID },
    holdout:
      "validateGridEvidence fixed last-20% OOS + 5-seed stress + passesCohortGate imported from scripts/growth-e3-cohort-expansion.ts (frozen, identical to E3)",
    elapsedSec: (Date.now() - t0) / 1000,
    screened: rows.length,
    valid: rows.filter((x) => x.status === "ok").length,
    passing: rows.filter((x) => x.pass).length,
    rows,
  };
  const outDir = join(import.meta.dir, "..", "autoresearch", "results");
  mkdirSync(outDir, { recursive: true });
  const jsonPath = join(outDir, "growth-e4-per-symbol-refit.json");
  const mdPath = join(outDir, "growth-e4-per-symbol-refit.md");
  writeFileSync(jsonPath, JSON.stringify(payload, null, 2));
  writeFileSync(mdPath, toMarkdown(generatedAt, rows));
  console.log(
    `\nPASS ${rows.filter((x) => x.pass).length}/${rows.length} | results: ${jsonPath}\ntable:   ${mdPath}`,
  );
}

if (import.meta.main) {
  await main();
}
