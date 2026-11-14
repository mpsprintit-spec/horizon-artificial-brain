# Final native integration — keputusan engineering

**Status: BLOCKED; belum diintegrasikan ke main.** CI dan regresi fungsional lulus, tetapi G8 terakhir86/90 dan supplementary append belum memenuhi anggaran. Tidak ada merge, force push, aktivasi controller default, atau penggantian Brain produksi.

## Mekanisme kandidat dan perubahan minimal

Dependency closure PR61/62/64/67/68/69 serta koreksi PR63 terbaru ada di branch `integration/native-engineering-closure-20261114`, PR70. Seluruh ancestor dipertahankan. Main tetap `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.

| Unit | Status aktual |
|---|---|
| Native operations, pemilihan melalui sinaps N, factual action contrast/causal credit | Implementasi existing dipertahankan dalam kandidat; regresi lulus; belum main |
| Signed relational encoderV2, policy ownership/admission | Versi/config/parameter N existing dipertahankan; input alias dan mixed-episode tests lulus |
| Mixed journal, checkpoint20/canonical7/WAL14, deterministic replay/recovery | Format dan law tidak diubah; genesis/suffix/continuation lulus |
| Startup PR63 cee28ff | READY secara independen dan kompatibel dengan closure; tetap belum merge |
| Promosi closure ke main | BLOCKED oleh latency G8 dan append overlayfs |

Perbaikan produksi cab120a: alokasi map propagasi diperkecil, input read-only tidak disalin dua kali, lookup registry kedua diganti alamat transient di bawah Brain lock, outbound scratch slice dipakai ulang, command JSON tervalidasi dipakai langsung untuk WAL, dan compact JSON array menggunakan buffer Encoder per-array. Byte encoding/checksum dan seluruh fsync/fence tetap dipertahankan.

Perbaikan cfeac007: cabang waktu pembentukan tak diketahui tetap menghasilkan no-decay yang sama; interval watermark yang identik dihitung ulang hanya bila perlu dalam satu settlement. Dua scalar arithmetic transient tidak menyimpan parameter neural. Tidak ada perubahan learning law atau encoder.

Kesetaraan diuji dengan216 transisi advance (termasuk IEEE754),720 transisi physical-memory,384 general-activity transitions dan27 tie-order cases. Dokumentasi lab dan artefak tetap ada; executable/package akuisisi Phase I–III tidak dimasukkan ke kandidat produksi, sebagaimana RESEARCH_EXCLUSIONS.json.

## Hasil Gate G8

| Source | Matriks lengkap | Keputusan |
|---|---:|---|
| Gate G terdahulu45b13f2/e9ae233 |81/90 | Gagal, dipertahankan |
| Perbaikan pertama1069c02 |89/90 | Gagal; seluruh9 failure lama lulus pada run ini, tetapi sparse4k/no-WAL gagal |
| Perbaikan kedua cfeac007 |86/90 | Gagal; ini hasil source terbaru, bukan diganti hasil terbaik |

Semua25 command per matriks selesai dengan expected exit yang benar. Refusal baseline memakai guard asli. Go1.23.2, GOMAXPROCS1/GOGC100, sequential ABBA, harness SHA256 f29039161c8e93d0f62c9e025a0473e3821399187daed5471dbb38629e6d6b48; fixture, sampel dan threshold asli tetap. Tidak ada run selesai dibuang.

| Kegagalan terbaru | Metrik | Baseline ms | Kandidat ms | Batas ms |
|---|---|---:|---:|---:|
| BenchmarkGateGThought/nodes=1000/dense=false/journal=true | thought_p95_ms | 0.6324 | 1.79555 | 0.82726 |
| BenchmarkGateGNativeStages/nodes=1000/dense=false | learning_p95_ms | 0.05769 | 0.187925 | 0.169228 |
| BenchmarkGateGNativeStages/nodes=40000/dense=true | learning_p95_ms | 78.64 | 100.79 | 94.468 |
| BenchmarkGateGThoughtNativeIdle/nodes=4000/dense=false/journal=true | thought_p95_ms | 1.4115 | 2.013 | 1.72323 |

Supplementary append overlayfs juga gagal: rerata per-run p95 baseline.43985ms, kandidat2.75295ms, batas.62782ms; bytes/op lulus. B1 p954.964ms versus B2 .5419ms tetap disimpan. Ini hasil gagal meskipun median B1 .2972ms dan B2 .2503ms.

Pada1069c02, satu failure sparse4k mempunyai median.8142/.8181ms, p953.888/1.128ms dan nol cell GC. Profil pengembangan cfeac007 tidak membuktikan perbaikan tail yang stabil. Jangan menyebut scheduler/host sebagai penyebab tunggal tanpa bukti. CPU profile memuat loading fixture; biaya itu dibedakan dari thought. Pembacaan CPU pressure/cgroup tidak tersedia di lingkungan ini (files kosong pada resource-context record).

Alokasi tetap membaik, misalnya development dense1k sekitar6519→256 alloc/op dan checkpoint4k16.3→8.1MB/op. Ini bukan pengganti empat latency failures terbaru atau bukti all-gates pass. [Matriks lengkap terbaru](evidence/cfeac007-SUMMARY.md) dan [89/90 yang dipertahankan](evidence/1069c02-SUMMARY.md) memuat seluruh90 perbandingan beserta p50/p95/p99, bytes, GC dan RSS.

## CI, kompatibilitas dan keselamatan

- Source freeze `cfeac007dec4c5d1fe86cc885a4ce12a07c9e144`, tree `8292bd089ac5d4742d0320bd9873bfd65035453c`. CI43620396195 SUCCESS (validate dan monitor), termasuk full Go tests, vet/build, nested modules, Bridge dan saved-state verification.
- Full root race coverage67 paket:7 paket CI ditambah60 paket lokal, tanpa menghapus assertion mekanisme. Saved A–E1 test juga dijalankan tersendiri dengan race detector (PASS153.72s).
- Standard vet/build lokal PASS. CLI, chat dan production Android arm64 CGO0 cross-build PASS, build metadata source SHA benar dan vcs.modified=false. Perangkat Android nyata dan hardware power-loss belum diuji.
- Gate G mixed/admission/ownership/legacy serta30 subprocess crash cases per encoder-pair lulus pada source terakhir. Tidak ada error state yang dipromosikan menjadi sukses.
- Gateway V1 dan probe V2 dengan race:1 authorized live dispatch,0 dispatch baru saat genesis/suffix recovery. Primitive native tidak memberi otorisasi perangkat.
- Binary aktual PR67 e9ae233 menolak V2 sebelum memutasi Brain; rollback checkpoint B+prefix cocok sequence42 dan complete continuation; full incompatible suffix ditolak. Prefix rollback adalah salinan bernama eksplisit, journal penuh tetap utuh.

## Pemulihan Brain A–E1

Tidak ada akuisisi ulang pada lineage utama, reset model, migrasi diam-diam atau transplantasi dari V6/Brain lain. Evaluasi memakai salinan disposable; checkpoint/journal sumber tetap byte-identik.

- Identity `horizon-primary-brain`, sequence75331, native updates10752,57 neuron/94 sinaps/24 PatternSynapse.
- Complete genesis replay, checkpoint+suffix, checkpoint roundtrip, checksum lama dan next-thought continuation cocok persis dengan oracle PR69 a3ce74c. Ownership activation/learning/DNF tetap satu pointer Brain.
- Semua nilai probe A/B/C/D dan36 E1 holdout probes cocok persis dengan oracle; C6/6, E1 35/36. Ini retensi, bukan training baru.
- Checkpoint `e1-main-continuation/07-E1-restarted.json`, journal `e1-main-continuation/events.jsonl` dalam arsip PR69 yang sama.
- Checksum `68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82`; checkpoint file SHA256 e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537; journal SHA256 ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede.
- Arsip asli: `docs/research/canonical_integration_20261114/evidence/Horizon_Canonical_Continuation_E1_20261114.tar.xz`, SHA256 7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3.

## Commit, PR dan bukti

- PR70: https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/70 (draft, base main; tidak merge).
-93a3ac6 mempertahankan merge63; cab120a perbaikan alokasi/encoding;1069c02 saved-state verification/production selection; cfeac007 physical-time arithmetic repair. Semua published, tanpa force push.
- CI: https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43620396195 . CI source pertama43617607078 juga sukses tetapi G8 tetap gagal.
- Laporan/evidence disimpan terpisah dari source freeze pada branch `research/native-integration-closure-evidence-20261114`; PR70 tetap menunjuk source yang benar-benar diuji.

Raw evidence: [archive](evidence/Native_Integration_Closure_Evidence_20261114.tar.xz),343 payload manifest-verified, SHA256 `fb9c011a5b3131bbc3c799f9dcda641cea6a2d108970743e0ce1257167954bd9`. Raw logs, seluruh failures, profiles dan crash states dipertahankan. Salinan model/journal yang sudah ada di arsip asal serta executable yang dapat dibangun ulang tidak diduplikasi; hash/alasannya dicatat pada EXCLUDED_REPRODUCIBLE.json.

## Status akhir main dan langkah konkret tersisa

**Main tetap3a9e7ce; tidak ada mekanisme native baru yang diintegrasikan ke main pada pekerjaan ini.** Implementasi dan regresi yang diperbaiki tersedia di PR70, dengan kompatibilitas serta learned state tetap terjaga.

Blocker adalah latency engineering, bukan kekurangan latihan A–E1. Langkah berikutnya: isolasi waktu CPU/digest/graph-read dan write/stat/Sync pada empat sel gagal serta append overlayfs; usulkan patch hanya bila sebab terukur, ulangi differential canonical-state/crash checks, lalu source freeze dan konfirmasi penuh90 dengan batas asli. Jangan menganggap86/90, CI hijau, atau keberhasilan retensi sebagai izin merge. Controller tetap opt-in.
