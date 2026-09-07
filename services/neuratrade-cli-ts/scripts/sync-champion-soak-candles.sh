#!/usr/bin/env bash
# Keep champion paper/demo candle caches fresh for 15m ladder soaks.
#
# market fetch-candles often saves 0 tip-ups (resume/gateway quirks) and may
# hit testnet when BYBIT_USE_TESTNET=true in the deployer env. Use the
# mainnet Bybit backfill into the MAIN cache, then incremental-copy into
# isolated soak homes.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MAIN_HOME="${HOME:-/root}/.neuratrade"
SYMBOLS="BTCUSDT,ETHUSDT,SOLUSDT,LINKUSDT"

echo "[1/2] tip-fill mainnet 15m into $MAIN_HOME (last ~3d)"
NEURATRADE_HOME="$MAIN_HOME" BYBIT_USE_TESTNET=false \
  bun run scripts/backfill-bybit-15m.ts --months=0.1 --symbols="$SYMBOLS"

echo "[2/2] incremental copy main → paper/demo soak DBs"
python3 scripts/seed-champion-soak-candles.py --incremental

echo "champion candle sync done $(date -u +%Y-%m-%dT%H:%M:%SZ)"
