# Gate G continuation record

Research branch: `research/gate-g-coexistence-20261113`.
Review: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/67 (draft,
stacked on PR64; no merge/default promotion authorized by this record).

Last verified implementation source:

- Local HEAD: `6ac1a6793c7f8330dca0a9ce15eb32dc64a652e5`.
- Successfully published code commit: `8deae08e7a5fbb4704c47bb563dbcdc73d6f2996`.
- Their exact source tree: `8294419e3e6ddb11c2c04d3e67bcd4a4c489917a`.
- Main: `3a9e7ce3ffadff86c792050056ebea4eda48da0b` (unchanged).
- Phase III: `0df3f19a33bd7de69ed496293591d734331c5e51`.
- Baseline local `1e12332dcf2dc3f5ed974959ebc656b980ee87ae`, published protocol
  `609ee918aa8533e0aca5706dea6046043bdc09bc` (same tree).

Terminal push lacked credentials. Publication through the connected repository
API preserves the native, main and PR63 parents, with exact tree verification.
Local/API author/time metadata differs; no source or previous PR history is
rewritten. The intermediate implementation unit was followed by the separate
passive-memory ownership unit; use the complete source/tree above.

## Verified technical work

- 27 mixed-order/seed cases match complete state from genesis and checkpoint
  suffix, plus continuing learning after restore. Native1/2/3 checked separately.
- 15 subprocess crash boundaries pass; raw plans, journals, checkpoints and
  expected full continuations are captured. Sensor orphan repair retains backup.
- Active/awaiting/pending admission, completed-pair receipts, outcome retry,
  incompatible trial start, corruption, matched-prefix rollback and downgrade
  guards pass. Existing composition can coexist after its explicit migrations.
- Actual production startup restores and continues a native episode, learns from
  observed feedback and adds no bootstrap event or memory overwrite.
- Real physical gateway mock: one live dispatch, zero recovery dispatches.
- Active runtime dependency closure and facade legacy isolation pass.
- Full ordinary Go suite and vet pass. A plain build initially failed because
  VCS discovery selected an outer workspace root; normal build from the isolated
  checkout passes without altering Go source or disabling validation.
- Android arm64 CLI/native-foundation/chat/production cross-builds pass.
- Nested adapters, local monitor checks/test/build, trained-dialogue bundle and
  live Bridge integration pass. Device performance has not been measured.

## Pending confirmation at this preservation checkpoint

- Full local `go test -race -timeout=20m ./...` remains running.
- Exact-head remote CI43466089302 remains running.
- Locked sequential ABBA benchmarks have not started; they must run after other
  local test/build workloads stop. G8 and the whole Gate G are not yet passed.
- G1–G7/G9/G10 have positive targeted evidence, pending the final full-check and
  resource evaluation. No default behavior/whole-foundation promotion is claimed.

## Reproduction and next action

See PROTOCOL.md and AUDIT.md. Preserve all existing negative development logs.

```sh
go test ./... -count=1
go test -race -timeout=20m ./... -count=1
go vet ./...
go build ./...
HORIZON_GATE_G_EVIDENCE_DIR=NEW_RAW_DIRECTORY go test -json ./services/ai/runtime ./services/ai/execution ./services/ai/core ./services/ai -run '^TestGateG' -count=1
python3 scripts/run_gate_g_confirmation.py --baseline BASELINE_CHECKOUT --candidate CANDIDATE_CHECKOUT --output NEW_PERFORMANCE_DIRECTORY
```

Before benchmarking copy the identical committed `gate_g_benchmark_test.go` into
baseline's untracked test harness path. Baseline must contain no integration code.
Use Go1.23.2 and the locked environment/sample counts. The runner records commands,
returns, resource measurements and order. A budget failure requires profiling and
repair, retaining the first result, rather than changing the acceptance budget.

Next: finish full race/CI; run locked benchmarks; evaluate G1–G10 separately;
publish raw evidence/report and update PR67. Do not merge failed gates. The
existing signed-encoder, exclusive causal localization and context retention
limitations from PR64 remain outside this integration increment.
