# V5 coordinated metareasoning — experimental, default off

This is an opt-in research route on the canonical Brain, not a replacement for
chat, language, device dispatch, existing profiles or production checkpoints.
No seed101 model is bundled or automatically imported. Decision benefits are
unproven until a separately sealed, paired experiment passes the net-return gate.

## Information and learning contract

Observed numeric trace -> canonical observation GRU -> learned Phase II world
posterior conditioned only on observed operation results -> MUSE action-specific
competence and expected terminal payoff -> Bayesian one-step value of computation
using the *same neural terminal policy* -> one final 58->16->8 actor and critic.
No old actor/critic logits enter this route; their9 former feature slots are zero.
Budget is a fraction of the three available computations. VOC is in reward units
and has no artificial multiplier. The known abstain payoff remains0.35.

MUSE selected-answer BCE and payoff regression learn from the actual0/1 terminal
reward. They do not receive the true class, counterfactual labels, or unexecuted
operation outcomes. World class hypotheses are integrated as a posterior mixture.
Abstain's payoff is a known environment contract, not a learned success label.

An atomic coordinated batch computes gradients at one parameter snapshot: the
unified head uses on-policy Monte Carlo actor-critic returns (including actual
operation charges); MUSE updates its immediate-success assessment row, reward
readout and recurrent terminal-plan input columns. Observation GRU, world decoder,
legacy actor/critic and unrelated N remain protected, including Adam moments.
These estimator and policy losses have an explicit stop-gradient boundary. This
bounded repair does not claim end-to-end gradients through Bayesian lookahead or
new learning of the observation GRU. Historical optimizer clocks are continued;
there is no guessed parameter or moment reset.

BMPS is restricted here to its finite Bayesian myopic continuation component:
E[payout of current terminal policy after the observed outcome] - current
terminal-policy payout - cost. It is neither perfect-information value nor an
optimal-controller oracle. Full non-myopic BMPS mixture/GP search remains in the
historical research APIs; it is not claimed validated for this coordinated route.
The terminal continuation masks operations and preserves the caller's legal
terminal options. It requires at least one legal terminal option. Exhausting
the budget in a simulated branch disables every computation in that branch.
It uses argmax in greedy mode and the actor probability mixture in stochastic
mode; no stochastic continuation is silently substituted during greedy evaluation.
Continuation uses the pre-VOC features to avoid recursive evaluation. The final
actor receives the computed VOC, so the two conditional terminal distributions
can differ. Treat that difference separately from the legality of either policy.
An informative outcome can still be unprofitable if this neural policy cannot use
it. Neural actor probabilities determine the final legal action; BMPS never
replaces that action. MUSE self-regulation is through these learned payoff and
computation estimates, not an independent selector or voting system.

## Opt-in native API and workbench

Use BrainRuntime.PreviewCoordinatedV5(observedTrace, drop) for read-only prediction.
Use ApplyCognitiveModel with exactly one Coordinated payload for observed learning.
The latter validates traces, budget/masks, actual costs and selected reward,
parameter hash and recomputed on-policy behavior before appending the redo WAL.

Build with `go build ./cmd/horizon-metareasoning-v5`. The workbench requires
`-experimental-v5 -state /explicit/isolated/profile` and exactly one of
`-request observed-trace.json` or `-command runtime-command.json`. There is no
default profile path, automatic import or ordinary runtime dispatch change.
An empty profile must explicitly initialize CognitiveModelV2, Phase2 and Phase4
via versioned runtime commands before using coordinated inference/learning.
Existing profiles restore from their exact checkpoint plus journal; journal-only
initialization after a crash is replayed, not silently replaced. Preview does not
write a new checkpoint. Never point a research trial at a user's only Brain copy.

## Versioning and rollback

Historical canonical schemas2–5, checkpoint schemas through18 and WAL through12
retain their meanings. A learned coordinated state carries a contract marker,
canonical schema6, checkpoint19 and explicit coordinated WAL13 commands. Wrong
contract, payload mixing and downgrades are rejected. Standard runtime commands
and profiles without initialization continue their historical schemas and path.

Code rollback can continue to read historical profiles. To revert a coordinated
profile with older code, retain its authenticated pre-coordination checkpoint and
complete journal prefix; older code must reject schema19 instead of dropping its
meaning. Do not truncate WALs or overwrite a production state with research data.

## Evidence

Native tests cover old GRU/BPTT and all1097 head derivatives, selected-outcome MUSE
numerical derivatives, canonical parameter ownership, frozen moments, explicit
mask/behavior/reward guards, causal feature interventions, cost accounting,
checkpoint19/WAL13 roundtrip, native redo and retry idempotency. The opt-in pilot
in experiment_test.go seals new splits and tests one candidate and one historical
control. Scientific success additionally needs at least+0.02 realized net return
versus abstain and a relevant control, with paired uncertainty excluding zero.
Technical integration alone must not be described as evidence of that benefit.

## Phase V exploratory training behavior

Coordinated V5 accepts the explicit stochastic mode `recurrent-explore-v5`.
For each state, the sampling policy is
`q = 0.75 * pi + 0.25 * Uniform(legal actions)`. Illegal actions have zero
mass. BMPS uses the same mixture for its pre-VOC stop continuation with the
terminal mask. Standard recurrent and greedy calls use the original policy.

The on-policy score is differentiated through the mixture:
`grad log q(a) = 0.75*pi(a)/q(a) * grad log pi(a)`. Critic targets use the
actual net return, entropy regularizes pi, and MUSE receives only the chosen
terminal outcome. This optimizes the exploratory behavior objective during
training. Greedy evaluation still tests the learned policy separately.

The existing trace mode field records this behavior in WAL13. A historical
reader rejects mixed trace replay as an unknown mode. Current readers retain
historical redo semantics and checkpoint19 uses the same parameter layout.
Phase V uses this opt-in mode to test an action visitation hypothesis.
Greater exploration or better calibration alone does not pass the net-return
gate.

## Phase VI target-policy credit

The optional `recurrent-target-v5` mode samples the Phase V mixture q on legal
actions. It learns from complete stochastic episodes. The runtime validates
each recorded behavior probability and reconstructs ordinary recurrent target
features, including its unmixed BMPS continuation, from the same observations.

For rho_t = pi(a_t|h_t)/q(a_t|h_t), the detached return is
G_t = r_t + rho_(t+1) G_(t+1), and the detached state-action weight is
W_t = product(rho_i, i <= t). The actor uses W_t (G_t - V_t) grad log pi_t.
The critic fits G_t with W_t. Entropy of pi uses W_(t-1) as a detached
visitation weight. This is a local entropy regularizer. It is not a gradient
through the occupancy distribution of an entropy-augmented objective. The
reward score is exact in expectation for stochastic pi with fixed features,
prior to finite-sample optimization and existing gradient clipping. Greedy
performance requires separate measurement.

The MUSE loss still uses only the selected terminal outcome, with the same
ownership and optimizer. Features and estimator ratios do not backpropagate
through the frozen recurrent substrate or the BMPS search. Mixed target and
legacy objectives in one batch are rejected. WAL13 records the new trace
mode and checkpoint19 keeps the established parameter layout. Readers that
do not support the new mode cannot apply its suffix. Existing modes retain
their behavior, and the new mode is opt-in.
