// Growth E3 cohort expansion (clever-cabin-lnj) — ADDITIVE-ONLY screen.
//
// Screens a wider Bybit-futures 15m pair cohort with the FROZEN champion grid
// logic + honest fees. No fitting per symbol: ONE frozen GridOptions runs
// through the frozen holdout gate (validateGridEvidence) per candidate.
//
// Frozen champion source: autoresearch/results/champion-soak.json
//   grid-relevant knobs: gridStepPct 1.3, gridMaxGrids 2,
//   gridPauseAfterLossBars 2, targetRatio 1.95, chopGateAdxThreshold 0,
//   trendFilterPeriod 0, positionFraction 1.
// Honest fees (program.md: maker 0.02, taker-exit 0.06):
//   feePct 0.02, takerExitFeePct 0.06, slippageBps 1.
// Gate mirrors scripts/verify-cohort-on-bybit.ts (Bybit-alignment gate).
//
//venue-filtered: exchange must be exactly "bybit-futures", timeframe 15m,
//   >= MIN_CANDLES bars, else the row is reported as venue-filtered.
// Frozen holdout: validateGridEvidence fixed OOS = last 20% of candles +
//   5-seed stress (makerFillProb 0.7, adverse selection). Never refit.
//
// ADDITIVE ONLY: opens the SQLite DB read-only, writes ONLY the two new
// result files below, never touches live workers, DBs, champion files,
// CLAIM gates, or ecosystem configs.
//
// Usage:
//   bun run scripts/growth-e3-cohort-expansion.ts [--min-candles=55000]
//
// Output (new files, additive):
//   autoresearch/results/growth-e3-cohort-expansion.json
//   autoresearch/results/growth-e3-cohort-expansion.md
import { Database } from "bun:sqlite";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { validateGridEvidence } from "../src/scalping/grid-validation.js";
import type { GridOptions, GridResult } from "../src/scalping/grid.js";

/** Frozen champion grid mapping — see header for the source knobs. */
export const FROZEN_CHAMPION_GRID: GridOptions = {
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
export const FROZEN_CHAMPION_SOURCE =
  "autoresearch/results/champion-soak.json (grid-relevant knobs + honestFees maker0.02/takerExit0.06)";

export const VENUE = "bybit-futures";
export const TIMEFRAME = "15m";
export const TIMEFRAME_MINUTES = 15;
export const MIN_CANDLES = 55000;
export const HOLDOUT_FRACTION = 0.2;
export const MINUTES_PER_MONTH = 43200;
export const BASELINE_SYMBOLS = ["BTC/USDT:USDT", "ETH/USDT:USDT"] as const;

export interface CohortGateMetrics {
  readonly profitableWindowPct: number;
  readonly compoundedReturnPct: number;
  readonly maxDrawdownPct: number;
  readonly fixedOosTrades: number;
  readonly confidenceLowerBoundPct: number;
  readonly stressWorstReturnPct: number;
  readonly stressLowerBoundPct: number;
  readonly windowCount: number;
}

export interface CohortRow {
  readonly symbol: string;
  readonly status: "ok" | "invalid" | "venue-filtered";
  readonly candles?: number;
  readonly detail?: string;
  readonly expectancyPct?: number;
  readonly tradesPerMonth?: number;
  readonly oosWinRatePct?: number;
  readonly oosReturnPct?: number;
  readonly profitFactor?: number | string;
  readonly gates?: CohortGateMetrics;
  readonly pass?: boolean;
  readonly failures?: string[];
}

/** Mean per-trade expectancy in percent (pnlPct is a fraction). */
export function meanTradeExpectancyPct(result: GridResult): number {
  if (result.trades.length === 0) return 0;
  const sum = result.trades.reduce((acc, t) => acc + t.pnlPct, 0);
  return (sum / result.trades.length) * 100;
}

/** Fixed-OOS throughput: trades per 30-day month over the holdout slice. */
export function tradesPerMonth(
  totalTrades: number,
  candleCount: number,
  holdoutFraction = HOLDOUT_FRACTION,
): number {
  const months =
    (holdoutFraction * candleCount * TIMEFRAME_MINUTES) / MINUTES_PER_MONTH;
  return months > 0 ? totalTrades / months : 0;
}

/** Venue filter: exact venue + enough 15m history. */
export function venueEligible(
  exchange: string,
  timeframe: string,
  candleCount: number,
  minCandles = MIN_CANDLES,
): boolean {
  return (
    exchange === VENUE && timeframe === TIMEFRAME && candleCount >= minCandles
  );
}

/** Bybit-alignment gate (mirrors scripts/verify-cohort-on-bybit.ts). */
export interface CohortGateVerdict {
  readonly pass: boolean;
  readonly failures: string[];
}
export function passesCohortGate(g: CohortGateMetrics): CohortGateVerdict {
  const failures: string[] = [];
  if (g.profitableWindowPct < 50) failures.push("windows<50%");
  if (g.compoundedReturnPct <= 0) failures.push("compounded<=0");
  if (g.maxDrawdownPct > 15) failures.push("dd>15%");
  if (g.fixedOosTrades < 30) failures.push("oos<30");
  if (g.confidenceLowerBoundPct < 0) failures.push("confLB<0");
  if (g.stressWorstReturnPct < 0) failures.push("stressRet<0");
  if (g.stressLowerBoundPct < 0) failures.push("stressLB<0");
  if (g.windowCount < 10) failures.push("windows<10");
  return { pass: failures.length === 0, failures };
}

export interface UnionEstimate {
  readonly symbols: string[];
  readonly meanReturnPct: number;
}

/**
 * Union portfolio growth estimate: equal-weight mean of per-symbol
 * walk-forward compounded returns. Each symbol runs the same frozen grid on
 * its own venue-filtered history, so the mean is the expected portfolio
 * return for capital split equally across the member symbols.
 */
export function unionGrowthEstimate(rows: readonly CohortRow[]): UnionEstimate {
  const members = rows.filter(
    (r): r is CohortRow & { gates: CohortGateMetrics } =>
      r.status === "ok" && r.pass === true && r.gates !== undefined,
  );
  const symbols = members.map((r) => r.symbol);
  const meanReturnPct =
    members.length === 0
      ? 0
      : members.reduce((acc, r) => acc + r.gates.compoundedReturnPct, 0) /
        members.length;
  return { symbols, meanReturnPct };
}

/** Baseline: equal-weight mean over the BTC/ETH baseline symbols (valid rows). */
export function baselineGrowthEstimate(
  rows: readonly CohortRow[],
): UnionEstimate {
  const members = rows.filter(
    (r): r is CohortRow & { gates: CohortGateMetrics } =>
      r.status === "ok" &&
      (BASELINE_SYMBOLS as readonly string[]).includes(r.symbol) &&
      r.gates !== undefined,
  );
  const symbols = members.map((r) => r.symbol);
  const meanReturnPct =
    members.length === 0
      ? 0
      : members.reduce((acc, r) => acc + r.gates.compoundedReturnPct, 0) /
        members.length;
  return { symbols, meanReturnPct };
}

function fmt(n: number, digits = 2): string {
  return Number.isFinite(n) ? n.toFixed(digits) : "n/a";
}

function toMarkdown(
  generatedAt: string,
  rows: CohortRow[],
  union: UnionEstimate,
  baseline: UnionEstimate,
): string {
  const ok = rows.filter((r) => r.status === "ok");
  const passCount = ok.filter((r) => r.pass).length;
  const uplift = union.meanReturnPct - baseline.meanReturnPct;
  const lines: string[] = [];
  lines.push(`# Growth E3 cohort expansion — ${VENUE} 15m`);
  lines.push(``);
  lines.push(`Generated: ${generatedAt}`);
  lines.push(`Frozen champion: ${FROZEN_CHAMPION_SOURCE}`);
  lines.push(
    `Frozen grid: step=${FROZEN_CHAMPION_GRID.gridStepPct} grids=${FROZEN_CHAMPION_GRID.gridMaxGrids} pause=${FROZEN_CHAMPION_GRID.gridPauseAfterLossBars} target=${FROZEN_CHAMPION_GRID.targetRatio} adx=${FROZEN_CHAMPION_GRID.chopGateAdxThreshold} fee=${FROZEN_CHAMPION_GRID.feePct} takerExit=${FROZEN_CHAMPION_GRID.takerExitFeePct} slip=${FROZEN_CHAMPION_GRID.slippageBps}bps posFrac=${FROZEN_CHAMPION_GRID.positionFraction}`,
  );
  lines.push(
    `Holdout (frozen): last 20% of candles per symbol + 5-seed stress; no per-symbol fitting.`,
  );
  lines.push(
    `Venue filter: exchange=${VENUE}, timeframe=${TIMEFRAME}, >= ${MIN_CANDLES} bars.`,
  );
  lines.push(
    `Screened: ${rows.length} symbols | valid: ${ok.length} | PASS: ${passCount}`,
  );
  lines.push(``);
  lines.push(
    `| Symbol | Bars | WinWin% | HistRet% | MaxDD% | OOS n | Expect%/tr | Trades/mo | OOS win% | OOS ret% | ConfLB | StressWorst | StressLB | Gate |`,
  );
  lines.push(`|---|---|---|---|---|---|---|---|---|---|---|---|---|---|`);
  for (const r of rows) {
    if (r.status !== "ok" || !r.gates) {
      lines.push(
        `| ${r.symbol} | ${r.candles ?? "n/a"} | — | — | — | — | — | — | — | — | — | — | — | ${r.status}${r.detail ? `: ${r.detail}` : ""} |`,
      );
      continue;
    }
    const g = r.gates;
    lines.push(
      `| ${r.symbol} | ${r.candles} | ${fmt(g.profitableWindowPct, 1)} | ${fmt(g.compoundedReturnPct)} | ${fmt(g.maxDrawdownPct)} | ${g.fixedOosTrades} | ${fmt(r.expectancyPct ?? 0, 5)} | ${fmt(r.tradesPerMonth ?? 0, 1)} | ${fmt(r.oosWinRatePct ?? 0, 1)} | ${fmt(r.oosReturnPct ?? 0)} | ${fmt(g.confidenceLowerBoundPct, 5)} | ${fmt(g.stressWorstReturnPct)} | ${fmt(g.stressLowerBoundPct, 5)} | ${r.pass ? "PASS" : "FAIL(" + (r.failures ?? []).join(",") + ")"} |`,
    );
  }
  lines.push(``);
  lines.push(
    `## Union portfolio growth estimate (equal-weight mean of PASS walk-forward HistRet%)`,
  );
  lines.push(``);
  lines.push(
    `- PASS members (${union.symbols.length}): ${union.symbols.join(", ") || "none"}`,
  );
  lines.push(`- Union mean return: ${fmt(union.meanReturnPct)}%`);
  lines.push(
    `- Baseline (${baseline.symbols.join(", ") || "none"}): ${fmt(baseline.meanReturnPct)}%`,
  );
  lines.push(`- Uplift vs baseline: ${fmt(uplift)}pp`);
  lines.push(``);
  lines.push(
    `Method: capital split equally across member symbols; portfolio return ~= mean of per-symbol compounded walk-forward returns. Throughput/expectancy are per-symbol (expectancy is scale-invariant; HistRet scales with positionFraction=1 frozen).`,
  );
  lines.push(``);
  return lines.join("\n");
}

async function main(): Promise<void> {
  const arg = (flag: string, fallback: string): string =>
    process.argv.find((a) => a.startsWith(flag))?.slice(flag.length) ??
    fallback;
  const minCandles = Number(arg("--min-candles=", String(MIN_CANDLES)));
  const home = process.env.NEURATRADE_HOME ?? `${process.env.HOME}/.neuratrade`;
  const db = new Database(`${home}/data/neuratrade.db`, { readonly: true });
  const symbols = db
    .query(
      `SELECT tp.symbol AS symbol, COUNT(*) AS n
       FROM ohlcv_data o JOIN exchanges e ON e.id = o.exchange_id
       JOIN trading_pairs tp ON tp.id = o.trading_pair_id
       WHERE e.name = ? AND o.timeframe = ?
       GROUP BY tp.symbol HAVING COUNT(*) >= ? ORDER BY tp.symbol ASC`,
    )
    .all(VENUE, TIMEFRAME, minCandles) as Array<{ symbol: string; n: number }>;
  const rows: CohortRow[] = [];
  const t0 = Date.now();
  for (const { symbol, n } of symbols) {
    if (!venueEligible(VENUE, TIMEFRAME, n, minCandles)) {
      rows.push({
        symbol,
        status: "venue-filtered",
        candles: n,
        detail: `below min-candles ${minCandles}`,
      });
      continue;
    }
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
    const candles = raw.map((r) => ({
      open: r.open,
      high: r.high,
      low: r.low,
      close: r.close,
      volume: r.volume,
      timestamp: new Date(Date.parse(r.ts)),
    }));
    const now = new Date(
      candles.at(-1)!.timestamp.getTime() + TIMEFRAME_MINUTES * 60 * 1000,
    );
    const r = validateGridEvidence(candles, {
      now,
      timeframeMinutes: TIMEFRAME_MINUTES,
      grid: FROZEN_CHAMPION_GRID,
      executionParityPassed: true,
    });
    if (r.kind !== "ok") {
      rows.push({
        symbol,
        status: "invalid",
        candles: candles.length,
        detail: r.failures.join("; "),
      });
      console.log(`${symbol}: INVALID -> ${r.failures.join("; ")}`);
      continue;
    }
    const gates: CohortGateMetrics = {
      profitableWindowPct: r.historical.profitableWindowPct,
      compoundedReturnPct: r.historical.compoundedReturnPct,
      maxDrawdownPct: r.historical.maximumDrawdownPct,
      fixedOosTrades: r.fixedOos.totalTrades,
      confidenceLowerBoundPct: r.confidence.lowerBoundPct,
      stressWorstReturnPct: r.stress.worstReturnPct,
      stressLowerBoundPct: r.stress.pooledLowerBoundPct,
      windowCount: r.historical.windows.length,
    };
    const { pass, failures } = passesCohortGate(gates);
    rows.push({
      symbol,
      status: "ok",
      candles: candles.length,
      expectancyPct: meanTradeExpectancyPct(r.fixedOos),
      tradesPerMonth: tradesPerMonth(r.fixedOos.totalTrades, candles.length),
      oosWinRatePct: r.fixedOos.winRate,
      oosReturnPct: r.fixedOos.totalReturnPct,
      profitFactor: Number.isFinite(r.fixedOos.profitFactor)
        ? r.fixedOos.profitFactor
        : "Infinity",
      gates,
      pass,
      failures,
    });
    console.log(
      `${symbol}: ${pass ? "PASS" : "FAIL(" + failures.join(",") + ")"} | win=${gates.profitableWindowPct.toFixed(1)}% ret=${gates.compoundedReturnPct.toFixed(2)}% dd=${gates.maxDrawdownPct.toFixed(2)}% oos=${gates.fixedOosTrades} exp=${meanTradeExpectancyPct(r.fixedOos).toFixed(5)}%/tr tpm=${tradesPerMonth(r.fixedOos.totalTrades, candles.length).toFixed(1)}`,
    );
  }
  db.close();
  const union = unionGrowthEstimate(rows);
  const baseline = baselineGrowthEstimate(rows);
  const generatedAt = new Date().toISOString();
  const payload = {
    generatedAt,
    task: "growth-E3-cohort-expansion (clever-cabin-lnj)",
    venue: VENUE,
    timeframe: TIMEFRAME,
    minCandles,
    frozenChampionSource: FROZEN_CHAMPION_SOURCE,
    frozenGrid: { ...FROZEN_CHAMPION_GRID },
    holdout:
      "validateGridEvidence fixed last-20% OOS + 5-seed stress (frozen, no per-symbol fitting)",
    elapsedSec: (Date.now() - t0) / 1000,
    screened: rows.length,
    valid: rows.filter((x) => x.status === "ok").length,
    passing: rows.filter((x) => x.pass).length,
    baseline,
    union,
    upliftPp: union.meanReturnPct - baseline.meanReturnPct,
    rows,
  };
  const outDir = join(import.meta.dir, "..", "autoresearch", "results");
  mkdirSync(outDir, { recursive: true });
  const jsonPath = join(outDir, "growth-e3-cohort-expansion.json");
  const mdPath = join(outDir, "growth-e3-cohort-expansion.md");
  writeFileSync(jsonPath, JSON.stringify(payload, null, 2));
  writeFileSync(mdPath, toMarkdown(generatedAt, rows, union, baseline));
  console.log(
    `\nPASS ${rows.filter((x) => x.pass).length}/${rows.length} | baseline ${baseline.meanReturnPct.toFixed(2)}% | union ${union.meanReturnPct.toFixed(2)}% | uplift ${(union.meanReturnPct - baseline.meanReturnPct).toFixed(2)}pp`,
  );
  console.log(`results: ${jsonPath}\ntable:   ${mdPath}`);
}

if (import.meta.main) {
  await main();
}
