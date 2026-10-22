# Physical-time passive retention (H08)

The `physical-memory-v1` retention law is also retained by current
`stable-sensory-projection-v1` serving runtimes (see [OBSERVED_FEEDBACK.md](OBSERVED_FEEDBACK.md)
and [LOCAL_PLASTICITY.md](LOCAL_PLASTICITY.md)).
Passive forgetting operates on existing
synapses and experience traces in the single canonical Brain. It adds no
semantic rules, answer store, second Brain or external model.

## Law and clocks

Let `r` be the current retention policy, with the existing fallback to 0.6.
For each connection, elapsed hours start at the later of its learning time
`LastModification` and the Brain's exact `MemoryDecay.AppliedAt` watermark:

- weight multiplies by `exp(-hours * (1-r) / 24)`;
- confidence multiplies by `exp(-hours * (1-r) / 48)`;
- an experience trace's priority uses the weight law, starting at its `CurrentAt`.

The constants are explicit engineering choices. The old confidence multiplier
`(1+exp(-x))/2` did not compose across intervals. Its replacement has the same
initial rate, with slower forgetting than weight. A 24-hour interval at retention
0.6 changes confidence 0.8 to 0.654984602462, independent of interval partition.

The watermark advances monotonically. Duplicate timestamps and clock rollback
do nothing; a later command resumes from the highest settled time. Future-dated
connections/traces are not aged before creation. Missing creation times are not
invented. Policy changes affect subsequent intervals only. Floating-point
composition is compared with tolerance 1e-12 in the controlled tests.

Retention settles after durable intent and before grounding, activation or
reinforcement at the command time. Learning therefore sees retained weights;
refreshing `LastModification` cannot bypass prior idle time. Numeric sensors
settle after their durable intent and before projection. Knowledge-import
microsteps share the same clock. Metadata-only action registration/surface
rebinding do not advance neural time. Candidate outcomes settle only when
accepted evidence promotes the substrate. An uncertain append cannot advance
the memory clock.

Passive retention neither reinforces connections nor consumes synaptic
eligibility. Activity, prediction error and consequences retain their separate
one-second eligibility clock and per-experience updates. H09 inactive activity
weakening is addressed in LOCAL_PLASTICITY.md. Bounded prediction adaptation
(H12) is covered in OBSERVED_FEEDBACK.md; bounded H13 predictive calibration
is covered in EVIDENCE_UNCERTAINTY.md; generalization (H23) remains open. Repeated actual learning/thought may change
weights even at the same physical timestamp; passive idempotence does not claim
idempotence of those experiences.

## Persistence and historical continuation

The journal stays at schema 3 with explicit `physical-memory-v1`. Existing
readers reject this unknown dynamics string. Empty, ordered-observation-v1 and
recurrent-context-v1 records execute their original laws and mutation placement.
Replay restores the current serving default after each historical reconstruction.

Canonical Brain v3 persists the exact, nanosecond-resolution memory watermark;
state without that clock remains v2, preserving old reference hashes. Checkpoint
v10 authenticates the new state and rejects a physical clock in pre-v10 envelopes.
Malformed/missing v3 clocks and unsupported encodings fail before target mutation.
Anchored checkpoint v7/v8/v9/v10 supports suffix recovery. Earlier restrictions
for v4-v6 remain in force. Recurrent context still requires at least v9, not v10.

When adopting a historical Brain, the previous `last_decay_unix` telemetry value
is used once as the already-paid time boundary. Its legacy float representation
has sub-microsecond precision loss; subsequent clocks use exact `time.Time`.
Previously lost weight is not reconstructed or restored by this migration.
Journal/checkpoint versions gate supported recovery; old standalone Brain loaders
predating strict canonical validation must not be used to round-trip v3 exports,
because they can ignore fields they do not understand.

Immutable fixtures are never regenerated against the candidate. The new
`testdata/recurrent_dynamics_v1` fixture comes from actual PR32 commit
`b2ea1f7a4395602fa9e1a067a5daf3ad2a458018` using Go 1.22.12. It includes repeated
same-time thought after an hour, a v9 prefix, full-state hashes and the next
historical thought. All older fixtures also verify their original next thought
with its dynamics explicitly pinned. New serving commands intentionally use the
corrected law; their next state need not equal an old-law thought.

## Reproduction and evidence

On actual PR32, for a synapse weight/confidence 0.8, trace priority 0.6 and
retention 0.6, with no additional learning:

| Schedule | PR32 weight | Corrected weight |
|---|---:|---:|
| +1 hour once | 0.786777163057 | 0.786777163057 |
| +1 hour 100 times at that timestamp | 0.151100482270 | 0.786777163057 |
| +24 hours once | 0.536256036829 | 0.536256036829 |
| +24 hours in 24 steps | 0.005390357599 | 0.536256036829 |
| +24 hours in 96 steps | 0.000000003005 | 0.536256036829 |

Tests cover weight/confidence/priority composition at 1/24/72 hours and three
retention settings; duplicate/backwards/pre-epoch/epoch time; future/missing/new
creation times; policy
changes; legacy adoption; unchanged learning/eligibility clocks; exact canonical
round trips; idle runtime schedules without curiosity/experiences; retention
before reinforcement; observation/sensor continuation; full genesis and suffix
replay; next thought after restart; invalid clocks/downgrades; and failed writes.
The existing complete-command, sensor-fault and ownership suites remain required.

Local validation used Go 1.22.12: root-module `go test ./...`, `go vet ./...`,
`go build ./...`, race tests for knowledge/learning/activation/runtime/bridge/core/
CLI, and `make bridge-smoke`. These checks passed. The root command does not
cover the separate nested Go modules, which this change does not modify.
