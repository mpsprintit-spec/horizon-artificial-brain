# Protocol deviations and measurement scope

These are disclosed deviations, not retroactive edits to `PROTOCOL.md` or its
frozen hash. No final data were used to change learning or choose a checkpoint.

1. An initial environment preflight skipped before acquisition because the
   reused observer helper needed `HORIZON_S2_SOURCE`. Its verified source copy,
   log and zero-acquisition state are retained; the environment routing was
   corrected before any experience was acquired.
2. The protocol prose says development Fit supplies constant/calibration/bin
   parameters and Check characterizes variance. The implemented Prepare uses
   **Fit for the constant**, and **development Check for SD, calibration limit,
   error tertiles and bin boundaries**, matching the older R2 helper convention.
   This implementation/prose discrepancy was identified during final reporting.
   The actual `METRIC_PROTOCOL.json` and its hash were fixed before training and
   final acquisition, applied identically to both capacities, and never altered.
   Paired RMSE treatment effects, feature growth, ablation errors, source
   integrity and retention do not use those bins/thresholds. Every actual S2 arm
   already fails normal-stream RMSE/correlation gates independently of this
   discrepancy; no ability PASS is claimed. Future protocols should state the
   development roles consistently. No retrospective recalibration was done.
3. Independent training-seed processes overlapped on the same host; each had
   GOMAXPROCS=2 and fixed scientific clocks. Their inclusive training times are
   observational. Clean resource comparisons use separate sequential process
   phases after all training/retention activity completed.
4. The broad current-state-and-operation-only input audit was added after final
   acquisition, so it uses only stored development Check cases. It is not a new
   confirmatory holdout condition; see `DEVELOPMENT_INPUT_AUDIT.md`.
5. The initial raw Go metadata-structure comparison falsely flagged diagnostic
   nil/empty slices despite exact serialized replay. Independent field/byte
   checks and preserved negative evidence are documented in
   `PERSISTENCE_COMPARISON_AUDIT.md`. No Brain or frozen analysis code changed.

Post-registration helper changes add sealing/provenance checks, the stored-data
supplement and measurement audits; they do not change the numerical learner,
registered training schedule, original final interventions or selected models.
The exact confirmatory core/harness/analysis hashes remain in the seal. Report
the three-seed precision and the tested maximum of 40 actual features; do not
generalize to a trained saturated 64-feature network or Android latency.
