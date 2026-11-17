# Final confirmatory replacement before any scored prediction

The initial source lock was commit 80e67b9b1e02dc1b624652071bd436018e21f002.
Its acquisition completed, but evaluation failed restoring the very first
checkpoint: `unknown proposal diagnosis`. Zero final prediction files and zero
final scores were generated. The validator admitted only the old diagnostic
protocol name although the writer/learner already recorded the new explicitly
versioned audit/refinement protocol. This is an engineering compatibility
error, not evidence against a trained capability.

Repair only the restore validator's recognition and bounded validation of
candidate/fit diagnostic metadata. Native solver, search, admission gates,
all synapses, 24 final checkpoints, experiences, masks, thresholds and analysis
are unchanged. Add recovery/replay regression tests for both new protocols,
and preflight all 24 saved checkpoints before attempting final evaluation.

Preserve initial lock as `INITIAL_CONFIRMATORY_LOCK.json`, initial acquisition
guard as `INITIAL_FINAL_STARTED`, the whole initial unscored holdout as
`holdout-unscored-compatibility-preflight/`, and the failed evaluation log.
These observations cannot be used for fitting, checkpoint selection, repair
selection, metrics, or final inference. No training is repeated.

Freeze the corrected compatible source and the exact same 24 checkpoints
again, **before acquiring a new final holdout**. Replacement final seeds,
registered before acquisition: **2900037,3000037,3100037** for persistent,
IID and shifted dynamics. Counts remain 32 independent streams ×24 steps per
condition. Everything else in `PROTOCOL.md`, including all decision and
retention rules, remains in force. The replacement set is the sole scored
confirmatory dataset. This amendment replaces only the original final seeds
and adds the recovery preflight; it does not authorize model/threshold tuning.

## Boundary-mask correction before any aggregate final analysis

While generating the replacement predictions, code inspection found that
`p.History=[]float64{0,0,0}` changed history-presence from absent to present at
the first step of a stream. It did not affect any learned checkpoint: training
already masks the three encoded channels directly and preserves context.
Correct evaluation to use that identical encoded-channel mask, and add a
regression that an absent-history input is unchanged by magnitude ablation.
No aggregate final metrics, model comparison or checkpoint selection had been
performed. Preserve all replacement prediction arrays and observations under
`holdout-unscored-mask-preflight/` and `evaluation-mask-preflight/`, including
their source lock. Do not analyze those arrays for the final decision.

Refreeze the corrected intervention implementation and the same 24 models.
The **sole scored final holdout** now uses freshly preregistered seeds
**3200037,3300037,3400037**, persistent/IID/shifted, still 32×24 each. The earlier
replacement seeds above refer only to preserved preflight evidence. No
training, threshold, learning opportunity, environment law, statistical method,
or candidate checkpoint changes. This correction is disclosed even though its
effect is confined to the first step and history-available interventions would
otherwise have been unaffected.
