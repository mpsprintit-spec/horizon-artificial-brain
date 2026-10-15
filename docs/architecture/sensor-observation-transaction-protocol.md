# Sensor Observation Transaction Protocol

Status: proposed; implementation is not yet complete.

## Problem

`CognitiveOrchestrator.ProcessSensorObservation` currently projects a numeric observation into the canonical neural substrate before `CognitiveProcess` appends and syncs the corresponding process event. If the append or sync fails, neural state can change without a durable event that can reconstruct that change.

A second failure window exists when temporal reinforcement fails after the process event has been recorded. Replay also temporarily detaches the event log while applying mutations through public runtime methods, without excluding all concurrent runtime mutations for the full replay duration.

## Code path map

The implementation should address these paths together:

- `services/ai/runtime/sensor_observation.go`: `CognitiveOrchestrator.ProcessSensorObservation` currently validates and deduplicates, calls `ProjectVectorPopulationAt`, then calls `CognitiveProcess`, then `LearnObservedTransition`.
- `services/ai/runtime/brain.go`: `BrainRuntime.mu` protects many runtime state changes, while public methods such as `Process`, `CognitiveProcess`, `LearnExperience`, `ThinkAt`, and `LearnObservedTransition` take that mutex independently. Nested calls need internal locked helpers to avoid deadlock.
- `services/ai/runtime/event_log.go`: `EventLog.Append` writes and syncs each record. `ReplayEventLog` currently detaches the event log under a short lock and then calls public mutators, so other mutations can interleave during replay.
- `services/ai/runtime/sensor_observation_test.go`: existing tests cover numeric validation, projection, duplicate IDs, out-of-order timestamps, concurrency, and deterministic sensor replay. Extend this file with failure-injection and replay-isolation cases.

## Required invariants

1. **Single mutation order.** Sensor projection, process-event persistence, temporal reinforcement, sequence advancement, and processed-event indexing must be ordered under one runtime transaction boundary.
2. **No false commit.** An event must not be treated as successfully processed unless the system can recover to a well-defined state after any failure.
3. **Replay isolation.** Replay must exclude normal runtime mutations for its entire duration; restoring the event-log pointer under a short mutex is not sufficient.
4. **Idempotent recovery.** A retry must not apply the same sensor sample twice. The sample ID and durable sequence must be checked within the same serialized boundary as mutation.
5. **No unsafe log-order swap.** Appending the final event before projection is not sufficient on its own: a crash or projection error can leave a durable event whose neural effect was never applied.
6. **Preserve deterministic replay.** The stored sensor observation remains the source of truth for reconstructing its projection; replay must not depend on a stale, previously assigned node population.

## Proposed transaction protocol

Use a durable intent/commit protocol, rather than relying on an in-memory mutex to provide crash consistency:

1. Validate and clone the observation before entering the mutation boundary.
2. Acquire the runtime-wide transaction boundary shared by every state-mutating entry point and replay.
3. Check timestamp ordering, duplicate sample identity, and the next sequence while holding that boundary.
4. Append and sync a versioned **sensor-observation intent** containing the complete normalized observation and stable sample ID.
5. Apply projection, cognitive processing, and temporal learning through internal helpers that assume the transaction boundary is already held. Do not call public methods that reacquire the same non-reentrant lock.
6. Append and sync a **commit marker** for the intent. Only then publish the sample as processed and expose the committed sequence.
7. If any step after intent persistence fails, mark the runtime as recovery-required and reject further mutations until deterministic recovery resolves the intent. Do not silently continue from partially mutated neural state.
8. During recovery, replay committed transactions in sequence. For an intent without a commit marker, restore a known checkpoint and deterministically reapply the intent, or stop with an explicit recovery error if rollback/replay cannot be guaranteed.
9. Keep replay exclusive for its full duration and restore runtime bookkeeping and event-log attachment in a single cleanup path.

## Recommended implementation sequence

1. Introduce a runtime transaction/recovery state model and define exactly which public mutation entry points must acquire it. Audit every public mutator before changing locking.
2. Extract lock-assuming internal helpers for the existing process, cognitive process, learning, temporal reinforcement, and replay paths. Preserve the public API as wrappers.
3. Version the durable sensor intent and commit records, and teach the event-log reader to validate their ordering and reject orphaned or conflicting records.
4. Implement recovery-required behavior and deterministic recovery from a checkpoint plus the durable log. Do not attempt in-place continuation after a partial sensor transaction.
5. Refactor sensor observation processing to use the transaction protocol end-to-end, including duplicate and timestamp checks.
6. Make replay exclusive for the full replay duration without recursively taking the same non-reentrant lock.
7. Add fault-injection tests at each persistence and mutation boundary, then run the full test suite and race detector.

## Implementation constraints

- Do not introduce a second independent mutex that protects only sensor calls; it would not serialize ordinary `Process`, learning, thinking, outcome, and replay mutations.
- Do not claim crash atomicity from mutexes alone. The durable intent/commit and recovery-required behavior are needed for failures across process restarts.
- Preserve the existing event-log schema or introduce a deliberate schema-version migration with compatibility tests.
- Add fault-injection tests for intent append failure, projection failure, temporal reinforcement failure, commit-marker failure, duplicate retries, and replay concurrent with every mutation class.
- Run the full Go test suite and race detector in CI before treating the implementation as complete.

## Acceptance criteria

- A failed persistence operation cannot silently leave the runtime accepting new mutations from an unknown partial state.
- A sensor sample is applied at most once after recovery.
- Replay reconstructs the same canonical brain and activation checkpoint as the original committed run.
- Replay and ordinary mutations cannot interleave.
- Existing event-log repair and legacy replay tests continue to pass.
