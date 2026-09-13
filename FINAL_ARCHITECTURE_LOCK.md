# Horizon — Final Architecture Lock

## Status

This document defines the architectural boundary for Horizon. It is the controlling architectural contract for all future implementation work. It is not a versioned prototype specification.

**Architecture is now locked. Future changes must evolve the neural substrate without reintroducing the retired rigid cognitive architecture.**

## Core principle

Horizon is one continually learning distributed artificial brain. Intelligence is an emergent property of its persistent neural network dynamics, not a collection of hard-coded cognitive engines.

## Architectural invariants

1. Knowledge is represented as distributed neural activity and long-lived network structure.
2. A node is not a concept or fact. Concepts are population/distributed patterns.
3. Synapses carry adaptive state and temporal/history information; a single static weight is insufficient as the final conceptual model.
4. Learning modifies existing structure and forms new structure only when required.
5. Repeated experience reuses and modifies existing representations rather than creating independent copies.
6. Activation and propagation are continuous and stateful.
7. Plasticity is gradual and subject to stability/homeostatic constraints.
8. Decay is distinct from contradiction and does not mean deletion of knowledge.
9. Contradictory experience causes network adaptation/reconsolidation while preserving relevant historical traces.
10. No hard-coded semantic relation types such as IS_A, HAS, or CAN_DO are part of the intelligence substrate.
11. No Trigger Engine, Knowledge Engine, Reasoning Engine, Critical Thinking Engine, or equivalent cognitive rule engine is part of the intelligence architecture.
12. Language, reasoning, abstraction, critical evaluation, prediction, and other higher capabilities must emerge from learned network dynamics.
13. Sensors and actuators are interfaces to the brain, not sources of cognition.
14. LLMs are not part of Horizon's intelligence or long-term memory.
15. One brain identity and one persistent neural knowledge substrate are maintained.
16. Implementation optimization may change data structures, scheduling, parallelization, and storage representation, but may not violate these invariants.
17. Thinking is an internal recurrent process and must not require a new external input or a mandatory output.
18. Internal thought may continue through activation, association, prediction, simulation, evaluation, uncertainty, memory reconstruction, and new internal states without a hard-coded sequence of cognitive stages.
19. The thinking path must operate on the same persistent neural substrate; it must not switch to a separate answer database, semantic rule table, or legacy cognitive pipeline.
20. Repeated thought must modify and reuse the same distributed substrate rather than creating a parallel representation of the same knowledge.

## Retired architecture boundary

The previous rigid, component-driven cognitive architecture is **retired from the thinking path**.

The following are not permitted to participate in Horizon's cognitive process:

- hard-coded semantic parsing as the source of cognition
- fixed semantic relation vocabularies as the basis of thought
- rule-based Trigger/Reasoning/Critical-Thinking/Knowledge engines
- answer-selection pipelines that map a cue directly to a predefined response
- separate knowledge stores that duplicate the brain's long-term knowledge
- isolated modality-specific AIs acting as independent intelligence
- LLMs used as Horizon's internal reasoning or long-term memory
- any compatibility layer that silently becomes part of the recurrent thinking loop

Legacy implementations may remain temporarily for migration, compatibility, inspection, data conversion, or controlled experiments. **Legacy code is not architecture.** Its presence in the repository does not authorize its use by the brain's thinking path.

Any new cognitive mechanism must be implemented at the neural-substrate level and must preserve the invariants above. If an old component conflicts with this lock, the old component must be migrated, isolated, or retired rather than bending the new architecture around it.

## Thinking boundary

Horizon thinking is defined as:

> **Thinking = organized recurrent changes in internal state in which the network uses learned experience to form, maintain, modify, and evaluate possible states without requiring external stimulus or output.**

The canonical internal process is therefore substrate-driven:

```text
persistent neural substrate
        ↓
current internal state
        ↓
recurrent activation / association
        ↓
prediction / simulation / reconstruction
        ↓
evaluation / uncertainty / conflict
        ↓
new internal state
        ↓
recurrent continuation
        ↺
```

This is not a mandatory cognitive pipeline. The sequence above describes the class of dynamics Horizon must be able to support; the actual transitions must emerge from learned network state and adaptive dynamics.

## Required substrate capabilities

- neural state dynamics
- sparse distributed activation
- recurrent propagation
- adaptive synaptic state
- activity-dependent plasticity
- temporal traces / eligibility
- homeostatic regulation
- structural growth and pruning
- pattern formation and stabilization
- pattern separation and completion
- continual learning
- prediction-error-driven adaptation
- contradiction/reconsolidation
- persistent state
- deterministic/reproducible experimentation
- scalable execution

## Change-control rule

Every future architecture or implementation change must answer these questions before being accepted:

1. Does it preserve one Horizon brain identity?
2. Does it operate on the persistent neural substrate rather than create a parallel cognitive memory?
3. Does it allow cognition to emerge from network dynamics rather than hard-coded semantic rules?
4. Can internal thought continue without external input?
5. Does learning strengthen/restructure existing distributed representations instead of duplicating them?
6. Does the change preserve temporal dynamics, plasticity, uncertainty, contradiction handling, and historical traces?
7. If legacy code is involved, is it explicitly isolated from the thinking path?

A change that fails these checks is an architectural regression, even if it improves a benchmark, demo, API, or test in the short term.

## Completion standard

The project is not complete when it compiles, passes superficial tests, or produces convincing demonstrations. Completion requires implementation-level evidence that the substrate satisfies the architectural invariants and behavioral tests, with known failures either eliminated or explicitly blocking completion.

## Lock interpretation

When a future implementation decision conflicts with an older design, **this document takes precedence**. The correct direction is to evolve the distributed neural substrate, not to restore the retired rigid cognitive architecture.
