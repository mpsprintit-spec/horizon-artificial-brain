# Gate G transaction and neural ownership audit

## Baseline and integration ancestry

`BASELINE.json` contains fresh GitHub snapshots for PR60–65. Main was
`3a9e7ce3ffadff86c792050056ebea4eda48da0b`; Phase III was
`0df3f19a33bd7de69ed496293591d734331c5e51`. PR65 merge `45ecb98` and PR60
merge `32df59b` were already ancestors. Current main and the unmerged PR63
startup fix `1837968` were merged into this research branch without rewriting
PR61/62/64. PR64 exact-head CI42567905600 was verified successful. No existing
worktree, uncommitted benchmark, checkpoint, memory or user journal was replaced.

## Root causes

1. `runtime/journal_command.go` blanket-rejected production commands after native
   initialization, even at idle, while direct sensor/vector/import paths bypassed
   that guard. Removing it alone would allow observation/eligibility mutation in
   active trajectories and between factual trials.
2. Native parameters are signed action-value synapses with a fixed unit confidence
   and their own bounded observed-credit law. Generic world decay/plasticity can
   overwrite that contract. The actual mixed probe reached identical live/replay
   states but then failed checkpoint validation with `invalid native canonical
   parameter`. The original failure is retained in `development-mixed3.txt`.
3. A completed paired outcome adds a generated `native-contrast/<digest>` receipt.
   Checkpoint prefix verification reconstructed only input command receipts, so a
   checkpoint taken after pair learning could not authenticate its dedup index.
4. Pending action-trial mismatches were caught at outcome after activation had
   already changed. Their block and initial-state mismatch now fail at Begin.
5. Replay could previously detach an attached standalone live writer; a failed
   partially applied replay was not invariably retired. Recovery now requires a
   detached private runtime, prevalidates identity/schema/sequence and retires a
   partially failed replay. Supervisor publication remains exclusive.
6. Numeric learning payloads could alias native policy addresses through direct
   experience, consequence and spelling APIs. Central pre-append reference
   admission closes all three paths, including inline observation experiences.
   The positive world-learning control additionally exposed v1 policy edges
   being treated as competing world successors by observed association feedback;
   that feedback now uses the same canonical ownership fence.
7. An automatic numeric projection equal to the bias prototype recruited that
   internal unit by similarity, even without an explicit neural address. The
   legacy and current dynamics both failed the added adversarial test. Generic
   representation matching now excludes canonical native role addresses before
   candidate selection; world projection and exact checkpoint recovery remain
   tested. This repair follows the measuredaf3 code, whose failed resource result
   is retained rather than attributed to the repaired descendant.

## Entry points and transaction effects

| Journal type / entry | Canonical reads | Canonical writes / stable requirements | Receipt / sequence / replay |
|---|---|---|---|
| process / CognitiveProcess | N, prediction, context | activation; thresholds/activity/plasticity; cognitive/memory state for cognitive_process | event ID; one revision; schema3 or composition4 |
| observation / written envelope | source/time, existing projection, prediction | projection, activity, optional observed experience; temporal predecessor; inquiry selection | event ID + typed experience provenance; one revision before grounding |
| sensor_intent + sensor_commit | scoped numeric projection, prior sensor sample | projection, activity, sensory/temporal synapses, prediction; no episode/pair may be open | sample ID; intent SHA256; two physical records = one logical revision; schema3/4 |
| think / cognitive_think | current recurrent activity, learned world graph | recurrent state/plasticity, endogenous curiosity and memory dynamics; no synthetic observed evidence | recorded event time/cycles; one revision; no external output/dispatch required |
| learn | recorded observed experience/provenance | canonical learning paths/patterns, distributed synapses | existing experience IDs; one revision |
| inquiry_outcome | existing authorized action binding and pre-action prediction | observed evidence, grounding, prediction feedback, inquiry consequence and policy | input event ID + source/request evidence dedup; one atomic revision |
| outcome | registered action binding, explicit observed evidence | evidence/promotion and eligible canonical synapses | existing outcome/evidence single-use contract; one revision |
| consequence | eligible world synapses, observed consequence provenance | canonical consequence/inquiry learning | existing typed consequence contract; one revision |
| action_binding | world targets and request identity | adapter action provenance only; cannot bind internal policy units | exact repeat idempotent, changed targets rejected; one new revision |
| vector_ground | normalized numeric values and current projection | canonical numeric projection | one revision; no sensor predecessor substitution |
| knowledge_import | frozen validated source document | existing ordered exposure/learning microsteps; no native episode/pair | complete bounded payload before append; one journaled revision; historical detached semantics retained |
| spelling_exposure | typed numeric/written exposure | existing canonical spelling synapses | existing exposure ID/contract; one revision |
| composition_exposure / composition_perception | explicitly enabled contract and numerical scene/sequence | same canonical composition synapses and runtime continuation | existing exposure/perception ID, source/time; schema4; native idle only |
| native_control_command | canonical policy synapses and same activation engine | existing init/episode/step/observed-credit/discard transition | schema14; command digest; outcome source/ID single use; consumed factual-pair receipt |
| written/composition migration, surface rebinding | explicit reference revision/hash | serving contract/projections, with backup transaction | unchanged schema3/4; refused after native initialization, before migration backup/mutation |
| cognitive_model_command | frozen historical model contract | only its historical opt-in path without native | mutually exclusive with native; schema5–13; never selected by recurring native thought |

All neural mutators use `r.mu`; episode admission runs before both durable append
and the first canonical mutation. Direct detached grounding, temporal learning,
claim promotion and inquiry planning/completion honor episode ownership too.
Read-only inspection, detached inquiry prediction capture and inference do not
change a revision. Checkpoint reads all continuation state under `r.mu`; migration
holds that same owner lock. There is no new persistent queue or phase controller.
`knowledge.NativeAdmission` reads only constant-size metadata without graph/episode
cloning. Pending contrast takes precedence over the episode label; both block
external mutation. A stopped episode remains exclusive until observed settlement.

## One N and parameter ownership

The Brain, activation engine, learning unit and DNF retain their original shared
Brain pointer. Native Begin without an explicit observation reads current production
activity directly; no second state is projected or substituted. Each native decision
records the previous operation's state. The next real sensor update uses the learned
canonical graph. Tests compare the complete graph and continuation before/after
restart, not only the chosen operation or text output.

`NativeReservedNodesLocked` is a transient address-only ownership fence over the
existing bias/feature/effector nodes. Hot graph traversals use allocation-free
`NativeOwnershipLocked`: a pointer to those canonical role addresses and their
bounds, with exact membership checks for noncontiguous historical addresses.
Callers hold the Brain lock throughout use. There are no copied weights,
examples, episodes, learned caches or answers. An unconfigured Brain retains
the original fast path without membership lookups.
World propagation/curiosity cannot recruit the internal policy units as sensory
relays. Generic activity, prediction, observed feedback, consequence, sparse
plasticity and passive memory do not apply their incompatible weight/confidence
law to policy parameters. Those parameters still learn in their existing native
observed-credit method in the same registry. World synapses continue learning.
External stimuli/context/action bindings cannot inject or authorize native policy
units. Direct learning sequences, context/data populations, consequences and
spelling populations receive the same pre-append address validation. Explicit
NativeObservation retains the same anti-injection guard.

## Unified persistence and recovery

No new persistence schema is necessary: canonical7/checkpoint20/native WAL14,
production WAL3 and composition WAL4 already encode all of these states. Each
record keeps its existing contract rather than globally upgrading a mixed file.
`ReadEventLog` validates the physical stream and converts committed sensor pairs
into a single logical process event. All command times and neural samples are
recorded. Native outcome learning is one deterministic transaction, not one event
per synapse. Explicit observed source/ID/trace provenance remains mandatory.

The canonical checksum includes Brain/schema/identity/sequence, activation,
cognitive state, inquiry state, predecessor/time, processed IDs, journal prefix,
surface annotations, evidence, bindings and composition continuation. Checkpoint
write time, telemetry/inspection fences, derived caches, private replay flag and
locks are excluded; none is a neural input. A restore uses a unique temporary
file and refuses overwriting an already configured native runtime. Recovery loads
into a fresh unpublished runtime and reconnects its engines to that canonical N.

`journalProcessedIndex` reconstructs paired receipts on the second committed
observed outcome, retaining pending/discard semantics and exact receipt sequence.
The entire journal prefix is still authenticated; no history is truncated. A bad
or incomplete record fails closed. Explicit existing orphan-sensor recovery makes
and syncs a complete backup before removing only the verified uncommitted tail.

Historical valid native1/2/3 and production fixtures remain on their original
readers/laws. Invalid second-trial starts or unsafe historically interleaved active
trajectories are now rejected, not silently learned or repaired. There is no
V5→native migration or automatic serving migration. Downgrade cannot write a native
checkpoint to an older schema. Rollback uses the pre-opt-in checkpoint with the
matching prefix and a compatible pinned binary, retaining full later evidence.

## Physical boundary and legacy isolation

Native operations never call an actuator, grant a capability or convert a forecast
into observed evidence. The execution gateway still owns risk/authorization,
request ID and dispatch idempotency. A real gateway with a counting mock runs one
live authorized dispatch. Genesis and checkpoint-suffix replay add zero dispatches;
a fabricated request from an internal choice is rejected. These tests do not claim
new cross-device exactly-once delivery guarantees beyond the existing gateway.

The active runtime dependency closure excludes thinking/understanding/decision/
reasoning engines and execution dispatch. The facade's active Pulse/Think methods
are checked separately, since compatibility field/types elsewhere in `core` remain
legal. Pulse and recurrent thought run with the legacy Decision pointer absent.
V5 is not initialized by native opt-in; production startup and default controller
selection remain unchanged. A research controller is not promoted as default by
this integration.
