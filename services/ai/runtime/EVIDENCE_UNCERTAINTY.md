# Evidence-based predictive uncertainty (H13)

For current provenance accounting and checkpoint v12, see [EVIDENCE_PROVENANCE.md](EVIDENCE_PROVENANCE.md).

The predictive uncertainty law is retained by current `evidence-provenance-v1`
serving (see [STABLE_PROJECTION.md](STABLE_PROJECTION.md)), retaining H12 observation/feedback,
H09 activity, H08 physical-time and bounded recurrent context laws. This change
reads existing learned patterns, synapses and actual experience traces in one
Brain. It creates no answer database, semantic rules, external model or parallel
knowledge store.

## Scope and interface

The old Answer uncertainty was `1 - clamp(resonance)`. Resonance sums activity
times neural confidence over units, so adding active units could imply certainty
without adding learned evidence. Actual PR35 reproduces this:

| Probe | Raw resonance | Old Answer confidence | Old uncertainty | New confidence | New uncertainty |
|---|---:|---:|---:|---:|---:|
| Untrained, 1 input unit | 0.232500 | 0.250000 | 0.767500 | 0 | 1 |
| Untrained, 32 input units | 7.440000 | 0.250000 | 0 | 0 | 1 |
| One supplied A→B experience | 0.315699 | 0.250000 | 0.684301 | 0.390244 | 0.609756 |
| Twelve A→B experiences | 0.362499 | 0.250000 | 0.637501 | 0.904215 | 0.095785 |
| Twelve each A→B / A→C | 0.378970 | 0.250000 | 0.621030 | 0.510880 | 0.489120 |

`CognitiveOutput`, the default `Interpretation`, and `Answer` now carry a detached
`prediction_assessment`. Its scope is explicitly `learned-next-population`.
It contains the selected forecast population, alternative populations and their
estimated probabilities, unknown probability, effective evidence mass, context
support, and conditional alternative entropy (`ambiguity`).

In this version, Answer.Confidence is the estimate for that selected forecast;
Answer.Uncertainty.Level is its complement. Answer.NodeIDs remains the existing
ranked current-activity view, and is not relabeled as a semantic answer. The
assessment's own NodeIDs identifies the population to which confidence applies.
Raw activity, raw neural confidence and resonance remain separately available.
PredictionError retains the separate H12 direct observation comparison. None of
these fields establishes correctness of a natural-language interpretation.

## Readout law

The frozen real-observation boundary supplies the forecast and already credited
cue/context pattern references. For each full learned outcome population:

- Each member's effective support is pattern frequency times current bounded
  weight, confidence and cue/context quality. Redundant paths use maximum support.
- Members are averaged into one population's mass `m`, not added as independent
  observations or treated as mutually exclusive outcomes.
- If there are no credited ordered channels, exact matching source populations
  in the Brain's retained actual ExperienceTraces supply empirical outcome counts.
  Mean source-to-outcome synaptic weight times confidence weights those counts.
  Propagated/simulated targets without such traces supply no empirical count.

Let `M` be total effective mass and `q` that population's matching context quality.
Each outcome estimate is `q*m/(1+M)`, with remaining mass reserved for unknown outcomes.
One unit of unknown support prevents a single exposure implying certainty.
Context quality also caps confidence even after many repetitions. Frequency is
bounded at one million for numerical protection. These are explicit engineering
choices, not biological constants or a fitted universal Bayesian posterior.

The selected population follows the largest mean activation in the frozen neural
forecast among the supported outcomes, rather than silently replacing the neural
prediction with the frequency winner. Its reported confidence is that outcome's
evidence estimate. Equal forecast scores break deterministically by population IDs.
Missing evidence, missing prediction, expired/invalid observation credit or an
unrecognized population yields confidence zero. Internal thought does not add
counts or replace the real forecast. Time-based retention can reduce support.

Counts describe repeated experiences; they do not assert independent sources.
The historical H13 change did not close H15 provenance/independence accounting;
current bounded H15 accounting is specified in [EVIDENCE_PROVENANCE.md](EVIDENCE_PROVENANCE.md). Retained traces
are bounded by the existing Brain limit; an absent trace is not reconstructed.

## Inquiry

The complete observation path supplies this same uncertainty to the existing
learned inquiry policy. No new action rule, authorization or execution path is
introduced. With a learned useful change-view action, the 32-unit ambiguous
probe selects change_view; removing only the assessment restores the saturated
resonance readout and selects wait.

A newly built inquiry request captures the detached forecast it asks to resolve,
when the agenda matches the current revision. Completion compares direct returned
input to that captured population distribution. It does not subtract uncertainty
of an unrelated next prediction. Exact recognized outcomes receive a bounded
resolution proxy: prior uncertainty times prior known-outcome coverage. Missing,
partial and unmodeled outcomes conservatively receive zero estimated gain. This
is not a calculation of Bayesian expected information gain. Existing empirical
action-yield, costs, preferences and execution gates remain separate.

## Controlled calibration evidence

Twenty neuron-insertion/training-order seeds each train three opaque A→outcome
tasks: deterministic, balanced, and 80/20 noisy outcome frequencies. Each model
receives 40 training experiences. Twenty evaluation outcomes per model remain
only in the evaluator; they never enter runtime query input or learning.
The law/prior is fixed before this evaluation, with no fitting to those labels.
This gives 60 readouts scored against 1,200 held-out outcomes, not 1,200 independent
trained models or an unseen-task generalization benchmark.

| Measure | Result |
|---|---:|
| Multiclass Brier score, including unknown mass | 0.301672 |
| Ten-bin expected calibration error | 0.060863 |
| Same selected forecasts forced to certainty, Brier | 0.466667 |

The certainty comparator is an isolated readout ablation, not an end-to-end PR35
accuracy claim. After twelve actual A→B episodes, one contradictory A→C episode
raises uncertainty from 0.183037 to 0.221390. H12 reversal preferences remain
covered by their existing tests. Replicating population members across widths
1/4/32 preserves evidence mass, confidence and ambiguity; unrelated activity and
duplicate pattern paths cannot manufacture support. Twenty same-time thought
cycles preserve the assessment.

Historical PR36 numeric sensor tests exposed an H14 limitation: twelve stable
vector episodes leave eleven distinct retained outcome populations. The exact
vector receives confidence 0.081374 and unknown mass 0.491224; perturbations 0.01
and 0.1 in an unused channel fail exact evidence matching and receive confidence
zero. These are conservative uncertainty results, not robust sensor recognition
or successful calibration under representation drift. STABLE_PROJECTION.md
now documents bounded recovery improvements and the remaining limitations.

These results support bounded calibration on the specified distribution only.
General semantic accuracy, independent-source certainty, broad task transfer,
calibration under drift, and generalization H23 are not established.

## Persistence and compatibility

Journal schema 3 explicitly records `evidence-uncertainty-v1`; readers without the
new encoding reject it. Earlier encodings retain their historical readout and
inquiry laws. Outputs without the new assessment retain the legacy typed contract.
At H13, checkpoint v11 and canonical Brain v3 were unchanged: assessment is derived from
the persisted observation boundary and existing neural evidence, with no new
calibration memory or unauthenticated continuation cache. The full inquiry
command journals the captured request forecast.

The immutable observed_feedback_v1 fixture was generated against actual PR35
`e8a7b03682c6cf10177efb068194ece6300f0a55` using Go 1.22.12. Tests compare full
historical state and the next old-law thought from genesis and a v11 prefix.
New-law compound inquiry tests compare complete genesis/suffix/standalone restart,
the next thought, subsequent observation and derived forecast.

Local validation passed with Go 1.22.12: root-module `go test ./...`,
`go vet ./...`, `go build ./...`, race tests for knowledge/learning/activation/
runtime/Bridge/core/CLI, and `make bridge-smoke`. The separate nested Go modules
are unchanged and are not covered by the root command.
