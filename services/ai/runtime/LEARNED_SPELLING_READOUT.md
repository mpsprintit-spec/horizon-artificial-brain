# Learned synaptic spelling readout

**Experimental paired numerical I/O codec. P0 remains RED / OPEN.** This step
teaches a bounded Unicode spelling readout on the existing Brain. It does not
learn what a word means, select a semantic answer, discover synonyms, generalize
to unseen words, or generate unconstrained sentences. Main language training is
still blocked. The input spelling fingerprint from v2 is unchanged.

The 2026-10-21 repository audit's A01 requires a learned decoder, separately
evaluated from meaning. A02's activity/forecast mismatch was addressed by the
preceding [answer readout](NEURAL_ANSWER_READOUT.md). This experiment extends the
I/O side of that boundary without introducing a parallel answer memory.

## Fixed transport and learned state

`unicode-synaptic-v1` uses shared output cells for numerical `(position, bit)`
coordinates: 21 Unicode scalar bits plus one learned end bit. Receptive-field
prototypes contain only `position/64` and `bit/32`; these dyadic values are exact.
The scope is `written-output` / `unicode-synaptic-v1`. It contains no word, source
population, codepoint, grammatical role or concept label. A bit cell is an I/O
coordinate, not a concept or a character identity.

The **spelling itself is learned in ordinary dynamic synapses** from the source
population to those shared cells. Bit 1 is taught through an excitatory
connection, bit 0 through an inhibitory connection. `ConnectAt` / dynamic
reinforcement retain adaptive state and frequency. Opposite supervision applies
the existing reconsolidation gain to the opposing connection, retaining its
history and frequency. Repeated exposure reuses cells/connections.

For a detached population, the readout pools signed `weight * confidence` over
its equally weighted members. All members must provide conductance for every
required bit. The sign gives the bit; normalized opponent margin reports
ambiguity. Ties within `1e-12`, absent/partial support, invalid reconstructed
Unicode and missing termination produce a status with **no partial text**.
The end bit is learned too; word length is not stored in a side table. NUL is a
valid scalar and is distinct from the end bit.

`MinMargin` and `Conductance` are properties of this projection, **not
probabilities of a correct answer or understanding**. Unicode bit transport is
fixed; the population-to-bit association is trained. Removing its synapses
prevents decoding even when every coordinate and adapter label is retained.
The decoder never reads labels, member representations, word fingerprints,
pattern result names, or the teacher text.

All parameters remain in the same canonical Brain's units, projections and
synapses. There is no separate model/matrix/lexicon, no LLM and no semantic rule
engine. Output cells participate in the Brain's ordinary propagation/retention;
this is not another intelligence or a replacement recurrent thinking loop.

## Explicit interfaces and durability

Limits are **64 scalars and 64 source members** per exposure/readout. Source IDs
must exist, be distinct/positive and not themselves be output bit cells. Scalar
validation excludes surrogates and values above U+10FFFF. Weight/confidence must
be finite in `(0,1]`. Bank coordinates must be unique and valid; corrupt banks
fail before learning or decoding. No routine exports a live Brain pointer.

```go
// An explicitly supplied I/O lesson, not a factual claim learned from input.
exposure, err := knowledge.NewSpellingExposure(populationIDs, "Tíva🌌", .8, .8)
if err != nil { return err }
err = owner.WithMutation(func(rt *runtime.BrainRuntime) error {
    _, err := rt.LearnSpelling(exposure, eventTime)
    return err
})
```

`NewSpellingExposure` only converts valid external UTF-8 into numeric scalars.
`LearnSpelling` freezes the bounded exposure and checks references/bank shape
before appending. One schema-3 `spelling_exposure` command, with an explicit
codec version and **numeric scalars rather than a spelling string**, is fsynced
before mutation. It advances one revision and uses the existing physical memory
time boundary. Mixed payloads, invalid shapes and legacy sequence repair are
rejected. An uncertain append retires the runtime; verified recovery applies a
durable exposure once, including when its acknowledgment was lost.

`ReadLearnedSpelling` holds the runtime owner boundary and is read-only: no
projection, activation, clock advance, unit recruitment or revision is caused by
reading. `HorizonEngine.PulseWithSpelling` holds one supervisor lease across the
normal Pulse and this readout, decoding only the population already selected by
the native forecast. General activity and diagnostic labels cannot select it.
Its returned spelling remains separate from the cognitive interpretation.

This is an explicit typed interface. It does not automatically train from every
observation, enable language in normal CLI replies, add Bridge training/decoding
endpoints, or alter existing `Pulse` / `PulseWithLabels` behavior. The user's
Brain is not migrated, retrained or overwritten.

Existing checkpoint schema 13 and the canonical graph shape already persist all
required state; no new checkpoint fields/schema are added. Earlier readers can
retain ordinary graph data but cannot interpret the new codec API, and reject
the new journal command. Historical checkpoint/checksum readers remain intact.
Recovery tests cover genesis, full checkpoint and checkpoint-plus-suffix, with
identical continuation and the next text-free thought. The frozen H18 v12 test
still passes. A small prerequisite fix makes cognitive-state cloning preserve
nil versus allocated-empty collections: otherwise checkpointing a fresh Brain
after I/O learning but before its first cognitive transition changed its empty
cognitive state on restore. This does not alter neural dynamics or rewrite old
checkpoint files.

An actual H18 reader also accepts a newly generated schema-12 empty-state backup
with nil cognitive maps; its restored canonical Brain matches the backup. A
[frozen codec-v1 journal](testdata/learned_spelling_v1) pins this new I/O law's
numerical readout, canonical state and next thought for future compatibility.

## Controlled evidence and limits

| Test | Evidence |
| --- | --- |
| Numerical output independent of labels | Six supervised spellings across three numeric/allocation seeds and three label conditions (54 conditions), including reversed member order and member-vector drift. |
| Unicode | Accents, combining scalars, emoji, NUL, empty output, and the 64-scalar limit round-trip; invalid UTF-8/scalars are rejected. |
| Learned causality | Remove source synapses while keeping cells/coordinates/labels: decoding becomes untrained. Flip learned bit polarity: `B` becomes `C`; corrective supervision restores `B` and retains both connection histories. |
| Uncertainty | Mixed equally supported `A`/`B` populations are ambiguous; untrained members, missing end evidence and invalid reconstructed scalars return no text. |
| Same Brain / reuse | One shared positional bank; repeated supervision adds no per-word cell bank. Readout leaves canonical bytes/unit count unchanged. Brain-only persistence retains the numerical readout. |
| Managed application | Actual v2 text prefix through `PulseWithSpelling` selects its learned forecast and returns taught `Tíva🌌` despite the source annotation being `tiva`, hidden or forged. Same state/spelling across conditions and owner checkpoint/suffix recovery. |
| Journal/ownership failures | Invalid input fails before append/mutation; absent and durable/lost-ack append faults recover; mixed commands, concurrent owner updates and retired reads/writes are checked. |

All spellings in these tests were explicitly supervised. The six outputs are
`lantai`, `bola`, `Núra🌌` followed by NUL, decomposed `e` + combining acute,
precomposed `é`, and empty text. This is **not held-out spelling accuracy** or
proof of meanings in the world. The source populations are arbitrary neural
states or existing spelling routes, not declarations that one node is a concept.

This bounded linear readout assumes one supervised orthographic realization for
a sufficiently stable source population. Word senses, contextual form selection,
new-word composition, cross-modal conceptual identity and fluency are still open.
Teaching new edges can change later shared-Brain dynamics; long-run interference
and continual-learning stability are not established by these small fixtures.
No semantic improvement is claimed for ISF001's held-out actors or world model.

## Small cost probe

Linux/amd64, Go 1.22.12, AMD EPYC 9V74, GOMAXPROCS=4. Each source has four members;
each synthetic six-scalar spelling receives four I/O exposures. Setup/learning
and serialization are outside the readout timer. Medians below are of **three
benchmark means, 20 calls per mean**, not request p50/p95 or confidence intervals.

| Supervised spellings | Readout mean, median of 3 | Allocated bytes/call, median | Units / synapses | Shared output cells | Pretty canonical graph bytes |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 8 | 69.878 µs | 23,219 | 165 / 4,256 | 133 | 2,489,369 |
| 32 | 83.005 µs | 23,382 | 261 / 17,024 | 133 | 9,563,615 |

No units were added by the measured reads. The connection/JSON footprint is
large and remains a blocker for broad training. These are whole fixture graph
bytes, not a complete checkpoint, heap, peak RSS, energy or per-concept cost.
No previous decoder existed for a speedup comparison. The readout scans
projection metadata and costs roughly O(P + 22 * source_members * scalar_count);
H19 indexing/global scans remain open. Production serialization policy is not
changed. This probe is not an end-to-end cognition latency result.

Reproduce:

```sh
go test ./services/ai/knowledge ./services/ai/runtime ./services/ai/core -run 'Test(SynapticSpelling|LearnedSpelling|ManagedSpelling)' -count=1 -v
GOMAXPROCS=4 go test ./services/ai/knowledge -run '^$' -bench '^BenchmarkLearnedSpellingReadout$' -benchtime=20x -count=3
```

See [raw diagnostics](testdata/learned_spelling_diagnostics). Overall P0 closure
still requires semantic/causal generalization, concept/word-state binding,
multimodal transfer, scalable representation and broader reproducible evidence.

Local validation passed: Go 1.22.12 root-module `go test ./...`, vet/build,
six-package race tests (knowledge, learning, runtime, bridge, core, CLI), live
Bridge smoke, and the actual H18 reader check above. Monitor typecheck, all 37
Node tests and production build passed with the existing local dependency cache
(Node 24.19.0 / pnpm 11.25.0). No monitor source/manifest changed; nested Go
modules remain outside the root command. Exact-head CI is recorded on the draft.
