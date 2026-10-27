# HNL/2 — Extension for relational candle lessons

HNL/2 is internal research and remains a numeric teaching protocol. The native
numerical V5 learner is unchanged. HNL/1 bytes and existing question codes remain
compatible. A HNL/2 header adds C and D references after the legacy A/B fields;
all other bounded packet and response structures are unchanged. Replies echo
request version. The family magic remains 0x314c4e48; version is explicit uint16.

Question relation codes:

| Code | Question | Arity |
|---|---|---|
|1|Positive versus nonpositive|1|
|2|A greater than B|2|
|3|Same nonzero sign|2|
|4|A+B greater than C+D|4|
|5|Same nonzero direction of A-B and C-D|4|
|6|A and B equal at learned resolution|2|
|7|A+B and C+D equal at learned resolution|4|

The compiler routes these descriptions to distinct normalized question codes;
it never calculates an answer from the referenced values. Teach-only observed
labels are provided by the experiment author. Unknown/untrained questions stay
untrained. Equal is symmetric, sum operands are commutative, and SumEqual also
canonicalizes exchanging its two sides. Greater and SumGreater preserve order.
QuestionCode retains legacy coordinates 0..6 and adds C/D in 7..10; its 16D
shape and synaptic storage contract do not change.

Domain 3 in this experiment supplies 49 closed 4-hour OHLCV candles. Channels
0..3 are O,H,L,C in one shared positive affine coordinate system. Channel4 is
bounded log volume; channel5 is zero. Presence remains separate. Lag is ordinal,
not an inferred variable time interval. Context is in measurements/history, not
asset IDs. Training contrasts are hypothetical valid OHLC scenes made by the
teacher from old chart prototypes. They are not autonomous market simulations.

Equality is learned as a composition of existing discriminators. It is not an
exact floating-point equality instruction in production. A gap below learned N
resolution may return equal; precision_probe.py demonstrates a nonzero 1e-10
example. The raw decimal truth of saved historical cases is independently
checked. No finite-data result establishes unlimited identity or precision.

The native accepted-status rule is unchanged: observed-check error and a numeric
solver tolerance gate the readout. Neither confidence nor check RMSE is a
calibrated probability. These packets contain no trading authorization or motor
proposal. CRC is corruption detection; this is a trusted offline research API.

Implementations are in services/ai/teaching/{language,wire,bridge}.go. Teacher
truth, imaginary scenes and outcome labels are confined to *_test.go files and
are absent from the production executable. The complete patch includes HNL/1
and V5 foundations; the delta patch requires the exact HNL/1 final baseline.

## Use the active main engine

The core facade holds its existing RuntimeSupervisor owner lease throughout
each HNL operation. Teach takes the mutation lease; Ask takes a read lease.
No new Brain is created by these facade methods. Initialization remains
explicit and uses a fresh rollback-backup path:

```go
cfg := knowledge.DefaultCompositionConfig()
cfg.Version = knowledge.NeuralCompositionResolutionV5
err := engine.InitializeNeuralComposition(cfg, backupPath, time.Now().UTC())
reply, err := engine.ExchangeHNL(packetBytes)
sequence, err := engine.ObserveCompositionInput(perception, observedAt)
err = engine.SaveActiveBrain(memoryPath)
```

Handle errors before continuing. Existing journal/checkpoint lifecycle and
explicit recovery apply. The new contract is opt-in; default initialization
remains V1 and no existing trained memory is silently converted from V4.
Use the same existing active engine for all connected sensor adapters.

The offline research executable is built with
`go build -o /tmp/horizon-hnl ./cmd/horizon-hnl`. Its `-init` option explicitly
creates a fresh numerical V5 research session in a new directory. For an existing
validated checkpoint, use its matching native journal; do not overwrite a
serving Brain with a research model. Use `-request packet.hnl -reply reply.hnl`
and `-inspect -request reply.hnl` to exchange and inspect messages.

## Tests and historical lessons

`go test ./services/ai/teaching ./services/ai/core` covers the protocol, canonical
engine facade, persistence, learned composition, context, retention and boundary
checks. CI also race-tests teaching alongside the existing runtime/core tests.
The opt-in historical tests require `HORIZON_HNL_CURRICULUM` and
`HORIZON_HNL_PREVIOUS`, followed by `HORIZON_HNL_EQUALITY_PREVIOUS` for the equality
stage, plus an optional `HORIZON_HNL_RESULTS` output directory. They read an
explicitly supplied dataset and never download prices or connect to an exchange.

A saved small experiment evaluated already-observed relations using historical
data that had previously been inspected in development. This is evidence for
bounded numerical learning, not an untouched forecast benchmark. Teacher labels
and candidate operation families remain supplied by engineering. More precise
N distinctions, calibration, future-state learning and autonomous lesson design
require further validation.
