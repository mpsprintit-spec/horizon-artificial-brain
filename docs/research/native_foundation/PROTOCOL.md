# Native foundation controlled protocol — sealed before first behavioral run

Baseline: d7cb140b5996f1bddffeebbda693ebadfd714841. One canonical Brain per
independent experimental replica; native policy parameters are ordinary synapses.
V5 is not initialized. All experiments use Go and actual activation primitives.

Development seed 37; validation seed/data stream 83/200000; final replication
seeds 101, 211, 307, 401, 503 with test stream 900000. Development/validation
may diagnose implementation; final test must not drive hyperparameter changes.
Each seed starts independent synaptic initialization and independent training
observations. 2,000 episodes pilot, at most 8,000 if the pilot shows learning.
This is a small controlled substrate experiment, not pretrained human knowledge.

Preexisting circuit: distributed four-unit populations connected by numeric
synapses; initial activity varies in amplitude, sparsity, noise and chain depth.
These circuits are fixture assumptions, not a demonstration that structure was
learned. Operation selection is the learned quantity. Only observed environment
rewards train it. No task label, correct action, target answer or future state
enters the runtime policy. Terminal scoring is in the environment adapter.

Arms: untrained greedy, trained greedy, random, fixed 0/1/2/3 propagation cycles,
matched lambda=0 (TD0), lambda=1 (Monte Carlo), no-learning control, native
existing recurrent-thought baseline. Training uses stochastic softmax separately.
All arms evaluate identical sampled tasks, costs and limits. Primary metric:
observed reward minus declared compute tariff; report accuracy, operation count,
stopping, state/parameter changes and held-out returns. Best static is selected
on validation only. Confidence intervals use independent seed means (Student-t)
and paired seed differences; episodes within a seed are not independent models.

Transfer: new population combinations, longer chains, changed noise/amplitude,
changed cost/reward regime, and changed state relation. List each separately;
success on one is not broad generalization. Ablations remove relevant circuit
edges or perturb unrelated activity; distinguish temporal statistical credit
from identified causal effects. Report unsuccessful distributions without
renaming them into successes. Persist source, commands, checkpoints, raw results.

Performance: same host/toolchain, sequential baseline/candidate samples, separate
profiling. Include sparse/dense and scale, lazy vs eager effective eligibility,
latency percentiles, allocations, throughput, GC, RSS, journal/checkpoint/recovery.
Full-task >=3x p95 is an objective, not an acceptance claim forced by test choice.

## Validation decision (before independent final seeds)
Development/validation seeds 37/83 at 4,000 episodes: lambda=1 mean test-distribution
return .518247 versus .517186 (lambda=.8) and .514666 (lambda=0). Differences are
small; select lambda=1 for the direct delayed-return target and keep lambda=0 as
matched control. All three fail zero-shot longer chains. Do not tune on final
seeds. Add static4/static5 so the longer-chain comparison is meaningful. After
frozen evaluation, separately restore each trained checkpoint and use 512 fresh
longer-chain or reversal experiences; label these adaptation, not zero-shot
transfer. Re-evaluate original tasks to measure behavioral retention.
Runtime budgets and preflight planning were added during validation; they do
not change the numerical policy or transition law. Final checkpoints use the
completed bounded contract; development checkpoints are not promoted.
