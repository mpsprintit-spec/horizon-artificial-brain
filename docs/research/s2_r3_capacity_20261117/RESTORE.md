# Owner recovery and reproduction

Use the artifact linked by `ARTIFACT.md`. Verify its outer SHA-256, then:

```sh
python3 scripts/package_s2r3_capacity.py --verify /absolute/Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz
mkdir /absolute/r3-restored
tar -xJf /absolute/Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz -C /absolute/r3-restored
```

The self-authenticating `MANIFEST.json` contains every complete file's byte count
and SHA-256. `source/` includes the original full R2 owner archive unchanged,
not just its hash. Verify and extract that archive separately to obtain original
80735 and the entire prior lineage. Do not overwrite production state.

Use the compatible source commit in `BUNDLE.json` from the research branch, or
apply `implementation.patch` to exact #76 base. `reproduce/` contains standalone
phase/analysis/packaging scripts; the Git research branch owns the full code.

| Purpose | Checkpoint | Full journal |
| --- | --- | --- |
| Original R3 starting point / rollback | `s2r3/00-source.json` | `s2r3/source-events.jsonl` |
| Seed s, fixed arm c | `s2r3/seed-s/arm-c/final.json` | `s2r3/seed-s/arm-c/events.jsonl` |
| Every intermediate block | `s2r3/seed-s/arm-c/block-b.json` | Same full arm journal; it will replay later events, so use the prefix through that block sequence when intentionally restoring a historical block |
| Exact restarted model | `s2r3/seed-s/arm-c/restarted.json` | Same full arm journal |
| Native observations | Each `observer/final.json` | Corresponding `observer/events.jsonl` |
| Positive/negative isolated controls | `s2r3/information-controls/sS-positiveBOOL-cC/final.json` | Its `events.jsonl` |
| Recorded learning cost reproduction | `.../learning-cost-reproduction/final.json` | Its complete `events.jsonl` |

Here s=37/83/151, c=32/64, b=0..7. `final.json` is always block 7, never selected
by holdout score. `final-observer` is an evaluation observer, not a promoted
trained Brain. Native policy updates remain 10752.

Open a copied journal with `OpenEventLog`, recover using
`RebuildRuntimeFromCheckpointAndJournal(log,checkpointPath)`, then `SetEventLog`
only if continuing that isolated copy. Full suffix replay from source 80735 and
exact restart/rollback are in `s2r3/persistence.json`. Enlarged topology requires
compatible code and explicit restored capacity metadata; older binaries need
not accept it. Old 32 checkpoints remain supported by the new implementation.

To reproduce the entire experiment on a **new disjoint work directory**:

```sh
python3 scripts/run_s2r3_capacity.py --source /absolute/extracted-r2 --work /absolute/new-r3 --go /absolute/go/bin/go --phase all
python3 scripts/analyze_s2r3_capacity.py --work /absolute/new-r3 --output /absolute/new-r3/analysis
```

Go 1.22+ and Python/NumPy are needed. Saved phases and `FINAL_STARTED` prevent
accidental re-acquisition/overwrite. No need to re-acquire data to recover or
audit the saved result: all actual observations, pre-operation ledgers, full
Fit/Check, prediction arrays, intervention results and receipts are included.
The cost phase reexecutes an already recorded transaction exactly on a new copy;
it acquires zero additional experiences. The failed initial environment preflight
acquired zero samples and is preserved separately.
