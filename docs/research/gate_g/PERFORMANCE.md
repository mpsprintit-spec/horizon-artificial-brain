# Gate G — hasil performa konfirmatori14 November2026

**G8 gagal:81/90 perbandingan memenuhi anggaran,9 gagal.** Tidak ada perubahan ambang setelah hasil. Source lokal9cca3e6 / published45b13f2, tree99c16485; runtime codeb5b70fa /af1217f. Baseline1e12332 mencakup PR60/65 yang sudah merged.

Semua25 command terdaftar selesai dengan status yang diharapkan. Satu command persistence kehilangan managed session sebelum hasil akhir, diarsipkan sebagai interrupted lalu diulang utuh. Tidak ada hasil lengkap yang dikeluarkan dari analisis. Go1.23.2, GOMAXPROCS1/GOGC100, ABBA berurutan; seluruh WAL/Sync dan validasi aktif.

## Seluruh kegagalan anggaran

| Kasus | Metrik | Baseline | Kandidat | Batas |
|---|---|---:|---:|---:|
| BenchmarkGateGThought/nodes=1000/dense=false/journal=false | thought_p95_ms | 0.78345 | 1.1645 | 1.00097 |
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.7431 | 0.99925 | 0.954565 |
| BenchmarkGateGThought/nodes=1000/dense=true/journal=false | thought_p95_ms | 9.9525 | 12.62 | 11.5454 |
| BenchmarkGateGThought/nodes=4000/dense=true/journal=true | thought_p95_ms | 35.08 | 41.76 | 40.442 |
| BenchmarkGateGNativeStages/nodes=4000/dense=false | learning_p95_ms | 0.094475 | 0.2182 | 0.21337 |
| BenchmarkGateGNativeStages/nodes=4000/dense=true | learning_p95_ms | 10.0575 | 12.3135 | 12.169 |
| BenchmarkNativeActionPersistence/nodes=4000/mode=checkpoint | mean ms/op | 66.2995 | 98.3243 | 79.6594 |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=true/journal=true | thought_p95_ms | 35.08 | 42.26 | 40.442 |
| BenchmarkGateGThoughtNativeIdle/nodes=40000/dense=false/journal=false | thought_p95_ms | 16.915 | 20.78 | 19.5522 |

## Hasil yang membaik dan biaya handoff

| Kasus | Baseline | Kandidat |
|---|---:|---:|
| 40k sparse+WAL p95 ms | 18.115 | 13.245 |
| 4k dense+WAL allocations/op | 41040.5 | 25028 |
| 4k dense+WAL B/op | 6.74662e+06 | 6.29804e+06 |
| 40k restore mean ms | 13196 | 11335.1 |

| Ukuran world | Sensor→native p50 ms | p95 ms | p99 ms |
|---|---:|---:|---:|
| 1000 | 0.4612 | 0.998 | 1.906 |
| 4000 | 1.38 | 2.62 | 5.553 |
| 40000 | 13.97 | 17.24 | 25.92 |

Admission p95=0.06µs;0B/op;0 allocations/op. Batas10µs/512B terpenuhi.

## Batas interpretasi dan akar biaya

Seluruh perbandingan alokasi dalam matriks memenuhi anggaran. P95 yang gagal tetap memblokir G8. Profil lama/frozen mengidentifikasi map copy/iteration, sorting, activity/retention traversal dan JSON; typed sorting serta empty-local specialization memperbaiki alokasi dengan384 transisi kanonik identik dan27 tie-order cases. Optimasi tersebut belum menyelesaikan semua pelanggaran tail latency.

Profil sparse40k lama mencatat CPU11,475s→11,216s, sedangkan dense4k8,762s→10,450s. Profil learning4k3,033s→3,062s. Ini biaya proses beserta setup/seeding; tidak boleh disamakan dengan p95 thought atau dipakai membatalkan hasil konfirmasi. Penyebab semua delta belum terisolasi.

Go TempDir memakai tmpfs, sehingga biaya WAL mencakup software dan Sync pada tmpfs. Append ABBA overlayfs terpisah menghasilkan p951,1455→1,1565ms dan2084B/op tetap; memenuhi budget tambahan. Ini bukan latency Android atau bukti power-loss durability.

Percentile adalah rerata per-run; bukan pooled observations atau interval keyakinan. Thought100 sampel/cell/run, native20 factual pairs/run (tail deskriptif), persistence3 operasi/cell/run (mean deskriptif). RSS adalah peak proses kumulatif; GC per-cell mencakup input di luar timer. Tidak ada klaim percepatan terhadap baseline sensor bypass yang belum memiliki kontrak koeksistensi aman.

## Reproduksi dan langkah konkret berikutnya

[Matriks lengkap](evidence/performance-repaired-confirmation-20261114/SUMMARY.md), summary.json, runs.json, raw25 log dan interrupted log dipertahankan. Konfirmasi awal yang gagal juga tetap tersedia di performance-final/.

Profil ulang kasus failure1k, native learning4k dan checkpoint4k pada dua source yang dipin; pisahkan CPU dan wall time serta biaya JSON/prefix validation. Perbaiki hanya bottleneck yang terukur, buktikan state/urutan/durability tetap identik, lalu jalankan konfirmasi penuh baru. Jangan mengurangi validasi, menghapus WAL atau mengubah kriteria agar hasil terlihat lulus.
