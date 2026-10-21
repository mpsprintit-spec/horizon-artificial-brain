# Learned spelling diagnostics

Collected on 2026-10-21, Linux/amd64, Go 1.22.12, AMD EPYC 9V74.
Synthetic fixtures only; no personal Brain memory or deployment.

- `correctness.txt`: Unicode, label/representation ablation, uncertainty,
  causal bit/synapse controls, correction, persistence, journal failures,
  owner concurrency and the managed text-prefix-to-spelling facade.
- `benchmark.txt`: GOMAXPROCS=4, 20 calls per benchmark mean, three repeats;
  setup/serialization outside timing. Allocations are cumulative per operation,
  not peak RSS. See the main contract for the large graph footprint limitation.
- `go-test.txt`: root-module suite; nested modules are outside its scope.
- `race.txt`: six-package race suite.
- `codec-race.txt`: focused codec race suite after the final registry-read locking
  change; includes concurrent low-level connection updates and frozen v1 replay.
- `bridge-smoke.txt`: live loopback integration. A readiness retry may initially
  report connection refused before the listener is ready; the whole command
  exited successfully and all integration flows passed.
- `old-h18-empty-restore.json`: the actual H18 reader accepted a newly generated
  schema-12 rollback backup with nil initial cognitive maps, and its restored
  canonical Brain matched the backup. The reader binary digest is recorded.

No decoder speedup, end-to-end latency, energy, physical grounding or semantic
accuracy is claimed. Exact-head CI is recorded on the stacked draft PR.
