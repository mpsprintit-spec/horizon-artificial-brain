# Blank-Brain Reasoning Lab — research only

This experiment lives on a separate branch based on
`research/native-phase3-20261108`. It must not be merged into `main`
or enabled as a production route without new evidence.

## Scope

Start from `knowledge.NewBrain()` with zero neurons and zero synapses.
Create unlabeled numeric neural addresses, then use actual `LearnNeuralTrace`
experience events to acquire 18 transitions within the *same* canonical
substrate. These traces establish the small world dynamics; they do not
constitute pretrained philosophical knowledge. The native v3 policy is
initialized with random canonical synapses only after experience acquisition.

Each training task provides numeric opposing-evidence populations plus
independent distractors. Native operations are `propagate`, `focus`,
`relax`, and `stop`. Randomized matched executions compare a first
propagation against stopping, focusing or relaxing, with identical suffix
decisions if both arms continue. A stop is absorbing. The adapter only
submits an observed scalar outcome once that trajectory finishes; no
evaluator goal or truth labels are passed to `NativePolicy`.
The actual outcome difference updates the same Brain's policy synapses.

This tests computation selection, delayed causal credit, and whether the
network stops when the available evidence conflicts. It **does not** prove
deductive logic, argument validity, autonomous philosophical inquiry,
concept grounding, unsupervised discovery, or universal reasoning.
The traces supply a simple, externally observed relational environment.
A true zero-knowledge learner would need to discover such structure from
raw sensor streams without an engineered observation adapter; that is NOT
claimed here.

## Run

From this branch, in a Go 1.22+ environment:

```sh
go test ./internal/nativefoundation -run '^TestReasoning' -count=1 -v
mkdir -p /tmp/horizon-reasoning
go run ./cmd/horizon-reasoning-lab \
  -out /tmp/horizon-reasoning/seed73 \
  -seed 73 -blocks 192 -cases 96
```

The selected output directory must not exist. The program saves
`metrics.json` and `brain-checkpoint.json` there; it never reads or
modifies production memory. Additional fresh seeds and directories
must be used for independent replication.

## Readouts and acceptance

The JSON includes the untrained model, trained model, fixed-operation
and random baselines on a held-out numerical task set; it separately
tests reversed dominant evidence and conflicting evidence. It records
initial Brain emptiness, experience transitions, training executions,
and changed canonical policy parameters.

Required scientific follow-ups before a claim of learned reasoning:

1. Preregister gate values, separate development and final holdout seeds.
2. Compare trained vs best static and random baselines with confidence
   intervals across independently initialized brains.
3. Check actual causal reasoning on novel graph topology, unseen
   numeric identities, invalid premises and simultaneous inhibition.
4. Test belief revision with new observation *inside one continuing
   episode*, not merely a separately evaluated reversal.
5. Attribute changes to canonical N by synaptic ablation; verify
   replay, crash recovery and checkpoint restore.
6. Demonstrate efficiency and absence of hidden answer leakage.

Any failed gate remains a result, not evidence of success.
