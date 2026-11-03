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
The terminal continuation masks operations, preserving all answers and abstain.
It uses argmax in greedy mode and the actor probability mixture in stochastic
mode; no stochastic continuation is silently substituted during greedy evaluation.
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
