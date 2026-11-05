# V5 experimental scientific mechanism contract

Base main: 423a4a07e3e7dcd006cab7852d22b2d578312e18. Historical Phase X/XI
negative outcomes remain unchanged. No production Brain is imported/replaced.

## Scientific fidelity matrix before native fitting

| Principle | Existing mismatch | Minimal implementation | Verification |
|---|---|---|---|
| MUSE learned task-success assessment | Competence sigmoid but independent utility tanh | Reuse `metaUtilities(useMUSE=true)` for binary payoff; selected-outcome mixture BCE updates assessment row4 and plan-input weights; signed reward stays frozen | Domain, BCE derivatives, held-out Brier/NLL/support |
| MUSE competence-aware self-regulation | Feature connection alone did not demonstrate better regulation | Learned competence becomes the utility in Bayesian computation assessment and the one neural actor's inputs; actual observations/outcomes close the iterative learning loop | Causal interventions and measured decision outcomes; not a claim of benefit from connection alone |
| BMPS Eq2–5 and BO | Exact finite functions and GP/EI existed but coordinated V5 bypassed them | Reuse `SelectBMPS` scores/bounds, native belief and competence, stored `FitCognitiveBMPS` history; final action remains neural | Bounds, masks, costs, actual canonical weights, matched continuation audit |
| Chen recurrent actor–critic | V5 detached all58 features despite existing GRU backward | Recompute observed GRU and BPTT through complete actual history using existing native backward | 6,089 numerical derivatives, ownership, restart/crash and actual parameter changes |

Primary sources: MUSE https://arxiv.org/html/2411.13537v2 (journal DOI
10.1016/j.neunet.2025.108131); BMPS
https://auai.org/uai2018/proceedings/papers/269.pdf; Chen preprint
https://pmc.ncbi.nlm.nih.gov/articles/PMC13104915/ and author code
https://github.com/SixingChen028/recurrent-meta-reasoning .

MUSE's Dreamer example optimizes RSSM states. This patch does not port Dreamer
or claim an exact reproduction. Competence-aware planning and learned policy
regulation are Horizon's adaptation; efficacy still needs an outcome benefit.

## Explicit information, ownership and compatibility contract

Opt-in `NeuralMeta.Mode="scientific-fidelity-v5"`. Correct/wrong answer payoff
1/0 makes conditional success probability equal expected answer payoff. Abstain
is .35. No general signed-reward head is reinterpreted or clipped.

BMPS retains its original *best modeled terminal utility* assumption. Its score
is not claimed to equal the return achievable by an imperfect neural actor.
`BMPS` exposes information terms and weights, `VOC` its model approximation,
`ContinuationUtility` the separate terminal-only neural policy value, and
`PredictedJoint` only the prediction from N. Compare one-step/full actual
continuation separately; no oracle is used in inference or training.

Channels:0:32 observed GRU;32:37 competence;37:42 binary payoff;42:45 BMPS score;
45:48 cost;48 budget;49:52 myopic VOI;52:55 VPI;55:58 subset VPI. No new tensor
or neuron. DropMUSE removes direct assessment inputs, retaining its indirect
BMPS path. DropVOC removes all BMPS channels. Legacy-actor ablation is rejected.

Actor objective is on-policy pi over legal actions, full episodes only. Logged
draws/propensities must reproduce behavior. Return-to-go includes actual costs.
Advantages, critic targets and auxiliary assessment/BMPS inputs are detached.
Only the observed GRU block is recomputed/differentiated, including preceding
actual observations. No gradient passes through imaginary world branches.
This is a documented actor-critic semi-gradient adaptation, not full planner
differentiation. World decoder, old actor and signed reward are frozen.

Explicit `FidelityLearningV5` options are hashed into evidence. `AssessmentOnly`
uses q=.5*pi+.5*Uniform(legal terminal actions) for research acquisition; no
actor or observation-GRU update is made on those records. Only executed action
outcomes label competence. `FreezeRecurrence` is the matched actor control.
Assessment/recurrent Adam clocks continue the authenticated upstream clock;
the final head keeps its separate existing clock.

Checkpoint19/WAL13 topology is unchanged. New modes/options require the new
reader; older readers reject the unknown mode on WAL replay. Older checkpoints
and historical payloads retain their old route. Rollback uses the preserved
pre-experiment checkpoint/prefix, never truncated/repaired history. Default
runtime stays unchanged; directory fsync and ownership guards are preserved.

## Bounded preregistered native experiment

Seal source/protocol/SEALED_SPLITS before fitting. One authentic seed101 lineage.
256 train histories seed1120510010000,256 holdout seed1120510020000; depths0–3,
five contexts, original .015–.04 costs/payoff. Train excludes FNV history/context
buckets<4; holdout includes novel1/2 and seen groups. Not every prefix disjoint.

MUSE:64 batches x64 actual terminal outcomes=4,096. Existing LR/Adam, proper BCE,
no sweep or new capacity. Holdout scoring uses32 uniformly assigned, actually
executed answers/state, paired cases/actions before/after. Bootstrap2,000 states.
Continue actor pilot only if Brier AND NLL upper paired CI95<0, with no action
worsening>.005 Brier/.02 NLL. Preserve failure without extending training.

If justified, two actor arms start from the identical MUSE checkpoint/moments:
32 batches x16 actual episodes each, full BPTT versus frozen recurrence. Same
native MUSE/BMPS features, costs, initial state and environment streams. Training
seed1120560000000; evaluate512 new root cases greedily without forced operations
at seed1120550000000, plus frozen authentic Phase X parent. No bonus/shaping.

Whole-task gate stays +.02 realized net with paired lower CI95>0 against parent
and matched control. Reserve1,024 confirmation cases seed1120590000000; do not
collect/tune without qualifying pilot. No multi-seed before early benefit.
Unit/crash tests and forced acquisition are not evidence of autonomous efficacy.
