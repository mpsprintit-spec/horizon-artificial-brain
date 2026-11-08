# Phase II performance — measured gains and an unresolved regression

**Gate E is not passed for production integration.** The native controller and
relational readout allocate less, but the unchanged legacy whole-thought path
measures slower in this environment. The 3× whole-thought target is not met.

## Method and comparability

Baseline: authenticated PR61 tree `58ca1592cf9f60907a334346170d8138d018b920`,
which includes the PR60 performance changes. Candidate: the Phase II isolated
branch, with the same benchmark source for the legacy/controller comparison.
Both binaries use Go 1.23.2, Linux amd64, Intel Xeon Platinum 8573C, an eight-CPU
quota. Historical PR60 measurements used a different host (AMD EPYC); the old
58.830→23.945 ms result cannot be directly combined with this run.

Each latency cell uses 100 measured iterations after setup/warmup. ABBA runs
execute sequentially: baseline, candidate, candidate, baseline. Reported tables
average each run's percentile; these are **not pooled percentiles or confidence
intervals**. Raw per-run p50/p95/p99, ns/op, B/op, allocation counts, GC pause,
GC count, read counts, process CPU time and peak RSS are preserved. Throughput
is `1e9 / ns_per_op`; a controller operation here is a full three-decision episode,
whereas a readout operation is one inference. They are different workloads.

Initial runs use the host default GOMAXPROCS=9. Separate diagnostic and final
buffer comparisons use GOMAXPROCS=1. Process CPU/RSS include fixture construction;
Go allocation/latency metrics exclude it. Cumulative allocated bytes are not RSS.

## Complete controller boundary

40,000 world neurons / 160,000 world synapses, 64 active input neurons,
three controlled decisions, observed delayed feedback, provenance/dedup, and
optional fsync journal. Transition reads remain 524 per episode. This v1 workload
isolates the typed-snapshot improvement; it does not demonstrate v2 learning.

| Journal | p95 before, ms | p95 after, ms | Speed ratio | B/op before→after | Allocations before→after |
|---|---:|---:|---:|---:|---:|
| Off |9.1555|6.3830|1.434×|518865→397512|6495→3615|
| Fsync |17.7000|11.0020|1.609×|532241→411077|6670→3791|

The optimization replaces JSON snapshot roundtrips with an equivalent deep typed
copy. Numerical/replay tests and an isolation test verify that mutable maps and
slices are not shared. The WAL cost remains visible rather than being omitted.

## Legacy whole thought — unresolved

The legacy benchmark executes a real sensor command and one `CognitiveThink(1)`
on the same synthetic network. Nine neurons are active at the reported boundary.
No native controller is initialized.

| Measurement | Baseline thought p95, ms | Candidate, ms |
|---|---:|---:|
| Initial ABBA, journal off |35.210|52.885|
| Initial ABBA, journal fsync |41.085|59.565|
| GOMAXPROCS=1 diagnostic |28.415|32.900|
| Final isolated GOMAXPROCS=1 ABBA |30.485|33.520|

The final check ran after training/full tests finished, with no other deliberate
compute-heavy task. Its p50 is 12.155→12.290 ms; p99 is 39.560→48.355 ms.
Sensor-plus-thought throughput is approximately 34.27→33.11 iterations/s.
Allocated bytes remain 3,834,449→3,834,464 and allocations stay 1109 per iteration.
The candidate records six GC cycles versus seven in baseline; this does not
explain the timing difference. The final p95 regression is about 10%.

PR60 activation/plasticity hot-path source is unchanged. That fact alone is not
proof of equal runtime behavior: code layout, GC scheduling and host variability
remain possible contributors, but have not been isolated causally. Do not dismiss
the regression as noise or claim preservation of historical latency. Main retains
PR60 unchanged; this candidate stays draft. Earlier slower measurements remain in
the archive and machine-readable summary.

## Shared relational inference

Four local graph moments share 80 canonical readout synapses. Structural work
follows active nodes and their local edges. The topology-only lookup still has
sorting costs; it is not constant-time for dense activity.

The final optimization reuses the per-node target buffer and uses typed sorting.
Controlled ABBA compares v2 before/after that optimization on the same 40k/160k
network, GOMAXPROCS=1:

| Active fraction | Active nodes | p95 before→after, ms | B/op before→after | Allocations before→after |
|---|---:|---:|---:|---:|
|0.001|40|0.107450→0.065825|4953→1414|137→16|
|1.0|40000|82.870→74.000|3848778→328773|120016→16|

Sparse speedup is 1.632×; dense speedup is 1.120×. Allocation reductions are much
larger than latency gains. The sparse case reads 80 parameter and 160 structural
synapses; the dense case reads 80 and 160,000. There are no synaptic mutations
during inference. These are readout costs, not complete thought timings.

A size/density sweep also ran at 1k/4k/40k neurons and activity .001/.1/1.
The after-buffer matrix includes possible overlap with a later build/reproduction
process and is exploratory timing evidence only. Its allocations/read counts are
retained; primary speedup claims use the separate controlled ABBA above. No sparse
versus dense strategy switch or parallel mutation is introduced without evidence.

## Memory, GC and profiles

Controller/legacy ABBA process peak RSS is approximately 328–359 thousand KiB;
final isolated legacy runs range 317348–330652 KiB. Buffer ABBA runs range
310236–351868 KiB. These include loading the 40k fixture, not only the readout.
Every benchmark log records GC pauses/counts; resource sidecars record user/system
CPU seconds and voluntary/involuntary context switches.

CPU, allocation and mutex profiles were captured for 1000 sparse readouts.
They include initialization: JSON object decoding accounts for about 47.32% of
sampled cumulative CPU, and JSON validation about 23.55%. These are not isolated
policy CPU percentages. The mutex profile records about 35.23 ms of runtime lock
delay on a single-caller workload; it does not prove absence of contention under
concurrent callers. Race tests address correctness, not throughput under load.

## Pending contrast persistence

New v2 benchmark preserves one observed pending trial, checkpoints/restores it,
then replays the second observed trial from a journal suffix. One measured
iteration per cell is a cost diagnostic, **not a p95/p99 estimate**.

| World neurons / synapses | Checkpoint, ms | Restore pending, ms | Restore + replay pair, ms |
|---|---:|---:|---:|
|1000 / 4000|20.821|309.003|420.223|
|4000 / 16000|70.407|1233.776|1538.976|
|40000 / 160000|915.226|14557.140|13047.712|

The large case cumulatively allocates 159,740,728 bytes to checkpoint,
1,262,299,128 bytes to restore, and 1,422,585,480 bytes to restore/replay.
Process peak RSS is 763784 KiB. Replay appearing faster than restore is not an
improvement claim: each is one independent timed sample. Full canonical JSON
loading/validation remains expensive and is an explicit scalability limitation.

## Reproduction and artifacts

`scripts/summarize_native_phase2_performance.py EVIDENCE_DIRECTORY` reproduces
`performance.json` from `bench-*.txt` and the environment record. Raw profiles,
resource sidecars and binary hashes are in the evidence archive. Empty
`bench-isolated-A*`/`B*` files record an initial unmatched benchmark filter and
contain no measurements; they are not included as timing evidence. Correct final
runs are named `bench-isolated-final-*`.

```sh
go test -c ./services/ai/runtime -o runtime.test
GOMAXPROCS=1 ./runtime.test -test.run='^$' -test.bench='BenchmarkFoundationTransitions/nodes=40000/journal=false$' -test.benchtime=100x
GOMAXPROCS=1 ./runtime.test -test.run='^$' -test.bench='BenchmarkNativeRelationalPolicy' -test.benchtime=100x
./runtime.test -test.run='^$' -test.bench='BenchmarkNativeRelationalPersistence' -test.benchtime=1x
```
