# Empty-Brain Numeric Reasoning — 11 November 2026 development record

Research branch: `research/blank-brain-reasoning-20261111`
Draft PR: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/66
Base: `research/native-phase3-20261108`; main unchanged.

The experiment really ran in isolated GitHub Actions using the native
Go BrainRuntime. It starts from zero units/synapses, acquires 18 numeric
temporal experiences and trains by paired executed operation outcomes.
It does not use a language model, philosophical vocabulary, or
handwritten logical conclusion routine.

## Development pilot: 192 matched training pairs (384 executed episodes)

- Run: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43028218837
- Checkpoint/metrics artifact: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43028218837/artifacts/13858941720
- Seed 73; 96 evaluation tasks.
- Net return before: 0.088749; after: 0.136330.
- Accuracy before: 33.33%; after: 35.42%.
- Strongest fixed baseline was always-stop: 0.175668;
  one-propagation baseline: 0.167052.
- Updated canonical policy parameters: 448.
- Trajectories truncated: 47.92%.
- Conflict-only holdout: net return -0.717687, reported accuracy 0%
  under initial scoring.

**Gate FAILED:** trained policy did not beat simple fixed strategies
and did not reliably deal with conflicting evidence.

## Development extension: 2,048 matched pairs (4,096 executed episodes)

- Run: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43028571470
- Checkpoint/metrics artifact: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43028571470/artifacts/13860125857
- Same development seed/evaluation set; do not call this an independent holdout.
- Net return after: -0.175000 (before remained 0.088749).
- Every evaluation episode executed 7 non-stop operations,
  truncated 100%, and earned zero raw feedback before operation costs.
- Conflict-only accuracy was falsely shown as 100% because the metric
  accepted zero activation after exhausted computation as an abstention.
  **This is an invalid success claim.**

The scoring loophole was repaired in the later commit:
a valid correct outcome requires an explicit native stop and
must not be truncated. The two archival runs are kept as measured
development failures; their historical metric files are not silently
rewritten.

## Conclusions

1. Real internal plasticity occurred, but it is insufficient proof of
   reliable reasoning.
2. Blindly increasing the episode count produced policy collapse rather
   than improvement.
3. Paired first-action credit does not yet guarantee stable closed-loop
   stopping at subsequent internal states.
4. The current environment tests numeric causal propagation and evidence
   conflict as controlled proxies, not philosophical deduction.
5. Development on seed 73 is contaminated for final confirmation. New
   seeds, locked gates, independent baselines, signed/temporal evidence,
   causal ablation, replay and transfer tests are required.

**Promotion decision: REJECT as a proven reasoning mechanism; RETAIN as
isolated research evidence.** No main merge, no production integration.
