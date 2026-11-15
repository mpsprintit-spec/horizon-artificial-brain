# Phase III performance: measured causes, repairs and limits

All measurements below use the same exposed Intel Xeon Platinum 8573C cloud
environment, Linux amd64, Go1.23.2, GOMAXPROCS=1 and GOGC=100 unless a diagnostic
explicitly says otherwise. Heavy benchmark processes ran sequentially. These are
synthetic workloads, not Android measurements or a hardware-independent guarantee.
Do not combine this series with the older PR60 measurements from another host.

## Legacy whole-thought regression diagnosis

The original eight-run palindrome was PR60, PR61, PR62, Phase III, Phase III,
PR62, PR61, PR60. Each run used 40k world neurons, 160k synapses, three warmups
and 200 measured sensor-plus-thought iterations. No native controller was enabled.
The following are means of per-run percentiles, not percentiles of pooled samples.

| Source before the curiosity repair | Thought p50 ms | p95 ms | p99 ms | Complete iteration mean ms |
|---|---:|---:|---:|---:|
| PR60 main 32df59b |10.895|28.225|34.535|25.270700|
| PR61 0ef994a |11.020|24.405|31.990|25.499723|
| PR62 ea69b21 |10.960|26.905|33.230|25.460068|
| Phase III e8a59cf |10.785|27.845|34.770|25.234486|

The PR61-to-PR62 tail difference reproduced (about 10.2%), while mean latency
and bytes allocated were almost unchanged. It is not an established 10% increase
in every computation. An initial GOGC=off experiment also disabled collection
during fixture setup, retained setup garbage, and produced confounded long tails;
those logs are retained but are not primary evidence.

A revised ABBA disabled GC **only after ordinary setup and an explicit collection**.
With 100 measured iterations/run and zero measured collections, PR61 p95 was
13.035ms and PR62 13.095ms, a 0.46% difference. This implicates GC-associated
tail behavior. It does not identify a particular compiler layout or scheduling
change as the sole reason PR62 differed. No production GC setting was changed.

The 500-iteration legacy profile identified `applyCuriosityDrive` as 54.09% of
flat allocated bytes, with `Registry.Nodes` another 5.67%. CPU still includes
activity plasticity (38.37% cumulative) and memory dynamics (19.75% cumulative).
Profiles include fixture construction; percentages are not isolated per-step costs.

## Conservative repair integrated through PR65

Curiosity previously copied addresses, built a population of candidates and
stable-sorted it to consume just the winner. The existing registry visitor now
computes the same score/lowest-ID maximum in one pass. The score law, boost,
heartbeat, canonical write boundary and nonfinite stable-sort fallback remain.
A seeded 512-case old-versus-new numerical comparator covers ties, clamping,
negative frequency, graph growth and NaN. No neural law or schema changes.

Primary matched ABBA is main32df59b, candidate, candidate, main32df59b;
200 iterations/run. Candidate tree `cbfee3cd4e31e2dfa4ff004ed6d0cfe44f8a1834`
was published as `e6c86c9d5c2d96288428ad4eec42b114c40b8e64` and merged through
[PR65](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/65)
as `45ecb989b67da690b9bbc1b7a942bb8c55ca6f8e` after CI passed.

| Metric | Main before | Curiosity repair |
|---|---:|---:|
| Thought p50 ms |10.385|9.6475|
| Thought p95 ms |24.720|12.255|
| Thought p99 ms |33.590|14.575|
| Complete sensor+thought mean ms |24.187799|21.018072|
| Complete iterations/s |41.343|47.578|
| B/op |3,855,267.5|114,244.5|
| Allocations/op |1,215.5|1,186|
| Measured GC cycles / 200 iterations |13|0|

p95 ratio **2.017x**; allocated bytes decrease **97.04%**. Individual p95 values
were 24.86/24.58ms before and 13.35/11.16ms after. These two repeated run
statistics do not support a tight confidence interval. Peak RSS, including setup,
was 321016–321048KiB before and 310088–310232KiB after. Process CPU, GC pause,
context-switch and page-fault sidecars accompany `bench-curiosity-*.txt`.
The original raw rows also appear in
`services/ai/runtime/CURIOSITY_SELECTION_PERFORMANCE.md`.

The patch removes a measured common allocation source and improves the tail on
this workload. It neither proves a compiler-layout explanation nor eliminates all
full-graph work. The historical 3x target is not claimed.

## Native command metadata repair

CPU/allocation profiling of dense action pairs showed repeated episode-map cloning
and JSON serialization. Preflight needed metadata but called the full snapshot API.
`NativeControlRuntimeView` now returns a distinct, isolated metadata type. Full
episode digests are unchanged and computed under the Brain read lock only when
needed. Snapshot/checkpoint APIs are unchanged; no second state or knowledge store
is introduced. JSON map sorting and transition copies remain substantial costs.

Matched before/after ABBA, 30 pairs/run, 40k world neurons, WAL off. One pair
means two executed trials and seven commands, not one neural operation.

| Activity | Pair p95 before/after ms | B/pair before/after | Allocations/pair before/after |
|---|---:|---:|---:|
| 40 active (0.001) |1.2805 / 1.1481|191536 / 166937|2624 / 2510|
| 40k active (1.0) |1249.5 / 1196.5|139732585.5 / 119517102|2485217.5 / 2484905|

Sparse p95 varies materially across runs (before 1.509/1.052; after .9122/1.384).
The stronger result is removal of unnecessary copying, not a precise sparse
latency factor. Dense allocated bytes fall about 14.47%; dense execution is still
too expensive for a low-latency controller. `bench-view-*.txt` contains every row.
One frozen delayed model was reproduced with no retuning: behavioral metrics,
canonical Brain, activation, checksum, sequence and processed-events all matched
exactly. See `optimized-reproduction-equality.json`.

## Size, activity, journal and readout matrices

`bench-native-pair-matrix.txt` measures 1k/4k/40k neurons, activity fractions
.001/.1/1, WAL off/on, 30 pairs/cell, **before** the metadata repair. At 40k dense,
p95 was about1260ms without WAL and1480ms with WAL; allocated bytes were139.73MB
and145.82MB per pair. Matrix peak RSS377168KiB, including fixture construction.
This measures complete journaled execution, not fsync latency in isolation.

`bench-relational-matrix.txt` measures the unchanged Phase II shared readout,
100 iterations/cell. At 40k world neurons:

| Active neurons | Readout p50/p95/p99 ms | B/op | Parameter / structural reads |
|---|---:|---:|---:|
|40|.02127 / .02501 / .06712|1419|80 / 160|
|4000|2.291 / 2.542 / 2.712|33862|80 / 16000|
|40000|32.96 / 36.75 / 56.13|328784|80 / 160000|

The matrix peak RSS344880KiB includes the graph. Readout work follows active
nodes/connections; the four moments' representational limitation is unaffected.
Whole-thought size/WAL measurements are in `bench-legacy-size-matrix.txt` and
`bench-legacy-completed-*.txt`. The first combined log was truncated despite exit0;
missing cells were rerun individually and complete PASS output retained. Do not
infer missing measurements from process exit alone.

Native pair CPU, allocation and mutex profiles are retained. JSON map sorting
was16.88% cumulative CPU; relational feature construction17.01%. Allocation
sources included cloneState21.07%, cloneNativeEpisode11.69%, JSON Marshal42.53%
cumulative. Profile percentages overlap and include setup. There is no unsupported
claim of a separately measured learning-only or lock-contention speedup.

## Persistence resource cost

Version3 pending action pairs, one timed operation/cell; **single samples, not
latency percentiles**. Fixtures, schemas and replay equality were checked.

| World neurons | Checkpoint ms | Restore pending ms | Restore + replay pair ms |
|---|---:|---:|---:|
|1000|17.686|244.931|275.244|
|4000|72.478|1017.994|1142.475|
|40000|688.114|9640.967|10693.382|

At40k, allocated bytes were159713128 checkpoint,1262261800 restore and1422370224
restore/replay. Whole matrix peak RSS803764KiB; user/system CPU34.609/1.680s,
wall36.092s. JSON validation/restore costs remain a concrete scalability problem.
The metadata change deliberately does not alter the checkpoint format.

## Reproduction and evidence interpretation

Build test binaries before timed runs. Use the same source and toolchain for every
comparison, record hashes, run sequential ABBA and retain all rows and resources.

```
GOMAXPROCS=1 GOGC=100 go test -buildvcs=false ./services/ai/runtime -run '^$' -bench 'BenchmarkFoundationTransitions/nodes=40000/journal=false$' -benchtime=200x
GOMAXPROCS=1 GOGC=100 go test -buildvcs=false ./services/ai/runtime -run '^$' -bench BenchmarkNativeActionPair -benchtime=30x
GOMAXPROCS=1 GOGC=100 go test -buildvcs=false ./services/ai/runtime -run '^$' -bench BenchmarkNativeRelationalPolicy -benchtime=100x
GOMAXPROCS=1 GOGC=100 go test -buildvcs=false ./services/ai/runtime -run '^$' -bench BenchmarkNativeActionPersistence -benchtime=1x
python3 scripts/summarize_native_phase3.py EVIDENCE_DIRECTORY
```

`results.json` stores raw parsed benchmark rows and throughput (1e9/ns/op).
Profiles, test binaries, resource sidecars, exit records, diagnostic overlays and
unsuccessful/confounded runs remain in the evidence archive. Conservative legacy
optimization has its own integration evidence; broad native performance promotion,
larger scales, on-device Android and 3x speedup remain unproven.
