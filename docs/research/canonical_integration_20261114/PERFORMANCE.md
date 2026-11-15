# Performance diagnosis dan CI

## Tujuan

Profilkan hanya failure G8 yang sudah tercatat; jangan mengganti baseline,
threshold, WAL atau validity contract. Pisahkan diagnosis dari konfirmasi gate.

Baseline yang autentik berhasil ditemukan kembali sebagai commit GitHub609ee918
(tree `bc640beec3f9e4da864f2df0fd1b92292c82d1b4`), identik dengan tree baseline
lokal1e12332 dalam artefak ABBA terdahulu. Harness bersama disalin unchanged,
SHA256 `f29039161c8e93d0f62c9e025a0473e3821399187daed5471dbb38629e6d6b48`.
Candidate801ca9a mempunyai runtime produksi yang sama dengan source freeze E1;
PR69 tidak mengubah activation/learning/journal/checkpoint runtime.

## Pengujian diagnostik

Go1.23.2, GOMAXPROCS1/GOGC100; CPU profiler aktif. Thought100 sampel,
native20 pairs, checkpoint3 operasi. Profil mencakup proses/fixture setup,
bukan pengukuran CPU eksklusif p95. Angka berikut **bukan** ABBA konfirmatori.

| Failure family | Baseline diagnostic | Current diagnostic | Metrik |
| --- | ---: | ---: | --- |
| Sparse1k/no WAL | .5386 | .5635 | Thought p95 ms |
| Sparse1k/WAL | .6304 | .5290 | Thought p95 ms |
| Dense1k/no WAL | 6.740 | 8.179 | Thought p95 ms |
| Dense4k/WAL | 26.00 | 22.00 | Thought p95 ms |
| Learning4k sparse | .04229 | .03765 | Learning p95 ms |
| Learning4k dense | 6.377 | 4.755 | Learning p95 ms |
| Checkpoint4k | 64.864866 | 53.572178 | Mean ms/op, bukan p95 |
| Native-idle4k dense/WAL | Tidak ada baseline idle yang menerima mixed commands | 23.37 | Thought p95 ms |
| Native-idle40k sparse/no WAL | Tidak diprofilkan ulang counterpart baseline40k | 16.86 | Thought p95 ms |

Dense1k masih memperlihatkan tail lebih tinggi meskipun median4.323 vs4.365 ms,
B/op1.571MB vs1.683MB dan GC pause lebih rendah. Tidak ada bukti bahwa sekadar
menghapus satu pemeriksaan ownership/durability adalah perbaikan yang sah.

Profil dense4k current didominasi map assignment/access/iteration/hash;
Engine.advance mengambil38.11% cumulative CPU proses. Profil checkpoint4k
didominasi JSON validation/skip/rescan, serialisasi/copy serta SHA256. Native
learning juga memuat biaya feature traversal, hashing trajectory dan serialisasi
command; profil ini belum mengisolasi sebab tunggal sembilan delta tail lama.
Optimasi caching yang belum terbukti invalidation/replay-nya tidak ditambahkan.

Percobaan awal staging baseline tidak menyalin harness benchmark yang tidak
tracked dalam tree baseline; sejumlah command hanya menghasilkan PASS tanpa
baris benchmark. Hasil itu tidak digunakan sebagai pengukuran. Raw attempt
dipertahankan di g8-diagnosis, harness diperbaiki, dan baseline diulang dengan
assertion bahwa baris benchmark benar-benar ada (g8-baseline-corrected).
Semua pengukuran candidate dan kedua attempt tetap tersedia dalam archive.

## Keputusan G8

**BLOCKED: status konfirmasi terakhir tetap81/90, sembilan gagal.** Tidak ada
patch runtime yang dibuktikan menyelesaikan seluruh failure, sehingga tidak ada
konfirmasi90-cell baru atau klaim Gate G lulus. Anggaran asli tetap1.15x+.10ms
thought,1.20x+.10ms stages/persistence dengan allocation budgets existing.
Profil satu run tidak menghapus hasil negatif terdahulu. Prioritas berikutnya
adalah isolasi tail CPU/GC/wall-time pada1k dan repeated checkpoint4k dengan
source/harness yang dipin sebelum mengusulkan patch dan full confirmation.

Perubahan63 hanya memperbaiki startup; tidak mengubah hot path thought/native
dan tidak membutuhkan dependensi Gate G. E1 mengubah isi N melalui learning,
bukan sumber runtime. Tidak ada klaim performa real-device Android.

## Verifikasi final source

| Kandidat | Hasil |
| --- | --- |
| Main readiness cee28ff | Full Go tests PASS; full scoped race PASS (runtime972.301s); standard vet/build dan Android production cross-build PASS pada clone bersih |
| Main readiness CI43597512222 | SUCCESS untuk HEAD cee28ff2431b2a5f5d2ea61c668c4de262c25426 |
| Research source a5d2842 | Full Go tests PASS; full scoped race PASS (runtime1078.584s); standard vet/build dan Android CLI cross-build PASS pada clone bersih |
| Research source CI43597835457 | SUCCESS; termasuk langkah prepare parent PR68 dan TestContinual(EPrerequisites/E1) |
| Safety regressions | Mixed transactions, identity/ownership, policy admission, old versions, legacy isolation, corrupt journal, rollback,30 subprocess crashes PASS |
| Physical gateway regression | 1 dispatch live berizin;0 dispatch baru akibat replay; unauthorized native choice tetap ditolak |
| V6 trained-bundle verification | Fresh evaluation/persistence/context/reference and spelling synaptic ablation PASS; tidak melatih ulang V6 |

Worktree lokal sempat menghasilkan `error obtaining VCS status` untuk build
stamping. Diagnostics itu disimpan. Clone normal pada SHA/tree yang sama
berhasil dengan build metadata vcs.modified=false dan SHA yang tepat; standard
CI juga berhasil. Ini tidak diperbaiki dengan menurunkan validasi neural/WAL.

Source freeze dan publication evidence commit dibedakan. Source CI di atas
adalah verifikasi kode yang dilatih; publication berikut hanya menambah laporan
dan bukti. Status HEAD publication harus dibaca dari run aktual, bukan ditebak.

Belum diuji: perangkat Android nyata dan pemadaman daya hardware. Crash tests
adalah exit proses pada filesystem managed dengan pemanggilan Sync nyata.
