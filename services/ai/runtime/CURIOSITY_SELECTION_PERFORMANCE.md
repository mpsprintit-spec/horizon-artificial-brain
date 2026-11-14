# Curiosity selection: allocation repair without changing the policy

Baseline: main `32df59b9285d829ae73429c7f277621e63de7ce3` (PR60).
This change retains the existing curiosity formula, heartbeat, score/lowest-ID
tie break, output boost and canonical write boundary. It does not introduce or
enable a new cognitive controller.

`applyCuriosityDrive` previously copied all node addresses, accumulated every
positive candidate, stable-sorted them, then consumed only element zero. A
single-pass lexicographic maximum produces the same winner for finite scores.
It uses the existing PR60 stable address visitor, holds the Brain read lock,
never mutates membership or re-enters the registry, and stores only the winner.
Nonfinite in-memory states retain the historical stable-sort fallback. Nothing
new is persisted; there is no schema migration or neural learning-law change.

A 512-case seeded numerical comparison checks the old implementation against the
new one, including ties, negative frequency, clamping, registry growth and NaN
ordering. Existing curiosity/continuous/replay regression tests remain applicable.

## Matched performance measurement

Same exposed Intel Xeon Platinum 8573C environment, Go1.23.2, Linux amd64,
GOMAXPROCS=1, GOGC=100. The benchmark is the same synthetic 40,000-neuron /
160,000-synapse sensor-plus-one-thought workload used in native foundation work.
No native controller is initialized. Nine nodes are active at the measured
boundary. Three warmups precede 200 measured iterations. Sequential ABBA:
main, candidate, candidate, main; no other deliberate heavy workload runs.

| Metric | Main | Candidate |
|---|---:|---:|
| Thought p50, ms |10.385|9.6475|
| Thought p95, ms |24.720|12.255|
| Thought p99, ms |33.590|14.575|
| Complete sensor+thought mean, ms |24.187799|21.018072|
| Complete iterations/s |41.343|47.578|
| B/op, sensor+thought |3,855,267.5|114,244.5|
| Allocations/op |1,215.5|1,186|
| GC cycles / 200 iterations |13|0|

These are means of two run statistics, not pooled percentiles or confidence
intervals. The thought p95 ratio is **2.017x**; allocated bytes fall **97.04%**.
Peak process RSS (including fixture construction): baseline321016–321048KiB,
candidate310088–310232KiB. Process user/system CPU seconds are retained in the
Phase III raw archive. This does not establish Android on-device performance or
a 3x improvement. Do not multiply this ratio by an old result from another host.

The Phase III diagnostic found a PR61→PR62 p95 difference despite almost equal
mean latency/allocated bytes. Disabling GC only after normal setup and an explicit
collection reduced that difference from about10% to0.46% (13.035→13.095ms),
implicating GC-associated tails. Disabling GC during fixture construction instead
was confounded by retained setup garbage and is not the primary diagnostic.
A legacy allocation profile identifies curiosity candidate construction as the
largest workload allocation source. This patch removes that source; **production
GC settings are unchanged**. Other full-graph activity-plasticity and memory-dynamics
traversals remain CPU costs; this is not a claim that all scalability issues are solved.

## Raw primary samples

```text
goos: linux
goarch: amd64
pkg: github.com/project-horizon/horizon-core/services/ai/runtime
cpu: INTEL(R) XEON(R) PLATINUM 8573C
BenchmarkFoundationTransitions/nodes=40000/journal=false         	     200	  24420711 ns/op	         0.3512 GC_pause_ms	        13.00 GCs	         9.000 active_nodes	    160000 fixture_synapses	        10.68 sensor_p50_ms	        26.95 sensor_p95_ms	        34.94 sensor_p99_ms	        10.45 think_p50_ms	        24.86 think_p95_ms	        34.91 think_p99_ms	 3855274 B/op	    1216 allocs/op
PASS
goos: linux
goarch: amd64
pkg: github.com/project-horizon/horizon-core/services/ai/runtime
cpu: INTEL(R) XEON(R) PLATINUM 8573C
BenchmarkFoundationTransitions/nodes=40000/journal=false         	     200	  21235521 ns/op	         0 GC_pause_ms	         0 GCs	         9.000 active_nodes	    160000 fixture_synapses	        10.69 sensor_p50_ms	        13.58 sensor_p95_ms	        17.49 sensor_p99_ms	         9.684 think_p50_ms	        13.35 think_p95_ms	        16.60 think_p99_ms	  114235 B/op	    1186 allocs/op
PASS
goos: linux
goarch: amd64
pkg: github.com/project-horizon/horizon-core/services/ai/runtime
cpu: INTEL(R) XEON(R) PLATINUM 8573C
BenchmarkFoundationTransitions/nodes=40000/journal=false         	     200	  20800623 ns/op	         0 GC_pause_ms	         0 GCs	         9.000 active_nodes	    160000 fixture_synapses	        10.63 sensor_p50_ms	        12.62 sensor_p95_ms	        15.60 sensor_p99_ms	         9.611 think_p50_ms	        11.16 think_p95_ms	        12.55 think_p99_ms	  114254 B/op	    1186 allocs/op
PASS
goos: linux
goarch: amd64
pkg: github.com/project-horizon/horizon-core/services/ai/runtime
cpu: INTEL(R) XEON(R) PLATINUM 8573C
BenchmarkFoundationTransitions/nodes=40000/journal=false         	     200	  23954886 ns/op	         0.3904 GC_pause_ms	        13.00 GCs	         9.000 active_nodes	    160000 fixture_synapses	        10.57 sensor_p50_ms	        26.94 sensor_p95_ms	        33.42 sensor_p99_ms	        10.32 think_p50_ms	        24.58 think_p95_ms	        32.27 think_p99_ms	 3855261 B/op	    1215 allocs/op
PASS
```

The benchmark harness and process resource sidecars are retained with the Phase III evidence. The untracked local harness is not included in this production change.
