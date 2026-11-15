# Phase III continuation checkpoint — 2026-11-09

The foundations are **not complete**. Resume this branch; do not restart from PR61.

## Source and integration

- Branch: `research/native-phase3-20261108`, draft [PR64](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/64), stacked on PR62.
- Authentic parent: PR62 `ea69b2101004dbf10c07068e42436170c2adcf8d`.
- Published action-contrast mechanism: `d33e16f6b887df4f7ba18de7dbeba30007319739`.
- Published absorbing-stop extension: `e8a59cf82de0eba52879fcfffb9a322401a5efe4`.
- Main optimization [PR65](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/65) is merged as `45ecb989b67da690b9bbc1b7a942bb8c55ca6f8e`; exact-head CI42497869614 passed before merge.
- The subsequent commit containing this file adds metadata-copy optimization, retained v3 recovery evidence, benchmark instrumentation and the final scientific report. Its immutable HEAD/tree and publication verification are recorded in PR64 and the evidence archive. `git rev-parse HEAD HEAD^{tree}` identifies a checked-out copy.
- API and local commit identities can differ in author/time, but publication trees are compared exactly. Prior local anchors and PR61/62 history are preserved. No force push.

## Implemented and measured

Factual action contrasts use canonical readout synapses, matched executed trials,
full observed costs, explicit provenance and absorbing stop. v1/v2 laws remain.
Metadata-only preflight avoids cloning full episode maps without changing digests,
learning laws or persistence schemas. The main curiosity allocation repair is
retained in this branch. Relevant files are `knowledge/native_action_contrast.go`,
`knowledge/native_runtime_view.go`, runtime `native_control.go`/`brain.go`, and
`internal/nativefoundation/phase3_credit.go`/`recovery.go`.

Two independently seeded confirmations (one-step and delayed depth0–2) pass all
preregistered **scoped** numerical criteria. Noisy candidate-minus-unpaired return
is +.367572 [CI95 .065095,.670049] and +.538595 [.470252,.606938]. Clean controls
nearly tie. Wrong matching damages the candidate. Five independent model seeds
per arm; 2,048 executed training episodes and512 greedy evaluation cases/model.
All development failures, controls and checkpoints are retained. The32-block
pilot fails stopping. Neither final holdout may be reused for tuning.

Legacy main ABBA: thought p95 24.720→12.255ms,2.017x; B/op3,855,267.5→114,244.5.
This is the same-host comparison, not a multiplication with the historical PR60
ratio. GC-associated tails were isolated; the dominant curiosity allocation source
was removed. Native dense pairs and JSON checkpoint restore remain expensive.
Metadata-view reproduction matches behavior, Brain, activation, checksum and WAL
sequence exactly. A retained v3 pending-pair replay ends at sequence531 with
canonical/activation/checksum equality and no physical action.

## Validation record

Full regression, vet/build, native race and Android arm64 cross-build passed on
e8a59cf; exact-head CI42473928890 passed full race, nested modules, production
recovery, trained dialogue, live Bridge and monitor. The full local race suite
also passed on d33e16f. Optimized metadata/curiosity differential numerical and
native tests passed; the final full regression passed again. Final optimized
race/vet/build/Android exits are recorded in `optimized-validation.json` and the
new publication CI, not inferred from old results. The evidence archive and PR64
record the observed final statuses. Android compilation is not device testing.

## Gates and unresolved work

A passes for this increment; B is demonstrated in controlled environments;
F passes the tested checkpoint/WAL/crash boundaries. C is **not passed**: the
four-moment encoder cannot distinguish an executed inhibitory/excitatory pair.
D is **not fully passed**: learned-effect precision/recall is100%, but first-pair
exclusive synaptic credit is still50% (8 relevant/8 nuisance changes). E has a
verified conservative main repair; native dense/restore scaling needs more work.
G is **not passed**: native experiment guards still reject mixed legacy sensor/
inquiry commands. H observable-context sequential retention remains **unproven**.
The experimental controller stays off production defaults.

Next implement a justified signed/temporal relational representation in canonical
N on new development seeds, then test action-dependent effects, selective credit,
observable-context reversal and matched retention. Do not add strategy rules or
feed evaluator labels into policy. Only broaden sensor/inquiry integration after
these gates have adequate evidence. Performance follow-up should target measured
JSON/digest/restore costs while preserving versioned replay contracts.

## Reproduction and preservation

See REPORT.md, PROTOCOL.md, PERFORMANCE.md and results.json. The protocol contains
locked seed/stream/budget criteria. Raw evidence includes complete checkpoints,
WALs, source hashes, commands, all failed runs, profiles and benchmark resources.
The prior Phase II archive and all368 verified payloads are unchanged.

```
git status --short
git rev-parse HEAD HEAD^{tree}
go test -buildvcs=false ./...
go test -buildvcs=false -race ./internal/nativefoundation ./services/ai/knowledge ./services/ai/runtime -run 'TestPhase3|TestNative|TestCuriosity' -count=1
go run ./cmd/horizon-native-foundation -recovery-audit -phase3-credit action -out NEW_RECOVERY_DIRECTORY
python3 scripts/summarize_native_phase3.py EVIDENCE_DIRECTORY
```

Every experiment output directory must be fresh. Preserve all checkpoints and
historical journals. The final publication handoff must explicitly report any
remaining local edits; no file is considered remotely saved before ref publication.
