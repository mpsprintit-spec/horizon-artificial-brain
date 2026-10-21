# Frozen ISF001 readout evidence

These fixtures come from the user's synthetic experiment archive
`Horizon_Uji_Pengetahuan_Imajiner_ISF001_2026-10-21.zip`, version 2, including
`readout-check/Horizon_Pemeriksaan_Prediksi_dan_Penyebutan_2026-10-21.md`.
They are experimental data, not the user's personal Brain Memory.

The original experiment used commit
`0b893e905742e27e3c9c4ad2a79c5c4fb4747cc0` (PR #40). The complete, unchanged
schema-12 checkpoint is gzip-compressed here. The cases retain all 84 native
outputs, the 42 pairs of numeric input events, target populations for external
scoring, canonical post-case hashes and the former generic answer rankings.
No source modification, new training or target-based reranking produced them.

| File/content | SHA256 |
| --- | --- |
| Source ZIP, version 2 | `578c679a67fdd6b23b8fc766e4b2ecf7510a2ee94bd5fa57aaf59de1f7c086f2` |
| Decompressed checkpoint | `1d22349376aca4488f875b8f372fbddb9ea27eab7da0be08ff519e3b7961331b` |
| `isf001-checkpoint.json.gz` | `2e0e0fbcf59ba577906b5a722851efbca036a2a36ff6e64a0ea3572cb07571f8` |
| `isf001-cases.json.gz` | `5ac0eb4e897ab68ede5e7b0c773494f437dcce441e724a90330341b590a986a8` |

The checked-in compressed files are 49,789 and 64,211 bytes. Generate another
copy from the exact archive with `generate.py ZIP NEW_DIRECTORY`; it refuses an
existing output directory. Python/zlib versions can affect gzip container bytes;
the checkpoint payload digest is the authoritative source identity.

`TestNeuralAnswerReadoutFrozenISF001` restores each case independently, in normal,
hidden-label and permuted-label conditions. It requires every native output and
every canonical post-case hash to match the original experiment. Only the typed
answer projection changes. The target is never passed to the native processing
or readout function.

This is numeric replay of previously learned language populations, not visual
or physical grounding. The original generalization failures remain test
expectations: confident target selection is 18/18 on seen cases, 0/18 with new
actors, and 4/6 with new condition combinations. None is a generated sentence.
