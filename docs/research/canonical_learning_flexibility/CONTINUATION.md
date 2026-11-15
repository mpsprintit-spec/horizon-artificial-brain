# Continuation record — canonical learning flexibility

- Branch: `research/canonical-learning-flexibility-20261114`.
- PR: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/68 (draft, base PR67 branch).
- Frozen source HEAD / last code commit successfully published: `daf040ae5330d6f8949b36c9aacb1293d596943c`.
- Earlier implementation commit successfully published: `226706e7d499415fefb43b3fc203becbe974cf4d`.
- Evidence/report publication follows that source commit; resolve current branch HEAD with `git rev-parse HEAD` or GitHub. This record intentionally names its verified source parent rather than inventing its own containing commit SHA.
- Main remains `3a9e7ce3ffadff86c792050056ebea4eda48da0b`; no merge/force push/default activation.

## Saved training lineage

Archive: `evidence/Canonical_Learning_Flexibility_Evidence_20261114.tar.xz`.
SHA256 `b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0`.
196 payloads verified against MANIFEST.json. It contains full failed development
runs as well as validation, confirmation and 30 subprocess crash artifacts.

Continue `flex-confirmation-20261114/05-restart-continuation.json` with
`flex-confirmation-20261114/events.jsonl` from the SAME archive.
Sequence 10817, native updates 1536, identity horizon-primary-brain.
Canonical N hash `cfbe7b6f625b2832962acc777e095f7cc9d3303a41deccdeaf82d40e15287acb`.
Complete continuation hash `d2d9a3a0d648993dcff17b4a8e34bedb061c701373091d31e5c64351fa66af91`.
Checkpoint checksum `5b44d5164b126356ddd1332302864b3256ae9a1b4da6ea825584b2c4d6115561`.

Never substitute this research Brain for the user's production memory. Evaluation
copies are controls only. Native V2 uses existing v3 learning, canonical7 /
checkpoint20 / native journal14; old readers reject the encoder. Verified rollback
uses checkpoint 02-B plus rollback-B-prefix.jsonl at sequence42; the complete
journal and native learning remain preserved for a compatible binary.

## Changes and evidence

Production changes: knowledge/native_relational.go, native_control.go,
native_control_validation.go. Regression and continual harness sources are frozen
at the source commit above. AUDIT.md maps mechanisms and limits; PROTOCOL.md locks
targets; REPORT.md gives results; VERIFICATION.md gives exact reproduction.
All 795 Go/data file hashes still match source-and-lineage.json in the archive.

Local full tests, full scoped race suite, vet, host build, Android arm64 CLI build,
30 crashes, old-reader rollback, V2 physical-safety probe and source architecture
checks PASS. No new physical dispatch on replay. Seven genesis and seven suffix
complete-state/checksum comparisons PASS. A/B/C/D local targets and retention PASS.
C accuracy 3/6 -> 6/6; advantage error .762713 -> .363488. A/B pattern ablation
and policy-edge ablation show dependence on learned canonical state.

CI source-freeze run 43586914319 is recorded in evidence/ci.json; at preparation
its monitor job passed and validate was still running its race suite. Recheck the
actual latest run before claiming final CI completion or any merge readiness.
G1–G7/G9/G10 retain passing Gate G regressions; **G8 remains failed, 81/90**.
The scoped encoder benchmark is not a new G8 evaluation.

## Open limits and next action

1. Check CI for the actual source/report HEAD, retain any failure logs; no merge.
2. Continue the same final Brain with new data through existing runtime APIs.
3. Missing-intermediate reuse still fails; temporal eligibility and same-moment
   graph distinctions remain invisible to this encoder. Diagnose observability
   before any extension, rather than changing source merely because training fails.
4. Existing encoder-v1 learned policy migration to v2 is not implemented; no
   in-place model replacement is authorized by this experiment.
5. G8 repair, controller-default promotion, and real Android measurements remain
   separate work. CLI/Bridge expose current observations/composition; native
   episode training currently uses the existing explicit Go runtime API.
