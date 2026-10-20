# Local activity and retention (H09)

The local activity contract is retained by current `observed-feedback-v1`
serving (see [OBSERVED_FEEDBACK.md](OBSERVED_FEEDBACK.md)). It retains ordered observations, bounded
recurrent context and H08 physical-time passive retention on the same canonical
Brain. This change protects inactive pathways without a second memory store,
semantic rules, external model or fixed list of protected answers.

## Activity and error credit

The historical activity update subtracted weight/confidence from every unused
excitatory connection on each call. It also multiplied eligibility by 0.85 even
for empty activity at the same timestamp. Repeated unrelated observations could
therefore erase useful support despite little physical time passing.

For the new contract, `p` and `q` are clamped presynaptic/postsynaptic activity:

- Eligibility first uses the existing one-second physical half-life:
  `e_minus = e * 2^(-elapsed_seconds)`.
- Actual coactivity adds bounded evidence:
  `e_plus = e_minus + 0.15*p*q*(1-e_minus)`.
- If `p*q >= 0.05`, activity changes weight by `0.08*p*q*e_plus`, with the
  existing bounds [0.02, 0.95]. Excitatory confidence/frequency follow the existing
  active update; inhibitory connections retain their existing active behavior.
- An inactive or subthreshold connection receives no activity-based weight or
  confidence penalty. Subthreshold coactivity may accumulate eligibility.
  Empty updates forget eligibility only through elapsed time. The dedicated H08
  clock still ages weight/confidence/experience priority through passive time.

Prediction error has a separate locality condition: the connection's source
must have activity at least 0.05 in the compared predicted or actual state.
Otherwise a target's surprise cannot change that incoming connection solely
because it has residual eligibility or a stale `node.Activation`. Sources in the
prior prediction can still receive delayed credit. The existing positive and
negative error law and pattern reconsolidation remain enabled. Validated
consequences retain their signed eligible-path updates.

Activity, error and consequence updates under the new contract do not rewind
`LastEligibilityUpdate` on a backwards timestamp. Actual learning may still
change weights at the same timestamp; this is not a promise of learning
idempotence. The numeric rates/thresholds are explicit engineering choices,
not biological measurements or a proven optimal consolidation policy.

Protection applies to inactive paths at every experience count. Repetition
still supplies frequency/confidence; there is no binary threshold that freezes
an old connection. Relevant coactivity, source-supported negative error and
consequences can revise established pathways. H08 retention can weaken unused
knowledge over physical time without deleting its historical structure.

## Reproduction and controls

The controlled runtime task has 40 old mappings with two-member neural cue and
outcome populations. Three exposures train each source-to-outcome association.
A separate 40-mapping task starts untrained. Node insertion permutations use
seeds 7, 17 and 47. A fresh read-only neural probe measures the same synapses
without residual recurrent activity; these are activation scores, not calibrated
probabilities or a language-understanding benchmark.

On actual PR33 (`c6c7fc5ea36706d78e45064e31aee79ca23c557c`), 96 unrelated
`CognitiveProcess` observations over one second reduce mean old-target score
from 0.902720 to 0.016959. Forced choice against an untrained alternative remains
80/80, hiding that 0/80 target units reach the 0.25 support threshold.

| Condition | Old target mean | Old units >= 0.25 | New target mean | New units >= 0.25 |
|---|---:|---:|---:|---:|
| Trained old task, new task untrained | 0.902720 | 80/80 | 0 | 0/80 |
| PR33, 96 unrelated observations | 0.016959 | 0/80 | 0 | 0/80 |
| New contract, one-second idle | 0.902713731 | 80/80 | 0 | 0/80 |
| New contract, 96 empty activity updates | 0.902713731 | 80/80 | 0 | 0/80 |
| New contract, 96 unrelated observations | 0.902713731 | 80/80 | 0 | 0/80 |
| New contract, target-only overlap alternating with unrelated input | 0.902713731 | 80/80 | 0 | 0/80 |
| New task training after any new-contract interference condition | 0.902713731 | 80/80 | 0.902720 | 80/80 |

All three seeds require old support to match the same-time idle control within
1e-12 both before and after acquiring the new task. The overlap case activates
old outcomes without their old sources. It detects false credit as well as
destructive interference: activity-only protection would spuriously strengthen
those inactive-source associations through the global error updater.

A separate contradiction control trains competing outcomes for the same old
cues, then supplies eight controlled negative error steps with those sources
present, the old outcomes absent and the competing outcomes active. Old mean
support falls to 0.042934755; competing mean support is 1.159863946 with 80/80
target units above threshold and preferred over old outcomes. Scores can exceed
one in the existing activation dynamics. Original connections, experience
counts and historical patterns remain present. This is a synthetic causal-credit
control; it does not establish natural-language contradiction recognition.

Additional tests cover empty/source-only/target-only/subthreshold activity,
inhibitory paths, coactivity acquisition, eligibility interval composition,
duplicate/backwards clocks, positive/negative error, signed consequences,
stale-node credit, delayed source credit, normal observation/empty-input fallback/
internal thought routing, full restart, typed observations and numeric sensors.
The DNF integration fixtures now create connections at their simulated event
time rather than relying on a backwards wall-clock reset.

## Persistence and scope

Journal schema 3 records the explicit new dynamics string. Older readers reject
it. All earlier encodings keep their original activity/error/consequence laws.
Raw unversioned plasticity methods remain historical APIs; production activation,
thought and inquiry outcome paths select the recorded contract explicitly.

There are no new persisted fields: canonical Brain v3/checkpoint v10 remain
current. The physical retention clock still identifies its unchanged
`physical-memory-v1` law. Anchored v7-v10 recovery and older detached-import
restrictions remain as documented in COGNITIVE_JOURNAL.md. Changing the learning
policy is distinct from inventing historical activity or migrating weights.

The immutable `testdata/physical_memory_v1` fixture was generated against actual
PR33 using Go 1.22.12. Genesis and v10-prefix recovery must match its full state
and next historical thought under physical-memory-v1. Existing earlier
fixtures remain unchanged. A mixed old/new journal additionally verifies
restart and the next thought under the new serving policy.

This addresses the reproduced call-based inactive-path interference and the
tested shared-output credit case. Actively shared pathways can still compete;
it is not a general guarantee against catastrophic forgetting. H12 observed/forecast separation is addressed in OBSERVED_FEEDBACK.md.
H13 uncertainty calibration, H23 independent
generalization and broader continual-learning tasks retain their own criteria.

Local Go 1.22.12 validation passed: root-module `go test ./...`, `go vet ./...`,
`go build ./...`, seven-package race tests (knowledge/learning/activation/runtime/
bridge/core/CLI), and `make bridge-smoke`. Separate nested Go modules are unchanged
and are not covered by the root command.
