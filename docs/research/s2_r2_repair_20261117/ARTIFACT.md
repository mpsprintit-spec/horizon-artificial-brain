# S2-R2 complete recovery artifact

[Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz](https://chatgpt.com/library?file_id=libfile_3518010cf7a08191848d04052d77571d&account_id=personal)

- Size: **18,992,952 bytes**.
- SHA-256: `e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46`.
- **443/443 payloads authenticated**, including complete files, not hash-only stubs.
- Source original: sequence 76617; final: 80681; continuation: 80735.
- Repair: **PASS**; S2 ability: **PARTIAL**.

The saved owner artifact was retrieved again using its exact file identity and
rehashes to the identical SHA-256. XZ uses a 128 MiB dictionary to compress long
identical prefixes losslessly; every full checkpoint and journal remains in the
archive. No journal is represented only as a checksum or omitted prefix.

| Archive path | Contents |
| --- | --- |
| `s2r2/` | Revised research lineage, controls, final/continued/restarted checkpoints, full journals, observations, receipts, pre-operation ledger, protocol lock and metrics |
| `source/` | All 75 authenticated original S2 artifact payloads, with original manifest and A–E1/S1 prefix material |
| `failed-engineering-revision/` | Complete first R2 revision, strict S1-retention failure, checkpoint/WAL and exact replay proof; no final holdout was run on it |
| `execution/` | Phase, race, regression, failure, reproduction and recovery logs |
| `implementation/` | Native learner changes, tests, read-only probes and reproduction/analysis/packaging scripts |
| `docs/` | Report, frozen criteria, before-holdout amendment, evidence and restore instructions at packaging time |
| `MANIFEST.json` | Byte count and SHA-256 for every archived payload |

Use RESTORE.md for exact checkpoint/journal pairs. The external ARTIFACT.md is
written after saving, so it is deliberately outside the self-authenticating
bundle. Later GitHub CI and download-verification receipts can likewise appear
alongside this link without rewriting the immutable published archive.

No source artifact, production checkpoint or main branch was replaced. No PR
was automatically merged and no S3 acquisition was started.
