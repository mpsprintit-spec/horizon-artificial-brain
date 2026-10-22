# Frozen learned-continuation-v1 readout

Synthetic fixture produced with Go 1.22.12 by this new readout implementation
above parent `99820e0c7912440f84fa4e0a223a3880b88bbf7b` (draft PR #45).
It is not a personal Brain or proof of world/language understanding.

`events.jsonl` contains **25 commands**: four numeric population groundings,
three scalar spelling exposures (`na `, `lu`, and an empty terminator), and
eighteen paired numerical transition lessons. There is no real cognitive
observation/thought in this small fixture. The prefix is hypothetical input.

`reference.json` pins the original journal SHA256, canonical Brain SHA256,
readout contract, seed, time, and exact complete readout `na lu`. No whole
phrase is stored in the trained canonical Brain or learned as a single form.

```sh
go run -buildvcs=false services/ai/runtime/testdata/learned_continuation_v1/generate.go /absolute/new/output/directory
go test -count=1 ./services/ai/runtime -run '^TestLearnedContinuationFrozenV1$'
```

The generator refuses existing output directories. Future implementations must
preserve the frozen v1 contract; do not replace expectations to hide drift.
This exact readout fixture does not assert bitwise native future thinking:
the inherited opposing-channel roundoff issue is documented separately.
