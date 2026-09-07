# Growth E3 cohort expansion — bybit-futures 15m

Generated: 2026-09-07T05:51:11.033Z
Frozen champion: autoresearch/results/champion-soak.json (grid-relevant knobs + honestFees maker0.02/takerExit0.06)
Frozen grid: step=1.3 grids=2 pause=2 target=1.95 adx=0 fee=0.02 takerExit=0.06 slip=1bps posFrac=1
Holdout (frozen): last 20% of candles per symbol + 5-seed stress; no per-symbol fitting.
Venue filter: exchange=bybit-futures, timeframe=15m, >= 55000 bars.
Screened: 25 symbols | valid: 24 | PASS: 0

| Symbol | Bars | WinWin% | HistRet% | MaxDD% | OOS n | Expect%/tr | Trades/mo | OOS win% | OOS ret% | ConfLB | StressWorst | StressLB | Gate |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| ADA/USDT:USDT | 69120 | 46.2 | 21.52 | 42.71 | 131 | -0.14497 | 27.3 | 48.9 | -20.83 | -0.00461 | -24.43 | -0.00294 | FAIL(windows<50%,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| APT/USDT:USDT | 69120 | 46.2 | 21.31 | 28.26 | 184 | -0.00640 | 38.3 | 51.6 | -7.02 | -0.00338 | -28.73 | -0.00178 | FAIL(windows<50%,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| ARB/USDT:USDT | 69120 | 38.5 | -40.54 | 65.65 | 192 | -0.22636 | 40.0 | 47.4 | -39.28 | -0.00575 | -42.38 | -0.00394 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| ATOM/USDT:USDT | 69120 | 53.8 | -52.71 | 75.31 | 91 | 0.11434 | 19.0 | 53.8 | 7.66 | -0.00392 | -13.86 | -0.00168 | FAIL(compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| AVAX/USDT:USDT | 69120 | 15.4 | -84.98 | 84.98 | 109 | -0.29934 | 22.7 | 45.9 | -30.42 | -0.00778 | -49.60 | -0.00659 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| BCH/USDT:USDT | 69120 | 30.8 | -71.72 | 71.72 | 114 | -0.09080 | 23.8 | 50.0 | -13.18 | -0.00583 | -16.57 | -0.00274 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| BNB/USDT:USDT | 69120 | 53.8 | -18.24 | 36.38 | 31 | 0.05153 | 6.5 | 51.6 | 0.60 | -0.00823 | -10.83 | -0.00768 | FAIL(compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| BTC/USDT:USDT | 69120 | 38.5 | -36.24 | 45.09 | 30 | -0.44980 | 6.3 | 43.3 | -13.50 | -0.01645 | -18.75 | -0.01046 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| DOGE/USDT:USDT | 69120 | 46.2 | -61.49 | 70.26 | 83 | 0.16508 | 17.3 | 55.4 | 11.61 | -0.00380 | 4.61 | 0.00109 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0) |
| DOT/USDT:USDT | 69120 | 38.5 | -4.97 | 52.49 | 108 | 0.36831 | 22.5 | 59.3 | 43.71 | -0.00082 | 4.55 | 0.00004 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0) |
| ETC/USDT:USDT | 69120 | 23.1 | -63.15 | 76.33 | 81 | 0.55696 | 16.9 | 63.0 | 52.95 | 0.00127 | 34.64 | 0.00357 | FAIL(windows<50%,compounded<=0,dd>15%) |
| ETH/USDT:USDT | 69120 | 38.5 | -54.61 | 63.21 | 75 | 0.06889 | 15.6 | 53.3 | 2.73 | -0.00689 | -12.57 | -0.00164 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| FIL/USDT:USDT | 69120 | 30.8 | -21.47 | 52.31 | 206 | 0.26986 | 42.9 | 56.8 | 63.00 | -0.00109 | -0.31 | 0.00069 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0) |
| HBAR/USDT:USDT | 69120 | 38.5 | -74.21 | 87.09 | 72 | 0.47759 | 15.0 | 61.1 | 37.77 | -0.00098 | 16.26 | 0.00161 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0) |
| INJ/USDT:USDT | 69120 | 23.1 | -82.34 | 85.39 | 298 | 0.04447 | 62.1 | 52.7 | 3.42 | -0.00268 | -50.30 | -0.00220 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| LINK/USDT:USDT | 69120 | 38.5 | -54.70 | 65.01 | 86 | 0.06580 | 17.9 | 53.5 | 2.87 | -0.00461 | -15.75 | -0.00224 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| LTC/USDT:USDT | 69120 | 53.8 | 22.75 | 41.72 | 41 | 0.35985 | 8.5 | 58.5 | 14.36 | -0.00274 | 6.19 | 0.00164 | FAIL(dd>15%,confLB<0) |
| NEAR/USDT:USDT | 69120 | 30.8 | -69.61 | 81.93 | 327 | 0.19060 | 68.1 | 55.4 | 67.45 | -0.00118 | -55.57 | -0.00182 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| OP/USDT:USDT | 69120 | 30.8 | -86.47 | 86.47 | 217 | -0.10206 | 45.2 | 49.8 | -25.47 | -0.00438 | -49.64 | -0.00299 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| RENDER/USDT:USDT | 69120 | 38.5 | -92.71 | 93.90 | 208 | -0.09177 | 43.3 | 50.0 | -22.92 | -0.00464 | -34.24 | -0.00106 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| SOL/USDT:USDT | 69120 | 38.5 | -39.52 | 67.59 | 90 | 0.25582 | 18.8 | 56.7 | 22.21 | -0.00438 | -9.77 | -0.00272 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| SUI/USDT:USDT | 69120 | 46.2 | -22.09 | 49.21 | 145 | 0.13843 | 30.2 | 54.5 | 16.51 | -0.00279 | -29.34 | -0.00226 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| TRX/USDT:USDT | 69120 | — | — | — | — | — | — | — | — | — | — | — | invalid: fixed OOS trade count is below 30 |
| WIF/USDT:USDT | 69120 | 30.8 | -84.77 | 89.15 | 233 | 0.18680 | 48.5 | 55.4 | 43.07 | -0.00215 | -16.36 | -0.00084 | FAIL(windows<50%,compounded<=0,dd>15%,confLB<0,stressRet<0,stressLB<0) |
| XRP/USDT:USDT | 69120 | 46.2 | -22.02 | 48.20 | 54 | 0.75284 | 11.3 | 66.7 | 47.57 | 0.00095 | 37.28 | 0.00466 | FAIL(windows<50%,compounded<=0,dd>15%) |

## Union portfolio growth estimate (equal-weight mean of PASS walk-forward HistRet%)

- PASS members (0): none
- Union mean return: 0.00%
- Baseline (BTC/USDT:USDT, ETH/USDT:USDT): -45.43%
- Uplift vs baseline: 45.43pp

Method: capital split equally across member symbols; portfolio return ~= mean of per-symbol compounded walk-forward returns. Throughput/expectancy are per-symbol (expectancy is scale-invariant; HistRet scales with positionFraction=1 frozen).
