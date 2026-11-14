# Gate G — locked integration protocol (2026-11-13)

## Scope and hypothesis

Replace blanket native/production isolation with episode-consistency admission,
without a second Brain, parameter store, journal, controller or physical gateway.
Hypothesis: idle production/native transitions compose exactly through canonical N;
open episodes and factual pairs require exclusive neural mutation until settlement.
Native remains explicit opt-in. Gate C/D/H scientific results are not retested or
promoted by this integration. All experiment files use fresh temporary paths.

## Authentic baseline

GitHub main: `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.
Phase III: `0df3f19a33bd7de69ed496293591d734331c5e51` (PR64 draft).
PR60 merged `32df59b`; PR65 merged `45ecb98`, included in Phase III.
PR61/62/64 remain draft; PR63 is unmerged `1837968` and is included as a
separate merge ancestor for its verified production startup correction.
Full source metadata and previous scientific limitations are in BASELINE.json.
Existing worktrees and their uncommitted benchmark files are retained untouched.

## Transaction inventory and admission

| Command family | Reads / writes | Uninitialized / idle | Active / stopped awaiting outcome / pending pair | Provenance and durability |
|---|---|---|---|---|
| Native init/begin/step/outcome/discard | canonical policy synapses, activation, episode/eligibility/contrast | existing native preflight | existing native preflight only | WAL14, command digest; source/outcome single use; observed reward only |
| Sensor intent/commit | projection, activation, world synapses, temporal predecessor, predictions | accept | reject before intent/projection | existing schema3/4, sample ID, matching intent hash, one sequence |
| Process / cognitive process / written observation | same N, activation, projection, evidence, prediction | accept | reject before append/grounding | schema3/4, event ID and observation source/time |
| Think / cognitive think | recurrent activation, synapses, memory dynamics, curiosity | accept | reject before append | schema3/4, recorded time/cycles |
| Inquiry outcome / observed outcome / consequence / action binding | action provenance, canonical evidence, synapses, inquiry policy | accept | reject before append | existing schema3/4, authorization remains external; no dispatch on replay |
| Learn / spelling / composition exposure or perception | canonical synapses, existing composition state | accept if current contract validates | reject before append | existing schema3/4, typed experience/exposure IDs |
| Numeric grounding | canonical projection | accept | reject before append/projection | existing vector command, event time |
| Knowledge import | canonical projection/learning, multiple exposure microsteps | accept only with its existing deterministic contract | reject before import | existing one-revision import contract |
| Cognitive V5 controller | historical independent experimental policy | existing behavior without native; reject with native | reject | no silent fallback |
| Written/composition migration or rebinding | serving contract and canonical projection | existing behavior without native; initially reject with native until verified | reject | explicit migration/backup transaction, never implicit |
| Checkpoint | reads complete N/runtime/dedup/pending state | accept under runtime lock | accept under runtime lock | schema20 for native, prefix anchor, atomic rename + directory fsync |
| Read-only inspection/inquiry capture | reads detached state | accept | accept if truly read only | no journal revision or device permission |
| Recovery-required | unavailable live runtime | reject mutation/checkpoint | reject | supervisor privately rebuilds and publishes only validated result |
| Replay / checkpoint / migration in progress | private recovery or exclusive owner lock | no concurrent live mutation admitted by contract | episode rules still apply during replay | no actuator calls, failed replay must not become serving state |

This matrix is a transaction consistency policy, not a cognitive decision policy.
Direct detached legacy mutators must also reject while native owns an episode.
No persistent queue is introduced. Current complete journal/checkpoint containers
are retained unless a demonstrated incompatibility requires an explicit amendment.

## Development, validation, confirmation

1. Development: reproduce isolation/bypass failures; implement minimal fixes.
2. Validation: targeted adversarial cases, complete-state replay, compatibility.
3. Confirmation: frozen implementation, full specified matrix and crash subprocesses;
   preserve failed logs and any subsequent amendments. Do not tune seeds/fixtures.

Deterministic fixture seeds: 37, 83, 151. Main small fixtures: seven world units
plus real sensor/text populations and canonical policy units (native v1/v2/v3).
Performance fixtures: 1,000 / 4,000 / 40,000 world neurons, four synapses each;
sparse (nine initial active units) and dense (all world units). Record actual sizes.

## Required replay / continuity cases

Sensor→native; native→sensor; sensor→inquiry→native;
inquiry→native→observed outcome; written observation→native;
recurrent thought→native; native learning→checkpoint→sensor;
checkpoint→restart→native continuation. Compare genesis and every selected
checkpoint suffix including active, stopped, pending and resolved contrast.
Compare complete canonical Brain bytes, activation snapshot, runtime continuation
bytes (including dedup/action/evidence/composition), sequence and checksum, exactly.
Checkpoint envelope write time and ephemeral telemetry/inspection/cache identities
may differ; they are excluded only if no neural transition reads them. Durable
timestamps, pending contrast, policy weights and dedup must never be excluded.
Assert the same Brain pointer and all engine memory owners throughout live work.
Use the genuine sensor input state for begin; do not inject a replacement fixture
observation in the sensor-to-native test. Require learned synaptic changes and
subsequent live/recovered continuation equivalence, not just serialization success.

## Crash subprocess matrix and expected durable state

| Boundary | Expected recovery |
|---|---|
| payload preparation / before append | previous committed state |
| after write, before fsync | if complete bytes survive process exit, redo once; this does not prove power-loss durability |
| after fsync, before apply | redo complete transition once |
| after apply, before acknowledgment | redo once into fresh runtime; exact retry deduplicates |
| sensor intent without commit | strict rejection, explicit backed-up tail repair, then previous committed state |
| active native episode | recover same active trajectory, continue exactly |
| pending contrast / first outcome | recover pending pair and block external mutation |
| second outcome durable | exactly one learning update and consumed pair/outcome dedup |
| before checkpoint rename | previous checkpoint + committed suffix |
| after rename + directory sync | new checkpoint + committed suffix |
| supervisor rebuild | publish no partial runtime; restart recovers same complete state |

Use actual os.Exit subprocess faults, unique files, raw logs and checksums.
Corrupt/truncated records must fail closed; no successful acknowledgment fabricated.
Mock physical executor counts live dispatches; recovery must add exactly zero.

## Numeric resource budgets (locked before confirmatory benchmarks)

Environment: Go1.23.2, GOMAXPROCS=1, GOGC=100, same host, sequential ABBA.
Development profiling may precede confirmation. Confirmatory existing-thought
regression: mean of per-run p95 <= baseline*1.15 + 0.10ms, allocations <=
baseline*1.05 + 4096B/op. Admission p95 <=10us, <=512B/op; no network-size
scan or full native episode clone for admission. Native step/outcome/checkpoint/
restore regression budget <=baseline*1.20 + 0.10ms and bytes <=baseline*1.10
+4096B/op for matched workloads. Sensor-to-native handoff admission must remain
O(1); total full boundary recorded separately. Report p50/p95/p99, allocation
bytes/counts, GC count/pause, and process peak RSS. At least 100 observations per
thought cell/run; persistence >=3 observations per cell, explicitly descriptive.
No baseline speed claim for a mixed workload that baseline refuses to execute.
No budget obtained by turning off durability, validation or learning.

## Acceptance / failure / rollback

G1–G10 evaluated independently. Determinism requires 100% exact matches in all
valid registered cases; an unexplained mismatch blocks Gate G. Missing required
coverage or a failed resource budget also blocks a whole-gate pass.
Run full Go tests, full race tests, vet/build, nested adapter checks from CI,
Android arm64 cross-build and exact-head remote CI. Cross-build is not a device
benchmark. Architecture tests inspect the active package dependency closure and
runtime behavior, without globally banning migration/diagnostic packages.
Rollback uses the untouched pre-opt-in checkpoint and its matching journal prefix
under a pinned old binary; archive the complete later journal, never silently
truncate history. Older readers may reject mixed native history; rejection is
safer than silently dropping native transitions. No automatic V5→native migration.

## Development findings and confirmation freeze amendment

Before confirmation, the development probes exposed (and retained in evidence):

- Existing sensor/vector/import paths bypassed the global journal-command guard.
- Generic world confidence decay made native policy checkpoints fail restore;
  policy-parameter ownership must be honored inside the single canonical graph.
- A resolved factual-pair receipt was missing from prefix-index reconstruction.
- Invalid second action-trial input/block was previously rejected only at outcome.
  Reject these start mismatches before append; the original negative probes remain
  and now assert earlier atomic rejection. Invalid continuation still fails at
  outcome. No failed case is removed or relabeled as a valid learned sample.
- Initial test-harness errors (attempting to bind an already-bound genesis Brain,
  omitted checkpoint envelope/anchor fields in a checksum comparison, and trying
  to begin an episode after a deliberately extinguished activation) are retained.
  Their fixes use an exact serialized genesis, the complete authenticated envelope,
  and a new real production input. Valid crash and replay scenarios are retained.

Confirmation adds native→recurrent thought→native as a ninth mixed order; three
native model versions, enabled composition, the actual production startup path,
facade legacy isolation, corrupt records, and an explicit matched-prefix rollback.
The composition case uses two small existing-format exposures only; no scientific
Gate C/D curriculum is repeated. Migrations must precede native opt-in; live
serving migration with a configured native controller is deliberately refused.

Benchmark commands, fixed before confirmation:

- Matched sequential ABBA: baseline/candidate/candidate/baseline, identical common
  harness, GOMAXPROCS=1/GOGC=100. Baseline is commit `1e12332` (authentic Phase III
  plus current main/PR63 and this protocol, before any Gate G implementation).
- `BenchmarkGateGThought`: 100 samples/cell/run, 1k/4k/40k, nine initial units or
  all units, with/without WAL. The numeric Process used to seed the state is a
  genuine command outside the timed thought interval. Its resource/GC activity
  is included in descriptive whole-cell GC statistics, not thought B/op.
- `BenchmarkGateGNativeStages`: 20 factual pairs/cell/run, same sizes and sparse/
  dense states, durable WAL. Report begin, operation and observed-learning latency
  separately. Twenty-sample p95/p99 are descriptive order statistics, not tail
  probability estimates. Do not increase samples selectively to hide a failure.
- `BenchmarkNativeActionPersistence`: three samples/cell/run on both revisions;
  checkpoint, restore-pending and replay-pair; retain the original fixture.
- `BenchmarkGateGJournalAppend`: 100 samples/run on both revisions.
- Candidate-only: `BenchmarkGateGAdmission` 100,000 observations; actual mixed
  sensor/native handoff 100 samples/size; `BenchmarkGateGThoughtNativeIdle` 100
  samples/cell, compared against the same declared production regression budget.
  The authentic baseline refuses native-idle production commands, so no invented
  baseline mixed-workload measurement is permitted.

All original numeric budgets and 100% exact-state criteria remain unchanged.

### Pre-benchmark safety and validation amendment

Additional adversarial probes showed that direct numeric experience, consequence
and spelling payloads could name internal native policy units even at idle. All
three failed before the repair (`development-learning-alias-before.txt`). Central
pre-append reference admission now rejects those aliases, including inline
observation experiences. Positive world-address learning remains tested. This
extends the declared ownership consistency check without changing a learned law.

The first full local race command reached its 20-minute process limit during the
existing sealed composition curriculum, with no race report before timeout. The
failure and stack are retained. A bounded before/after CPU-profile probe uses the
same existing `separated/7` case, no data/seed/iteration change. Nil ownership-map
lookups on an unconfigured production Brain are bypassed by a short-circuit;
parameter protection with native enabled is retained. The original frozen-law
and complete-state tests must still pass. Resource budgets are unchanged.

The positive typed-learning control exposed another v1 ownership path:
`ApplyObservedAssociationFeedback` treated a native policy edge as a competing
world successor. Its complete parameter before/after failure is retained.
That feedback now honors the same address ownership fence. Ordinary world
experience, spelling and inquiry-consequence updates must remain functional and
leave policy parameters unchanged in all three historical native versions.

A development resource probe recorded native-idle p95 44.65ms versus baseline
34.31ms on the old context-only 40k workload (not a confirmatory result). CPU
profiling identified ownership membership hashes in the whole-graph activity and
memory scans. A transient bounds-plus-exact-set index avoids world-edge hashes;
the exact set still handles noncontiguous historical addresses. No weights or
neural state are cached; all original resource budgets remain fixed.

The initial 'sparse' timing fixture contained only ContextBoosts, which current
production dynamics treats as recurrent thought. Activity grew from nine to809
units rather than reseeding observed input. Confirmation now supplies real
StimulusNodeIDs and an explicit independent observation episode before timing
thought, implementing the original sparse/dense fixture intent. The important
old context-only failure is retained as `BenchmarkGateGContextContinuity` and
`BenchmarkGateGContextContinuityNativeIdle`: the 40k/non-dense/WAL cell is included
in ABBA and candidate-only regression checks under the SAME numeric budgets.
No failed workload is removed, and continuous-state functional tests are retained.

Full-race execution timeout amendment: retain every test and assertion, increase
only the full-suite operational deadline to40 minutes on this host/CI. The single
unchanged separated/7 curriculum case took52.57s before the nil fast path and
53.19s after; it did NOT demonstrate a speed improvement and is not reported as
one. CPU samples were dominated by race instrumentation. The original20-minute
failure remains a recorded negative result. This does not change any runtime
latency/allocation budget, seed, neural test, durability rule or gate criterion.
The workflow deadline is updated explicitly so the full suite can complete.

### Benchmark process deadline amendment

The first ABBA invocation stopped producing output in the baseline40k dense
thought cell before any whole-command result was recorded. Its partial output
is retained and is not assigned a passing status. Go1.23.2 testing.M explicitly
stops the test alarm before runBenchmarks; `go test -timeout=20m` therefore does
not bound benchmark execution. The runner now records the active process group
before starting and enforces an external20-minute command deadline, preserving
an explicit failed status on timeout. A fresh complete ABBA must be retained;
sample counts, inputs, harness, budgets and comparison rules are unchanged.

The first externally bounded attempt completed its baseline thought matrix but
failed in the context-continuity command: the unanchored first regex segment also
selected `BenchmarkGateGContextContinuityNativeIdle`, which baseline correctly
rejects. The failure is retained. Each slash-separated selection segment is now
anchored exactly, selecting the originally registered workload. A fresh ABBA is
required; no failed runtime measurement is hidden and no budget is altered.

### Backing-filesystem disclosure and supplementary append check

Filesystem inspection during the resource audit identified `/tmp` as tmpfs and
the checkouts as overlayfs. The common Go TempDir fixtures therefore measure
software/process-crash behavior with every Sync call retained, not physical
storage power-loss durability or Android fsync latency. This is disclosed for
all primary measurements. No primary workload or acceptance threshold changes.

After the main matrix, additionally run100 journal appends per revision in ABBA
with TMPDIR explicitly set to a fresh directory on checkout overlayfs, same
Go/GOMAXPROCS/GOGC/common harness. Before that supplementary measurement, fix its
budget to candidate mean run p95 <=baseline*1.20+0.10ms and B/op <=baseline*1.10
+4096. Retain actual filesystem labels, raw results and any failure. This check
does not establish device performance, power-loss durability or universal disk
latency. Do not run it concurrently with the primary timing matrix.

### Baseline diagnostic assumption failure — retained

The complete primary timing matrix executed, but its final expected-refusal
probe exited0 rather than1: baseline `BenchmarkGateGMixedHandoff` uses the sensor
entry point, which bypassed the old blanket guard. This is an audit finding, not
a fabricated baseline latency or a passing negative test. The original runs.json
and runner exit remain unchanged. All90 resource comparisons are retained; this
unexpected diagnostic exit also prevents calling that invocation protocol-clean.

A separate corrected diagnostic uses the original Process-bearing native-idle
thought fixture, exactly anchored at1k/non-dense/WAL, one sample. It must reject
with the actual blanket-guard error. Future runner metadata identifies this as
`baseline-global-guard-refusal`; the summary reader preserves the older diagnostic
contract for the retained original matrix. Do not rerun/discard the resource
matrix merely to hide this failed assumption, and do not reinterpret the sensor
bypass as verified safe coexistence or a matched mixed-runtime speed baseline.

### Post-confirmation projection ownership failure and repair

After freezing and completing the resource matrix, an additional adversarial
numeric input exactly equal to a native bias prototype recruited that private
unit through the generic representation matcher. Both legacy and current
dynamics failed. The original negative output is retained. The matcher must
exclude canonical native role addresses before similarity selection, using the
same ownership fence as explicit numeric IDs. World input still learns/reuses
world populations, and the new case must round-trip checkpoint+suffix exactly.

This is a real additional implementation change after measured codeaf3b946.
Do not claim the recorded performance matrix benchmarks the repaired descendant
or passes G8. Its failed budgets remain retained; the descendant needs technical
regression/CI verification and a future fresh resource confirmation after any
performance repair. No primary criterion or failed fixture is removed.

The follow-up compatibility probe supplied a correct full checkpoint checksum
around an old world population containing a native bias. The old validator and
reader accepted it and mutated a fresh recovery target. Retain that negative
result. Native validation must reject this role/provenance alias before restore,
leave the checkpoint unchanged, and preserve valid native1/2/3 fixtures. This is
validation of the existing single-substrate ownership contract, not a schema
migration or permission to alter user knowledge.
