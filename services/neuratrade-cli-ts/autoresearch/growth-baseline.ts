#!/usr/bin/env bun
/**
 * Growth E0 baseline (clever-cabin-pe9) — ADDITIVE-ONLY, READ-ONLY analysis.
 *
 * What it does: Monte Carlos $1k -> $1M using the FROZEN champion stats
 * (expectancy 0.0009808/trade, median maxDD 8.53%) at an estimated 3-7
 * trades/day, with live risk rails (15% maxDD kill-switch, 5% daily-loss
 * halt, leverage 1) and taker-exit fees already netted into the edge.
 * Reports median time-to-$1M, ruin rate, and sensitivity to trades/day.
 *
 * What it never does (additive-only contract):
 * - imports no live engine, worker, or DB code (pure math + seeded RNG);
 * - never writes knobs.ts, goals.ts, champion files, results/, or any DB;
 * - never promotes, keeps, or claims — it only REPORTS Monte Carlo numbers.
 *
 * Model (trade-level geometric compounding, leverage 1):
 * - per-trade log-return r ~ Normal(muLog, sigma), equity *= exp(r);
 * - muLog = ln(1 + expectancyPct/100). expectancyPct is in the native
 *   prepare.ts units (percent points of equity per trade, pnlPct mean),
 *   already NET of honest venue fees (maker 0.02%/side, taker 0.06% on
 *   stop/max-hold exits, 2bps slippage) per champion-soak.json
 *   ("honestFees ... rescored 2026-09-07"). No further fee is deducted.
 * - sigma is NOT a free parameter: it is calibrated by bisection so the
 *   simulated median window maxDD over CALIBRATION_TRADES (=100, about one
 *   symbol-month at ~5 trades/day x 20d) reproduces the frozen 8.53%.
 * - 15% maxDD kill-switch is TERMINAL for the path (counts as ruin);
 *   5% daily-loss halt skips the rest of that day's trades (not terminal).
 *
 * Run: bun run autoresearch/growth-baseline.ts [--paths N] [--years N] [--seed N]
 * Test: bun test autoresearch/growth-baseline.test.ts
 */

/** Frozen champion stats — the E0 input spec. Do not retune here. */
export const FROZEN_EXPECTANCY_PCT_PER_TRADE = 0.0009808;
export const FROZEN_MEDIAN_MAXDD_PCT = 8.53;
/** Estimated live throughput range (trades/day across the traded universe). */
export const TRADES_PER_DAY_RANGE = [3, 4, 5, 6, 7] as const;
/** Live risk rails. */
export const MAX_DD_KILL_PCT = 15;
export const DAILY_LOSS_HALT_PCT = 5;
export const LEVERAGE = 1;
/** Growth question. */
export const START_EQUITY = 1_000;
export const TARGET_EQUITY = 1_000_000;
/** Trades per reference window used for the sigma calibration (~1 sym-month). */
export const CALIBRATION_TRADES = 100;
/** Monte Carlo defaults (overridable via CLI flags). */
export const DEFAULT_PATHS = 2000;
export const DEFAULT_MAX_YEARS = 30;
export const DEFAULT_SEED = 20260907;

// ---------------------------------------------------------------------------
// Seeded RNG (mulberry32) + Box-Muller gaussian. Deterministic per seed.
// ---------------------------------------------------------------------------

export interface RngState {
  s: number;
}

export function createRng(seed: number): RngState {
  return { s: seed >>> 0 };
}

function nextUnit(r: RngState): number {
  r.s = (r.s + 0x6d2b79f5) >>> 0;
  let t = r.s;
  t = Math.imul(t ^ (t >>> 15), t | 1);
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
}

/** One standard-normal draw (Box-Muller, single-variate form). */
export function randn(r: RngState): number {
  let u1 = 0;
  while (u1 === 0) u1 = nextUnit(r);
  const u2 = nextUnit(r);
  return Math.sqrt(-2 * Math.log(u1)) * Math.cos(2 * Math.PI * u2);
}

// ---------------------------------------------------------------------------
// Small stats helpers.
// ---------------------------------------------------------------------------

export function median(xs: readonly number[]): number {
  const s = xs.filter(Number.isFinite).sort((a, b) => a - b);
  if (s.length === 0) return Number.NaN;
  const m = Math.floor(s.length / 2);
  return s.length % 2 === 1 ? s[m]! : (s[m - 1]! + s[m]!) / 2;
}

/** Peak-to-trough max drawdown in percent for one W-trade log-walk. */
export function simulateMaxDrawdownPct(
  sigma: number,
  muLog: number,
  windowTrades: number,
  rng: RngState,
): number {
  let logEq = 0;
  let peak = 0;
  let maxDdLog = 0;
  for (let i = 0; i < windowTrades; i++) {
    logEq += muLog + sigma * randn(rng);
    if (logEq > peak) peak = logEq;
    const dd = peak - logEq;
    if (dd > maxDdLog) maxDdLog = dd;
  }
  return (1 - Math.exp(-maxDdLog)) * 100;
}

export function medianMaxDrawdownPct(
  sigma: number,
  muLog: number,
  windowTrades: number,
  paths: number,
  seed: number,
): number {
  const rng = createRng(seed);
  const dds: number[] = new Array(paths);
  for (let p = 0; p < paths; p++) {
    dds[p] = simulateMaxDrawdownPct(sigma, muLog, windowTrades, rng);
  }
  return median(dds);
}

/**
 * Bisect the per-trade volatility sigma whose simulated median window maxDD
 * reproduces targetDdPct. Median DD is monotone increasing in sigma, so the
 * bisection is well-posed. Throws when the target is unbracketable.
 */
export function calibrateSigma(
  targetDdPct: number,
  muLog: number,
  windowTrades: number,
  seed: number,
  paths = 4000,
  iterations = 22,
): number {
  let lo = 0.0001;
  let hi = 0.2;
  const ddLo = medianMaxDrawdownPct(lo, muLog, windowTrades, paths, seed);
  const ddHi = medianMaxDrawdownPct(hi, muLog, windowTrades, paths, seed);
  if (!(ddLo <= targetDdPct && targetDdPct <= ddHi)) {
    throw new Error(
      `calibration unbracketed: target=${targetDdPct} lo=${ddLo.toFixed(3)} hi=${ddHi.toFixed(3)}`,
    );
  }
  for (let i = 0; i < iterations; i++) {
    const mid = (lo + hi) / 2;
    const dd = medianMaxDrawdownPct(mid, muLog, windowTrades, paths, seed);
    if (dd < targetDdPct) lo = mid;
    else hi = mid;
  }
  return (lo + hi) / 2;
}

// ---------------------------------------------------------------------------
// Growth Monte Carlo: $1k -> $1M with live risk rails.
// ---------------------------------------------------------------------------

export interface GrowthSimOptions {
  readonly tradesPerDay: number;
  readonly paths: number;
  readonly maxYears: number;
  readonly seed: number;
  readonly sigma: number;
  readonly muLog: number;
  readonly startEquity?: number;
  readonly targetEquity?: number;
  readonly maxDdKillPct?: number;
  readonly dailyLossHaltPct?: number;
}

export interface GrowthSimResult {
  readonly tradesPerDay: number;
  readonly paths: number;
  readonly sigma: number;
  /** Share of paths that reached $1M within the horizon (0..1). */
  readonly successRate: number;
  /** Share of paths stopped by the 15% maxDD kill-switch (0..1). */
  readonly ruinRate: number;
  /** Share neither successful nor ruined at the horizon (0..1). */
  readonly censoredRate: number;
  /** Median calendar days to $1M among successes (null when none). */
  readonly medianDaysToTarget: number | null;
  readonly medianYearsToTarget: number | null;
  /** Median calendar days to kill-switch among ruins (null when none). */
  readonly medianRuinDays: number | null;
}

export function runGrowthMonteCarlo(opts: GrowthSimOptions): GrowthSimResult {
  const startEquity = opts.startEquity ?? START_EQUITY;
  const targetEquity = opts.targetEquity ?? TARGET_EQUITY;
  const kill = (opts.maxDdKillPct ?? MAX_DD_KILL_PCT) / 100;
  const halt = (opts.dailyLossHaltPct ?? DAILY_LOSS_HALT_PCT) / 100;
  const maxDays = Math.max(1, Math.round(opts.maxYears * 365));
  const rng = createRng(opts.seed);

  let success = 0;
  let ruin = 0;
  const daysToTarget: number[] = [];
  const ruinDays: number[] = [];

  for (let p = 0; p < opts.paths; p++) {
    let equity = startEquity;
    let peak = startEquity;
    let ruined = false;
    let doneDay = -1;
    let day = 0;
    for (; day < maxDays; day++) {
      const dayOpen = equity;
      for (let t = 0; t < opts.tradesPerDay; t++) {
        if (equity >= targetEquity) break;
        equity *= Math.exp(opts.muLog + opts.sigma * randn(rng));
        if (equity > peak) peak = equity;
        // 15% maxDD kill-switch: terminal for the path.
        if (1 - equity / peak >= kill) {
          ruined = true;
          break;
        }
        // 5% daily-loss halt: skip the rest of today's trades, resume next day.
        if (1 - equity / dayOpen >= halt) break;
      }
      if (ruined) break;
      if (equity >= targetEquity) {
        doneDay = day + 1;
        break;
      }
    }
    if (ruined) {
      ruin++;
      ruinDays.push(day + 1);
    } else if (equity >= targetEquity && doneDay > 0) {
      success++;
      daysToTarget.push(doneDay);
    }
  }

  const medDays = daysToTarget.length > 0 ? median(daysToTarget) : null;
  const medRuin = ruinDays.length > 0 ? median(ruinDays) : null;
  return {
    tradesPerDay: opts.tradesPerDay,
    paths: opts.paths,
    sigma: opts.sigma,
    successRate: success / opts.paths,
    ruinRate: ruin / opts.paths,
    censoredRate: (opts.paths - success - ruin) / opts.paths,
    medianDaysToTarget: medDays,
    medianYearsToTarget: medDays === null ? null : medDays / 365,
    medianRuinDays: medRuin,
  };
}

/**
 * Drift-only analytic benchmark: trades needed for 1000x on edge alone
 * (no volatility, no guards). Puts the Monte Carlo in context.
 */
export function driftOnlyTradesToTarget(
  muLog: number,
  startEquity = START_EQUITY,
  targetEquity = TARGET_EQUITY,
): number {
  return Math.log(targetEquity / startEquity) / muLog;
}

function fmtDays(d: number | null): string {
  if (d === null) return "never";
  if (d < 1000) return `${Math.round(d)}d`;
  return `${Math.round(d)}d (${(d / 365).toFixed(1)}y)`;
}

function parseFlag(name: string, fallback: number): number {
  const i = process.argv.indexOf(name);
  if (i < 0) return fallback;
  const v = Number(process.argv[i + 1]);
  return Number.isFinite(v) && v > 0 ? v : fallback;
}

function main(): void {
  const paths = parseFlag("--paths", DEFAULT_PATHS);
  const maxYears = parseFlag("--years", DEFAULT_MAX_YEARS);
  const seed = parseFlag("--seed", DEFAULT_SEED);

  const muLog = Math.log(1 + FROZEN_EXPECTANCY_PCT_PER_TRADE / 100);
  const sigma = calibrateSigma(
    FROZEN_MEDIAN_MAXDD_PCT,
    muLog,
    CALIBRATION_TRADES,
    seed,
  );
  const driftTrades = driftOnlyTradesToTarget(muLog);

  console.log("Growth E0 baseline (clever-cabin-pe9) — $1k -> $1M Monte Carlo");
  console.log(
    `frozen: expectancy=${FROZEN_EXPECTANCY_PCT_PER_TRADE}/trade (pct, net of taker-exit fees) ` +
      `medianDD=${FROZEN_MEDIAN_MAXDD_PCT}% guards=${MAX_DD_KILL_PCT}%/${DAILY_LOSS_HALT_PCT}% lev${LEVERAGE}`,
  );
  console.log(
    `calibrated sigma=${sigma.toFixed(6)}/trade over ${CALIBRATION_TRADES}-trade window ` +
      `paths=${paths} horizon=${maxYears}y seed=${seed}`,
  );
  console.log(
    `drift-only benchmark: ${Math.round(driftTrades).toLocaleString()} trades for 1000x on edge alone ` +
      `(=${(driftTrades / 5 / 365).toFixed(0)}y at 5/day before volatility/guards)`,
  );
  console.log(
    "trades/day | success | ruin (kill15%) | censored | median time-to-$1M | median ruin day",
  );
  const rows: GrowthSimResult[] = [];
  for (const k of TRADES_PER_DAY_RANGE) {
    const r = runGrowthMonteCarlo({
      tradesPerDay: k,
      paths,
      maxYears,
      seed: (seed + k * 7919) >>> 0,
      sigma,
      muLog,
    });
    rows.push(r);
    console.log(
      `${String(k).padStart(10)} | ${(r.successRate * 100).toFixed(2)}% | ${(r.ruinRate * 100).toFixed(2)}%        | ` +
        `${(r.censoredRate * 100).toFixed(2)}%    | ${fmtDays(r.medianDaysToTarget).padStart(12)} | ` +
        (r.medianRuinDays === null
          ? "n/a"
          : `${Math.round(r.medianRuinDays)}d`),
    );
  }
  console.log(
    JSON.stringify(
      { kind: "growth-e0-baseline", muLog, sigma, paths, maxYears, seed, rows },
      null,
      2,
    ),
  );
}

if (import.meta.main) main();
