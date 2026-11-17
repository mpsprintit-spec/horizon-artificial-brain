# Signal path and engineering interpretation

Archived environment revision-3 source: `48cb2174018d576257d8ac6e46e79ae49f6285e1`.
Primary full-state revision-4 source is identified by CONFIRMATORY_LOCK/BUNDLE.
FULL_STATE_PROTOCOL documents the correction of the earlier nine-mean
information audit: its 26.28% gain does not survive full-state conditioning.
With independent per-neuron stimulus variation, the primary full-state audit
finds 25.02% benefit, CI95 [19.28%,30.53%], before native revision-4 training.

| Stage | Actual implementation | Causal/provenance consequence |
|---|---|---|
| Actual operation | `activation/native_operation.go:PrepareNativeOperation` | Existing canonical recurrent pulse law, signed synapses, confidence, squash and stop. Experimental gain modifies a copy of the chosen propagate operation only. |
| Observed environment | `runtime/native_control.go:NativeStep.EnvironmentGain` | Only explicit nontraining evaluation actions may use finite gain in (0,1]. It cannot alter the policy config or become policy-training reward. WAL15 rejects downgrade; nominal actions retain WAL14. |
| S1 forecast | `s1_internal_dynamics_test.go:s1Predict` | Three existing learned numerical heads on the same Brain/N; targets are poststate projections. Subtracting the same prestate makes the residual identical to change-prediction error. |
| Error observation | `s2r4_temporal_test.go:s2r4Collect` | After actual Step, mean of three absolute residuals. No formula assigns an error target from latent regime. |
| History storage | Experience adapter's three-float buffer | Last operation's three residual magnitudes, not three temporal lags. Updated only after actual target operation; carried across closed native episodes within one outer stream. Ledger/data persist it. Autonomous neural storage/recovery of this adapter buffer is not claimed. |
| Pre-operation evidence | Same collector | Input/forecast/estimate fsynced before native target command, with before/operation sequence IDs and previous sample ID. Strictly increasing times in v2/v3/training/final. |
| Numerical input | `s2_self_evaluation_test.go:s2Input`, `neural_resolution.go:nInput` | Existing codec unchanged; prepending retains S1 reverse-lag coordinates. Forecast at 56–58; magnitudes 59–61; history presence 63; history action 64. |
| Source neurons | Original 80735 `NResolution.Sources` | Coordinates 59/60/61 map to native sources 270/271/272; presence/action to 273/274. Input coordinates are not proof of useful feature representation. |
| Feature representation | `knowledge/neural_metric.go:nMetricEvaluate` | Signed learned axes, center, quantizer, metric reconstruction and tanh(code) activation on N. Source history reaches only direct kind-1 feature 266; other source circuits have zero direct history axis. |
| S2 readout | Canonical query 257–260 | Source uses metric basis. Feature 266 history coefficients .00290446446/.00035468987/.00216034615, center .343707303, delta 3.61199957e-6, readout weight −.0296251977. Its activation is saturated in v3 development; a history intervention can flip sign while preserving saturation. Metric variation remains measurable. |
| Readout learning | `learnNMetric` | Still runs at 32/32. Bounded R2 proposals, independent Fit/Check, all old witness/precision/shared gates remain. |
| Recruitment | Existing residual direction and product search | At the feature ceiling, new directions and products are not enumerated. Below it, one fit-residual axis plus fit-ranked top-eight nonconstant product pairs are considered. Candidate traces now retain proposed axes/parents, fitted weights, solver failure/pivot/bound telemetry and all rejections. |
| Minimal research proposal | `neural_metric_refinement.go`, explicit refinement protocol | Searches generic normalized residual directions within existing task-local leaf neurons. Excludes shared heads and every parent circuit; does not name history channels. Check and retention still decide admission. No default change. |
| Persistence | Native checkpoint/rebuild, same single owner | Full historic WAL prefix retained. Explicit audited/refinement protocol names must be recognized by a compatible validator. Validator compatibility and masking bugs are fully disclosed in confirmatory protocol/history. |

Signed residual direction is lost when the adapter keeps magnitudes; the
three projection magnitudes and previous-operation channel remain distinct.
The experiment does not test a signed-history or multi-lag curriculum and
cannot rule them out as future representation improvements.

Source feature 266 mixes weak magnitude coordinates with current state and
operation. A scalar mixed projection plus saturated activation is not the
same as independently accessible temporal magnitude. Metric reconstruction
preserves more variation; the auxiliary accessibility probes test that
separately. Numerical coefficient bounds and quantization remain active.

The detailed native receipts identify actual proposed candidates and rejection
reasons. The top-eight cutoff/ranking law is audited in source; no widened-
search counterfactual or exhaustive all-pair admission experiment was run.
Consequently a negative result cannot uniquely attribute failure to the
top-eight cutoff. Saturation, history covariance, shared protection, native
historical-witness retention and independent external S2 retention are kept
as separate evidence rather than folded into a claim of learned low utility.

The original S2 generator's fully observed state, fixed graph/gain/time and
given S1 forecast deterministically specify its next actual error. Reexecuting
256 stored operations reproduces projections and error exactly. Raw history
therefore adds no new causal state once these variables are fixed, although it
can act as a proxy when a learned representation discards sufficient state.
The new persistent environment is deliberately different: hidden gain affects
actual operations and raw history has independently measured conditional
information. Its negative IID counterpart prevents assuming persistence alone
or mere history availability establishes information value.
