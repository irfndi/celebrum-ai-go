# TimesFM OOS results (research, non-commercial self-run)

Source runs: /tmp/timesfm-wf-{btc,sol,eth}-full.json (1191 origins each,
ctx 256 / horizon 12 / step 12, friction 0.16pc), gate-filter + 5-seed
stress files alongside. Research-only: never production (criterion 7).

## Walk-forward, frozen tail (criterion 1-3)

| Symbol | Model trades | Model net% | Model dirAcc% | Base trades | Base net% | Base dirAcc% | >=30 trades |
| --- | --- | --- | --- | --- | --- | --- | --- |
| BTC | 2 | +2.16 | 100 (2/2) | 345 | -72.01 | 46.4 | FAIL |
| SOL | 7 | +2.33 | 71.4 | 492 | -113.90 | 48.4 | FAIL |
| ETH | 3 | -3.51 | 33.3 | 419 | -76.72 | 49.2 | FAIL (also negative, PF 0.04) |

Model beats the baseline on net return everywhere, but the baseline is
terrible (300-500 friction-bled trades). Coverage is 0.17-0.59pc: the
model is so selective it cannot matter for growth (2-7 trades per
~5 months vs the hundreds needed).

## Stress + filter (criterion 4)

- BTC standAsideBand@3.00: gate4pass true, 5/5 seeds positive, but winner
  +7.88pc < baseline +8.89pc on clean data (filter cuts DD, gives up return).
- SOL standAsidePoint@1.00: gate4pass true, 4/5 seeds positive, winner
  +26.04pc < baseline +33.61pc. Same trade.

## Verdict: NO-GO as entry alpha

Fails gate 1 (>=30 OOS trades) on all three symbols; ETH also fails
gates 2-3 outright. Selectivity, not accuracy, is the binding constraint:
even BTC/SOL direction calls do not convert into trade counts that can
compound. Do not wire into entries. Revisit only with a looser operating
point re-run through the same 8 gates.
