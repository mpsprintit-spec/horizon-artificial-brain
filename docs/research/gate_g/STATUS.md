# Gate G — final continuation record,14 November2026

**Decision: Gate G NOT PASSED. G8 needs repair;81/90 comparisons meet budget,9 fail.**
G1–G7/G9/G10 pass the registered technical/functional cases. Native stays opt-in;
PR67 stays draft. No main merge/default promotion or user-state overwrite.

## Branch and exact source

Branch: `research/gate-g-coexistence-20261113`.
PR: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/67.
Main: `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.
Phase III: `0df3f19a33bd7de69ed496293591d734331c5e51`.
Frozen measured/tested local source: `9cca3e645203ceff84de3f214923b210f478a66e`.
Published equivalent: `45b13f2e46c515f0ec5f67a49f2063c9217935a2`.
Exact common tree: `99c1648594661cffe1a5f00ba974fedffd915e66`.
Runtime code unit: localb5b70fa / publishedaf1217f.
Final report/evidence-only revision is the commit containing this record; get its
full HEAD from git rev-parse HEAD or PR67. Do not confuse that docs suffix with
the measured source. Native/main/PR63 ancestry is preserved without force push.

## Completed checks and evidence

-27 mixed order/seed cases: full canonical bytes, activation, sequence, checksum,
  genesis replay, checkpoint suffix and further learning after restore match.
-15 real subprocess crash boundaries pass; orphan repair preserves a byte-equal
  backup, paired outcome retry learns once; no recovery physical dispatch.
- Native models1/2/3, actual production startup, composition, corruption,
  downgrade/matched-prefix rollback and active legacy isolation pass.
- Explicit and automatic private-policy address aliases are blocked; correctly
  checksummed unsafe world/policy checkpoints are rejected before target mutation.
- Frozen activity law equivalence:384 transitions; typed sort/tie order27 cases.
- Latest full ordinary Go suite, targeted race, vet/plain build,4 Android arm64
  cross-builds and83 focused test-pass records have no failure.
- Race covers all69 packages:62 local remaining packages plus7 exact-source CI
  packages. CI43550972852 passed validate/monitor, nested adapters, trained
  dialogue/recovery, Android and live Bridge. Prior local20m full-race timeout
  remains retained; it is not relabeled as a successful monolithic run.

## Performance verdict and remaining work

Original measured codeaf3b946 failed9/90 budgets; its baseline sensor-refusal
assumption also failed because the old sensor route bypassed the blanket guard.
Corrected Process-bearing negative control rejects as expected.

Repaired source45b13f2 completed all25 planned commands with expected exits,
including corrected negative control. One managed session disappeared during
B1-persistence; partial output is retained and the whole unfinished command was
repeated unchanged. Completed results were preserved; remaining commands ran
one per session in the original ABBA order. No benchmark remains active.

Repaired result:81/90 within budget,9 failures. All allocation checks pass;
remaining failures are thought/native learning/native-idle p95 and checkpoint4k
mean latency. Full cases/limits are in PERFORMANCE.md and the retained summary.
40k sparse+WAL p95 improves18.115→13.245ms; dense4k allocations41040.5→25028/op,
but dense4k p95 still fails35.08→41.76ms against40.442ms limit. Do not infer a
whole-gate pass from selected improvements.

Next implementation: profile the remaining1k, learning4k and checkpoint4k cells
on pinned sources, isolate CPU/wall/JSON/prefix costs, implement only a verified
bottleneck repair, preserve complete-state equivalence and all safety checks,
freeze source, then run a fresh full confirmation with UNCHANGED budgets.
No fully isolated cause or verified fix for all remaining tail regressions is
claimed. Do not merge while G8 fails. Android execution and power-loss behavior
remain unmeasured; C/D/H scientific limits from PR64 remain unchanged.

## Reproduction and publication

REPORT.md and RESULTS.json cover all subgates/deliverables. Raw commands, source
hashes, failing/passing logs, profiles and synthetic crash artifacts are retained
in evidence/Gate_G_Evidence_20261114.zip with per-payload manifest; ARCHIVE.json
records the archive digest. The initial technical-evidence.zip stays immutable.
The final artifact commit must be successfully published before claiming these
newer artifacts are saved; final PR metadata records that published commit.

```sh
go test ./... -count=1
go vet ./...
go build ./...
python3 scripts/run_gate_g_confirmation.py --baseline BASELINE --candidate CANDIDATE --output NEW_DIRECTORY
python3 scripts/summarize_gate_g_confirmation.py NEW_DIRECTORY
```

Keep the common benchmark harness identical on baseline and candidate. Baseline
is1e12332 (published609ee918), with no integration source copied into it. Use
short managed sessions/continuous polling; step runner can continue the preserved
command order after a session loss. Never run fault tests on user-state paths.
