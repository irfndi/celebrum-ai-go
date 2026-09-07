#!/usr/bin/env bun
/**
 * THROWAWAY SPIKE — TimesFM OOS scoping diagnostic (E4, clever-cabin-0g7).
 *
 * Research-only. Read-only SQLite access. No imports from src/, no orders,
 * no DB writes, no risk-state changes. Safe to delete at any time.
 *
 * Prints: 15m/5m panel coverage per cohort symbol, the frozen-holdout OOS
 * plan sizes (same oosStart = floor(n*0.8) rule as grid-validation.ts), and
 * the inference latency budget against the 900s bar loop.
 *
 * Usage:
 *   bun run scripts/timesfm-oos-spike.ts [--json]
 */

import { Database } from "bun:sqlite";
import { join } from "node:path";

export const COHORT = [
  "BTC/USDT:USDT",
  "SOL/USDT:USDT",
  "ETH/USDT:USDT",
] as const;
export const EXCHANGE = "bybit-futures";
export const BAR_MS_15M = 900_000;
export const CLIENT_TIMEOUT_MS = 120_000;
export const PAPER_INTERVAL_S = 60;

export interface OosPlan {
  readonly totalBars: number;
  readonly oosStart: number;
  readonly oosBars: number;
  readonly origins: number;
  readonly coveredBars: number;
}

export function planFrozenHoldout(
  totalBars: number,
  contextBars: number,
  horizon: number,
  stepBars: number,
  maxOrigins: number,
): OosPlan {
  if (!Number.isInteger(totalBars) || totalBars < 0)
    throw new Error("totalBars must be a non-negative integer");
  if (!Number.isInteger(contextBars) || contextBars < 1)
    throw new Error("contextBars must be a positive integer");
  if (!Number.isInteger(horizon) || horizon < 1)
    throw new Error("horizon must be a positive integer");
  if (!Number.isInteger(stepBars) || stepBars < 1)
    throw new Error("stepBars must be a positive integer");
  if (!Number.isInteger(maxOrigins) || maxOrigins < 0)
    throw new Error("maxOrigins must be a non-negative integer");
  // Same frozen rule as validateGridEvidence: last 20% tail.
  const oosStart = Math.floor(totalBars * 0.8);
  const oosBars = totalBars - oosStart;
  // Same origin bound as buildTimesFmEvaluationOrigins: last origin needs a
  // full horizon of future (index <= n - horizon - 1), starting at context-1.
  const lastOrigin = totalBars - horizon - 1;
  let origins = 0;
  for (let index = contextBars - 1; index <= lastOrigin; index += stepBars)
    origins += 1;
  // Origins before the holdout belong to in-sample history, not OOS scoring.
  let oosOrigins = 0;
  for (
    let index = Math.max(contextBars - 1, oosStart);
    index <= lastOrigin;
    index += stepBars
  )
    oosOrigins += 1;
  const capped = maxOrigins > 0 ? Math.min(oosOrigins, maxOrigins) : oosOrigins;
  return {
    totalBars,
    oosStart,
    oosBars,
    origins: capped,
    coveredBars: capped * stepBars,
  };
}

export interface LatencyBudget {
  readonly barLoopMs: number;
  readonly mustP95Ms: number;
  readonly shouldTotalMs: number;
  readonly perSymbolMustMs: number;
}

export function latencyBudget(symbols: number): LatencyBudget {
  if (!Number.isInteger(symbols) || symbols < 1)
    throw new Error("symbols must be a positive integer");
  return {
    barLoopMs: BAR_MS_15M,
    mustP95Ms: CLIENT_TIMEOUT_MS,
    shouldTotalMs: PAPER_INTERVAL_S * 1000,
    perSymbolMustMs: Math.floor(CLIENT_TIMEOUT_MS / symbols),
  };
}

interface PanelRow {
  readonly symbol: string;
  readonly timeframe: string;
  readonly bars: number;
  readonly minTs: string;
  readonly maxTs: string;
}

function homeDir(): string {
  return (
    process.env.NEURATRADE_HOME ?? join(process.env.HOME ?? "~", ".neuratrade")
  );
}

function loadPanels(dbPath: string): PanelRow[] {
  const db = new Database(dbPath, { readonly: true });
  try {
    const rows = db
      .query(
        `SELECT tp.symbol AS symbol, o.timeframe AS timeframe, COUNT(*) AS bars,
                MIN(o.timestamp) AS minTs, MAX(o.timestamp) AS maxTs
         FROM ohlcv_data o
         JOIN exchanges e ON e.id = o.exchange_id
         JOIN trading_pairs tp ON tp.id = o.trading_pair_id
         WHERE e.name = ? AND o.timeframe IN ('15m', '5m')
         GROUP BY tp.symbol, o.timeframe`,
      )
      .all(EXCHANGE) as PanelRow[];
    return rows;
  } finally {
    db.close();
  }
}

function main(): void {
  const asJson = process.argv.includes("--json");
  let panels: PanelRow[] = [];
  let dbNote = "";
  try {
    panels = loadPanels(join(homeDir(), "data", "neuratrade.db"));
  } catch (error) {
    dbNote = `panel lookup skipped: ${error instanceof Error ? error.message : String(error)}`;
  }
  const byKey = new Map(panels.map((p) => [`${p.symbol}/${p.timeframe}`, p]));
  const contextBars = 256;
  const horizon = 12;
  const budget = latencyBudget(COHORT.length);
  const symbols = COHORT.map((symbol) => {
    const panel15 = byKey.get(`${symbol}/15m`);
    const panel5 = byKey.get(`${symbol}/5m`);
    const total = panel15?.bars ?? 0;
    return {
      symbol,
      panel15m: panel15 ?? null,
      panel5m: panel5 ?? null,
      oos15m: planFrozenHoldout(total, contextBars, horizon, horizon, 0),
      oos15mCapped96: planFrozenHoldout(
        total,
        contextBars,
        horizon,
        horizon,
        96,
      ).origins,
    };
  });
  const report = {
    note: "THROWAWAY SPIKE — read-only diagnostic, not wired into any trading path.",
    exchange: EXCHANGE,
    contextBars,
    horizon,
    stepBars: horizon,
    dbNote,
    symbols,
    latencyBudget: budget,
    goNoGoReminder: [
      "fixed-OOS trades >= 30 per symbol",
      "model net return + profit factor > baseline after 0.06%/2bps friction",
      "direction accuracy > 50% and >= baseline + 5pp",
      "net-positive under >=3/5 adverse-selection stress seeds",
      "p95 per-request < 120s (MUST), 3-symbol total < 60s (SHOULD)",
      "TimesFM 3.0 commercial/production license cleared, else 2.x re-run",
    ],
  };
  if (asJson) {
    console.log(JSON.stringify(report, null, 2));
    return;
  }
  console.log("TimesFM OOS spike (read-only, throwaway)");
  console.log(
    `exchange=${EXCHANGE} context=${contextBars} horizon=${horizon} step=${horizon}`,
  );
  if (dbNote) console.log(dbNote);
  for (const s of symbols) {
    const p = s.oos15m;
    console.log(
      `${s.symbol}: 15m bars=${s.panel15m?.bars ?? 0} ` +
        `[${s.panel15m?.minTs ?? "?"} .. ${s.panel15m?.maxTs ?? "?"}] ` +
        `| 5m bars=${s.panel5m?.bars ?? 0} ` +
        `| frozen tail start=${p.oosStart} tail bars=${p.oosBars} ` +
        `| OOS origins=${p.origins} (capped96=${s.oos15mCapped96})`,
    );
  }
  console.log(
    `budget: bar loop=${budget.barLoopMs}ms must-p95<${budget.mustP95Ms}ms ` +
      `should-total<${budget.shouldTotalMs}ms`,
  );
}

if (import.meta.main) main();
