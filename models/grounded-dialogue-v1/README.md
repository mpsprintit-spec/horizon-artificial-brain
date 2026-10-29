# Trained contextual dialogue Brain (V6)

This opt-in research bundle publishes the actual result of the internal staged
language curriculum: 27 learned N circuits, 29 query heads and 521 total units.
The authenticated sequence-126 checkpoint, its matching committed journal and
neutral alphabet codec are shipped together. `start.json.gz` permits replay of
the learning journal from its initial checkpoint. Compression is lossless;
`manifest.json` records SHA-256 and size for both stored and unpacked files.

There is one active canonical Brain per engine. This is a fresh trained Brain,
not a migration or union of an existing V5 deployment's knowledge. Its native
numerical composition and learned spelling produce the answers; no LLM, teacher
answer lookup, or external model is used by the serving binary. The supervised
curriculum and its supplied meanings remain test-only code.

## Prepare and use

For interactive terminal chat, run `bash scripts/horizon-chat.sh`; see the
[Termux guide](../../docs/TERMUX_CHAT.md). The Go serving package embeds only the
final checkpoint, matching journal and codec. Frozen evaluation answers and the
replay starting checkpoint stay in the repository, outside the chat binary.
The Python preparation command below still extracts the complete evaluation bundle.

From the repository root, choose a **new, nonexistent** state directory. Its
parent must exist. The preparation command verifies all assets before writing
private state files and refuses an existing destination, including symlinks.

```bash
python3 scripts/prepare_grounded_dialogue.py --state /tmp/horizon-dialogue-state
go run ./cmd/horizon-dialogue \
  -brain /tmp/horizon-dialogue-state/language-brain.json \
  -codec /tmp/horizon-dialogue-state/codec.json \
  -request models/grounded-dialogue-v1/request.json
```

The example returns `Lampu itu sekarang menyala.` from the observed room in the
request. The request supplies measurements, their timestamps and raw utterance
history, not an expected answer. `-reply /tmp/dialogue-reply.json` saves the full
numeric and spelling trace. The CLI performs read-only inference.

To explicitly select this Brain for a new main-process state directory:

```bash
HORIZON_BRAIN_MEMORY_PATH=/tmp/horizon-dialogue-state/brain_memory.json \
HORIZON_CHECKPOINT_PATH=/tmp/horizon-dialogue-state/language-brain.json \
HORIZON_EVENT_LOG_PATH=/tmp/horizon-dialogue-state/events.jsonl \
go run ./services/ai
```

Production startup verifies checkpoint/journal identity and skips bootstrap for
the trained Brain. Core callers use `HorizonEngine.ReadGroundedDialogue(codec,
request)` through that engine's active supervisor. The ordinary Pulse path and
HTTP Bridge do not automatically become dialogue endpoints; applications must
pass observed room/history evidence to this typed API. Selecting these files
for an existing deployment requires an explicit operational choice; never pair
the checkpoint with another journal or overwrite an existing Brain's files.

## Reproduce validation

```bash
HORIZON_DIALOGUE_MODEL=/tmp/horizon-dialogue-state \
HORIZON_DIALOGUE_FINAL_CASES=/tmp/horizon-dialogue-state/evaluation.json \
go test ./services/ai/runtime \
  -run '^TestGroundedDialogue(FreshEvaluation|PersistenceContextAndSynapticAblation)$' -count=1

HORIZON_DIALOGUE_MODEL=/tmp/horizon-dialogue-state \
go test ./services/ai -run '^TestProductionGroundedDialogueBundle$' -count=1

HORIZON_DIALOGUE_MODEL=/tmp/horizon-dialogue-state \
go test -race ./services/ai/runtime \
  -run '^TestGroundedDialogueConcurrentCanonicalRead$' -count=1
```

CI runs these checks explicitly so trained-model tests cannot silently skip.
The production test restores via the main startup path, checks all 11 supported
and clarification categories, runs three cognition cycles at fixed hourly
timestamps, then checkpoints and restarts. This bounded check does not establish
long-term retention under arbitrary continuous learning. Evaluation also checks
that read-only answers do not mutate the Brain, that context changes reference,
that full journal replay preserves outputs, and that removing learned spelling
synapses removes the answer.

The frozen evaluation contains 480 supported questions, 16 ambiguous questions
requiring clarification, and 16 outside-scope probes. The published checkpoint
answers all 480 supported questions exactly with the expected state/reference,
and all 16 clarification cases correctly. Outside-scope probes make no
substantive claim; they are not counted as natural-language successes. These
measurements concern this finite synthetic task, not unrestricted language.

## Scope and limits

The room has three entities (lamp, fan and door), binary current/previous states
and supplied physical-kind coordinates. The codec contains 21 input forms and
17 output forms; generation has six positions. Lexical facets are pooled, so
full word order and subject/object roles are not represented. History is
supplied by the caller, not autonomously remembered dialogue. A greeting such
as `Halo` yields `Objek yang mana?`, which is not a natural greeting response.
Open vocabulary, free conversation, arbitrary world knowledge and trained V5
migration remain unsupported. Numeric precision gates come from alphabet code
spacing, not calibrated confidence probabilities.

See [the dialogue contract](../../services/ai/runtime/GROUNDED_DIALOGUE.md) and
[V6 composition](../../services/ai/runtime/METRIC_CONTEXT_V6.md) for the typed API
and numerical limits. The default composition contract remains unchanged.
