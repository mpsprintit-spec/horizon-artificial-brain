# Phase II validation record

Local Go 1.23.2 validation completed successfully. Scientific failures remain
separate from these technical passes. Exact published-head CI is recorded in the
PR and delivery evidence, not inferred from PR60/61's earlier CI.

| Check | Result / evidence |
|---|---|
| Full module regression `go test ./...` | PASS, `release-all-tests.log` |
| Targeted race: knowledge, activation, runtime, nativefoundation | PASS, `release-race.log` |
| `go vet ./...` | PASS, `release-vet.log` |
| `go build ./...` | PASS, `release-build.log` |
| Native research CLI, Android arm64, CGO disabled | PASS, `release-android.log`; cross-build, not on-device execution |
| Pending-pair checkpoint and WAL suffix | PASS, canonical/activation/checksum equality at sequence 531 |
| Final functional reproduction, seed 2027 | Exact equality of functional metrics; elapsed wall time excluded |
| Matched old-task retention evaluation | All 15 frozen checkpoints restored and evaluated on the same cases as post-adaptation |

The local release suite covers final runtime/knowledge/activation mechanisms.
The final retention helper and CLI help text were added afterward and receive
their own targeted test/build; published CI validates the complete committed tree.

## New meaningful checks

- Shared readout: permutation invariance, new graph members after initialization,
  and canonical parameter updates. Multi-channel synapses are summed in stable
  numeric order; reciprocal/nonreciprocal mixtures exercise the computed ratio.
- Typed metadata clone equals the historical JSON clone and remains isolated when
  nested maps/slices/decisions are mutated.
- Paired learning: positive and negative delayed rewards, common active-feature
  cancellation, inactive-feature protection, and factual mean calibration through
  bias without changing common feature weights.
- Runtime: pending trial restart, journal suffix replay, duplicate outcome,
  completed-contrast reuse rejection, explicit discard, and budget rejection
  before WAL/state mutation.
- Subprocess crash before append and after fsync/before apply of the second
  outcome. Recovery distinguishes the non-durable and durable cases and avoids
  double learning.
- Research output refuses a nonempty directory; existing checkpoint bytes remain
  unchanged. Boundary evaluation uses exclusive creation for its result file.

An initial overbroad policy-read limit broke the historical v1 budget test. It
was corrected to apply the combined policy/operation budget only to v2. The failed
`full-tests.log`/`race-tests.log` are retained alongside subsequent passing logs;
they are not the release status. No old production learning law was replaced.

## Persistent evidence and compatibility

`persistent-pair-recovery/` contains the initialized fixture, before checkpoint
with a pending contrast, journal, after checkpoint, rebuilt checkpoint and
verification JSON. It executes 64 real training episodes before the paired
boundary. The canonical model remains v2 inside canonical7/checkpoint20/WAL14.
Physical actions executed: zero. Native operation replay has no actuator path.

Regression tests retain v1 serialization/replay, corrupt-checkpoint and incomplete
journal rejection, deduplication and canonical Brain identity checks. New optional
fields use `omitempty`; legacy v1 configuration and checkpoints remain accepted.
Older binaries cannot read a v2 model. This is an explicit opt-in version boundary,
not a silent migration. No actual user checkpoint or journal was replaced/deleted.

The retained WAL demonstrates one concrete recovery history; unit/process tests
cover additional failure boundaries. It does not establish every filesystem,
power-loss, storage-corruption or concurrent scheduling scenario.

## Scope of integration

PR60 stays merged on main. Phase II is an isolated research continuation of PR61,
with main merged in without rewriting history. Production defaults, legacy inquiry
authorization/risk checks and physical-action permissions are unchanged. No
MUSE/BMPS/V5 dependency was introduced into the native path. The existing runtime
package still contains those frozen historical experiments.

Gate C has bounded same-family transfer evidence, Gate D lacks behavioral
superiority, hidden reward reversal fails retention, and Gate E has an unresolved
whole-thought latency regression. These block production integration regardless of
successful technical checks. Rollback of the experiment needs no data migration:
keep v1/main checkpoints with their compatible binary, and retain v2 research
checkpoints with this branch. Do not load v2 files with an older binary.
