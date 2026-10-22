# Frozen canonical propagation v1

Synthetic fixture generated with Go 1.22.12 above parent commit
`3f3716cb283372a18cf12a0ae17fdf2ff515118f` (draft PR #46).
It starts with an empty Brain and explicitly selects `canonical-propagation-v1`.

The journal has 18 commands: three numeric groundings, two numeric spelling
exposures (`na 🌌` and an empty terminator), twelve local paired lessons, and
one two-frame numeric cognitive observation. There is no personal Brain data.

`reference.json` pins the journal, the complete detached schema-14 checkpoint
after replay, and both the complete native output and complete detached
checkpoint after each of eight additional real thoughts. These are exact SHA256
comparisons, with no floating-point tolerance. Fixed clocks are part of the
fixture. The checkpoint digest includes all persisted continuation state and
the propagation encoding; the runtime is detached so it has no journal anchor.

```sh
go run -buildvcs=false services/ai/runtime/testdata/canonical_propagation_v1/generate.go /absolute/new/output/directory
go test -count=1 ./services/ai/runtime -run '^TestCanonicalPropagationFrozenV1$'
```

The generator refuses existing directories. Preserve this v1 reference when
adding future numerical laws. Do not regenerate it to conceal behavioral drift.
This is numerical/recovery evidence, not evidence of language understanding or
an implementation of the proposed physical micro-information model.
