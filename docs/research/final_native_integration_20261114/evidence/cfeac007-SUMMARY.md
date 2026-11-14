# Gate G performance confirmation

Complete: **True**. Passed locked observed comparisons: **False** (86/90).

Values are means of per-run measurements. Raw samples/runs and every failed comparison remain in summary.json and the original logs.

| Workload | Metric | Baseline | Candidate | Limit | Result |
|---|---|---:|---:|---:|---|
| BenchmarkGateGThought/nodes=1000/dense=false/journal=false | thought_p95_ms | 0.5569 | 0.51845 | 0.740435 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=false | B/op | 29011 | 20885.5 | 34557.6 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.6324 | 1.79555 | 0.82726 | FAIL |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | B/op | 30788.5 | 21911 | 36423.9 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=false | thought_p95_ms | 7.6 | 5.5555 | 8.84 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=false | B/op | 1.68348e+06 | 1.12866e+06 | 1.77175e+06 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=true | thought_p95_ms | 7.8765 | 6.111 | 9.15797 | PASS |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=true | B/op | 1.68537e+06 | 1.12959e+06 | 1.77374e+06 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=false | thought_p95_ms | 1.097 | 1.051 | 1.36155 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=false | B/op | 28978 | 20897.5 | 34522.9 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=true | thought_p95_ms | 1.4115 | 0.99215 | 1.72323 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=false/journal=true | B/op | 30735 | 21926.5 | 36367.8 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=false | thought_p95_ms | 29.055 | 23.18 | 33.5132 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=false | B/op | 6.74416e+06 | 4.52561e+06 | 7.08546e+06 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=true | thought_p95_ms | 27.65 | 23.53 | 31.8975 | PASS |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=true | B/op | 6.74615e+06 | 4.5266e+06 | 7.08755e+06 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=false | thought_p95_ms | 21.94 | 17.56 | 25.331 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=false | B/op | 28985.5 | 20942 | 34530.8 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=true | thought_p95_ms | 22.27 | 18.435 | 25.7105 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=false/journal=true | B/op | 30709.5 | 21964 | 36341 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=false | thought_p95_ms | 310.55 | 243.85 | 357.233 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=false | B/op | 5.55561e+07 | 3.63508e+07 | 5.8338e+07 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=true | thought_p95_ms | 302.75 | 233.55 | 348.262 | PASS |
| BenchmarkGateGThought/nodes=40000/dense=true/journal=true | B/op | 5.55582e+07 | 3.63517e+07 | 5.83402e+07 | PASS |
| BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | thought_p95_ms | 27.825 | 20.805 | 32.0988 | PASS |
| BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | B/op | 561997 | 358837 | 594193 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | operation_p95_ms | 0.09451 | 0.09271 | 0.213412 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | learning_p95_ms | 0.05769 | 0.187925 | 0.169228 | FAIL |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | B/op | 91872 | 83794.5 | 105155 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | operation_p95_ms | 5.0675 | 4.79 | 6.181 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | learning_p95_ms | 1.4985 | 1.5915 | 1.8982 | PASS |
| BenchmarkGateGNativeStages/nodes=1000/dense=true | B/op | 3.56048e+06 | 3.42266e+06 | 3.92062e+06 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | operation_p95_ms | 0.078615 | 0.06158 | 0.194338 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | learning_p95_ms | 0.037255 | 0.042545 | 0.144706 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | B/op | 91930 | 83895 | 105219 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | operation_p95_ms | 20.495 | 23.23 | 24.694 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | learning_p95_ms | 6.085 | 6.621 | 7.402 | PASS |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | B/op | 1.42156e+07 | 1.36775e+07 | 1.56412e+07 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | operation_p95_ms | 0.09227 | 0.10523 | 0.210724 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | learning_p95_ms | 0.05898 | 0.042725 | 0.170776 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=false | B/op | 92105.5 | 83998 | 105412 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | operation_p95_ms | 256.1 | 281.15 | 307.42 | PASS |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | learning_p95_ms | 78.64 | 100.79 | 94.468 | FAIL |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | B/op | 1.25629e+08 | 1.20343e+08 | 1.38196e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | ns/op | 3.73565e+07 | 1.36069e+07 | 4.49278e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | B/op | 4.36741e+06 | 2.28655e+06 | 4.80825e+06 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | ns/op | 2.12383e+08 | 1.93283e+08 | 2.5496e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | B/op | 2.62049e+07 | 2.62044e+07 | 2.88295e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | ns/op | 2.22103e+08 | 2.56857e+08 | 2.66624e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | B/op | 3.08497e+07 | 3.05157e+07 | 3.39387e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | ns/op | 6.47553e+07 | 4.99844e+07 | 7.78064e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | B/op | 1.63202e+07 | 8.10065e+06 | 1.79563e+07 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | ns/op | 7.8315e+08 | 7.5879e+08 | 9.39879e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | B/op | 1.37921e+08 | 1.37924e+08 | 1.51718e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | ns/op | 9.15976e+08 | 9.03834e+08 | 1.09927e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | B/op | 1.42512e+08 | 1.46508e+08 | 1.56768e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | ns/op | 6.32715e+08 | 5.81496e+08 | 7.59358e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | B/op | 1.59707e+08 | 7.78015e+07 | 1.75681e+08 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | ns/op | 7.65963e+09 | 7.45249e+09 | 9.19165e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | B/op | 1.26225e+09 | 1.26226e+09 | 1.38848e+09 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | ns/op | 8.83692e+09 | 8.69341e+09 | 1.06044e+10 | PASS |
| BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | B/op | 1.39579e+09 | 1.42238e+09 | 1.53537e+09 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | thought_p95_ms | 0.5569 | 0.3663 | 0.740435 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | B/op | 29011 | 20896 | 34557.6 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.6324 | 0.7666 | 0.82726 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | B/op | 30788.5 | 21962 | 36423.9 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | thought_p95_ms | 7.6 | 5.722 | 8.84 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | B/op | 1.68348e+06 | 1.12857e+06 | 1.77175e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | thought_p95_ms | 7.8765 | 5.927 | 9.15797 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | B/op | 1.68537e+06 | 1.12969e+06 | 1.77374e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | thought_p95_ms | 1.097 | 1.029 | 1.36155 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | B/op | 28978 | 20940 | 34522.9 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | thought_p95_ms | 1.4115 | 2.013 | 1.72323 | FAIL |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | B/op | 30735 | 21917 | 36367.8 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | thought_p95_ms | 29.055 | 21.51 | 33.5132 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | B/op | 6.74416e+06 | 4.52566e+06 | 7.08546e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | thought_p95_ms | 27.65 | 21.68 | 31.8975 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | B/op | 6.74615e+06 | 4.52653e+06 | 7.08755e+06 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | thought_p95_ms | 21.94 | 16.86 | 25.331 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | B/op | 28985.5 | 20900 | 34530.8 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | thought_p95_ms | 22.27 | 17.16 | 25.7105 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | B/op | 30709.5 | 21993 | 36341 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | thought_p95_ms | 310.55 | 233.2 | 357.233 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | B/op | 5.55561e+07 | 3.63506e+07 | 5.8338e+07 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | thought_p95_ms | 302.75 | 242.7 | 348.262 | PASS |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | B/op | 5.55582e+07 | 3.63516e+07 | 5.83402e+07 | PASS |
| BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | thought_p95_ms | 27.825 | 21.34 | 32.0988 | PASS |
| BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | B/op | 561997 | 359004 | 594193 | PASS |
| BenchmarkGateGAdmission | admission_p95_ms | nan | 4e-05 | 0.01 | PASS |
| BenchmarkGateGAdmission | B/op | nan | 0 | 512 | PASS |

## Descriptive resources

| Revision | Workload | p50 ms | p95 ms | p99 ms | B/op | allocs/op | Peak RSS MB | GC count/cell | GC pause ms/cell |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| baseline | BenchmarkGateGThought/nodes=1000/dense=false/journal=false | 0.2838 | 0.5569 | 0.6713 | 29011 | 334.5 | 24.06 | 2 | 0.05671 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=false/journal=true | 0.2974 | 0.6324 | 1.0838 | 30788.5 | 345.5 | 24.06 | 2 | 0.106855 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=true/journal=false | 4.975 | 7.6 | 11.2375 | 1.68348e+06 | 10529 | 24.905 | 60.5 | 1.8045 |
| baseline | BenchmarkGateGThought/nodes=1000/dense=true/journal=true | 5.183 | 7.8765 | 12.055 | 1.68537e+06 | 10540 | 24.905 | 70 | 2.1855 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=false/journal=false | 0.85505 | 1.097 | 2.752 | 28978 | 334 | 49.25 | 0.5 | 0.01047 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=false/journal=true | 0.8763 | 1.4115 | 1.629 | 30735 | 344.5 | 49.53 | 0.5 | 0.012575 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=true/journal=false | 21.505 | 29.055 | 114.805 | 6.74416e+06 | 41025 | 49.53 | 114.5 | 4.7405 |
| baseline | BenchmarkGateGThought/nodes=4000/dense=true/journal=true | 21.275 | 27.65 | 57.115 | 6.74615e+06 | 41036.5 | 50.16 | 119 | 3.881 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=false/journal=false | 17.075 | 21.94 | 58.845 | 28985.5 | 334.5 | 333.6 | 0.5 | 0.011085 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=false/journal=true | 17.165 | 22.27 | 27.745 | 30709.5 | 344 | 333.6 | 0 | 0 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=true/journal=false | 257.2 | 310.55 | 340.8 | 5.55561e+07 | 405572 | 337.25 | 141.5 | 5.204 |
| baseline | BenchmarkGateGThought/nodes=40000/dense=true/journal=true | 257.8 | 302.75 | 318.5 | 5.55582e+07 | 405585 | 338.15 | 145 | 5.5735 |
| baseline | BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | 18.76 | 27.825 | 51.95 | 561997 | 4530.5 | 305.15 | 2 | 0.077475 |
| baseline | BenchmarkGateGNativeStages/nodes=1000/dense=false | 0.2888 | 0.46985 | 0.5924 | 91872 | 1205 | 24.895 | 1 | 0.01558 |
| baseline | BenchmarkGateGNativeStages/nodes=1000/dense=true | 17.065 | 18.59 | 20.34 | 3.56048e+06 | 67122 | 24.895 | 16.5 | 0.2321 |
| baseline | BenchmarkGateGNativeStages/nodes=4000/dense=false | 0.2449 | 0.36265 | 0.4345 | 91930 | 1207.5 | 47.575 | 0 | 0 |
| baseline | BenchmarkGateGNativeStages/nodes=4000/dense=true | 69.895 | 76.725 | 148.395 | 1.42156e+07 | 271700 | 49.645 | 30 | 0.7142 |
| baseline | BenchmarkGateGNativeStages/nodes=40000/dense=false | 0.2666 | 0.39755 | 0.53165 | 92105.5 | 1206.5 | 331.2 | 0 | 0 |
| baseline | BenchmarkGateGNativeStages/nodes=40000/dense=true | 882.2 | 988.95 | 1063.25 | 1.25629e+08 | 2.72477e+06 | 331.2 | 32.5 | 1.0849 |
| baseline | BenchmarkGateGJournalAppend | 0.0035105 | 0.0093795 | 0.28692 | 1811 | 11 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | — | — | — | 4.36741e+06 | 32501.5 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | — | — | — | 2.62049e+07 | 53189.5 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | — | — | — | 3.08497e+07 | 58560.5 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | — | — | — | 1.63202e+07 | 122508 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | — | — | — | 1.37921e+08 | 200688 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | — | — | — | 1.42512e+08 | 206042 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | — | — | — | 1.59707e+08 | 1.2025e+06 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | — | — | — | 1.26225e+09 | 1.9695e+06 | — | — | — |
| baseline | BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | — | — | — | 1.39579e+09 | 1.97496e+06 | — | — | — |
| candidate | BenchmarkGateGThought/nodes=1000/dense=false/journal=false | 0.24875 | 0.51845 | 0.8207 | 20885.5 | 143 | 24.62 | 2 | 0.044025 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=false/journal=true | 0.3698 | 1.79555 | 17.1846 | 21911 | 149 | 24.62 | 2 | 0.76463 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=true/journal=false | 3.59 | 5.5555 | 6.317 | 1.12866e+06 | 256 | 24.96 | 40 | 0.60715 |
| candidate | BenchmarkGateGThought/nodes=1000/dense=true/journal=true | 3.7735 | 6.111 | 10.706 | 1.12959e+06 | 262 | 27.025 | 40 | 0.903 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=false/journal=false | 0.80145 | 1.051 | 1.4715 | 20897.5 | 143 | 48.395 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=false/journal=true | 0.8269 | 0.99215 | 1.17465 | 21926.5 | 149.5 | 49.485 | 0 | 0 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=true/journal=false | 16.965 | 23.18 | 70.875 | 4.52561e+06 | 365 | 50.11 | 90.5 | 5.4325 |
| candidate | BenchmarkGateGThought/nodes=4000/dense=true/journal=true | 16.805 | 23.53 | 52.9 | 4.5266e+06 | 370.5 | 50.11 | 87.5 | 4.577 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=false/journal=false | 15.755 | 17.56 | 21.125 | 20942 | 143.5 | 333.7 | 0.5 | 0.011225 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=false/journal=true | 15.945 | 18.435 | 31.775 | 21964 | 150 | 333.7 | 1 | 0.0192 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=true/journal=false | 193.6 | 243.85 | 260.95 | 3.63508e+07 | 1024 | 352.15 | 100 | 4.6355 |
| candidate | BenchmarkGateGThought/nodes=40000/dense=true/journal=true | 194.05 | 233.55 | 258.65 | 3.63517e+07 | 1030 | 377.7 | 99.5 | 3.6985 |
| candidate | BenchmarkGateGContextContinuity/nodes=40000/dense=false/journal=true | 16.41 | 20.805 | 36.955 | 358837 | 263.5 | 299.2 | 1 | 0.0264 |
| candidate | BenchmarkGateGNativeStages/nodes=1000/dense=false | 0.29235 | 0.42935 | 0.5033 | 83794.5 | 1136.5 | 24.22 | 0 | 0 |
| candidate | BenchmarkGateGNativeStages/nodes=1000/dense=true | 16.605 | 18.475 | 19.1 | 3.42266e+06 | 62189.5 | 24.22 | 15.5 | 0.24285 |
| candidate | BenchmarkGateGNativeStages/nodes=4000/dense=false | 0.234 | 0.50335 | 0.75805 | 83895 | 1138 | 45.96 | 0 | 0 |
| candidate | BenchmarkGateGNativeStages/nodes=4000/dense=true | 70.285 | 88.69 | 161.15 | 1.36775e+07 | 251764 | 48.245 | 28 | 0.6906 |
| candidate | BenchmarkGateGNativeStages/nodes=40000/dense=false | 0.2692 | 0.36535 | 0.378 | 83998 | 1138 | 333.05 | 0 | 0 |
| candidate | BenchmarkGateGNativeStages/nodes=40000/dense=true | 894.5 | 1058 | 1104 | 1.20343e+08 | 2.52484e+06 | 333.05 | 30.5 | 0.93775 |
| candidate | BenchmarkGateGJournalAppend | 0.003435 | 0.006535 | 0.007987 | 1811 | 11 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=checkpoint | — | — | — | 2.28655e+06 | 27410 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=restore-pending | — | — | — | 2.62044e+07 | 53178.5 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=1000/mode=replay-pair | — | — | — | 3.05157e+07 | 58711.5 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | — | — | — | 8.10065e+06 | 102410 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=restore-pending | — | — | — | 1.37924e+08 | 200716 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=4000/mode=replay-pair | — | — | — | 1.46508e+08 | 206182 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=checkpoint | — | — | — | 7.78015e+07 | 1.00241e+06 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=restore-pending | — | — | — | 1.26226e+09 | 1.96957e+06 | — | — | — |
| candidate | BenchmarkNativeActionPersistence/nodes=40000/mode=replay-pair | — | — | — | 1.42238e+09 | 1.97509e+06 | — | — | — |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=false | 0.2464 | 0.3663 | 0.5 | 20896 | 143 | 23.97 | 1 | 0.014 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=false/journal=true | 0.2843 | 0.7666 | 1.488 | 21962 | 149 | 23.97 | 2 | 0.03863 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=false | 3.587 | 5.722 | 8.729 | 1.12857e+06 | 255 | 25.35 | 40 | 12.58 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=1000/dense=true/journal=true | 3.913 | 5.927 | 6.272 | 1.12969e+06 | 262 | 25.35 | 45 | 0.8096 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=false | 0.8388 | 1.029 | 1.233 | 20940 | 144 | 49.07 | 0 | 0 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | 0.8352 | 2.013 | 3.166 | 21917 | 149 | 49.89 | 1 | 0.02371 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=false | 15.48 | 21.51 | 75.71 | 4.52566e+06 | 365 | 52.57 | 85 | 2.487 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | 15.89 | 21.68 | 27.99 | 4.52653e+06 | 371 | 52.57 | 85 | 2.539 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | 15.34 | 16.86 | 18.04 | 20900 | 143 | 334.1 | 0 | 0 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=true | 15.36 | 17.16 | 19.1 | 21993 | 150 | 334.1 | 0 | 0 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=false | 177.8 | 233.2 | 276.7 | 3.63506e+07 | 1023 | 387.6 | 99 | 3.149 |
| candidate-only | BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=true/journal=true | 186.6 | 242.7 | 264.8 | 3.63516e+07 | 1029 | 387.6 | 100 | 7.361 |
| candidate-only | BenchmarkGateGContextContinuityNativeIdle/nodes=40000/dense=false/journal=true | 17.49 | 21.34 | 44.75 | 359004 | 265 | 315.6 | 1 | 0.02435 |
| candidate-only | BenchmarkGateGAdmission | 3e-05 | 4e-05 | 4.1e-05 | 0 | 0 | — | — | — |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=1000 | 0.3178 | 0.762 | 0.8853 | 77106 | 843 | 24.07 | 1 | 0.01338 |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=4000 | 1.101 | 2.419 | 3.522 | 78244 | 842 | 48.86 | 0 | 0 |
| candidate-only | BenchmarkGateGMixedHandoff/nodes=40000 | 15.46 | 16.71 | 17.69 | 93290 | 843 | 333.8 | 1 | 0.01881 |

## Interpretation limits

- Means of per-run percentiles are compared; observations are not pooled.
- Native stage tails use 20 pairs/run and are descriptive.
- Persistence uses three operations/run and reports means, not p95/p99.
- RSS is cumulative process peak; GC statistics also include untimed seeding.
- No baseline latency exists for mixed handoff: baseline rejects it.
- This statistical resource check alone cannot certify the other Gate G requirements.
