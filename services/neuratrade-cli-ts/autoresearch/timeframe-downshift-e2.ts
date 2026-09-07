/**
 * Growth E2 timeframe downshift (clever-cabin-umi) — ADDITIVE-ONLY probe.
 *
 * Evaluates champion-style ladder-grid knobs on 15m (baseline), 5m and 1m
 * Bybit-futures panels with HONEST taker-exit fees + slippage, venue-filtered
 * loads, and a frozen trailing 30d holdout reported SEPARATELY per timeframe.
 *
 * ADDITIVE ONLY: this file creates no live workers, writes no DBs, never
 * touches champion files (champion-soak.json / whitelist / knobs.ts) and never
 * edits CLAIM bars in goals.ts. It only READS the frozen champion copy below
 * and READS the venue DB in readonly mode. Results go to the NEW file
 * autoresearch/results/timeframe-downshift-e2.json (+ .md table).
 *
 * Venue rule (mirrors prepare.ts): exact symbol match within ONE exchange
 * (bybit-futures); aliases and venues are never merged. 15m baseline panel is
 * resampled from the 5m source (same as prepare.ts); 5m panel is native (no
 * resample); 1m panel attempts a venue-filtered 1m load and reports
 * insufficient-data honestly when the venue has no 1m candles.
 *
 * Fees (mirror prepare.ts honest schedule): maker 0.02%% per side on entries
 * and target exits, taker 0.06%% per side on stop/liquidation/max-hold exits,
 * 2 bps slippage.
 */

import { Database } from "bun:sqlite";
import { resampleCandles } from "../src/scalping/grid-universe.ts";
import { runLadderGridBacktest } from "../src/scalping/ladder-grid.ts";
import type { Candle } from "../src/market-data/types.ts";
import { checkKeepGuards } from "./goals.ts";
import type { AutoresearchKnobs } from "./knobs.ts";

/** Live venue — the only exchange whose candles may enter any panel. */
export const E2_EXCHANGE = "bybit-futures" as const;

/** Honest cost schedule — must match prepare.ts. */
export const E2_FEE_PCT = 0.02;
export const E2_TAKER_EXIT_FEE_PCT = 0.06;
export const E2_SLIPPAGE_BPS = 2;

/**
 * Frozen champion-style copy (champion-soak.json, 2026-09-06, honest-fee
 * rescore 2026-09-07). A COPY so this probe can never mutate the champion.
 * Read-only: compare with `bun run` notice if the champion moves on.
 */
export const E2_CHAMPION_KNOBS: AutoresearchKnobs = {
  rungs: 2,
  gridStepPct: 1.3,
  gridMaxGrids: 2,
  gridPauseAfterLossBars: 2,
  stopRatio: 1.58,
  targetRatio: 1.95,
  maxHoldBars: 39,
  trendFilterPeriod: 0,
  chopGateAdxThreshold: 0,
  positionFraction: 1,
};

export interface E2TimeframeSpec {
  readonly panelTimeframe: string;
  /** Venue source timeframe read from the DB. */
  readonly sourceTimeframe: string;
  readonly tfMinutes: number;
  /** Frozen trailing 30d tail, in panel bars. */
  readonly holdoutBars: number;
  /** Confirm-like selection geometry, in panel bars (1d step, 30d fwd). */
  readonly stepBars: number;
  readonly forwardBars: number;
  readonly firstBar: number;
  readonly minSymbols: number;
  readonly minWindows: number;
}

/** 30d of panel bars for a timeframe. */
export function holdoutBarsForTf(tfMinutes: number): number {
  return Math.round((30 * 24 * 60) / tfMinutes);
}

export function geometryForTf(
  panelTimeframe: string,
  sourceTimeframe: string,
): E2TimeframeSpec {
  const tfMinutes =
    panelTimeframe === "15m"
      ? 15
      : panelTimeframe === "5m"
        ? 5
        : panelTimeframe === "1m"
          ? 1
          : NaN;
  if (!Number.isFinite(tfMinutes) || tfMinutes <= 0)
    throw new Error(`unsupported panelTimeframe ${panelTimeframe}`);
  const perDay = (24 * 60) / (tfMinutes as number);
  return {
    panelTimeframe,
    sourceTimeframe,
    tfMinutes: tfMinutes as number,
    holdoutBars: holdoutBarsForTf(tfMinutes as number),
    stepBars: Math.round(perDay), // 1d step
    forwardBars: Math.round(perDay * 30), // 30d forward
    firstBar: Math.round(perDay * 7), // 7d warmup
    minSymbols: 3,
    minWindows: 3,
  };
}

export const E2_SPECS: readonly E2TimeframeSpec[] = [
  geometryForTf("15m", "5m"), // baseline: 5m source resampled to 15m (== prepare.ts)
  geometryForTf("5m", "5m"), // native 5m, no resample
  geometryForTf("1m", "1m"), // venue-filtered 1m load (expected: no venue candles)
];

export interface E2Panel {
  readonly symbols: readonly string[];
  readonly aligned: ReadonlyMap<string, Candle[]>;
  readonly refLen: number;
  readonly exchange: string;
  readonly sourceTimeframe: string;
  readonly panelTimeframe: string;
  readonly holdoutBars: number;
  readonly reason: string;
}

export function median(xs: readonly number[]): number {
  const s = xs.filter(Number.isFinite).sort((a, b) => a - b);
  if (!s.length) return Number.NaN;
  const m = Math.floor(s.length / 2);
  return s.length % 2 ? s[m]! : (s[m - 1]! + s[m]!) / 2;
}

function homeDb(explicit?: string): string {
  if (explicit) return explicit;
  const home = process.env.NEURATRADE_HOME ?? `${process.env.HOME}/.neuratrade`;
  return `${home}/data/neuratrade.db`;
}

function tableExists(db: Database, name: string): boolean {
  try {
    const row = db
      .query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`)
      .get(name) as { name: string } | null;
    return !!row;
  } catch {
    return false;
  }
}

interface RawRow {
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  timestamp: string;
}

/**
 * Venue-filtered load: exact symbol match within ONE exchange + source
 * timeframe. Never merges aliases (BTC/USDT vs BTC/USDT:USDT) or venues.
 * DB is opened READONLY. Resamples up to the panel timeframe when the source
 * is finer (5m -> 15m); native when source == panel.
 */
export function loadE2Panel(opts: {
  readonly spec: E2TimeframeSpec;
  readonly symbols?: number;
  readonly dbPath?: string;
  readonly minCandles?: number;
}): E2Panel {
  const { spec } = opts;
  const topN = opts.symbols ?? 8;
  const minCandles = opts.minCandles ?? Math.min(spec.forwardBars, 6000);
  const db = new Database(homeDb(opts.dbPath), { readonly: true });
  try {
    db.exec("PRAGMA busy_timeout = 30000;");
    const hasExchanges = tableExists(db, "exchanges");
    const select = `SELECT c.open_price AS open, c.high_price AS high, c.low_price AS low,
              c.close_price AS close, c.volume, c.timestamp
       FROM ohlcv_data c JOIN trading_pairs tp ON tp.id = c.trading_pair_id`;
    const symRows = (
      hasExchanges
        ? db
            .query(
              `${select} JOIN exchanges e ON e.id = c.exchange_id
       WHERE e.name = ? AND c.timeframe = ?
       GROUP BY tp.id, tp.symbol ORDER BY COUNT(*) DESC LIMIT ?`,
            )
            .all(E2_EXCHANGE, spec.sourceTimeframe, topN + 4)
        : db
            .query(
              `${select}
       WHERE c.timeframe = ?
       GROUP BY tp.symbol ORDER BY COUNT(*) DESC LIMIT ?`,
            )
            .all(spec.sourceTimeframe, topN + 4)
    ) as Array<{ symbol: string; count?: number }>;
    // The GROUP-BY projection above drops tp.symbol; re-query symbol list
    // with an explicit symbol column for clarity and safety.
    const listed = (
      hasExchanges
        ? db
            .query(
              `SELECT tp.symbol AS symbol, COUNT(*) AS count
       FROM ohlcv_data c JOIN trading_pairs tp ON tp.id = c.trading_pair_id
       JOIN exchanges e ON e.id = c.exchange_id
       WHERE e.name = ? AND c.timeframe = ?
       GROUP BY tp.id, tp.symbol ORDER BY count DESC LIMIT ?`,
            )
            .all(E2_EXCHANGE, spec.sourceTimeframe, topN + 4)
        : db
            .query(
              `SELECT tp.symbol AS symbol, COUNT(*) AS count
       FROM ohlcv_data c JOIN trading_pairs tp ON tp.id = c.trading_pair_id
       WHERE c.timeframe = ?
       GROUP BY tp.symbol ORDER BY count DESC LIMIT ?`,
            )
            .all(spec.sourceTimeframe, topN + 4)
    ) as Array<{ symbol: string; count: number }>;
    void symRows;
    const aligned = new Map<string, Candle[]>();
    for (const row of listed.slice(0, topN)) {
      const rowsDb = (
        hasExchanges
          ? db
              .query(
                `${select} JOIN exchanges e ON e.id = c.exchange_id
       WHERE tp.symbol = ? AND e.name = ? AND c.timeframe = ?
       ORDER BY c.timestamp DESC LIMIT ?`,
              )
              .all(row.symbol, E2_EXCHANGE, spec.sourceTimeframe, 200_000)
          : db
              .query(
                `${select}
       WHERE tp.symbol = ? AND c.timeframe = ?
       ORDER BY c.timestamp DESC LIMIT ?`,
              )
              .all(row.symbol, spec.sourceTimeframe, 200_000)
      ) as RawRow[];
      const base: Candle[] = rowsDb.toReversed().map((r) => ({
        exchange: E2_EXCHANGE,
        symbol: row.symbol,
        timeframe: spec.sourceTimeframe,
        open: r.open,
        high: r.high,
        low: r.low,
        close: r.close,
        volume: r.volume,
        timestamp: new Date(r.timestamp),
      }));
      const panel =
        spec.sourceTimeframe === spec.panelTimeframe
          ? base
          : resampleCandles(base, spec.tfMinutes, spec.panelTimeframe);
      if (panel.length >= minCandles) aligned.set(row.symbol, panel);
    }
    // Clip to the common time range.
    let t0 = 0;
    let t1 = Number.POSITIVE_INFINITY;
    for (const cs of aligned.values()) {
      t0 = Math.max(t0, cs[0]!.timestamp.getTime());
      t1 = Math.min(t1, cs[cs.length - 1]!.timestamp.getTime());
    }
    const clipped = new Map<string, Candle[]>();
    for (const [s, cs] of aligned) {
      const keep = cs.filter(
        (c) =>
          c.timestamp.getTime() >= t0 &&
          c.timestamp.getTime() <= t1 &&
          Number.isFinite(c.close) &&
          c.close > 0,
      );
      if (keep.length >= minCandles) clipped.set(s, keep);
    }
    const symbols = [...clipped.keys()].sort();
    const refLen =
      symbols.length > 0
        ? Math.min(...symbols.map((s) => clipped.get(s)!.length))
        : 0;
    if (
      symbols.length < spec.minSymbols ||
      refLen <= spec.holdoutBars + spec.forwardBars
    ) {
      return {
        symbols,
        aligned: clipped,
        refLen,
        exchange: E2_EXCHANGE,
        sourceTimeframe: spec.sourceTimeframe,
        panelTimeframe: spec.panelTimeframe,
        holdoutBars: spec.holdoutBars,
        reason:
          symbols.length === 0
            ? `insufficient_data_no_${spec.sourceTimeframe}_venue_candles`
            : "insufficient_bars",
      };
    }
    return {
      symbols,
      aligned: clipped,
      refLen,
      exchange: E2_EXCHANGE,
      sourceTimeframe: spec.sourceTimeframe,
      panelTimeframe: spec.panelTimeframe,
      holdoutBars: spec.holdoutBars,
      reason: "ok",
    };
  } finally {
    db.close();
  }
}

export interface E2SliceResult {
  readonly score: number;
  readonly guardsOk: boolean;
  readonly medianLogReturn: number;
  readonly medianReturnPct: number;
  readonly medianDrawdownPct: number;
  readonly winRatePct: number;
  readonly tradesPerSymMonth: number;
  readonly expectancyPct: number;
  readonly windows: number;
  readonly symbols: number;
  readonly reason: string;
}

function baseOpts(knobs: AutoresearchKnobs) {
  return {
    rungs: knobs.rungs,
    gridStepPct: knobs.gridStepPct,
    gridMaxGrids: knobs.gridMaxGrids,
    gridPauseAfterLossBars: knobs.gridPauseAfterLossBars,
    feePct: E2_FEE_PCT,
    takerExitFeePct: E2_TAKER_EXIT_FEE_PCT,
    slippageBps: E2_SLIPPAGE_BPS,
    initialCapital: 10_000,
    leverage: 1,
    trendFilterPeriod: knobs.trendFilterPeriod,
    stopRatio: knobs.stopRatio,
    targetRatio: knobs.targetRatio,
    maxHoldBars: knobs.maxHoldBars,
    chopGateAdxThreshold: knobs.chopGateAdxThreshold,
    positionFraction: knobs.positionFraction,
    conservativeIntrabar: true,
  };
}

function summarizeSlices(
  rets: number[],
  dds: number[],
  trades: number,
  wins: number,
  pnlSum: number,
  spec: E2TimeframeSpec,
  symbols: number,
  reason: string,
): E2SliceResult {
  const windows = rets.length;
  if (windows < spec.minWindows) {
    return {
      score: Number.NEGATIVE_INFINITY,
      guardsOk: false,
      medianLogReturn: Number.NaN,
      medianReturnPct: Number.NaN,
      medianDrawdownPct: Number.NaN,
      winRatePct: Number.NaN,
      tradesPerSymMonth: Number.NaN,
      expectancyPct: Number.NaN,
      windows,
      symbols,
      reason,
    };
  }
  const windowMonths = (spec.forwardBars * spec.tfMinutes) / (60 * 24 * 30);
  const tradesPerSymMonth =
    windowMonths > 0 ? trades / windows / windowMonths : Number.NaN;
  const winRatePct = trades > 0 ? (wins / trades) * 100 : Number.NaN;
  const expectancyPct = trades > 0 ? pnlSum / trades : Number.NaN;
  const medianReturnPct = median(rets);
  const medianDrawdownPct = median(dds);
  const medianLogReturn = median(
    rets
      .filter(Number.isFinite)
      .map((r) => Math.log(1 + Math.max(-0.95, r / 100))),
  );
  const g = checkKeepGuards({
    medianLogReturn,
    winRatePct,
    medianDrawdownPct,
    tradesPerSymMonth,
    expectancyPct,
  });
  return {
    score: Number.isFinite(medianLogReturn)
      ? medianLogReturn
      : Number.NEGATIVE_INFINITY,
    guardsOk: g.ok,
    medianLogReturn,
    medianReturnPct,
    medianDrawdownPct,
    winRatePct,
    tradesPerSymMonth,
    expectancyPct,
    windows,
    symbols,
    reason: g.reason,
  };
}

/** Confirm-like selection: walk-forward over the prefix BEFORE the frozen tail. */
export function evaluateE2Selection(
  knobs: AutoresearchKnobs,
  panel: E2Panel,
  spec: E2TimeframeSpec,
  opts: { readonly maxSteps?: number } = {},
): E2SliceResult {
  const maxSteps = opts.maxSteps ?? 10;
  const selectionLen = Math.max(0, panel.refLen - spec.holdoutBars);
  const lastStartBar = selectionLen - spec.forwardBars - 1;
  if (panel.symbols.length < spec.minSymbols || lastStartBar <= spec.firstBar) {
    return {
      score: Number.NEGATIVE_INFINITY,
      guardsOk: false,
      medianLogReturn: Number.NaN,
      medianReturnPct: Number.NaN,
      medianDrawdownPct: Number.NaN,
      winRatePct: Number.NaN,
      tradesPerSymMonth: Number.NaN,
      expectancyPct: Number.NaN,
      windows: 0,
      symbols: panel.symbols.length,
      reason: panel.reason !== "ok" ? panel.reason : "insufficient_bars",
    };
  }
  const bo = baseOpts(knobs);
  const rets: number[] = [];
  const dds: number[] = [];
  let trades = 0;
  let wins = 0;
  let pnlSum = 0;
  let steps = 0;
  for (
    let bar = spec.firstBar;
    bar <= lastStartBar && steps < maxSteps;
    bar += spec.stepBars, steps++
  ) {
    for (const symbol of panel.symbols) {
      const cs = panel.aligned.get(symbol)!;
      const startIdx = cs.length - panel.refLen + bar;
      const endIdx = Math.min(
        cs.length - spec.holdoutBars,
        startIdx + spec.forwardBars,
      );
      if (endIdx - startIdx < spec.forwardBars * 0.9) continue;
      try {
        const r = runLadderGridBacktest(cs.slice(startIdx, endIdx), bo);
        trades += r.trades.length;
        wins += r.trades.filter((t) => t.win).length;
        pnlSum += r.trades.reduce((s, t) => s + (t.pnlPct ?? 0), 0);
        rets.push(r.totalReturnPct);
        dds.push(r.maxDrawdownPct);
      } catch {
        /* skip transient window failure */
      }
    }
  }
  return summarizeSlices(
    rets,
    dds,
    trades,
    wins,
    pnlSum,
    spec,
    panel.symbols.length,
    "ok",
  );
}

/** Frozen holdout: ONE trailing 30d window per symbol (never used for selection). */
export function evaluateE2Holdout(
  knobs: AutoresearchKnobs,
  panel: E2Panel,
  spec: E2TimeframeSpec,
): E2SliceResult {
  if (panel.symbols.length < spec.minSymbols) {
    return {
      score: Number.NEGATIVE_INFINITY,
      guardsOk: false,
      medianLogReturn: Number.NaN,
      medianReturnPct: Number.NaN,
      medianDrawdownPct: Number.NaN,
      winRatePct: Number.NaN,
      tradesPerSymMonth: Number.NaN,
      expectancyPct: Number.NaN,
      windows: 0,
      symbols: panel.symbols.length,
      reason: panel.reason !== "ok" ? panel.reason : "insufficient_symbols",
    };
  }
  const bo = baseOpts(knobs);
  const rets: number[] = [];
  const dds: number[] = [];
  let trades = 0;
  let wins = 0;
  let pnlSum = 0;
  for (const symbol of panel.symbols) {
    const cs = panel.aligned.get(symbol)!;
    if (cs.length < spec.holdoutBars * 0.9) continue;
    try {
      const r = runLadderGridBacktest(
        cs.slice(cs.length - spec.holdoutBars),
        bo,
      );
      trades += r.trades.length;
      wins += r.trades.filter((t) => t.win).length;
      pnlSum += r.trades.reduce((s, t) => s + (t.pnlPct ?? 0), 0);
      rets.push(r.totalReturnPct);
      dds.push(r.maxDrawdownPct);
    } catch {
      /* skip transient window failure */
    }
  }
  return summarizeSlices(
    rets,
    dds,
    trades,
    wins,
    pnlSum,
    spec,
    panel.symbols.length,
    "ok",
  );
}

export interface E2Row {
  readonly panelTimeframe: string;
  readonly sourceTimeframe: string;
  readonly exchange: string;
  readonly symbols: readonly string[];
  readonly refLen: number;
  readonly holdoutBars: number;
  readonly selection: E2SliceResult;
  readonly holdout: E2SliceResult;
  readonly loadReason: string;
}

export function runE2Downshift(opts: {
  readonly symbols?: number;
  readonly maxSteps?: number;
  readonly dbPath?: string;
}): E2Row[] {
  return E2_SPECS.map((spec) => {
    const panel = loadE2Panel({
      spec,
      symbols: opts.symbols,
      dbPath: opts.dbPath,
    });
    if (panel.reason !== "ok") {
      const empty: E2SliceResult = {
        score: Number.NEGATIVE_INFINITY,
        guardsOk: false,
        medianLogReturn: Number.NaN,
        medianReturnPct: Number.NaN,
        medianDrawdownPct: Number.NaN,
        winRatePct: Number.NaN,
        tradesPerSymMonth: Number.NaN,
        expectancyPct: Number.NaN,
        windows: 0,
        symbols: panel.symbols.length,
        reason: panel.reason,
      };
      return {
        panelTimeframe: spec.panelTimeframe,
        sourceTimeframe: spec.sourceTimeframe,
        exchange: E2_EXCHANGE,
        symbols: panel.symbols,
        refLen: panel.refLen,
        holdoutBars: spec.holdoutBars,
        selection: empty,
        holdout: empty,
        loadReason: panel.reason,
      };
    }
    return {
      panelTimeframe: spec.panelTimeframe,
      sourceTimeframe: spec.sourceTimeframe,
      exchange: E2_EXCHANGE,
      symbols: panel.symbols,
      refLen: panel.refLen,
      holdoutBars: spec.holdoutBars,
      selection: evaluateE2Selection(E2_CHAMPION_KNOBS, panel, spec, {
        maxSteps: opts.maxSteps,
      }),
      holdout: evaluateE2Holdout(E2_CHAMPION_KNOBS, panel, spec),
      loadReason: "ok",
    };
  });
}

function fmt(x: number, digits = 4): string {
  return Number.isFinite(x) ? x.toFixed(digits) : "n/a";
}

if (import.meta.main) {
  const t0 = Date.now();
  const rows = runE2Downshift({ symbols: 8, maxSteps: 10 });
  console.log(
    "timeframe | src | syms | refLen | slice | score(medLogRet) | expectancy% | tpsm | win% | dd% | windows | reason",
  );
  for (const r of rows) {
    for (const [name, s] of [
      ["select", r.selection],
      ["holdout", r.holdout],
    ] as const) {
      console.log(
        `${r.panelTimeframe} | ${r.sourceTimeframe} | ${r.symbols.length} | ${r.refLen} | ${name} | ${fmt(s.score)} | ${fmt(s.expectancyPct)} | ${fmt(s.tradesPerSymMonth, 2)} | ${fmt(s.winRatePct, 1)} | ${fmt(s.medianDrawdownPct, 2)} | ${s.windows} | ${r.loadReason}/${s.reason}`,
      );
    }
  }
  console.log(`elapsedMs=${Date.now() - t0}`);
}
