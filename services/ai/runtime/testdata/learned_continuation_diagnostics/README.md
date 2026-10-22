# Neural continuation diagnostics

All new fixtures and probes are synthetic and separate from any personal Brain.

- `correctness.txt`: composition, weak/missing/ambiguous/expired contexts,
  label controls, independent causal ablations, repetition/bounds, ownership,
  exact recovery/preview, inherited native roundoff and core text integration.
- `go-test.txt`: complete root-module test run.
- `race.txt`: activation, knowledge, learning, runtime, Bridge, core and CLI.
- `bridge-smoke.txt`: live temporary Bridge persistence/replay and sensor/monitor
  integration. Initial readiness connection refusals may precede final success.
- `parent-roundoff.json` and `parent_roundoff_probe.go`: real-thought rounding
  difference on **unmodified parent PR #45**, without any new preview API.

Validation commands (Go 1.22.12, Linux/amd64):

```sh
go test -count=1 ./...
go vet ./...
go build ./...
go test -race -count=1 ./services/ai/activation ./services/ai/knowledge ./services/ai/learning ./services/ai/runtime ./services/ai/bridge ./services/ai/core ./services/ai/cmd/horizon-cli
make bridge-smoke
go test -count=1 -v ./services/ai/activation ./services/ai/runtime ./services/ai/core -run 'Test(ContinuationHypothesis|LearnedContinuation|ManagedContinuation|RetiredRuntime)'
```

Vet/build succeed without output. These logs are correctness diagnostics, not
benchmarks or a scale-training result. The root module excludes nested modules.
Monitor source is unchanged and is checked by the exact-head CI on the PR.

To reproduce the inherited roundoff separately, run from an unmodified
`99820e0c7912440f84fa4e0a223a3880b88bbf7b` checkout's module root:

```sh
go run -buildvcs=false /absolute/path/to/new/checkout/services/ai/runtime/testdata/learned_continuation_diagnostics/parent_roundoff_probe.go
```

The probe uses the parent module, trains local numerical transitions, saves and
restores a temporary checkpoint, and runs one real thought on each instance.
Its JSON preserves the difference rather than asserting bitwise equality.
The exact differing fragments can vary with old equal-target iteration order.
The new preview uses total ordering; old actual/hypothetical callers retain
their historical law. The next-native-transition test explicitly checks a
`1e-14` relative roundoff bound and exact structure, not bitwise determinism.
