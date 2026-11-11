# V5 full executing-policy gradient repair

Base: PR58 main d7cb140b5996f1bddffeebbda693ebadfd714841.
This is an explicit learning option inside scientific-fidelity-v5. It adds no
parameters, predictor, controller, or production activation.

## Cause and scope

PR58 differentiated only the first 32 actor inputs. Its other inputs are
deterministic functions of the same trainable GRU: MUSE planning, the frozen
world decoder, conditioning and BMPS information features. Thus its documented
semi-gradient is not the total derivative of its executing policy. A numerical
test of a detached surrogate alone could not detect the missing paths.

On the authentic first PR58 training batch, a sampled numerical audit finds a
missing recurrent-gradient component. This is a correctness defect relative to
the full actor objective, not proof of the dominant cause of zero computations.
Matched root diagnostics also show negative computation value under the diffuse
stochastic continuation even where greedy continuation can profit. Return-to-go
already includes the actually incurred costs; replacing its action-independent
critic baseline with an unexecuted reward is neither necessary nor allowed.

## Explicit contract

Set `FidelityLearningV5.FullFeatureGradient=true` only for recurrent actor
learning. The false value is omitted from JSON, preserving old evidence hashes
and historical replay. It cannot be combined with AssessmentOnly or
FreezeRecurrence. The inference mode and all numerical inference outputs are
unchanged for fixed parameters.

Reverse differentiation now covers competence, observed-support softmax,
frozen decoder transforms, MUSE hypothetical GRUs, and the piecewise BMPS
maxima, then observed-history BPTT. Maxima use the forward first-argmax branch;
exact ties are nondifferentiable and do not imply a unique derivative.
The chain rule uses the native belief only. The planner still retains its
original best-modeled-terminal assumption. It is not relabeled actual-policy
VOC and does not override the neural actor.

Costs, observed masks, returns and advantages remain detached. The ownership
mask still allows only the 4,992 recurrent coordinates and 1,097 actor/critic
head coordinates for this loss. MUSE's 993 coordinates learn only from the
existing selected-outcome assessment loss. World parameters, legacy actor,
signed reward, and all outside weights/moments remain frozen. Adam history,
clocks and learning rates are preserved.

This follows recurrent actor-critic differentiation with detached returns and
advantages (Chen et al., 2026, author `eyechoice/modules/a2c.py`). BMPS uses
Callaway et al. UAI2018 equations 2–5. Native MUSE competence uses the learned
assessment contract (Valiente and Pilly, arXiv:2411.13537v2). Horizon remains an
adaptation, not a reproduction of the papers' tasks or the MUSE Dreamer model.

## Bounded validation protocol, fixed before fitting

Start both arms from authentic PR58 recurrent sequence155520 and identical
13,543 N coordinates/Adam moments. Compare old semi-gradient control against
full-feature-gradient candidate. No historical fitting is repeated.
Exactly 32 batches x32 on-policy actual episodes per arm; no reward bonus,
alternative-action label, propensity mixing, sweep or architecture changes.
All losses, learning rates and ownership are otherwise identical.

Training seeds1120660000000+i. Development64 cases seeds1120640000000+i
every8 batches, no adaptive stopping/tuning. Sealed final1024 root cases
seeds1120650000000+i. Calibration256 histories seed1120620000000 with
32 executed uniform terminal answers/history seed1120630000000.
One seed101 neural lineage; episode seeds are not independent trained models.

Final greedy evaluation compares Phase X parent, PR58, candidate, matched
control and .35 abstention. Gate: paired mean improvement at least .02 and
lower CI95 above zero against each comparison. Bootstrap2,000 paired cases,
seed1120670000000. Count only autonomously selected computations for the
autonomy gate and check their actual-policy value, precision and cost.
Offline forced-compute evaluation and unit fixtures are diagnosis only.

## Recovery and compatibility

Checkpoint19/WAL13 and directory fsync are unchanged. An older reader cannot
validate new-option evidence (the digest includes the true option), avoiding
silent replay with the wrong update. Recovery uses the preserved parent and
strict full WAL prefix plus authenticated suffix; no truncation, invented
parameters or optimizer reset. Never install experimental state in production.

Required checks include full Go tests/vet/build, all-coordinate full-policy
finite differences, rare actions, masks, stale evidence, ownership, authentic
replay, process exit after fsync/idempotency, language/knowledge regression and
Android arm64 cross-build. Report scientific gates separately from these checks.

## Recorded bounded result

The numerical test covers6,089 coordinates, maximum error8.90e-11. An authentic
first-batch audit agrees within5.61e-12 on sampled recurrent coordinates.
Each arm executes392 training computations in1,024 episodes. Both end at
sequence155552 with13,543 N coordinates;6,461 protected coordinates and their
Adam moments are bitwise unchanged. Strict candidate replay reproduces its
entire normalized checkpoint, including moments.

On1,024 new greedy cases, Phase X returns0.3634277; PR58, matched control and
candidate each return0.3541504 with zero computations. Candidate minus Phase X
is-0.0092773, paired CI95[-0.0280273,+0.0088867]; versus control it is exactly
zero on these cases. Autonomous computation and whole-task gates fail.
Fixing the missing derivative does not establish its dominance as a behavioral
cause. No production activation or large training follows this negative result.

Under the fixed stochastic continuation, all15 root diagnostic computation
values are negative; three become positive under greedy continuation. These
are different policy contracts. Native child-utility error remains: on90 new
states/180 operations, BMPS ranking regret is0.006324; actual-policy model
evaluation lowers it to0.003211 but still has MAE0.035455. It is an evaluator,
not an additional inference controller or reward label. The next functional
repair must demonstrate competent child terminal decisions and calibrated
relative child utility on actual experience before expecting selective thinking.
