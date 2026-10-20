# Bounded Brain inspection and streamed persistence (H18)

H18 separates a monitor's bounded view from complete canonical persistence.
The baseline is [PR #40](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/40),
commit `0b893e905742e27e3c9c4ad2a79c5c4fb4747cc0`.

## Read contract

`GET /v1/brain/inspection?section=neural_units&limit=256&order=activity`
returns one `brain.inspection.page` (schema 1). The same auth, CORS, rate limit and
managed-runtime read lease as the existing Bridge protect this endpoint. Responses
use `Cache-Control: no-store`.

- `data.items`: complete records from one selected canonical collection.
- `data.total`: total records in that collection, rather than the loaded count.
- `data.counts`: whole-network unit, synapse, population, pattern and trace counts.
- `limit`: at most 512 primary records; default 128.
- `byte_limit`: the entire encoded HTTP body, including digest and newline,
  is at most 524,288 bytes. A page may contain fewer than `limit` records.
- `next_cursor`: continuation under the same revision/inspection epoch. It is
  empty at the end; no partial page is presented as a complete graph.
- `hash_scope: inspection_page` and `page_hash`: SHA-256 of the exact compact page
  JSON with the `page_hash` field omitted. The wire field is last; the browser
  verifies original UTF-8 bytes, not a differently escaped JS reserialization.
  This digest is neither a global Brain hash nor an authentication mechanism.
- `inspection_epoch`: ephemeral runtime/read lifetime, separate from the one
  persistent Brain identity. It is never neural state or checkpoint memory.

`section` covers all 24 canonical top-level fields: neural units, synapses,
populations, temporal patterns, sensory/experience/developmental traces,
episodes/bootstrap experiences/provenance, all state maps, memory clock,
learning policy, inquiry state, canonical schema and canonical Brain identity.
State maps return `{key,value}` records; singleton state objects return one row.
A test compares pagination of every section against complete canonical data.

Default order is ascending internal ID (or array index/map key for those
collections). Neural units also support `order=activity`: descending activation,
then ascending ID. Synapses have a deterministic source/target/inhibition/kind/
map-bucket/list-index order so parallel edges remain distinct. This inspection
order need not reproduce the historical full serializer's tie permutation;
complete records and multiplicity are preserved, and canonical bytes are
unchanged. Cursor tokens are read positions, not signatures or authorization.

A revision change, runtime replacement, or successful standalone load/restore
invalidates old cursors with HTTP 409. The client starts a new view rather than
combining pages from different revisions. Cross-section/order or malformed
cursors return 400. Paging does not pin or freeze the Brain. Traversing a whole
unchanging revision therefore requires a quiet owner or an isolated recovered
checkpoint; continuous mutation can require retry. Direct legacy mutable aliases
that bypass the runtime owner are outside this revision-consistency contract.

A unit page additionally returns at most 1,024 connections whose two endpoints
are in that unit window, with `related_total` reporting the actual within-window
count. A shared byte budget can reduce the returned edge count further. These
are not all outbound or all global connections. Global synapse inspection is a
separate collection. State previews contain at most 16 short-key entries per
map, with a shared 8 KiB preview budget; full maps are paginated separately.

Oversized individual records return HTTP 413. A conservative preflight avoids
first building an enormous vector/history/string JSON buffer. No neuron,
representation or historical record is shortened in the persistent Brain to fit
the monitor. Complete saves/checkpoints remain the export/recovery boundary for
records too large for this inspection interface.

## Memory and consistency

Selection scans the registry/connection index and retains only a bounded heap of
`limit+1` keys, then encodes selected records under the Brain read locks. It never
calls `CanonicalJSON`, `Registry.Nodes` or `Patterns.All` on the page path.
Returned JSON, preview maps and connections are detached from the live substrate.
Runtime ownership covers selection, counts, revision, read epoch and page digest.

Memory for the normal page path depends on page/record budgets rather than the
whole encoded graph. Time still depends on total units/connections: the current
implementation scans them for selection and counts. Large reads can still hold
the owner lock for seconds. H19's indexes/counters/global-scan work remains.

## Browser behavior

The browser first requests bounded inspection. A real HTTP 404 (including the
older Bridge's plain-text 404) selects the legacy handshake/full-snapshot path.
An invalid page/hash does not trigger a full snapshot fallback. On a new Bridge,
connection and refresh use pages, avoiding full graph materialization and a
full-state handshake digest. The WebSocket explicitly requests
`telemetry=revisions&inspection=bounded`; its schema-2 handshake reports the
inspection epoch and page scope, without a canonical-state hash. Default and
ordinary revision/canonical WebSockets retain their existing contracts.

The graph shows a live activity window, loaded-vs-total units and within-window
edges, with next/refresh controls. Inspector collections have independent page
controls. State panels are labeled previews; the hash label switches to Page
hash. Graph refreshes retain H17's coalescing interval and bounded read cadence.
Cursors may expire between user clicks while cognition continues; this is shown
as a changed-view error, not silently mixed data. New runtime epochs reset the
cursor after recovery. Superseded read results/old sockets are discarded and
manual disconnect does not schedule an unwanted reconnect.

The client checks body limits while reading a streaming Fetch response and
cancels oversized transfers, then checks the wire digest and row limits. The
3D layout receives only the loaded unit window. A legacy Bridge fallback still
has its original full-object memory/latency limits.

## Complete persistence and identical hash laws

`WriteCanonicalJSON(io.Writer)` emits the exact existing pretty canonical bytes;
`CanonicalHash()` feeds those bytes directly into SHA-256. Canonical scalar/
escaping/map rules still come from Go's standard JSON encoder. Unit/edge entries
are emitted separately; a bounded reusable indentation buffer avoids retaining
the whole pretty document. No representation vectors or edge dynamic states are
copied into a second full neural graph.

The canonical serializer retains an ordered node-pointer index and a compact
source/edge-pointer index. Global edge sorting deliberately reproduces the old
serializer's comparison and tie behavior, rather than changing saved hashes.
Auxiliary indexes are still O(N+E), approximately 8 bytes per node pointer and
16 bytes per edge reference on this amd64 run, plus pattern/metadata work and
per-record encoding. Large single nested records/maps can still need their own
encoding buffers. This is not a claim of constant memory for arbitrary data.

`SaveBrain` writes canonical bytes directly to a private temp file, flushes,
syncs and renames, preserving the existing complete file format and bytes.
Canonical telemetry and ordinary full handshakes use the streaming hash. The
legacy `CanonicalJSON`/full `MonitorSnapshot` APIs still return the entire
materialized result and retain their batch encoder and known memory limits.
Other legacy full-inspection/migration paths are also not bounded pages.

Checkpoint **schema 12 and its logical v5/v6/v7 checksum laws remain unchanged**.
New checkpoint files are compact JSON. The Brain is streamed exactly once to
both the file's Brain field and the checksum's compact Brain field. A fixed-width
checksum slot in the private temporary file is patched only after the complete
logical checksum succeeds; flush/write-at/sync/close precede the atomic rename.
A failed encoding or I/O operation cannot replace the previous checkpoint.

All canonical Brain fields, recurrent activation/cognitive/predecessor state,
action bindings, evidence ledger, dedup index, surface annotations, inquiry
state and committed journal anchor remain present/authenticated. Restore still
accepts historical pretty checkpoints. The negative physical-clock test now
mutates parsed fields rather than depending on spaces in JSON, and asserts that
tampering actually happened before checking rejection without mutation.

There is no new canonical/checkpoint/journal/dynamics version, neural algorithm,
second cognitive memory, semantic rule engine, LLM or alternate thinking path.
Thinking, plasticity, evidence, uncertainty, timing and history retain their laws.

## Scale measurements

Linux/amd64, Go 1.22.12, AMD EPYC 9V74, 9 visible processors, an 8-CPU quota and
8 GiB cgroup memory limit. Each measurement is an isolated test process using
identical test-only fixture code on baseline/candidate. It builds fixed 16D
numeric units directly in the test registry, avoiding quadratic recruitment or
large fixture-load buffers. Setup is timed separately and excluded from the
operation. The zero-edge fixtures are untrained synthetic storage workloads,
not a learned or richly active brain.

A GC precedes measurement; `/proc/self/clear_refs` resets VmHWM to current RSS
(all recorded probes report success). VmRSS/VmHWM come from `/proc/self/status`;
allocation is the difference in Go TotalAlloc. Peak RSS below includes the
resident Brain and operation; allocation is cumulative, not live/peak memory.
A 6 GiB soft Go memory limit applies equally to both implementations, not a hard
virtual-address limit. These are single local runs, not p50/p95, confidence
intervals or CI performance gates. Raw measurements and the opt-in child harness
are retained in [testdata/inspection_performance](testdata/inspection_performance).

One million units, zero seed synapses; decimal GB/MB:

| Operation | Seconds before → after | Peak RSS GB before → after | Allocated GB before → after | File bytes MB before → after |
| --- | ---: | ---: | ---: | ---: |
| hash | 7.962 → 6.519 | 2.921 → 0.640 | 2.892 → 0.726 | 0.00 → 0.00 |
| save | 7.877 → 6.571 | 2.912 → 0.641 | 2.892 → 0.726 | 727.05 → 727.05 |
| checkpoint | 18.874 → 3.805 | 5.156 → 0.641 | 6.322 → 0.720 | 783.05 → 494.05 |

The saved canonical file stays byte-compatible (727,051,643 bytes). Its hash is
`sha256:9d80dcee53c2b95f6c8203c4b31be06e3a10b54d3e114bef9cd22ad6f2bc3818`
on both implementations. The checkpoint becomes smaller because its envelope/
Brain are compact; the logical checksum and complete recovery data are retained.

The one-million **legacy native full MonitorSnapshot** baseline takes 13.180 s,
allocates 3.788 GB and peaks at 2.941 GB RSS. Its stored raw JSON entries total
721.05 MB (not the complete HTTP wire size). That all-object path is retained;
streaming does not make a return value containing the entire graph bounded.

The new **256-record page** on that same one-million fixture takes
0.262 s, returns 126,875 wire bytes and peaks at
0.330 GB total RSS; the measured increase above its
already resident Brain is 1.79 MB.
This is a bounded view, so it must not be presented as a faster complete snapshot.

On **10 million units / zero synapses**, the page succeeds in
3.800 s with 126,877 wire bytes and about
1.01 MB cumulative operation allocation. The Brain
is already resident at 3.142 GB RSS; peak RSS is
3.145 GB, an increase of
3.74 MB. Fixture setup takes
8.478 s. No full 10-million JSON snapshot, checkpoint or
restore completion is claimed, and the capped legacy child refuses a full-object
probe above one million units in this environment.

A complete **one-million checkpoint → restore → canonical-hash comparison**
succeeds in 46.309 s total, including writing and both comparison
hashes. The entire canonical hash equals the baseline above. Peak RSS is
3.709 GB and cumulative allocation is
6.242 GB; the probe holds both original/recovered
Brains solely for verification. This exposes the remaining cost of the current
full-materializing restore rather than calling it bounded streaming recovery.

A second fixture includes 100,000 units and 400,000 nonzero ring synapses;
these seed connections are not driven by an active learning scenario:

| Operation | Seconds before → after | Peak RSS GB before → after | File MB before → after |
| --- | ---: | ---: | ---: |
| save | 3.693 → 2.579 | 1.208 → 0.295 | 290.92 → 290.92 |
| checkpoint | 7.466 → 1.500 | 1.997 → 0.300 | 312.52 → 200.02 |

Its 256-unit page, including bounded within-window edges, is
504,298 bytes, completes in 0.093 s and peaks at
0.172 GB RSS (resident Brain before read:
0.167 GB). This checks actual synapse-bearing output
against the response limit rather than extrapolating from zero edges.

## Reproduction and remaining bounds

The scale child is skipped in ordinary tests/CI. Copy only
`inspection_scale_fixture_test.go` and `inspection_scale_external_test.go` into
the baseline knowledge test directory to measure PR #40, leaving production
code unchanged. Page and checkpoint-restore probes are candidate-only modes.
Compile one test binary per worktree and run fresh processes:

```sh
go test -c ./services/ai/knowledge -o /tmp/h18.test
HORIZON_INSPECTION_SCALE_MODE=snapshot HORIZON_INSPECTION_SCALE_N=1000000 /tmp/h18.test -test.run='^TestInspectionScaleChild$' -test.v
HORIZON_INSPECTION_SCALE_MODE=page HORIZON_INSPECTION_SCALE_N=10000000 /tmp/h18.test -test.run='^TestInspectionScaleChild$' -test.v
HORIZON_INSPECTION_SCALE_MODE=checkpoint HORIZON_INSPECTION_SCALE_N=1000000 /tmp/h18.test -test.run='^TestInspectionScaleChild$' -test.v
HORIZON_INSPECTION_SCALE_MODE=checkpoint_restore HORIZON_INSPECTION_SCALE_N=1000000 /tmp/h18.test -test.run='^TestInspectionScaleChild$' -test.v
HORIZON_INSPECTION_SCALE_MODE=save HORIZON_INSPECTION_SCALE_N=100000 HORIZON_INSPECTION_SCALE_FANOUT=4 /tmp/h18.test -test.run='^TestInspectionScaleChild$' -test.v
```

The writer does not solve large-graph restoration, metadata that itself grows
very large, complete journal-prefix reads, WebSocket event-history backfill or
full-graph materialization through legacy APIs. Per-input full-save scheduling
still incurs complete serialization time. H19 global scans and H20 event-log/
backpressure budgets remain separate work. There is no release/readiness or
10-million-neuron learning-capability claim.

Correctness checks cover frozen old encoder bytes (including parallel-edge ties,
escaping/float values and compact embedding), legacy checksums, complete section
pagination, detached results, activity order, cursor fences after mutation/
restore/runtime handover, oversized records, concurrent owner commands, failed
save/checkpoint atomicity, explicit WebSocket hash scope, and actual Go wire
bytes verified by the production browser helper. Existing complete checkpoint,
suffix/genesis replay and next-thought fixtures continue to run unchanged.

Local validation passed with Go 1.22.12: `go test ./...` in the root Go
module, `go vet ./...`, `go build ./...`, and race checks for knowledge,
learning, runtime, bridge, core and horizon-cli. `make bridge-smoke` passed,
including the authenticated live inspection/production-browser path and existing
visual/browser sensor and persistence/replay checks. Monitor TypeScript checking,
all 37 Node tests and the production build passed (Node 24.19.0, pnpm 11.25.0).
Nested Go modules outside the root module are not covered by `go test ./...`.
The scale figures above are explicit local probes, not regular CI workloads.
