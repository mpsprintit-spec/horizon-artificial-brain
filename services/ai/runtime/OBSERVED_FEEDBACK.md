# Observation, simulation and outcome feedback (H12)

Serving uses `observed-feedback-v1`, retaining ordered observations, bounded
context, physical-time retention and local activity from the preceding contracts.
All learning operates on the existing synapses and pattern index of one Brain.
No semantic parser, external model, answer table or second knowledge store is
introduced.

## Evidence boundary

The previous observation path used recurrently propagated activity as the actual
state for coactivity and prediction-error learning. A cue could activate its old
predicted outcome and reinforce that connection before the next observation had
confirmed it. Internal thought could likewise increment synaptic confidence and
confirmation frequency while replacing the forecast used for later feedback.

Current stimulus population IDs are now retained separately from inferred
activity. Ordered processing and inference still operate normally. Activity
learning uses only the supplied stimulus populations. Context/data streams
continue to gate inference, but are not treated as independently observed next
outcomes. Propagated targets remain available in activation/prediction outputs;
their presence does not assert empirical confirmation.

The implicit temporal learner likewise uses the stimulus population alone;
context/data members cannot silently become observed transition outcomes.

`ObservationPrediction` holds one detached continuation boundary: actual source
population, observation time/episode, the forecast made at that observation, and
references to the eligible learned pattern channels. The next real observation
compares that forecast with direct input, credits its actual prior source, and
replaces the boundary. Thought cannot rewrite it or consume the outcome twice.
Episode changes/reset, backwards observation clocks and expiry prevent reuse of
invalid causal credit. Historical-mode observations clear newer pending credit.

Source eligibility uses the existing one-second half-life from the real source
observation, with a maximum feedback age of two minutes. The signed synapse
adjustment is `0.12 * error * abs(target_difference) * source_eligibility *
source_activity`; confidence receives half that signed change. Existing weight
bounds remain [0.05, 0.95]. Recent verified source eligibility is retained without
rewinding its clock. Negative evidence does not increment confirmation counts.

Only the best eligible learned cue/context channels receive pattern feedback;
other contexts retain their pattern strength/history. Weight changes by the
existing reconsolidation gain 0.12 times error and channel quality, with half that
change applied to confidence. Pattern structures and historical counts are not
deleted. These are explicit engineering laws and thresholds, not measured
biological constants or calibrated probabilities.

## Simulation, teaching and action outcomes

Internal thought continues to change recurrent activity, thresholds, usage,
curiosity and predictions on the same substrate. It applies passive retention
and elapsed-time eligibility decay, while supplying no new empirical coactivity,
synaptic confirmation count, or signed world-outcome error. CognitiveThink does
not overwrite the last real learning signal/priority with its internal mismatch.
Its output PredictionError remains an internal consistency discrepancy, not new
world evidence. Empty-input fallback follows the same distinction.

Explicit supervised Experience sequences are also external evidence. A supplied
unconditioned first successor weakens existing outgoing alternatives through
bounded feedback scaled by the supplied confidence and existing conductance.
Conditional/deeper-prefix counterevidence revises its matching full pattern,
preserving the shared context-free projection. All observed members of a
population are retained when deciding which outcomes are absent. Learning then
reinforces the supplied successor using the existing history/count mechanisms.
Pattern feedback is restricted to the matching learned ordered cue/context.
This avoids preserving an obsolete association solely through historical
frequency averaging; it does not freeze old connections or discard history.

Inquiry results compare the captured pre-action forecast against direct grounded
outcomes. Verified action-binding targets supply the causal source for action
feedback; an unrelated last observation does not receive that credit. Existing
context-free action/outcome traces are revised through their bound sources, and
signed consequence learning remains enabled. Journal replay reproduces this
numeric update without authorizing or executing another external action.

## Controlled evidence

On actual PR34 (`dadbd38e0e4aa06879c4b9c3d799199253efdcd9`), the sequential
observation probe learns 12 A→B episodes, then observes 12 A→C episodes. Each cue
and outcome is one second apart, and each episode has an explicit identity.
B still wins after the change: B=0.491743, C=0.487505. This reproduces the H12
failure on the current integration chain, with a smaller gap than the old audit.

| New experiences after change | New-law B | New-law C | C minus B |
|---:|---:|---:|---:|
| 1 | 0.455844 | 0.258276 | -0.197568 |
| 3 | 0.407587 | 0.354748 | -0.052840 |
| 5 | 0.368020 | 0.384778 | 0.016758 |
| 8 | 0.319311 | 0.418905 | 0.099594 |
| 12 | 0.268601 | 0.458461 | 0.189860 |

Twenty neuron-insertion permutations in each training direction (40 runs) switch
on new experience five, retain the new preference thereafter, and exceed a 0.15
margin by experience twelve. No-change controls retain their original mapping.
These are activation scores, not probabilities or a semantic-language benchmark.

The mechanism is tested with separate controls:

- A cue produces a nonzero inferred target in both old/new inference, but only
  the old activity path counts that unobserved target as empirical reinforcement.
- Removing the saved observed-feedback boundary while keeping new inference and
  activity accounting leaves the twelve-experience margin at -0.225799.
- Adding five internal thought cycles between each cue and outcome leaves the
  complete twelve-experience margin at 0.189860. Thought cannot fabricate extra
  observations or replace the frozen forecast.
- A still-valid old context keeps its B preference while the changed context
  favors C. Supervised reversal also passes in both directions and retains the
  still-valid context.
- Production numeric sensor observations at vector seeds 7/17/47 produce new
  outcome margins 0.556/0.568/0.562 after twelve changed episodes, without surface
  annotations. Population projections and feedback use the same Brain.

These controls support the identified mechanisms on these tasks. They do not
establish a universal optimal adaptation rate or show that self reinforcement
was the sole cause of every historical H12 case. Partial observations, ambiguous
outcomes, broad task interference and longer delays require further evaluation.
H13 uncertainty calibration and H23 independent generalization remain open.

## Persistence and recovery

Journal schema 3 records the new dynamics string; older readers reject it.
Earlier dynamics retain their original activity, thought and feedback behavior.
Canonical Brain v3 is unchanged. Checkpoint v11 additionally authenticates the
pending observation/forecast boundary; v10 and earlier cannot carry it.
Validation checks time/episode/source-trace agreement, neuron references, finite
values and pattern references before mutating the destination. All cache maps
are defensively copied. Anchored v7-v11 suffix recovery is supported, with the
existing restrictions on older detached imports retained.

The immutable `testdata/local_activity_v1` fixture was generated using Go 1.22.12
against actual PR34. It verifies complete state and the next historical thought
under local-activity-v1 from genesis and a v10 prefix. New-law tests checkpoint
between cue and outcome, insert thought, then compare full genesis/suffix/restart
state and subsequent feedback/thought. Malformed references, episode/time
inconsistency and downgrades are rejected before target mutation. Existing WAL,
duplicate, retirement and historical fixtures remain required.

Local validation passed with Go 1.22.12: root-module `go test ./...`,
`go vet ./...`, `go build ./...`, race tests for knowledge/learning/activation/
runtime/bridge/core/CLI, and `make bridge-smoke`. The root command does not cover
the repository's separate nested Go modules; this change does not modify them.
