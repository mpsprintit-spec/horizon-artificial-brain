# Gate G performance confirmation

Complete: **True**. Passed locked observed comparisons: **False** (89/90).

Values are means of per-run measurements. Raw samples/runs and every failed comparison remain in summary.json and the original logs.

| Workload | Metric | Baseline | Candidate | Limit | Result |
|---|---|---:|---:|---:|---|
| BenchmarkGateGThought/nodes=1000/dense=false/journal=false | thought_p95_ms | 0.5742 | 0.51795 | 0.76033 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=false | B/op | 29017 | 20883.5 | 34563.9 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.63705 | 0.52075 | 0.832607 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | B/op | 30723 | 21933 | 36355.2 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=false | thought_p95_ms | 7.23 | 6.395 | 8.4145 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=false | B/op | 1.68359e+06 | 1.12863e+06 | 1.77186e+06 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=true | thought_p95_ms | 7.41 | 5.9095 | 8.6215 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=true | B/op | 1.68515e+06 | 1.12959e+06 | 1.77351e+06 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=false | thought_p95_ms | 1.0497 | 2.508 | 1.30716 | FAIL |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=false | B/op | 29032.5 | 20907.5 | 34580.1 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=true | thought_p95_ms | 1.5005 | 1.01055 | 1.82558 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=true | B/op | 30747.5 | 21953 | 36380.9 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=false | thought_p95_ms | 25.215 | 21.77 | 29.0972 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=false | B/op | 6.74505e+06 | 4.52558e+06 | 7.0864e+06 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=true | thought_p95_ms | 25.32 | 21.96 | 29.218 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=true | B/op | 6.74614e+06 | 4.52667e+06 | 7.08755e+06 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=false | thought_p95_ms | 19.985 | 18.785 | 23.0828 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=false | B/op | 29038.5 | 20926.5 | 34586.4 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=true | thought_p95_ms | 19.585 | 19.27 | 22.6227 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=true | B/op | 30763 | 21911.5 | 36397.2 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=false | thought_p95_ms | 319.55 | 231.4 | 367.582 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=false | B/op | 5.55556e+07 | 3.63499e+07 | 5.83375e+07 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=true | thought_p95_ms | 307.75 | 256.25 | 354.012 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=true | B/op | 5.55572e+07 | 3.63512e+07 | 5.83392e+07 | PASS |
| BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | thought_p95_ms | 23.02 | 23.45 | 26.573 | PASS |
| BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | B/op | 561716 | 358858 | 593898 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | operation_p95_ms | 0.10207 | 0.09788 | 0.222484 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | learning_p95_ms | 0.03708 | 0.058185 | 0.144496 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | B/op | 91997 | 83824.5 | 105293 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | operation_p95_ms | 4.9825 | 4.921 | 6.079 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | learning_p95_ms | 3.37 | 1.4305 | 4.144 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | B/op | 3.56069e+06 | 3.42284e+06 | 3.92085e+06 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | operation_p95_ms | 0.128845 | 0.13328 | 0.254614 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | learning_p95_ms | 0.045405 | 0.07778 | 0.154486 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | B/op | 91909.5 | 83828.5 | 105196 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | operation_p95_ms | 22.455 | 19.975 | 27.046 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | learning_p95_ms | 7.175 | 5.196 | 8.71 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | B/op | 1.42161e+07 | 1.36768e+07 | 1.56418e+07 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | operation_p95_ms | 0.19175 | 0.067815 | 0.3301 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | learning_p95_ms | 0.2304 | 0.06655 | 0.37648 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | B/op | 91933.5 | 83927 | 105223 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | operation_p95_ms | 261.15 | 266.9 | 313.48 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | learning_p95_ms | 90.08 | 94.685 | 108.196 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | B/op | 1.25628e+08 | 1.20343e+08 | 1.38195e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | ns/op | 1.4681e+07 | 1.36212e+07 | 1.77172e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | B/op | 4.36733e+06 | 2.28634e+06 | 4.80816e+06 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | ns/op | 2.05609e+08 | 1.99786e+08 | 2.46831e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | B/op | 2.65413e+07 | 2.62035e+07 | 2.91995e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | ns/op | 2.44563e+08 | 2.30854e+08 | 2.93576e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | B/op | 3.01723e+07 | 2.98379e+07 | 3.31936e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | ns/op | 5.507e+07 | 4.98363e+07 | 6.6184e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | B/op | 1.63198e+07 | 8.1003e+06 | 1.79559e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | ns/op | 7.66865e+08 | 7.73136e+08 | 9.20338e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | B/op | 1.33934e+08 | 1.33933e+08 | 1.47331e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | ns/op | 8.67391e+08 | 8.86998e+08 | 1.04097e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | B/op | 1.42514e+08 | 1.50499e+08 | 1.5677e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | ns/op | 6.17439e+08 | 5.37934e+08 | 7.41027e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | B/op | 1.59708e+08 | 7.78014e+07 | 1.75683e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | ns/op | 7.6325e+09 | 7.59528e+09 | 9.1591e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | B/op | 1.26226e+09 | 1.26226e+09 | 1.38849e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | ns/op | 8.73169e+09 | 8.51316e+09 | 1.04781e+10 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | B/op | 1.39579e+09 | 1.39579e+09 | 1.53537e+09 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | thought_p95_ms | 0.5742 | 0.5228 | 0.76033 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | B/op | 29017 | 20896 | 34563.9 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.63705 | 0.7592 | 0.832607 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | B/op | 30723 | 21921 | 36355.2 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | thought_p95_ms | 7.23 | 7.614 | 8.4145 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | B/op | 1.68359e+06 | 1.12864e+06 | 1.77186e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | thought_p95_ms | 7.41 | 5.723 | 8.6215 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | B/op | 1.68515e+06 | 1.12956e+06 | 1.77351e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | thought_p95_ms | 1.0497 | 1.226 | 1.30716 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | B/op | 29032.5 | 20858 | 34580.1 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | thought_p95_ms | 1.5005 | 1.086 | 1.82558 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | B/op | 30747.5 | 21946 | 36380.9 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | thought_p95_ms | 25.215 | 21.49 | 29.0972 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | B/op | 6.74505e+06 | 4.5259e+06 | 7.0864e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | thought_p95_ms | 25.32 | 21.39 | 29.218 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | B/op | 6.74614e+06 | 4.52668e+06 | 7.08755e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | thought_p95_ms | 19.985 | 18.11 | 23.0828 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | B/op | 29038.5 | 20872 | 34586.4 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | thought_p95_ms | 19.585 | 20.54 | 22.6227 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | B/op | 30763 | 21926 | 36397.2 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | thought_p95_ms | 319.55 | 263.8 | 367.582 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | B/op | 5.55556e+07 | 3.63496e+07 | 5.83375e+07 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | thought_p95_ms | 307.75 | 240.9 | 354.012 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | B/op | 5.55572e+07 | 3.63509e+07 | 5.83392e+07 | PASS |
| BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | thought_p95_ms | 23.02 | 18.21 | 26.573 | PASS |
| BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | B/op | 561716 | 358996 | 593898 | PASS |
| BenchmarkGateGAdmission | admission_p95_ms | nan | 4e-05 | 0.01 | PASS |
| BenchmarkGateGAdmission | B/op | nan | 0 | 512 | PASS |

## Descriptive resources

| Revision | Workload | p50 ms | p95 ms | p99 ms | B/op | allocs/op | Peak RSS MB | GC count/cell | GC pause ms/cell |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| baseline | BenchmarkGateGThought/nodes=1000/dense=false/journal=false | 0.28335 | 0.5742 | 0.7745 | 29017 | 335 | 23.845 | 2 | 0.042355 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=false/journal=true | 0.2968 | 0.63705 | 0.83335 | 30723 | 344.5 | 23.845 | 2 | 0.0423 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=true/journal=false | 5.0365 | 7.23 | 10.003 | 1.68359e+06 | 10529.5 | 25.66 | 54.5 | 1.66 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=true/journal=true | 5.0285 | 7.41 | 9.4105 | 1.68515e+06 | 10538.5 | 25.66 | 55.5 | 1.0122 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=false/journal=false | 0.8425 | 1.0497 | 1.51 | 29032.5 | 335 | 49.875 | 0.5 | 0.00927 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=false/journal=true | 0.9259 | 1.5005 | 2.259 | 30747.5 | 345 | 49.875 | 0.5 | 0.01044 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=true/journal=false | 21.065 | 25.215 | 50.905 | 6.74505e+06 | 41031.5 | 50.275 | 112 | 3.386 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=true/journal=true | 20.665 | 25.32 | 38.19 | 6.74614e+06 | 41036.5 | 51.18 | 117.5 | 2.911 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=false/journal=false | 16.71 | 19.985 | 23.5 | 29038.5 | 335 | 331.55 | 0.5 | 0.01489 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=false/journal=true | 16.655 | 19.585 | 31.295 | 30763 | 345 | 331.55 | 0 | 0 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=true/journal=false | 266.25 | 319.55 | 346.85 | 5.55556e+07 | 405569 | 345.75 | 139.5 | 4.6025 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=true/journal=true | 265.6 | 307.75 | 333.35 | 5.55572e+07 | 405578 | 345.9 | 144.5 | 6.9235 |
| baseline | BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | 18.495 | 23.02 | 44.425 | 561716 | 4529 | 317.3 | 2 | 0.064125 |
| baseline | BenchmarkGateGNativeStages/nodes=1000/dense=false | 0.26805 | 0.3846 | 0.41765 | 91997 | 1209.5 | 24.22 | 0.5 | 0.00712 |
| baseline | BenchmarkGateGNativeStages/nodes=1000/dense=true | 17.06 | 23.225 | 70.53 | 3.56069e+06 | 67127 | 24.22 | 17 | 1.54135 |
| baseline | BenchmarkGateGNativeStages/nodes=4000/dense=false | 0.3675 | 0.71655 | 1.46005 | 91909.5 | 1206.5 | 47.07 | 0 | 0 |
| baseline | BenchmarkGateGNativeStages/nodes=4000/dense=true | 74.215 | 79.89 | 213.95 | 1.42161e+07 | 271704 | 50.095 | 28.5 | 2.7908 |
| baseline | BenchmarkGateGNativeStages/nodes=40000/dense=false | 0.47335 | 1.4055 | 7.958 | 91933.5 | 1205 | 333.3 | 0 | 0 |
| baseline | BenchmarkGateGNativeStages/nodes=40000/dense=true | 881.4 | 982.8 | 1019.05 | 1.25628e+08 | 2.72477e+06 | 333.3 | 32.5 | 0.92975 |
| baseline | BenchmarkGateGJournalAppend | 0.0034655 | 0.0064195 | 0.0136665 | 1811 | 11 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | — | — | — | 4.36733e+06 | 32500 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | — | — | — | 2.65413e+07 | 53176.5 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | — | — | — | 3.01723e+07 | 58561.5 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | — | — | — | 1.63198e+07 | 122503 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | — | — | — | 1.33934e+08 | 200710 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | — | — | — | 1.42514e+08 | 206064 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | — | — | — | 1.59708e+08 | 1.20251e+06 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | — | — | — | 1.26226e+09 | 1.96952e+06 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | — | — | — | 1.39579e+09 | 1.97497e+06 | — | — | — |
| candidate | BenchmarkGateGThought/nodes=1000/dense=false/journal=false | 0.25625 | 0.51795 | 0.70975 | 20883.5 | 142.5 | 24.58 | 2 | 0.03764 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=false/journal=true | 0.2635 | 0.52075 | 0.61865 | 21933 | 149 | 24.58 | 2 | 0.04312 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=true/journal=false | 3.9295 | 6.395 | 13.017 | 1.12863e+06 | 256 | 27.075 | 38.5 | 1.6655 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=true/journal=true | 3.798 | 5.9095 | 7.006 | 1.12959e+06 | 261.5 | 27.55 | 41 | 0.7529 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=false/journal=false | 0.81615 | 2.508 | 10.455 | 20907.5 | 143.5 | 48.775 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=false/journal=true | 0.8307 | 1.01055 | 1.136 | 21953 | 149.5 | 49.4 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=true/journal=false | 16.235 | 21.77 | 67.72 | 4.52558e+06 | 365 | 49.795 | 90 | 3.2855 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=true/journal=true | 16.19 | 21.96 | 32.73 | 4.52667e+06 | 371 | 51.69 | 91.5 | 3.8665 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=false/journal=false | 16.44 | 18.785 | 22.045 | 20926.5 | 144 | 333.85 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=false/journal=true | 16.995 | 19.27 | 22.16 | 21911.5 | 149 | 333.85 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=true/journal=false | 187.7 | 231.4 | 258.85 | 3.63499e+07 | 1018 | 336.7 | 99.5 | 3.5495 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=true/journal=true | 201.6 | 256.25 | 272.05 | 3.63512e+07 | 1026 | 355.6 | 100 | 3.1345 |
| candidate | BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | 19.16 | 23.45 | 36.985 | 358858 | 263 | 303.15 | 1 | 0.0279 |
| candidate | BenchmarkGateGNativeStages/nodes=1000/dense=false | 0.315 | 0.53355 | 0.6567 | 83824.5 | 1136.5 | 24.26 | 0.5 | 0.0113 |
| candidate | BenchmarkGateGNativeStages/nodes=1000/dense=true | 16.965 | 18.51 | 20.24 | 3.42284e+06 | 62190.5 | 24.26 | 16 | 0.2598 |
| candidate | BenchmarkGateGNativeStages/nodes=4000/dense=false | 0.3538 | 0.4559 | 0.54855 | 83828.5 | 1137 | 48.065 | 0 | 0 |
| candidate | BenchmarkGateGNativeStages/nodes=4000/dense=true | 68.685 | 75.86 | 92.73 | 1.36768e+07 | 251762 | 48.245 | 30 | 0.74845 |
| candidate | BenchmarkGateGNativeStages/nodes=40000/dense=false | 0.2701 | 0.4454 | 0.5453 | 83927 | 1134.5 | 331.2 | 0 | 0 |
| candidate | BenchmarkGateGNativeStages/nodes=40000/dense=true | 869.05 | 973.85 | 1126.2 | 1.20343e+08 | 2.52484e+06 | 331.95 | 30 | 0.88465 |
| candidate | BenchmarkGateGJournalAppend | 0.003581 | 0.007291 | 0.0197555 | 1811 | 11 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | — | — | — | 2.28634e+06 | 27405 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | — | — | — | 2.62035e+07 | 53180 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | — | — | — | 2.98379e+07 | 58702.5 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | — | — | — | 8.1003e+06 | 102406 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | — | — | — | 1.33933e+08 | 200692 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | — | — | — | 1.50499e+08 | 206206 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | — | — | — | 7.78014e+07 | 1.00241e+06 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | — | — | — | 1.26226e+09 | 1.96954e+06 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | — | — | — | 1.39579e+09 | 1.97508e+06 | — | — | — |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | 0.2573 | 0.5228 | 1.209 | 20896 | 143 | 24.21 | 2 | 0.04152 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | 0.2939 | 0.7592 | 1.437 | 21921 | 149 | 24.21 | 2 | 0.04613 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | 3.959 | 7.614 | 22.6 | 1.12864e+06 | 256 | 25.31 | 40 | 1.504 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | 3.544 | 5.723 | 6.16 | 1.12956e+06 | 261 | 25.31 | 42 | 0.6837 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | 0.8606 | 1.226 | 2.319 | 20858 | 143 | 45.53 | 0 | 0 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | 0.873 | 1.086 | 1.717 | 21946 | 149 | 48.82 | 1 | 0.02203 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | 15.68 | 21.49 | 25.39 | 4.5259e+06 | 367 | 50.12 | 90 | 2.602 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | 15.98 | 21.39 | 23.25 | 4.52668e+06 | 371 | 50.12 | 91 | 2.291 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | 16.25 | 18.11 | 37.75 | 20872 | 143 | 330.7 | 1 | 0.02112 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | 16.94 | 20.54 | 29.34 | 21926 | 149 | 330.7 | 1 | 0.01692 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | 199.9 | 263.8 | 291.3 | 3.63496e+07 | 1015 | 330.7 | 99 | 3.195 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | 177.5 | 240.9 | 254.1 | 3.63509e+07 | 1024 | 337.9 | 100 | 4.077 |
| candidate-only | BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | 16.63 | 18.21 | 29.23 | 358996 | 264 | 302.6 | 1 | 0.02377 |
| candidate-only | BenchmarkGateGAdmission | 3e-05 | 4e-05 | 5e-05 | 0 | 0 | — | — | — |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=1000 | 0.3306 | 0.6772 | 0.7214 | 77038 | 843 | 24.28 | 1 | 0.01546 |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=4000 | 0.9876 | 2.249 | 3.9 | 78321 | 843 | 48.27 | 2 | 0.05525 |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=40000 | 16.4 | 20.72 | 33.87 | 93337 | 844 | 333 | 0 | 0 |

## Interpretation limits

- Means of per-run percentiles are compared; observations are not pooled.
- Native stage tails use 20 pairs/run and are descriptive.
- Persistence uses three operations/run and reports means, not p95/p99.
- RSS is cumulative process peak; GC statistics also include untimed seeding.
- No baseline latency exists for mixed handoff: baseline rejects it.
- This statistical resource check alone cannot certify the other Gate G requirements.
