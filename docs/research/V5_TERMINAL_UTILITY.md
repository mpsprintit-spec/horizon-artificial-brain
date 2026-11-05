# V5 observed terminal utility alignment

The native runtime accepts an explicit experimental
`CoordinatedBatchV5.TerminalUtility` update. Ordinary inference and the
legacy coordinated learning path keep their existing contracts.

The protocol `v5-observed-terminal-utility-v1` learns the existing
58-16-8 head from terminal answer counts collected under uniform action
assignment. Each `TerminalUtilityCaseV5` provides a complete legal observed
recurrent trace and four `{Observed, Correct}` sufficient statistics.
The answers are actions 3–6. The protocol assumes q=.25 for those actions
and exact environmental abstain payoff .35. The dataset SHA-256 and the
evidence digest bind the raw source and the complete supplied batch.
Counts must come from executed actions, including unsuccessful trials.
Do not supply oracle labels or rewards for unexecuted alternatives.

For N observed assignments at a state, the answer target is
4*Correct_a/N. This is an inverse-propensity mean, which may be noisy
and need not sum to one in a finite batch. It is not a counterfactual
reward label. Abort or use a different declared protocol if assignment
was selected-policy or has another propensity.

Two explicit objectives are supported. `policy_value` minimizes
-sum pi_a Uhat_a. `logscore` minimizes -sum Uhat_a log pi_a.
Both retain the existing entropy regularizer. The second objective's
population minimizer without entropy is proportional to nonnegative
terminal payoffs. Its greedy ranking matches payoff ranking, while its
stochastic objective differs from expected-return policy gradient.

Only the existing 1,097 head parameters and their Adam moments update.
The critic has zero loss gradient but its inherited moments decay under
the common head clock. World, recurrent representation and MUSE
parameters/moments are frozen. No new neural units, parameters or actor
are added. Actor selection still uses native logits and legal masks.

The Phase X same-lineage pilot and independent terminal confirmation
found positive terminal-learning differences with observed outcomes.
Whole-task greedy mental-operation selection and the historical +.02
net-return gate failed. The mode remains explicit experimental training
and must not become a production default.

New runtimes retain checkpoint19/WAL13 and replay old events. New
terminal-utility events require code that understands this protocol.
An older binary must fail closed on that suffix. Rolling back such
research learning requires the pinned pre-update checkpoint and WAL,
or a compatible reader. Do not replace a production Brain with a
research checkpoint.

The experimental evidence, raw outcomes, seeds, ownership and normalized
replay receipts are delivered separately in the Phase X audit archive.
