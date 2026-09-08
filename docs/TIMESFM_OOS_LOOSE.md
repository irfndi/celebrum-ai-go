# TimesFM loose operating-point OOS re-run (research, non-commercial self-run)

Task: clever-cabin-tf0. Prior OOS (`docs/TIMESFM_OOS_RESULTS.md`) failed gate 1:
2-7 trades per 1191 frozen-tail origins under the tight rule (whole q10-q90 band
beyond +/-friction). This re-run loosens the quantile-band operating point to the
endpoint — band excludes zero (long if q10 > 0, short if q90 < 0, else flat) —
and runs the SAME 8 go/no-go gates from `docs/TIMESFM_OOS_SCOPE.md` (friction
fee 0.06pc + slippage 2bps, 5-seed stress). Research-only: never production.

## Operating point

- Primary loose rule `directionalBand@0.00`: long if q10 > 0, short if q90 < 0,
  else flat. Same 1191 frozen-tail origins per symbol (15m, ctx 256, horizon 12,
  step 12, tail from `floor(71507 x 0.8)` = 57205, dataEnd 2026-09-07T05:45Z).
- No new inference: re-scores the saved observations in
  `/tmp/timesfm-wf-{btc,sol,eth}-full.json` (recompute reproduces the saved tight
  numbers exactly: 2/7/3 trades). Grid leg reuses the same saved forecasts with
  the real grid engine (`runGridBacktest` + validated per-symbol candidates at
  0.06pc/2bps) on the identical tail bars, plus the same 5-seed
  adverse-selection + taker-stop stress (makerFillProb 0.7).
- New files only: `services/neuratrade-cli-ts/scripts/timesfm-oos-loose.ts`,
  `scripts/timesfm-oos-loose.test.ts` (6 pass), output `/tmp/timesfm-oos-loose.json`.

## Gate table (loose rule, per symbol)

| Gate | BTC (16 trades) | SOL (19 trades) | ETH (10 trades) |
| ---- | --------------- | --------------- | --------------- |
| 1. Coverage >= 30 fixed-OOS trades | FAIL (16) | FAIL (19) | FAIL (10) |
| 2. Net AND PF > baseline | PASS (+0.32 vs -72.01; PF 1.06 vs 0.50) | PASS (+10.14 vs -113.90; PF 3.31 vs 0.58) | FAIL (net -3.03 vs -76.72 but PF 0.48 < 0.62) |
| 3. dirAcc > 50pc AND >= base+5pp | PASS (62.5 vs 46.4) | PASS (78.9 vs 48.4) | PASS (70.0 vs 49.2, moot — gate 2 fails) |
| 4. Stress: net-positive on >=3/5 seeds | PASS 5/5, vacuous (1 grid trade, +0.86pc; baseline +7.06/31 trades beats it clean) | FAIL 0/5 (1 trade, -2.65pc) | FAIL 0/5 (2 trades, -4.64pc) |
| 5. Latency p95 < 120s/req, 3-symbol < 60s | PASS (see below) | PASS | PASS |
| 6. RSS headroom, no swap, soak unaffected | FAIL (unmeasured) | FAIL (unmeasured) | FAIL (unmeasured) |
| 7. License cleared in writing (or 2.x re-run) | FAIL (research-only, no approval) | FAIL | FAIL |
| 8. Scope hygiene (additive only) | PASS | PASS | PASS |

Verdict: NO-GO on all three symbols. BTC/SOL pass gates 2-3 loose but fail gate 1;
ETH additionally fails gate 2; SOL/ETH fail gate 4 outright.

## Why coverage cannot be fixed with a quantile rule (binding constraint)

Quantile-band threshold ladder, same 1191 origins (trades / net%):

| Threshold t (long if q10 > t, short if q90 < -t) | BTC | SOL | ETH |
| --- | --- | --- | --- |
| 0.16 (tight, prior) | 2 / +2.16 | 7 / +2.33 | 3 / -3.51 |
| 0.10 | 6 / +0.77 | 11 / +2.74 | 3 / -3.51 |
| 0.05 | 9 / +0.45 | 15 / +5.62 | 6 / -3.54 |
| 0.00 (loosest possible) | 16 / +0.32 | 19 / +10.14 | 10 / -3.03 |

Even at t = 0 the ceiling is 10-19 trades — the loosest quantile rule still fails
gate 1 by ~2-3x on every symbol. Cause: median band width is 1.26pc (BTC) /
1.97pc (SOL) vs friction-scale moves (~0.16pc); the band straddles zero on
~98.5pc of origins, so band-excludes-zero is intrinsically rare. Loosening also
decays the edge monotonically (BTC net +2.16 -> +0.32). There is no quantile
threshold left to try — t = 0 is the floor.

Second binding constraint (diagnostic, point rule |point| >= 0.25pc): coverage is
plentiful (378/533/431 trades) but edge is gone — net -50.27/-48.87/-52.27,
dirAcc 47.4/52.3/49.9 (all fail gate 3; BTC/ETH below 50pc, SOL +3.9pp < +5pp).
Selectivity was load-bearing: the only profitable quantile calls are the rare
high-conviction ones, and there are never enough of them.

## Grid + stress + rolling (loose rule)

- Grid `directionalBand@0.00` on the frozen tail: BTC 1 trade +0.86pc, SOL 1 trade
  -2.65pc, ETH 2 trades -4.64pc — vs grid-native baselines +7.06 (31 trades),
  +24.44 (45), +15.18 (56). The overlay vetoes ~98.6pc of bars and the grid
  itself is what works on this tail; the forecast adds no entries of value.
- 5-seed stress: BTC 5/5 positive on a single +0.86pc trade (passes the letter of
  gate 4, means nothing); SOL/ETH 0/5. Winner trails baseline on clean data on
  all three symbols.
- Rolling honesty analogue (10 contiguous origin blocks, profitable-block share):
  BTC 30pc, SOL 50pc, ETH 20pc — no consistency claim survives.

## Gates 5-8 evidence

- Gate 5 PASS: inference config unchanged from the measured runs — 298
  requests/symbol at avg 0.69-0.90s/req, probe avg 1.44s, grid-filter runs avg
  0.37-1.68s, zero timeouts over ~1200 requests. Per-request << 120s MUST;
  ~3-5s for 3 symbols << 60s SHOULD. Caveat: per-request p95 was never recorded;
  future probes should log the distribution. No new inference was needed for
  this re-run.
- Gate 6 FAIL: sidecar RSS / swap headroom never measured in any run.
- Gate 7 FAIL: TimesFM 3.0 non-commercial, non-production license; no written
  approval. Research on our own box only.
- Gate 8 PASS: `git status` shows only the two new script files from this task;
  no src/, live/scalp, worker-protocol, DB-schema, or package.json changes; DB
  access read-only; no orders; no server processes.

## Bottom line

Say plainly: NO-GO as entry alpha on BTC/SOL/ETH. The binding constraint is
coverage at the quantile level — the bands are ~10x wider than the moves they
must sign, so even the loosest quantile rule (t = 0) tops out at 10-19 trades
vs the 30-trade minimum, and the point-based alternative that reaches coverage
fails accuracy/edge. Do not wire into entries. No further threshold tuning on
this frozen tail (that would be holdout overfitting); any revisit needs a
different signal construction, not a different threshold.

## Reproduce

```bash
bun test scripts/timesfm-oos-loose.test.ts   # 6 pure-function tests
bun run scripts/timesfm-oos-loose.ts         # offline re-score + grid + stress
# reads /tmp/timesfm-wf-{btc,sol,eth}-full.json, read-only DB, writes /tmp/timesfm-oos-loose.json
```
