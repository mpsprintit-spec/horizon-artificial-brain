# S2-R5 sequential development and confirmatory protocol

Registered before any R5 training. Six immutable training seeds:
**37,83,151,211,307,419**. Source: R2 sequence 80735, checksum
9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3.
Every arm starts from a separate byte-identical R2 checkpoint/full WAL.
No R3/R4 arm supplies any parameter or state. Development-only R4 observations
are allowed for calibration diagnosis. No R4 final holdout is used for tuning.

Order: authenticate sources/native replay; exact contract audit; calibration;
fixed-basis B/C development; only if fixed-basis history gain is below 10%
and original solver diagnostics are insufficient, test generic block leaf
refinement; only then test controlled research capacity 64. A finite negative
search cannot establish global impossibility. The report must distinguish
representation evidence, solver limitations and insufficient evidence.

A is original audited R4 learner, ceiling 32. B is native bias-only constrained
calibration. C is fixed-basis constrained readout. Cw adds development-only
retention coverage anchors to C. Optional D uses query-local block-sparse leaf
refinement at ceiling 32; optional E permits controlled recruitment, ceiling
64. All persistent arms have a paired magnitude-masked arm with identical
presence, previous operation and forecast. At least the final selected repaired
method and A are also trained on IID and paired masked IID streams. Mechanism
controls full-direction versus block-direction, unconstrained versus
constrained readout, uninformative blocks, and affine calibration are diagnostic
stages, not silently combined with confirmatory evidence.

Each training seed: 16 streams x16 real native transitions per condition;
4 lessons, each 2 whole streams Fit and following 2 whole streams Check
(32+32 observations). All arms receive identical observed operations, number
of lessons and fixed final lesson-4 stopping; no early stopping on holdout.
Observations are replayed to each copied Brain before numerical lessons.
Fit augments lawful legacy Fit-role witnesses, never current Check or new
Check-role anchors. Time is ordered simulated nanoseconds as in final R4;
wall costs are measured separately. No parameter/runtime gain is learned
from hidden environment variables.

Fresh development: base seed 5100011, 24x16 streams each persistent and IID.
Development Fit/Check: streams 0..7/8..15 (lessons use 2+2 disjoint streams);
streams 16..23 are development evaluation only. Coverage anchors are 64 new
old-distribution observations, seed 5200011; only coverage stages may use
them. Existing witness snapshots and full provenance remain available.
New external old-distribution retention: seeds 8900003 and 9000007,
128 observations each (normal and shifted); these remain sealed until final
source/checkpoint lock and are not anchors. Historical external R2 holdout
limit stays **0.01766457**, independently of observed R5 outcomes. Additional
retention limit: same-distribution source RMSE *1.05 +1e-7, registered now.
A–E1 original oracle <=1e-7; S1 prediction shift exactly zero on historical,
new-retention and final input coverage; original native policy unchanged.

Final independent seeds **8100017,8200031,8300043**, persistent/IID/shifted;
32 streams x24 steps each. Their data must not exist before a clean committed
scientific source and all fixed final checkpoints/configs are SHA-256 sealed.
No checkpoint/threshold/seed selection after final acquisition. Any engineering
error preserves failed evidence and requires explicit pre-evaluation amendment.

Solver: 256 scaled cyclic coordinate descent sweeps; ridge 1e-12;
change stopping 1e-11; multiplier path
[0,1e-4,1e-3,.01,.1,1,10,100,1e3,1e4,1e5,1e6,1e8];
40-step feasible segment bisection and factor .999999 for strict interior.
Native signed bounds, original group retention/sign protections, shared guard,
Check admission and required precision are unchanged. Fixed-basis mode stays
incumbent. All rejected and zero updates remain evidence, never success claims.

Analysis: RMSE, MAE, means/bias, ten-bin calibration, residual quantiles,
per-operation results and per-seed results. 5,000 paired hierarchical bootstrap
replicates, resampling six training seeds and whole evaluation streams; report
CI95 for paired RMSE relative improvements. Streams, not rows, are independent
resampling units. History gate: >=10% gain over identically trained masked
control AND positive CI95 lower bound AND conditional permutation/ablation
and historical+new retention. IID equivalence: paired history-effect CI90 must
be inside **[-3%,+3%]**; a CI spanning zero alone is inconclusive. Permutation
donors match presence, operation, previous history operation and nearest
Fit-scaled full observed prestate/forecast/context, omit targets/history;
intervention seeds and neighbor budget 12 fixed before final acquisition.
Constant/bias/affine offline controls fit ONLY development/training Fit.
These controls are never installed as a second Brain or runtime head.

Gate C requires a committed independently useful native update and all
retention gates; tiny/no-op numerical changes are not learning evidence.
Failed external retention disqualifies promotion even if internal gates pass.
Gate B target mean prediction/actual ratio in [0.8,1.2], with exact ratio and
calibration effect separated from temporal gain. No overall PASS unless every
applicable gate passes. No main/production change or automatic merge.

Resources: wall training/iterations/candidates/allocations/RSS, checkpoint and
full WAL bytes, inference p50/p95, recovery wall time, Go tests/build/vet,
targeted race tests and Android arm64 cross-build. Desktop timing cannot
establish Android speed. Publish all failures and full recovery artifacts.

## Final engineering specification, before confirmatory acquisition

Readout and new/changed circuit coefficients use a 40-bit dyadic significand
before native validation. This fixes the disclosed population-averaging issue;
it does not change a gate. Unchanged source circuit coefficients stay exact.
The original witness bank and the 64 extra anchors are checked as TWO SEPARATE
banks. Concatenating them into a single query sum could dilute old protections
and is explicitly forbidden. The solver penalty may combine losses, but each
original and additional constraint must separately pass the original evaluator.

Block search splits each generic 7-coordinate encoded sequence row into its
first and second three-measurement groups, ranks Fit residual directions and
tries the top four. It does not name, prefer or force history. Block mode tries
only block directions; the full-dimensional direction has its own control.
D replaces only a safe task-local leaf; E recruits a new circuit without
altering old circuits. Fixed32 slot deletion/pruning is not implemented because
no audited equivalence certificate permits deleting a protected circuit.
The zero-variance block [65,66,67] is an explicitly degenerate negative
representation control; false temporal reliance is tested independently by
IID equivalence. It cannot establish performance for arbitrary noisy blocks.

Training pairs: persistent A/B/C/Cw/D/E, IID A/D/E; each full+masked,
18 models per training seed, 108 total. Every model gets 256 observations and
four opportunities (128 Fit and 128 Check temporal observations, plus lawful
legacy Fit-role witnesses). Same seed/condition observes byte-identical native
operations. Capacity 64 stays research-only; four lessons permit at most four
recruited features. No claim extrapolates these results to a filled 64 basis.

Permutation: 10 preregistered seeds 9100031..9100040; 12 neighbors; full
36-neuron state plus 9 previous-state projections and 3 S1 forecasts (48
continuous dimensions), exact operation/previous operation/history operation/
presence; donors from another stream. Nearest-neighbor matching is approximate
and its distance distribution must be reported. Ten fixed calibration bins
use boundaries [0,.01,.02,.04,.06,.08,.12,.2,.4]. Scalar controls (constant,
bias, affine) are fitted only on each training seed's designated Fit streams,
sealed before final; no coefficient is installed as an external runtime model.
All inference resource measurements are desktop observations on a shared host.
