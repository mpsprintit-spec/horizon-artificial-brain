# Complete cognitive commands and checkpoint v9

This change addresses the reproduced H03/H04 cases in the consolidated
2026-10-19 audit, using PR #27 as its baseline. The production integration
target is the root Go module: Bridge, the core Pulse facade, the continuous
runtime and its supervisor. Tests under `go test ./...` do not cover the
repository's separate nested Go modules. This is not closure of all 24 audit
findings or a claim of semantic understanding.

## Transaction boundary

`ProcessObservation` now owns grounding, cognitive processing, temporal
reinforcement, optional experience learning, inquiry completion/planning and
publication under one runtime lock. Duplicate IDs and invalid input are
rejected before grounding or appending. Caller-owned input is copied; filling
experience defaults does not change the caller's experience.

Each complete command is encoded and fsynced before its first mutation. A
successful command advances exactly one logical revision. Optional experience
learning inside an observation no longer consumes a separate revision. Output,
dedup index and telemetry refer to that complete revision. The telemetry hash
is captured after the entire transition, including temporal learning.

| Journal type | Deterministic transition |
| --- | --- |
| `observation` | Full adapter input, optional experience, inquiry options and original event/envelope |
| `cognitive_process` | Activation, prediction, learning signal, memory dynamics and cognitive delta |
| `cognitive_think` | Curiosity, recurrent thought, learning signal, memory dynamics and cognitive delta |
| `action_binding` | Validated binding for later observed outcomes |
| `inquiry_outcome` | Bound evidence, complete observation, pre-action prediction, consequence plasticity, outcome trace and preference update |
| `process`, `think`, `learn`, `outcome`, `consequence` | Their existing explicit lower-level operations; `think` retains its context map |
| `vector_ground`, `knowledge_import` | Existing numeric grounding and complete document import |
| `sensor_intent` + `sensor_commit` | Existing two-record numeric sensor protocol, interpreted as one revision |
| `surface_rebinding` | Explicit adapter-only selection of an existing written population, with a verified revision/state reference |

New records use journal schema 3 with explicit written surface and neural
dynamics encodings. Schemas 1 and 2 replay with their original dynamics.
See [ORDERED_OBSERVATIONS.md](ORDERED_OBSERVATIONS.md) for ordered frames and
conditional pattern prediction. [RECURRENT_CONTEXT.md](RECURRENT_CONTEXT.md)
describes current bounded context dynamics and episode controls. See [WRITTEN_TEXT_CONTRACT.md](WRITTEN_TEXT_CONTRACT.md)
for text/language consistency, legacy reference evidence and migration.

An action binding registration now consumes a revision. Re-registering the
same binding is a no-op; changing an existing request ID's targets is rejected.
Explicit bootstrap activation is a journaled `process` with numeric context.
Partial standalone grounding, temporal reinforcement and inquiry planning or
completion are refused when a journal is attached; use the complete
observation/inquiry boundary. Manual checkpoint restore requires a detached
runtime; serving runtimes recover through their supervisor.

## Uncertain writes and recovery

A complete command is a **redo commitment**. If the append reports an error,
its durable status may be unknown. The runtime becomes recovery-required and
closes its telemetry hub without applying the command. If the full command
was actually written and its acknowledgment was lost, recovery applies it
once. A retry using the same observation ID is then rejected by the recovered
dedup index. The contract does not promise rollback or exactly-once network
delivery.

An application error or panic after a durable append also retires the runtime.
Discard that instance and rebuild privately from a verified journal/checkpoint;
publish the new runtime only after successful replay. Malformed records, mixed
command payloads, sequence gaps and unknown command types fail closed.
Legacy sequence renumbering cannot repair the new complete commands. Sensor
tail repair cannot erase a compound command hidden in sensor framing.

The numeric sensor protocol retains its own intent/commit recovery rules.
An uncommitted sensor intent must be explicitly repaired before reconstruction;
it is not a redo command. No full Brain clone is created for ordinary writes.

`inquiry_outcome` contains observed causal data, including the captured
prediction. Live ingress still validates the authorized execution and expiry.
The journal contains no reusable authorization capability, and replay neither
authorizes nor repeats an external device action.

Observation envelopes verify a supplied payload hash before normalization.
Payload JSON is normalized to the representation written by `encoding/json`,
then its stored hash/byte count is recomputed so replay sees the same payload.
This does not change the legacy hash-token fallback for nonnumeric envelopes.

## Checkpoint and historical data

Checkpoint **v9** adds authenticated bounded recurrent context and episode state;
anchored v7/v8 remain supported for offset recovery. Pre-v9 envelopes cannot
carry recurrent eligibility.

Checkpoint **v8** additionally authenticates the last ordered observation and
full population pattern frames. V7 remains supported for journal-offset recovery.
Pre-v8 envelopes cannot carry ordered state.

Checkpoint **v7** added authenticated action bindings, accumulated evidence and
the evidence ledger's node/request/source dedup keys to the existing complete
canonical substrate, recurrent activation, cognitive state, temporal
predecessor, event index, inquiry preferences, adapter annotations and journal
anchor. These additions are runtime continuation metadata; they do not add a
second knowledge store or lexical identity to the canonical neural substrate.

Canonical v2 Brain loading now preserves population provenance exactly.
Bootstrap metadata synthesis remains confined to legacy graph migration;
ordinary learned populations are not relabeled as bootstrap populations.

Historical checkpoints v4-v6 retain detached read/import compatibility using
their original checksums. They cannot provide verified journal-offset recovery
because they lack outcome continuation state. Verified checkpoint-offset startup requires v7, v8 or v9. Do not upgrade an old file by merely changing its schema
or checksum. Migration needs a verified complete baseline, including any
action/evidence state, or a history demonstrably sufficient to reconstruct it.

Historical `process` and `think` records keep their original lower-level
meaning. Old logs did not identify whether a higher-level cognitive caller
also performed unlogged grounding, learning, decay, curiosity or plasticity.
Those omitted effects cannot be inferred or reconstructed by this change.
Full replay of new commands is exact relative to the same explicit foundation;
it is not evidence that an incomplete historical journal describes past live
state completely.

The H05/H06 alias migration and supported ownership boundaries are documented
in [MANAGED_OWNERSHIP.md](MANAGED_OWNERSHIP.md).
Text/language encoding (H24), sequence-role preservation (H11), decay age (H08),
evidence-source independence (H15), performance and behavioral findings remain
open. Preserving evidence counts here does not validate their independence.

## Regression evidence

Tests reproduce duplicate text grounding, uncertain observation WAL writes,
incomplete cognitive replay and lost checkpoint outcome continuation. The
mixed recovery test combines numeric sensors, text/envelopes, optional
experience learning, document import, cognitive and lower-level processing,
contextual thought, action binding, outcomes and inquiry consequences. It
compares canonical bytes and all persisted runtime continuation state across
live execution, genesis replay, checkpoint suffix replay and the next response.

Fault tests cover ten command types, a durable command with a lost
acknowledgment, late invalid input, concurrent duplicate observations, payload
smuggling, checkpoint tampering and the actual Bridge/Pulse ingress paths.
Existing numeric sensor, monitor lifetime and supervisor handover tests remain
part of the root suite. CI also race-tests the evidence ledger's learning
package alongside knowledge, runtime, Bridge and core.

Race validation exposed a continuous-service cancellation edge: a ready ticker
could start another cycle after a callback cancelled its context and eventually
block an additional callback. The service now checks cancellation before each
cycle and immediately after the completed transaction's callback. It never
interrupts an in-progress journal transition halfway through application.
