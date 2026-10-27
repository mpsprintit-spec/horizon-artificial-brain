# HNL/1 — Horizon Numeric Language (internal research)

For the HNL/2 extension, active-engine API and integration instructions, see
[HNL2.md](HNL2.md). Existing HNL/1 packet bytes and numerical query identities
remain compatible.

HNL/1 is a deliberately small bidirectional teaching language between a lesson
author and Horizon's existing numerical V5 learner. A teacher can be a person,
a deterministic experiment script, or ValTherOn preparing observations. It is
not a shared native language of two architectures, a transfer of LLM weights,
or a claim that numeric identifiers alone create understanding. There is no
model-provider call or outside fitted model in this implementation.

## The common contract

Every request is a binary packet containing numeric fields only:

| Item | Meaning |
|---|---|
| version, kind | 1; Teach=1, Ask=2, Answer=3 |
| ID, source, stream | unsigned 64-bit provenance; excluded from neural input |
| at | trusted-envelope time, Unix nanoseconds |
| question | domain, relation opcode, one or two `(lag, channel)` references |
| examples | episode identity, observation time, fit/check/probe role, samples |
| evidence | signed observed result and time; Teach only |
| reply | native value, uncertainty/error, N contributions or learning diagnosis |

The grammar has three question forms: `Sign(A)` (positive versus nonpositive),
`Greater(A,B)` (strict comparison) and `SameSign(A,B)` (two nonzero values with
the same sign). Labels are +1 for true and -1 for false; exact-zero semantics
are supplied by the teacher, not inferred from the opcode. Domain is a supplied coordinate/normalization contract, not a
learned ontology. The author supplies actual examples and their observed labels.
An opcode is encoded into a distinct question vector; **the bridge never
executes the named relation to generate a prediction**. A blank Brain is not
able to answer a relation merely because its opcode was received.

Each measurement is a supplied bounded binary64 value on one of six channels,
at one of 64 lag positions. Lag zero means latest, not an inferred duration;
irregular real-time sampling must be handled by the domain adapter. An explicit
presence flag distinguishes observed zero from missing input. A measurement's
units, normalization and operational meaning must be stable within its domain.
The wire carries ordered observation references, not an arbitrary causal graph.
The current compiler preserves the V5 input limit; it does not lift that limit.

Commutative `SameSign` operands and sample ordering are canonicalized. Ordered
`Greater` operands retain their order. This is syntactic equivalence only;
empirical semantic reuse remains the learner's responsibility. Source/stream IDs
are provenance, so contextual information that can change the correct answer
must be supplied in measured channels or time lags, not hidden in those IDs.

## Learning and replying

`CompileLesson` produces the native `NResolutionLesson`: 4..64 fit observations
and 4..64 check observations, with disjoint episode identities. Native learning
chooses among readout, resolution, new discriminator/boundary and learned product
candidates using its existing diagnosis and retention checks. All fitted
parameters stay in canonical dynamic synapses. HNL stores no learned model.

The legacy scene/contact envelope is a constant scaffold and a no-op motor;
it contains no target. Signed outcomes exist only in the teaching evidence.
`CompileAsk` rejects any outcome and produces exactly the numerical input
consumed by native `ReadComposition`. Replies return the native value and check
RMSE, plus each N's code, resolution and contribution. Teaching replies include
the transaction's detached diagnosis, candidate errors and learned-N count.
Receipt extraction occurs under the native transaction lock, so concurrent
lessons cannot exchange their diagnoses. Native default behavior is preserved.

Question relation codes: 1 sign, 2 greater, 3 same-sign. Native response status:
1 untrained, 2 uncertain, 3 numeric prediction, 4 teaching receipt. Diagnosis
codes: 0 none, 1 insufficient evidence, 2 readout, 3 resolution, 4 direction,
5 boundary, 6 relation, 7 conflicting observations. These are diagnostic
categories supplied by engineering, not discoveries of physical causes.
RMSE and acceptance are not calibrated probabilities or action authorization.

Packets are little-endian: magic uint32 `0x314c4e48`, version uint16, kind
uint8, four 64-bit envelope fields (ID/source/stream/at), question (domain
uint16, relation uint8, two pairs of uint8), uint16 example count, each example
(role uint8, episode uint64, observation int64, sample count uint8, samples
`lag uint8, present uint8, six float64`, outcome-present uint8 followed by
optional float64+int64), reply-present uint8 and optional numeric reply, CRC32
uint32. `wire.go` is authoritative for reply layout. Size is capped at 1 MiB;
all counts are bounded before allocations. CRC is accidental-corruption
detection, not authentication or encryption. This is an offline trusted-teacher
protocol, not a public network endpoint.

The bridge rejects future evidence and packets ahead of its trusted simulator
clock, malformed references, duplicate episodes/lags, corrupt packets and
unknown operations. Observed labels are still trusted teacher assertions; the
protocol does not establish their truth or statistical independence. Native WAL
provides exactly-once lesson IDs; an unknown delivery result requires journal
recovery before resubmitting. Ask has no side effects on Brain knowledge.

## Reusable interface

```go
packet := teaching.Packet{
    Version: teaching.Version, Kind: teaching.Ask,
    ID: 12, Source: 1, Stream: 1, At: observed.UnixNano(),
    Question: teaching.Question{Domain: 1, Relation: teaching.SameSign,
        A: teaching.Ref{Lag: 0, Channel: 0},
        B: teaching.Ref{Lag: 0, Channel: 1}},
    Examples: []teaching.Example{{Role: teaching.Probe, Episode: 20,
        ObservedAt: observed.UnixNano(), Samples: []teaching.Sample{{
            Present: true, Values: [6]float64{0.31, -0.27}}}}},
}
wire, err := teaching.Marshal(packet)
reply, err := (teaching.Bridge{Runtime: isolatedV5}).Exchange(wire)
decoded, err := teaching.Unmarshal(reply)
// Inspect decoded.Reply, including uncertainty and contributing N units.
```

The offline executable `cmd/horizon-hnl` creates a fresh session with
`-session NEW_DIR -init`. Exchange one saved binary packet using
`-session DIR -request lesson.hnl -reply receipt.hnl`. Decode a packet using
`-inspect -request receipt.hnl`. A trusted replay clock can be supplied with
`-clock-ns UNIX_NANOSECONDS`; changing packet timestamps cannot advance it.
The process exclusively locks its native journal and recovers checkpoint plus
committed journal before exchange. Use an isolated directory and the explicitly
enabled V5 research Brain. No production Brain migration is provided.

## What is actually tested

Two fresh Brains receive the **same four labelled lessons once**, in different
orders: sign of x, finer x distinction, sign of y, and the relation of x/y.
The reversed-condition Brain sees the relation before either prerequisite.
Both use the same native learning algorithm, input representation and number
of lessons. Held-out relation examples are read without labels. The relation
is then revisited in the reversed condition to test whether prerequisites
unlock it. This is a narrow, hand-designed, deterministic synthetic curriculum,
not a blinded statistical study or market prediction experiment.

Other checks cover packet canonicalization, time/presence, target isolation,
malformed wire/CRC/bounds, trusted clocks, duplicate lessons, nuisance invariance,
read-only inference, equality with direct native inference, checkpoint and full
native WAL replay. A separate temporal comparison lesson and its inverse test
ordered references and reuse of one learned discriminator across two questions.
Existing knowledge/runtime regressions must still pass.

A curriculum effect must not be credited to binary encoding. HNL is the
communication/grounding contract; the ordering experiment separately tests the
V5 learner's prerequisite dependency. Broader semantics, automatic discovery
of a grammar, general causal diagnosis, future-state rollouts and real-chart
improvement remain open. New concepts and operation families require evidence,
versioned contracts and further validation.
