# Growth E4 per-symbol grid refit — bybit-futures 15m

Generated: 2026-09-08T09:34:05.443Z
Task: clever-cabin-6a8 (follow-up to E3 clever-cabin-lnj, 0/24 PASS).
Refit symbols: LTC/USDT:USDT, XRP/USDT:USDT, ETC/USDT:USDT (E3 closest-but-failing).
Selection (no tail peek): first 80% of candles per symbol, walk-forward train=11520 test=4320; 162 structural candidates per symbol, fees frozen (maker 0.02 / takerExit 0.06 / slip 1bps / lev 1 / posFrac 1).
Final eval (frozen, SAME as E3): validateGridEvidence last-20% tail + 5-seed stress + passesCohortGate (imported from scripts/growth-e3-cohort-expansion.ts).
Refit: 3 symbols | valid: 0 | PASS: 0

| Symbol | Fitted knobs | SelWin% | SelRet% | SelDD% | Sel | WinWin% | HistRet% | MaxDD% | OOS n | Expect%/tr | Trades/mo | OOS win% | OOS ret% | ConfLB | StressWorst | StressLB | Gate | Frozen | Δpp |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| LTC/USDT:USDT | step=1.3 grids=2 pause=4 target=1.95 adx=20 | 80.0 | 77.19 | 6.48 | SEL-PASS | — | — | — | — | — | — | — | — | — | — | — | — | invalid: fitted: fixed OOS trade count is below 30 | FAIL(dd>15%,confLB<0) | — |
| XRP/USDT:USDT | step=1.7 grids=2 pause=0 target=1.5 adx=0 | 90.0 | 165.28 | 1.56 | SEL-PASS | — | — | — | — | — | — | — | — | — | — | — | — | invalid: fitted: fixed OOS trade count is below 30 | FAIL(windows<50%,compounded<=0,dd>15%) | — |
| ETC/USDT:USDT | step=1.7 grids=2 pause=0 target=1.95 adx=25 | 80.0 | 90.74 | 5.41 | SEL-PASS | — | — | — | — | — | — | — | — | — | — | — | — | invalid: fitted: fixed OOS trade count is below 30 | FAIL(windows<50%,compounded<=0,dd>15%) | — |

Notes: CLAIM bars (goals.ts 0.08/0.001) untouched — no promotion implied. Expectancy>=0.001 column (Expect%/tr) is informational vs the expectancy claim bar. DB opened read-only; only these two result files written.
