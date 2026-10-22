# Written text contract (H24)

New written input uses `written-text-v1`: `text`, `language` and the historical
`web-search` transport name share the existing `language` numerical namespace.
Values are trimmed and lowercased with the same normalization used
by sequence learning. Input source is provenance and never changes the vector.
`UserInputPerception` continues to report `text`; `FromWebSearch` now also reports
`text` and retains its separate source. Observations and knowledge exposures
retain the caller's source/modality metadata.

This is an adapter encoding correction, not a semantic parser. Audio, vision,
IMU and numeric sensory coordinate systems retain their separate encodings.
No neuron receives a lexical identity, and no answer table or cognitive rule
engine is introduced. Sequence order, roles and negation are still H11 work.

## Shared paths

| Path | Written contract |
| --- | --- |
| `EncodeObservation` and normal knowledge grounding | `written-text-v1` |
| Runtime sequence learning, including inline observation learning | Same language vectors; existing anchor learning, connection and pattern rules; adapter labels are retained |
| Facade `Pulse` / `PulseWithLabels` | Real text perception, shared grounding, detached readout |
| Bridge `POST /v1/observations` | Same complete observation command and shared grounding |
| Knowledge document compilation/import | Same vectors for written datum values; original source modality remains in provenance |
| Live and detached label lookup | Recognizes written aliases while keeping physical modalities separate |

`Experience.Sequence` is the existing lexical sequence API. Its `Modality` is
evidence provenance, not a switch that turns strings into acoustic or image
measurements. Physical observations must use their sensory/numeric adapters.
The old low-level `StoreAt` remains a compatibility operation; managed learning
uses the versioned surface path and preserves one occurrence per experience.

Adapter bindings reuse an already learned population even after its numerical
members move. New lexical sequences retain their existing anchor learning
shape; this patch does not force every learned word to grow a new population.
Target labels are stored separately from the canonical neural JSON and are
authenticated in complete checkpoints.

## Journal and checkpoint compatibility

Written encoding was introduced in **schema 2**. New records now use schema 3
for ordered neural dynamics (see [ORDERED_OBSERVATIONS.md](ORDERED_OBSERVATIONS.md)). Commands that consume written surfaces
record `surface_encoding: "written-text-v1"`. Schema 1 remains readable with
its historical encoding, original annotations and learning behavior. A mixed
journal is valid: each replayed command chooses its recorded contract inside
an isolated runtime, then the serving contract is restored before publication.
Replay must never run concurrently with serving operations.

An absent surface encoding means the historical contract, including for raw
administrative records. Unsupported values are rejected. A versioned encoding
cannot be inserted into a schema 1 record: older binaries would ignore that
field. Schema 2 makes old readers reject the record instead of interpreting
new input with their old encoder. Reader, repair, event-index and sensor-tail
paths accept the supported schema versions and reject invalid payloads.

Checkpoint v8 is the current complete runtime format; v7 remains readable and
supports offset recovery. Historical logical-state checksums stay exact when
comparing the original envelope version; absent adapter/trace fields are omitted.
Unknown checkpoint adapter contracts are rejected before changing the target
runtime. New serving operations use the new contract after restoration;
historical records do not silently upgrade themselves during replay.

Keep a pre-upgrade checkpoint and journal copy when rollback is needed.
Do not point a schema-1-only binary at the upgraded journal. Brain-only exports
still lack the complete runtime and adapter continuation state; production
restart uses authenticated checkpoints and the committed journal.

## Explicit historical rebinding

Old `text` and `language` populations may both exist for the same surface.
They are never automatically merged or deleted. The default written path
preserves the established language namespace and existing versioned bindings.
To retain a different historical written route, select its exact checkpointed
surface and modality through a reviewable rebinding plan.

Within one supervisor mutation lease:

```go
err := h.Supervisor().WithMutation(func(r *runtime.BrainRuntime) error {
    if err := r.Checkpoint(referencePath); err != nil {
        return err
    }
    plan, err := r.PlanWrittenSurfaceRebinding("NAKU", "text")
    if err != nil {
        return err
    }
    return r.ApplyWrittenSurfaceRebinding(plan, eventTime)
})
```

Planning is read-only. The plan includes the selected old annotation, its
population IDs and a hash of the revision, canonical Brain and complete adapter
metadata. A stale revision, modified reference or changed population fails
before a journal append. Ambiguous historical annotations are rejected.

Applying records one `surface_rebinding` revision before changing any binding.
Only the current written adapter association changes. Original annotations,
neurons, representations, synapses and traces remain intact. The new annotation
retains `rebound_from` and `reference_hash`; its provenance survives subsequent
observations and checkpoints. Input and new sequence learning both follow the
selected population. If old text/language routes conflict, the plan explicitly
selects one; it does not infer that their learned histories are equivalent.

The plan requires an existing historical surface annotation. An unannotated
unit cannot be assigned a word by inverting its hash. Re-present the original
source through the journaled input/learning API to establish such labels;
historical replay itself intentionally preserves the original metadata.

Append failure leaves the binding untouched and retires the runtime under the
existing recovery-required contract. Genesis replay and checkpoint-offset
recovery reproduce the receipt and the next observations. These typed methods
also reject a retired runtime.

## Evidence and limits

On parent commit `68708eb852ee900d636edec0fe3e4d8d0fbe1f59`, six exposures to
`naku melo tiva` / `sopo keri luma` left both targets absent from normal facade
text readout (activation zero). Current integration tests train through the
journaled runtime API, omit the targets from the prompts, and verify:

- Both learned prefixes reuse their trained units and both held-out targets
  appear in the facade neural readout with labels, including case/space variants.
- Source changes and written aliases keep the same prefix units. Checkpoint
  restart plus suffix replay preserves the learned state and retrieval.
- The actual managed Bridge handler retrieves the target before and after
  restart and keeps exactly the three trained units for that scenario.
- Written document import and subsequent observations/sequence learning reuse
  populations; audio/vision/IMU controls remain separate.
- A fixture generated by the actual parent code preserves its full-state
  checksum, checkpoint-offset recovery, historical genesis foundation and
  exact next recurrent output. See `testdata/legacy_surface_v1`.
- Explicit rebinding retains all neural state, source provenance, write-ahead
  behavior, mixed-version replay and restart continuity; invalid plans do not
  change state or append a command.

Target presence and adapter labels are not semantic answer accuracy. The
external language lab's compatibility-token scores are not used as proof of
active-path success. This patch does not claim the full language curriculum,
role/negation understanding, cross-modal transfer, decay or interference fixes.
