#!/usr/bin/env bun
/**
 * Growth E1 sizing sweep (clever-cabin-cvk) — READ-ONLY analysis.
 *
 * What it does: sweeps positionFraction x leverage on the FROZEN champion
 * knobs (autoresearch/results/champion-soak.json) under account-DD kill
 * budgets of 15/30/50% and reports growth-rate vs maxDD vs ruin per budget.
 *
 * What it never does (additive-only contract):
 * - never writes knobs.ts, champion-soak.json, champion.json, claimed.json,
 *   live workers, DBs, or CLAIM bars (goals.ts untouched);
 * - never promotes, keeps, or claims — it only REPORTS holdout numbers.
 *
 * Method (mirrors autoresearch/prepare.ts without importing its hardcoded
 * leverage=1 path):
 * - honest venue fees: maker 0.02% per side, taker 0.06% on stop/
 *   liquidation/max-hold exits, 2bps slippage, conservative intrabar;
 * - frozen holdout convention: exactly ONE trailing window per symbol over
 *   the reserved tail (panel.holdoutBars); selection-prefix bars are never
 *   scored, so a second look at the holdout cannot leak into selection;
 * - DD budget N maps to the ladder engine's account kill maxDrawdownPct=N
 *   (pause-then-retry kill, NOT a permanent death — ruin is measured
 *   separately, see isRuined);
 * - growth-rate = median log-return per holdout window,
 *   maxDD = median window maxDrawdownPct (true peak-to-trough),
 *   ruin = share of windows with totalReturnPct <= -budget, any liquidation,
 *   or total loss (<= -99%).
 */

import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  runLadderGridBacktest,
  type LadderOptions,
} from "../src/scalping/ladder-grid.ts";
import {
  loadAlignedPanel,
  toDatasetProvenance,
  type AlignedPanel,
} from "./prepare.ts";
import type { AutoresearchKnobs } from "./knobs.ts";

/** Honest venue fees — must match prepare.ts (maker 0.02 / taker-exit 0.06). */
export const HONEST_MAKER_FEE_PCT = 0.02;
export const HONEST_TAKER_EXIT_FEE_PCT = 0.06;
export const HONEST_SLIPPAGE_BPS = 2;
export const SWEEP_INITIAL_CAPITAL = 10_000;

/** Account-DD kill budgets under test (percent below peak). */
export const DD_BUDGETS_PCT = [15, 30, 50] as const;
/** Sizing grid: fraction of equity across all rungs x leverage. */
export const SWEEP_POSITION_FRACTIONS = [0.25, 0.5, 1] as const;
export const SWEEP_LEVERAGES = [1, 2, 3] as const;

export interface SizingCell {
  readonly positionFraction: number;
  readonly leverage: number;
  readonly maxDrawdownPct: number;
}

export interface WindowOutcome {
  readonly symbol: string;
  readonly logReturn: number;
  readonly returnPct: number;
  readonly maxDrawdownPct: number;
  readonly trades: number;
  readonly ruined: boolean;
  readonly liquidations: number;
}

export interface BudgetRow {
  readonly budgetPct: number;
  readonly positionFraction: number;
  readonly leverage: number;
  readonly windows: number;
  /** Median log-return per holdout window (growth-rate). */
  readonly growthLogRet: number;
  readonly medianReturnPct: number;
  readonly medianMaxDrawdownPct: number;
  /** Share of holdout windows ruined (0..1). */
  readonly ruinRate: number;
  readonly totalTrades: number;
  readonly withinBudget: boolean;
}

export interface SweepReport {
  readonly generatedAt: string;
  readonly base: AutoresearchKnobs;
  readonly dataset: ReturnType<typeof toDatasetProvenance>;
  readonly honestFees: {
    readonly makerFeePct: number;
    readonly takerExitFeePct: number;
    readonly slippageBps: number;
  };
  readonly budgets: readonly number[];
  readonly rows: readonly BudgetRow[];
  /** Max-growth row per budget whose median DD stays within budget (if any). */
  readonly recommended: Readonly<Record<number, BudgetRow | null>>;
}

export function median(xs: readonly number[]): number {
  const s = xs.filter(Number.isFinite).sort((a, b) => a - b);
  if (s.length === 0) return Number.NaN;
  const m = Math.floor(s.length / 2);
  return s.length % 2 === 1 ? s[m]! : (s[m - 1]! + s[m]!) / 2;
}

/** Log-return of a percent return, floored like prepare.ts. */
export function logReturn(returnPct: number): number {
  return Math.log(1 + Math.max(-0.95, returnPct / 100));
}

/**
 * Frozen champion geometry + one sizing cell. Pure: never mutates `base`.
 * maxDrawdownPct is the engine's pause-then-retry account kill, not ruin.
 */
export function buildSweepBacktestOptions(
  base: AutoresearchKnobs,
  cell: SizingCell,
): LadderOptions {
  return {
    rungs: base.rungs,
    gridStepPct: base.gridStepPct,
    gridMaxGrids: base.gridMaxGrids,
    gridPauseAfterLossBars: base.gridPauseAfterLossBars,
    feePct: HONEST_MAKER_FEE_PCT,
    takerExitFeePct: HONEST_TAKER_EXIT_FEE_PCT,
    slippageBps: HONEST_SLIPPAGE_BPS,
    initialCapital: SWEEP_INITIAL_CAPITAL,
    leverage: cell.leverage,
    trendFilterPeriod: base.trendFilterPeriod,
    stopRatio: base.stopRatio,
    targetRatio: base.targetRatio,
    maxHoldBars: base.maxHoldBars,
    chopGateAdxThreshold: base.chopGateAdxThreshold,
    positionFraction: cell.positionFraction,
    maxDrawdownPct: cell.maxDrawdownPct,
    conservativeIntrabar: true,
  };
}

/** Frozen holdout slice: the reserved trailing tail for one symbol. */
export function holdoutSlice(panel: AlignedPanel, symbol: string) {
  const candles = panel.aligned.get(symbol) ?? [];
  const tail = panel.holdoutBars;
  return candles.slice(Math.max(0, candles.length - tail));
}

/**
 * Ruin per holdout window: breached the DD budget, any liquidation trade,
 * or total loss. Independent of the engine's pause-then-retry kill, which
 * re-anchors and keeps trading instead of dying.
 */
export function isRuined(
  totalReturnPct: number,
  liquidations: number,
  budgetPct: number,
): boolean {
  if (!Number.isFinite(totalReturnPct)) return true;
  if (liquidations > 0) return true;
  if (totalReturnPct <= -budgetPct) return true;
  return totalReturnPct <= -99;
}

function evaluateCellOnHoldout(
  panel: AlignedPanel,
  base: AutoresearchKnobs,
  cell: SizingCell,
): WindowOutcome[] {
  const opts = buildSweepBacktestOptions(base, cell);
  const out: WindowOutcome[] = [];
  for (const symbol of panel.symbols) {
    const slice = holdoutSlice(panel, symbol);
    if (slice.length < panel.holdoutBars * 0.9) continue;
    try {
      const r = runLadderGridBacktest(slice, opts);
      const liquidations = r.trades.filter((t) => t.isLiquidation).length;
      out.push({
        symbol,
        logReturn: logReturn(r.totalReturnPct),
        returnPct: r.totalReturnPct,
        maxDrawdownPct: r.maxDrawdownPct,
        trades: r.trades.length,
        ruined: isRuined(r.totalReturnPct, liquidations, cell.maxDrawdownPct),
        liquidations,
      });
    } catch {
      // One bad window never fails the sweep; it counts as ruined.
      out.push({
        symbol,
        logReturn: Number.NaN,
        returnPct: Number.NaN,
        maxDrawdownPct: Number.NaN,
        trades: 0,
        ruined: true,
        liquidations: 0,
      });
    }
  }
  return out;
}

function summarizeCell(
  budgetPct: number,
  cell: SizingCell,
  outcomes: readonly WindowOutcome[],
): BudgetRow {
  const growthLogRet = median(outcomes.map((o) => o.logReturn));
  const medianMaxDrawdownPct = median(outcomes.map((o) => o.maxDrawdownPct));
  const ruined = outcomes.filter((o) => o.ruined).length;
  return {
    budgetPct,
    positionFraction: cell.positionFraction,
    leverage: cell.leverage,
    windows: outcomes.length,
    growthLogRet,
    medianReturnPct: median(outcomes.map((o) => o.returnPct)),
    medianMaxDrawdownPct,
    ruinRate: outcomes.length > 0 ? ruined / outcomes.length : Number.NaN,
    totalTrades: outcomes.reduce((sum, o) => sum + o.trades, 0),
    withinBudget:
      Number.isFinite(medianMaxDrawdownPct) &&
      medianMaxDrawdownPct <= budgetPct,
  };
}

/** Full grid per budget on the frozen holdout tail. Pure (no IO). */
export function runGrowthSizingSweep(
  panel: AlignedPanel,
  base: AutoresearchKnobs,
  budgets: readonly number[] = DD_BUDGETS_PCT,
  positionFractions: readonly number[] = SWEEP_POSITION_FRACTIONS,
  leverages: readonly number[] = SWEEP_LEVERAGES,
): SweepReport {
  const rows: BudgetRow[] = [];
  for (const budgetPct of budgets) {
    for (const positionFraction of positionFractions) {
      for (const leverage of leverages) {
        const cell: SizingCell = {
          positionFraction,
          leverage,
          maxDrawdownPct: budgetPct,
        };
        rows.push(
          summarizeCell(
            budgetPct,
            cell,
            evaluateCellOnHoldout(panel, base, cell),
          ),
        );
      }
    }
  }
  const recommended: Record<number, BudgetRow | null> = {};
  for (const budgetPct of budgets) {
    const eligible = rows.filter(
      (r) =>
        r.budgetPct === budgetPct &&
        r.withinBudget &&
        Number.isFinite(r.growthLogRet),
    );
    eligible.sort((a, b) => b.growthLogRet - a.growthLogRet);
    recommended[budgetPct] = eligible[0] ?? null;
  }
  return {
    generatedAt: new Date().toISOString(),
    base: { ...base },
    dataset: toDatasetProvenance(panel),
    honestFees: {
      makerFeePct: HONEST_MAKER_FEE_PCT,
      takerExitFeePct: HONEST_TAKER_EXIT_FEE_PCT,
      slippageBps: HONEST_SLIPPAGE_BPS,
    },
    budgets: [...budgets],
    rows,
    recommended,
  };
}

function fmt(n: number, digits = 4): string {
  return Number.isFinite(n) ? n.toFixed(digits) : "n/a";
}

/** Markdown growth-vs-DD-vs-ruin table, one section per budget. */
export function renderMarkdownTable(report: SweepReport): string {
  const lines: string[] = [
    "# Growth E1 sizing sweep (frozen champion, holdout tail)",
    "",
    `- base: rungs=${report.base.rungs} step=${report.base.gridStepPct} grids=${report.base.gridMaxGrids} stop=${report.base.stopRatio} target=${report.base.targetRatio} hold=${report.base.maxHoldBars} pf=${report.base.positionFraction}`,
    `- dataset: ${report.dataset.exchange}/${report.dataset.timeframe}->${report.dataset.panelTimeframe} hash=${report.dataset.panelHash} symbols=${report.dataset.symbols.length} holdoutBars=${report.dataset.holdoutBars}`,
    `- honest fees: maker ${report.honestFees.makerFeePct}% / taker-exit ${report.honestFees.takerExitFeePct}% / slip ${report.honestFees.slippageBps}bps`,
    `- ruin = window ret <= -budget OR any liquidation OR total loss`,
    "",
  ];
  for (const budget of report.budgets) {
    lines.push(`## DD budget ${budget}% (account kill = ${budget}%)`);
    lines.push("");
    lines.push(
      "| posFrac | lev | growthLogRet | medRetPct | medMaxDD% | ruinRate | trades | withinBudget |",
    );
    lines.push("| --- | --- | --- | --- | --- | --- | --- | --- |");
    for (const r of report.rows.filter((x) => x.budgetPct === budget)) {
      lines.push(
        `| ${r.positionFraction} | ${r.leverage} | ${fmt(r.growthLogRet)} | ${fmt(r.medianReturnPct, 2)} | ${fmt(r.medianMaxDrawdownPct, 2)} | ${fmt(r.ruinRate, 2)} | ${r.totalTrades} | ${r.withinBudget ? "yes" : "no"} |`,
      );
    }
    const rec = report.recommended[budget];
    lines.push("");
    if (rec) {
      lines.push(
        `Recommended @${budget}%: posFrac=${rec.positionFraction} lev=${rec.leverage} growth=${fmt(rec.growthLogRet)} dd=${fmt(rec.medianMaxDrawdownPct, 2)} ruin=${fmt(rec.ruinRate, 2)}.`,
      );
    } else {
      lines.push(`Recommended @${budget}%: none within budget.`);
    }
    lines.push("");
  }
  return lines.join("\n");
}

function arg(name: string, fallback: string): string {
  const hit = process.argv.find((a) => a.startsWith(`--${name}=`));
  return hit?.split("=")[1] ?? fallback;
}

function loadFrozenChampionKnobs(championPath: string): AutoresearchKnobs {
  const raw = JSON.parse(readFileSync(championPath, "utf8")) as {
    knobs: AutoresearchKnobs;
  };
  if (!raw?.knobs) throw new Error(`no knobs in ${championPath}`);
  return { ...raw.knobs };
}

if (import.meta.main) {
  const here = dirname(fileURLToPath(import.meta.url));
  const championPath = arg(
    "champion",
    join(here, "results", "champion-soak.json"),
  );
  const outJson = arg("out", join(here, "results", "growth-sizing-sweep.json"));
  const base = loadFrozenChampionKnobs(championPath);
  const panel = loadAlignedPanel({
    symbols: Number(arg("symbols", "8")),
    dbPath: arg("db", "") || undefined,
  });
  const report = runGrowthSizingSweep(panel, base);
  mkdirSync(dirname(outJson), { recursive: true });
  writeFileSync(outJson, `${JSON.stringify(report, null, 2)}\n`);
  const md = renderMarkdownTable(report);
  writeFileSync(outJson.replace(/\.json$/, ".md"), `${md}\n`);
  console.log(md);
}
