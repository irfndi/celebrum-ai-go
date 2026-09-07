#!/usr/bin/env bun
/**
 * Overnight mutate → screen → (confirm on KEEP) → keep/discard.
 * Panel loaded once. Parallel-safe champion updates via lockfile.
 * Never touches live kill-switch or position rows.
 */
import { mkdirSync, appendFileSync, writeFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { knobs as seedKnobs, type AutoresearchKnobs } from "./knobs.ts";
import {
  loadAlignedPanel,
  evaluateKnobsOnPanel,
  evaluateHoldoutOnPanel,
  toDatasetProvenance,
  isProvenanceCompatible,
  type AlignedPanel,
  type DatasetProvenance,
  type EvaluateResult,
} from "./prepare.ts";
import {
  mutateKnobs,
  hardRestartKnobs,
  shouldKeep,
  renderKnobsModule,
} from "./mutate.ts";
import { withFileLock, readJsonFile, writeJsonFile } from "./lock.ts";
import { CLAIM_BARS, GOAL_VERSION, meetsClaimBars } from "./goals.ts";

const here = dirname(fileURLToPath(import.meta.url));
const resultsDir = join(here, "results");
const knobsPath = join(here, "knobs.ts");
const championPath = join(resultsDir, "champion.json");
const championLock = join(resultsDir, "champion.lock");
const claimedPath = join(resultsDir, "claimed.json");
const ledgerPath = join(resultsDir, "ledger.jsonl");
const goalsPath = join(resultsDir, "goals.md");

function arg(name: string, fallback: string): string {
  const hit = process.argv.find((a) => a.startsWith(`--${name}=`));
  return hit?.split("=")[1] ?? fallback;
}

const trials = Number(arg("trials", "500"));
const worker = Number(arg("worker", "0"));
const workers = Number(arg("workers", "1"));
const panelSymbols = Number(arg("symbols", "8"));
const screenSteps = Number(arg("screen-steps", "12"));
const screenBudget = Number(arg("screen-budget-sec", "45"));
const confirmSteps = Number(arg("confirm-steps", "40"));
const confirmBudget = Number(arg("confirm-budget-sec", "180"));

/** Deterministic RNG per worker for diverse mutations. */
function mulberry32(seed: number): () => number {
  let t = seed >>> 0;
  return () => {
    t += 0x6d2b79f5;
    let r = Math.imul(t ^ (t >>> 15), 1 | t);
    r ^= r + Math.imul(r ^ (r >>> 7), 61 | r);
    return ((r ^ (r >>> 14)) >>> 0) / 4294967296;
  };
}

const rng = mulberry32(0x9e3779b9 ^ ((worker + 1) * 0x85ebca6b));

mkdirSync(resultsDir, { recursive: true });

interface ChampionState {
  knobs: AutoresearchKnobs;
  /** Confirm-phase median log-return (claim metric). */
  score: number;
  /** Screen-phase median log-return — used only for the cheap gate. */
  screenScore: number;
  guardsOk: boolean;
  /** Confirm-phase risk/edge for score-tie KEEP decisions. */
  medianDrawdownPct: number;
  expectancyPct: number;
  /**
   * Dataset that produced these scores (exchange + timeframe + panel hash).
   * Null until (re)evaluated. Scores are only comparable within one dataset.
   */
  dataset: DatasetProvenance | null;
}

function loadChampionUnlocked(panel?: AlignedPanel): ChampionState {
  const raw = readJsonFile<
    ChampionState & {
      screenScore?: number;
      medianDrawdownPct?: number;
      expectancyPct?: number;
      dataset?: DatasetProvenance | null;
    }
  >(championPath);
  if (raw?.knobs) {
    const state: ChampionState = {
      knobs: raw.knobs,
      score: Number.isFinite(raw.score) ? raw.score : Number.NEGATIVE_INFINITY,
      screenScore: Number.isFinite(raw.screenScore)
        ? (raw.screenScore as number)
        : Number.NEGATIVE_INFINITY,
      guardsOk: Boolean(raw.guardsOk),
      medianDrawdownPct: Number.isFinite(raw.medianDrawdownPct)
        ? (raw.medianDrawdownPct as number)
        : Number.POSITIVE_INFINITY,
      expectancyPct: Number.isFinite(raw.expectancyPct)
        ? (raw.expectancyPct as number)
        : Number.NEGATIVE_INFINITY,
      dataset: (raw.dataset as DatasetProvenance | null) ?? null,
    };
    // A panel change (venue, symbols, span) makes stored scores incomparable.
    // Invalidate them so the search re-earns the crown on the new dataset.
    if (panel && !isProvenanceCompatible(state.dataset, panel)) {
      console.log(
        `champion dataset mismatch (stored=${state.dataset?.panelHash ?? "none"}:${state.dataset?.exchange ?? "none"} current=${panel.panelHash}:${panel.exchange}) — invalidating old scores.`,
      );
      return {
        knobs: state.knobs,
        score: Number.NEGATIVE_INFINITY,
        screenScore: Number.NEGATIVE_INFINITY,
        guardsOk: false,
        medianDrawdownPct: Number.POSITIVE_INFINITY,
        expectancyPct: Number.NEGATIVE_INFINITY,
        dataset: null,
      };
    }
    return state;
  }
  return {
    knobs: { ...seedKnobs },
    score: Number.NEGATIVE_INFINITY,
    screenScore: Number.NEGATIVE_INFINITY,
    guardsOk: false,
    medianDrawdownPct: Number.POSITIVE_INFINITY,
    expectancyPct: Number.NEGATIVE_INFINITY,
    dataset: null,
  };
}

function persistChampionUnlocked(state: ChampionState): void {
  writeJsonFile(championPath, state);
  // Only worker 0 writes knobs.ts to avoid thrash; others still update champion.json.
  if (worker === 0) {
    writeFileSync(knobsPath, renderKnobsModule(state.knobs));
  }
}

export interface LedgerRow {
  readonly ts: string;
  readonly worker: number;
  readonly trial: number;
  readonly decision: string;
  readonly axis: string | null;
  readonly knobs: AutoresearchKnobs;
  readonly screen?: EvaluateResult;
  readonly result?: EvaluateResult;
  /** Frozen-holdout gate result (present on CLAIMED / HOLDOUT_REJECT rows). */
  readonly holdout?: EvaluateResult;
  readonly championScore?: number;
  readonly championScreenScore?: number;
}

function appendLedger(row: LedgerRow): void {
  appendFileSync(ledgerPath, `${JSON.stringify(row)}\n`);
}

function goalsClaimed(r: EvaluateResult): boolean {
  return r.phase === "confirm" && r.guardsOk && meetsClaimBars(r);
}

function writeGoals(status: string, r: EvaluateResult | null): void {
  // Never downgrade a durable claim.
  if (
    status !== "CLAIMED" &&
    readJsonFile<{ claimedAt?: string }>(claimedPath)?.claimedAt
  ) {
    return;
  }
  const c = CLAIM_BARS;
  const body = `# Autoresearch goals (${GOAL_VERSION})

Status: **${status}**
Updated: ${new Date().toISOString()}
Worker: ${worker}/${workers}

| Goal | Target | Current (confirm) |
| --- | --- | --- |
| Profitability (med log-ret) | ≥ ${c.minMedianLogReturn} | ${r ? r.medianLogReturn.toFixed(4) : "n/a"} |
| Win rate | ≥ ${c.minWinRatePct}% | ${r ? r.winRatePct.toFixed(1) : "n/a"} |
| Throughput (trades/sym-mo) | ≥ ${c.minTradesPerSymMonth} | ${r ? r.tradesPerSymMonth.toFixed(1) : "n/a"} |
| Drawdown (med) | ≤ ${c.maxMedianDrawdownPct}% | ${r ? r.medianDrawdownPct.toFixed(1) : "n/a"} |
| Expectancy | ≥ ${c.minExpectancyPct} | ${r ? r.expectancyPct.toFixed(4) : "n/a"} |

v1/v2 soak baselines live in champion-soak.json (paper/testnet).
Live trading remains frozen until credential rotation + position reconciliation.
`;
  writeFileSync(goalsPath, body);
}

/** Holdout gate: same bars as CLAIM_BARS, on the frozen tail, evaluated once. */
function holdoutPasses(h: EvaluateResult): boolean {
  return h.phase === "holdout" && h.guardsOk && meetsClaimBars(h);
}

function persistClaim(
  r: EvaluateResult,
  holdout: EvaluateResult,
  knobs: AutoresearchKnobs,
): void {
  writeJsonFile(claimedPath, {
    claimedAt: new Date().toISOString(),
    worker,
    knobs,
    dataset: toDatasetProvenance(panel),
    result: r,
    holdout,
  });
  writeGoals("CLAIMED", r);
}

/**
 * Final gate for a confirm-phase claim candidate: evaluate the frozen holdout
 * exactly once (outside the champion lock), then claim only if it also passes.
 * A HOLDOUT_REJECT means the candidate overfit selection — keep searching.
 */
function runHoldoutGate(
  candidate: EvaluateResult,
  candidateKnobs: AutoresearchKnobs,
  trial: number,
  axis: string | null,
  screen: EvaluateResult | undefined,
): "CLAIMED" | "HOLDOUT_REJECT" | "CLAIMED_BY_OTHER" {
  console.log("  confirm meets claim bars → evaluating frozen holdout once...");
  const holdout = evaluateHoldoutOnPanel(candidateKnobs, panel, {
    budgetSec: confirmBudget,
  });
  console.log(
    `  holdout score=${holdout.score.toFixed(4)} guardsOk=${holdout.guardsOk} reason=${holdout.reason} windows=${holdout.windows} elapsedMs=${holdout.elapsedMs}`,
  );
  return withFileLock(championLock, () => {
    if (readJsonFile<{ claimedAt?: string }>(claimedPath)?.claimedAt) {
      return "CLAIMED_BY_OTHER" as const;
    }
    if (holdoutPasses(holdout)) {
      appendLedger({
        ts: new Date().toISOString(),
        worker,
        trial,
        decision: "CLAIMED",
        axis,
        knobs: candidateKnobs,
        screen,
        result: candidate,
        holdout,
      });
      persistClaim(candidate, holdout, candidateKnobs);
      return "CLAIMED" as const;
    }
    appendLedger({
      ts: new Date().toISOString(),
      worker,
      trial,
      decision: "HOLDOUT_REJECT",
      axis,
      knobs: candidateKnobs,
      screen,
      result: candidate,
      holdout,
    });
    writeGoals("IN_PROGRESS", candidate);
    return "HOLDOUT_REJECT" as const;
  });
}

console.log(
  `autoresearch w${worker}/${workers}: trials=${trials} panelSymbols=${panelSymbols} screen=${screenSteps}@${screenBudget}s confirm=${confirmSteps}@${confirmBudget}s`,
);

if (readJsonFile<{ claimedAt?: string }>(claimedPath)?.claimedAt) {
  console.log("GOALS already CLAIMED (claimed.json present) — exiting.");
  process.exit(0);
}

console.log("loading candle panel once...");
const panel = loadAlignedPanel({ symbols: panelSymbols });
console.log(
  `panel ready: ${panel.symbols.length} symbols, refLen=${panel.refLen}, loadedMs=${panel.loadedMs} venue=${panel.exchange}/${panel.timeframe}->${panel.panelTimeframe} hash=${panel.panelHash} holdoutBars=${panel.holdoutBars}`,
);

function evalScreen(k: AutoresearchKnobs): EvaluateResult {
  return evaluateKnobsOnPanel(k, panel, {
    phase: "screen",
    maxSteps: screenSteps,
    budgetSec: screenBudget,
  });
}

function evalConfirm(k: AutoresearchKnobs): EvaluateResult {
  return evaluateKnobsOnPanel(k, panel, {
    phase: "confirm",
    maxSteps: confirmSteps,
    budgetSec: confirmBudget,
  });
}

// Seed / backfill screenScore under lock (screen vs confirm are not comparable).
const seedClaimCandidate = withFileLock(championLock, () => {
  let champ = loadChampionUnlocked(panel);
  const needsSeed =
    !Number.isFinite(champ.score) ||
    champ.score === Number.NEGATIVE_INFINITY ||
    !isProvenanceCompatible(champ.dataset, panel);
  const needsScreenBackfill =
    !needsSeed &&
    (!Number.isFinite(champ.screenScore) ||
      champ.screenScore === Number.NEGATIVE_INFINITY);

  if (needsSeed) {
    console.log("evaluating seed champion (screen → confirm)...");
    const screen = evalScreen(champ.knobs);
    const base = evalConfirm(champ.knobs);
    champ = {
      knobs: champ.knobs,
      score: base.score,
      screenScore: screen.score,
      guardsOk: base.guardsOk,
      medianDrawdownPct: base.medianDrawdownPct,
      expectancyPct: base.expectancyPct,
      dataset: toDatasetProvenance(panel),
    };
    persistChampionUnlocked(champ);
    appendLedger({
      ts: new Date().toISOString(),
      worker,
      trial: 0,
      decision: "SEED",
      axis: null,
      knobs: champ.knobs,
      screen,
      result: base,
    });
    writeGoals("IN_PROGRESS", base);
    console.log(
      `SEED confirm=${base.score.toFixed(4)} screen=${screen.score.toFixed(4)} guardsOk=${base.guardsOk} reason=${base.reason}`,
    );
    if (goalsClaimed(base)) return { knobs: champ.knobs, result: base, screen };
    return null;
  } else if (needsScreenBackfill) {
    console.log("backfilling champion screenScore for fair gate...");
    const screen = evalScreen(champ.knobs);
    champ = {
      ...champ,
      screenScore: screen.score,
      dataset: toDatasetProvenance(panel),
    };
    persistChampionUnlocked(champ);
    console.log(
      `backfill screenScore=${screen.score.toFixed(4)} (confirm champ=${champ.score.toFixed(4)})`,
    );
    return null;
  } else if (!isProvenanceCompatible(champ.dataset, panel)) {
    // Scores looked seeded but belong to another dataset — rebind on re-eval.
    champ = { ...champ, dataset: toDatasetProvenance(panel) };
    persistChampionUnlocked(champ);
    return null;
  }
  return null;
});

if (seedClaimCandidate) {
  const gate = runHoldoutGate(
    seedClaimCandidate.result,
    seedClaimCandidate.knobs,
    0,
    null,
    seedClaimCandidate.screen,
  );
  if (gate === "CLAIMED") {
    console.log("GOALS CLAIMED on seed (+holdout) — stopping.");
    process.exit(0);
  }
  console.log(
    `seed claim blocked by frozen holdout (${gate}) — continuing search.`,
  );
}

for (let i = 1; i <= trials; i++) {
  if (readJsonFile<{ claimedAt?: string }>(claimedPath)?.claimedAt) {
    console.log("GOALS CLAIMED by another worker — stopping.");
    process.exit(0);
  }
  const localChamp = loadChampionUnlocked(panel);
  // Occasional hard restart (~8%) escapes local maxima; else 1–2 axis mutate.
  let next: AutoresearchKnobs;
  let axis: string;
  if (rng() < 0.08) {
    next = hardRestartKnobs(rng);
    axis = "HARD_RESTART";
  } else {
    const first = mutateKnobs(localChamp.knobs, rng);
    next = first.next;
    axis = first.axis;
    if (rng() < 0.4) {
      const second = mutateKnobs(next, rng);
      next = second.next;
      axis = `${first.axis}+${second.axis}`;
    }
  }
  console.log(`\n[w${worker} trial ${i}/${trials}] mutate ${axis} → screen...`);
  const screen = evalScreen(next);
  console.log(
    `  screen score=${screen.score.toFixed(4)} guardsOk=${screen.guardsOk} elapsedMs=${screen.elapsedMs}`,
  );

  // Compare screen-to-screen only (never against confirm score).
  // Small absolute slack lets near-misses pay for confirm — without it the
  // loop plateaus when champScreen is a hard ceiling (v2 overnight stall).
  const SCREEN_SLACK = 0.0025;
  const screenPromising =
    Number.isFinite(screen.score) &&
    (localChamp.screenScore === Number.NEGATIVE_INFINITY ||
      screen.score > localChamp.screenScore - SCREEN_SLACK);

  if (!screenPromising) {
    appendLedger({
      ts: new Date().toISOString(),
      worker,
      trial: i,
      decision: "DISCARD_SCREEN",
      axis,
      knobs: next,
      screen,
      championScreenScore: localChamp.screenScore,
      championScore: localChamp.score,
    });
    console.log(
      `  DISCARD_SCREEN (screen ${screen.score.toFixed(4)} <= champScreen ${localChamp.screenScore.toFixed(4)} - ${SCREEN_SLACK})`,
    );
    continue;
  }

  console.log("  screen promising → confirm...");
  const result = evalConfirm(next);

  const decision = withFileLock(championLock, () => {
    const champ = loadChampionUnlocked(panel);
    const keep = shouldKeep({
      candidateScore: result.score,
      candidateGuardsOk: result.guardsOk,
      championScore: champ.score,
      championGuardsOk: champ.guardsOk,
      candidateDrawdownPct: result.medianDrawdownPct,
      championDrawdownPct: champ.medianDrawdownPct,
      candidateExpectancyPct: result.expectancyPct,
      championExpectancyPct: champ.expectancyPct,
    });
    appendLedger({
      ts: new Date().toISOString(),
      worker,
      trial: i,
      decision: keep ? "KEEP" : "DISCARD_CONFIRM",
      axis,
      knobs: next,
      screen,
      result,
      championScore: champ.score,
      championScreenScore: champ.screenScore,
    });
    if (keep) {
      const nextState: ChampionState = {
        knobs: next,
        score: result.score,
        screenScore: screen.score,
        guardsOk: result.guardsOk,
        medianDrawdownPct: result.medianDrawdownPct,
        expectancyPct: result.expectancyPct,
        dataset: toDatasetProvenance(panel),
      };
      persistChampionUnlocked(nextState);
      // A confirm claim candidate must still pass the frozen holdout, which
      // runs exactly once outside the lock (never as a selection signal).
      if (goalsClaimed(result)) return "CLAIM_CANDIDATE" as const;
      writeGoals("IN_PROGRESS", result);
      return "KEEP" as const;
    }
    writeGoals("IN_PROGRESS", result);
    return "DISCARD_CONFIRM" as const;
  });

  if (decision === "CLAIM_CANDIDATE") {
    const gate = runHoldoutGate(result, next, i, axis, screen);
    console.log(
      `  ${gate} confirm score=${result.score.toFixed(4)} guardsOk=${result.guardsOk} reason=${result.reason} elapsedMs=${result.elapsedMs}`,
    );
    if (gate === "CLAIMED") {
      console.log("\nGOALS CLAIMED (+holdout) — stopping loop.");
      process.exit(0);
    }
    if (gate === "CLAIMED_BY_OTHER") {
      console.log("GOALS CLAIMED by another worker — stopping.");
      process.exit(0);
    }
    continue;
  }

  console.log(
    `  ${decision} confirm score=${result.score.toFixed(4)} guardsOk=${result.guardsOk} reason=${result.reason} elapsedMs=${result.elapsedMs}`,
  );
}

console.log("\nloop finished without claim — champion retained.");
process.exit(1);
