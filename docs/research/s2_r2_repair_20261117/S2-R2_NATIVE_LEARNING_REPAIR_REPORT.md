# S2-R2 — Native Learning Repair Report

**Repair: PASS. Kemampuan S2: PARTIAL.** Penilaian berlaku untuk repair proposal
numerik native dengan perlindungan representasi bersama dan estimasi magnitude
error tiga proyeksi neural S1 pada operasi propagate/stop.

| Penilaian | Keputusan dan bukti |
| --- | --- |
| Tujuan | Menerima langkah sinaptik bermanfaat yang terlewat oleh full refit, dengan retensi dan evaluasi terpisah. |
| Implementasi | Satu learner v6 pada Brain/N yang sama; bounded joint, kontribusi fitur terpisah, provenance, retry, dan guard lintas query. |
| Pengujian | 256 holdout baru; RMSE 0,00957043, gain 14,17% terhadap konstan, korelasi pooled 0,519732; restart/replay/rollback dan retensi lulus. |
| Keputusan | Repair diterima sebagai mekanisme penelitian opsional. Kemampuan memerlukan perbaikan: gain setiap stream belum 20%; korelasi stream normal belum 0,5. |
| Langkah berikutnya | Tinjau PR bertingkat dan keterbatasan estimasi error pada S2; jangan mempromosikan checkpoint, merge otomatis, atau memulai S3. |

## 1. Sumber, branch dan diagnosis

Basis S2-R1: `345446fd2b9f9bd660b6394e4fc8d1df54f07418`, PR #75 di atas #74.
Main tetap `11c6e0a78547d68a0be8f78b3c801e912722608d`.
Source asli `Horizon_S2_Native_Self_Evaluation_20261116.tar.xz` memiliki SHA-256
`661d007850d3026ed22636fdc4d96e9113115fb2b9e2d66856ad88e4ba60a49f`.
75 payload diverifikasi; recovery asli sequence 76617 dan checksum
`b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d` cocok.
Sumber, journal historis dan Brain produksi A–E1 tidak dimodifikasi.

Audit S2-R1 dan sembilan kelompok evidence yang diminta digunakan: trace admission,
candidate trials, partial steps, direction contributions, input, positive controls,
root causes, witness controls, dan preservasi S1/S2. Encoding benar; full refit
mengganti seluruh readout query; admission menolak proposal yang benar-benar
memburuk. Partial steps historis adalah diagnosis mekanistik, bukan hasil repair.
Ceiling 32 bukan penyebab stall S2 lama. Check/witness overlap historis tidak
disembunyikan dan tidak dinyatakan independen secara retrospektif.

## 2. Proposal native yang diubah

`NResolutionLesson.Proposal` adalah opt-in eksplisit di WAL. Tanpa opsi ini,
perilaku v6 lama tetap berlaku. Solver `nFit`/`nFitMetric`, signed parameter writer,
feature graph, activation, recurrent dynamics, policy dan causal credit tetap
mekanisme yang sama. Tidak ada regressor eksternal, learner kedua atau jawaban S2
hardcoded. Python hanya mengaudit dan menghitung metrik.

Grid yang ditetapkan sebelum holdout: **1/64, 1/16, 1/4, 1/2**, dengan full step
sebagai kontrol. Bounded joint memakai `current + alpha*(full-current)` pada head
query yang basis readout dan graph lamanya identik. Koefisien fitur baru berawal
nol. Pergantian activation/metric tidak diinterpolasi. Semua bobot/bias tetap
finite dan dalam batas signed 7,99.

Kontribusi fitur terpisah mem-fit satu koefisien terhadap residual incumbent;
bias dan bobot lama tetap pada proposal itu. Bobot lama tetap boleh berubah
melalui proposal joint atau full yang lolos gate. Validity, precision, redundancy,
check gain dan retention asli tidak diturunkan. Receipt menyimpan strategi,
alpha, hash parameter, alasan rejection dan evidence fit/check/witness.

Pemenang bounded/decoupled yang full step pada kelompok pencarian sama ditolak
tersimpan di `evidence/bounded-winners-missed-by-full-step.json`. Ini bukti
proposal bermanfaat yang benar-benar dipasang, bukan sekadar kenaikan jumlah commit.

## 3. Kegagalan engineering pertama dan koreksi sebelum holdout

Percobaan delapan blok pertama disimpan lengkap sebagai **FAIL untuk revisi
engineering awal**. A–E1 tetap identik, tetapi S1 bergerak hingga
**0,00046141326817925044**, melanggar batas 1e-12 yang telah ditetapkan.
Full-refit resolution mengubah fitur bersama 254 dan 246. Gate witness lama
mengizinkan perubahan kecil ini dalam batas kuantisasi; itu belum memenuhi
persyaratan preservasi S1 yang lebih ketat pada penelitian ini.

Amendemen `REPAIR_AMENDMENT.md` dibuat **sebelum holdout pertama**.
`preserve_shared: true` menghitung closure dependensi semua head lain yang sudah
known, lalu menolak perubahan representasi dalam closure tersebut dengan alasan
`cross-query-representation-change`. Readout query tetap plastis; fitur baru dan
fitur yang tidak dipakai head lain tetap dapat berubah. Opsi default false
mempertahankan replay persis WAL revisi awal maupun WAL historis.

Verifikasi diulang dari checkpoint S2 asli, dengan jadwal delapan blok dan seed
yang sama; tidak menyalin parameter percobaan gagal, tidak memilih checkpoint
terbaik, tidak menambah seed/pelajaran untuk mengejar PASS. Sebanyak 512 observasi
development aktual dipakai ulang beserta ledger pra-operasi dan WAL observer.
Ada dua akuisisi R2 delapan blok, termasuk yang gagal; biaya ini tidak disamarkan.
A–E1/S1 dan 16 command S2 lama tidak dilatih ulang. Hanya revisi yang lolos
engineering diizinkan memasuki satu final holdout. Tidak ada iterasi setelahnya.

Label diagnostik awal untuk nol kandidat juga diperbaiki dari invalid numerical
candidate menjadi candidate-search-exhausted. Perbandingan 20 control commands
membuktikan angka, trials dan keputusan admission tidak berubah; bukti awal
tersimpan di `evidence/development-label-correction/`.

## 4. Independence dan retry

Current fit/check ditolak bila berbagi episode atau encoded input persis,
termasuk ketika caller mengganti nama episode. Witness yang overlap current check
excluded dari fit, tetapi tetap dipakai untuk retensi. Witness check baru berperan
evaluation-only. Seluruh witness sumber tetap identik byte-field hingga checkpoint
continuation; budget 128 tidak diperbesar dan witness historis tidak dievikt.
Pengalaman baru lengkap tetap ada dalam WAL meskipun tidak masuk sampel witness.
Ini membatasi cakupan jaminan retensi; bukan bukti terhadap semua pengalaman dunia.

Pada akuisisi utama, setiap command memakai 64 fit baru + 32 witness historis dan
64 check baru yang tidak overlap. Nama field atau hash yang berbeda tidak dijadikan
bukti tunggal independence. Episode, measurement, query, uncertainty, target dan
provenance tersedia untuk audit. Input dikunci/di-hash sebelum native target step;
forecast adalah read S1 sebelum operasi, history berasal dari error operasi lalu.
Tidak ada target error operasi kini di input pra-operasi.

Fingerprint pencarian mencakup model, fit/check, witnesses, precision dan konfigurasi.
Retry identik setelah rejection menghasilkan duplicate-proposal; command tetap
tercatat dan sequence tetap maju. Duplicate event ID tetap ditolak tanpa mutasi.
Receipt membedakan exhaustion, no check improvement, retention regression,
insufficient identifiable information, duplicate dan invalid numerical candidate.
Uji source dengan race detector dan replay persis lulus; fixture label sintetis
untuk retry ditandai engineering-only dan tidak masuk bukti kemampuan.

## 5. Protokol dan pembelajaran berurutan

Development: empat pasangan native fit/check 64+64; seed dan kondisi pada DESIGN.
Lima arm dimulai dari source identik. Strategi gabungan dipilih sebelumnya;
bukan arm terbaik yang dipilih setelah melihat holdout. Continuation berisi
**8 command**, masing-masing 64 fit + 64 check baru; normal/shift bergantian.
Checkpoint kandidat wajib blok kedelapan, apa pun nilainya.

| Blok | RMSE sebelum pada check baru | RMSE sesudah | Commit | Fitur |
| --- | ---: | ---: | --- | ---: |
| 1 | 0.00838395 | 0.00822298 | True | 31 |
| 2 | 0.01366675 | 0.01003743 | True | 32 |
| 3 | 0.01122454 | 0.00987494 | True | 32 |
| 4 | 0.01362111 | 0.01227002 | True | 32 |
| 5 | 0.01087369 | 0.00923135 | True | 32 |
| 6 | 0.01782311 | 0.01202769 | True | 32 |
| 7 | 0.00858859 | 0.00781755 | True | 32 |
| 8 | 0.00978278 | 0.00899817 | True | 32 |

Accepted commands: v6 lama **5/8**, independent full refit **3/8**, bounded joint
**8/8**, new-feature-only **2/8**, gabungan **8/8**. Fitur mencapai ceiling 32
setelah dua growth; pembaruan readout masih berjalan sesudah itu. Ceiling memang
membatasi growth lanjutan di R2, berbeda dari diagnosis stall S2 asli pada 30 fitur.

Full holdout: seed 51001 dan 51031, 128 kasus per stream, sesudah code/config,
checkpoint, konstanta, bin, cutoffs dan aturan keputusan dibekukan.
`CONFIRMATORY_LOCK.json` menyimpan hash; kode native yang dikunci berasal dari
commit `42e343e7b7869eb836b8c65582211b571d81bf98`. Tidak ada perubahan learner,
tuning, penambahan kasus atau pemilihan ulang checkpoint setelah holdout.

## 6. Holdout independen: kemampuan tetap PARTIAL

Semua model memprediksi **256 observasi identik** dengan target error aktual native.
Metrik dalam satuan proyeksi neural ternormalisasi, bukan probabilitas kalibrasi
atau akurasi klasifikasi yang diada-adakan. Konstan 0,015406036556304356 hanya
berasal dari development-fit; batas kalibrasi 0,003 dengan lima batas bin tetap.

| Kondisi | RMSE | MAE | Korelasi | Calibration error |
| --- | ---: | ---: | ---: | ---: |
| S2 asli, tanpa continuation | 0.01106686 | 0.00808159 | 0.294787 | 0.00255129 |
| v6 lama, continuation | 0.01158955 | 0.00876010 | 0.421099 | 0.00553222 |
| Full refit + pemisahan data/guard | 0.01008800 | 0.00768679 | 0.468882 | 0.00191965 |
| Bounded joint | 0.00957043 | 0.00737742 | 0.519732 | 0.00161795 |
| Hanya kontribusi fitur baru | 0.01038753 | 0.00773490 | 0.418843 | 0.00231654 |
| Gabungan terdaftar | 0.00957043 | 0.00737742 | 0.519732 | 0.00161795 |
| Konstan dari development-fit | 0.01115066 | 0.00804539 | -0.000000 | 0.00143471 |
| Query native belum dilatih | 0.02014670 | 0.01684074 | 0.000000 | 0.01684074 |

Gain gabungan terhadap konstan **14,1716%**, terhadap S2 asli pada data yang sama
**13.5217%**, terhadap full refit independen
**5.1306%**.
Hasil S2 historis 5,77% memakai holdout lain sehingga tidak dijadikan pembandingan
langsung dengan persentase baru. Bounded-joint dan gabungan menghasilkan prediksi
bit-identik pada holdout ini: manfaat tambahan memasukkan arm decoupled dalam
strategi gabungan **belum terbukti**. Arm decoupled sendiri tetap diuji dan berbeda.

| Stream | N | RMSE kandidat | RMSE konstan | Gain | Korelasi | Calibration error |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Normal, seed 51001 | 128 | 0.00772944 | 0.00838536 | 7.82% | 0.435292 | 0.00133203 |
| Bergeser, seed 51031 | 128 | 0.01111045 | 0.01335515 | 16.81% | 0.524534 | 0.00197696 |

Kedua gain gagal target 20%; korelasi normal gagal 0,5. Kalibrasi kedua stream
lulus. Pooled AUROC high-vs-low **0,736513**; gap estimasi high-low positif untuk
propagate dan stop. Hasil ini mendukung peningkatan terbatas, bukan PASS kemampuan.
Exit 0 pada harness berarti evidence konsisten, bukan semua gate ilmiah lulus.

## 7. Forecast, history dan ablation

Delapan koordinat tambahan kini memiliki source neuron dan jalur signed menuju
axis baru; parameter native dibandingkan langsung dengan graph readout, tolerance
1e-12. Lihat `evidence/signed-channel-attribution.json`. Dua fitur baru: produk
neuron 241×244 (unit 265) dan arah neural unit 266. Ini hasil recruitment native,
bukan hubungan yang ditanam untuk membuat kanal tampak digunakan.

| Intervensi pada target tetap | RMSE | Delta RMSE terhadap full | Perubahan output maksimum |
| --- | ---: | ---: | ---: |
| Forecast dihapus | 0.0095917094 | +2.128e-05 | 4.237e-04 |
| Magnitude history dihapus | 0.0095705103 | +7.906e-08 | 5.767e-06 |
| History-action dihapus | 0.0095687113 | -1.720e-06 | 3.152e-05 |
| History-presence dihapus | 0.0095991076 | +2.868e-05 | 2.713e-04 |
| Magnitude dan action dihapus | 0.0095687899 | -1.641e-06 | 3.505e-05 |
| Magnitude dipermutasi bersyarat | 0.0095704073 | -2.395e-08 | 5.375e-06 |
| History-action dipermutasi | 0.0095646944 | -5.737e-06 | 6.304e-05 |
| Forecast dipermutasi | 0.0095647111 | -5.720e-06 | 3.472e-04 |
| Seluruh tambahan dipermutasi | 0.0095592151 | -1.122e-05 | 4.099e-04 |
| Aktivitas fitur baru di-clamp | 0.0177191942 | +8.149e-03 | 3.690e-02 |
| Readout fitur baru dihapus | 0.0177191942 | +8.149e-03 | 3.690e-02 |
| Seluruh readout S2 dihapus | 0.0201466973 | +1.058e-02 | 4.413e-02 |

Magnitude history mempunyai efek neural yang dapat diukur, tetapi manfaat
prediktif independennya **tidak terbukti**: penghapusannya hanya menaikkan RMSE
sekitar 7,91e-8, sedangkan permutasi bersyarat justru sedikit memperbaiki hasil.
Permutasi mempertahankan current action, prior action, history-action dan presence.
Menghapus history-action juga memperbaiki hasil; itu tidak boleh disebut bukti
penggunaan magnitude error. Forecast removal merugikan sedikit, tetapi permutasi
forecast memperbaiki hasil, sehingga manfaat independennya juga belum kokoh.

New-feature clamp dan penghapusan readout baru menaikkan RMSE menjadi 0,01771919;
seluruh readout S2 dihapus menjadi 0,02014670. Representasi/readout neural yang
baru bermanfaat secara keseluruhan, tetapi ini tidak membuktikan tiap kanal
tambahan berguna. Clamp graph diagnostik tidak pernah dipersist sebagai model
unit-axis yang valid. Seluruh 13 kondisi identik setelah restart.

## 8. Retensi A–E1 dan S1

Kontrol A–E1 menerima jadwal native yang sama tanpa numerical learning, kemudian
diselaraskan sequence/waktu. Oracle A/B/C/D/E1 historis tidak diubah.
Delta maksimum **0**, C **6/6**, E1 **35/36**, sebelum maupun setelah continuation.
Native policy state lengkap identik dengan source; counter tetap **10.752**.

Tiga proyeksi S1 identik pada **1.536 koordinat dari 512 observasi training-check**,
dan pada **768 koordinat holdout tersimpan** sesudah restart/continuation.
Context contrast dan semua readout S1 karenanya tetap sama pada observasi tersebut.
Ablation 400 edge readout S1 pada salinan disposable mengubah output maksimum
0,5342721624, membuktikan readout sinaptik masih berkontribusi. Tidak ada model
ablation disimpan sebagai checkpoint. **PARTIAL dan kegagalan retensi historis
S1 tetap dicatat; tidak diubah menjadi PASS secara retrospektif.**

## 9. Checkpoint, WAL dan recovery

| State | Sequence | Neuron | Sinaps | PatternSynapses | Fitur |
| --- | ---: | ---: | ---: | ---: | ---: |
| Source S2 asli | 76617 | 264 | 6244 | 24 | 30 |
| Final S2-R2 | 80681 | 274 | 6408 | 24 | 32 |
| Continuation S2-R2 | 80735 | 274 | 6408 | 24 | 32 |

Final checksum: `9c6cbd9e7d7caeeaaa0a788d0122a94fab32701ceda2c8bb6476fa3c522d47d8`.
Continuation checksum: `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.
Source → final: 164 sinaps ditambah, 360 berubah, **nol dihapus**. Pertumbuhan
bukan kriteria PASS; metrik, ablation dan retensi di atas adalah dasar keputusan.

Restart final, replay seluruh suffix dari source 76617, rollback source pair,
checkpoint checksum dan exact prefix **44.021.001 byte** journal historis lulus.
Sesudah holdout, jadwal continuation yang sudah ditetapkan menjalankan 6 observasi
serta 8 observasi untuk 4-fit/4-check dan satu command tambahan. Command itu
commit, MSE check 6,72793e-5 → 5,29860e-5. Checkpoint continuation bukan pengganti
checkpoint confirmatory dan tidak dipilih memakai skor holdout.

Gunakan `s2r2/s2r2-final.json` + `s2r2/s2r2-final-events.jsonl` untuk final;
gunakan `s2r2/s2r2-continuation.json` +
`s2r2/training-combined/events.jsonl` untuk kelanjutan terbaru. Restore memakai
implementasi S2-R2 dengan proposal yang tersimpan di WAL. Revisi engineering
yang gagal disimpan terpisah lengkap dan replay-nya juga persis.

## 10. Validasi engineering dan CI

Go 1.22.12 linux/amd64. Full Go tests, vet, build; directed race native/metric;
full race tujuh package runtime/knowledge/learning/teaching/bridge/core/CLI;
Android arm64 service/chat builds; source-backed retry race; signed persistence,
independence, dua tugas numerik non-S2, concurrent receipt ownership dan fault
injection bounded lost-ack/interrupted-installation lulus. Recovery fault
injection memulihkan tepat satu command dan menolak duplicate retry.

Default learner dicocokkan dengan checkout S2-R1 asli pada 4 control commands dan
8 training commands: receipt dan complete canonical state identik. Tersedia
hasil sebelum serta sesudah safeguard. Build-tag probe/audit hanya alat diagnosis,
bukan jalur serving baru. Detail CI mutakhir ditulis pada `CI_STATUS.json`;
log dan hasil negatif lokal tersedia di `evidence/`. Push CI untuk kode native
yang dibekukan, run **44015206135**, telah lulus job validate dan monitor.

Uji non-S2 tambahan awal meminta presisi 1e-5 pada nilai baru langsung setelah
satu recruitment; prediksi -0,07875 terhadap target -0,09 belum memenuhi
presisi itu, sementara head lama tetap identik. Seperti curriculum precision
yang sudah ada, enam lesson refinement terdaftar pada fixture generik
memenuhi presisi tersebut tanpa mengubah threshold. Ini pengujian numerik
terpisah, bukan tambahan lesson S2 atau tuning holdout. Log kegagalan disimpan.

## 11. Artefak, reproduksi dan keputusan berikutnya

`ARTIFACT.md` menunjuk arsip persisten lengkap: source asli, final/continuation,
seluruh WAL/prefix, kontrol, receipt, actual observations, ledger pra-operasi,
manifest, kode reproducible, dan revisi engineering gagal. `RESTORE.md` dan
`REPRODUCE.md` menjelaskan pemulihan tanpa melatih ulang source serta reproduksi
penelitian. Manifest mengautentikasi payload; hash tanpa file bukan deliverable.

Repair memenuhi tujuan lokal: langkah bermanfaat yang terlewat full refit diterima,
perlindungan lama tetap ada, independent check valid, representasi S1 dipertahankan,
dan hasil dapat direplay. **PASS Repair tidak menaikkan status S2 menjadi PASS.**
S2 masih memerlukan perbaikan estimasi stream normal/shift dan pengujian informasi
history yang lebih jelas. Review perubahan opsional ini sebagai satu PR non-draft
berbasis branch S2-R1. Main, Brain produksi dan kesiapan S3 tidak dipromosikan.
