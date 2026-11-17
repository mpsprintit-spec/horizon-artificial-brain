# S2-R3 complete owner recovery artifact

[Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz](https://chatgpt.com/library?file_id=libfile_2cd2f0c3c4f881918341a31c89d99bd9&account_id=personal)

[Standalone S2-R3 diagnosis report](https://chatgpt.com/library?file_id=libfile_1a1162a2067881918ff683da7aef168b&account_id=personal)

- Size: **93,164,080 bytes**.
- SHA-256: `0137ba463fd78e85b130cba5213d37747582cf32ddf4acf6757c9d5d99a9d6f5`.
- **653/653 payloads authenticated**, with complete files rather than hash-only stubs.
- Original R2 full archive included unchanged, SHA-256 `e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46`; all 443 original payloads were verified before R3 acquisition.
- Source continuation **80735**; six final checkpoints **84799**.
- Three training seeds **37 / 83 / 151**; actual final features **32 / 40** under fixed ceilings 32 / 64.
- Diagnosis **INCONCLUSIVE**; actual S2 ability **PARTIAL**; strict retention and native persistence validated.

The saved owner artifact was downloaded again. Its size and SHA-256 match
exactly, every archived payload reauthenticates, and all six complete final
checkpoints and full journals are present. Final native checksums, feature
metadata, original R2 archive integrity and the published source commit were
checked from that fresh download:
[evidence/owner-download-verification.json](evidence/owner-download-verification.json).

The compatible scientific source is commit
`3e917d35eb117a30387484d62db6d6e18208ba92`, tree
`e40329f833f7d91c420e99f2a58eb89769e4d559`, based on unchanged #76
`7196c8a7f4a619040fe3933fe1c5363e5be661bc`. Publication used GitHub Git Data
after the local Git transport lacked write authentication; the published tree
was verified byte-identical to the locally tested scientific tree. No main or
S2-R2 branch ref was moved.

| Archive path | Complete contents |
| --- | --- |
| `s2r3/` | Source, every arm/block/final/restarted checkpoint, all full journals, receipts, native observations, causal input/feature/readout interventions, controls, profiles, sealed protocols and analyses |
| `source/` | Original immutable R2 recovery archive, including all earlier lineage and negative engineering evidence |
| `execution/` | Acquisition, Go/race/vet/build, recovery, resource/profile, analysis, metadata comparison and infrastructure logs |
| `preflight-no-acquisition/` | Initial environment-routing preflight; zero new experiences |
| `docs/` | Scientific report, code audit, adaptive requirements, deviations, restore instructions and compact evidence |
| `reproduce/` | Phase runner, frozen analysis, verifier/packager and narrowly checked metadata-audit finalizer |
| `implementation.patch`, `BUNDLE.json` | Source delta and exact published scientific revision |
| `MANIFEST.json` | Byte count and SHA-256 for every payload |

See [RESTORE.md](RESTORE.md) for native checkpoint/journal pairs. XZ uses a
128 MiB dictionary to retain all identical historical prefixes losslessly;
no journal prefix is omitted. The raw Go diagnostic nil/empty-slice comparison,
initial false persistence summary and independently verified correction are
preserved. No native parameter or frozen scientific analysis was changed.

This artifact pointer and fresh-download receipt are intentionally outside
the immutable self-authenticating archive because they were written after
saving. No research Brain was promoted, no main/production state was replaced,
no PR was automatically merged and no S3 acquisition occurred.
