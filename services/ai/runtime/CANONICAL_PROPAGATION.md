# Explicit canonical neural propagation

Status: opt-in numerical law above draft PR #46. P0 remains open.

## Problem and numerical contract

Historical native propagation sorts outgoing synapses by target ID alone.
Independent excitatory and inhibitory channels can share that target. Their
stored list order may differ after canonical serialization and restoration.
Floating-point addition is order dependent, so an otherwise exact recovery can
produce a few ulps of difference on its next real thought. The unmodified PR #45
reproducer in `testdata/learned_continuation_diagnostics/` demonstrates this
without calling a continuation preview.

`canonical-propagation-v1` inherits `evidence-provenance-v1` and orders outgoing
channels by target ID, excitatory before inhibitory, then the existing Kind
identity. Ontology-free dynamic channels have empty Kind. The compatibility
tie-breaker does not interpret a semantic label. Valid persisted connections
have unique target/polarity/Kind identities. Source neurons already use sorted
IDs. The detached pointer slice is sorted; stored lists/history are unchanged.

For one receiver, the same floating-point terms are now accumulated in the same
order during observation microsteps, real internal thought, the next forecast,
and versioned inquiry prediction. No epsilon, rounding, quantization, plasticity
change, new neuron field, semantic table, or additional knowledge store is used.
This specifies software arithmetic, not a biological law or a new physical
neuron model. Exactness is tested on Go 1.22/amd64; arbitrary hardware/compiler
equivalence is not established.

## Opt-in and recovery

```go
rt := runtime.NewCanonicalPropagationRuntime() // always an empty canonical Brain
```

The historical `NewBrainRuntime` default remains `evidence-provenance-v1`.
Normal CLI, Bridge and core constructors do not select the new law. Existing
Brains are not migrated. Learned continuation previews accept either law;
the independently versioned preview arithmetic and frozen v1 output are kept.

| Boundary | Canonical propagation contract |
| --- | --- |
| Journal | Existing schema 3 with explicit `dynamics_encoding` on every command |
| Append | Reject crossing into/out of canonical propagation before writing; first-record law cached under the log lock, checked again after reopening |
| Direct replay | Caller supplies a matching runtime; the entire batch's law timeline is checked before applying its first command |
| Genesis rebuild | Select the recorded law, require the original empty canonical foundation, retain that law for future commands |
| Checkpoint writer | Opt-in schema 14, explicit dynamics encoding; historical writers still emit schema 13 |
| Checkpoint checksum | New v9 digest binds complete existing v8 state plus dynamics encoding; v4–v13 checksum laws unchanged |
| Checkpoint/suffix rebuild | Require a matching law across the entire journal, including the authenticated prefix and an empty suffix |
| Owner handover | Restored runtime retains the law and appends future commands under it; old owner is retired |
| Legacy mutable entry points | Brain-only load, index-only import, mutable Brain callback and unversioned DNF access rejected in opt-in runtimes |
| Written v2 migration | Rejected before backup creation because its historical rollback format cannot retain this law |

The new encoding in an old checkpoint envelope is rejected, even when its old
checksum is valid. Schema 14 requires exactly `canonical-propagation-v1` and a
matching checksum. Historical readers reject the unsupported encoding/schema.
There is no implicit upgrade or downgrade and no migration/rollback command.
To end an experiment, discard its isolated runtime and start a fresh historical
runtime or restore its complete historical checkpoint through that reader.

An empty journal has no recorded law. Recover an opt-in runtime before its first
command from its schema-14 zero checkpoint, or explicitly create it anew.
Standalone log attachment retains its legacy API; append detects a mismatched
law before durable bytes/neural application and the runtime requires recovery
under its existing append-failure rule. Full journal validation remains the job
of the existing strict reader/rebuild path, not the bounded first-record check.

## Evidence

`canonical_propagation_test.go` reuses the previous teacher's local paired
lessons with three numeric/allocation seeds. After 48 lessons and one numeric
context observation, its bounded control reads `Bola jatuh.`. Teacher-supplied
factorization and numeric context remain explicit; no new grammar learning or
new-actor generalization is claimed.

For each seed, genesis replay, zero-checkpoint replay, middle-checkpoint suffix,
full-checkpoint empty-suffix recovery, and standalone checkpoint restore match
the live runtime before continuation. Each then runs eight additional real
thoughts: **120 recovered/live comparisons** of complete native output and
complete detached checkpoint bytes, with no tolerance. Fixed clocks and equal
journal attachment state are required. Repeated preview reads do not change
canonical/runtime state or source journal bytes.

The small frozen fixture in `testdata/canonical_propagation_v1/` independently
pins an 18-command journal and eight future native output/checkpoint digests.
Opposing-list reversal tests cover real propagation, observation/forecast,
internal thought and read-only inquiry forecasts. Negative cases cover mixed
law timelines, wrong readers/foundations, append/reopen, incompatible zero
checkpoints, checksum/law tampering, and unsupported mutable imports/migration.
The previous learned-continuation fixture and historical recovery tests remain
unchanged. Their legacy roundoff allowance is confined to the historical law.

## Model and resource boundary

The design source `Horizon_Model_Fisika_Informasi_dan_Komputasi_2026-10-21.md`
(v0.1) is a theoretical scenario model. Its compact arrays, physical equations,
32-byte neuron / 16-byte synapse payloads and millisecond budgets are not the
current Go object graph or measured performance. This change implements none
of those candidate state equations and makes no speed or memory improvement
claim.

Sorting keeps the previous O(d log d) work and detached pointer-slice allocation
per active sender; it adds channel tie comparisons. Journal-law detection reads
at most one 16 MiB record per opened writer, with a 64 KiB initial scan buffer.
Full replay validation remains O(journal length); foundation checking compares
complete canonical bytes; checkpoint restore still materializes the graph.
No scalability gate is closed by these bounded correctness experiments.

Before redesigning or quantizing the substrate, define the resolution and
distinguishability of reusable learned micro-information N. Do not set a neuron
quota per object. The four-member populations in this fixture are controlled
teacher inputs inherited from the previous experiment, not such a quota.
Binary identifiers or Unicode transport bits are not evidence of learned N
content. Distinct learned states must remain distinguishable; equivalent
information should reuse canonical representation based on learned evidence.

The next model experiments must specify measurable local contributions, causal
ablation, held-out transfer, retention, precision drift, and measured bytes/time
before changing the production substrate. The proposed physical model's error
routing, stability, local-learning rule and whole-system costs remain unresolved.
