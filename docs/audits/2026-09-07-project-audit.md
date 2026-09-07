# Project audit: 2026-09-07

Audited revision: `ec869a951376486507b4b91db99d83a4f9d21cdc` locally and on the authorized server. Runtime observations: approximately 03:19-03:27 UTC / 10:19-10:27 Asia/Jakarta.

**Verdict: operational processes are running, but the project is neither all green nor safe to leave with only credential rotation outstanding.** No evidence supports real-money promotion.

## Findings, ordered by priority

### P0: scheduled candle sync clears safety stops

`services/neuratrade-cli-ts/scripts/seed-champion-soak-candles.py:244` unconditionally inserts or updates `risk_kill_switch` to `engaged=0`, including `--incremental`. The champion ecosystem runs this every 15 minutes. Both isolated soak databases showed disengaged switches updated at 03:15:15/16 UTC, matching the candle sync log.

A manually engaged or risk-triggered stop is erased by unrelated data maintenance. Preserve the switch during incremental synchronization; initial schema creation must not authorize recurring disengagement. Tracked: **clever-cabin-9m4**. Impact currently observed in paper/testnet homes, not evidence of a mainnet order.

### P1: paper candles are fresh timestamps containing unfinished prices

`services/neuratrade-cli-ts/scripts/backfill-bybit-15m.ts:137` uses `INSERT OR IGNORE` without excluding the current open candle. `scripts/seed-champion-soak-candles.py:190` copies only timestamps newer than its watermark, so later final values cannot repair existing rows. The watermark is per timeframe, not per symbol, which can also strand lagging symbols.

Direct comparison for BTC's completed 2026-09-07 03:00 UTC 15-minute candle:

| Source | Open | High | Low | Close | Volume |
| --- | ---: | ---: | ---: | ---: | ---: |
| Paper SQLite | 79828.7 | 79842 | 79828.6 | 79842 | 3.141 |
| Bybit public final candle | 79828.7 | 79943.9 | 79714.3 | 79752.1 | 591.009 |

HOLD on these rows is not proof that the completed market candle produced no signal. Use closed candles or a bounded overlapping upsert, with per-symbol watermarks. Existing issue updated: **clever-cabin-08m**.

### P1: soaks do not run the advertised frozen champion

`services/neuratrade-cli-ts/src/cli/scalp.ts:3318` gives whitelist `gridParams` precedence over CLI arguments derived from `champion-soak.json`. Runtime state confirms the old whitelist values win:

| Parameter | Frozen soak file | Actual persisted engines |
| --- | ---: | ---: |
| gridStepPct | 1.3 | 1.29 |
| gridMaxGrids | 2 | 3 |
| gridPauseAfterLossBars | 2 | 4 |
| targetRatio | 1.95 | 1.9 |

The persisted stop ratio and hold duration come from the newer CLI knobs, yielding a hybrid configuration. Generate and verify a single frozen configuration before attributing forward results to the champion. Tracked: **clever-cabin-cdv**.

### P1: demo signals use a different market from paper/research

`services/neuratrade-cli-ts/src/cli/scalp.ts:2876` chooses the repository gateway for paper and the live gateway for `--live`. `src/market-data/gateways/bybit.ts:5` defaults that public feed to `api-testnet.bybit.com`. Copying mainnet candles into the demo database therefore does not control demo signal generation.

At the same persisted timestamp, LINK's demo grid base was **1436.694**, while paper's was **13.087**. Testnet matching behavior cannot validate the same price process used by the mainnet research panel. Explicitly separate signal data from execution environment and document what the demo is intended to prove. Tracked: **clever-cabin-1e1**.

### P1: search/archive cannot recover from an abandoned champion lock

`services/neuratrade-cli-ts/autoresearch/lock.ts:26` creates an exclusive file without owner identity or stale-lock recovery. A worker death while holding it leaves future writers permanently timing out. `scripts/archive-ledger-r2.sh:19` stops those workers, so scheduled maintenance can trigger this condition. PM2 restart does not remove the lock.

The catch at `lock.ts:37` also catches callback failures and retries the entire mutation, masking the original error and potentially replaying ledger/state writes. An isolated runnable check confirmed three callback executions after one failing request and failure to acquire a pre-existing abandoned lock. Use a crash-released lock or validated owner recovery; retry acquisition failures only. Tracked: **clever-cabin-a3h**.

### P1: main CI is red; local quality checks independently fail

At the audited revision:

- [Validation run 34076547424](https://github.com/irfndi/NeuraTrade/actions/runs/34076547424) fails Telegram formatting and the CLI test suite.
- [Lightweight Native Tests run 34076547425](https://github.com/irfndi/NeuraTrade/actions/runs/34076547425) calls removed `make test-scripts` and fails immediately.
- CLI tests: **1309 pass, 1 fail**, reproduced locally after installing frozen dependencies.
- Failing test: `services/neuratrade-cli-ts/tests/e2e/live-execution-safety.test.ts:97`, expecting rejection for an unvalidated sandbox grid profile. Current demo exemption and test expectation disagree. This failure alone does not demonstrate a mainnet bypass; the mainnet-disabled test passes.
- CLI lint: five errors, including complexity 23 versus maximum 20 in `autoresearch/prepare.ts:209`.
- CLI formatting: four files fail (`autoresearch/knobs.ts`, `loop.ts`, `run-once.ts`, `results/champion-soak.json`). Telegram CI formatting fails `config.ts` and `src/commands/status.ts`.

Tracked: **clever-cabin-ss4**. Restore checks without weakening production safety gates.

### P1: champion score is tuned research evidence, not untouched OOS proof

`services/neuratrade-cli-ts/autoresearch/prepare.ts:110` loads candles without an exchange filter, merges symbol aliases, and labels them Bybit futures. Top-symbol selection also omits venue filtering. Dataset identity is not bound to the persisted champion; a different panel after restart can leave incomparable old scores in place.

The confirm phase repeatedly evaluates the same panel while selecting knobs, with 30-day windows shifted one day and a maximum of 40 start positions. Those windows overlap and are reused for optimization; no untouched final holdout appears before CLAIM. The current score was read from persisted results, not independently rescored in this audit. Venue contamination is a code-level exposure; its magnitude was not quantified against the full production database.

Keep CLAIM as a research milestone. Require venue/dataset provenance and a frozen final holdout before treating the result as independent OOS evidence. Existing issue extended: **clever-cabin-c2s**.

### P2: active CLI dependency audit is absent from CI

Frozen CLI `bun audit` reports **one high and two moderate** advisories for transitive `lodash@4.17.21` under `alchemy > @prisma/dev > @mrleebo/prisma-ast > chevrotain > @chevrotain/gast`. High advisory: [GHSA-r5fr-rjxr-66jc](https://github.com/advisories/GHSA-r5fr-rjxr-66jc). Exploit reachability was not established. `.github/workflows/validation.yml` audits Telegram dependencies only. Tracked: **clever-cabin-x0i**.

### P2: Cloudflare can keep obsolete survivors indefinitely

`services/neuratrade-cli-ts/src/cloudflare/worker.ts:111` writes the whitelist only when a scan has survivors. A valid zero-survivor scan leaves the old whitelist in KV; GET returns it without freshness metadata. Scheduled exceptions are logged and swallowed. Consumers cannot distinguish fresh eligibility from an old successful scan. Source-level finding; deployed Worker state was not authenticated or independently verified. Tracked: **clever-cabin-gzk**.

### P2: requested wake conditions have no verified notification path

The inspected server crontab, PM2 apps, and systemd timers contain no matching CLAIM/entry/disk notifier. Existing `scripts/demo-soak-monitor.sh` inspects the default home's `grid_paper_trades`, not isolated champion `ladder_paper_trades`, and treats deliberately stopped PM2 apps as unhealthy. It writes logs rather than proving delivered notifications.

Actual ladder entry logs use **OPENED**, not ENTER. Demo ETH logged OPENED at **2026-09-07 00:40:30 UTC / 07:40:30 Jakarta**. A literal ENTER search misses it. Tracked: **clever-cabin-8rs**. No notification integration was created or message sent during this audit.

## Verified runtime snapshot

| Claim | Current evidence |
| --- | --- |
| Four workers running | All four online; advancing trial logs; included in saved PM2 dump |
| Champion approximately 0.0673 | Persisted score 0.06727774123419354, median DD 8.5346363656, expectancy 0.000980831808567052 |
| Only expectancy prevents CLAIM | Incorrect: score also fails required 0.08; expectancy requires 0.001, inclusive |
| CLAIM exists | No current `claimed.json`; historical `claimed-v1.json` exists |
| Goal display describes champion | Not reliably: `loop.ts` writes the most recent discarded confirm result into `goals.md` too |
| Soaks only HOLD, no entries | Paper has zero closes and no open rungs; demo records one open ETH rung and zero closes |
| Demo is testnet | PM2 `BYBIT_USE_TESTNET=true`; isolated demo home; exchange position not independently queried |
| Candle sync stopped means broken | No: one-shot job is stopped between scheduled quarter-hour runs; recent success logs present |
| Reboot supervision | `pm2-root` and cron enabled; champion/search apps present in saved dump; actual reboot not exercised |
| Server disk below 85% | 76%, approximately 18 GB available; research ledger approximately 224 MB |
| Local disk below 85% | **False: 93%, approximately 32 GiB available** |
| Weekly archive configured | Sunday 03:30 Asia/Jakarta; next run September 13, 2026 (September 12, 20:30 UTC) |
| Archive proven | No archive execution log yet; R2 env exists with mode 0600; upload/restore not tested |
| Credential rotation completed | Unverified; existing **clever-cabin-ztm** remains open; no secrets printed, rotated, or used for authenticated provider calls |

The archive uploads before pruning, which is the correct ordering. It still needs an observed successful upload and restore, plus crash-safe worker coordination. Rotation must update the archive consumer and revoke the exposed credential; file presence does not prove credential validity.

The older cached-false kill-switch defect in **clever-cabin-vjs** has changed: current `isEngaged()` reads SQLite every time, and current switch tests pass. Its open tracker status must not be mistaken for proof that the old implementation remains deployed. This does not resolve the scheduled reset finding.

## Verification and scope

| Check | Result |
| --- | --- |
| CLI frozen install | Pass; restored missing local `ky` and `zod`; lockfile unchanged |
| CLI `bun run typecheck` | Pass after frozen install |
| CLI `bun test` | 1309 pass, 1 fail, 1310 tests across 110 files |
| CLI `bun run lint` | Fail: five errors plus warnings |
| CLI `bun run fmt:check` | Fail: four files |
| CLI `bun audit` | Fail: one high, two moderate advisories |
| Telegram frozen install | Pass |
| Telegram `bun run typecheck` | Pass |
| Telegram `bun test` | 186 pass, zero fail across 18 files |
| Lock failure reproduction | Confirmed callback replay and abandoned-lock failure using temporary files |
| Production reads | PM2 metadata, saved process inventory, cron/timers, logs, targeted read-only SQLite queries, public Bybit candles |

Full local verification output is retained in `/tmp/neuratrade-audit-*-20260907.log`. Initial CLI tests before dependency repair produced 45 failures and nine errors; those were installation noise and are superseded by the frozen-install result above.

This audit covers the active TS runtime, research/soak path, deployment checks, scheduling/storage, and selected security boundaries. It is not a line-by-line security certification of every historical component. No live orders, risk settings, workers, credentials, or production files were changed. A broad production SQLite aggregate was stopped when too expensive; targeted bounded reads supplied the reported evidence. Provider access-log review, credential revocation, authenticated exchange reconciliation, offsite restore, actual reboot/failure recovery, and independent final OOS evaluation remain unverified.

Eight new issues were linked to **clever-cabin-8rr**; existing data, holdout, and kill-switch issues were annotated. No issue was closed without the mandatory QA gate. TimesFM OOS scoping remains separate work and does not remedy these evidence defects.

Tracker workflow drift: the installed `bd` has no `sync` command. `bd dolt push` reports no configured remote. All eight new findings were verified in the tracked `.beads/issues.jsonl` export for publication through the repository's Git remote; this is not a separate Dolt replication guarantee.
