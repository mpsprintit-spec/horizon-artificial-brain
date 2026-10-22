# Learned neural continuation preview

**Experimental, bounded numerical I/O. P0 remains RED / OPEN.** This adds an
autoregressive preview using the same Brain's learned temporal channels and
synaptic spelling projection. It can combine separately taught local transitions;
it does not demonstrate learned grammar, general understanding, world-model
generalization, or readiness for large-scale language training.

## Mechanism and architecture

`ReadLearnedContinuation(seed, maxSteps, at)` projects an explicit numerical
prefix through the existing recurrent substrate. The current actual activation
and eligible observation history supply its initial state. At each step it:

1. Simulates the expanded prefix without mutation, using dynamic synapses and
   current ordered/recurrent pattern channels on the canonical Brain.
2. Reads the existing population evidence assessment. A selected population
   must have confidence **greater than 0.5** to pass this experimental I/O gate.
3. Decodes that population with `unicode-synaptic-v1`.
4. Appends the predicted population only to a detached hypothetical prefix.

Each step resimulates that prefix from the same committed actual state. Predicted
populations never enter the real observation history, event log, plasticity,
frequency counts, evidence ledger, or native observation forecast. The full
canonical/runtime continuation is byte-identical before and after a preview.
No units, connections, patterns, revisions, or clock/memory-time updates are
created by reading it.

A learned **empty spelling** terminates the output. Spaces, punctuation, Unicode
and all other characters come from the learned pieces; no formatting template,
word joiner, punctuation heuristic, expected-target list or answer string is
consulted. This empty-piece termination is an explicit I/O protocol, not an
inferred grammatical rule. Incomplete attempts expose diagnostic steps, but
publish no final `Text` or `Scalars`.

Learning uses the **existing** `LearnExperience` numeric frame fields and
`LearnSpelling` scalar supervision. Temporal patterns, contextual support,
adaptive connection state, contradiction feedback and history remain on the
same canonical Brain. There is no separate decoder model, semantic engine,
language database, LLM, or mandatory language stage in recurrent thinking.
Nodes are population members, not word/concept identities.

## Typed interfaces and bounds

```go
// Existing teacher interface: numerical frames, never lexical lookup.
_, err := active.LearnExperience(learning.Experience{
    NeuralSequence: [][]knowledge.NodeID{sourcePopulation, nextPopulation},
    NeuralContext:  [][]knowledge.NodeID{contextPopulation},
    Weight: .9, Confidence: .9,
}, eventTime)

// Explicit hypothetical read under the supervisor lease.
preview, err := active.ReadLearnedContinuation(
    &knowledge.ObservationTrace{Stimulus: [][]knowledge.NodeID{sourcePopulation}},
    8, eventTime,
)
```

Input budgets are validated before cloning; all referenced units must exist.
The runtime lock holds the whole preview at one revision. Retired runtimes and
historical dynamics contracts are rejected. An absent/ambiguous/weak forecast,
undecodable spelling, or exhausted budget remains an explicit non-complete
status. `Spelling` is absent on steps that did not attempt decoding.

| Bound | Value |
|---|---:|
| Input frames, across stimulus/context/data | 32 |
| Members per input or selected output population | 64 |
| Forecast steps, including learned termination | 32 |
| Final output Unicode scalars | 512 |
| Scalars per learned spelling piece | Existing codec limit 64 |

The response retains identity, revision, event time, `hypothetical: true`, and
bounded selected-population summaries. Alternative populations are counted,
not copied. An oversized selected population is rejected without truncating it
into an apparent complete population. `MinConfidence` is the weakest individual
forecast, **not a joint probability or semantic confidence**. The 0.5 gate is
not a calibration claim. A completed hypothesis can still express a wrong
underlying state.

`HorizonEngine.PulseWithContinuationPreview(input, maxSteps)` holds one owner
lease across the real Pulse and preview. It seeds only the already-selected
native Answer and refuses an uninterpreted or weak native forecast. Native
Answer, activation, and real Pulse state remain identical to ordinary Pulse.
This typed API does not enable automatic CLI/Bridge language replies or training.

## Controlled composition evidence

The test teacher provides six spelling pieces: `Bola`, `Gelas`, a space,
`diam.`, `jatuh.`, and the empty terminator. Eight **two-frame** transitions are
taught six times each. Context populations independently gate the initial noun
and the later motion piece; a shared space connects the local transitions.
No complete sentence or joint object/motion context is taught.

All four combinations (`Bola diam.`, `Bola jatuh.`, `Gelas diam.`, `Gelas jatuh.`)
complete on three numerical-geometry/allocation seeds: **12/12 controlled cases**.
Expected text is computed only after the API returns. The teacher supplied the
factorization and context organization. This is a small compositional I/O test,
not held-out actor/role understanding, grammar induction, a physical observation,
or a replacement for the failed semantic-generalization protocols.

Additional evidence:

- Zero transition lessons give unknown; one round remains below the confidence
  gate; six rounds complete. Missing, ambiguous and expired contexts stop.
- Hidden/forged adapter annotations give identical previews. Canonical knowledge
  contains no quoted teacher words or complete sentences.
- Zero spelling conductance while retaining coordinates, connections and history
  prevents realization. Separately removing temporal patterns while spelling
  still works prevents continuation.
- Learned repeated populations produce `haha` without a cycle-detection shortcut.
  Unterminated loops, scalar overflow and oversized populations stop at their
  declared budgets, without final partial text.
- Genesis replay, revision-zero recovery, checkpoint suffix recovery and full
  checkpoint restoration preserve the complete pre-thought continuation and
  preview **exactly**. Repeated/concurrent owner reads create no evidence.
- Actual v2 text input through the core facade produces learned pieces `Tíva`
  and ` 🌌` from the native selected state, with identical Pulse/preview results
  under original, hidden and forged labels. Weak input forecasts remain uncertain.

The three small pre-query fixtures have 167/170/173 units, 1,582 connections,
32 pattern channels, and respectively 1,084,166 / 1,087,653 / 1,091,165 pretty
canonical JSON bytes. These are whole synthetic fixture sizes, not heap,
checkpoint size, per-word cost, latency or throughput. Preview reads add no
structure. The orthographic footprint remains substantial for six pieces.

## Reproducibility and existing roundoff limitation

`learned-continuation-v1` uses total synapse ordering (target, polarity and
existing channel identity) in its **new hypothetical projection only**.
Historical `advanceHypothetical` callers retain their old ordering and law;
the actual recurrent thinking path is unchanged. Existing frozen fixtures stay
readable. No new checkpoint schema, checksum, journal command or automatic
migration is required.

The frozen [v1 fixture](testdata/learned_continuation_v1) pins 25 numerical
journal commands, canonical state and the exact detached readout `na lu`.
It contains no user's Brain and does not claim semantic knowledge.

The new recovery probe also exposed a pre-existing limitation: actual native
propagation sorts opposing connections only by target. Canonical restoration
can change the order of equal-target floating additions. The unmodified PR #45
parent reproduces `0.1773466423623914` versus `0.17734664236239145` after the next
real thought **without calling a preview**. This remains open; no bitwise native
next-thought guarantee is claimed. Only that inherited next-transition test
allows relative floating roundoff up to `1e-14`, while requiring identical
structure, populations, IDs, counts and timestamps. Pre-thought recovery,
preview results and before/after-preview state checks remain exact. Longer-run
effects and a versioned repair to native propagation require separate work.

See [diagnostics and the parent reproducer](testdata/learned_continuation_diagnostics).
Local validation covers root Go tests, vet/build, seven-package race tests
(including activation), and live Bridge smoke. Monitor source is unchanged;
exact-head validate/monitor CI is recorded on the draft PR. Separate nested Go
modules remain outside root-module testing.

## Remaining work

Context factorization, latent concept binding, unseen actor/role transfer,
productive natural grammar, vocabulary acquisition, multimodal meaning and
large-scale continual learning remain unproven. Prefix resimulation, pattern
scans, native candidate allocation and codec scans still depend on graph size;
bounded output is not a bound on global compute or temporary memory. H19/H20
indexing/persistence work and the original codec footprint remain open.
No personal Brain was trained/migrated, no main curriculum was run, and no
service or physical device was deployed.
