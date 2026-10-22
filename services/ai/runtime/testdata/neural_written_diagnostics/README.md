# Neural written grounding diagnostics

These are isolated synthetic fixtures, not the user's Brain or a main language
curriculum. The P0 neural-knowledge alert remains RED / OPEN.

- `correctness.txt`: explicit v2 label ablation, drift, learned-structure joint
  ablation, migration/recovery/failure/contract tests, frozen H18 continuation,
  managed facade and authenticated Bridge handler.
- `grounding-benchmark.txt`: three means of 20 calls per contract/fixture,
  Go 1.22.12, GOMAXPROCS=4; setup and checkpoint writing are not timed operations.
- `old-binary-rollback.json`: an independently compiled H18 reader restores the
  candidate-written v12 backup directly and with its copied journal prefix.
  Both hashes match the pre-migration Brain. The same old reader rejects the
  candidate's schema-13 post-migration checkpoint.

Compile `../legacy_written_grounding_v1/generate.go` from an H18 checkout
(`8a4f9ced05d715b338bab4d324011e8c4f6c8596`) with `-buildvcs=false`.
Use a fresh output directory, then run on the candidate:

```sh
HORIZON_WRITTEN_ROLLBACK_PROBE=/absolute/new/probe-dir go test ./services/ai/runtime -run '^TestWrittenGroundingRollbackProbe$' -count=1 -v
```

The H18-built helper accepts `restore /absolute/new/probe-dir/backup.json`
and `recover /absolute/new/probe-dir/backup.json`. Hash its original canonical
stdout bytes (without its one final println newline) and compare with the
probe's `reference.json`. Post-migration `after-migration.json` must fail in H18.
Keep the old reader's source provenance; do not substitute the candidate reader.

The operation benchmark and focused test reproduction commands are in
[NEURAL_WRITTEN_GROUNDING.md](../../NEURAL_WRITTEN_GROUNDING.md).

`go-test.txt`, `race.txt` and `bridge-smoke.txt` retain the final local Go checks.
The Bridge readiness loop emits one initial connection retry before the server
is ready; the full smoke script then passes and exits successfully.
