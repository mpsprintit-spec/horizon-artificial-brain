# Verification and reproduction

Frozen source: `daf040ae5330d6f8949b36c9aacb1293d596943c`.
Local compiler: Go 1.23.2 linux/amd64; Linux 6.18.57, overlayfs. GitHub CI uses
Go 1.22. This directory only contains documentation/evidence changes after freeze.

## Encoder cost

Prelocked comparison: 100 iterations per cell, 3 repeats, GOMAXPROCS=1. Same
canonical runtime, graph fixture, activation and operation set. Median across
repeats below; raw output in evidence/encoder-benchmark.log. Alert threshold was
2x time or bytes/op; **no cell crosses that alert**. Other long validation jobs
were running in the managed environment; these are scoped engineering readings,
not an isolated hardware performance certification or a new G8 decision.

| World neurons | Density | Old encoder, us/op | Signed encoder, us/op | Time ratio | B/op old -> new | allocs/op old -> new |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| 1000 | .001 | 6.245 | 7.605 | 1.218 | 1067 -> 1459 | 15 -> 18 |
| 1000 | 1 | 512.550 | 592.597 | 1.156 | 9307 -> 9667 | 17 -> 19 |
| 4000 | .001 | 5.348 | 8.711 | 1.629 | 1134 -> 1497 | 17 -> 19 |
| 4000 | 1 | 2264.673 | 2472.742 | 1.092 | 33851 -> 34239 | 16 -> 19 |
| 40000 | .001 | 18.393 | 26.155 | 1.422 | 1417 -> 1791 | 16 -> 19 |
| 40000 | 1 | 30539.876 | 31780.503 | 1.041 | 328768 -> 329149 | 16 -> 19 |

Structural reads are exactly unchanged in every cell: 4, 4000, 16, 16000,
160 and 160000 respectively. Parameter reads increase 80 -> 144 for this
four-operation fixture, as expected from 4 -> 8 feature cells plus bias.
No graph cache, journal bypass, weakened validation or learning-law change.
The old encoder remains the old basis; the extra cost belongs to the new opt-in
basis. **PR67 G8 remains failed at 81/90; this is not its 90-cell benchmark.**

## Checks

| Command / check | Status |
| --- | --- |
| `go test ./...` | PASS; runtime package 130.866 s |
| `go vet ./...` | PASS, exit0 |
| `go build ./...` | PASS, exit0 |
| Full knowledge/learning/runtime/teaching/bridge/core/CLI race suite | PASS, exit0; runtime package 1052.684 s, GOMAXPROCS=2 |
| New V2 + Gate G + ordered/recurrent regression | PASS |
| Subprocess crash harness, both encoders | PASS, 30/30 children exit86 at registered boundaries |
| Actual old binary reject/rollback probe | PASS, with exact complete-continuation comparison |
| Gateway mock, old encoder + signed encoder, `-race` | PASS; each 1 authorized live dispatch, 0 new recovery dispatch |
| Architecture dependency/core/production-startup Gate G regression | PASS |
| Android arm64 CLI cross-build, CGO_ENABLED=0 | PASS; ELF ARM aarch64, /system/bin/linker64 |
| Android device latency/resource measurements | Not available; no on-device claim |

Android binary SHA256:
`ff2025525fb1ac08659390d042e2c8a105807ca21cb3573d48bc5b0076783f97`.

GitHub PR: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/68

Source-freeze workflow: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43586914319

CI status is recorded with exact commit/run/job identifiers in evidence/ci.json;
do not infer final-documentation HEAD status from a source-freeze run.

## Reproduce

Use a Go compiler on PATH (the managed workspace's system `go` was a board-game
executable; the pinned compiler above was installed separately). From repository:

```sh
git checkout daf040ae5330d6f8949b36c9aacb1293d596943c
HORIZON_FLEX_EVIDENCE_DIR=/tmp/horizon-flex-new-run \
  go test ./services/ai/runtime -run '^TestContinualLearningFlexibility$' -count=1 -v
HORIZON_GATE_G_EVIDENCE_DIR=/tmp/horizon-flex-crashes \
  go test ./services/ai/runtime -run '^Test(GateGProcessCrash|NativeV2GateGProcessCrash)$' -count=1 -v
go test ./...
go test -race -timeout=30m ./services/ai/knowledge ./services/ai/learning ./services/ai/runtime ./services/ai/teaching ./services/ai/bridge ./services/ai/core ./services/ai/cmd/horizon-cli
go vet ./...
go build ./...
GOMAXPROCS=1 go test ./services/ai/runtime -run '^$' \
  -bench '^BenchmarkNativeEncoderVersions$' -benchtime=100x -count=3 -benchmem
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/horizon-cli-android ./services/ai/cmd/horizon-cli
```

Use a **new empty output directory** for a new training experiment. To continue
the saved Brain, extract the evidence archive and use its final checkpoint and
complete journal together; never rerun the curriculum into its existing journal.
Keep the next training stage in that lineage, not a replacement fixture.

To reproduce the supplemental compatibility checks: create an isolated worktree
at PR67 e9ae233, copy evidence/compatibility/old-reader-test.go.txt to
services/ai/runtime/flex_old_reader_test.go, and set HORIZON_FLEX_COMPAT_DIR to
the extracted confirmation directory. Run TestFlexOldBinaryCompatibility.
The V2 gateway probe is compiled at source-freeze daf040a with
evidence/compatibility/v2-physical-test.go.txt copied to
services/ai/execution/flex_v2_recovery_test.go; run its named test with `-race`.
These probes are additional isolated tests, not production source modifications.
