# Continuation — final native integration

- Main:3a9e7ce3ffadff86c792050056ebea4eda48da0b; tidak diubah.
- Source branch:integration/native-engineering-closure-20261114; PR70 draft.
- Last pushed source:cfeac007dec4c5d1fe86cc885a4ce12a07c9e144,
  tree8292bd089ac5d4742d0320bd9873bfd65035453c; CI43620396195 SUCCESS.
- Current G8:86/90, empat gagal; sebelumnya1069c02 menghasilkan89/90.
  Kedua matriks utuh ada di evidence; jangan memilih hasil terbaik.
- Append overlayfs:p952.75295ms >.62782ms; bytes/op PASS.
- Full tests/vet/build/nested/Bridge/CI PASS; root race67 paket covered7CI+60local,
  saved-state race PASS; Android arm64 CLI/chat/production cross-build PASS.
- No merge, force push, controller default, user checkpoint mutation or retraining.

Primary lineage tetap archive PR69 yang sama, sequence75331/updates10752:
e1-main-continuation/07-E1-restarted.json + events.jsonl. Original archive SHA
7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3.
Full hashes, ownership and exact retention/replay evidence ada di REPORT.md.
Jangan memakai hasil test disposable sebagai pengganti lineage utama.

Next: measured diagnosis of the four current failed cells and append Sync/stat
boundary; retain all logs, fix only proven bottlenecks, keep original thresholds,
then full new frozen confirmation. No more A–D/E1 acquisition or architecture work.

```sh
python3 scripts/prepare_native_continuation.py --destination /tmp/horizon-saved-new
HORIZON_NATIVE_CONTINUATION_DIR=/tmp/horizon-saved-new \
  go test ./services/ai/runtime -run '^TestNativeIntegrationSavedContinuation$' -count=1 -v
go test ./...
go test -race -timeout=40m ./...
go vet ./...
go build ./...
python3 scripts/run_gate_g_confirmation.py --baseline BASELINE609EE918 --candidate FROZEN_CANDIDATE --output NEW_EMPTY_DIRECTORY
python3 scripts/summarize_gate_g_confirmation.py NEW_EMPTY_DIRECTORY
```

Use Go compiler1.22+; measurements pinned1.23.2. Common harness on baseline must
match SHA f29039161c8e93d0f62c9e025a0473e3821399187daed5471dbb38629e6d6b48.
Do not overwrite earlier outputs. Source publication and evidence publication
are separate; current branch HEAD/PR70 give the exact published commit.
