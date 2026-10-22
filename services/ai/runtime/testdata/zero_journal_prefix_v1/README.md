# Historical revision-zero journal anchor

This small synthetic fixture was produced with Go 1.22.12 by the **unmodified**
PR #44 checkout at `56afe5dd449eeebc6cb69e0e13561589fb54002f`. It does not
contain a user's Brain. The generator ran from that checkout's module root,
before the new empty-prefix reader was used.

`zero-checkpoint.json` is the original schema-13 checkpoint at revision zero,
bound to an empty, open journal. Its authenticated anchor is SHA256 of `null`:

`74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b`

`events.jsonl` contains two later learning commands, with fixed timestamps.
`reference.json` pins the original file digests, the complete detached runtime
checksum after both commands, and the next text-free thought's output digest
and resulting complete runtime checksum. Those references came from the
original live writer; its broken offset-recovery path was not used to create
the expected state.

The regression verifies the original checkpoint without rewriting its checksum,
replays the suffix, and compares the continuation and next thought. It also
requires a new revision-zero writer to produce byte-identical checkpoint output.
Do not regenerate expectations with a new implementation to hide a compatibility
regression.

To reproduce in a **new** directory, run the generator using the historical
module, even though the generator file itself lives in the new checkout:

```sh
cd /absolute/path/to/unmodified/PR44/checkout
git rev-parse HEAD # Must be 56afe5dd449eeebc6cb69e0e13561589fb54002f.
go run -buildvcs=false /absolute/path/to/new/checkout/services/ai/runtime/testdata/zero_journal_prefix_v1/generate.go /absolute/new/output/directory
```

Run the regression from the new checkout:

```sh
go test -count=1 ./services/ai/runtime -run '^TestZeroPrefixFrozenPR44CheckpointAndNextThought$'
```
