# Revision telemetry and snapshot boundaries (H17)

H17 removes full neural-graph serialization from each runtime event when no
subscriber requests it. It changes monitoring work, not neural dynamics or
canonical persistence. The baseline is [PR #39](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/39)
(`cebb9763548ff9b6e598e01f130ffd201ab56b70`).

## Integrity contract and compatibility

| Reader | Contract | Whole-Brain serialization on every published event |
| --- | --- | --- |
| No telemetry subscribers | No event envelope/map is built | No |
| `Telemetry().SubscribeRevisions()` | Live schema 2: committed revision, original event hash, cognitive delta/prediction error when available, plasticity values | No |
| `Telemetry().Subscribe()` | Legacy event, including exact canonical neural-Brain hash | Yes |
| Mixed subscriptions | One build; legacy subscribers receive the full hash, revision subscribers receive schema 2 without that field | Yes, required by the legacy subscription |
| Monitor snapshot | Existing full graph view and its canonical neural-Brain hash at one locked revision | Explicit read boundary |
| Handshake / health | Same exact canonical hash and revision, without constructing and decoding an unused monitor graph | Explicit read boundary |
| Checkpoint / `SaveBrain` | Existing complete canonical persistence; checkpoint also binds recurrent/runtime state | Existing persistence boundary |

WebSocket clients opt in with `/v1/brain/stream?telemetry=revisions`. An omitted
parameter or `telemetry=canonical` keeps the legacy full-hash stream. Unsupported
values return HTTP 400 before upgrading. The HTTP handshake advertises the
`revision_stream` capability; the WebSocket handshake echoes the selected
`telemetry_mode`. The browser selects revisions only when that capability is
present, retaining compatibility with older Bridges.

Schema 2 deliberately omits `canonical_state_hash`; it does **not** claim to
attest the entire neural state after every event. Clients requiring that proof
must keep the canonical stream. This is an explicit opt-in contract, not an
incremental or Merkle hash masquerading as the old SHA-256 digest.

`event_hash` preserves the existing encoding of
`{type, sequence, event?, observation?, outcome?}` exactly. It is not a digest of
every telemetry field and is not an authentication mechanism. Full-state hashes
remain SHA-256 of the unchanged pretty-printed `CanonicalJSON()` bytes, including
all previously covered neural fields. The checkpoint checksum continues to bind
the complete persisted logical state, beyond that neural JSON alone.

No canonical/checkpoint/journal/dynamics schema changes or neural-state cache are
introduced. In particular, a cache keyed only by `seq` would be unsound for
legacy standalone mutation aliases. Historical immutable replay/checkpoint
fixtures retain their old bytes, checksums and continuation laws.

## Publication and lifetime

All existing publication sites use one demand-driven builder after the complete
transition and any durable commit. No listeners means no event hashing, graph
encoding or plasticity-map copy. Revision-only listeners require no graph read.
One or more canonical listeners trigger one full encoding per publication.

Recipient selection happens under the hub read lock; serialization happens
outside that lock while the runtime owner lock still protects the completed
revision. Unsubscribe/Close may proceed during a build. Delivery checks that
selected subscribers are still live and sends/closes under the same hub lock.
Late subscribers do not receive an unhashed event built for an earlier recipient
set; their handshake snapshot establishes the new boundary. Overflow or encoding
failure closes selected subscriptions so clients resynchronize, rather than
accepting the hash of a failed/partial encoding. Runtime retirement and uncertain
journal commits still close both kinds of subscription.

Legacy raw `ThinkAt` has no telemetry publication, as before. This change does
not invent new events or revisions for operations that did not publish them.

## Browser full-graph reads

The graph is still rendered from actual runtime snapshots. Revision events are
notifications/cognitive deltas, not enough to reconstruct every changed neuron,
synapse or historical field. During ordinary event flow, dirty revisions share
one pending refresh. Full-graph reads start at most once per 1,000 ms and never
overlap in the scheduler. A burst does not create one queued HTTP request per
event. When events stop, the last dirty revision is read and polling stops.
Initial connection and gap resynchronization remain explicit read boundaries.
The one-second cadence is a minimum start interval, not a latency guarantee:
a large snapshot or slow transport may take longer.

The event cursor can advance while a snapshot is in flight. Applying that
snapshot updates the graph at its own labeled **Snapshot revision / Snapshot
hash**, without rewinding event continuity. Older graph snapshots are ignored.
Legacy same-revision hash mismatches still fail. Failed reads do not poison the
refresh queue; reset cancels pending work and discards old in-flight results.
An explicitly verified new connection resets the store's revision epoch, so a
new server/recovery boundary can start below the previous connection's cursor.

## Measurement setup

Local Linux/amd64, Go 1.22.12; CPU reports AMD EPYC 9V74, 9 visible processors,
8-CPU cgroup quota, 8 GiB memory limit. Baseline and candidate ran sequentially
in isolated worktrees. The exact same benchmark source runs on PR #39 via an
optional interface: it uses the old canonical subscription there and the new
revision subscription here. `HORIZON_BENCH_STREAM=canonical` explicitly selects
legacy hashing on the candidate.

`BenchmarkTelemetryTransitions` has 24 cases: 1,000 / 4,000 / 40,000 seed units;
0 / 4 outgoing synapses per seed unit; no subscriber / one drained subscriber;
no journal / synchronous intent+commit journal. Seed units have deterministic
16D numerical representations, a fixed ring connectivity, and no populations or
experience history. Fixture construction/loading, three warm-up cycles and
subscription/journal setup are outside event timing and allocations; `setup_s`
reports the measured subrun's setup cost separately.

Each measured round processes one scalar physical motion field and then one
internal thought cycle. The sensor normally recruits and reuses four scoped
units, giving final totals 1,004 / 4,004 / 40,004. The seed network's ring is not
connected to that new sensor population: this controls total graph/synapse work
with sparse observed activity; it is **not** a richly active recurrent workload.
The seed edge count below excludes any small learned sensor connections.

Each case has 20 measured rounds, with per-event p50 and nearest-rank p95.
`B/op` covers the **pair** of transitions, not one event, and is cumulative
allocation, not peak/resident memory. These are descriptive local samples, not
statistical confidence intervals or CI performance gates. They exclude HTTP,
rendering, full-graph refreshes and per-request `SaveBrain` materialization.
The source and both complete 24-case matrices are retained in
[testdata/telemetry_performance](testdata/telemetry_performance).

One drained subscriber (legacy baseline → revision candidate); times are
**p50 / p95 milliseconds**, allocation is decimal MB per sensor+thought pair:

| Seed units | Seed synapses | Journal | Sensor before → after | Thought before → after | Allocation MB/pair before → after |
| ---: | ---: | :---: | ---: | ---: | ---: |
| 1,000 | 0 | off | 6.64 / 7.63 → 0.26 / 0.34 | 6.68 / 7.59 → 0.33 / 0.50 | 4.46 → 0.14 |
| 1,000 | 0 | on | 7.30 / 8.15 → 1.35 / 5.65 | 6.92 / 8.33 → 0.73 / 3.22 | 4.36 → 0.14 |
| 1,000 | 4,000 | off | 26.37 / 30.62 → 0.68 / 1.49 | 26.82 / 31.12 → 0.75 / 1.06 | 26.70 → 0.32 |
| 1,000 | 4,000 | on | 29.05 / 35.25 → 1.49 / 1.70 | 28.46 / 34.02 → 1.01 / 1.39 | 27.34 → 0.32 |
| 4,000 | 0 | off | 25.70 / 34.14 → 1.05 / 1.31 | 28.03 / 37.41 → 1.62 / 2.80 | 21.07 → 0.47 |
| 4,000 | 0 | on | 26.79 / 31.72 → 1.83 / 2.08 | 26.59 / 30.65 → 1.67 / 2.41 | 21.08 → 0.47 |
| 4,000 | 16,000 | off | 100.00 / 108.60 → 2.81 / 3.46 | 98.74 / 112.50 → 3.21 / 4.02 | 120.14 → 1.18 |
| 4,000 | 16,000 | on | 108.10 / 140.30 → 3.99 / 6.95 | 109.00 / 128.20 → 3.73 / 5.29 | 115.95 → 1.18 |
| 40,000 | 0 | off | 274.60 / 329.90 → 14.49 / 18.84 | 281.80 / 330.20 → 22.39 / 27.82 | 276.54 → 5.10 |
| 40,000 | 0 | on | 271.60 / 287.90 → 14.85 / 17.55 | 281.00 / 297.10 → 20.76 / 25.69 | 279.90 → 5.10 |
| 40,000 | 160,000 | off | 1142.00 / 1339.00 → 40.15 / 47.99 | 1164.00 / 1325.00 → 47.30 / 62.62 | 1399.09 → 12.14 |
| 40,000 | 160,000 | on | 1073.00 / 1217.00 → 40.55 / 51.60 | 1182.00 / 1313.00 → 44.60 / 47.70 | 1412.51 → 12.15 |

Without subscribers, the 40,000-unit / zero-seed-synapse / no-journal case falls
from sensor 265.4 / 310.0 ms to 13.49 / 19.28 ms, and thought 277.9 / 339.9 ms to
20.03 / 22.95 ms. Pair allocation falls from 256.40 MB to 5.10 MB. The complete
matrix retains every no-subscriber and journal case, rather than assuming those
paths have the same cost as subscribed telemetry.

A separate canonical compatibility run on 4,000 units / 16,000 seed synapses /
no journal retains the old cost: sensor 106.0 / 127.8 ms and thought
108.6 / 127.5 ms, with 113.42 MB allocation per pair. Opting in to full hashes,
or keeping one legacy listener in a mixed hub, intentionally forfeits the
revision-only serialization saving.

## Bounded sensor–action–outcome cycle

`BenchmarkTelemetrySensorimotor` independently measures a scripted cycle on
4,000 seed units / 16,000 seed synapses: motion sample, internal thought,
action binding to that sample's population, then an observed encoder outcome.
Each cycle exercises four committed publications. Each action has its own
request ID; outcome reports share one source/episode and do not invent
independent evidence. There are three untimed warm-up cycles and 20 timed cycles.
This validates runtime command cost, not real actuator control or task success.

Times are complete-cycle **p50 / p95 milliseconds**; allocation is decimal MB
per four-command cycle. The subscriber is legacy before and revisions after.

| Subscriber | Journal | Cycle before → after | Allocation MB/cycle before → after |
| :---: | :---: | ---: | ---: |
| false | false | 416.10 / 481.10 → 5.37 / 8.01 | 236.18 → 1.16 |
| false | true | 452.20 / 577.50 → 6.96 / 8.22 | 241.22 → 1.16 |
| true | false | 428.90 / 480.80 → 5.26 / 7.07 | 236.18 → 1.16 |
| true | true | 409.40 / 454.90 → 7.49 / 10.10 | 235.34 → 1.17 |

The raw four-case results for each implementation are retained with the matrix.

## CPU, allocation and owner-lock profiles

A separate 100-round profile uses the 4,000-unit / 16,000-seed-synapse,
subscriber-on, journal-off case, plus one `LastSequence()` reader requesting
access every millisecond. CPU, allocation, mutex and block profiles are collected
for each implementation. Profile files include benchmark calibration, fixture
loading and warm-up as well as transitions; setup is especially visible after
the optimization. These samples are separate from the unprofiled latency table.

On the baseline, `telemetryEventLocked` accounts for 77.25% cumulative CPU,
`CanonicalJSON` 72.53%, and `MarshalIndent` 68.76%. Those overlapping cumulative
percentages must not be added. The candidate profile has no canonical encoding
samples on the event path; activity plasticity is now the largest transition
cost (40.69% cumulative CPU), with fixture loading also prominent.

Total sampled allocation over the entire profile process is 12,307.39 MiB
before and 257.29 MiB after, including fixture setup. Canonical construction
accounts for 98.00% cumulative allocation before and is absent from candidate
event stacks. This is total allocated memory, **not** a high-water RSS result.

The owner mutex attributes 25.31 seconds of cumulative contended delay before
and 0.618 seconds after. The separate blocking profile shows the polling reader
waiting 25.31 seconds vs 0.619 seconds in `LastSequence`. This is a synthetic
single-reader contention measurement over the same event count; it is not a
per-event lock latency or a guarantee for simultaneous snapshot/checkpoint
readers. Full textual CPU/allocation/mutex/block summaries are retained alongside
the raw benchmark outputs.

## Full snapshot cost remains visible

`BenchmarkTelemetryReadBoundary`, 4,000 seed units / 16,000 seed synapses,
10 reads each, reports:

| Boundary | Baseline mean | Candidate mean | Baseline allocation | Candidate allocation |
| --- | ---: | ---: | ---: | ---: |
| Full monitor snapshot | 183.69 ms | 195.23 ms | 74.12 MB | 70.77 MB |
| Handshake | 180.49 ms | 108.57 ms | 75.80 MB | 56.39 MB |

The snapshot algorithm/format is unchanged; the differing single-run snapshot
numbers do not establish a snapshot speedup. Handshake avoids the unused graph
decode but still serializes and hashes the entire canonical Brain. Health uses
that same integrity boundary. No million-neuron snapshot scalability claim is
made here. H18's memory/streaming/pagination work and H19's global scans remain.
Bridge configurations that save the full Brain after every input also still pay
that serialization cost, independently of telemetry mode.

## Reproduction

Run with Go 1.22.x from the repository root. On PR #39, copy only the benchmark
source from this change into the corresponding runtime test directory; do not
change baseline production code.

```sh
go test ./services/ai/runtime -run '^$' -bench '^BenchmarkTelemetryTransitions$' -benchtime=20x -count=1
go test ./services/ai/runtime -run '^$' -bench '^BenchmarkTelemetryReadBoundary$' -benchtime=10x -count=1
go test ./services/ai/runtime -run '^$' -bench '^BenchmarkTelemetrySensorimotor$' -benchtime=20x -count=1
HORIZON_BENCH_STREAM=canonical go test ./services/ai/runtime -run '^$' -bench '^BenchmarkTelemetryTransitions$/^nodes=4000$/^fanout=4$/^monitor=true$/^journal=false$' -benchtime=20x -count=1
HORIZON_BENCH_READER=1 go test ./services/ai/runtime -run '^$' -bench '^BenchmarkTelemetryTransitions$/^nodes=4000$/^fanout=4$/^monitor=true$/^journal=false$' -benchtime=100x -count=1 -cpuprofile=/tmp/h17.cpu -memprofile=/tmp/h17.alloc -mutexprofile=/tmp/h17.mutex -blockprofile=/tmp/h17.block -o /tmp/h17.test
go tool pprof -top -cum /tmp/h17.test /tmp/h17.cpu
go tool pprof -top -alloc_space /tmp/h17.test /tmp/h17.alloc
go tool pprof -top /tmp/h17.test /tmp/h17.mutex
go tool pprof -top /tmp/h17.test /tmp/h17.block
```

## Correctness and architecture

The tests cover off / revisions / canonical / mixed subscriptions, with and
without a journal. After each raw input, cognitive input, written envelope,
physical sensor, action binding, outcome and internal-thought command, complete
continuation state is byte-equal across modes. Legacy telemetry digests match
independently hashed canonical bytes. Checkpoint restoration, committed journal
replay and the next internal thought preserve the same complete logical state.

Hub tests cover empty/closed demand, late joins during a build, cancellation,
encoding failure, overflow and concurrent closure with both subscription modes.
Actual WebSocket tests verify default/explicit canonical and negotiated revision
frames. Browser tests cover coalescing, non-overlap, final dirty refresh, failure
recovery, old-result disposal, capability negotiation, and revision/hash checks.

One persistent Brain, existing learning laws, recurrent internal thought,
physical timing, provenance, uncertainty and history remain unchanged. Temporary
telemetry/snapshot data are inspection views, never a second cognitive memory.
No semantic rule engine, LLM, answer store or alternate thinking path is added.

Local validation passed: `go test ./...` and `go vet ./...` for the root Go
module, `go build ./...`, race tests for knowledge/learning/runtime/Bridge/core/CLI,
and `make bridge-smoke` (including production browser sensor handlers against a
live authenticated Bridge). Monitor `pnpm check`, all 32 tests, and `pnpm build`
passed on Node 24.19.0; upstream monitor CI uses Node 22. Separate nested Go
modules are outside these root-module checks. The draft PR records exact-head CI.

Current bounded monitor reads and complete streamed persistence are described in
[BOUNDED_INSPECTION.md](BOUNDED_INSPECTION.md); earlier measurements above retain
their original baseline scope.
