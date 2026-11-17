# Owner recovery and reproduction

Use the complete archive linked in `ARTIFACT.md`. Verify the outer SHA-256,
then authenticate every payload without executing training:

```sh
python3 scripts/package_s2r4_temporal.py --verify /absolute/Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz
mkdir /absolute/restored-r4
tar -xJf /absolute/Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz -C /absolute/restored-r4
```

`MANIFEST.json` authenticates the complete byte length and SHA-256 of every
payload except itself. `BUNDLE.json` identifies the compatible scientific
source commit and R3 base. The original R3 owner archive is included unchanged
under `source/`, including the original R2 archive and full preceding lineage.
Verify/extract those nested manifests if restoring the complete historical
source tree, rather than only a final research checkpoint.

Use the compatible Git research source commit identified by **BUNDLE.json**,
or apply `implementation.patch` to exact R3
`6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c`. Later delivery commits add reports,
receipts and artifact pointers. Scientific source hashes remain frozen.
Older binaries must reject the explicitly new proposal protocols/WAL15;
do not use an old reader that silently ignores an environmental gain.

| Purpose | Checkpoint | Complete native journal |
|---|---|---|
| Original rollback/source | `s2r4/00-source.json` | `s2r4/source-events.jsonl` |
| Fixed final seed/arm | `s2r4/training/seed-S/ARM/final.json` | Same directory `events.jsonl` |
| Intermediate b=0..3 | Same directory `block-b.json` | Same full journal; full recovery advances to final. Use a checkpoint-only detached evaluation copy to inspect an intermediate state. Never truncate historical WAL. |
| Exact restarted state | Same directory `restarted.json` | Same full journal |
| Native observation sequence | `observer-persistent/observer-final.json` or IID counterpart | Corresponding full `events.jsonl` |
| Final observation-only state | `s2r4/holdout/CONDITION/observer-final.json` | Same directory `events.jsonl` |

S = 37/83/151; ARM = A32/B32/C32/B0-32/D32/D0-32/A64/C64.
No arm is installed as production or declared a canonical successor.
All are fixed final lesson-4 results, including negative and retention-failing
arms. Native policy counters and full historical provenance are preserved.

Open a copied journal with `OpenEventLog`, then recover with
`RebuildRuntimeFromCheckpointAndJournal(log,checkpointPath)`. Attach it with
`SetEventLog` only when intentionally continuing an isolated copy. Restoring
80735 against an arm's complete journal must reproduce that arm's final state
exactly. `persistence.json` records all per-arm restart/replay comparisons.
Do not modify the source checkpoint/journal or overwrite production.

History is an experience-adapter buffer, reset at stream boundaries. Its input
values and target provenance persist in observation files, pre-operation
ledgers, environment records and learning WAL. The experiment does not establish
autonomous recovery of an in-progress history buffer from neural memory.
All final streams are closed; that limitation does not change neural
checkpoint/WAL recovery or the saved weights.

`development-v1/` and `development-v2/` preserve exploratory failures;
`diagnosis-v3/` preserves the full earlier reduced-state experiment and its
full-state correction, scores, complete recovery proofs and time-control audit.
`s2r4/development/` is the final complete-state development revision. Initial unscored final
data and mask-preflight predictions are explicitly named and must not be mixed
with the primary full-state `s2r4/holdout/` set. `PROTOCOL.md`,
`CONFIRMATORY_PROTOCOL.md` and `FULL_STATE_PROTOCOL.md` explain every revision. No failed experiment is
deleted to improve a reported score.

For **fresh independent reproduction**, use a new disjoint work directory and
the verified extracted R2 source tree. Go 1.22.12 and Python/NumPy were used:

```sh
python3 scripts/run_s2r4_temporal.py --source /absolute/extracted-r2 --work /absolute/new-r4 --go /absolute/go/bin/go --phase development
python3 scripts/analyze_s2r4_fullstate.py --work /absolute/new-r4 --output /absolute/new-r4/analysis --phase development
python3 scripts/run_s2r4_temporal.py --source /absolute/extracted-r2 --work /absolute/new-r4 --go /absolute/go/bin/go --phase development-learning
```

Register the same training protocol in the new work directory before training;
the saved `TRAINING_LOCK.json` supplies exact settings and its protocol hash.
Run `--phase train` for the three registered seeds. Use the **fullstate** analysis
script for every primary phase, including Fit-only `--phase interventions`; commit the scientific source,
and run `freeze_s2r4_temporal.py --work ...` before `--phase acquire-final`.
Then generate the target-blind input interventions, run `--phase evaluate-final`,
fullstate analysis `--phase final`, and `--phase persistence`. The optional
`parallel_s2r4_persistence.py` runs each complete model proof independently
and merges all 24 results without reducing tests. Behavioral oracle images
omit adapter dedup metadata only, preserve the full canonical Brain/sequence,
and pass an exact parity check; they are evaluation-only, not recovery files.
Phase logs cannot be
overwritten. One-shot acquisition cannot be repeated in the same fresh work.
Saved original locks contain original absolute provenance paths; regenerate
locks for a fresh reproduction rather than editing them to bypass validation.
No additional acquisition or retraining is needed merely to recover/audit
the saved research result.

## Published source and exact local history

The original frozen scientific commit is `cee0fa80dfa1667ee2738c3467463eb1fb8313ab`.
Published commit `262ab99b10a5aa7bdb22b2821c71b0803272e60a` has its identical tree.
Use either source for reproduction. To recover the original commit IDs, start
from a repository containing exact R3 base `6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c`,
then fetch the small bundle delivered alongside this document:

```sh
git bundle verify docs/research/s2_r4_temporal_20261117/S2R4_Source_History_20261117.bundle
git fetch docs/research/s2_r4_temporal_20261117/S2R4_Source_History_20261117.bundle refs/heads/research/s2-r4-temporal-error-history-20261117:refs/remotes/recovery/s2r4
git checkout cee0fa80dfa1667ee2738c3467463eb1fb8313ab
```

`PUBLICATION_MAPPING.json` authenticates the bundle and records both identical
tree mappings. Later delivery commits change documentation only. The complete
research archive and its manifest remain unchanged by this publication step.
