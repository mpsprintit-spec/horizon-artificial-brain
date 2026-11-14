# Canonical learning flexibility — protocol 1

## Tujuan

Several small curricula must run on one persistent training Brain, through the
existing canonical runtime, without changing source between curricula. Separate
engineering, training, and generalization decisions. No production memory is used.

Baseline verified 2026-11-14: main `3a9e7ce3ffadff86c792050056ebea4eda48da0b`;
PR61 `0ef994a1d8842b36dbc6b994ecafcb0bf9951174`; PR62
`ea69b2101004dbf10c07068e42436170c2adcf8d`; PR63
`1837968ce0ba5707e3f15a36b14a7c87b1cc357f`; PR64
`0df3f19a33bd7de69ed496293591d734331c5e51`; PR67
`e9ae2331c9c020f3d905a79cc0a303599ac83889`. All listed PRs remain draft/open,
unmerged. PR67 head workflow 43555949031 succeeded. Its G8 remains **81/90**.
Branch from PR67, retain ancestry; no merge to main and no native default.

## Implementasi

Reuse GroundVectorAt/sensor, LearnExperience, CognitiveProcess/Think,
ProcessObservation, ApplyNativeControl and observed outcomes. Only repair the
demonstrated sign collision in local-relational-v1: opt-in local-relational-v2
adds four signed counterparts of the existing four moments. No semantic roles,
new store, new learning law, implicit upgrade, or generic API wrapper.

Development precedes validation and confirmation. Record the frozen source
commit and runner/data hashes before confirmation. During A–D no source changes;
only input experiences and their observed consequences change. Preserve failures.

## Pengujian

Training lineage: one Brain from an empty valid checkpoint, deterministic
numeric grounding into distinct distributed populations, seed 37. Populations
are anonymous numeric adapter addresses, not built-in cognitive concepts.
Each stage saves its complete checkpoint; evaluation copies never replace the
main lineage or feed evaluation outcomes back into it.

| Curriculum | Experiences and local target | Confirmatory scope |
| --- | --- | --- |
| A: order | 8 balanced repetitions of two reversed prefixes ending in distinct populations; mean target-minus-competitor prediction >= .1 for each case and improvement over untrained | Fresh event IDs/times; simultaneous population-member permutation and novel irrelevant prefix |
| B: context | Continue A; 8 balanced repetitions of identical cue with two different contexts and outcomes; each margin >= .1 and improves over pre-B | Same contexts, new samples/permutations/prefixes; retain A |
| C: consequence | Continue B; opt-in v3 control with v2 encoder, propagate/stop; 48 balanced rounds, four amplitudes .3/.5/.7/.9; two states: activity before learned outgoing connections, or at outcome population | Fresh amplitudes .4/.6/.8; both alternatives physically execute in the numerical test environment on evaluation copies; mean optimal-action probability increases >= .1, final greedy accuracy >= 5/6, mean absolute fitted factual net-return advantage error decreases >= 30% |
| D: reuse | Continue C; measure two previously learned links in the same episode with an actually observed intermediate; then 8 balanced joint exposures, no new mechanism | Fresh prefix/permutation samples; both predictions correct with margin >= .1; report pre-D reuse separately; retain A/B/C |

C reward is measured activity in the specified outcome populations after an
actually executed operation, bounded to [0,1]. Compute cost .05 is deducted by
the existing learner. No hypothetical teacher return becomes observed feedback.
Pair trials have identical initial state and exogenous block; admission freezes
world learning until the pair is settled. LR .5, gamma/lambda 1, temperature .15,
epsilon .1, max steps 2, max active 4096, max reads 100000. No actuator is installed.
D's intermediate is supplied by the simulator as an observation, not a predicted
fact or an automatic neural composition claim. Probe missing-intermediate reuse
separately as exploratory evidence, without making it an A/B/C training failure.

Engineering acceptance: exact complete-continuation equality (canonical N,
all learning parameters, activation/context, provenance/dedup, sequence) after
genesis replay and every checkpoint+suffix replay; identical checkpoint checksum;
native-v1 historical tests unchanged; unknown encoder/dimension mismatch rejected;
policy-role injection rejected before append. Reuse Gate G admission, crash,
physical safety, architecture and historical compatibility regression tests.
New encoder active/pending checkpoint recovery and outcome dedup must pass.
Android arm64 cross-build must compile; no on-device performance claim.

Performance is a scoped encoder comparison, **not a G8 rerun**: fixed graph and
activity fixtures 1k/4k/40k, sparse .001/dense 1, single thread, 100 iterations,
3 repeats, allocs and structural reads. Lock exploratory alert at >2x median
ns/op or B/op for the opt-in eight-channel encoder vs old four-channel encoder;
investigate an alert; historical/default path must remain unchanged.

## Keputusan

Report each target separately, including retention. Engineering success does not
erase a training failure. One unexplained replay mismatch blocks engineering
acceptance. Classify failures: curriculum, representation, learning, integration,
retention, resource. Never tune production source to a training score. Freeze
another protocol/version before any new confirmatory experiment.

## Langkah berikutnya / rollback

Preserve main, production checkpoints and complete journals. V1 remains V1;
V2 requires new opt-in initialization on this training lineage. Existing models
are not reinitialized or migrated. An old binary rejects V2's explicit encoder
identifier; rollback uses a pre-V2 checkpoint plus its matching journal prefix.
Keep full raw logs/checkpoints and source hashes, commit coherent changes on the
research branch, publish a draft stacked PR without claiming G8 or default readiness.
