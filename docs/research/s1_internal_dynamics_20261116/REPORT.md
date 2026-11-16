# HORIZON — S1 Internal Dynamics Prediction

Tanggal: 16 November 2026. Baseline main: `ae26b2ea39e7bae2c7f3d5b2660e5063168780bb`.
Branch: `research/s1-internal-dynamics-20261116`. Brain: `horizon-primary-brain`.

**Keputusan: PARTIAL — perlu perbaikan pada pengukuran retensi formal.**
Pembelajaran prediksi neural, generalisasi, atribusi sinaptik dan pemulihan
terbukti dalam cakupan di bawah. Batas retensi awal tidak lulus; hasil negatif
itu dipertahankan. Kontrol tambahan dengan waktu oracle yang setara menunjukkan
hasil A–E1 persis sama dengan setelah S1. Tidak ada pelatihan S2, promosi state
produksi, atau merge ke main.

## 1. Tujuan dan cakupan

Melanjutkan A–E1 untuk memprediksi aktivitas internal berikutnya sebelum
operasi dijalankan. Target kontinu adalah rata-rata aktivitas pada tiga
populasi empat neuron asli: gate (populasi 5) dan dua populasi tujuan (6, 7).
Nilainya dibagi dua sehingga tetap dalam rentang codec; RMSE dan toleransi
laporan memakai satuan normalisasi ini. Prediksi delta diperoleh dengan
mengurangi keadaan awal yang telah diamati; error delta sama dengan error
keadaan berikutnya. Ini bukan rekonstruksi seluruh state plastis Brain.

Operasi yang dipakai adalah **propagate(gain=1)** dan **stop**, yang memang
sudah terdaftar pada kebijakan kanonik. Focus/relax tidak ditambahkan. Pilihan
operasi diarahkan adapter penelitian untuk mengumpulkan pengalaman; belum ada
klaim bahwa Brain secara mandiri meminta atau menggunakan prediksi ini.

## 2. Audit kesiapan dan perbaikan minimal

| Mekanisme | Bukti penggunaan dan batas |
| --- | --- |
| Satu N | Activation, learning dan DNF menunjuk pemilik Brain yang sama. Semua unit baru masuk registry kanonik yang sama. |
| Aktivasi dan recurrent dynamics | Pasangan sebelum/sesudah berasal dari `ApplyNativeControl`, termasuk operasi propagate sebelumnya; bukan simulasi F di adapter. |
| Inspeksi internal | `CurrentState` dan snapshot dipakai sebagai sensor numerik; codec hanya mengemas pengukuran yang sudah tersedia sebelum operasi. |
| Pembelajaran prediksi | `LearnCompositionWithDiagnosis` → `learnNMetric` yang sudah ada. Arah, produk fitur, resolusi, bias dan readout tersimpan pada pasangan sinaps excitatory/inhibitory di N. |
| Plastisitas/eligibility | Penulisan parameter native memperbarui `Dynamic.Weight`, eligibility, confidence, frequency dan waktu modifikasi. Candidate fitting/error diagnosis adalah kode native yang sudah ada, bukan model eksternal. |
| Kebijakan operasi dan causal credit | Tetap tersedia dan dipertahankan. Override evaluasi serta outcome `Learn=false` menutup trajectory tanpa melatih kebijakan. S1 tidak diklaim memakai TD/REINFORCE kebijakan untuk fit numerik. |
| Persistence | Checkpoint schema20, WAL, validasi prefix, checksum dan replay digunakan tanpa perubahan hukumnya. |

Hambatan awal: admission menolak registrasi numerik pada Brain native, dan
backup migrasi mencoba downgrade schema. Perbaikan hanya membuka registrasi
**proyeksi kosong** pada native idle dan kontrak numerik v6, serta mempertahankan
schema20 untuk dua backup. Migrasi nonkosong, versi numerik lain, dan transaksi
native aktif tetap ditolak sebelum append/mutasi. Uji penolakan, coexistence,
replay dan race tersedia. Tidak ada perubahan pada rumus aktivasi, operasi,
learning rate, hukum memori, atau aturan plastisitas.

API numerik memerlukan registrasi adapter dan bank generik. Registrasi kosong
ini tidak berisi teks, pemetaan bahasa, atau pelatihan bahasa. Numeric queries
adalah koordinat pengukuran. Witnesses dalam Brain adalah pengalaman terbatas;
readout tetap identik saat witnesses dihapus pada salinan evaluasi.

## 3. Lineage dan pengalaman

Sumber checkpoint dan journal asli diautentikasi dengan SHA-256 yang dipin oleh
`prepare_native_continuation.py`, lalu dipulihkan secara native. Awal:
sequence **75.331**, update kebijakan native **10.752**, neuron **57**, synaps
**94**, PatternSynapses **24**. Tidak ada reacquisition A–E1, transplant bobot,
atau penggantian lineage. Satu salinan penelitian berlanjut melalui observasi,
12 putaran pembelajaran, checkpoint, restart dan continuation.

Adapter mengemas 36 aktivitas neuron dunia lama, sembilan rata-rata keadaan
sebelumnya, operasi yang akan dijalankan, operasi sebelumnya, dan presence.
Nilai hasil berikutnya tidak menjadi input. Forecast dibuat dan dibuktikan
read-only sebelum operasi aktual. Keadaan awal menggunakan amplitudo kontinu
seragam dalam masing-masing populasi; cakupan belum mencakup variasi bebas
antarsemua neuron dalam satu populasi.

- Fit: 64 transisi, seed37; check pengembangan: 64 transisi, seed41.
- Kedua stream menjadi pengalaman pembelajaran; check berulang bukan holdout final.
- Anggaran tetap: 12 putaran × 3 query = **36 perintah pembelajaran**;
  **27 commit numerik nyata**, **26 fitur neural**. Pengalaman diulang tanpa
  dihitung sebagai bukti independen baru.
- Holdout final: 96 kasus seed83 dan 96 kasus seed151; amplitudo/kombinasi baru,
  termasuk konteks operasi stop sebelumnya. Tidak digunakan untuk memilih stage
  atau melatih ulang setelah hasil dilihat.
- Waktu observasi disimulasikan tetap pada timestamp checkpoint sumber.
  Hukum waktu native tetap berlaku; generalisasi terhadap umur sinaps yang
  jauh berbeda belum diuji.

Protocol dan amandemen audit sebelum pembelajaran tersimpan. Percobaan
kompatibilitas awal tidak melakukan S1 learning. Perbandingan JSON diformat
kemudian diperbaiki menjadi perbandingan nilai; aktivasi sesaat dibedakan dari
parameter pembelajaran. Ambang kemampuan tidak diubah setelah holdout.

## 4. Hasil kemampuan kontinu

Akurasi berarti proporsi koordinat dengan absolute error ≤0,05 pada nilai
normalisasi; bukan persentase prediksi seluruh Brain yang tepat sempurna.

| Kondisi | RMSE holdout | MAE | Akurasi dalam toleransi |
| --- | ---: | ---: | ---: |
| A–E1 sebelum S1 | 0.202894 | 0.176917 | 6.25% |
| Inisialisasi numerik tanpa S1 | 0.202894 | 0.176917 | 6.25% |
| Setelah S1 | **0.024033** | **0.018060** | **95.31%** |
| Ablation semua sinaps incident fitur S1 | 0.202894 | 0.176917 | 6.25% |
| Ablation readout saja; geometri dan bias dipertahankan | 0.174113 | 0.137601 | 30.21% |

Baseline A–E1 belum memiliki query prediksi S1; status readout disabled/untrained
menghasilkan nol sesuai protocol. Sebagai pembanding tambahan yang lebih kuat,
prediksi sederhana mempertahankan state sekarang memiliki RMSE
**0.095790**. S1 juga lebih baik daripada pembanding itu.

Penurunan RMSE terhadap kontrol: **88.15%**.
Seed83: RMSE **0.023821**;
seed151: **0.024243**. Keduanya memenuhi batas
RMSE ≤0,06 dan penurunan ≥30%.

Kontras keadaan dengan operasi/konteks pendahulu yang sama menghasilkan RMSE
kontrol **0.114854** → **0.029917**;
korelasi kontras prediksi–aktual **0.965357**. Ini menunjukkan
penggunaan keadaan neural, bukan sekadar nama operasi. Konteks historis yang
redundan dengan seluruh keadaan sekarang belum dibuktikan memiliki kontribusi
kausal independen; jangan mengklaim demikian.

Ablation readout mengubah 272 sinaps pada salinan evaluasi.
Menghapus witnesses tidak mengubah satu pun prediksi. Jadi hasil memerlukan
parameter sinaptik yang dipelajari, bukan lookup jawaban atau menjalankan
rumus operasi untuk meramal jawabannya.

## 5. Retensi dan alasan PARTIAL

| Oracle | Sebelum | Setelah S1 | Kontrol tanpa S1 dengan waktu oracle setara |
| --- | --- | --- | --- |
| A — ordered/temporal | 8/8 margin positif | 8/8 margin positif | Persis sama dengan setelah S1 |
| B — recurrent context | 8/8 margin positif | 8/8 margin positif | Persis sama dengan setelah S1 |
| C — native selection | 6/6 | 6/6 | 6/6 |
| D — observed intermediate | 8/8 margin positif | 8/8 margin positif | Persis sama dengan setelah S1 |
| E1 — nuisance resistant | 35/36 | 35/36 | 35/36 |

Perbandingan langsung terhadap baseline asli memberi selisih margin maksimal
**1.24193893858e-06**, di atas batas protocol **1e-7**.
Gate ini **FAIL**, dan status keseluruhan tetap **PARTIAL**.

Akar penyebab yang ditemukan ada pada jadwal oracle: helper historis `flexAt`
menentukan waktu observasi dari sequence. S1 menambah 518 revision, sehingga
oracle pasca-S1 memberi waktu **0,518 detik** lebih lambat. Decay native normal
berjalan selama probe. Kontrol tambahan dinaikkan sequence-nya lewat trajectory
native nyata tanpa pembelajaran; tidak mengedit sequence/bobot atau mengubah
oracle. Pada waktu setara, seluruh hasil A/B/C/D dan E1 persis sama.

Tidak ditemukan catastrophic forgetting akibat pembelajaran S1 dalam pengukuran
itu. Seluruh parameter lama berada dalam batas 1e-7; delta bobot Dynamic lama
maksimal **4.16542200732e-09**, berasal dari settlement
waktu awal. Topologi dan 24 PatternSynapses tetap; update kebijakan tetap 10.752.
Diagnosis kontrol tambahan menjelaskan kegagalan formal, tetapi tidak mengganti
ambang atau menghapus hasil negatif pertama.

## 6. Checkpoint, pemulihan dan continuation

| State | Sequence | Update kebijakan native | Neuron / synaps / PatternSynapses |
| --- | ---: | ---: | --- |
| A–E1 asli | 75.331 | 10.752 | 57 / 94 / 24 |
| S1 selesai belajar | **75.849** | **10.752** | **256 / 5.820 / 24** |
| Restart tanpa belajar ulang | 75.849 | 10.752 | 256 / 5.820 / 24 |
| Continuation empat probe nyata tanpa belajar | **75.866** | **10.752** | 256 / 5.820 / 24 |

Counter 10.752 adalah counter kebijakan operasi yang sudah ada, **bukan**
counter semua pembelajaran di N. S1 memiliki 36 command numerik dan 27 commit
numerik di atas; tidak menaikkan counter kebijakan dengan reward buatan.

Checkpoint final dan restart identik byte-for-byte. Native checksum final:
`7c63ef68dab03bd9fe1d7d56ef047d54e445c6d5ccfc3c8e75b8cb2d797ba6b0`.
Native checksum continuation:
`15105e910a7d1fcc0b1cdf4b2883493c498abbabc5afae3626dd0f9bb18ba2bf`.

| File di `s1/lineage/` | SHA-256 |
| --- | --- |
| `s1-final.json` | `4d9553ceb0f6bef04035b612263aa75df31a7d8cbd588293359303159d79e053` |
| `s1-restarted.json` | `4d9553ceb0f6bef04035b612263aa75df31a7d8cbd588293359303159d79e053` |
| `s1-continuation.json` | `d9bf6662a17c262922806672bcf53fe1484e5eb552ff0ec6699c21142333d64b` |
| `events.jsonl` | `3c692c4dccaa33ebac748bcafe06be8b4a21a036317d3fe0e0ec813060b0b9dd` |

Runtime/writer dihentikan, lalu proses lain memulihkan pasangan state dan journal.
Prediksi holdout identik tanpa pelatihan ulang. Replay suffix dari checkpoint
pra-S1 dan A–E1 asli cocok; prefix journal A–E1 tetap identik. Continuation
dari checkpoint final juga cocok dengan checkpoint continuation. Verifikasi
artefak terakhir membaca state hasil simpan dan tidak melakukan training.

Arsip menyertakan seluruh journal, checkpoint setiap stage, dua pasangan
rollback, sumber A–E1 terverifikasi, fit/check/holdout, prediksi, diagnosis,
metrik, log dan manifest SHA-256. Kontrol evaluasi diberi nama khusus dan
bukan pengganti lineage `s1/lineage/`.

Paket lengkap: [tautan unduhan dan checksum](ARTIFACT.md).

## 7. Validasi engineering dan cara memulihkan

Lulus: seluruh test paket activation/knowledge/learning; **62 test runtime
terarah**, termasuk oracle A–E1 asli; coexistence/penolakan migrasi; **3 test
race**; dan **6 test produksi kanonik**. Validasi terarah ini tidak diklaim
sebagai pengulangan seluruh G8/performance matrix. Log lengkap tersedia.

Dari checkout branch penelitian, setelah ekstraksi arsip ke direktori baru:

```sh
python3 scripts/package_s1_internal_dynamics.py --verify PATH/Horizon_S1_Native_Dynamics_20261116.tar.xz
python3 scripts/run_s1_internal_dynamics.py --source PATH/source --work PATH/s1 --verify-only
```

Untuk reproduksi penelitian dari sumber yang sama di lokasi **baru**:

```sh
python3 scripts/run_s1_internal_dynamics.py --source PATH/source --work /tmp/horizon-s1-new
```

Perintah verify-only tidak melatih ulang. Runtime native dapat memulihkan
`lineage/s1-continuation.json` bersama `lineage/events.jsonl`; jangan menunjuk
file ini ke direktori Brain produksi. SHA-256 hanya memeriksa integritas byte;
pemulihan Go juga memvalidasi checksum native, sequence dan prefix journal.

## 8. Kesiapan S2 dan penghentian

State S1 dapat dipulihkan dan meneruskan pengalaman, dengan prediksi yang
bertahan serta retensi fungsional pada waktu setara. Ia disimpan sebagai
**kandidat sumber S2**, bukan state produksi atau persetujuan memulai S2.
Sebelum S2, audit perlu menerima kontrak retensi yang membandingkan waktu yang
setara, dengan tetap mempertahankan laporan kegagalan protocol S1 pertama.
Tidak ada alasan dari hasil ini untuk mengulang A–E1 atau membangun Brain baru.

Pekerjaan berhenti pada hasil, checkpoint, journal dan laporan S1 ini. PR siap
ditinjau sebagai penelitian dengan status PARTIAL. Merge memerlukan audit dan
persetujuan pemilik; S2 memerlukan instruksi berikutnya.
