# Synaptic spelling codec v1

Synthetic numerical I/O fixture generated with Go 1.22.12 by the new codec v1
implementation above parent `12aa24e142f9b025e803642d8df3592a1ce7dd13` (PR #43).
It is not a user's Brain or evidence of semantic understanding.

`events.jsonl` contains a numerical population grounding, four scalar-only I/O
exposures for `A🌌`, and one text-free thought. The frozen reference contains the
decoded spelling and canonical/output SHA256 digests before/after that thought.
The test replays the prefix and checks the original readout and next thought;
future codec changes must preserve v1 rather than regenerate its expectations.

Reproduce into a new directory:

```sh
go run -buildvcs=false services/ai/runtime/testdata/learned_spelling_v1/generate.go /absolute/new/directory
go test ./services/ai/runtime -run '^TestLearnedSpellingFrozenV1' -count=1
```

The generator refuses an existing directory. Regeneration is for byte comparison,
not permission to replace the frozen fixture after changing the v1 laws.
