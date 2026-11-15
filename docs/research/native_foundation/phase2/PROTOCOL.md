# Native foundation Phase II — locked confirmatory protocol

Source: PR61 0ef994a1d8842b36dbc6b994ecafcb0bf9951174 plus main
32df59b9285d829ae73429c7f277621e63de7ce3 (PR60). Both exact heads have
successful CI. Main was merged into an isolated branch without rewriting history.
Architecture lock SHA256: fd3f1a5080f71eaed0a8a1de4bffd9432d9aac054660a4da76c5877fa7586ced.

## Development and selection

Development seeds 1709,1723, stream7200000, 2500 episodes, 80 cases.
A four-channel local relational encoder shares 80 canonical readout synapses
(activity/second moment crossed with reciprocal/nonreciprocal edge mass), rather
than allocating parameters per world NodeID. Graph topology is GIVEN, not learned.
This is a fixed numerical encoding with a learned linear projection, not a claim
of end-to-end learned representations or a full graph neural network.

Softmax .12 transferred, but additional longer-chain training collapsed to stop.
TD0 did not repair collapse. Epsilon-greedy .1 did repair that development failure
on both seeds, while preserving the same greedy results. Select lambda1 and
Epsilon .1. This is an exploration change, not a hardcoded operation strategy.
Historical v1 keeps softmax and the same update law.

Paired observed trials use identical action sequences and unchanged parameters.
Only AFTER both executions does squared difference-return regression update
canonical weights. Common input features cancel, including the bias. Actual
zero-input trials anchor action intercepts (one per eight pairs), because
contrast-only observations do not identify absolute values. Explicit intervention
mode records that actions were assigned by an experiment; logged softmax or
Epsilon probabilities are NOT the assignment propensity in that mode.
Randomization and matched exogenous conditions are adapter assumptions, not
certified by a pair identifier. No causal role labels are given to the learner.

Pilot credit budgets 32 and512 pairs protected common distractors, but greedy
returns equalled matched unpaired learning. This is NOT evidence of improved
behavior. Keep this null result and test fresh seeds without further tuning.

## Confirmatory runs — do not tune on these results

Independent seeds: 2027,2039,2053,2063,2081. Task stream:9300000.
Train2500 episodes per policy; evaluate256 held-out tasks per distribution.
Arms: identity readout / shared relational readout, BOTH Epsilon .1, lambda1;
relational softmax .12 is an exploration ablation. Random, untrained, static0,
static3, static5, static7 are frozen comparison policies on identical tasks.
Static3 is the fixed depth0–3 reference. Static5/static7 are explicitly
length-aware references for depths4–5/6–7, not selected after seeing the holdout.

Train only depths0–3 on graph0. Test:
- same distribution, independent samples;
- untrained depths4–5;
- an equivalent graph with disjoint/permuted IDs and new position;
- twice-wide populations;
- new graph length, untrained depths6–7;
- new population combinations, noise and amplitude.
Each model contains all four disjoint graphs from initialization. A separate unit
test adds graph members after initialization. Thus the functional population test
is not advertised as a full continual structural-growth experiment.

Adaptation: separate checkpoint restore,512 new longer-chain or reversed-reward
experiences, new held-out samples for both new task and original-task retention.
A hidden reversal supplies no context distinguishing the old and new reward
functions. Successful retention of mutually incompatible policies in identical
states is not identifiable; report observed forgetting, never imply it is solved.

Causal arms:512 randomized paired interventions versus the identical executions
learned separately;256 new tasks. Every eighth pair also gets one identical
zero-input anchor in both arms. Perturbed population is sampled without consulting
its reward role. The nuisance is correlated at baseline, then intervened upon.
Precision probe: hold active nuisance fixed, intervene on signal, execute
propagate→stop in both trials. This assigned sequence is an experimental probe,
not the required cognitive path. Precision excludes bias. The original one-outcome
50% metric and paired two-outcome metric have different information budgets;
behavior comparison uses exactly matched executed data. No causal precision claim
for shared weights that influence both relevant and irrelevant populations.

Report seed means, Student t95% intervals (df4,2.776445), paired seed differences,
training vs greedy evaluation, all failures and adaptation separately. Five seeds
support a bounded experiment, not broad intelligence/generalization claims.

## Verification and integration rules

Canonical7/checkpoint20/WAL14 container formats remain; explicit model version2
requires a new opt-in initialization. Existing v1 is accepted unchanged. Old
binaries reject version2. No automatic checkpoint conversion, user-data replacement,
world-plasticity change, physical action replay or V5 dependency is introduced.
Pending contrast is a single bounded factual experience, persisted with the same
Brain. It freezes learning until matched completion; observed outcome IDs and
completed contrast IDs are single-use across recovery.

Run full tests, numerical/provenance/pending-recovery tests, race tests, vet,
build and Android arm64 cross-build. Benchmark PR61 before/after on the same
machine; separately report shared inference sparse/dense and sizes1k/4k/40k,
read counts, allocations, GC, CPU/mutex profiles and checkpoint/replay costs.
No claimed 3x whole-thought improvement from a policy microbenchmark.
Do not merge a completed-foundation claim while any required gate fails.

## Follow-up diagnostic after the primary null behavioral result

The primary causal task uses signal amplitude0.3–1 with cost0.04, so almost every
case has the same optimal first operation. To test a decision boundary without
retraining or changing the learner, freeze existing checkpoints and evaluate
amplitude0–0.08 with independent nuisance0–1.8. Development streams7500000+seed
on32-pair checkpoints show failures in BOTH arms. Confirm on the already-frozen
512-pair checkpoints with a fresh stream9600000+seed and512 cases. These are
follow-up diagnostics, not a relabelled primary endpoint. No hyperparameter is
selected from these results. Accuracy threshold0.45 is inapplicable here; use
net return and regret relative to evaluator-only max(0,amplitude−0.04).

## Second candidate: factual mean calibration (new holdout)

The frozen primary paired model failed the boundary diagnostic despite good
parameter precision. Difference loss leaves the intercept unidentified. Candidate
`contrast_mean=true` also fits the observed pair's mean return through canonical
bias synapses ONLY, preserving cancellation for common input weights. This does
not inject hypothetical rewards or oracle labels. It is not a universal nonlinear
causal model. Pilot seeds1709,1723 at512 pairs recovered the evaluator optimum on
the boundary diagnostic. Keep the original paired candidate as an ablation.
The 32-pair calibration pilot still failed; both budgets are retained in the
confirmation instead of selecting only the favorable budget.

Fresh confirmation seeds3041,3061,3083,3109,3121, task stream10700000; boundary
stream10800000. Budgets32 and512 pairs,512 cases per evaluation, all three arms:
unpaired / original paired / paired-centered. Train and evaluate from scratch;
compare matched executed exposures and seed-paired returns. No later tuning on
these seeds. Retain all earlier failures. Common activity can affect outcomes
through `focus`; the claim of zero nuisance effect is conditioned on the
propagate→stop probe, not on every possible operation.
