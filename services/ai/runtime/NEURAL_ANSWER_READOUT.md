# Neural forecast readout and language boundary

P0 neural knowledge remains **RED / OPEN**. This step fixes a mismatch between
the selected neural forecast and the typed answer. It does not add a trained
language decoder or improve learned semantic generalization.

## Problem and contract

`ToCognitiveInterpretation` previously copied general activity ranking into
`Answer.NodeIDs`, although its confidence already described the distinct
`PredictionAssessment` selection. In the frozen ISF001 example for final
location, the selected forecast was population `[154]`; the activity ranking
placed it tenth. A caller reading the first generic answer ID therefore read
another population even when the native forecast selected the trained target.

| Surface | Contract now |
| --- | --- |
| `State.ActiveNodeIDs`, `Interpretation.RankedNodeIDs` | Unchanged general activity, for diagnostics. |
| `Answer.NodeIDs` | Detached copy of the population already selected by `PredictionAssessment`; every member is retained. |
| `Answer.Status` | `neural-forecast` for a native selected population; otherwise `uninterpreted`, with no activity fallback. |
| `Answer.Confidence`, `Uncertainty`, `PredictionAssessment` | Existing values and laws, unchanged. A candidate remains uncertain; no new selection threshold is added. |
| `PulseWithLabels` second return value | Still labels the activity ranking for diagnostic compatibility. It is not language generation. |
| Normal CLI reply | `Jawaban bahasa belum tersedia.` until an actual language decoder exists. Activation and selected forecast IDs remain separate debug fields. |
| Bridge observation/sensor receipt `answer` | Numeric summary: `status`, `node_ids`, `scope`, `confidence`, `uncertainty`. Existing `active_node_ids` remains separate. No text, labels or alternatives list is added. |

An unknown or absent assessment cannot turn activation into a selected answer.
Historical outputs without an assessment retain their old scalar confidence and
uncertainty for compatible inquiry dynamics; their answer status is still
`uninterpreted`. Do not treat those scalars as confidence in a named answer.

Clients that used `Answer.NodeIDs` as the activity ranking must move that use to
`State.ActiveNodeIDs` or `Interpretation.RankedNodeIDs`. The Bridge `answer`
property is additive. It contains a numerical prediction, not an assertion that
a question was understood or a world event was correctly modeled.

This is a detached presentation change on the same Brain. There is no new answer
memory, lexical lookup, semantic parser, cognitive engine, learning law, journal
command or checkpoint schema. Internal inquiry still consumes the existing
uncertainty. Recurrence does not require an output. No user Brain is migrated or
retrained. The startup banner also stops claiming retired reasoning/understanding
engines and a ready language engine.

## Frozen experiment

The [fixture and provenance](testdata/neural_answer_readout) are from the updated
ISF001 archive, generated before this change on PR #40. All 42 numeric cases are
restored separately in three metadata conditions: normal, labels hidden, labels
permuted. Targets are used only for scoring after processing. No new training,
external selection, prompt-specific rule or replacement label is introduced.

The criterion `confidence >= 0.5` is inherited from the external test protocol;
it is not a serving threshold or a general calibration/accuracy claim.

| Split | Old activity first ID equals target | New candidate equals target with confidence >= 0.5 |
| --- | ---: | ---: |
| Seen cases | 0/18 | 18/18 |
| New actors | 0/18 | 0/18 |
| New condition combinations | 0/6 | 4/6 |

All three metadata conditions must produce these same counts. Every native
`CognitiveOutput`, including the original selected assessment and uncertainty,
must match the original experiment exactly; all 126 canonical post-case hashes
must also match. The test preserves the failed held-out expectations rather
than recategorizing them as success. This is the correction of a readout channel,
not an improvement to the native forecasts.

Additional regressions cover a multi-unit low-confidence population (no top-one
truncation or threshold), missing/unknown selection, independent detached copies,
checkpoint query continuation, journal suffix recovery, and the next text-free
thought. Authenticated managed Bridge tests cover the numeric receipt before and
after owner recovery, including v2 written grounding with labels removed; both
text and numeric sensor ingress keep untrained activity out of the answer. CLI
tests cover untrained input and an actual learned forecast without displaying
adapter labels as normal language replies.

The frozen numeric experiment does not establish label-independent text input
grounding, physical meaning, visual coordinates, word senses, new-actor
generalization or learned spelling. The separate
[written numeric grounding contract](NEURAL_WRITTEN_GROUNDING.md) remains in force.
A learned numeric character/phoneme decoder and generalization/causal grounding
remain subsequent P0 work. Main language training is still blocked.

The subsequent [experimental synaptic spelling codec](LEARNED_SPELLING_READOUT.md)
adds an explicitly supervised numerical I/O readout on the same Brain. It does
not automatically enable normal CLI language replies or close the semantic,
generalization or scaling gates here.

Reproduce the focused checks:

```sh
go test ./services/ai/runtime -run '^TestNeuralAnswerReadout' -count=1 -v
go test ./services/ai/cmd/horizon-cli ./services/ai/bridge -run 'Test(CLIAnswerDoesNot|UntrainedObservation|ManagedBridge)' -count=1 -v
```

Local validation passed on 2026-10-21 with Go 1.22.12: root-module
`go test ./...`, `go vet ./...`, `go build ./...`, six-package race checks
(knowledge, learning, runtime, bridge, core, horizon-cli), and `make bridge-smoke`.
The frozen fixture generator reproduced both compressed files byte for byte.
Monitor typechecking, all 37 Node tests and its production build passed with the
existing local dependency cache (Node 24.19.0 / pnpm 11.25.0); no monitor source
or dependency manifest changed. Nested Go modules are outside the root test
command's scope. CI on the draft PR records the exact final head.
