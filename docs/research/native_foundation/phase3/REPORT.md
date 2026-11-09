# Native Phase III — factual action credit and remaining gates

Status: **research continuation; the three foundations are NOT complete and
are not enabled as production defaults.** PR64 is stacked on PR62. This report
separates implemented laws, functional experiments and unresolved capabilities.

## Authentic source and recovery

Research began from main `32df59b9285d829ae73429c7f277621e63de7ce3` (PR60).
The separately verified conservative optimization PR65 is now merged as
`45ecb989b67da690b9bbc1b7a942bb8c55ca6f8e`; experimental control remains draft.
PR61 remains at `0ef994a1d8842b36dbc6b994ecafcb0bf9951174`; PR62 remains at
`ea69b2101004dbf10c07068e42436170c2adcf8d`. Neither history was rewritten.
Phase III branch: `research/native-phase3-20261108`, PR64. Its first mechanism
commit is `d33e16f6b887df4f7ba18de7dbeba30007319739`; delayed continuation is
`e8a59cf82de0eba52879fcfffb9a322401a5efe4`. Local/API publication trees were
compared exactly. The final published head is recorded in the PR/status.

Architecture Lock SHA256:
`fd3f1a5080f71eaed0a8a1de4bffd9432d9aac054660a4da76c5877fa7586ced`.
All 368 Phase II evidence payload hashes were rechecked. Its archive, checkpoints,
WAL, historical worktrees and two untracked old benchmark harnesses were preserved.
No production checkpoint was edited. All new experiments require fresh directories.
Old final holdouts were sealed; no tuning used them.

## What changed

Model `native-operation-control-v3` adds an explicit **factual action contrast**.
Both trials execute through the same BrainRuntime and activation engine. They
start from the same neural state, have an adapter-supplied matching scope,
different first operations and a controlled continuation. The first observed
outcome creates a bounded pending trial and makes no weight update. The second
observed outcome permits the contrast. Wrong state, scope, source, outcome reuse
or changed continuation is rejected. A first stop is absorbing; it cannot execute
later scheduled primitives. Where both arms continue, their suffix must match.

The update fits

`(G_A - G_B) - (Q_N(s,a) - Q_N(s,b))`,

where each G includes every executed computation cost. Gamma/lambda must be one;
truncated trials are excluded. Two selected operation populations receive opposite
normalized updates through existing canonical synapses. Existing per-ID and shared
relational readouts remain available. There is no separate value network or weight
table. A common action-independent reward offset cancels. Absolute Q levels are
unidentified by this objective and must not be interpreted as calibrated world
returns by a downstream consumer.

The historical Sarsa(lambda), feature-contrast and bias-mean calibration paths
remain. `propagate`, `focus`, `relax`, `stop` are unchanged. They modify/reuse the
existing activation state, can run without a new observation, and do not create
empirical evidence. Assigned training continuations are experimental treatments;
greedy evaluation is free to choose and repeat any primitive within its budget.

The parameter-inspection research helper now uses canonical ParameterSources
rather than assuming per-ID inputs. Its two initial failed relational runs are
retained, not represented as successful experiments.

Command preflight now uses an isolated metadata view instead of cloning full
episode maps. Digests and checkpoint APIs remain unchanged. A frozen delayed
model reproduces exactly after this repair, including canonical state and checksum.
The separate curiosity allocation repair is also retained from main; neither
optimization changes the learning objective or uses final holdouts for tuning.

## Root causes and hypotheses

| Hypothesis | Evidence and interpretation |
|---|---|
| H1: decisions insufficiently differentiated | New tasks require stop for low benefit or already-terminal activity, and propagation for sufficient future benefit. Both actions become optimal. This is a stronger diagnostic, not proof H1 caused every old failure. |
| H2: insufficient value representation | The common nuisance reward is nonlinear in activity; an absolute linear readout is misspecified. Contrasting actions removes the shared term here. State-dependent nonlinear action effects remain a limitation. |
| H3: loss of absolute calibration | Phase II feature effects can be precise while its intercept/value calibration fails. The clean ablation makes the controls nearly tie; action differences are invariant to the matched offset. Absolute calibration is unnecessary for these argmax decisions, not solved generally. |
| H4: invalid matching | Deliberately mismatched exogenous offsets degrade the candidate substantially. MatchKey records the adapter's assertion; it cannot certify an unobserved world condition. |
| H5: policy cannot exploit feature contrasts | Directly fitting operation advantages produces a measurable behavioral gain in two fresh confirmations. This supports objective alignment under the declared intervention assumptions. |
| H6: exploration interference | Every control receives the same assigned randomized actions and four factual executions per block. The gain here cannot be attributed to more favorable exploration. Autonomous exploration is not established by this experiment. |

The nuisance-reward construction is motivated by the distinction between action
features and an action-independent nonlinear reward term in
[Krishnamurthy, Wu and Syrgkanis (2018)](https://proceedings.mlr.press/v80/krishnamurthy18a.html).
Horizon does **not** implement their bandit algorithm or inherit its regret bound.

## Confirmatory functional evidence

Each model uses 512 factorial blocks, **2,048 executed training episodes**. Two
state interventions and two randomized distinct actions form a block. Goals,
depth and causal labels are evaluator-only. The learner receives neural activity,
performed operations and scalar observed rewards. Independent model seeds, matched
controls, new task streams and criteria were committed before final evaluation.
There are five independent models per arm, not 2,560 independent models.

One-step seeds: 7001,7013,7019,7039,7043; stream18100000; 512 test cases/model.
Delayed seeds: 8009,8011,8017,8039,8053; stream19100000; 512 test cases/model.
Development/validation streams and all failed pilots are in PROTOCOL.md and raw
results. Confidence intervals are paired seed-level Student-t, df4.

| Greedy result | One-step tasks | Mixed depths 0–2 |
|---|---:|---:|
| Untrained native policy | -0.404758 | -0.278793 |
| Random operations | -0.078057 | -0.094721 |
| Best reported static baseline | 0.085698 (one propagate) | 0.044454 (two propagates) |
| Unpaired learning, noisy reward | -0.269596 | -0.452912 |
| Phase II feature + mean, noisy reward | -0.289653 | -0.368138 |
| Action contrasts, noisy reward | **0.097977** | **0.085684** |
| Deliberately unmatched action pairs | -0.224114 | -0.230383 |
| Action contrasts, clean reward | 0.097977 | 0.085684 |
| Unpaired learning, clean reward | 0.097979 | 0.085695 |
| Evaluator-optimal trajectory fraction | 99.6875% | 96.0156% |
| Mean regret | 0.00000275 | 0.00029460 |
| Propagate-versus-stop advantage RMSE | 0.001669 | 0.005501 |

Noisy action-minus-unpaired return: **+0.367572**, CI95 **[+0.065095,+0.670049]**
one-step; **+0.538595**, CI95 **[+0.470252,+0.606938]** delayed. Against the Phase II
feature-mean control: +0.387630 [0.134568,0.640691] and +0.453822
[0.306740,0.600903]. Clean noninferiority criteria also pass; no clean superiority
is claimed. The candidate's noisy and clean behavior is identical in this matched
environment because the offset cancels algebraically.

The delayed evaluation's depth-specific optimum fractions are 100% at depth0,
97.01% at depth1, **92.23% at depth2**. The registered aggregate criterion passes;
that does not erase the depth2 errors. Effects in the reported precision/RMSE
probe are measured at the one-step decision boundary, not every temporal link.
The stochastic assigned-intervention training returns are 0.007831 and -0.017000;
they are not greedy performance. Changed canonical parameters: 208/464 in the
one-step model, 272/464 in the delayed model. World synaptic laws are unchanged.

The 32-block pilot failed stopping and repeatedly exhausted the budget. At 512
blocks behavior improved. This is learned decision change, not a coded stop rule.
All prespecified **scoped candidate** numerical criteria pass. No broad foundation
gate is automatically passed by these narrow criteria.

## Attribution, representation and retention limits

Controlled learned **effect** precision/recall is 100% in the confirmation probes.
This is not exclusive synaptic credit precision. The explicit first-pair probe
changes eight relevant and eight active nuisance synapses: **50% exclusive
non-bias precision**. The candidate learns nuisance cancellation over randomized
experience; it does not identify every irrelevant feature before updating it.
Gate D therefore remains incomplete despite the new behavioral evidence.

The Phase II relational ablation on development data learns with all 80 shared
parameters and benefits from action pairing. It does not demonstrate new graph
families. A new executed counterexample changes an outgoing connection from
excitatory to inhibitory: the four-moment encoder produces identical values, but
the actual next state changes. No amount of readout training can separate identical
features. A richer learned representation needs the omitted signed/temporal
relations and genuinely available context. It was not implemented in this increment.

No new observable-context sequential adaptation/retention confirmation has been
completed. Earlier Phase II longer-chain retention remains historical evidence;
hidden reward reversal remains a known failure. The present mixed-task training
must not be described as proof of continual learning, context reversal, or transfer
to unseen branching/recurrent/inhibitory graph families.

## Persistence, compatibility and runtime scope

Canonical schema7, checkpoint20 and WAL14 containers are unchanged. Model version3
is explicit; an authentic Phase II binary rejects a v3 checkpoint with an invalid
model-version error. There is no silent downgrade or migration.

New tests cover pending-pair checkpoint/restore, duplicate outcome, changed input
and continuation rejection before mutation, delayed net cost, factual offset
cancellation, and subprocess exits before append and after fsync before apply.
A retained version3 audit restores a checkpoint containing the first trial, then
replays the journal suffix through sequence531: canonical state, activation and
checksum match exactly, with zero physical actions. Recovery produces the same
continuation. Existing corruption, incomplete-journal,
provenance, deterministic replay and concurrency tests are retained. These are
specific tested boundaries, not an assertion that every crash is handled.

Local full Go regression, vet/build and Android arm64 cross-build pass. The full
local race suite passed on d33e16f (runtime 980.493s); the changed native paths were
race-tested again on e8a59cf. Exact-head e8a59cf CI
[42473928890](https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/42473928890)
passed full race, nested adapters, trained-dialogue recovery, Android build, Bridge
smoke and monitor. Android compilation is not on-device latency validation.
Later performance-only changes and final publication checks are recorded in STATUS.

The native experiment already uses one authentic BrainRuntime, canonical N and
activation engine. However `runJournalCommandLocked` deliberately rejects mixed
legacy commands while the native experiment is enabled. Real sensor/inquiry/
continuous-runtime coexistence is **not integrated**. No external physical action
is invoked by native commands or replay. Expanding this boundary before the
representation, credit and performance gates pass would be unsafe.

## Gate status and claim taxonomy

| Gate | Status | Evidence / blocker |
|---|---|---|
| A Architecture | Pass for this increment | One canonical N; no new learned weight store, cognitive engine, or V5 dependency. |
| B Operations | Pass in controlled environments | Recurrent native operations, learned stop/continue, actual delayed continuations. |
| C General policy | **Not passed** | New samples transfer; broad graph families and richer learned relational representation unproven; signed-edge collision demonstrated. |
| D Causal credit | **Not passed in full** | Randomized factual contrasts improve behavior and learned effects; exclusive nuisance-credit precision still 50% on the first-pair probe. |
| E Performance | **Conservative repair verified; native promotion incomplete** | GC-associated tail diagnosed; PR65 removes the dominant curiosity allocation source with 2.017x p95 gain on matched main. Native dense execution/restore remain costly. See PERFORMANCE.md. |
| F Persistence | Pass for tested boundaries | Canonical parameters, pending trials, WAL suffix, dedup and crash boundaries tested; explicit version rejection. |
| G Integration | **Not passed** | Opt-in experimental runtime path only; no production sensor/inquiry coexistence or default enablement. |
| H Continual learning | **Not yet proven** | New context-dependent sequential retention experiment remains outstanding. |

**Implemented:** factual action credit, absorbing-stop contract, canonical updates,
matching/replay validation, metadata-copy repair and performance instrumentation.
**Tested:** numerical/functional contracts, regression, native race, compatibility,
checkpoint/WAL and Android compilation as itemized above.
**Functionally demonstrated:** stopping/continuation and improved noisy net return
in two independently confirmed controlled environments.
**Generalized:** new held-out instances within these tasks only; broader claims absent.
**Causally validated:** initial-operation effects under randomized matched factual
trials; arbitrary observational credit and exclusive synaptic localization unproven.
**Performance verified:** individual measured workloads, not a 3x whole-thought gain.
**Integrated:** conservative curiosity optimization merged to main via PR65; neural
foundations remain on the research branch and experimental canonical runtime.
**Not yet proven:** full Gates C/D/E/G/H, end-to-end sensor control, observable-context
retention, and scalable general intelligence. No experimental controller is enabled
as the production default.

## Reproduction and next diagnostic

Use fresh output directories. `scripts/summarize_native_phase3.py EVIDENCE_DIR`
reproduces results.json, paired CIs and scoped-criteria outcomes. Exact commands,
source hashes, exit statuses, failed runs, checkpoints and profiles are retained.

```
go test ./...
go test -race ./internal/nativefoundation ./services/ai/knowledge ./services/ai/runtime -run 'TestPhase3|TestNative' -count=1
go run ./cmd/horizon-native-foundation -phase3-credit action -train 512 -test 512 -seeds 7001,7013,7019,7039,7043 -stream 18100000 -out NEW_ONE_STEP_DIRECTORY
go run ./cmd/horizon-native-foundation -phase3-credit action -phase3-depth 2 -train 512 -test 512 -seeds 8009,8011,8017,8039,8053 -stream 19100000 -out NEW_DELAYED_DIRECTORY
```

Do not reuse these final streams for tuning. The next mechanism must address the
signed/temporal representation collision and observable context in canonical N,
then test selective credit and matched retention on new development data. The
current action-contrast law is a reusable controlled-training tool, not a replacement
for that missing representation or a justification to activate production control.
