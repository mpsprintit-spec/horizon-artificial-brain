# Managed neural ownership (H05/H06)

The supported production path has one canonical Brain, one BrainRuntime and
one RuntimeSupervisor. `HorizonEngine` keeps its supervisor private and returns
the same owner handle through `Supervisor()`. Every facade operation resolves
the active runtime through that owner; no raw neural alias or cached legacy
cognitive engine is created by the facade.

The ownership migration from PR #28 did not change checkpoint v7 or the neural
representation. Subsequent written encoding and journal schema changes are
specified in [WRITTEN_TEXT_CONTRACT.md](WRITTEN_TEXT_CONTRACT.md). The integration
target remains the root Go module; separate nested modules and alternative
owner prototypes are not implicitly integrated.

## API migration

| Previous access | Supported replacement |
| --- | --- |
| `h.Runtime` / `h.Orchestrator` | Facade `Pulse`, `PulseSignals`, `PulseWithLabels`, `ThinkActive`; specialized typed runtime commands inside `h.Supervisor().WithMutation(...)` |
| `h.Knowledge`, its registry, neurons or synapses | `h.InspectActiveBrain()` and detached `Graph` values |
| `h.Learning`, `h.Thinking`, `h.Language` | Typed operations on the active runtime; legacy engines are not part of facade cognition |
| Assigning `h.Supervisor` | `h.Supervisor()` returns the existing owner handle; replacement uses verified owner recovery |
| `h.WithActiveBrain(...)` | Detached inspection or a typed runtime command; the deprecated managed callback now returns an error without invocation |
| Managed `r.WithBrainView(...)` / `r.DNF()` | `r.InspectBrain()`; mutable callback returns `ErrManagedRuntimeAccess`, DNF returns nil |
| Managed `r.SetEventLog(log)` | `h.Supervisor().AttachEventLog(log)` before the first revision, outside a lease callback |
| In-place `ReplayEventLog` on a managed runtime | `RecoverFromCheckpoint` / `RecoverFromCommittedJournal` on its owner |
| Direct journaled `VerifyKnowledgeClaim` | `ObserveOutcome` with explicit binding and recorded evidence |

The removed fields intentionally produce compile-time migration errors.
Zero-value facade instances fail closed rather than falling back to raw
runtime/orchestrator fields. Configuration interfaces such as perception and
the external execution gateway are still separate from neural ownership.

## Lifetime and synchronization

`NewBrainRuntime` atomically claims an in-memory Brain instance once. It returns
nil if that instance is already bound. A new runtime must not create an
independent sequence counter or recurrent state over an existing runtime's
same mutable Brain. This guard is monotonic and is not persisted. Recovery
constructs a fresh private Brain instance with the same persisted identity and
state, verifies it, then publishes it through the existing supervisor.

`NewRuntimeSupervisor` claims the runtime under its lock and rejects a second
owner with `ErrRuntimeAlreadyOwned`. The ownership claim is never cleared,
including at shutdown or handover. A recovered runtime is marked supervised
before publication and its writer is transferred internally by the owner.

Owner callbacks may use typed runtime methods but must not retain their runtime
argument or schedule asynchronous work outside the lease. A deliberately
cached runtime cannot detach its managed journal, enter private replay,
rewrite its processed-event index, acquire another supervisor, or export its
mutable Brain through the legacy callback/fabric paths. Ordinary supported
mutators remain serialized by the runtime lock and reject retirement.

`AttachEventLog` takes the owner lease itself. It accepts an open, exclusively
supplied empty writer only at revision zero; reattaching the identical writer
is a no-op. Replacing a writer, attaching nonempty history, or joining a writer
to a nonzero runtime requires verified recovery. Failed attachment leaves the
healthy runtime and its availability unchanged. Do not share an EventLog writer
between owners or call `AttachEventLog` from within an owner callback.

Direct claim promotion now takes the same runtime lock as cognition and checks
availability before touching promotion state. When a journal is attached, this
unjournaled compatibility helper is refused. Hypothetical inquiry prediction
holds the runtime lock through target lookup and prediction, and rejects an
unavailable runtime before reading its live state.

Journal append and close check the file pointer under the same log mutex. A
concurrent close therefore causes an append error, rather than racing an
unlocked pointer check and dereferencing a closed/nil file. Runtime commands
treat that uncertain append according to the existing recovery-required
contract.

The CLI wraps its output writer with one mutex shared by commands and the
autonomous monitor. Locking only the monitor print helpers left ordinary REPL
output racing the same writer; concurrent command/monitor output is now part
of CLI race validation.

## Compatibility limits

Standalone, unjournaled experiments can still use legacy `DNF()` and
`WithBrainView`. They are trusted mutable interfaces; any previously retained
Brain, registry, engine or fabric pointer must be discarded before enrollment
into managed production. The Go API cannot revoke arbitrary pointers that an
external caller already owns. Directly constructing legacy engines over a
caller-owned Brain remains outside the managed facade contract. Raw EventLog
administration is likewise trusted and is not an authorization boundary.

The default facade no longer provides those aliases. No global pointer map or
second knowledge store is used to track ownership. Detached inspections and
outputs remain valid after handover. The inspection's new `Importance` scalar,
like its existing plasticity and activation fields, is copied from the source
revision and cannot modify a live neuron.

## Regression evidence and remaining scope

On baseline PR #28, reproductions admitted two runtimes over one Brain, eight
concurrent supervisors over one runtime, a managed mutable callback/fabric,
writer detachment through a cached runtime, managed private replay, and claim
promotion that changed a retired Brain. The facade also exported all legacy
neural aliases. The corresponding tests now reject these cases.

The retired API matrix checks 31 mutation/inspection entry points, exact
recovery errors and unchanged canonical/runtime continuation state. Additional
tests cover journal attachment, append/close concurrency, callback refusal,
detached inspection, active CLI/Pulse/startup continuation and supervisor
handover. Existing H06 monitor/telemetry tests cover cached hubs, overflow,
failed recovery, subscription shutdown and epoch retirement; they remain part
of race validation. This is not a claim about arbitrary external socket timing
or code using unsupported raw pointers.

The ownership patch changes API access rather than learning rules. The H24
text/language correction is documented in [WRITTEN_TEXT_CONTRACT.md](WRITTEN_TEXT_CONTRACT.md).
Sequence/role preservation (H11), decay/interference and serialization cost
remain separate work.
