# Final native engineering closure — frozen criteria

Baseline main: 3a9e7ce3ffadff86c792050056ebea4eda48da0b. Dependency source:
PR69 a3ce74c0281f853412134e16ccd414c7046a9e7e plus verified PR63
cee28ff2431b2a5f5d2ea61c668c4de262c25426, preserving merge ancestry.
Authentic performance baseline: 609ee918aa8533e0aca5706dea6046043bdc09bc.

## Fixed engineering worklist

The nine failures in docs/research/gate_g/PERFORMANCE.md remain authoritative:
thought 1k sparse with/without WAL; thought 1k dense without WAL; thought 4k
dense with WAL; native learning 4k sparse/dense; checkpoint 4k; native-idle
thought 4k dense with WAL and 40k sparse without WAL.

Reuse existing profiles and measure development changes separately. Candidate
repairs remove transient map growth/copies, per-edge slice allocations and
duplicate JSON encoding, never validation, append/fsync or canonical state.
No learned parameter cache or alternate substrate. A frozen historical advance
implementation and exact JSON comparisons must prove numerical/byte equality.

Confirmation uses the unchanged Gate G runner, common benchmark harness and
summarizer: Go1.23.2, GOMAXPROCS=1, GOGC=100, sequential ABBA. All 90 comparisons
must pass the original budgets: thought p95 <= baseline*1.15+.10ms and bytes
<= baseline*1.05+4096; native/persistence <= baseline*1.20+.10ms and bytes
<= baseline*1.10+4096; admission <=10us and512 B/op. Existing sample counts,
fixtures and seeds stay fixed. Keep every negative result. A new patch requires
a new source freeze and complete confirmation, not selection among finished runs.

## Preservation and functional checks

Use PR69's saved checkpoint75331 /10752 updates and its complete journal.
Verify original archive/checkpoint/journal hashes before loading. No A–D/E1
training on the continuation Brain; no reinitialization or implicit migration.
Compare full canonical/activation/continuation state, genesis replay,
checkpoint+suffix, old checksum, and existing A–E1 probes against the saved
results. Evaluation uses detached copies; it cannot replace the primary lineage.

Run ownership/admission/legacy, mixed transactions, old readers/rollback,
subprocess crashes and physical replay tests, full Go regression/race, vet,
build, Android arm64 cross-build, and CI on final integration source. Separate
historical acquisition harnesses from saved-state integration checks; do not
change learning laws or claimed training results. Report hardware Android and
hardware power-loss as untested if unavailable.

No controller default change. No production memory/checkpoint overwrite. Merge
is allowed by this request only after every mandatory integration gate passes;
otherwise publish the repaired branch and concrete remaining blockers.
