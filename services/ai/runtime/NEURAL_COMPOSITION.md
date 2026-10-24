# Native numerical composition

This increment integrates a bounded learned composition mechanism into the
root Go module. It is **disabled by default**. It does not migrate an existing
Brain or connect a physical actuator merely by building or starting Horizon.

The controlling contract is [FINAL_ARCHITECTURE_LOCK.md](../../../FINAL_ARCHITECTURE_LOCK.md).
All learned word projections, shared readout, positional gates, biases and
actuator-response parameters are dynamic excitatory/inhibitory connections in
the one canonical Brain. There is no lexical semantic table or external model.
Detached matrix values are temporary working arrays read from those synapses.
Internal coordinate populations also participate in the native activation and
recurrent thought path. Thinking can continue without an external observation.

## What this implementation learns

Numerical written observations contribute to six shared internal coordinates.
Learned positional gates make word order matter. The coordinates influence a
reference query, pointing contribution, desired physical state, and execution
gate. A shared affine motor-response model is learned from actual numerical
before/contact/after demonstrations. Reference selection compares the query
with currently perceived patches, and uses their **current** coordinates.

The first physical I/O contract is intentionally bounded: six written positions,
seven segmented patches, three synthetic appearance channels, physical
on/off state and one signal measurement, plus pointing presence/coordinates.
The physical output is contact x/y, normalized setpoint and force. This supplied
segmentation/output schema is engineering scaffolding; it is not learned vision,
unrestricted grammar, biological universal grammar, natural speech recognition,
or complete causal prediction of the whole room. Existing spelling hashes remain
an input codec, not meaning. Unknown written forms are explicitly refused.

Gradient-derived updates are bounded and preserve synaptic frequency, temporal
eligibility and historical timestamps. Practice support is capped below certainty;
it is not promoted into independent outcome evidence. Repeated appearances of
one written population in an utterance share an aggregated parameter update.
Contradictory supervision adapts the existing connections instead of replacing
the Brain or erasing a separate learned table.

## Explicit initialization and rollback

The typed runtime interface first requires the existing explicit
`written-neural-v2` migration. Composition never silently falls back to routing
on adapter word annotations. `PlanNeuralComposition(config)` returns a detached
configuration bound to the current revision and canonical/annotation hash.
`ApplyNeuralComposition(plan, newBackupPath, at)` verifies that reference,
reserves new backup destinations, syncs the complete old schema-13 checkpoint
and matching journal prefix, then commits the composition migration.

The managed facade provides `InitializeNeuralComposition(config, backupPath,
at)`. If written migration is still required, it performs that separately
journaled migration first, using `backupPath + ".written"` for its pre-change
copy. Both migrations retain existing neurons and learning. A later failure
does not undo an earlier durable migration. Inspect status and use the matching
backup pair when recovery to the earlier contract is intended.

CLI commands:

```text
composition status
composition initialize /new/path/pre-composition.json
composition learn /path/numerical-exposure.json
composition scene /path/current-scene.json
```

Initialization is an explicit opt-in administration operation.

For the production service, `HORIZON_COMPOSITION_INIT_BACKUP=/new/path/rollback.json`
requests the same initialization before asynchronous serving begins. An already
enabled Brain retains its persisted configuration on restart. Setting
`HORIZON_COMPOSITION_TRAINING=true` enables the teaching endpoint only when
`HORIZON_BRIDGE_TOKEN` is also configured. Neither setting is enabled by default.

The training
file is a `knowledge.CompositionExposure`: ID, source, episode, independence and
causal group, reliability, numerical written vectors and before-scene, teaching
motor, and measured after-scene. A scene is `knowledge.CompositionScene`, with
`observed_at`, `point: [x,y,present]` (presence is -1 or 1), and one to seven
patches. Each patch has `values: [appearance0,appearance1,appearance2,state,signal]`
and `xy: [x,y]`. Coordinates are finite in [-1,1]; state is -1 or 1.
`knowledge.EncodeObservation(word, "language").Values` supplies the existing
16-dimensional written codec when preparing numerical teaching data. It adds no
semantic classification. File input is limited to 1 MiB and rejects extra fields.

Ordinary CLI language input still uses `PulseWithLabels`; an enabled composition
readout is shown as a numerical proposal/status. It is not presented as generated
language, and does not execute a device.

To roll back to the pre-composition binary, restore its schema-13 checkpoint
with **its copied physical journal prefix**. Preserve the full newer journal;
the old binary cannot consume composition commands. A feature switch does not
remove learned synapses. Restoring memory also does not reverse physical actions.

## Transactions and versioned persistence

`LearnComposition(exposure, at)` validates/freezes all data and checks the word
budget and duplicate ID before any append or neural recruitment. One complete
redo command owns all grounding and learning, and exactly one revision. No
external operation or microstep can publish a partial lesson. The journal keeps
the actual causal provenance; repeating one teacher's curriculum does not make
it independent evidence.

Composition commands and all subsequent records use journal schema **4**, with
an explicit `composition_encoding`. Replay validates the complete serving-mode
timeline; activation requires its migration and cannot silently revert. Schemas
1–3 retain their original contracts and physical sensor intent/commit framing.
The sensor protocol remains one intent/commit transaction, including a supplied
simulator composition scene. Legacy sequence renumbering refuses schema 4.

An uncertain append or interrupted application retires the runtime. Recovery
uses a fresh private runtime through the existing supervisor and applies a
durable command once, including a command whose acknowledgment was lost.
Duplicate IDs remain rejected after restart. Replay never calls a device.

Enabled checkpoints use schema **14**, checksum-v9. They authenticate all neural
parameters, native activation/recurrent state, action/evidence continuation,
dedup index, composition configuration, and the current scene/written input.
The last scene and words are perceptual/recurrent process continuation, not
another store of learned knowledge. Missing or malformed coordinate topology is
rejected before modifying a restore target. A Brain-only graph is insufficient
for versioned serving. With composition enabled, `SaveBrain` writes a complete
checkpoint; `LoadBrain` recognizes that complete file only on a zero-revision,
writer-detached startup. Live replacement uses supervisor recovery.

Non-composition writers retain schema 13 and its byte/checksum laws. Historical
checkpoint schemas 4–13 remain readable; authenticated offset recovery retains
its existing minimum versions and framing requirements.

## Pulse, simulator, Bridge and thought

After initialization, ordinary written observations update the current numeric
input inside their existing observation transaction. `Pulse`, `PulseWithLabels`,
Bridge `/v1/observations` and native autonomous thought share the active owner and
neural readout. The readout is returned separately as `composition`; the existing
native forecast is not relabeled as a semantic answer. Long utterances continue
through the native observation path but receive a position-window refusal for
this bounded readout.

`ObserveCompositionScene(id, source, scene)` is a typed simulator adapter into
the existing numerical sensor boundary. It constructs physical measurement fields,
preserves coordinates and supplies `composition_scene`. Alternatively submit the
same bounded scene alongside an ordinary `modality: "simulator"` sensor request
at `/v1/sensor-observations`. Its timestamp must match the sensor observation.
This is an explicitly supplied simulator measurement contract, not an object
recognizer for unsegmented camera data. Stale/reversed scene updates are refused.

`ReadComposition(input, at)` is genuinely read-only: it never grounds unknown
words, initializes parameters, recruits neurons, changes timestamps, or alters
activation. It can be used for detached numerical evaluation inside an owner
lease. Status inspection returns copies. Missing/old scenes, unknown forms,
untrained controllers, insufficient reference separation, and unsupported
dimensions cannot create executable proposals. The default scene age is two
seconds and is bounded in the persisted configuration. Numerical confidence
scores are not calibrated probabilities of meaning or safety.

Bridge supervision is separately configured with `AllowCompositionTraining` and
a nonempty authentication token. Only then does authenticated
`POST /v1/composition/exposures` accept strict numerical supervision. It is off by
default; ordinary observations do not become supervised truth. No remote
migration or unauthenticated activation endpoint is added.

## Explicit numerical action boundary

Proposals never authorize themselves. An application can explicitly register a
`NumericMotorActuator` with the existing execution gateway and call
`HorizonEngine.RequestCompositionAction(proposal, actuatorID, requestID,
approver, expiry)`. This verifies the current committed revision, sensor time,
and agreement with the current neural/physical state, then uses `SafetyBoundary`
and the gateway. The composition facade requires the existing medium-risk
authorization path; its confidence does not replace approval.

The owner mutation lease spans validation, durable action binding and adapter
dispatch to prevent a stale Brain/scene handover. Adapters must respect the
provided context deadline, use request IDs for physical idempotency, and must
not synchronously call back into the same owner. The request's motor values are
detached copies. A frozen pre-action local-state prediction is authenticated
with the action binding so later observations can be compared with what was
predicted before execution. Actual outcomes continue through the existing
bound evidence/inquiry interfaces or explicit numerical teaching supervision.

The gateway reserves a numerical request ID before dispatch, so an error that
may represent a lost physical acknowledgment is not automatically retried.
This is not a guarantee of exactly-once delivery across arbitrary networks or
device restarts. Reconcile actual physical state before deciding on a new action.
No real actuator is configured by this increment.

## Validation and scope of evidence

The test-only teacher/simulator reproduces the archived 82-lesson/236-heldout
protocol. Teaching labels exist only in `_test.go`; native inference receives
present numerical input without gold targets or future sensor states.
Tests train through the public native operation from an empty Brain and explicit
migrations, including 56 silent motor demonstrations and 24 rounds of 82 lessons.
This differs from the earlier standalone prototype's initial 104-unit foundation.

Initial native results across seeds 7,17,47:

| Initialization | Seen | Held-out, known forms | Unknown forms accepted | Missing-pointing cases accepted |
| --- | ---: | ---: | ---: | ---: |
| Separated | 246/246 | 678/678 | 0/30 | 0/12 |
| Zero | 246/246 | 674/678 | 0/30 | 0/12 |
| Norm-matched random | 246/246 | 676/678 | 0/30 | 0/12 |

The stricter reference gate can abstain on some known inputs in the random
condition. These counts concern the bounded synthetic task. They do not prove
that a special universal-grammar seed is necessary or that ordinary camera
perception and unconstrained language are solved. Mixed backgrounds were seen
in nonlanguage demonstrations; fresh randomized layouts were held out.

Additional tests cover duplicate appearances, stale scenes, unknown forms,
malformed late fields, bounded growth, concurrent duplicate teaching, durable
lost acknowledgments, interrupted application, complete native restore, genesis
and suffix replay, the next thought, earlier native ordered memory, repeated
observation/thought, live facade restart, Bridge recovery, CLI opt-in/restart,
and numerical gateway authorization/idempotency. Retention evidence is bounded
to these tasks, not a claim that all long-term interference is solved.

Run root tests, vet/build, the existing race suite and `make bridge-smoke`.
The normal behavior suite uses all three seeds; race validation covers each
initialization at seed 7 and the dedicated concurrency/recovery cases, avoiding
redundant statistical repetitions under instrumentation.
`BenchmarkNativeCompositionReadout` measures read-only query cost; it includes
input freezing and native weight lookup but excludes training, checkpointing,
network and device latency. Report durable ingress/fsync separately rather
than treating that number as end-to-end response time. Nested Go modules are
outside root `go test ./...` and are unchanged by this increment.
