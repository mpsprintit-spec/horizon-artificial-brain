# Learning infrastructure audit

## Tujuan

Identify actual reusable mechanisms and demonstrable bottlenecks, while
preserving FINAL_ARCHITECTURE_LOCK.md and PR67's canonical ownership contract.

## Implementasi: existing mechanisms

Paths below are relative to services/ai/.

| Mechanism / source function | Accepted input | Learned in canonical N | Engineer/adapter choices | Change needed for another task? |
| --- | --- | --- | --- | --- |
| learning/bootstrap.go Experience; learning/ordered_experience.go LearnExperienceWithDynamics | Ordered surface streams or numeric population frames, independent context/data streams, reliability/evidence metadata | Pair synapses with dynamic confidence/eligibility/history; canonical PatternSynapse population traces, conductance, confidence/frequency/evidence | Encoding, stream boundaries, exposure/reliability, teacher's presented observations | Usually data only. Legacy LearnExperienceWithEncoding ignores numeric-only input; current runtime deliberately calls versioned dynamics, not this legacy helper |
| runtime/brain.go LearnExperience; runtime/ordered_experience.go | Frozen validated Experience | Same Brain as observation/thought; one journal revision | Source, timestamp; ExperienceID is provenance, NOT universal API dedup | Data only; repeated explicit experience calls are repeated exposures. Do not silently assume retry idempotency |
| runtime/orchestrator.go ProcessObservation; observation_command.go processObservationCommand | Text/tokens/context/data plus optional inline experience | Grounded populations, observed plasticity, recurrent prediction feedback, ordered associations; inline experience shares observation transaction | Input adapter, modality; bounded token/numeric encoding | Data only, one orchestrator references runtime, no separate Brain |
| runtime/sensor_observation.go ProcessSensorObservation; vector_grounding.go GroundVectorAt | Bounded numeric channels, fields, coordinates/delta or vector | Distributed populations, observed activation/prediction/transition plasticity; vector grounding alone does not invent a sensory transition | Numeric feature extraction, calibration, noise tolerance | Data/adapter only; hardware-specific decoding belongs outside cognition |
| runtime/brain.go CognitiveProcess / CognitiveThink / ThinkAt; activation/ordered.go | Current persistent activity and optional new observed trace | Recurrent activation; observed feedback plasticity only when real evidence arrives | Cycle/resource budgets, reset/episode control | No task-specific thinking code required; thought alone is not empirical evidence |
| knowledge/native_control.go NativePolicy/LearnNativeOutcome; runtime/native_control.go ApplyNativeControl | Current activity or explicit observed numeric state; generic operation set; bound observed outcome | Learned operation values in ordinary signed synapses from feature/bias cells to effector populations in SAME N; trajectory addresses/history in BrainState | Primitive gains/costs, exploration, budgets; adapter defines measured outcome | New experience normally data only; policy can only learn distinctions visible to its encoder |
| knowledge/native_relational.go nativeRelationalFeatures | Local active graph/activity | Readout projection weights are canonical synapses; encoder itself is fixed numerical mechanism | Fixed basis and pooling invariance | Sign information loss proven by Phase III counterexample; minimal versioned extension implemented here |
| knowledge/native_action_contrast.go learnNativeActionContrast | Two actually executed matched initial states/actions, observed rewards and common continuation | Initial action advantage in canonical parameter edges | Environment matching; MatchKey does not prove hidden-world equality | Reusable learning law; do not use hypothetical comparator |
| knowledge/neural_composition.go LearnCompositionAt; neural_composition_sequence.go; runtime/neural_composition.go | Numeric scene/word/point vectors, demonstrated numeric effects/sequences | Bank neurons and adaptive signed parameter synapses in same Brain | Fixed numeric I/O dimensions, environmental demonstrations, loss/budgets | Flexible within supported interface; not an unrestricted arbitrary-graph policy. Enabling/configuring composition is versioned and opt-in |
| CLI neural_composition.go; bridge/server.go and neural_composition.go | Existing observation, inquiry, sensor/composition transport | Delegates to canonical runtime | Transport surface and authorization | No direct native episode training command currently exposed; existing Go API is sufficient for this experiment, no new wrapper added |
| runtime/persistence.go, journal_command.go, checkpoint_journal_rebuild.go | Validated transaction/checkpoint | One canonical state, exact-prefix suffix replay | Explicit versions, fsync/rename durability, opt-in controller | No new schema needed for the sign basis; encoder identifier is explicit compatibility boundary |

PatternSynapse is existing canonical serialized neural population-pattern
conductance/trace state, not a new semantic answer database introduced here.
Its ordered matching is a mechanism limit; it should not be confused with learned
general graph computation. Current bootstrap surface traces can have singleton
anchors; the continual experiment uses distributed numeric populations instead.

## Pengujian: representation diagnosis

Old local-relational-v1 pools activity and squared activity with reciprocal /
nonreciprocal connectivity, normalizes by fan-out, and uses absolute edge weights.
The Phase III TestNativeRelationalInhibitoryCollisionDocumentsLimit stays intact:
flipping an edge's Inhibitory flag gives identical policy readout but different
actual next state. More readout training cannot recover an erased observable.

V2 preserves those exact four channels and adds their signed counterparts.
For active node activity v, outgoing magnitude m, reciprocal magnitude r,
signed outgoing sum s and reciprocal signed sum t, the added channels are
v(s-t)/m, v²(s-t)/m, vt/m, v²t/m (zero when m=0), with the same fan-out scaling.
Accumulation order is deterministic; the same structural traversal/read budget
is used. No source/target semantic IDs or preferred operation are encoded.

| Distinction | Old encoder | V2 / remaining limit | Decision |
| --- | --- | --- | --- |
| Excitation vs inhibition | Collides despite different transitions | Signed moments expose the missing observable | Repair justified; preserve old version |
| Temporal eligibility/history with equal instantaneous graph/activity | Discarded | Still discarded; executable negative control | No speculative temporal channels: A/B already use existing ordered dynamics; future history-dependent policy needs a separately justified representation extension |
| Context preserved in activation/degree/reciprocity | Sometimes distinguishable | Same limited local pooling, plus signs | A/B contextual learning does not require a new encoder |
| Two context populations swapped under a graph automorphism | May collide | Still invariant without distinguishing structural/activity evidence | Cannot promise arbitrary role/context separation; keep negative cases |
| Different graphs with same local moments | Collide | Some sign differences separated; many same-moment structures still collide | No universal relational expressivity claim |

## Admission / continuity

| Native state | Observation/experience/thought | Native commands | Checkpoint |
| --- | --- | --- | --- |
| Uninitialized | Existing contracts | Explicit init | Yes |
| Idle/completed and settled | Allowed through shared journal | Begin | Yes |
| Active or stopped awaiting outcome | Rejected before append/mutation | Valid step/bound observed outcome | Yes, lock-consistent |
| Pending factual comparator | World/parameter mutation rejected | Matched trial or explicit discard | Yes, preserves pending state |
| Recovery required | Writes rejected | Writes rejected | Existing guarded persistence contract |

This is episode/provenance consistency, not a task-specific cognitive rule.
Native operations change the SAME activation and read SAME world synapses;
they do not execute an actuator or make their prediction an observation.
Production numeric input cannot address policy cells (command_admission.go).

## Keputusan

Reuse existing learning APIs. Only demonstrated sign loss requires production
code repair in this unit. Keep v1/no-encoder bit-compatible; new V2 opt-in
identifier changes feature count from four to eight, with no automatic migration.
Canonical schema 7/native checkpoint 20/journal 14 already carry arbitrary role
addresses/configuration; no schema bump or destructive model conversion.
V3 action-credit model remains V3 because its learning law is unchanged.

## Langkah berikutnya

Execute PROTOCOL.md after source freeze. Retain the exact old limitations and
G8 failure. Use actual training evidence to distinguish curriculum needs from
representation or integration failure; don't infer a missing generic API from
a small training failure.
