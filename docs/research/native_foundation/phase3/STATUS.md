# Phase III continuation checkpoint

Source parent: PR62 `ea69b2101004dbf10c07068e42436170c2adcf8d`.
Working branch: `research/native-phase3-20261108`. No production merge.
The exact commit is the repository history containing this file (publication
identity is recorded in the PR; API and local commit timestamps may differ).

Implemented: opt-in model v3, factual initial-action contrasts, explicit matching
scope, same-state and continuation validation, canonical advantage updates,
existing WAL/checkpoint integration, and a matched factorial experiment runner.
No new learned parameter store. v1/v2 laws and historical evidence are retained.

Tested: all `go test ./...` packages passed on candidate code; v3 pending-pair
restart, duplicate handling and subprocess crash before append / after fsync
before apply passed. Added an executed inhibitory-transition counterexample to
the v2 encoder. Race, full final-head verification and CI still pending.

Development: 32 blocks fail learned stopping. At 128 blocks action pairing begins
to improve behavior; 512 blocks solve this limited task. On fresh validation
seeds 6007/6011, noisy returns are action .098591, feature-mean -.183470,
unpaired -.414988, unmatched -.377585. Without offsets the first three approaches
are essentially tied (.09859). This is a conditional benefit, not general causal
reasoning. Exact raw metrics are in development_results.json.

Causal limits: initial-action credit under identical continuation only; adapter
matching of hidden variables must be true. Wrong claimed matching is intentionally
not detectable from a tag alone. Active nuisance parameters can still change.
Identity readout has no new graph-transfer evidence. Inhibitory encoder collision
proves a representation limit: identical features, different executed transition.

Next: run locked fresh confirmation, broader regression/race/Android/CI checks,
independent performance ABBA; preserve all raw checkpoints. The richer learned
relational encoder, observable-context retention, arbitrary temporal contribution
localization and production sensor/inquiry integration remain incomplete.

Reproduce (Go 1.23.2 locally; CI also checks Go 1.22):

```
go test ./...
go test ./services/ai/runtime -run 'TestNative(ActionContrast|ContrastCrash|RelationalInhibitory)' -count=1
go run ./cmd/horizon-native-foundation -phase3-credit action -seeds 6007,6011 -train 512 -test 256 -stream 17100000 -out NEW_EMPTY_DIRECTORY
```

All experiment output directories must be fresh. Never overwrite an old checkpoint.
