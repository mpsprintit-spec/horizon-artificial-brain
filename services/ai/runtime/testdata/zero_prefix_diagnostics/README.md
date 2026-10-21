# A10 recovery diagnostics

All probes use temporary synthetic Brains. No personal Brain, physical device,
published service, or main-branch deployment is involved.

Source audit: `Horizon_Audit_Ulang_PR43_2026-10-21.zip`, SHA256
`ceaa87caffd0ad650e0ff60df675bcc292657dfc1beabba93725db1afab9132f`.
`reproducer.go` is the unchanged `probe/zeroanchor/main.go` from that archive.

`reproducer-before.json` was generated with unmodified PR #44
`56afe5dd449eeebc6cb69e0e13561589fb54002f`. The same public-API source generated
`reproducer-after.json` using this candidate. The zero-checkpoint anchor is
identical; empty→append recovery changes from an error to success. The nonempty
revision-one control still recovers revision two.

`regression-before.txt` is an intentional failing run of the new regressions
against the exact parent `checkpoint_journal_anchor.go`, supplied with a Go
overlay. It reproduces the three allocated-empty hash failures, two zero→one
recovery failures, and the zero-prefix supervisor failure. Exit status 1 and
the expected errors were verified; this is evidence of the original bug.

`focused.txt`, `go-test.txt`, `race.txt`, and `bridge-smoke.txt` record passing
candidate runs with Go 1.22.12 on Linux/amd64:

```sh
go test -count=1 ./services/ai/runtime -run 'Test(JournalPrefixHashCanonicalEmptyAndUnchangedNonempty|CheckpointJournal|ZeroPrefix)'
go test -count=1 ./...
go vet ./...
go build ./...
go test -race -count=1 ./services/ai/knowledge ./services/ai/learning ./services/ai/runtime ./services/ai/bridge ./services/ai/core ./services/ai/cmd/horizon-cli
make bridge-smoke
```

Vet and build exited successfully without output. Bridge smoke may log an
initial refused connection while its listener starts; all readiness and final
integration checks passed. Test durations are diagnostic, not benchmarks.

The root Go module does not cover the repository's separate nested modules.
Monitor code is unchanged; the exact-head CI also validates its typecheck,
37 tests, and build. No language curriculum or scale/performance experiment
was run by this patch.

Run the unchanged public-API probe from either checkout's module root:

```sh
go run -buildvcs=false /absolute/path/to/new/checkout/services/ai/runtime/testdata/zero_prefix_diagnostics/reproducer.go
```

The program preserves diagnostic errors in JSON rather than exiting nonzero;
inspect all three recovery error fields and the control revision. Assertions
for the repaired behavior live in the regression tests.
