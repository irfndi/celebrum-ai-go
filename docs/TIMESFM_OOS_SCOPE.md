# TimesFM Forecast Entries — OOS Scope (E4, clever-cabin-0g7)

Status: SCOPE ONLY. No live wiring, no worker changes, no DB writes, no new
service dependencies. TimesFM output must not place, resize, or close any
position until the go criteria below all pass.

## 1. What already exists (reuse, do not rebuild)

- `services/timesfm-forecast/` — uv-managed CPython 3.12 sidecar,
  `timesfm[torch]==3.0.1`, JSON-lines over stdio. Research-only by design
  (see its README). Persistent worker process: weights load once, then one
  JSON request per forecast.
- `services/neuratrade-cli-ts/src/services/timesfm-client.ts` — Bun wrapper
  (`TimesFmWorker`). Default checkpoint `google/timesfm-3.0-pytorch`,
  device `auto`, batch-size 4, per-request timeout 120s.
- `services/neuratrade-cli-ts/src/cli/timesfm.ts` — `scalp timesfm-forecast`
  CLI (context 32–15360 bars, horizon 1–1024, `--dry-run` validates the
  protocol without weights). Builds log-close + log1p-volume series.
- `services/neuratrade-cli-ts/src/scalping/timesfm-evaluation.ts` — causal
  walk-forward scorer: origins only see candles through `index`, scores only
  the held-out future close, friction-aware, with a no-model baseline.
- `services/neuratrade-cli-ts/src/scalping/timesfm-grid-filter.ts` — entry
  overlay policies (`standAsidePoint`, `directionalPoint` (+contrarian),
  `standAsideBand`). Overlay controls NEW entries only; exits stay grid-native.
- `services/neuratrade-cli-ts/scripts/timesfm-walkforward.ts` and
  `scripts/timesfm-grid-filter.ts` — research-only OOS harnesses. Read-only
  DB access, no orders, no risk-state changes.
- `services/neuratrade-cli-ts/scripts/timesfm-oos-spike.ts` (new, throwaway)
  — prints panel coverage, the frozen-holdout OOS plan sizes, and the latency
  budget. Read-only. Delete or keep as a diagnostic; never import from `src/`.

## 2. Model variant + license

- Variant: TimesFM 3.0 PyTorch (`timesfm[torch]==3.0.1`,
  checkpoint `google/timesfm-3.0-pytorch`). Proposed OOS config: context 256
  bars, horizon 12 bars, quantiles on, symmetric averaging on, znorm off,
  log-volume as second target (matches `timesfm-walkforward.ts` defaults).
- License risk (biggest non-technical item): per
  `services/timesfm-forecast/README.md`, the 3.0 checkpoint is under Google's
  separate NON-COMMERCIAL, NON-PRODUCTION license. OOS research on our own box
  is the current use; any live-money trigger needs license verification plus
  written approval FIRST.
- Fallback if 3.0 cannot be cleared: TimesFM 2.x checkpoints (Apache-2.0).
  Expect lower accuracy; re-run the same OOS protocol to quantify the gap.
- Action: confirm license + approval before E4 leaves scoping. No approval =
  automatic NO-GO for anything beyond research.

## 3. Data needs (reuse stored panels, no new collection)

Live engine is `bybit-futures`. Stored coverage today (`~/.neuratrade`):

| Panel | Bars/symbol | Span |
| ----- | ----------- | ---- |
| 15m BTC/SOL/ETH | ~69,120 | 2024-08 → 2026-08 (~2y) |
| 5m BTC/SOL/ETH | ~107,700 | 2025-08 → 2026-08 (~1y) |

- Primary OOS: 15m panels (same bars the validated grid candidates in
  `src/scalping/grid-candidate.ts` were fitted on). 256-bar context = 64h of
  market memory; 12-bar horizon = 3h ahead.
- Secondary: 5m panels for a sensitivity check only (shorter span, fewer
  fixed-OOS trades; do not promote on 5m alone).
- No new exchange, symbol, or backfill work in scope. If a panel is stale
  (>48h behind, same freshness rule as `grid-validation.ts`), refresh with the
  existing backfill scripts before scoring.

## 4. Inference latency vs the 900s bar loop

- A 15m bar closes every 900s. That is the hard real-time budget: one forecast
  per symbol per bar, so 3 forecasts (BTC/SOL/ETH) per 900s.
- Bun client timeout is 120s per request; paper-trading iterations default to
  60s spacing. Targets:
  - MUST: p95 per-request latency < 120s on CPU (else the client kills it).
  - SHOULD: all 3 symbols < 60s total (fits inside one paper iteration).
  - NICE: all 3 symbols < 30s total (headroom for retries + exits).
- Why this should fit: sidecar stays alive between requests (no reload cost);
  256×12 is a small forward pass; batch-size 4 covers 3 symbols in one batch.
- Must still MEASURE on the actual server (CPU model + torch threads unknown).
  Procedure: run `scripts/timesfm-walkforward.ts --max-origins 10 --device cpu`
  per symbol, record per-request `latencyMs` from the worker response, compute
  p50/p95. Try `--torch-threads` = core count if p95 misses. The spike script
  prints the budget table without needing weights.

## 5. CPU / RAM on the server

- Weights: TimesFM-3 class checkpoint downloads once (GB-scale bandwidth, one
  time) and lives in the HF cache (`--cache-dir` or default). Pin the
  checkpoint hash before OOS so results are reproducible.
- RAM: expect low-single-digit GB RSS for the sidecar + weights on CPU.
  Measure `RSS` during the 10-origin probe above; server must hold sidecar +
  soak engine + SQLite cache with headroom (fail if swap is touched).
- CPU: inference is the only load (no training). Start with CPU; GPU is out of
  scope unless p95 misses the MUST target after thread tuning.
- Isolation: run OOS probes on a non-live host (or paused soak) first. The
  sidecar never shares memory with the trading loop — stdio JSON only — so
  the blast radius of a crash is a failed diagnostic, not a missed exit.

## 6. OOS protocol (reuse the frozen holdout)

Reuse the grid gate's frozen definitions verbatim so TimesFM is judged on the
same unseen data as the grid itself (`grid-validation.ts`):

- Frozen holdout: last 20% tail (`oosStart = floor(n × 0.8)`). On ~69,120 15m
  bars that is ~13,824 bars (~144 days) per symbol. Minimum 30 fixed-OOS
  trades or the window is invalid — same rule.
- Rolling honesty: train 11,520 / test 4,320 bars (120d/45d at 15m), ≥10
  windows; report profitable-window share alongside the fixed tail.
- Causality: origins from `buildTimesFmEvaluationOrigins` only (context
  through `index`, score at `futureIndex`); step = horizon (12) so forecasts
  never overlap; bound with `--max-origins` for cost.
- Friction: fee 0.06% + 2bps slippage (walkforward defaults — tougher than the
  grid candidates' 0.02%/1bp). TimesFM must clear the tougher bar.
- Baseline: the scorer's no-model baseline direction on every origin; report
  direction accuracy, MAE, win rate, profit factor, net return for model AND
  baseline side by side. Quantile gating (`standAsideBand`) fails closed on
  incomplete bands.
- Stress analogue: re-score the winning policy under the gate's 5-seed
  adverse-selection + taker-stop stress (makerFillProb 0.7). A filter that
  only works on clean fills is a NO-GO.
- Symbols: BTC, then SOL, then ETH (cohort order). Promote per symbol, never
  as a basket average hiding one loser.

## 7. Integration points (decide AFTER OOS, in this order)

1. `standAsidePoint` / `standAsideBand` (signal-feed FILTER): TimesFM vetoes
   new grid entries when the point forecast is strong against the grid side
   or the quantile band is too wide. Safest first step — it can only reduce
   trading, never invent entries. Recommended if OOS passes.
2. `directionalPoint` (ENTRY TRIGGER): TimesFM picks long/short on weak-grid
   bars. Needs strictly stronger evidence (see go criteria). Not recommended
   as a first integration.
3. Never in scope: TimesFM touching exits, size, leverage, kill-switch,
   circuit-breaker, or risk guards. Exits stay grid-native (`runGridBacktest`
   parity); risk files are not opened in this epic.

Wiring (only after GO): overlay array threaded into the grid paper engine as
an entry veto, behind an env flag defaulting OFF, with per-bar provenance
logging (forecast id → origin → decision). That design is NOT part of this
scoping task.

## 8. Cost estimate

- API/model fees: $0 (local weights, no vendor calls).
- One-time: checkpoint download (bandwidth) + ~1 engineer-day for the
  10-origin latency probe and full OOS runs (mostly machine time).
- Recurring if promoted: zero marginal $ cost; only CPU minutes on the
  existing server (3 inferences / 900s — negligible vs the soak loop).
- Only real cost is calendar time for enough OOS origins to be significant
  (≥96 origins/symbol ≈ full-tail coverage at step 12; CPU-bound, unattended).

## 9. Go / no-go criteria

ALL must hold on the frozen 15m tail per symbol, else NO-GO for that symbol:

1. Coverage: fixed-OOS trades ≥ 30 (gate minimum; more is better).
2. Edge vs baseline: model net return > baseline net return AND model profit
   factor > baseline profit factor after 0.06%/2bps friction.
3. Accuracy: direction accuracy > 50% AND above baseline by ≥ 5pp.
4. Stress: winning policy stays net-positive under ≥3 of 5 adverse-selection
   seeds (clean-fill-only winners are rejected).
5. Latency: p95 per-request < 120s (MUST), 3-symbol total < 60s (SHOULD).
6. Resources: sidecar RSS fits with headroom; no swap; soak loop unaffected.
7. License: 3.0 commercial/production use cleared in writing, OR fallback 2.x
   re-run passes criteria 1–6 on its own.
8. Scope hygiene: no changes to live/scalp paths, worker protocol, DB schema,
   or service `package.json` during evaluation (this task adds docs + a
   read-only spike only).

Pass (1)–(8) → allow a BEHIND-FLAG paper veto integration proposal (new task,
new review). Fail any → record the failing metric and stop; do not tune the
protocol until it passes (no holdout overfitting).

## 10. Spike script

`services/neuratrade-cli-ts/scripts/timesfm-oos-spike.ts` (throwaway):

```bash
bun run scripts/timesfm-oos-spike.ts            # coverage + OOS plan + budget
bun run scripts/timesfm-oos-spike.ts --json     # one JSON document
bun test scripts/timesfm-oos-spike.test.ts      # pure-function tests
```

Read-only SQLite, no imports from `src/`, no orders, no state changes. It
proves panel coverage, sizes the frozen-tail OOS (origins per symbol at
context 256 / horizon 12 / step 12), and prints the latency budget — without
needing model weights. Full inference numbers come from the 10-origin probe
in section 4.
