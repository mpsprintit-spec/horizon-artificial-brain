# Artefak pemulihan S2 — VERIFIED, hasil ilmiah PARTIAL

[Horizon_S2_Native_Self_Evaluation_20261116.tar.xz](https://chatgpt.com/library?file_id=libfile_f3dad8127018819192a6c31c44d72f5b&account_id=personal)

- Akses mengikuti izin pemilik file; tautan bukan unduhan publik anonim.
- Ukuran **61.142.188 byte**.
- SHA-256 **`661d007850d3026ed22636fdc4d96e9113115fb2b9e2d66856ad88e4ba60a49f`**.
- **75 payload** telah dibaca dan cocok ukuran/SHA-256 manifest internal.
- Setelah ekstraksi ke direktori baru, `TestS2Verify` memulihkan checkpoint/WAL
  menggunakan mekanisme native dan memeriksa estimasi yang identik.
- Verifikasi artefak tidak melatih ulang atau menghasilkan holdout baru.

| Komponen | Path di dalam arsip |
| --- | --- |
| Checkpoint final | `s2/lineage/s2-final.json` |
| Checkpoint continuation | `s2/lineage/s2-continuation.json` |
| Journal A–E1 → S1 → S2 lengkap | `s2/lineage/events.jsonl` |
| Sumber S1 asli | `source/s1/lineage/` |
| Rollback checkpoint/prefix-WAL | `s2/rollback-observed.json` dan `.events.jsonl` |
| Protokol, receipt, holdout, kontrol, diagnosis | `s2/` |
| Kode akuisisi asli yang hash-nya dicatat sebelum holdout | `s2/acquisition_harness.go.txt` |
| Kode reproduksi/verifier dan manifest | `reproduce/`, `MANIFEST.json` |

Final: sequence76602, SHA-256
`60ae19a96f22183596e04b5a6965a6ecef2b9efd45a75755dcf1ebd97c5c3c42`, native checksum
`f89ea7ef61330d6ebf6bc0cd60bf5aec49fe6a487e440843cbc646dfefdc13a3`.

Continuation: sequence**76617**, SHA-256
`b85a818a84efb8185cdbf7003e14a8acd3d070c798f3ae7e053729ec9fc76ba0`, native checksum
`b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d`.

Journal lengkap: **44.021.001 byte**, SHA-256
`e7157d7c1e3b03c38abd3825e3a6bd8aef0bcbdf748b291e48d99de356d7d527`,
76618 physical records / sequence76617. Physical record tambahan berasal dari
intent/commit historis pada revision10, bukan duplikasi pembelajaran S2.

Lihat [RESTORE.md](RESTORE.md) untuk verifikasi dan recovery tanpa pelatihan,
[REPORT.md](REPORT.md) untuk gate ilmiah, dan
[S3_SOURCE_CANDIDATE.json](S3_SOURCE_CANDIDATE.json) untuk metadata lineage.
Continuation dapat dipertahankan, tetapi **kesiapan kemampuan S3 belum
terpenuhi**, checkpoint tidak dipromosikan ke produksi, dan PR tidak di-merge otomatis.
