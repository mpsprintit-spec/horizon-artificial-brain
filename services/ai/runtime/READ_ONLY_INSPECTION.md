# Horizon Neural Read Inspection Contract (PR #24)

The canonical brain is a single substrate owned by `BrainRuntime` and
published through `RuntimeSupervisor`. Inspection must not export a mutable
`*knowledge.Brain`, `*NeuralUnit`, synapse pointer, or registry pointer.

## Allowed pathways

- `HorizonEngine.InspectActiveBrain()` is the public facade. It obtains a
  supervisor read lease and a runtime lock, then creates a detached
  `runtime.BrainInspection` value tagged with the exact runtime revision.
- `BrainInspection.Graph` is `knowledge.BrainReadSnapshot`. Its
  `NeuralUnits()`, `Synapses()`, `Patterns()`, and `CanonicalBytes()`
  return copies. All internal slices are private.
- `BrainReadSnapshot.HasNode`, `NodeForSurface` and `SurfaceForNode`
  are lookup operations over detached data. Lexical adapter annotations
  remain separate from the canonical numeric substrate.
- `BrainRuntime.ResolveNodeLabels` is for hot cognition paths. It reads
  labels under the runtime lock, avoiding a full graph serialization every
  cycle. `PulseWithLabels` holds the same supervisor mutation lease for
  cognition and label resolution.
- `evolution.ClusterSnapshot` and `ComputeSnapshotStats` consume values.
  The clustering path builds edge statistics with a single sparse graph scan,
  rather than recomputing incoming links per neuron.

## Forbidden production patterns

Production Bridge, startup, and CLI code must not invoke
`WithActiveBrain` or `WithBrainView` to inspect live neurons. They also
must not cache `HorizonEngine.Knowledge` or `HorizonEngine.Runtime`.
`inspection_architecture_test.go` guards these source boundaries.

These legacy public APIs still exist for compatibility, and are **not**
security boundaries. External applications holding raw pointers can still
mutate the canonical Brain. A later explicit breaking API migration would be
needed to remove these aliases and enforce complete process-wide ownership.

## Performance and lifetime

- Do not construct full read snapshots while checking an ordinary user
  input or an unrelated CLI command. The CLI fast path avoids these copies.
- A snapshot allocates O(number of units + synapses + patterns) memory and
  may copy canonical JSON. It is appropriate for inspection, not for every
  cognitive microstep.
- Snapshots remain detached and valid after a supervisor handover.
  Their `Revision` is the source runtime revision at capture, not a
  live revision subscription or another independent neural brain.
- No snapshot method mutates the live substrate. Returned collections can
  be modified by callers, but this affects only their private copies.

## Verification

The automated regression suite exercises defensive copies, lexical
annotation separation, numerical statistics equivalence, checkpoint
handover, closed-supervisor fail-closed behavior, and CLI hot-path
allocation limits. This is a narrower claim than perfect encapsulation:
the existing legacy aliases remain explicitly untrusted.
