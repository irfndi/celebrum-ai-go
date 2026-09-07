# TimesFM OOS Latency / Data Probe — Results (E4, clever-cabin-0g7)

Date: 2026-09-07. Scope: `docs/TIMESFM_OOS_SCOPE.md` sections 3–5 ONLY.
Research-only. No live wiring, no worker changes, no DB writes,
no new service dependencies. Nothing committed (this file left uncommitted
for reviewer verification).

## 1. Sidecar inventory (`services/timesfm-forecast/`)

- uv-managed CPython 3.12 sidecar, `timesfm[torch]==3.0.1` (see `pyproject.toml`).
- JSON-lines over stdio; persistent worker process (weights load once, then one
  JSON request per forecast). `--validate-only` exercises the protocol
  without weights.
- Defaults: checkpoint `google/timesfm-3.0-pytorch`, device `auto`,
  per-core batch size 4, torch threads unset (0).
- Bun wrapper `src/services/timesfm-client.ts` (`TimesFmWorker`):
  default checkpoint `google/timesfm-3.0-pytorch`, batch-size 4,
  per-request timeout 120s.
- License: 3.0 checkpoint under Google's NON-COMMERCIAL, NON-PRODUCTION
  license (per sidecar README). No approval sought in this probe —
  criterion 7 remains OPEN.

## 2. Stored panel inventory (`~/.neuratrade/data/neuratrade.db`, read-only)

From `scripts/timesfm-oos-spike.ts` (read-only, no weights needed):

| Panel (bybit-futures) | Bars/symbol | Span |
| --------------------- | ----------- | ---- |
| 15m BTC/SOL/ETH | 69,120 | 2024-08-23 → 2026-08-13 (~2y) |
| 5m BTC/SOL/ETH | ~107,713 | ~1y |

- Frozen tail (last 20%): start index 55,296, 13,824 bars (~144d) per symbol.
- OOS origins at context 256 / horizon 12 / step 12: 1,151 per symbol
  (96 capped in the standard harness). Panel freshness: max timestamp
  2026-08-13 (stale vs today 2026-09-07 — refresh with existing backfill
  scripts before full OOS scoring).

## 3. Weights (GB-scale → kept OUTSIDE the repo)

- No download was needed: checkpoint already in the local HF cache
  (`~/.cache/huggingface/hub/models--google--timesfm-3.0-pytorch/`, 1.2 GB).
  Nothing was written inside the repo.
- Pinned hash: snapshot `43046b85ec22d584a13f8098c2ed39c889e129c2`;
  `model.safetensors` blob `a7592b0a8432baee54483254e5647856911ce69e09d09a9bb65904b2d98f17da`
  (1,322,898,824 bytes). Probe ran with `HF_HUB_OFFLINE=1 --local-files-only`.

## 4. Latency probe (10 tail origins, persistent sidecar, batch 4, device cpu)

Config: context 256, horizon 12, step 12, quantiles on, symmetric averaging on,
znorm off — matches the proposed OOS config. Origins are the last 10 tail
origins (indices 68991–69099, step 12). Probe script lives in
`/tmp/timesfm-latency-probe.ts` (NOT in the repo). Per-request `latencyMs`
is the worker-reported inference time (excludes weights load); wall time and
cold-start-to-first-response measured separately; RSS sampled via `ps` on the
worker process.

| Symbol | Per-request worker latency (ms) [4,4,2 series] | p50 | p95 (max of 3) | Worker total | Cold start (spawn+load+infer) | Sidecar RSS peak |
| ------ | ---------------------------------------------- | --- | -------------- | ------------ | ----------------------------- | ---------------- |
| BTC | 1075 / 712 / 203 | 712 ms | 1075 ms | 1991 ms | 7.3 s | 1381 MB |
| SOL | 604 / 456 / 310 | 456 ms | 604 ms | 1370 ms | 5.6 s | 1413 MB |
| ETH | 658 / 354 / 206 | 354 ms | 658 ms | 1217 ms | 5.4 s | 1412 MB |

Pooled across 9 requests: min 203 ms, max 1075 ms.
3-symbol warm total: ~4.6 s. 3-symbol cold total (all restarts): ~18.3 s.

Machine: local Mac arm64, 8 CPUs, 8 GB RAM — NOT the server.

## 5. Budget verdict (scope doc section 4)

- MUST p95 per-request < 120 s: PASS. Observed worst 1075 ms (~0.9% of budget, ~111x headroom).
- SHOULD 3-symbol total < 60 s: PASS. Warm ~4.6 s; even full cold restart ~18.3 s.
- NICE 3-symbol total < 30 s: PASS on this machine.
- Resources (criterion 6, partial): sidecar RSS ~1.4 GB; fits in local 8 GB
  with headroom, no swap pressure observed. Server fit still unmeasured.

## 6. Go / no-go (this probe covers criteria 5 + partial 6 ONLY)

- Criterion 5 (latency): GO on the local machine.
- Criterion 6 (resources): PROVISIONAL GO locally; server measurement still required.
- NOT covered by this probe: criteria 1–4 (full OOS scoring), 7 (3.0 license
  clearance — still OPEN, automatic NO-GO for live), 8 (hygiene — held:
  repo files untouched except this new results file; probe script in /tmp;
  raw outputs in `/tmp/probe-{btc,sol,eth}.json`).
- Caveat: the scope doc requires measurement on the actual server
  (CPU model + torch threads unknown there). Server re-run required before
  any behind-flag integration proposal. If p95 misses there, try
  `--torch-threads` = core count first; GPU is out of scope.
