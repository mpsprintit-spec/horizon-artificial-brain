# Memulihkan S2-R2 tanpa pelatihan ulang

Gunakan arsip pada ARTIFACT.md dan implementasi branch
`research/s2-r2-native-learning-repair-20261117`. Cocokkan SHA-256 arsip dan
jalankan verifier sebelum mengekstrak ke direktori penelitian kosong:

```sh
python3 scripts/package_s2r2_repair.py --verify /absolute/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz
mkdir /absolute/r2-restored
tar -xJf /absolute/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz -C /absolute/r2-restored
```

| Tujuan | Checkpoint di dalam arsip | Journal |
| --- | --- | --- |
| Kandidat yang diuji holdout | `s2r2/s2r2-final.json` | `s2r2/s2r2-final-events.jsonl` |
| Continuation terbaru | `s2r2/s2r2-continuation.json` | `s2r2/training-combined/events.jsonl` |
| Rollback ke sumber asli | `s2r2/00-source.json` | `s2r2/source-events.jsonl` |
| Revisi engineering yang gagal | `failed-engineering-revision/s2r2-final.json` | `failed-engineering-revision/training-combined/events.jsonl` |

Final sequence 80681, native checksum
`9c6cbd9e7d7caeeaaa0a788d0122a94fab32701ceda2c8bb6476fa3c522d47d8`.
Continuation sequence 80735, native checksum
`9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.
Rollback source sequence 76617, native checksum
`b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d`.

Gunakan fungsi native `OpenEventLog` dan
`RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)`, lalu `SetEventLog`
jika salinan penelitian memang akan dilanjutkan. Journal terbaru mereplay suffix
sesudah checkpoint yang diberikan: pasangan final memakai journal final yang
dibekukan agar tidak otomatis maju ke continuation. Semua pasangan di atas telah
diautentikasi dan direplay. Jangan memotong atau mengedit journal historis.

Source asli lengkap berada di `source/`, dengan MANIFEST aslinya; prefix
A–E1 → S1 → S2 tetap utuh. `implementation/` memuat file implementasi/audit yang
digunakan; checkout repository menyediakan dependensi kode lainnya. Gunakan
implementasi S2-R2 saat membaca opsi proposal dalam journal baru. Kompatibilitas
yang diuji adalah implementasi baru membaca checkpoint/WAL lama, bukan binary
lama memahami kontrak proposal penelitian yang baru.

Pemulihan bukan pelatihan baru. Jangan memanggil Prepare/Train S1 atau S2 dan
jangan memasang state penelitian ini menjadi Brain produksi secara otomatis.
