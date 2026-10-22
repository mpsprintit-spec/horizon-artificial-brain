# Neural knowledge P0: numerical written grounding, first correction

**21 October 2026 — P0 remains RED / OPEN.** This implements a bounded first
correction following `HORIZON_RED_ALERT_P0_NEURAL_KNOWLEDGE(1).md` (20 October).
It does not establish semantic understanding, learned language generation,
generalization, or readiness for main language training. H19 remains pending.
Baseline: H18 / PR #41, commit
`8a4f9ced05d715b338bab4d324011e8c4f6c8596`.

The controlling architecture is [FINAL_ARCHITECTURE_LOCK.md](../../../FINAL_ARCHITECTURE_LOCK.md):
one persistent distributed Brain, learned neural dynamics, no concept-per-node
ontology, semantic rule engine, answer table or LLM.

## Audit: where words and labels actually participate

| Boundary | Existing mechanism | Finding / change |
| --- | --- | --- |
| Perception | Text is normalized/tokenized; order is retained | Input strings are media, not learned semantic identities. |
| Encoder | Written aliases share a deterministic 16D hash-derived vector | Encoding remains unchanged. A spelling fingerprint is not meaning and is not a trained codec. |
| Sequence learning | `StoreSurfaceAt` in v1 consults `populationForWrittenSurface` | Label annotations select an existing anchor/population after member drift. The P0 concern is a real routing dependency, despite labels being absent from canonical neural JSON. |
| Runtime input | `GroundObservationWithEncodingAt` in v1 consults surface annotations before numerical projection | v2 selects a scoped numeric prototype/population, without a label lookup or lexical fallback. |
| Knowledge import | Compilation produces numeric exposure vectors and keeps source/provenance; active grounding uses the runtime's contract | v2 follows the same numeric written route. Import is not a semantic parser or a proof of factual knowledge. |
| Activation, ordered patterns, recurrence | Learned IDs, population frames, dynamic synapses and numeric state | No new cognitive engine or answer-selection rule is added. |
| Interpreter | `NeuralInterpreter` exposes ranked neural IDs and scores | Ranked activity is not a semantic answer or a generated sentence. |
| Label readout | `SurfaceForNode`, `ResolveNodeLabels`, `PulseWithLabels` | These still provide diagnostic names. They are not a learned decoder; callers must not score matching labels as semantic understanding. |
| Compatibility tools | `Fetch`, detached `NodeForSurface`, old thinking/context/understanding packages and CLI inspection | Lexical compatibility remains in the repository. It is not authorization to restore those retired engines to canonical cognition. |
| Persistence | Canonical Brain excludes annotations; full checkpoints authenticate them | v2 adds numeric receptive fields to that same canonical substrate and separately authenticates its serving contract. |

The old single-anchor sequence-learning shape is retained. These are spelling
observations, not a declaration that one neuron represents one concept. This
patch does not discover word senses, unify synonyms, bind arbitrary modalities
to one concept, or construct a compositional world/event model.

## The new contract

`written-neural-v2` is explicit and opt-in. Existing runtimes and historical
replay retain `written-text-v1` until the typed migration is applied.
**The compatibility default is not an automatically repaired production Brain.**

The encoder, written aliases and source normalization remain the same. The
population selector changes:

- Only numerical `ProjectionPopulation.Prototype` and member IDs determine
  repeated written input's route. The fixed scope names codec coordinates
  (`language` / `written-neural-v2`), never a word, object class or semantic role.
- A prototype remains recoverable when member vectors drift. Learning and input
  reuse the same stored route. New sequence anchors and unknown observation
  populations use the existing neural recruitment rules and sizes.
- Adapter annotations may still be recorded/updated for provenance and debug
  output. This bookkeeping touches labels; it cannot select or redirect the
  numerical route.
- Case/space/source aliases retain their existing spelling representation.
  Alias normalization is not evidence of synonym understanding.
- Non-written compatibility strings retain their historical behavior. Physical
  sensor inputs already use their numeric measurement interfaces. This is not
  a claim that every legacy modality/tool in the repository is label independent.

The new prototype lookup still scans its collection, and annotation bookkeeping
also scans metadata. It does not solve H19's indexing, global scans or lock time.

## Controlled migration and rollback

Under a supervisor mutation lease:

```go
err := owner.WithMutation(func(rt *runtime.BrainRuntime) error {
    plan, err := rt.PlanNeuralWrittenGrounding()
    if err != nil {
        return err
    }
    return rt.ApplyNeuralWrittenGrounding(plan, newBackupPath, eventTime)
})
```

Planning is read-only. It returns detached numeric prototypes/member IDs and a
reference binding the committed revision, canonical Brain and original adapter
annotations. Review this plan against the source checkpoint. Same-priority
conflicts, invalid members and numeric prototype collisions are rejected.
The migration follows the effective v1 language binding, including an earlier
reviewed written rebinding. Separate historical text populations are retained;
their learning histories are not automatically merged.

Applying checks the exact reference and routes before any append/mutation.
Both backup destinations must be new:

1. Write a complete **schema-12** checkpoint to `newBackupPath`, retaining the
   pre-migration state and contract so the H18 binary can read it.
2. With a journal attached, copy its exact pre-migration physical prefix to
   `newBackupPath + ".events.jsonl"` under the writer lock. Sync files and the
   containing directory before the new command can be appended.
3. Append/fsync one `written_grounding_migration` receipt. It contains the
   revision/state reference, exact numerical routes and backup file digests.
4. Add the numeric receptive fields to the same canonical Brain, enable v2,
   advance one revision and invalidate prior inspection cursors.

No existing unit, representation, synapse, pattern, annotation or journal
record is deleted or rewritten. Canonical hashes change because numerical
projection records are added; this is not a claim of an unchanged overall
Brain hash across migration. Existing neural unit/synapse values remain intact.
Backup digests are integrity references, not signatures or proof of semantics.
Supported operations require the owner boundary; legacy raw mutable aliases are
outside this consistency guarantee.

Stale/edited plans and existing backup paths fail before neural mutation or
journaling. An uncertain append retains the backups and retires the runtime;
recover through its supervisor. Recovery redoes a committed receipt without
creating or overwriting any local backup file.

For rollback to H18, restore the **pre-migration checkpoint and its copied old
journal prefix**. Do not ask the older binary to consume the later v2 journal.
The current full journal is retained. Do not replace it with a truncated file.

This implementation does not automatically migrate the user's Brain, add an
HTTP migration endpoint or turn on a CLI migration command. The typed API
makes the change reviewable inside the existing owner boundary.

## Recovery and format boundaries

New ordinary checkpoints use **schema 13**, with a checksum-v8 extension that
authenticates `neural_written_grounding`. Historical v4-v12 checkpoints and
their checksum laws remain readable without modification. The special pre-migration backup
uses v12/checksum-v7. The H18 streaming writer still writes the complete Brain
once to file/hash and atomically commits after checksum patch and sync.

Journal schema 3 carries the new command and `written-neural-v2` encoding.
Older readers reject the unknown command/encoding. Recovery validates the
whole supplied contract timeline before applying its first command: v2 cannot
appear without migration, and the contract cannot silently revert afterward.
Checkpoint mode tampering, old-schema flags and invalid scoped routes fail
before target mutation. Equivalent Unicode-escaped JSON spellings are decoded
and checked too; valid escaped v2 state remains readable.

The enabled serving contract survives complete checkpoint restore, genesis
replay, and suffix replay from both pre- and post-migration checkpoints.
Metadata-only written rebinding is rejected after activation; changing a debug
label must not redirect a numeric input. Brain-only load on an enabled runtime
is rejected: use complete checkpoint recovery. Brain-only files do not carry
the full serving/adapter/recurrent contract and must not be used to resume v2.
Compatibility runtime construction and standalone loading also reject a v2
scoped graph instead of silently attaching it to v1.

Migration is an offline full-state boundary, not a bounded inspection page.
Its reference currently materializes canonical JSON; backup and journal-prefix
validation also retain the existing large-state costs. A single migration
receipt must fit the existing 16 MiB journal-record budget. Large migrations
are not established by these small fixtures.

## Evidence and remaining P0 gates

| Gate | Evidence here | Status / limit |
| --- | --- | --- |
| Hide/change labels | Two learned prefixes, three unit-allocation seeds, three annotation variants, member-vector drift | Same neural output, canonical continuation and next text-free thought after migration; narrow numerical routing evidence. |
| Learned-structure control | Remove synapses and patterns together on an isolated copy | Target prediction score drops from about 1.356 to 0 for both prefixes; this joint ablation does not isolate the individual mechanisms or prove semantics. |
| Old failure control | Same drift fixture without migration | Hiding labels grows 6 units to 12. The test deliberately reproduces the old dependency. |
| Graph preservation | Compare unit/synapse values and original annotations around migration | No new/changed units or deleted old data; numeric projection metadata is added. |
| Active interfaces | Managed facade and authenticated production Bridge handler | Prefix retrieval reuses trained units with labels removed; checkpoint/suffix owner recovery succeeds. |
| Persistence/replay | Genesis plus pre-/post-migration suffix replay; frozen actual H18 v12 fixture | Same complete continuation and next thought; rollback also verified with a separately compiled H18 reader. |
| Failed operations | Invalid/stale plans, occupied paths, uncertain append, retired runtime, contract/mode tampering | Rejection and owner recovery behavior tested. |
| Learned spelling decoder | No new trained character/phoneme decoder | **OPEN.** Hash-derived spelling vectors are still encoding, not learned language generation. |
| Held-out roles/negation/order | This patch adds no compositional neural mechanism | **OPEN.** The earlier audit's weak separation on new combinations is not fixed or reclassified as accuracy. |
| Multiple expressions / one concept | Case aliases work; synonyms/cross-modal concept binding not demonstrated | **OPEN.** No semantic identity claim. |
| Event/causal prediction without text | Subsequent recurrent thought is unchanged by label ablation | **OPEN** for world-state, counterfactual and causal task correctness. |
| User memory / main curriculum | Synthetic isolated fixtures only | **Not migrated/trained. P0 remains RED / OPEN.** |

The next [neural answer readout step](NEURAL_ANSWER_READOUT.md) exposes the native
selected forecast separately from activity ranking. It does not supply the
learned decoder or close the generalization/grounding gates above.

A later [bounded synaptic spelling experiment](LEARNED_SPELLING_READOUT.md) learns
numeric I/O through ordinary shared-Brain synapses. The migration itself still
uses the original fingerprint encoder; semantic/causal and scaling gates remain
open. No user memory or main language curriculum is automatically trained.

## Small routing-cost measurements

Linux/amd64, Go 1.22.12, AMD EPYC 9V74, GOMAXPROCS=4. The same candidate binary
compares explicit v1 and migrated v2 contracts on fresh synthetic spelling
fixtures with one anchor per spelling and no learned inter-word edges.
Setup/training/migration and serialization are outside the operation timer.
Each row reports the median of **three benchmark means, 20 calls per mean**;
these are not p50/p95 request latency or confidence intervals.

| Anchors | v1 / v2 mean grounding time, median of 3 | Allocation v1 / v2 | Complete checkpoint bytes v1 / v2 |
| ---: | ---: | ---: | ---: |
| 100 | 1.927 / 1.767 µs | 288 / 456 B per call | 88,891 / 142,406 |
| 1,000 | 14.528 / 12.503 µs | 288 / 456 B per call | 882,093 / 1,415,604 |

No units were added by migration or repeated grounding. v2 adds one numerical
projection per existing spelling route (100 / 1,000 here). At 1,000 anchors,
the checkpoint grows by **533,511 bytes**. Additional persistent metadata and
per-call allocation are real costs; these measurements are not a scalability
or whole-cognition speedup claim. Setup including migration is retained in the
raw benchmark. Peak RSS, energy, broad activity, HTTP latency, full semantics
and million-concept migration were not measured in this probe.

Reproduce:

```sh
GOMAXPROCS=4 go test ./services/ai/runtime -run '^$' -bench '^BenchmarkWrittenGroundingContracts$' -benchtime=20x -count=3
go test ./services/ai/runtime ./services/ai/core ./services/ai/bridge -run 'Test(NumericWritten|LegacyWrittenGrounding|H18Schema|ManagedFacadeUsesNumeric|ManagedBridgeNumeric)' -count=1 -v
```

See [raw diagnostics](testdata/neural_written_diagnostics) and the independently
generated [H18 fixture](testdata/legacy_written_grounding_v1).

Final local validation passed with Go 1.22.12: root-module `go test ./...`,
`go vet ./...`, `go build ./...`, six-package race checks (knowledge, learning,
runtime, bridge, core, horizon-cli) and `make bridge-smoke`. Monitor TypeScript
checking, all 37 Node tests and the production build passed with Node 24.19.0
and pnpm 11.25.0, using the existing local dependency cache. No monitor source
or dependency manifest was changed. Nested Go modules are outside the root test
command's scope. Exact-head CI is recorded on the stacked draft PR.
