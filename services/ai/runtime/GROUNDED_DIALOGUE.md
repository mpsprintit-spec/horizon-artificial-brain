# Grounded dialogue: internal V6 research

This opt-in adapter uses the existing active Brain and signed canonical
synapses. The ordinary Pulse path and default contract are unchanged. Start
with a new explicitly initialized V6 research Brain; no trained V5 migration
is provided by this experiment.

## Learned steps and supplied structure

Input word forms have neutral alphabet coordinates. The codec contains no
semantic classes, answers, grammar weights or complete reply strings. Teacher
meaning labels and example utterances are in test-only curriculum code.

The Brain learns seven lexical readouts, reference selection, temporal choice,
state/claim agreement, compatibility and six output motor slots. The adapter
replays raw utterance history through the learned reference readout. Selected
referent and time codes address measured room coordinates. Coordinate access,
binary sensors, their physical kind and the meaning-field layout are supplied
engineering structure, not autonomous variable discovery.

Every emitted token is chosen by a native numerical head. Its Unicode form is
decoded from the existing synaptic spelling representation. Surface labels and
teacher answer functions cannot supply a fallback. The only string processing
after that selection is capitalization, spacing and punctuation.

All operations run under one runtime revision. The facade uses the existing
RuntimeSupervisor owner. No additional Brain, external model, network service
or execution gateway participates. Readout does not learn or allocate N.

## Teaching and precision

Teach physical scalar distinctions, lexical meanings, explicit references,
contextual references, unresolved references, agreement, negation, compatibility
and utterance production in stages. Corrections are selected using native
diagnosis and fit/check evidence; retained correct distinctions remain protected.

Zero-valued contextual gates need centered, scale-normalized interaction search.
A constant parent cannot add an identifiable distinction via a product on that
lesson. V6 readout fitting suppresses unidentifiable constant columns in both
activation and metric modes. Otherwise tiny inactive values can create excessive
coefficients, and irrelevant constant firing can block resolution refinement.
The V5 learner and readout contract are unchanged.

Acceptance uses numerical I/O resolution: one quarter of the smallest referent
code spacing, and one quarter of the output alphabet spacing. This is neither a
probability nor a trading confidence threshold. A failed numerical acceptance
produces no completed text. It is not overridden because a word looks plausible.

## Explicit limits

This curriculum covers three simulated room entities, current/previous binary
state, short questions, one principal referent and a single negation. The
seven lexical facets are pooled, so full word-order semantics, nested scope and
arbitrary subject/object binding are not represented. This is a known limitation,
not evidence for a learned universal grammar.

There are six fixed output positions and a supplied finite vocabulary. Teacher
grammar labels shape the learned production pattern; free linguistic style and
human-level dialogue are not claimed. Raw history and room snapshots are
provided by the caller, not automatic long-term conversational memory.

Limits remain 32 learned N circuits, 32 heads, 128 retained witnesses, 40 input
alphabet entries, eight input tokens and 16 history turns. Decoder vocabulary
populations and spelling bit coordinates are separate I/O structure, not one
semantic N allocated per word.

An unknown greeting can currently elicit an inappropriate generic room
clarification. Out-of-domain abstention must not be scored as natural-language
success. Future work needs richer intent, ordered role binding, expandable
context and adaptive production, with new held-out evaluation.

## Offline use

Use the saved matching V6 checkpoint and codec from the experiment:

```bash
go build -buildvcs=false -o /tmp/horizon-dialogue ./cmd/horizon-dialogue
/tmp/horizon-dialogue -brain /absolute/language-brain.json -codec /absolute/codec.json -request /absolute/request.json -reply /absolute/reply.json
```

The CLI performs inference only. The production executable does not contain
the test-only teacher lexicon or its realization function. Typed API:

```go
out, err := engine.ReadGroundedDialogue(codec, observedRequest)
```
