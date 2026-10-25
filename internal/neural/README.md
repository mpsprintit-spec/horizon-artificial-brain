# Horizon Neural Substrate

This package is the computational substrate for the single Horizon brain.

It intentionally contains no semantic vocabulary, fact table, intent classifier,
reasoning rules, or hard-coded knowledge. External experience is injected as
continuous numeric activity. Persistent state consists of neural and synaptic
state plus temporal traces and adaptive topology.

## Contract

- `Tick` advances persistent recurrent dynamics.
- `Learn` changes synaptic state from activity, eligibility and reward.
- `StructuralAdaptation` adapts topology from accumulated coactivity and usage.
- `Save`/`Load` persist the neural state; they are not a sentence/fact database.
- IDs identify runtime graph elements only; they do not represent concepts.

The substrate is not considered cognitively complete until representation
formation, prediction, pattern completion/separation, contradiction adaptation,
continual-learning retention, and embodied I/O are validated experimentally.
