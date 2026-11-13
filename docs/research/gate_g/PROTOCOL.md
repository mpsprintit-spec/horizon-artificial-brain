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
