# Benchmark fondasi native — 6 November 2026

**Target ≥3× p95 pada workload sparse besar belum tercapai.** Pada synthetic
40.000 neuron / 160.000 sinaps, whole-thought p95 tanpa jurnal membaik **2,46×**,
dengan jurnal **2,84×**. Alokasi turun besar tanpa perubahan canonical bytes
pada uji ekuivalensi. Ini tidak menyatakan semua gate kognitif lulus.

## Metode

Baseline `d7cb140b5996f1bddffeebbda693ebadfd714841`; kandidat konservatif
`87fc532c3d14fb0e9b7ee37523ad308aba503de3`, tree
`6d7bfa05f3bec5e60489eeb3ab3dc4e58773649c`. Linux amd64, AMD EPYC 9V74,
CPU quota 8 core, Go 1.23.2, `-test.cpu=8`. Mesin virtual bersama; bukan
Android atau CPU yang diisolasi/fixed-frequency.

Harness identik `foundation_benchmark_test.go` dicopy ke baseline dan kandidat,
dibangun sebelum pengukuran. Urutan **baseline → kandidat → kandidat → baseline**;
100 pasangan sensor+thought/cell, 3 warmup. Tidak ada training/regression suite
bersamaan. Setup di luar timer. Tabel adalah **rata-rata quantile per-run**,
bukan quantile gabungan atau CI. p99 dengan 100 sampel sensitif terhadap outlier.

Fixture: 1k/4k/40k neuron, fanout4. Sensor asli merekrut populasi terpisah;
sekitar 9 neuron aktif setelah warmup. Ini bukan jaringan 40k neuron terlatih
pengguna. Baseline/kandidat menjalankan hukum yang sama. Kernel eksperimen baru
dilaporkan terpisah.

## Whole runtime

| Neuron | WAL | Thought p50 baseline→kandidat (ms) | p95 (ms) | p99 (ms) | Percepatan p95 |
|---:|:---:|---:|---:|---:|---:|
| 1.000 | tidak | .824 → .278 | 1.859 → .609 | 3.028 → 2.779 | 3,06× |
| 1.000 | ya | 1.176 → .553 | 2.436 → 1.327 | 4.658 → 2.069 | 1,84× |
| 4.000 | tidak | 3.420 → .917 | 7.773 → 1.579 | 10.220 → 3.803 | 4,92× |
| 4.000 | ya | 3.746 → 1.242 | 6.918 → 2.342 | 10.623 → 3.224 | 2,95× |
| 40.000 | tidak | 45.245 → 17.595 | **58.830 → 23.945** | 65.635 → 25.705 | **2,46×** |
| 40.000 | ya | 47.410 → 17.365 | **68.890 → 24.235** | 92.410 → 27.660 | **2,84×** |

| Neuron, tanpa WAL | B/pasangan baseline→kandidat | Pasangan/detik baseline→kandidat |
|---:|---:|---:|
| 1.000 | 362.605 → 151.586 | 526,7 → 1.172,9 |
| 4.000 | 1.222.687 → 385.032 | 135,6 → 477,2 |
| 40.000 | 12.188.230 → **3.834.736** | **11,44 → 28,28** |

Pada 40k tanpa WAL: penurunan bytes 68,54%; allocation count 561.270,5→1.109,5
(**99,80%**). GC 22→6,5 per 100 pasangan, total GC pause 1,640→.451 ms. Peak
RSS proses seluruh matrix baseline 304–355 MiB; kandidat 290–300 MiB. RSS
mencakup fixture, loader dan Go heap yang dipertahankan, bukan satu command.

WAL menggunakan durable path aktual. Pada 40k mean pasangan kandidat tanpa
WAL 35,36 ms versus dengan WAL 36,40 ms: overhead sekitar 1,04 ms/pasangan;
throughput 28,28 versus 27,47 pasangan/s. Fsync/jitter storage dapat mendominasi
jaringan kecil. Ini bukan model latency storage universal.

## Regresi yang ditemukan dan diperbaiki

Kandidat pertama `bf355196` langsung mengiterasi map untuk menghapus sorting.
Alokasinya rendah, tetapi p95 40k menjadi **97,52–103,90 ms**, dibanding
baseline batch sama 55,25–65,50 ms. Kandidat itu **tidak di-merge**.

CPU profile kandidat gagal: activity callback cumulative 43,98%, memory decay
33,41%; map iterator init/next memakan 27,28% flat CPU. Koreksi memakai indeks
**alamat** canonical berurutan, dibangun ulang saat cardinality append-only
berubah. Indeks tidak menyimpan bobot; reload membangun registry baru.

Pertumbuhan registry, concurrent access, isolasi salinan `Nodes`, restore,
dan canonical-byte parity diuji. Fixture concurrency pertama terlalu mirip
secara angular sehingga registry memang menggabungkannya; fixture dikoreksi
ke vektor yang terpisah. Itu asumsi test yang salah, bukan neuron hilang oleh
cache.

## Sparse, dense, dan lazy-time kernel

50 sampel/cell. P95 dalam ms. Eager memakai local activity law kandidat;
lazy adalah utility observed-only eksperimen; operation adalah propagasi baru,
**bukan ekuivalen seluruh legacy thought**.

| Neuron | Aktivitas | Eager p95 | Lazy p95 | Native propagate p95 |
|---:|---:|---:|---:|---:|
| 1.000 | .1% | .2539 | .000290 | .009063 |
| 1.000 | 10% | .4174 | .04489 | .2081 |
| 1.000 | 100% | .5103 | .4695 | 1.804 |
| 4.000 | .1% | .7390 | .000931 | .01429 |
| 4.000 | 10% | .8571 | .1521 | .6049 |
| 4.000 | 100% | 1.481 | 1.476 | 16.57 |
| 40.000 | .1% | 11.27 | .06363 | .06723 |
| 40.000 | 10% | 11.73 | 2.597 | 6.502 |
| 40.000 | 100% | 20.47 | **23.31** | 82.76 |

Pada 40k/.1% terdapat 40 neuron aktif. Eager membaca/mematerialisasi eligibility
160k sinaps; lazy membaca 160 edge dan hanya memutasi edge coactive. Pada 10%,
lazy membaca 16k edge dan melakukan 15.990 update coactive. Dense lazy **13,9%
lebih lambat** di p95 daripada eager. Cabang dense menghindari lookup sumber
berulang tetapi tidak menjamin kemenangan. Utility belum menjadi default.

Nilai efektif lazy eligibility dibandingkan dengan eager sampai toleransi
2e-15, termasuk excitatory/inhibitory dan interval inactive. Timestamp fisik
berbeda; klaim byte-identik hanya untuk optimasi konservatif. Integrasi lazy
global masih memerlukan audit semua consumer dan migrasi eksplisit.

Operation dense 40k mengalokasikan ~13,45 MB dan 121.270 objek per iterasi,
p50/p95/p99 69,15/82,76/92,76 ms. Alokasi benchmark mencakup reset state;
quantile hanya mengukur eksekusi operasi. Cloning dan adjacency sorting masih
mahal. Plan memakai state/confidence map equality, menghindari dua hashing JSON
besar sambil menolak plan stale/foreign/reused. Build indeks pertama ikut
teramortisasi pada beberapa baris, misalnya 6.554 B/op eager dense.

## Episode policy lengkap

Begin + dua propagasi + stop + observed outcome/learning, termasuk readout,
provenance, dedup, state copies dan WAL opsional. Ada 64 source aktif. Trajectory
disamakan melalui inverse-CDF sampling; ini bukan evaluasi mutu belajar.

| Neuron dunia | WAL | Episode p50/p95/p99 (ms) | B/episode | Alloc/episode |
|---:|:---:|---:|---:|---:|
| 1.000 | tidak | 2.301 / 4.012 / 5.062 | 521.311 | 6.501 |
| 1.000 | ya | 4.647 / 6.580 / 7.958 | 535.301 | 6.674 |
| 4.000 | tidak | 2.344 / 5.194 / 7.628 | 520.229 | 6.497 |
| 4.000 | ya | 5.224 / 12.730 / 13.660 | 534.181 | 6.670 |
| 40.000 | tidak | 2.015 / 2.340 / 2.478 | 518.593 | 6.493 |
| 40.000 | ya | 4.141 / 5.016 / 7.119 | 531.903 | 6.666 |

524 transition-synapse reads/episode, **di luar** tambahan read Q dan validasi.
Peak RSS matrix 317 MiB. 40k bukan otomatis lebih cepat: heap besar menunda GC;
tidak ada GC pada jendela 100 episode 40k, sementara 1k mengalami 17–18 GC.
~0,52 MB/episode tetap tekanan alokasi nyata. Ini belum termasuk pembelajaran
representasi sensor, aksi fisik atau global retention legacy. Jangan menyebut
2,34 ms ini speedup whole-thought yang setara.

## Checkpoint, restart, dan replay

Lima pengukuran/ukuran; mean, bukan p95. Writer aktual; replay memuat checkpoint
lalu 100 thought command. Aktivitas fixture replay kosong: ini biaya recovery
serta global maintenance, bukan tugas kognitif terlatih.

| Neuron | Checkpoint (ms) | Restore (ms) | Restore + replay100 (ms) |
|---:|---:|---:|---:|
| 1.000 | 16,46 | 133,48 | 269,26 |
| 4.000 | 52,03 | 516,18 | 831,03 |
| 40.000 | 558,45 | **4.836,81** | **7.938,86** |

40k checkpoint mengalokasikan total ~159,5 MB/op; restore ~692,7 MB;
replay100 ~1.255 MB. Ini cumulative allocation, **bukan semuanya live sekaligus**.
Peak RSS matrix persistence ~842 MiB. Decoding/validation JSON dan rekonstruksi
masih membatasi skala; belum ada bukti layak jutaan neuron di HP.

## CPU, alokasi, dan contention

Profiling terpisah, 200 pasangan 40k. Profile mencakup setup fixture, walaupun
benchmark timer tidak. CPU kandidat: activity cumulative 25,07%, memory decay
20,82%, JSON validation 18,44%. Allocation-space profile mencatat curiosity
drive ~650 MB (34,35%) dan banyak biaya loader/JSON setup. Jangan mengatribusikan
semua profile kepada steady-state thought.

Mutex profile satu writer menunjukkan ~6,5 ms runtime internal contention.
Block profile didominasi channel wait testing/harness. Ini **bukan bukti nol
contention** pada banyak sensor simultan. Race/read-concurrency correctness
diuji terpisah; latency concurrent-load SLO belum dibuktikan.

Bottleneck tersisa: scan activity/retention global meskipun sparse, curiosity
allocation, cloning state/episode dan parsing checkpoint. Menghilangkan scan
memerlukan lazy-time/active index konsisten di seluruh consumer. Perubahan
tersebut belum dipaksakan hanya agar target 3× tercapai.

## Reproduksi dan evidence

Bangun test binary dari baseline/kandidat dengan benchmark yang sama:

```sh
go test -c -o evidence/baseline-runtime.test ./services/ai/runtime
# Jalankan pada baseline, lalu ulangi pada kandidat sebagai perf-runtime.test.
go test -c -o evidence/native-runtime.test ./services/ai/runtime
python3 scripts/measure_native_foundation.py evidence
go test ./services/ai/runtime -run '^$' -bench '^BenchmarkNativeController$' -benchtime=100x -cpu=8 -benchmem
go tool pprof -top evidence/perf-profile.cpu
```

[performance.json](performance.json) berisi agregat lengkap;
[sparse_performance.json](sparse_performance.json) berisi seluruh kernel cells.
Arsip evidence memuat raw ABBA, RSS/CPU, CPU/heap/mutex/block profiles, kandidat
gagal awal, hasil koreksi dan test logs. Run di luar `stable-verified/` adalah
pengembangan/kandidat ditolak, bukan hasil performa final.
