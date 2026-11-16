# S1 — frozen protocol, 2026-11-16

Base: ae26b2ea39e7bae2c7f3d5b2660e5063168780bb. One continuing research
copy of horizon-primary-brain, A–E1 sequence 75331, native policy updates 10752,
57 neurons, 94 synapses, 24 temporal patterns. Source SHA-256 values are pinned
by scripts/prepare_native_continuation.py. No production promotion, S2, or merge.

## Objective and boundaries

Predict continuous internal activation after a configured native operation,
before executing it. Use the existing signed-synaptic numeric v6 learner, not
an external regressor or an exact simulation of the transition law. No new
learning rule. Its transient candidate solver is existing substrate code;
learned directions, products, resolution, bias and readouts live in canonical N.

The original policy affords propagate(gain=1) and stop. Focus and relax are
not added to that policy for this experiment. Evaluation overrides select real
operations but never teach the policy. The numeric learner receives observed
outcomes afterwards. A zero, non-learning policy outcome closes the trial;
it is a bookkeeping value and supplies no learning or S1 target.

The numeric API currently requires an adapter registration and generic banks.
An empty written projection registration is compatibility metadata, with no
text, lexical lessons, language training, or existing-population conversion.
Only annotation-free native registration and v6 numeric coexistence are allowed.
Both rollback backups must retain native schema20. All original A–E1 neural
units, synapses, patterns and policy must be checked for retention.

## Observation and representation

The adapter reads the runtime's actual pre-state, the requested operation, and
the preceding observed internal state/action. It encodes the original 36 world
neuron activations divided by 2, followed by nine previous population means
(divided by 2) and one operation coordinate (-1 propagate, +1 stop), previous
operation coordinate, and previous-state presence. Fixed coordinate packing
contains no next-state data or transition-law calculation.

Three continuous targets are the actual next-state mean activations of the
four-neuron populations 5, 6 and 7, divided by 2. These measure retained gate
activity and spread into the two learned outcome populations. They do not
reconstruct every neuron or the entire plastic state of the Brain.

Fit/check streams use independently generated observed states (seed37/seed41),
64 samples each, uniform population amplitudes in [0.05,0.65], both operations,
with/without one preceding real propagate. Twelve passes through all three
numeric queries are the fixed training budget, on the same state and journal.
Each pass reuses the observed fit/check evidence, not independent evidence.
Save pretraining and stages after passes 1, 4, 8 and 12. Never select a stage
using final holdout performance. Structural growth is decided by existing N.

Final holdouts use seeds83 and151, 96 samples each. They use amplitudes in
[0.07,0.73], previously unseen combinations, both operations and preceding
propagate/stop contexts. Sample hashes and episode IDs must be disjoint.
These holdouts are evaluated once after the fixed training budget.

## Frozen metrics and acceptance criteria

Primary: RMSE over all samples and three continuous targets; report MAE and
fraction of target coordinates within absolute error 0.05 (accuracy).
Report each operation and each holdout seed separately. Missing/untrained
readout means prediction zero, with status disclosed. Never substitute truth.

Compare untouched A–E1 (no numeric learner), initialized/no-S1-learning control,
post-S1, and evaluation-only ablation of all synapses incident on S1-created
numeric feature nodes. Ablation retains query topology and experience witnesses.
An additional witness-removal readout must be unchanged, demonstrating that
prediction reads synapses rather than an answer table.

Context contrast: for matched operation/prior-action, compare predictions and
actual changes between independently varied internal pre-states. Contrast RMSE
must improve at least 20% over no-learning control, and predicted versus actual
contrast Pearson correlation must be at least 0.80.

PASS requires BOTH holdout seeds: RMSE <=0.06 and >=30% reduction against both
untouched and initialized controls; pooled coordinate accuracy >=80%; pooled
ablation RMSE at least 20% worse than post-S1; a nonzero prediction synapse
delta; context criterion above; exact readouts after process restart; full
checkpoint/journal replay and immutable original A–E1 neural parameters.
Retain A/B/D margins within 1e-10 of the saved oracle, C 6/6 and E1 >=35/36.

PARTIAL: observed synaptic learning exists but any capability/retention criterion
fails. BLOCKED: no usable lawful learning path can be established. Scientific
outcome and engineering validation are reported separately; test success does
not make a negative capability result PASS.

## Persistence and termination

Checkpoint and full journal retain the original byte prefix. Verify native
checksum, SHA-256, identity and sequence; close the writer and use a separate
process to restore. Compare continuous predictions and A–E1 oracles without
retraining. Verify suffix replay from pretraining and the original A–E1
checkpoint. Save a further native-operation continuation pair from the same
S1 lineage. Native policy update count is reported separately from committed
numeric learning updates: prediction acquisition need not retrain that policy.

One research branch and one non-draft review PR. No merge or S2 execution.
