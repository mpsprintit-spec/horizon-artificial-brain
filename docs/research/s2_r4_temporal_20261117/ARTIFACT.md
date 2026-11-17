# S2-R4 owner artifact

The full recovery archive is delivered in two parts. Both parts are required;
concatenate part1 then part2 before extracting. All files were saved successfully,
then both parts were freshly retrieved and checked byte-for-byte by SHA-256.

- [S2-R4_TEMPORAL_ERROR_HISTORY_LEARNING.md](https://chatgpt.com/library?file_id=libfile_b8c0e4d0ecbc8191a2a91928f7f609b6&account_id=personal)
- [Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part1](https://chatgpt.com/library?file_id=libfile_f46346e1b1e08191a14ea298e8d3ec17&account_id=personal)
- [Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part2](https://chatgpt.com/library?file_id=libfile_4c7ef8691fcc819180569bce39bc25c1&account_id=personal)
- [S2-R4_PEMULIHAN_ARTEFAK.md](https://chatgpt.com/library?file_id=libfile_4c95b775f3848191bd3fbe241e0ccfce&account_id=personal)

The restored archive is `Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz` (**626908084 bytes**).
Outer SHA-256: `4d29df0bae3e92e808fd540d6a54d817d8f641c89047e14e19f41103cb83ab0a`.
**2,250/2,250 authenticated payloads verified.**

| Part | Bytes | SHA-256 |
|---|---:|---|
| Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part1 | 314572800 | `086c785d4f35b00eff2cafaae381d0f3368f5c6df6ebaa03315e814ddadcee53` |
| Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part2 | 312335284 | `0e7bc420b1033ff9431769f8cd0649a66e1e2c7a05181c8809a2ce4821771c8f` |

```sh
cat Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part1 Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz.part2 > Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz
python3 scripts/package_s2r4_temporal.py --verify /absolute/Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz
mkdir restored-r4
tar -xJf Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz -C restored-r4
```

The complete manifest covers original and research checkpoints, full historical
WAL prefixes and new suffixes, all failed experiments, observations/ledgers,
learning/candidate receipts, metrics, source/configuration hashes, reproduction
scripts and the unchanged R3 source archive. See [RESTORE.md](RESTORE.md).
Neither source checkpoint nor journal was changed. No research model is promoted.
Main and the production Brain remain untouched.

## Exact scientific source

The pre-holdout frozen local scientific commit is
`cee0fa80dfa1667ee2738c3467463eb1fb8313ab`. The published equivalent is
`262ab99b10a5aa7bdb22b2821c71b0803272e60a`; **both have exactly the same Git tree**
`2d7b820ff0cb30fe7a8428bb4d031cf684f8b1fb`.
The publication service assigns commit metadata, so commit IDs differ without
any source content change. `PUBLICATION_MAPPING.json` records both mappings.
`S2R4_Source_History_20261117.bundle` preserves the exact original local commit
history through `f342b03ad7d5511a025812aa7a0c55a69c823ea9`, verified with `git bundle verify`.
The bundle requires only the verified R3 base.

Brain source: R2 continuation sequence **80735**, native checksum
`9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.
Scientific verdict: **S2 PARTIAL**. The primary 32-feature arm did not demonstrate
history learning; 64-feature diagnostic gains did not clear all ability/retention
gates. See the report for all negative results and limitations.
