# Native numerical capacity dependency audit

Audited base: #76, 7196c8a7f4a619040fe3933fe1c5363e5be661bc.

| Layer | Code / dependency | Consequence |
| --- | --- | --- |
| Input | `neural_resolution.go:nInput`, `nDimensions=MaxCompositionSequence*7` | 32 is not the input dimension, total neuron limit, or semantic concept quota. The existing codec is unchanged. |
| Exposure | `NResolutionLesson.Proposal`, runtime `LearnCompositionWithDiagnosis` | The research ceiling is an explicit journaled lesson option; both arms otherwise have identical evidence and law. |
| Model load | `neural_resolution.go:loadNModel`, `compositionParameter` | Axes, centers, resolutions and query weights are reconstructed from signed canonical synapses. No separate fitted model persists. |
| Readout | `neural_metric_learning.go:learnNMetric` calls `consider("readout",...)` before the capacity condition | Readout learning still runs at 32/32. Shared circuit resolution proposals also run, subject to existing protection. |
| Growth gate | The `len(before.Circuits)<...` block in `learnNMetric` | At ceiling, both new residual-direction and product candidate enumeration are skipped completely. This is a real opportunity restriction, not evidence that unenumerated candidates would have helped. |
| Residual direction | Fit-only centered covariance of current residual with encoded measurements | Exactly one normalized residual axis is proposed. A weak temporal channel can lose to stronger input covariance; capacity alone does not alter this search law. |
| Product search | Kinds 2/3, nonconstant parent pairs, centered/normalized covariance ranking, top 8 | Reuses old feature outputs. Only eight ranked products are fitted. It cannot create a distinction absent from every parent output. Ranking sees fit only. |
| Solver | `neural_metric.go:nFitMetric` | Scaled least squares, ridge 1e-12, <=16 iterations, pivot 1e-15, signed coefficient magnitude <=7.99; unchanged. More columns can worsen conditioning. |
| Quantizer | `nMetricEvaluate`, `nMetricDelta` | Rounded code; metric reconstruction; activation=tanh(code). Small deltas can saturate activation. Metric readout is a separate existing basis, not automatically selected because capacity rises. |
| Proposal | `neural_metric_proposal.go:nMetricProposals` | Existing full-refit, bounded-joint and residual-only new coefficient proposals; same fractions, basis checks and signed bounds in both arms. |
| Shared protection | `nMetricSharedChange` traces old readout/parent dependencies | Protects circuits used by other known heads, including S1. Query-specific readout weights remain plastic. |
| Admission | `consider`, `nMetricRegression`, quality/redundancy/precision tests | Check improvement, validity, redundancy and old-witness retention remain identical. A capacity option grants no admission exception. |
| Installation | `newNNode`, `writeNParameter`, topology append | At most one winning feature is installed per lesson. Usually one feature neuron, plus any previously absent input source nodes; paired excitatory/inhibitory synapses encode axis/parent, center, delta and readout parameters. Exact observed growth is in the structure maps. |
| Witnesses | `nMetricIndependentEvidence`, fixed `maxNWitnesses=128` | Current Fit/Check are independent; historical evaluation-role witnesses excluded from fit. Full old witness bank preserved; new observations remain in WAL when bank is full. This is bounded retention evidence, not universal retention. |
| Persistence | `canonical_persistence.go`, `neural_resolution_validation.go`, runtime native recovery | Old validator used the same 32 ceiling. The new explicit admitted restore metadata permits <=64 only for v6, checks all old topology/signed-parameter invariants, and is replayed from the capacity-bearing lesson. |

## Experimental implementation boundary

`maxNFeatures=32` remains the production default and the v5 growth ceiling.
`NMetricProposalConfig.FeatureLimit` accepts only omitted/32/64; no arbitrary
unbounded limit. A nil or omitted option keeps historical behavior. Even an
enlarged checkpoint does not implicitly opt a later lesson into further growth:
that lesson must request its search ceiling explicitly. Lowering the requested
ceiling never silently prunes an existing larger graph.

`NResolutionState.ExperimentalFeatureLimit` is a restore bound written only by
an admitted research lesson. It never creates neural parameters by itself.
The compatible reader accepts old checkpoints with omitted metadata. Old
binaries are expected to reject >32 features; no backward-binary compatibility
is claimed. New checkpoints stay research artifacts and are not installed as
production state.

The ceiling originally protects resource use and serialized topology, including
candidate search and solver size. It is not merely a UI/display number. Raising
it increases an opportunity for learned structural growth; it says nothing
about the independent usefulness of that growth.

Runtime receipts distinguish readout, resolution, missing-direction/boundary,
missing-relation, duplicate/invalid, retention and no-improvement outcomes.
The final report pairs those receipts with actual feature activity, ablations,
new synapse maps, memory, latency and recovery measurements.
