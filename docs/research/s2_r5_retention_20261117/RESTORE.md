# Complete owner recovery and reproduction

Verify the combined archive outer SHA-256 from ARTIFACT.md, then authenticate
all payloads with `python3 scripts/package_s2r5_retention.py --verify ARCHIVE`.
The manifest covers full checkpoints and complete historical/new journals,
observations, pre-operation ledgers, actual neural poststates, source archives,
all failed development runs, solver traces, native receipts, configuration
locks and final statistics. No historical WAL was truncated.

Extract in a new directory. `BUNDLE.json` identifies the exact scientific
source commit/tree. `scientific-source.tar.gz` contains the entire compatible
source tree; no GitHub download is required to build it. The implementation
patch applies to exact R4 0069e6eb27dbe9334746dd80cc31e5ed52214867.
Source archives include the original complete R4 artifact and the original R2
artifact. Authenticate and extract the R2 archive into a disjoint source root
if running the opt-in research harness. Its native Brain is sequence 80735,
checksum 9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3.

| Purpose | Checkpoint | Full journal |
|---|---|---|
| Original rollback | r5/00-source.json | r5/source-events.jsonl |
| Saved final model | r5/training/seed-S/ARM/final.json | Same folder events.jsonl |
| Restarted final model | Same folder restarted.json | Same full journal |
| Source copy for each arm | Same folder state.json | Same full journal |
| Actual training observations | observer-persistent/observer-final.json or independent equivalent | Same observer folder events.jsonl |
| New final observations | r5/holdout/CONDITION/observer-final.json | Same folder events.jsonl |
| Additional retention | r5/new-retention-0/observer-final.json or -1 | Same folder events.jsonl |

Seeds: 37/83/151/211/307/419. Arms: A/A0/B/B0/C/C0/Cw/Cw0/D/D0/E/E0,
AI/AI0/DI/DI0/EI/EI0. None is a canonical successor or production replacement.
Use the compatible source, `OpenEventLog`, and
`RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)` on COPIED files.
80735 plus an arm's full WAL must reconstruct exactly its saved final state.
Checkpoint-only detached evaluation is distinct from full recovery. Oracle
images omit adapter dedup metadata only; never use them as continuation files.

Fresh reproduction requires Go 1.22.12 (or a separately validated compatible
version), Python+NumPy, an empty isolated work root and authenticated source.
The wrappers reject existing logs/forks and never change main or production.
Run prepare (with extracted --r4), development, fixed, anchors, coverage,
structure, capacity, negative-block, then copy the frozen TRAINING_LOCK.json
into the new work root with current paths/protocol hashes. `--phase train`
runs the six fixed seeds; `--jobs 3` matches the original training concurrency.
Use `--phase calibration-controls` and `--phase preflight`; freeze Fit-only
statistics with `analyze_s2r5_retention.py --phase freeze-metrics`. Commit the
compatible source, freeze it with `freeze_s2r5_retention.py --binary ...`, then
run acquire-final once, construct target-blind interventions, evaluate-final,
final analysis, and persistence for all seeds. The saved locks contain original
absolute provenance paths: regenerate them for fresh reproduction; do not edit
the saved locks to bypass verification. Recovering the existing saved models
does not require retraining or acquisition.

History is a finite experience-adapter buffer whose measurements/provenance
are saved before the target operation. All streams are closed. This research
does not claim autonomous neural recovery of an interrupted adapter buffer.
The zero-block control is degenerate; IID equivalence is a separate test.
Native solver search is finite and is not a global infeasibility certificate.
The original dyadic-reconstruction failure remains archived and is described
in ENGINEERING_FAILURES.md. No failed run was replaced by a positive-only report.
