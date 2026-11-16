# S2 — native neural error anticipation

Baseline `11c6e0a78547d68a0be8f78b3c801e912722608d`; source is the unmodified
S1 continuation, sequence 75866. S1 remains scientifically PARTIAL. Production
continues to use its existing canonical bundle.

## Engineering contract

Use the existing v6 signed-synaptic learner and numerical readout on the same
Brain. No serving law, controller, policy, numerical feature ceiling, or source
parameter is reset or replaced. Native operations supply actual outcomes.
Native operation-policy learning is disabled for these observation trials;
its acquired 10752 updates are retained. Numerical lessons have their own WAL
receipts and are counted separately. Existing recurrent activity, eligibility,
plasticity, causal policy and recovery remain available, but this experiment
does not claim to train a new causal operation-selection policy.

The research adapter requests propagate/stop, measures projections, computes
post-operation residuals, and encodes observed context. It neither simulates the
transition to predict its answer nor selects operations from S2 output.

## Representation

Target: mean absolute error across S1's three normalized projections. This is
a nonnegative continuous error magnitude, not a probability, interval, or a
claim about the complete Brain state. The signed residuals and absolute errors
are computed only after a real operation completes. Error recognition is this
measured teaching signal. Learned anticipation is evaluated independently.

The input preserves S1's 48 measurements and their source-coordinate positions.
Two preceding rows add the actual three S1 forecasts, the immediately preceding
trial's three absolute errors, history presence and the preceding trial's
operation. History is an explicit previously observed sensor context; it is not
a table indexed by the current target. No current after-state, current residual,
sample ID, seed, or target-derived reliability group enters the forward input.
The output uses a new numeric query in the existing population registry. Its
readout shares the existing feature substrate with S1; it is not a second model
or Brain. Learned parameters reside in canonical signed synapses. Witnesses
are permitted as native learning evidence, and must be removable at readout
without changing predictions.

## Fixed acquisition budget and splits

One research lineage, with 64 development trials (seed 211), then 64 training
trials (seed 223), followed by 16 numerical lessons. After lesson 8, observe
64 new training trials (seed 227) on that same runtime. Lessons 9–16 use the
new block plus the native learner's retained witnesses. This tests continued
adaptation after new actual errors, without resetting S1 or S2. The API's independent
development set is explicitly used for diagnosis/feature admission, so it is
not reported as independent final evidence. No new acquisition of S1 occurs.
All sixteen lessons concern the new error query. No S1 training lesson is
replayed as acquisition. Fixed final stage: lesson 16, without choosing the best
checkpoint. No budget increase based on the final holdout is allowed.

After development observation, `PROTOCOL.json` records the target scale,
constant baseline, calibration bins, quantile cutpoints and numeric gates.
It must be SHA-256 bound before any S2 learning or final holdout generation.
Final holdout is generated once after training: two independent streams of 96
trials, seeds 307 and 401. The second stream extends the amplitude range from
0.05–0.65 to 0.07–0.73 and includes preceding stop contexts, within S1's tested
operation/range envelope. Training, development and holdout IDs are disjoint.
The final holdout cannot select a stage or tune thresholds.

## Preregistered decision formulae

Freeze from development: scale = 1 (the raw physical projection-error units),
first-block training-mean constant baseline, an additional full-training-mean
constant baseline (computed only from the two training blocks), five equal-width estimated-error bins spanning
zero to max development error (plus overflow), and actual-error low/high groups
at development terciles. Required precision for native learning is 0.002.
Development variance must exceed 1e-8 and its high-low tercile gap 0.003, or
identifiability is declared insufficient. No artificial reliability labels.

PASS requires all of: holdout RMSE <= 0.8 times both constant-baseline RMSEs and <=
0.8 times untrained RMSE in each stream; pooled Pearson correlation >= 0.5;
calibration error <= max(0.003, 0.25 * development error SD); actual-low and
actual-high group MAE both <= the respective constant baseline; estimated
high-low group gap positive within each operation; feature/readout ablation
RMSE >= 1.2 times candidate RMSE; at least one committed native lesson and
changed synapses; readout independent of witnesses; no future information and
readout state mutation; exact restart/replay predictions and native continuation;
time-matched A–E1 oracle delta <= 1e-7, C=6/6, E1>=35/36; S1 prediction drift
<=1e-7 on the identical newly generated evaluation distribution, with unchanged
native policy state. S1 historical retention FAIL is not revised by these S2
criteria. Previous-error context is separately ablated and its incremental
contribution is reported, including a null/negative result.

PARTIAL means learned synaptic adaptation with unmet capability/retention
criteria. BLOCKED means source/CI/recovery prerequisites or a legitimate native
learning route are unavailable. Engineering integrity failures stop acquisition.
No claim of S2 PASS can be based on arithmetic error recognition alone.

## Persistence

Retain the entire original S1 WAL prefix, add native observations and numerical
lesson receipts, checkpoint the fixed final stage, restart in a fresh process,
verify exact state through source/final suffix replay and a prefix rollback pair,
then append four new non-learning native trials without reacquisition. Save the
continuation for review as an S3 candidate. Never promote it to production.
