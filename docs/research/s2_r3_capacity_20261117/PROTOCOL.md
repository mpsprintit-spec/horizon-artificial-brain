# S2-R3 — protocol registered before acquisition

Date: 2026-11-17. Base #76: 7196c8a7f4a619040fe3933fe1c5363e5be661bc.
No production changes, no promotion, no S3. Historical R2 ability stays PARTIAL.

## Source and fixed intervention

Authenticate all 443 R2 payloads and the complete immutable outer archive
SHA e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46.
Recover s2r2/s2r2-continuation.json with s2r2/training-combined/events.jsonl
through native recovery: sequence 80735, native checksum
9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3.
All observer, learning and control forks start here. Stop if verification fails.

The ONLY treatment is per-lesson feature_limit=32 versus 64 in the existing
combined bounded proposal, fractions [1/64,1/16,1/4,1/2], PreserveShared=true.
Metadata records the admitted restore bound. Nil/default lessons retain the
old 32 search ceiling. No threshold, candidate law, codec, solver or retention
change. Both arms receive the same native commands and data in the same order.
At most one feature can be added per command by the existing native law.
Eight commands therefore test an opportunity to grow from 32 to at most 40;
they do not test a saturated 64-feature network. This bound is not increased
after observing results.

## Data, seeds, budget

Three training seeds: 37, 83, 151. A seed specifies an independent eight-block
experience schedule; the learner itself is deterministic (there is no hidden
optimizer RNG). Fixed GOMAXPROCS=2 for every scientific phase; same x86_64 host.
Each seed has its own observer fork, independent of both learning arms.
Observers execute only actual propagate/stop operations with Learn=false.
They never learn S2. Every pre-input/forecast digest is fsynced before its label
is observed. Entire observer checkpoints and journals are retained.

Development: new native observations 64 fit + 64 check in each of normal and
shifted streams, seeds 830011/830017 and 830023/830029. No development learning
or model selection. Use development fit only for constant/calibration/bin
parameters and development check to characterize variance; never R2 holdout.

Sequential training: b=0..7, 64 fit + 64 check per block, normal on even b,
shifted on odd b. Fit seed=840000+1000*s+20*b+1, check seed=...+7.
Fresh independent acquisition per set, zero carried history across sets.
3,072 new training observations total. Every arm also receives the observer's
exact nonlearning native schedule before the identical learning lesson so that
A–E1 retention can be compared at a matched native state/sequence.
Save every block, receipt, diagnostic trial and before/after prediction.
Use block 7 as final unconditionally, including rejected commands.

Independent evaluation: once code/config/final checkpoint hashes are sealed,
acquire 512 normal (seed 990011) + 512 shifted (990031) observations on a separate
80735 observer. 1,024 shared paired cases for all six final models and the
source. No scoring/checkpoint selection until this seal. At error SD ~0.012,
512 per stream estimates a mean error at SE ~0.00053; paired effects can be
more precise. This is a moderate-effect design, not assurance of power for
arbitrarily small effects. Report CIs and inconclusive/equivalence boundaries.
No additional samples if a result fails. 30-minute timeout per training seed;
20 minutes per other phase; fixed 3 seed budget. Infrastructure failures are
retained; no scientific reruns or seed replacement.

## Interventions registered before final data

On disposable, read-only evaluation copies, with unchanged actual targets:
full; mask error magnitude; mask forecast; mask previous operation (the S1
previous-operation coordinate, distinct from error-history operation); mask
history presence; mask history operation; mask all extras (forecast, history
magnitude/presence/operation); conditional-permute history magnitude;
conditional-permute forecast. Current neural state and operation stay present.
History permutation strata: stream/current operation/previous operation/
history operation/history presence. Forecast strata: stream/current operation/
previous operation/history presence and quartile of a pre-input-only neural
state projection. Seeds 970001..970010 (ten permutations); no label matching.
Permutation is an input sensitivity experiment, not a physical causal claim.
Forecast is deterministic from the full prestate/operation under frozen S1;
it can improve learnability without adding conditional information.

Feature interventions: clamp all newly recruited feature outputs; zero only
new readout weights; clamp each old feature in turn and each new feature in
turn; zero each new feature readout separately; zero all query weights/bias.
The native graph is detached, never saved. Feature clamps affect descendants;
readout clamps do not. Record parent graph and signed synaptic weights,
activations, codes, metric values/errors, saturation, per-stream residuals,
correlation/redundancy and diagnostic candidate rejection frequencies.

All paired CIs use 2,000 bootstrap replicates, seed 980017. Resample four-case
temporal blocks separately within stream; resample training seeds as an outer
cluster for pooled treatment comparisons. Also show each seed/stream effect.
Report 95% intervals, exact sample counts and the limited three-seed precision.
History temporal dependence requires block rather than individual-row CIs.

## Isolated information controls

Separate copies, query index 4 (positive) / 5 (negative), both capacities, from
80735. Three control seeds 37/83/151; four lessons, 64 fit/64 check per lesson;
new independent 512-case test per seed/type. Same native learner and law.
Other pre-input channels fixed to a stored development prestate. A temporal
latent u in [0.01,0.09] is observed at t-1 as history magnitude; positive next
outcome=0.01+0.5*u, negative next outcome=0.01+0.5*v with independent v.
These synthetic control labels never enter actual S2 training or evaluation.
Seeds=1200000+1000*s+100*b+offset (fit 1/check 7/test 901); separate negative RNG.
Controls establish only access to an informative temporal channel. They cannot
establish ability to predict actual internal error. Positive control passes
if independent RMSE improves >=50% over train-only constant on both capacities
and removal/permutation of history worsens error with positive paired 95% CI.
Negative control reports generalization versus constant and paired history
effects. Failure of the positive control blocks a claim supporting H3.

## Retention, persistence, cost

Every block: S1 three projections on the same development observations, strict
max shift <=1e-12. All old witnesses unchanged. Final and restarted A–E1 use
native-schedule-matched control: all historical A/B/C/D/E1 oracles, max delta
<=1e-7, C=6/6 and E1>=35/36. Native policy state exact, updates remain 10752.
Signed parameters, feature metadata and full state must survive restart and
full suffix replay from 80735. Rollback to original paired checkpoint/WAL must
be exact. Original checkpoint schema stays readable; >32 topology requires the
new explicit capacity metadata and compatible code, not a claim that old
binaries accept enlarged checkpoints.

Measure same-host per-lesson wall time, trials/rejections, Go allocation and
live heap, process RSS high water, per-inference p50/p95 and allocated bytes,
checkpoint/recovery time/bytes, full WAL and suffix bytes, nodes/synapses.
Use CPU profiles to attribute candidate-search cost to native learner functions.
Feature-install cost is reported as an inclusive cost/upper bound when it
cannot be isolated from transaction validation/journal work. Never equate a
configured 64 ceiling with actual resource use or prediction quality.

## Decision rules fixed in advance

Capacity confirmed only if extra features actually recruited, pooled paired
95% CI for relative RMSE gain over 32 excludes zero, >=2% point improvement
in both streams, positive direction in all three seed-pooled comparisons,
beneficial new-feature and new-readout ablations with positive paired CI,
and all strict retention/persistence gates pass. Smaller or mixed improvement
is PARTIAL, not a confirmed capacity bottleneck.

Representation bottleneck supported only with measured ineffective/redundant
features or candidate limitations and corresponding intervention evidence.
Failure of 64 alone is insufficient. H3 supported only if history benefit is
absent at both capacities with upper 95% bound <2% relative RMSE benefit for
both mask and conditional permutation, controls behave as expected, and the
three-seed uncertainty is reported. Otherwise H3 remains INCONCLUSIVE.

Actual S2 ability may be re-evaluated with historical gates: per stream RMSE
<=0.8*development-fit constant baseline, correlation>=0.5, calibration <=
max(0.003,0.25*development SD), positive high-minus-low estimate gap per operation,
all-readout ablation pooled RMSE >=1.2*full, independent beneficial newly learned
contribution, and strict retention. No feature-count-based PASS.
No adaptive controller will be implemented. Explain evidence needed for reuse,
growth, reorganization, consolidation, pruning and retaining the current graph.
