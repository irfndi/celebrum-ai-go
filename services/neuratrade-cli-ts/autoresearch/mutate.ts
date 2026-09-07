/**
 * Mutation + keep/discard helpers for the overnight loop.
 */
import type { AutoresearchKnobs } from "./knobs.ts";

const AXES = [
  "gridStepPct",
  "stopRatio",
  "maxHoldBars",
  "targetRatio",
  "rungs",
  "gridMaxGrids",
  "gridPauseAfterLossBars",
  "chopGateAdxThreshold",
] as const;

/** Prefer risk-shaping axes when climbing toward lower drawdown. */
const WEIGHTED_AXES: readonly Axis[] = [
  "gridStepPct",
  "gridStepPct",
  "stopRatio",
  "stopRatio",
  "stopRatio",
  "maxHoldBars",
  "targetRatio",
  "rungs",
  "gridMaxGrids",
  "gridPauseAfterLossBars",
  "chopGateAdxThreshold",
];

type Axis = (typeof AXES)[number];

function clamp(n: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, n));
}

function round(n: number, digits: number): number {
  const p = 10 ** digits;
  return Math.round(n * p) / p;
}

const SCORE_EPS = 1e-9;

export interface MutateKnobsResult {
  readonly next: AutoresearchKnobs;
  readonly axis: Axis;
}

type MutableKnobs = {
  -readonly [K in keyof AutoresearchKnobs]: AutoresearchKnobs[K];
};

type AxisMutator = (
  next: MutableKnobs,
  base: AutoresearchKnobs,
  rng: () => number,
) => void;

const AXIS_MUTATORS = {
  gridStepPct: (next, base, rng) => {
    next.gridStepPct = round(
      clamp(base.gridStepPct * (0.7 + rng() * 0.8), 0.2, 3.0),
      2,
    );
  },
  stopRatio: (next, base, rng) => {
    next.stopRatio = round(
      clamp(base.stopRatio * (0.7 + rng() * 0.8), 0.5, 3.0),
      2,
    );
  },
  maxHoldBars: (next, base, rng) => {
    next.maxHoldBars = Math.round(
      clamp(base.maxHoldBars * (0.5 + rng()), 4, 96),
    );
  },
  targetRatio: (next, base, rng) => {
    next.targetRatio = round(
      clamp(base.targetRatio * (0.7 + rng() * 0.8), 0.8, 4.0),
      2,
    );
  },
  rungs: (next, base, rng) => {
    next.rungs = clamp(Math.round(base.rungs + (rng() < 0.5 ? -1 : 1)), 1, 3);
  },
  gridMaxGrids: (next, base, rng) => {
    next.gridMaxGrids = clamp(
      Math.round(base.gridMaxGrids + (rng() < 0.5 ? -1 : 1)),
      2,
      6,
    );
  },
  gridPauseAfterLossBars: (next, base, rng) => {
    next.gridPauseAfterLossBars = clamp(
      Math.round(base.gridPauseAfterLossBars + (rng() < 0.5 ? -2 : 2)),
      0,
      12,
    );
  },
  chopGateAdxThreshold: (next, _base, rng) => {
    const choices = [0, 20, 25, 30, 35];
    next.chopGateAdxThreshold =
      choices[Math.floor(rng() * choices.length)] ?? 0;
  },
} satisfies { readonly [A in Axis]: AxisMutator };

export function mutateKnobs(
  base: AutoresearchKnobs,
  rng: () => number = Math.random,
): MutateKnobsResult {
  const axis = WEIGHTED_AXES[Math.floor(rng() * WEIGHTED_AXES.length)]!;
  const next = { ...base };
  AXIS_MUTATORS[axis](next, base, rng);
  return { next, axis };
}

/**
 * Wide random restart to escape local maxima when 1–2 axis mutates plateau.
 */
export function hardRestartKnobs(
  rng: () => number = Math.random,
): AutoresearchKnobs {
  const chopChoices = [0, 0, 20, 25, 30];
  return {
    rungs: 1 + Math.floor(rng() * 3),
    gridStepPct: round(0.4 + rng() * 2.2, 2),
    gridMaxGrids: 2 + Math.floor(rng() * 4),
    gridPauseAfterLossBars: Math.floor(rng() * 9),
    stopRatio: round(0.8 + rng() * 1.8, 2),
    targetRatio: round(1.0 + rng() * 2.2, 2),
    maxHoldBars: Math.round(8 + rng() * 72),
    trendFilterPeriod: 0,
    chopGateAdxThreshold: chopChoices[Math.floor(rng() * chopChoices.length)]!,
    positionFraction: 1,
  };
}

export interface ShouldKeepInput {
  candidateScore: number;
  candidateGuardsOk: boolean;
  championScore: number;
  /** Once a champion has passed guards, never regress to a failing candidate. */
  championGuardsOk: boolean;
  /** Optional tie-breakers when scores are equal within SCORE_EPS. */
  candidateDrawdownPct?: number;
  championDrawdownPct?: number;
  candidateExpectancyPct?: number;
  championExpectancyPct?: number;
}

export function shouldKeep(input: ShouldKeepInput): boolean {
  if (!Number.isFinite(input.candidateScore)) return false;
  // Climb from a failing seed on score alone; after first guard-pass, require guards.
  if (input.championGuardsOk && !input.candidateGuardsOk) return false;

  if (input.candidateScore > input.championScore + SCORE_EPS) return true;

  // Tie on score: prefer lower drawdown, then higher expectancy.
  const tied =
    Math.abs(input.candidateScore - input.championScore) <= SCORE_EPS;
  if (!tied) return false;
  return winsOnDrawdown(input) || winsOnExpectancy(input);
}

function winsOnDrawdown(input: ShouldKeepInput): boolean {
  const cDd = input.candidateDrawdownPct;
  const hDd = input.championDrawdownPct;
  return (
    Number.isFinite(cDd) &&
    Number.isFinite(hDd) &&
    (cDd as number) < (hDd as number) - SCORE_EPS
  );
}

function winsOnExpectancy(input: ShouldKeepInput): boolean {
  const cExp = input.candidateExpectancyPct;
  const hExp = input.championExpectancyPct;
  if (!Number.isFinite(cExp) || !Number.isFinite(hExp)) return false;
  if ((cExp as number) <= (hExp as number) + SCORE_EPS) return false;
  // Only use expectancy when DD is not worse.
  const cDd = input.candidateDrawdownPct;
  const hDd = input.championDrawdownPct;
  return (
    !Number.isFinite(cDd) ||
    !Number.isFinite(hDd) ||
    (cDd as number) <= (hDd as number) + SCORE_EPS
  );
}

export function renderKnobsModule(k: AutoresearchKnobs): string {
  const body = JSON.stringify(k, null, 2);
  return `/**
 * THE editable surface for autoresearch (Karpathy train.py analogue).
 * Agents / the mutation loop may change these values. Nothing else.
 */
export interface AutoresearchKnobs {
  readonly rungs: number;
  readonly gridStepPct: number;
  readonly gridMaxGrids: number;
  readonly gridPauseAfterLossBars: number;
  readonly stopRatio: number;
  readonly targetRatio: number;
  readonly maxHoldBars: number;
  readonly trendFilterPeriod: number;
  readonly chopGateAdxThreshold: number;
  readonly positionFraction: number;
}

/** Current champion knobs — overwritten only on KEEP. */
export const knobs: AutoresearchKnobs = ${body};
`;
}
