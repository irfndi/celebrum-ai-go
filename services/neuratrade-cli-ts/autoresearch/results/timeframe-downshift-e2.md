# Growth E2 timeframe downshift (clever-cabin-umi) — results

ADDITIVE-ONLY probe. No live workers, DBs, champion files, or CLAIM gates touched.
Venue: `bybit-futures`, exact-symbol venue-filtered loads (aliases/venues never merged).
Costs (== `prepare.ts` honest schedule): maker 0.02%/side, taker-exit 0.06%/side, 2 bps slippage.
Knobs: frozen copy of `champion-soak.json` (rungs 2, step 1.3, grids 2, pause 2, stop 1.58, target 1.95, maxHold 39).
Symbols (same 8 for 15m + 5m): PUMPFUN, SOL, SUI, TAO, TUT, XAUT, XRP, ZEC (/USDT:USDT).
15m baseline is RE-MEASURED on the current panel (not the historic stored score) so the TF comparison is apples-to-apples.
Bar-count params apply as-coded in panel bars: maxHold 39 ≈ 9.75d at 15m but ≈ 3.25d at 5m.

## Selection (confirm-like, prefix before frozen tail)

| TF | src | score (medLogRet) | expectancy% | trades/sym-mo | win% | DD% | windows | guards |
|----|-----|-------------------|-------------|---------------|------|-----|---------|--------|
| 15m | 5m→15m | 0.0000 | -0.0015 | 126.34 | 49.5 | 7.93 | 80 | FAIL (logret≤0, wr<52, exp≤0) |
| 5m | 5m | -0.0249 | -0.0018 | 120.36 | 47.6 | 6.66 | 80 | FAIL (same) |
| 1m | 1m | n/a | n/a | n/a | n/a | n/a | 0 | NO DATA (venue has no 1m) |

## Frozen 30d holdout (trailing tail, one window/symbol, reported separately)

| TF | holdout bars | score | expectancy% | trades/sym-mo | win% | DD% | guards |
|----|--------------|-------|-------------|---------------|------|-----|--------|
| 15m | 2880 | -0.0565 | -0.0026 | 236.25 | 47.8 | 15.80 | FAIL (also DD>12) |
| 5m | 8640 | -0.0327 | -0.0021 | 326.50 | 48.6 | 13.46 | FAIL (also DD>12) |
| 1m | 43200 | n/a | n/a | n/a | n/a | n/a | NO DATA |

## Readout

- Downshift does NOT rescue the champion geometry: 5m selection scores worse than the 15m baseline (-0.0249 vs 0.0000), expectancy stays negative on both TFs, and both holdouts fail KEEP guards with DD above 12.
- 5m trades more on holdout (326 vs 236 tpsm) but at worse-or-equal expectancy — throughput without edge.
- 1m is unevaluable on this venue (zero 1m candles for bybit-futures); no cross-venue substitution was made.
- Verdict: do not promote a 5m/1m downshift; edge must come from geometry, not timeframe.

## Reproduce (readonly)

```bash
cd services/neuratrade-cli-ts
bun autoresearch/timeframe-downshift-e2.ts
bun test autoresearch/timeframe-downshift-e2.test.ts
```
