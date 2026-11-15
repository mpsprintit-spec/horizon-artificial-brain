# Native architecture retirement evidence

Baseline: `6b6db6c9ec0f373fed51f38a415d0a36baf7e812`.
Archive: `archive/pre-canonical-cleanup-20261115`.

The implementation removes 103 obsolete source, test and package documentation
files. The inventory, removed APIs and final production dependency graph are
recorded alongside this report. All 115 protected source/artifact files have
identical SHA-256 hashes to the baseline. Native initialization only loses two
retired-model guards and an obsolete diagnostic; all other bytes in that file
and the physical-memory update law remain unchanged.

## Completed local validation

- Entire Go regression suite with the saved PR69 A–E1 lineage and trained dialogue fixtures.
- Build and vet for all Go packages; tests for all 15 nested adapter modules.
- Bridge integration smoke, Android arm64 chat build, fixture-preparation tests.
- Race tests for retirement boundaries, DNF compatibility, canonical facade/CLI,
  mixed replay, process-crash recovery and ownership protection.
- No retired package in the dependency closure of five production executable roots.

The saved continuation test compares complete state, checksum, readouts and
next-thought continuation against the unchanged PR69 oracle. The production
Brain, saved fixture, frozen oracle and G8 harness/criteria are not modified.

## Merge prerequisites still being collected

The original frozen G8 confirmation and GitHub CI results will be attached before
any merge. The source commit can be reviewed while these checks run.

Local worktree Go builds use explicit `GIT_DIR`/`GIT_WORK_TREE` to select the
correct checkout metadata; no VCS/build check is disabled. The dependency-query
test disables executable VCS stamping because it inspects only the import graph.
