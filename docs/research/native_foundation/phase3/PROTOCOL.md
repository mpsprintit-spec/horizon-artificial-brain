# Phase III — development protocol and gates

Source: PR62 `ea69b2101004dbf10c07068e42436170c2adcf8d`, tree
`22afd45d51b46ecf2956e988f6a382b8c13826ff`; main remains PR60
`32df59b9285d829ae73429c7f277621e63de7ce3`. Both PR62 CI runs succeeded.
Phase II's 368 evidence payload hashes were verified without a mismatch.
Its source worktree is clean. The historical baseline/performance worktrees each
retain an untracked benchmark harness; neither was overwritten or cleaned.

All old holdouts are sealed. Development seeds: 5003,5011. Validation seeds:
6007,6011. Reserved confirmation seeds: 7001,7013,7019,7039,7043. Do not run the
confirmation before recording the selected candidate, budget and numerical gates.

## Candidate 1: matched factual action contrasts

H5: Phase II's difference objective compares features under identical actions;
it does not directly train the difference between alternative operations.
H3: absolute reward offsets may add variance/calibration requirements unnecessary
for deciding which operation is preferable. Test an observed action-advantage
objective while preserving Phase II's feature contrasts as a separate baseline.

Model version `native-operation-control-v3` opts into the new contract; no schema
migration. Older binaries reject v3. Parameters are the same canonical synapses.
Two executed trials have identical initial state, the same adapter MatchKey,
different first operations, and identical continuation (with absorbing stop).
Only gamma=1/lambda=1, complete explicitly intervened trials are supported.
Fitting `(G_A-G_B)-(Q(s,a)-Q(s,b))` updates the two selected output populations.
The common absolute value remains unidentified and is irrelevant to argmax.
This is initial-operation credit under the controlled continuation, not proof of
individual credit for arbitrary different trajectories. The adapter's key is
auditable matching metadata, not proof that hidden conditions were matched.

## Development experiment

Execute a randomized 2x2 factorial block: two states (randomly remove one active
population) and two uniformly sampled distinct first operations. Every trial
really executes its assigned primitive and stop through BrainRuntime. A shared
exogenous scalar reward offset is held fixed across all four executions. Reward
is .25 times observed goal activity plus the offset, bounded within [-1,1].
The goal population and effect labels are evaluator-only. The learner sees only
neural activity, performed actions and scalar factual rewards.

Initial states include pre-goal and already-terminal populations. Thus propagate
and stop are optimal in different cases. Focus/relax remain available and may be
learned as inferior here; this first diagnostic does not claim broad operation
composition. Costs remain .04 in this diagnostic; observable varying tariffs and
broader graph families require later experiments.

Compare action contrast, Phase II feature contrast plus mean calibration,
ordinary unpaired learning, and deliberately unmatched action pairs. All receive
the same four state/action executions per block. Unpaired uses action-pair order;
feature pairs require a different grouping/order because only one pair can be
pending. Record this difference. Independent mismatch RNG prevents the negative
control from silently changing later tasks. Repeat without the reward offset.
This tests randomized interventions and nuisance-offset cancellation, not
identification from arbitrary observational confounding.

Pilot budgets: 32/128/512 blocks, 128 independent test cases. Development stream
16100000. Validation stream 17100000. No former Phase II holdout is reused.
Primary measures: net return, regret, evaluator-optimal trajectory fraction,
operation count, and propagated-versus-stop advantage RMSE. Effect precision and
recall use a prespecified absolute effect threshold .02 for controlled input
interventions; do not relabel these as exclusive-synapse change precision.
Also record false nuisance effect, signal effect error, parameter changes and
checkpoint. Greedy evaluation performs no learning.

## Required before promotion

This first candidate is not the full Phase III. Unit tests must establish cost
accounting, common factual offset cancellation, delayed continuation, rejection
of changed state/scope/continuation, v1/v2 regression and pending-pair replay.
Full confirmation, causal parameter attribution, learned richer representation,
observable context/retention, performance root-cause work and real runtime
integration remain gates rather than assumed successes. No production merge.

## Locked confirmation: candidate 1 (2026-11-08)

Development and validation support 512 blocks (2,048 real, randomized
intervention episodes per model). All four arms are frozen: action, feature plus
mean calibration, unpaired, unmatched. Repeat each with and without shared reward
offsets. Use identity features for this credit diagnosis; the relational encoder
is a development ablation, not a silently substituted confirmatory candidate.
Learning rate .15, gamma/lambda 1, max steps 12, all other Config defaults unchanged.
Final seeds 7001,7013,7019,7039,7043; test stream 18100000; 512 cases per seed.
Training generator remains seed+13100000; independent mismatch RNG seed+15100000.

Prespecified scoped success criteria (all required for the candidate claim):

- In noisy conditions, mean paired net-return gain against BOTH unpaired and
  feature-mean controls >= .01, with paired seed-level 95% Student-t CI lower > 0.
- Mean greedy regret <= .002 and evaluator-optimal trajectory fraction >= .95.
- Mean effect precision >= .95, recall >= .95, advantage RMSE <= .01.
- Clean condition: paired action minus unpaired return CI lower >= -.002.
- Exact-head contract/replay/crash tests pass. Any failed criterion is reported.

Five independent seeds, not 2,560 independent models; CIs use df=4, t=2.776445.
Report static0, static1, random, and untrained greedy matched evaluations too.
The unmatched negative control and clean ablation test mechanism assumptions;
they do not establish causal identification without randomized/matched trials.
Training metrics describe assigned interventions, not greedy policy performance.

This confirmation CANNOT pass the entire Gate D: action contrasts can initially
update active irrelevant features. Effective causal influence accuracy is distinct
from exclusive synaptic credit precision. Nor can identity-readout credit results
pass Gate C, H, or runtime integration. These remain separate mandatory gates.
No changes after final evaluation will be tuned against this holdout.

## Extension: absorbing stop and delayed action effects

After candidate-1 confirmation was sealed, inspection found the initial suffix
validator too restrictive for adaptive computation: a first `stop` necessarily
prevents any later scheduled primitive. The extension compares suffix decisions
where both arms are still active; an absorbing first stop may have no suffix.
Different suffix decisions while both arms continue are still rejected. This is
a computation/experimental matching contract, not a mandatory policy schedule.

The `-phase3-depth 2` environment samples depths 0,1,2. Training interventions
share `propagate -> stop` continuation for depth 2; the terminal reward is only
submitted after the entire executed sequence. Greedy evaluation is unconstrained
and must learn both continuation and stopping. Each block still uses four factual
trials, identical across controls. It never tells the learner the depth or optimum.

Development 5003/5011 and validation 6007/6011 at 512 blocks are promising. Keep
512 blocks, .15 learning rate and all other settings. New confirmation seeds:
8009,8011,8017,8039,8053; test stream 19100000; 512 cases. Run action, feature,
unpaired and unmatched, noisy and clean. Reuse the numerical candidate criteria
above, reporting depth-specific behavior separately. These seeds are not the
candidate-1 final holdout. Failure is retained; no retuning on either final set.

The research parameter-inspection helper was corrected to enumerate canonical
ParameterSources, so the existing relational encoder can be ablated too. Its
first failed executions are retained; they produced no claimed scores. Relational
ablation uses development seeds only and is not broad generalization evidence.
