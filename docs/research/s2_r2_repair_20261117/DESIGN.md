# S2-R2 design, fixed before new observations

Source: original S2 continuation 76617, checksum
`b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d`.
Base: S2-R1 `345446fd2b9f9bd660b6394e4fc8d1df54f07418`.
Both PRs are open; push 43918026310 and PR 43918079489 CI succeeded.

## Proposal and evidence

The same v6 learner enumerates the same candidate feature graph and full refit.
An explicit per-lesson `proposal` enables incumbent-relative proposals. Nil keeps
the exact historical law, including journal replay. No production opt-in.
Fractions are 1/64, 1/16, 1/4, 1/2, chosen as a geometric coverage grid, not fitted
to S2-R1 scores. A full step remains a control. A new-feature-only proposal fits
one coefficient to the incumbent residual while holding old weights and bias.
Joint proposals interpolate the query head only when the old feature graph and
readout basis are identical. Other query heads stay unchanged. All old validity,
precision, redundancy and retention gates apply. No thresholds are weakened.

New check experiences cannot share an episode or exact encoded measurement with
fit. Overlapping historical witnesses are excluded from fit, never removed from
retention. New check witnesses are explicitly evaluation-only. When the existing
128-witness budget is full, retain it without evicting historical experience;
all new experience and evidence provenance remain in the complete WAL. This
bounded retention sample cannot guarantee retention of every past observation.
Fit/check hashes and candidate parameter hashes are included in receipts.
Identical rejected search fingerprints yield an auditable duplicate diagnosis;
the command still advances and is journaled by the unchanged runtime transaction.

## Fixed experiment plan

Development: four independent native-observed fit/check pairs, 64 + 64 per pair,
seeds 41011/41017, 41023/41039, 41047/41051, 41057/41077. Pairs 3–4 use the
existing shifted stimulus distribution. Compare unchanged v6, independent
full-refit, bounded joint, new-feature contribution, and combined. Each arm
starts from the same continuation. Combined is the predetermined continuation
strategy if directed engineering checks pass and at least one actual bounded
or decoupled proposal is admitted with zero retention violations. No search for
additional seeds or lessons to manufacture that outcome.

Continuation: exactly eight sequential blocks, each with 64 fresh fit and 64
fresh check observations, seeds 42001+20*b and 42007+20*b, b=0..7. Alternate
original and shifted stimulus distributions. One learning command per block.
Also train the four control arms on these identical observations. Save every
block and receipt. Final checkpoint is the eighth block, irrespective of score.
No early stopping, best-checkpoint selection or extra acquisition after failure.

Final holdout: two streams, seeds 51001 and 51031, 128 observations each, original
and shifted. Generate and score once only, after code/config/checkpoint hashes,
development-derived constant, calibration limit, bins and error cutoffs are
frozen. Every arm predicts the same observed cases. Never load the old S2 final
holdout for selection. Final evaluation is a disposable copy, not new training.

Primary ability gates, each stream: RMSE <= 0.8 times training-only constant
baseline; Pearson correlation >= 0.5; calibration <= max(0.003,0.25*development
error SD); high-error versus low-error estimate gap positive for each operation.
Cutoffs are development error tertiles. Learned-readout ablation must increase
pooled RMSE by >=20%; newly learned feature/readout contribution must be nonzero
and independently beneficial. A–E1 time-matched oracle delta <=1e-7, C=6/6,
E1>=35/36; S1 predictions on shared observations differ <=1e-12. Calibration is
weighted mean absolute bin error using five development-derived boundaries.
Report RMSE, MAE, correlation, calibration, AUROC for high versus low error,
sequential adaptation, all input/feature/readout interventions and failures.

History interventions separately mask magnitude, presence and action; permute
error history within (current action, previous action, history action, presence)
strata with seed 61001. Permute forecast and all extras within current-action
strata with seed 61031. These are computational input interventions on fixed
truth, not claims about causal physical dynamics. History independently helps
only if its removal AND conditional permutation worsen held-out error. A
negative result is valid and does not automatically invalidate generic repair.

Persistence: final restart and full suffix replay from original 76617; rollback
to the original checkpoint/WAL prefix; exact old WAL prefix; duplicate event
and identical-retry receipts; signed parameters and native policy unchanged
except admitted query updates; six continuation observations seed 71001 plus
one preregistered 4-fit/4-check learning command from eight observations seed
71021, followed by restart/replay and retention. This later checkpoint is not
selected using final holdout or substituted for the confirmatory checkpoint.

Engineering and ability decisions are separate. Scientific test exit codes
check evidence integrity, never turn failed ability gates into PASS. No S3,
main modification, production checkpoint promotion or automatic merge.
