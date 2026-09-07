# NeuraTrade Autoresearch — program.md

You are running an overnight research loop to improve **ladder grid** expectancy.
Live money is frozen. Do not touch kill-switch, credentials, or SQLite position rows.

## Goal version: v2 (claim when ALL true on a kept champion)

1. **Profitability:** median OOS window log-return ≥ **0.08** (v1 was > 0; v1 champ ≈ 0.062).
2. **Win rate:** overall win rate ≥ **55%** (v1 was 48%).
3. **Throughput:** ≥ 4 trades per symbol-month (secondary — never optimize this alone).
4. **Risk:** median window max drawdown ≤ **10%** (v1 was 15%).
5. **Expectancy:** ≥ **0.001** per trade.

Until claimed: keep looping. Prefer falsification over storytelling.

KEEP guards (softer than claim): WR ≥ 52%, DD ≤ 12%, log-ret > 0, expectancy > 0, ≥ 4 trades/sym-mo.

## Parallel with paper / testnet

- Soak knobs are frozen in `results/champion-soak.json` (promoted from the
  current search champion when you choose to validate live/paper behavior).
- This loop may overwrite `results/champion.json` as it climbs — that does **not**
  restart soaks until you re-promote + restart paper/demo.
- Backtest assumes infinite divisibility (no contractSpecs); live venue minimums can block small partitions (e.g. BTC $50/rung → 162%>100% guard HOLD). Accept no-fill; do not raise maxNotionalPct without re-validation.

## Rules

- Edit **only** `knobs.ts` (or accept the mutation loop writing it).
- Never edit `prepare.ts`, risk/, paper-trading/, or live ecosystem configs from this loop.
- Each trial has a **fixed wall-clock budget** (`--budget-sec`, default 180).
- Metric is computed by `prepare.ts` — trust the printed `score` / `guardsOk`.
- **KEEP** only if `guardsOk` and `score` strictly beats the current champion (see `shouldKeep`).
- **DISCARD** otherwise; revert knobs to champion.
- Log every trial to `results/ledger.jsonl` (the runner does this).

## How to run

```bash
cd services/neuratrade-cli-ts
make -C ../.. autoresearch-once
make -C ../.. autoresearch-loop          # 1 worker, panel cached, screen→confirm
make -C ../.. autoresearch-parallel      # 4 pm2 workers, shared champion lock
```

Each trial: cheap **screen** (~7d) → only promising knobs pay full **confirm** (~30d).
Candle panel loads once per process.

## Mutation hints (for agents)

Prefer one-knob changes. Useful axes: `gridStepPct`, `stopRatio`, `maxHoldBars`,
`targetRatio`, `rungs`, `gridMaxGrids`, `gridPauseAfterLossBars`, `chopGateAdxThreshold`.
Avoid leverage > 1 in this loop. Avoid fee/slippage edits (frozen in prepare: maker 0.02, taker-exit 0.06).
