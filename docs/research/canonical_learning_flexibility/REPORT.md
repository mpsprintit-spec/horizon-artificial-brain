# Canonical learning flexibility — 14 November 2026

## Tujuan

Membuktikan beberapa kurikulum sederhana dapat dilanjutkan pada satu Brain
persisten, melalui mekanisme yang sudah tersedia dan versi kode yang sama.
Target lokal ditetapkan dalam [PROTOCOL.md](PROTOCOL.md). Peta API, parameter
yang dipelajari, konfigurasi insinyur, dan batas representasi ada di
[AUDIT.md](AUDIT.md).

Baseline autentik main: `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.
HEAD PR61/62/63/64/67 serta status draft/open/unmerged tercatat dalam protokol.
Branch ini bertumpu pada PR67 `e9ae2331c9c020f3d905a79cc0a303599ac83889`;
ancestry penelitian dan koreksi startup PR63 dipertahankan.

## Implementasi

Perbaikan produksi terbatas pada knowledge/native_relational.go,
native_control.go, dan native_control_validation.go:

1. Encoder opt-in `local-relational-v2` memperluas empat momen existing dengan
   empat momen bertanda. Inhibisi yang sudah ada dalam N kini dapat dibaca policy.
2. Inisialisasi dan validasi menggunakan jumlah fitur berdasarkan versi encoder.
   Semua bobot tetap berupa sinaps biasa dalam canonical N.
3. Rekonstruksi graph untuk validator checkpoint mempertahankan Inhibitory.
   Tanpa ini, uji pengembangan menemukan logged-value mismatch untuk V2.

Encoder lama tidak diubah maknanya. Model v1/v2/v3, container canonical7,
checkpoint20, native journal14 tetap didukung. Journal produksi3/composition4
tidak dinaikkan. Tidak ada migrasi otomatis atau konversi bobot policy lama.
V2 hanya tersedia melalui inisialisasi eksplisit; model dengan encoder V1 yang sudah ada tetap
memakai encoder lamanya. Migrasi learned policy encoder V1 ke V2 belum diimplementasikan.

Tidak ada API belajar baru: reuse GroundVectorAt, ProcessSensorObservation,
LearnExperience, CognitiveProcess/Think, ProcessObservation, dan
ApplyNativeControl. Uji mixed-pending menggabungkan teks/inquiry/native;
kurikulum menggabungkan sensor numerik/pengalaman/operasi/outcome/thought.
Admission episode dan perlindungan alamat policy dari PR67 dipakai kembali.
Controller tetap opt-in; main dan seluruh checkpoint produksi tidak diubah.

Source freeze: `daf040ae5330d6f8949b36c9aacb1293d596943c`, tree
`fc071fa258ee62d8287d21e43bc034a63a7dcdce`. Semua 795 file Go/data yang dicatat
dalam manifest tetap byte-identik selama dan setelah konfirmasi A–D.
Harness/data ada di runtime/continual_flexibility_test.go dan
runtime/testdata/continual_flexibility_v1.json. Tidak ada source patch di antara
kurikulum. Fixture, feedback meter, dan probe berada dalam lingkungan pengujian.

## Pengujian

### Garis keturunan dan parameter

Satu Brain latihan dimulai dari checkpoint kosong yang valid. Sembilan input
numerik Walsh membentuk populasi terdistribusi; satu paket sensor produksi
masuk sebelum A. Brain utama tidak diganti sepanjang A–D. Pointer Brain yang
dimiliki runtime, activation engine dan LearningUnit diperiksa identik pada
setiap checkpoint. Evaluasi, ablasi dan rekonstruksi recovery memakai salinan
yang tidak pernah menggantikan garis keturunan utama.

| Titik | Sequence | Neuron | Sinaps | PatternSynapse kanonik | Bobot sinaps berubah* | Update native kumulatif |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Foundation | 10 | 40 | 0 | 0 | 0 | 0 |
| A: urutan | 26 | 40 | 12 | 8 | 12 | 0 |
| B: konteks | 42 | 40 | 20 | 16 | 20 | 0 |
| Native opt-in | 43 | 57 | 92 | 16 | 72 | 0 |
| C: konsekuensi | 10795 | 57 | 92 | 16 | 40 | 1536 |
| D: penggunaan ulang | 10811 | 57 | 94 | 24 | 22 | 1536 |
| Restart + continuation | 10817 | 57 | 94 | 24 | 22 | 1536 |

\* Perbandingan bobot bertanda terhadap checkpoint sebelumnya, termasuk sinaps
baru dan perubahan waktu yang kecil. Native opt-in adalah alokasi parameter awal,
bukan hasil belajar. Saat C tepat 40 sinaps policy berubah, tidak ada bobot world
yang berubah atau store baru. L1 perubahan bobot policy C = 4.728322024458442;
delta maksimum = 0.17304040741509016. D menambah dua sinaps world; bobot policy
tetap. Rincian lengkap ada dalam synaptic-deltas.json di paket bukti.

### Hasil latihan dan retensi

| Target lokal | Sebelum | Setelah | Setelah D/restart | Keputusan |
| --- | --- | --- | --- | --- |
| A, margin prediksi target–pesaing, minimum 8 kasus | 0 | 1.390366 | 1.377930 | Lulus >= .1 dan meningkat |
| B, konteks berbeda/cue sama, minimum 8 kasus | 0 | 1.305261 | 1.355519 | Lulus; A tetap bertahan |
| C, pilihan greedy optimal, 6 kondisi baru | 3/6 | 6/6 | 6/6 | Lulus >= 5/6 |
| C, probabilitas operasi optimal menurut epsilon-policy | .50 | .95 | .95 | Lulus kenaikan >= .1 |
| C, MAE nilai advantage dari dua eksekusi faktual | .762713 | .363488 | .363485 | Turun 52.34%, lulus >= 30% |
| D, dua prediksi dengan intermediate yang diamati, minimum 8 pengukuran | 1.305219 sebelum D | 1.305220 | 1.305220 | Reuse sudah ada setelah B/C; D tidak diklaim menciptakannya dari nol |

Margin adalah selisih aktivitas numerik, bukan probabilitas. Probabilitas .95
berasal dari epsilon .1, bukan klaim kalibrasi keyakinan. C menggunakan 1536
pasangan / 3072 trajectory yang benar-benar dijalankan. Reward dihitung dari
aktivitas outcome setelah operasi, lalu biaya komputasi diperhitungkan learner.
Tidak ada reward dari trajectory hipotetis.

A/B memiliki kasus baru berupa prefix netral dan permutasi anggota populasi;
C diuji pada amplitudo .4/.6/.8 yang tidak dipakai latihan. C tetap merupakan
keluarga kondisi kecil yang dilatih. Pada D, simulator memberikan intermediate
sebagai pengalaman nyata, tanpa memilihnya dari jawaban prediksi Horizon.
Kurikulum gabungan menambah pengalaman; tidak ada pipeline komposisi produksi baru.

Ablasi PatternSynapse pada salinan Brain menurunkan seluruh margin A/B menjadi
sekitar +/-1e-9. Ablasi sinaps policy kanonik menurunkan C ke 3/6. Ini memisahkan
perubahan karena pembelajaran neural dari jawaban yang ditulis dalam evaluator.

### Persistence, recovery, keselamatan

Satu journal berisi **10818 record / 10817 sequence**; satu record tambahan adalah
sensor intent yang dipasangkan dengan commit. Jenis record: 9 vector_ground,
1 sensor_intent, 1 sensor_commit, 48 learn, 10757 native_control_command,
1 cognitive_process, 1 think. Full journal, semua checkpoint, hasil probe,
manifest source/lineage, dan kontrol disimpan bersama.

| Pemeriksaan | Hasil dan batas |
| --- | --- |
| Genesis replay | 7/7 checkpoint cocok byte-exact untuk complete continuation |
| Checkpoint + suffix | 7/7 cocok; checksum checkpoint recovered sama |
| Restart lalu observation/thought/native/outcome | State live dan restored identik; hasil belajar tetap ada |
| Subprocess crash | 15/15 boundary V1 dan 15/15 V2, child exit 86; state durable lengkap cocok |
| Pending factual pair / outcome ID | Pending bertahan; outcome kedua belajar sekali; exact retry idempotent, reward berubah ditolak |
| Policy-role input | Input LearnExperience yang menunjuk semua fitur V2 ditolak sebelum append/mutasi; fence Gate G lainnya lolos |
| Physical boundary | Gateway counting mock: 1 dispatch live yang berizin, **0 dispatch baru saat genesis/suffix recovery**, termasuk probe encoder V2 dengan race detector |
| Legacy isolation | Regresi dependency boundary dan jalur recurrent core/production lolos; tidak ada engine legacy baru |
| Pembaca lama aktual | Binary source PR67 e9ae233 menolak checkpoint V2 tanpa mutasi dan menolak suffix V2 pada record 44 |
| Rollback | Binary lama memulihkan checkpoint B sequence42 + salinan prefix 43 record; complete continuation identik dengan checkpoint B |

Probe compatibility/physical tambahan dikompilasi pada worktree source yang
dicatat; sumber probe dan log disimpan dalam evidence/compatibility. Seluruh
15 boundary crash memakai harness Gate G yang sama. Ini crash proses pada
overlayfs dengan Sync nyata, bukan simulasi pemadaman daya perangkat.
Rollback menggunakan salinan prefix bernama eksplisit; journal penuh tetap ada.

### Hasil negatif dan diagnosis

| Hasil | Penyebab / klasifikasi | Tindakan dan keputusan |
| --- | --- | --- |
| Input one-hot 16 dim berbagi anchor; dua kondisi C memiliki readout identik | Kalibrasi kurikulum + representasi: jarak L1 ternormalisasi hanya .0625; pooling policy kehilangan identitas sel terisolasi | Data pengembangan dikalibrasi menjadi Walsh +/-1 berjarak .5. Kasus one-hot, journal dan kegagalannya dipertahankan sebagai batas nyata, tidak dianggap terselesaikan |
| 48 ronde: choice lolos tetapi fit advantage hanya turun sekitar 24.5% | Kurikulum belum cukup untuk target fit lokal | Tambah pengalaman menjadi 192 ronde sebelum freeze/konfirmasi; aturan belajar dan ambang tidak diubah |
| Restart-begin tanpa aktivitas setelah trial mengosongkan state | Kesalahan asumsi harness; admission benar menolak episode kosong | Continuation memakai observasi baru melalui boundary existing; tidak melonggarkan admission |
| D tanpa intermediate yang diamati tetap mendekati nol, juga sesudah D | Batas kurikulum/representasi recurrent-context existing; prediksi tidak diubah menjadi fakta | Hasil eksploratori gagal, disimpan; tidak diklaim sebagai komposisi internal mandiri |
| Eligibility/history berbeda dengan graph/activity sama tetap collision | Representasi policy | Test negatif eksplisit; V2 hanya memperbaiki tanda sinaps |
| Graph/context berbeda tetapi momen lokal sama masih collision | Representasi policy | Tidak ada klaim bahwa delapan momen membedakan semua struktur atau peran |

Fase development, validation dan confirmation dibedakan dalam paket bukti.
Perubahan representasi tidak dipicu sekadar rendahnya skor: sign collision
dibuktikan oleh kasus tandingan Phase III, yang tetap dipertahankan.

### Biaya dan verifikasi engineering

Hasil benchmark encoder dan status CI dicatat dalam [VERIFICATION.md](VERIFICATION.md).
Benchmark ini membandingkan dua basis encoder pada runtime yang sama, bukan
mengulang evaluasi G8. **Status G8 PR67 tetap gagal: 81/90.** Tidak ada promosi
controller menjadi default atau klaim performa Android on-device.

## Keputusan

**Engineering: diterima untuk cakupan perbaikan opt-in ini**, berdasarkan
regresi versi, input ownership, replay/checkpoint, crash dan physical safety.
**Training: A/B/C/D memenuhi target lokal protokol pada satu garis keturunan.**
**Generalisasi: terbukti hanya pada variasi yang dicatat**, termasuk amplitudo
interpolasi dan prefix/permutasi; missing-intermediate reuse tetap gagal.
Hasil ini menguji fleksibilitas infrastructure, bukan mengesahkan aktivasi default.

PR penelitian: [#68](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/68),
draft, bertumpuk di atas PR67, tidak di-merge ke main. Subgate Gate G existing
G1–G7/G9/G10 tetap memiliki regresi yang lolos; G8 tetap penghalang promosi.

## Langkah berikutnya

Pertahankan checkpoint `05-restart-continuation.json` dan full journal dalam paket
bukti sebagai garis keturunan latihan untuk kurikulum berikutnya. Gunakan data
pengalaman baru melalui API existing. Jika tugas berikut membutuhkan temporal
eligibility atau membedakan graph dengan momen sama, buat diagnosis observabilitas
yang terpisah sebelum mengubah encoder. Migrasi policy lama dan perbaikan G8
tetap pekerjaan terpisah; jangan menghapus parameter lama demi memudahkan upgrade.

Paket bukti: [Canonical_Learning_Flexibility_Evidence_20261114.tar.xz](evidence/Canonical_Learning_Flexibility_Evidence_20261114.tar.xz), 196 payload, 43,448,996 byte sebelum kompresi; SHA256 `b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0`. Setiap payload diverifikasi terhadap manifest setelah ekstraksi.
