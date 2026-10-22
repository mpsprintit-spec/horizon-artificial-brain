# Canonical propagation validation evidence

Run date: 22 October 2026 (Asia/Jakarta). Go 1.22.12, linux/amd64.
Parent: `3f3716cb283372a18cf12a0ae17fdf2ff515118f` (draft PR #46).
Contract: `../../CANONICAL_PROPAGATION.md`.

All experiments use newly created synthetic Brains. No personal Brain,
curriculum training, migration of an existing Brain, merge or deployment was
performed. The physical/information model remains a design proposal; P0 stays
open.

| Check | Result | Evidence |
| --- | --- | --- |
| Focused canonical propagation, frozen previous continuation, retired API matrix | PASS | `correctness.txt` |
| Three seeds × five recovery paths × eight real thoughts | 120 exact recovered/live output and complete detached-checkpoint comparisons | `correctness.txt` |
| New frozen v1 replay plus eight native future states, repeated ten times | PASS | `frozen-repeat.txt`; `../canonical_propagation_v1/reference.json` |
| Root-module `go test -count=1 ./...` | PASS; runtime package 62.850 s | `go-test.txt` |
| `go test -race -count=1` across seven packages | PASS; runtime package 389.157 s | `race.txt` |
| Root-module `go vet ./...` and `go build ./...` | Both exit 0 | No stdout/stderr |
| Live `make bridge-smoke` | Exit 0; all four smoke families pass | `bridge-smoke.txt` |

The race packages are activation, knowledge, learning, runtime, Bridge, core,
and horizon-cli. Root-module checks exclude nested Go modules. The monitor is
unchanged and is validated by the required GitHub `monitor` job on the published
commit. Its status/run link is recorded in the draft PR description.

Package test durations include harness work and concurrent validation load;
they are not cognitive latency, a speedup measurement, or a storage benchmark.
The smoke log's initial connection refusal is its readiness retry; the script
then reaches the server, passes every check, and exits 0.

Future implementations must preserve the frozen v1 journal/output/checkpoint
digests. Do not replace golden expectations or loosen comparisons to conceal
drift. The inherited historical-law roundoff allowance in PR #46 is unchanged;
the new canonical-law comparisons use exact bytes only.
