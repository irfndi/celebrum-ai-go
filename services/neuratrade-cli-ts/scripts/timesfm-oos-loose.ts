#!/usr/bin/env bun
/**
 * TimesFM LOOSE operating-point OOS re-run (research-only, offline).
 *
 * Prior OOS (docs/TIMESFM_OOS_RESULTS.md) failed gate 1: the tight walk-forward
 * rule (whole q10-q90 band beyond +/-friction) fires 2-7 trades per 1191 frozen
 * origins. This script re-scores the SAME saved frozen-tail observations with a
 * looser quantile-band operating point (band excludes zero: long if q10 > 0,
 * short if q90 < 0) through the SAME 8 go/no-go gates. No new inference, no
 * weights, no orders, no DB writes. Read-only DB access for the grid leg.
 *
 * ADDITIVE ONLY: new file, no src/ changes, never imported by src/.
 */
import { Database } from "bun:sqlite";
import { join } from "node:path";
import { runGridBacktest } from "../src/scalping/grid.js";
import type { GridOptions } from "../src/scalping/grid.js";

export interface LooseGridPolicyResult {
  readonly policy: string;
  readonly totalReturnPct: number;
  readonly maxDrawdownPct: number;
  readonly totalTrades: number;
  readonly winRatePct: number;
  readonly profitFactor: number | null;
}

export interface LooseSeedResult {
  readonly seed: number;
  readonly winner: LooseGridPolicyResult;
  readonly baseline: LooseGridPolicyResult;
}

export interface LooseGridLeg {
  readonly cleanWinner: LooseGridPolicyResult;
  readonly cleanBaseline: LooseGridPolicyResult;
  readonly seeds: readonly LooseSeedResult[];
  readonly positiveSeeds: number;
  readonly gate4pass: boolean;
}

export interface LooseBlockResult {
  readonly block: number;
  readonly origins: number;
  readonly trades: number;
  readonly netReturnPct: number;
}

export interface LooseSymbolResult {
  readonly symbol: string;
  readonly candleCount: number;
  readonly oosStart: number;
  readonly tailBars: number;
  readonly originCount: number;
  readonly frictionCostPct: number;
  readonly baseline: LooseMetrics;
  readonly ladder: Record<string, LooseMetrics>;
  readonly loose: LooseMetrics;
  readonly pointDiagnostic: LooseMetrics;
  readonly blocks: readonly LooseBlockResult[];
  readonly profitableBlocksPct: number;
  readonly grid: LooseGridLeg;
}

export interface LooseReport {
  readonly ok: boolean;
  readonly researchOnly: boolean;
  readonly looseRule: string;
  readonly friction: string;
  readonly symbols: Record<string, LooseSymbolResult>;
}
import {
  candidateForSymbol,
  VALIDATED_BTC_GRID_CANDIDATE,
} from "../src/scalping/grid-candidate.js";

export type TradeDirection = "long" | "short" | "flat";

export interface LooseObservation {
  readonly originIndex: number;
  readonly actualReturnPct: number;
  readonly pointForecastReturnPct: number;
  readonly q10ReturnPct: number | null;
  readonly q90ReturnPct: number | null;
  readonly baselineDirection: TradeDirection;
  readonly baselineNetReturnPct: number;
}

export interface LooseMetrics {
  readonly trades: number;
  readonly coveragePct: number;
  readonly netReturnPct: number;
  readonly profitFactor: number | null;
  readonly winRatePct: number;
  readonly directionAccuracyPct: number;
}

/** Loosened quantile-band rule: the band only needs to exclude zero. */
export function looseDirection(
  q10ReturnPct: number | null,
  q90ReturnPct: number | null,
  thresholdPct: number,
): TradeDirection {
  if (q10ReturnPct !== null && q10ReturnPct > thresholdPct) return "long";
  if (q90ReturnPct !== null && q90ReturnPct < -thresholdPct) return "short";
  return "flat";
}

function signedNet(
  direction: TradeDirection,
  actualReturnPct: number,
  frictionCostPct: number,
): number {
  if (direction === "long") return actualReturnPct - frictionCostPct;
  if (direction === "short") return -actualReturnPct - frictionCostPct;
  return 0;
}

function directionOf(returnPct: number): TradeDirection {
  if (returnPct > 0) return "long";
  if (returnPct < 0) return "short";
  return "flat";
}

export function scoreDirections(
  observations: readonly LooseObservation[],
  directionOfObs: (observation: LooseObservation) => TradeDirection,
  netOfObs: (observation: LooseObservation) => number,
  forecastReturnOf: (observation: LooseObservation) => number,
): LooseMetrics {
  const scored = observations.map((observation) => ({
    observation,
    direction: directionOfObs(observation),
    net: netOfObs(observation),
  }));
  const trades = scored.filter((entry) => entry.direction !== "flat");
  const grossProfit = trades.reduce(
    (sum, entry) => sum + Math.max(0, entry.net),
    0,
  );
  const grossLoss = trades.reduce(
    (sum, entry) => sum + Math.max(0, -entry.net),
    0,
  );
  const winners = trades.filter((entry) => entry.net > 0).length;
  const directionHits = trades.filter(
    (entry) =>
      entry.direction === directionOf(entry.observation.actualReturnPct),
  ).length;
  return {
    trades: trades.length,
    coveragePct:
      observations.length === 0
        ? 0
        : (trades.length / observations.length) * 100,
    netReturnPct: trades.reduce((sum, entry) => sum + entry.net, 0),
    profitFactor:
      grossLoss === 0 ? (grossProfit > 0 ? null : 0) : grossProfit / grossLoss,
    winRatePct: trades.length === 0 ? 0 : (winners / trades.length) * 100,
    directionAccuracyPct:
      trades.length === 0 ? 0 : (directionHits / trades.length) * 100,
  };
}

interface RawCandle {
  readonly open_price: number;
  readonly high_price: number;
  readonly low_price: number;
  readonly close_price: number;
  readonly volume: number;
  readonly timestamp: string;
}

interface GridCandle {
  readonly exchange: string;
  readonly symbol: string;
  readonly timeframe: string;
  readonly open: number;
  readonly high: number;
  readonly low: number;
  readonly close: number;
  readonly volume: number;
  readonly timestamp: Date;
}

function parseTimestamp(value: string): Date {
  const normalized =
    value.endsWith("Z") || value.includes("+")
      ? value
      : `${value.replace(" ", "T")}Z`;
  const timestamp = new Date(normalized);
  if (!Number.isFinite(timestamp.getTime()))
    throw new Error(`bad timestamp: ${value}`);
  return timestamp;
}

function loadCandles(
  homeDir: string,
  symbol: string,
  timeframe: string,
  endIso: string,
): GridCandle[] {
  const db = new Database(join(homeDir, "data", "neuratrade.db"), {
    readonly: true,
  });
  try {
    const rows = db
      .query(
        `SELECT o.open_price, o.high_price, o.low_price, o.close_price, o.volume, o.timestamp
         FROM ohlcv_data o
         JOIN exchanges e ON e.id = o.exchange_id
         JOIN trading_pairs tp ON tp.id = o.trading_pair_id
         WHERE e.name = ? AND tp.symbol = ? AND o.timeframe = ? AND o.timestamp <= ?
         ORDER BY julianday(o.timestamp) ASC`,
      )
      .all("bybit-futures", symbol, timeframe, endIso) as RawCandle[];
    return rows.map((row) => ({
      exchange: "bybit-futures",
      symbol,
      timeframe,
      open: row.open_price,
      high: row.high_price,
      low: row.low_price,
      close: row.close_price,
      volume: row.volume,
      timestamp: parseTimestamp(row.timestamp),
    }));
  } finally {
    db.close();
  }
}

const SYMBOLS = [
  { key: "btc", symbol: "BTC/USDT:USDT" },
  { key: "sol", symbol: "SOL/USDT:USDT" },
  { key: "eth", symbol: "ETH/USDT:USDT" },
] as const;

const SEEDS = [7, 19, 42, 101, 20260802];
const LOOSE_THRESHOLD = 0;
const LADDER = [0.16, 0.1, 0.05, 0];
const POINT_DIAGNOSTIC = 0.25;

async function main(): Promise<void> {
  const homeDir =
    process.env.NEURATRADE_HOME ?? join(process.env.HOME ?? ".", ".neuratrade");
  const out: LooseReport = {
    ok: true,
    researchOnly: true,
    looseRule:
      "directionalBand@0.00 (long if q10>0, short if q90<0, else flat)",
    friction: "fee 0.06pc + slippage 2bps (walkforward frictionCostPct 0.16)",
    symbols: {},
  };
  for (const { key, symbol } of SYMBOLS) {
    const saved = await Bun.file(`/tmp/timesfm-wf-${key}-full.json`).json();
    if (saved.researchOnly !== true)
      throw new Error(`${key}: not researchOnly`);
    const observations = saved.report.observations as LooseObservation[];
    const frictionCostPct = saved.report.frictionCostPct as number;
    if (observations.length !== 1191)
      throw new Error(`${key}: expected 1191 origins`);
    if (Math.abs(frictionCostPct - 0.16) > 1e-9)
      throw new Error(`${key}: friction changed`);

    const baseline = scoreDirections(
      observations,
      (o) => o.baselineDirection,
      (o) => o.baselineNetReturnPct,
      () => 0,
    );
    const ladder: Record<string, LooseMetrics> = {};
    for (const threshold of LADDER) {
      ladder[`qband@${threshold.toFixed(2)}`] = scoreDirections(
        observations,
        (o) => looseDirection(o.q10ReturnPct, o.q90ReturnPct, threshold),
        (o) =>
          signedNet(
            looseDirection(o.q10ReturnPct, o.q90ReturnPct, threshold),
            o.actualReturnPct,
            frictionCostPct,
          ),
        (o) => o.pointForecastReturnPct,
      );
    }
    const loose = ladder["qband@0.00"]!;
    const pointDiag = scoreDirections(
      observations,
      (o) =>
        Math.abs(o.pointForecastReturnPct) >= POINT_DIAGNOSTIC
          ? directionOf(o.pointForecastReturnPct)
          : "flat",
      (o) =>
        Math.abs(o.pointForecastReturnPct) >= POINT_DIAGNOSTIC
          ? signedNet(
              directionOf(o.pointForecastReturnPct),
              o.actualReturnPct,
              frictionCostPct,
            )
          : 0,
      (o) => o.pointForecastReturnPct,
    );

    // Rolling honesty analogue: 10 contiguous origin blocks over the frozen tail.
    const blockSize = Math.floor(observations.length / 10);
    const blocks = Array.from({ length: 10 }, (_, block) => {
      const slice = observations.slice(
        block * blockSize,
        (block + 1) * blockSize,
      );
      const m = scoreDirections(
        slice,
        (o) => looseDirection(o.q10ReturnPct, o.q90ReturnPct, LOOSE_THRESHOLD),
        (o) =>
          signedNet(
            looseDirection(o.q10ReturnPct, o.q90ReturnPct, LOOSE_THRESHOLD),
            o.actualReturnPct,
            frictionCostPct,
          ),
        (o) => o.pointForecastReturnPct,
      );
      return {
        block,
        origins: slice.length,
        trades: m.trades,
        netReturnPct: m.netReturnPct,
      };
    });

    // Grid leg on the identical frozen tail (same bars the walk-forward used).
    const candleCount = saved.candleCount as number;
    const dataEnd = saved.dataEnd as string;
    const candles = loadCandles(homeDir, symbol, "15m", dataEnd);
    if (candles.length !== candleCount) {
      throw new Error(
        `${key}: panel changed (${candles.length} vs ${candleCount})`,
      );
    }
    const oosStart = Math.floor(candles.length * 0.8);
    const tail = candles.slice(oosStart);
    const sorted = [...observations].sort(
      (a, b) => a.originIndex - b.originIndex,
    );
    const overlay: TradeDirection[] = Array(tail.length).fill("flat");
    for (let i = 0; i < sorted.length; i += 1) {
      const obs = sorted[i]!;
      const rel = obs.originIndex - oosStart;
      const nextRel =
        i + 1 < sorted.length
          ? sorted[i + 1]!.originIndex - oosStart
          : tail.length;
      const direction = looseDirection(
        obs.q10ReturnPct,
        obs.q90ReturnPct,
        LOOSE_THRESHOLD,
      );
      for (let bar = rel + 1; bar <= nextRel && bar < tail.length; bar += 1) {
        if (bar >= 0) overlay[bar] = direction;
      }
    }
    const candidate =
      candidateForSymbol(symbol) ?? VALIDATED_BTC_GRID_CANDIDATE;
    const baseOpts: GridOptions = {
      gridStepPct: candidate.gridStepPct,
      gridMaxGrids: candidate.gridMaxGrids,
      gridPauseAfterLossBars: candidate.gridPauseAfterLossBars,
      feePct: 0.06,
      slippageBps: 2,
      initialCapital: 10000,
      trendFilterPeriod: candidate.trendFilterPeriod,
      leverage: candidate.leverage,
      onlyWithTrend: candidate.onlyWithTrend,
      targetRatio: candidate.targetRatio,
      chopGateAdxThreshold: candidate.chopGateAdx,
    };
    const toPolicyResult = (
      name: string,
      result: {
        readonly totalReturnPct: number;
        readonly maxDrawdownPct: number;
        readonly totalTrades: number;
        readonly winRate: number;
        readonly profitFactor: number | null;
      },
    ): LooseGridPolicyResult => ({
      policy: name,
      totalReturnPct: result.totalReturnPct,
      maxDrawdownPct: result.maxDrawdownPct,
      totalTrades: result.totalTrades,
      winRatePct: result.winRate,
      profitFactor: result.profitFactor,
    });
    const run = (
      name: string,
      entryOverlay: readonly TradeDirection[] | undefined,
    ): LooseGridPolicyResult => {
      const options: GridOptions =
        entryOverlay === undefined
          ? baseOpts
          : { ...baseOpts, entryDirectionByBar: entryOverlay };
      return toPolicyResult(name, runGridBacktest(tail, options));
    };
    const gridCleanWinner = run("directionalBand@0.00", overlay);
    const gridCleanBaseline = run("baseline", undefined);
    const stressSeeds = SEEDS.map((seed) => {
      const stress = {
        makerFillProb: 0.7,
        adverseSelection: true,
        takerExitFeePct: 0.06,
        fillSeed: seed,
      };
      const winner = runGridBacktest(tail, {
        ...baseOpts,
        ...stress,
        entryDirectionByBar: overlay,
      });
      const base = runGridBacktest(tail, { ...baseOpts, ...stress });
      return {
        seed,
        winner: {
          policy: "directionalBand@0.00",
          totalReturnPct: winner.totalReturnPct,
          maxDrawdownPct: winner.maxDrawdownPct,
          totalTrades: winner.totalTrades,
          winRatePct: winner.winRate,
          profitFactor: winner.profitFactor,
        },
        baseline: {
          policy: "baseline",
          totalReturnPct: base.totalReturnPct,
          maxDrawdownPct: base.maxDrawdownPct,
          totalTrades: base.totalTrades,
          winRatePct: base.winRate,
          profitFactor: base.profitFactor,
        },
      };
    });
    const positiveSeeds = stressSeeds.filter(
      (s) => s.winner.totalReturnPct > 0,
    ).length;

    out.symbols[key] = {
      symbol,
      candleCount,
      oosStart,
      tailBars: tail.length,
      originCount: observations.length,
      frictionCostPct,
      baseline,
      ladder,
      loose,
      pointDiagnostic: pointDiag,
      blocks,
      profitableBlocksPct:
        (blocks.filter((b) => b.netReturnPct > 0).length / blocks.length) * 100,
      grid: {
        cleanWinner: gridCleanWinner,
        cleanBaseline: gridCleanBaseline,
        seeds: stressSeeds,
        positiveSeeds,
        gate4pass: positiveSeeds >= 3,
      },
    };
    console.log(
      `${key.toUpperCase()}: loose trades=${loose.trades} net=${loose.netReturnPct.toFixed(2)}% ` +
        `pf=${loose.profitFactor} da=${loose.directionAccuracyPct.toFixed(1)}% | ` +
        `grid trades=${gridCleanWinner.totalTrades} ret=${gridCleanWinner.totalReturnPct.toFixed(2)}% | ` +
        `stress ${positiveSeeds}/5 positive`,
    );
  }
  await Bun.write("/tmp/timesfm-oos-loose.json", JSON.stringify(out, null, 1));
  console.log("wrote /tmp/timesfm-oos-loose.json");
}

if (import.meta.main) {
  await main();
}
