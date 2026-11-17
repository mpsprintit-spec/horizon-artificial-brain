# S2-R3 — Neural Capacity & Predictive Information Diagnosis

Tanggal penelitian: 17 November 2026. Repository:
`mpsprintit-spec/horizon-artificial-brain`. Branch penelitian diturunkan dari
S2-R2 `7196c8a7f4a619040fe3933fe1c5363e5be661bc`; target PR adalah branch
S2-R2, bukan `main`. Baseline main tetap
`11c6e0a78547d68a0be8f78b3c801e912722608d`.

**Status diagnosis utama: INCONCLUSIVE. Kemampuan S2: PARTIAL.**

Tambahan kapasitas benar-benar digunakan, tetapi tidak memperbaiki S2 secara
konsisten. Pada holdout baru yang sama, RMSE gabungan tiga model kapasitas 32
adalah **0.00999193**, sedangkan kapasitas 64 yang mencapai 40 fitur adalah
**0.01057028**. Peningkatan relatif arm 64 terhadap 32 adalah **−5.79%**, CI95
**[−15.91%, +2.48%]**. Pada stream normal, efeknya **−11.32%**, CI95
**[−17.19%, −5.10%]**; ketiga seed memburuk. Tidak ada dasar untuk mempromosikan
64 sebagai kapasitas produksi atau mengklaim H1 terbukti.

### Ringkasan penilaian

| Penilaian | Hasil |
| --- | --- |
| Tujuan | Memisahkan batas kapasitas, kualitas representasi, dan manfaat independen history pada S2 |
| Implementasi | Satu learner native, opsi kapasitas per lesson yang dijournal, default 32, read-only probes; tanpa controller baru |
| Pengujian | Tiga seed; 4.352 observasi aktual baru; 1.024 kasus confirmatory; kontrol informasi; ablation; retensi dan persistence |
| Keputusan | Implementasi eksperimen dan pemulihan tervalidasi; penambahan kapasitas belum layak dipromosikan; penyebab utama S2 belum terisolasi |
| Langkah berikutnya | Diagnosis kualitas recruitment/readout dan akses kanal temporal sebelum merancang adaptive capacity |

## 1. Verifikasi sumber Brain

Arsip asli `Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz`, ukuran
18.992.952 byte, mempunyai SHA-256
`e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46`.
**443/443 payload terautentikasi.** Recovery menggunakan `OpenEventLog` dan
`RebuildRuntimeFromCheckpointAndJournal` pada pasangan:

| Komponen | Nilai |
| --- | --- |
| Checkpoint | `s2r2/s2r2-continuation.json` |
| Journal | `s2r2/training-combined/events.jsonl` |
| Sequence | **80735** |
| Native checksum | `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3` |
| Neuron / sinaps / fitur | 274 / 6408 / 32 |
| Native policy updates | 10.752 |

Activation, learning dan DNF menunjuk pemilik Brain yang sama. Semua arm
dimulai dari salinan byte-identik pasangan ini. Tidak ada akuisisi ulang
A–E1/S1/S2/R2, Brain kosong, pemindahan parameter antar-arm, atau penggantian
checkpoint produksi. Sequence 80681 hanya digunakan sebagai **kontrol replay
historis**, tidak pernah menjadi sumber pelatihan R3.

## 2. Audit batas 32 dan ketergantungan kode

Peta lengkap ada di [CODE_AUDIT.md](CODE_AUDIT.md). Temuan kode dan runtime:

1. `maxNFeatures=32` membatasi jumlah circuit numerik dan validasi topologi
   tersimpan; bukan jumlah seluruh neuron atau dimensi input.
2. Readout dan proposal resolusi tetap dicari saat ceiling tercapai. Pada R3,
   arm 32 melakukan 24 learning commands dan 22 pembaruan diakui.
3. Kedua jenis kandidat growth—residual direction dan product—tidak dienumerasi
   ketika circuit sudah mencapai ceiling.
4. Growth yang diterima membuat neuron fitur dan signed synapses native;
   sumber input baru juga dapat memerlukan neuron bila belum tersedia. R3
   aktual menambah delapan neuron per arm 64, tanpa sumber tambahan.
5. Product search memakai parent yang bervariasi, ranking covariance Fit dan
   paling banyak delapan product candidate. Satu residual direction diusulkan.
   Menaikkan kapasitas tidak mengubah kualitas/ranking pencarian tersebut.
6. Parameter, quantizer, readout dan parent edge dipulihkan dari synapses
   eksitator/inhibitor kanonik; tidak ada estimator kedua yang disimpan.
7. Batas lama juga merupakan kontrak sumber daya/persistence. Reader kompatibel
   diperlukan untuk topologi >32; memperbesar konstanta global saja tidak cukup.

## 3. Perubahan eksperimental

Tambahan native terbatas pada `NMetricProposalConfig.FeatureLimit` (omitted,
32 atau 64) dan `NResolutionState.ExperimentalFeatureLimit` sebagai restore
bound dari lesson yang diakui. Default produksi dan jalur v5 tetap 32.
Lesson berikutnya harus meminta ceiling eksperimen secara eksplisit; memuat
checkpoint besar tidak mengaktifkan growth lebih jauh secara implisit.

Learning admission, precision `.002`, candidate ranking, signed bounds,
regularisasi, bounded fractions `[1/64,1/16,1/4,1/2]`, strategi `combined`,
retention dan `PreserveShared=true` tidak diubah. Hanya ceiling berbeda antar-arm.
Probe dan clamps menggunakan model terlepas di bawah read lock; tidak pernah
dipersist sebagai Brain hasil ablation.

## 4. Protokol, seed dan provenance

[PROTOCOL.md](PROTOCOL.md) ditulis sebelum akuisisi. Development: 256 observasi
baru, seed 830011/830017 dan 830023/830029. Pelatihan: seed **37, 83, 151**;
delapan blok per seed, 64 Fit + 64 Check per blok, normal/bergeser bergantian.
Fit seed `840000+1000*s+20*b+1`; Check `...+7`. Setiap set memulai history kosong.

Observer terpisah dari learner menghasilkan pengalaman melalui operasi native
aktual dengan `Learn=false`. Kedua arm mendapat input dan native command yang
identik, berurutan sama; native trajectory digest yang sama harus diterima.
Learning deterministik; seed memilih pengalaman, bukan optimizer acak.
GOMAXPROCS=2, host x86_64 yang sama, clock ilmiah tetap. Batas 30 menit per seed,
20 menit per fase lain. Tidak ada pergantian seed atau penambahan lesson setelah
melihat skor buruk. Waktu Train tiga seed sekitar 607,53 / 633,12 / 636,12 detik.

**4.352 ID episode dan input digest unik**, seluruhnya mempunyai ledger yang
di-fsync sebelum native command pembentuk target. Tidak ada overlap Fit/Check,
antar-set atau final; evidence hash Fit/Check/witness identik antar kapasitas.
Tidak ada holdout R2 yang dipakai melatih, memilih kapasitas atau checkpoint.
Historical witness bank tetap utuh; 32 witness Fit legacy yang diizinkan
mekanisme R2 tetap merupakan pengalaman lama, bukan akuisisi ulang.

Metric protocol dan candidate final disegel sebelum final dibuka. Ada
ketidaksesuaian prose/implementasi mengenai sumber development bins: constant
berasal dari Fit, sedangkan SD, cutoffs, bins dan calibration limit berasal dari
development Check. Nilai aktual dibekukan sebelum pelatihan dan tidak diubah.
Ini tidak memengaruhi paired RMSE utama; semua arm gagal gate normal RMSE dan
korelasi terlepas dari bins. Lihat [PROTOCOL_DEVIATIONS.md](PROTOCOL_DEVIATIONS.md).

## 5. Pertumbuhan per blok

| Blok | Fitur arm 32, setiap seed | Fitur arm 64, setiap seed | Pembaruan arm 32: 37 / 83 / 151 | Pembaruan arm 64 |
| --- | ---: | ---: | --- | --- |
| 0 | 32 | 33 | ya / ya / ya | 3/3 |
| 1 | 32 | 34 | ya / ya / ya | 3/3 |
| 2 | 32 | 35 | ya / ya / tidak | 3/3 |
| 3 | 32 | 36 | tidak / ya / ya | 3/3 |
| 4 | 32 | 37 | ya / ya / ya | 3/3 |
| 5 | 32 | 38 | ya / ya / ya | 3/3 |
| 6 | 32 | 39 | ya / ya / ya | 3/3 |
| 7 | 32 | 40 | ya / ya / ya | 3/3 |

RMSE sebelum/sesudah **setiap** blok, biaya dan inventory ada di
[growth-summary.csv](evidence/growth-summary.csv); seluruh trials dan receipts
lengkap ada di artefak. Semua 48 transaksi utama terlaksana: 22/24 update pada
32 dan 24/24 pada 64. Final selalu blok 7, tidak dipilih berdasarkan holdout.
Law lama hanya dapat menambah satu fitur per command: studi menguji **32 versus
hingga 40 fitur di bawah ceiling 64**, bukan jaringan yang sudah dilatih sampai
64. Tidak ada kesimpulan bahwa fitur ke-41 sampai 64 tidak bermanfaat.

## 6. Peta neuron/sinaps baru

Arm 32: tetap 274 neuron, 6408 sinaps. Seluruh arm 64 menambah node fitur
**275–282** pada substrate yang sama.

| Seed | Neuron final | Sinaps final | Tambahan sinaps | Fitur baru: direction / activation product / metric product |
| --- | ---: | ---: | ---: | --- |
| 37 | 282 | 6740 | 332 | 1 / 5 / 2 |
| 83 | 282 | 6738 | 330 | 1 / 4 / 3 |
| 151 | 282 | 6956 | 548 | 3 / 3 / 2 |

`analysis/new-neural-structure.json` dalam artefak memuat setiap neuron dan
sinaps baru lengkap. `signed-synaptic-paths.json` memverifikasi ulang bobot axis,
parent dan readout dari pasangan signed synapses terhadap graph hasil loader;
seluruhnya cocok dalam 1e-12. Tidak ada parameter dipindahkan dari kontrol
sintetis atau arm lain.

## 7. Kontribusi, redundansi dan residual

[new-feature-contributions.csv](evidence/new-feature-contributions.csv) memuat
**semua 24 fitur baru**: parent, bobot signed, kontribusi RMS, saturation,
korelasi, residual novelty, serta ablation per stream dan CI. Artefak juga
memuat output, code, activation, metric/error bound setiap fitur pada semua
kasus dan ablation tiap fitur lama.

Rank matriks 32 fitur lama plus bias adalah 33 pada observasi akhir. Variansi
fitur baru yang tidak dijelaskan proyeksi linear fitur lama berkisar sekitar
**2,97%–74,66%**; tidak ada fitur baru di bawah ambang deskriptif 1%. Proyeksi ini
tanpa label dan hanya diagnosis geometrik; bukan bukti informasi target baru.
Banyak activation baru berada pada saturation `|tanh(code)|>=.99`, termasuk
semua delapan fitur baru seed 83 dan 151. Saturation adalah kondisi yang diukur,
bukan bukti otomatis bahwa fitur tidak berguna; beberapa head memakai metric.

Mematikan seluruh fitur baru atau hanya readout barunya pada arm 64 menaikkan
RMSE gabungan dari **0.01057028 ke 0.03052366**, biaya relatif **+188,77%**,
CI95 **[+3,72%, +380,23%]**. Kedua clamp identik pada kasus ini. Artinya graph
yang sudah beradaptasi menggunakan fitur baru. Ini **tidak** membuktikan bahwa
prosedur menambah fitur mengalahkan pelatihan arm 32: readout lama juga telah
beradaptasi bersama fitur baru. Total efek antar-arm justru tidak membaik.

Residual sebelum/sesudah tiap recruitment disimpan pada Check per blok; residual
independen akhir dibandingkan antar-arm dan terhadap setiap feature clamp.
Tidak ada neuron kanonik dihapus berdasarkan korelasi atau ablation ini.

## 8. Hasil independent holdout

Setelah seal: **512 normal + 512 bergeser**, seed 990011/990031, satu akuisisi.
Semua model mengerjakan kasus yang sama. CI menggunakan 2.000 paired bootstrap,
seed 980017, blok temporal empat kasus per stream dan resampling seed pelatihan
sebagai cluster luar. Tiga seed membatasi presisi inferensi antarpelatihan.

| Seed | Kapasitas / fitur aktual | RMSE pooled | RMSE normal | RMSE bergeser | Korelasi pooled |
| --- | --- | ---: | ---: | ---: | ---: |
| 37 | 32 / 32 | 0.01090911 | 0.00874768 | 0.01270808 | 0.2670 |
| 37 | 64 / 40 | 0.01072419 | 0.00993719 | 0.01145725 | 0.4592 |
| 83 | 32 / 32 | 0.00925585 | 0.00841141 | 0.01002944 | 0.4953 |
| 83 | 64 / 40 | 0.01079232 | 0.00968997 | 0.01179207 | 0.3219 |
| 151 | 32 / 32 | 0.00973842 | 0.00869345 | 0.01068164 | 0.4587 |
| 151 | 64 / 40 | 0.01018382 | 0.00913843 | 0.01113147 | 0.4631 |

| Efek 64 terhadap 32; positif berarti perbaikan | Estimasi | CI95 |
| --- | ---: | --- |
| Pooled | −5.79% | [−15.91%, +2.48%] |
| Normal | −11.32% | [−17.19%, −5.10%] |
| Bergeser | −2.37% | [−17.09%, +10.71%] |

Sumber continuation 80735 mempunyai RMSE **0.01630707 pada kasus baru ini**;
kedua prosedur continuation memperbaikinya. Itu tidak sama dengan perbandingan
terhadap baseline konstan, dan bukan penulisan ulang hasil historis R2. Skor
historis R2 0.00957043 berasal dari checkpoint final 80681 dan holdout lain;
tidak boleh dianggap skor terukur checkpoint 80735 pada semua distribusi.
Tidak ada model seed 83 dipilih atau dipromosikan karena skornya.

## 9. Audit informasi history dan forecast

Seluruh masks dan permutasi asli ditetapkan sebelum holdout: magnitude,
forecast, previous operation S1, history presence, history operation, semua
extras S2; sepuluh conditional permutations untuk history dan forecast.
Konteks operasi dan stream dipertahankan; forecast juga dicocokkan dengan
quartile proyeksi prestate. Ini intervensi komputasional dengan target tetap,
bukan intervensi fisik pada dunia atau estimator mutual information.

| Biaya penghilangan input, pooled | Arm 32 | Arm 64 |
| --- | ---: | ---: |
| History magnitude | +0.00032%; CI [−0.00023%, +0.00111%] | −0.00299%; CI [−0.04769%, +0.04001%] |
| Permutasi history bersyarat | −0.00006%; CI [−0.00035%, +0.00018%] | +0.00305%; CI [−0.02815%, +0.03940%] |
| Forecast | +0.0533%; CI [−0.0280%, +0.1801%] | +0.3337%; CI [−0.4112%, +1.4368%] |

Manfaat independen history magnitude tidak terdemonstrasi; upper bound biaya
mask/permutation di tiap stream berada di bawah margin 2% yang ditetapkan.
Namun ini adalah batas efek **model yang dilatih**, bukan bukti bahwa sumber
history secara umum tidak mengandung informasi.

Previous operation mempunyai efek lebih besar: mask-nya menghasilkan RMSE
0.01310632 / 0.01509482 untuk 32 / 64. Jangan mencampur kanal ini dengan
historical-error magnitude atau history operation. Detail semua kondisi dan CI
ada di [input-feature-interventions.json](evidence/input-feature-interventions.json).

Forecast merupakan fungsi S1 yang dibekukan dari prestate/operasi/konteks
terkode; transformasinya dapat mempermudah pembelajaran meski tidak menambah
informasi kondisional. Semua 1.024 prestate aktual hanya mempunyai unit aktif
di 36 unit populasi yang dikodekan. Generator mengambil stimulus acak baru tiap
episode: keterbatasan sinyal temporal independen **masuk akal secara struktural**,
tetapi H3 empiris tetap harus memenuhi kontrol positif.

Mask `no-extras` confirmatory menghapus extras yang ditambahkan S2. Audit yang
lebih luas, hanya current state dan current operation tersisa termasuk menghapus
sembilan proyeksi keadaan sebelumnya dan flags S1, dilakukan pada **128
development Check tersimpan**, bukan tambahan kondisi final. Lihat
[DEVELOPMENT_INPUT_AUDIT.md](DEVELOPMENT_INPUT_AUDIT.md).

## 10. Kontrol informasi terisolasi

Setiap kontrol adalah salinan 80735, query terpisah, empat lesson identik,
64 Fit/64 Check per lesson dan 512 uji independen. Target sintetis positif
memuat sinyal temporal; negatif menggunakan target independen dari history.
Tidak ada label/parameter kontrol memasuki pelatihan S2 aktual.

| Seed | Positive control RMSE, 32 | Positive control RMSE, 64 |
| --- | ---: | ---: |
| 37 | 0.01152769 | 5.1816e−8 |
| 83 | 0.01156857 | 7.2825e−9 |
| 151 | 0.01154351 | 2.2716e−7 |

Semua arm 64 mempelajari kontrol positif dan lolos causal mask/permutation;
hanya perlu **33 fitur**, kemudian native precision gate menghentikan growth.
Semua arm 32 gagal: basis activation yang dipilih konstan saat history berubah;
satu metric feature bervariasi, tetapi candidate audit memuat invalid/bounded
fit dan shared-representation rejections. Diagnosis ini tidak memisahkan
koefisien bound, quantization dan shared protection sebagai penyebab tunggal.
Kontrol negatif bertindak sesuai harapan dan tidak menunjukkan keuntungan
independen >2% terhadap constant.

Karena kontrol positif **32 gagal**, prasyarat kuat H3 pada kedua kapasitas tidak
lulus. Tidak sah menyimpulkan bahwa history aktual tidak informatif hanya
karena kedua model S2 hampir tidak memakainya. Sebaliknya, sukses sintetis 64
tidak membuktikan pemahaman error aktual. Semua hasil kontrol negatif dan
gagal tetap tersimpan.

## 11. Retensi A–E1, S1 dan representasi lama

Untuk setiap seed/arm, observer kontrol menerima native schedule yang sama dan
sequence disejajarkan. Ordered/temporal A, contextual B, operation selection C,
intermediate reuse D, dan nuisance-resistant E1 semuanya memenuhi oracle lama:

- A–D oracle delta **0**, E1 delta **0**; C **6/6**, E1 **35/36**.
- Setiap blok: tiga proyeksi S1 pada 128 observasi development identik,
  maximum prediction shift **0**; batas strict tetap 1e−12.
- Final independen: **3.072 koordinat S1 per model**, delta **0**.
- Witness historis dan native policy state exact; updates tetap **10.752**.
- Pemeriksaan A–E1 setelah restart memberikan hasil yang sama.

Retensi ini relatif terhadap sumber 80735 dan baterai pengujian tersebut;
status/kegagalan historis S1 atau R2 tidak dihapus. Tidak ada klaim retensi atas
semua pengalaman yang pernah mungkin dialami.

## 12. Biaya komputasi dan sumber daya

Pengukuran sequential process terpisah setelah pelatihan/retensi selesai,
GOMAXPROCS=2 pada host x86_64 yang sama; 1.024 inferensi per model. Lingkupnya
readout numerik native, bukan seluruh siklus berpikir atau benchmark HP.

| Seed | Kapasitas | p50 / p95 inferensi (µs) | Alokasi / inferensi (KiB) | Learning terakhir (detik) |
| --- | ---: | --- | ---: | ---: |
| 37 | 32 | 135.26 / 273.13 | 157.86 | 1.391 |
| 37 | 64 | 166.93 / 376.06 | 204.67 | 13.883 |
| 83 | 32 | 146.96 / 391.15 | 161.30 | 1.375 |
| 83 | 64 | 158.59 / 347.88 | 200.87 | 13.422 |
| 151 | 32 | 141.23 / 288.16 | 161.30 | 1.311 |
| 151 | 64 | 188.47 / 455.92 | 204.66 | 15.616 |

Median learning terakhir naik sekitar **10,10×**. Reeksekusi transaksi yang sudah
tercatat pada salinan disposable menghasilkan full state exact; **nol
pengalaman baru**. Biaya learning inklusif validation/search/journal/install,
bukan pengukuran terpisah satu instruksi penambahan neuron. Profiles tersedia
untuk seluruh seed/arm: pada 37/64, `nMetricRedundant` memakan sekitar 63,4%
cumulative sampled CPU, dan seluruh `learnNMetric` sekitar 90,0%. Kandidat diuji
berulang terhadap fitur lama dan observasi; ini peluang diagnosis biaya berikutnya,
bukan optimasi yang diterapkan agar arm tertentu menang.

Alokasi total pada pengukuran transaksi sekitar **315–332 MiB versus
3124–3236 MiB**; ini cumulative allocation, bukan resident RAM. RSS proses
inferensi sekitar **385–454 MiB**, termasuk runtime, source/checkpoint/WAL,
data, dan efek GC; perbedaan antar-arm tidak monoton. Live heap, RSS/HWM sebelum
dan sesudah learning/inferensi disimpan. VmHWM adalah batas inklusif proses,
bukan puncak RAM fase learning yang tersampel secara terpisah.

Checkpoint sekitar **13,10 MiB versus 13,27–13,36 MiB**, full WAL sekitar
**49,86 MiB** di kedua arm; suffix baru sekitar 3,92 MiB. Ukuran journal sama
karena mencatat pengalaman/proposal, bukan menyalin seluruh hasil parameter
setiap command. Recovery sekitar **3,08–3,57 detik**. Nilai tepat dan biaya
checkpoint/replay tersedia di [resource-summary.csv](evidence/resource-summary.csv)
dan artefak. Tambahan 8 neuron/330–548 sinaps tidak menghasilkan keuntungan
prediktif konsisten untuk membayar biaya tersebut. Tidak ada alasan biaya/manfaat
untuk memilih 64 sebagai default dari studi ini.

## 13. Persistence, replay dan koreksi pemeriksaan

Keenam arm final memiliki sequence **84799**. Checkpoint, native checksum,
restart predictions, full suffix replay dari **80735**, rollback ke sumber,
signed parameters, original WAL prefix, policy, feature metadata dan witnesses
lulus. Reader baru juga mereplay seluruh R2 historis 76617→80681 **exact**.
Checkpoint lama tetap didukung; binary lama tidak dijanjikan menerima >32.

Pemeriksaan awal seluruh struktur Go `NResolutionState` menghasilkan false
meski semua state bytes exact. Audit independen membuktikan satu-satunya
perbedaan adalah `Last.Trials[].RejectedBy` nil versus slice kosong panjang nol,
yang sama-sama dihilangkan JSON. Semua fitur/parent/head/source/control/capacity
exact secara struktural; seluruh serialized NResolution dan full state exact.
Koreksi terverifikasi hanya memperbaiki **pengukuran metadata**, tidak Brain
atau kriteria ilmiah. Initial persistence/analysis negatif, exact mismatch paths,
hash dan amendment disimpan. Analisis confirmatory dijalankan kembali dengan
**SHA kode yang sama**, tanpa evaluasi ulang holdout. Lihat
[PERSISTENCE_COMPARISON_AUDIT.md](PERSISTENCE_COMPARISON_AUDIT.md).

## 14. Keputusan tiga hipotesis

| Hipotesis | Keputusan | Dasar dan batas |
| --- | --- | --- |
| H1 kapasitas tidak cukup | **Tidak terkonfirmasi** | Semua arm 64 merekrut 8 fitur, tetapi efek independen tidak konsisten dan normal memburuk. Tidak menolak kemungkinan kapasitas membantu tugas/jadwal lain. |
| H2 representasi tidak efektif | **Belum terisolasi** | Akses history pada kontrol 32 bermasalah; saturation dan rejections terukur. Namun fitur baru tidak sekadar duplikat dan bermanfaat bagi model yang sudah menyesuaikan readout. Tidak ada intervensi reorganisasi terkontrol untuk menetapkan sebab utama S2. |
| H3 informasi prediktif kurang | **Belum terkonfirmasi** | Efek history S2 sangat kecil pada kedua kapasitas, tetapi positive control 32 gagal. Ketidakgunaan model tidak sama dengan ketidakadaan informasi. |

Root cause yang **terbukti** adalah ceiling menghentikan kesempatan recruitment,
bukan readout learning; kontrol 32 mempunyai bottleneck akses representasi
temporal dalam konfigurasi yang diuji. Root cause kemampuan **S2 aktual** belum
dibedakan secara meyakinkan antara pemakaian representasi dan informasi yang
tersedia. Karena itu status utama tetap **INCONCLUSIVE**, bukan PASS atau
DIAGNOSED yang dipaksakan.

## 15. Evaluasi kemampuan S2

Seluruh enam model tetap **PARTIAL** menurut kriteria yang disegel. Tidak ada
yang memenuhi gate normal RMSE ≤0,8×constant dan korelasi ≥0,5; normal correlation
sekitar 0,166–0,294. Satu stream bergeser dapat membaik tanpa membuktikan seluruh
kemampuan. Semua calibration, context gaps, readout ablation dan kegagalan
tersimpan di [actual-S2-ability.json](evidence/actual-S2-ability.json).
Hasil historis R2 (repair PASS, ability PARTIAL, RMSE 0.00957043) dipertahankan
apa adanya, tidak diklaim berubah menjadi PASS karena jumlah fitur.

## 16. Implikasi adaptive neural capacity

**Belum siap mengaktifkan adaptive capacity controller final.** Penelitian
memberi alasan untuk mempertahankan batas/default sekarang sambil menyelidiki
kualitas proposal dan kanal temporal. [ADAPTIVE_CAPACITY_REQUIREMENTS.md](ADAPTIVE_CAPACITY_REQUIREMENTS.md)
merinci bukti neural, keuntungan aktual, biaya dan retensi untuk **reuse,
growth, reorganisasi, konsolidasi, pruning, dan mempertahankan graph**.

Pengalaman berikutnya perlu mengikat prestate, tindakan native yang benar-benar
dijalankan, perubahan error pada pengalaman independen, biaya aktual dan retensi
tertunda ke native policy/causal credit pada N yang sama. Korelasi, jumlah node,
atau perbaikan Check lokal saja tidak cukup untuk mengajari kebijakan bahwa
growth menguntungkan. Konsolidasi/pruning memerlukan joint ablation dan
equivalence lintas konteks, bukan menghapus fitur setelah satu tes.

## 17. Keterbatasan ilmiah

Tiga seed dan 512 kasus per stream memberikan informasi tentang efek moderat,
bukan jaminan power untuk semua efek 2%. Ceiling 64 hanya diuji sampai 40 fitur.
Perintah growth maksimal satu per lesson dan top-eight ranking tidak diubah.
Kontrol positif 32 gagal; tidak ada klaim mutual information nol atau seluruh
arsitektur salah. Novelty adalah proyeksi linear deskriptif tanpa label, bukan
uji independensi informasi lengkap. Clamps dapat keluar dari distribusi;
permutation mempertahankan konteks terbatas, bukan semua variabel. Retention
bank dan oracles terbatas. Timing/RAM mencakup overhead proses dan profiling;
Android hanya dibuild, tidak dibenchmark. Dev-role discrepancy dan supplementary
diagnostic dilaporkan terbuka; tidak dikoreksi dengan tuning holdout.

## 18. Pengujian engineering

`go test ./...`, `go vet ./...`, `go build ./...`, tujuh package race tests,
capacity admission/checkpoint/replay/invalid-option tests, tagged harness
compilation, Android arm64 production/chat builds, original R2 replay dan seluruh
fase penelitian berhasil. Checksum/metadata audit menangkap dan menjelaskan
kesalahan perbandingan harness tanpa mengurangi gate state. Log, CPU profiles,
binary hashes dan lingkungan disimpan; CI GitHub dilaporkan terpisah pada PR
setelah push. Berhasilnya pengujian engineering tidak mengubah hasil ilmiah.

## 19. Artefak dan reproduksi

[RESTORE.md](RESTORE.md) memuat pasangan checkpoint/journal setiap arm dan
instruksi recovery native. [ARTIFACT.md](ARTIFACT.md) menyediakan file pemulihan
lengkap beserta ukuran, SHA-256 dan manifest. Artefak mencakup **seluruh**
checkpoint per blok/final/restart, full journals termasuk prefix historis,
receipts/trials, observasi/provenance, seluruh masks/permutations/clamps,
kontrol positif/negatif, hasil gagal, konfigurasi/seal, profiles, test logs,
skrip reproduksi dan arsip asli R2 yang tidak dimodifikasi. File sungguhan
disertakan, bukan hash-only stubs. Source implementation dipin ke commit
penelitian dan delta terhadap #76.

## 20. Langkah penelitian selanjutnya

Prioritaskan diagnosis native yang memisahkan ketidakpekaan activation terhadap
history, invalid metric fit/signed bounds, dan shared-protection rejection
menggunakan kontrol temporal yang sama dan salinan lineage yang sama. Uji
reuse/reorganisasi kecil versus growth dengan budget tetap, sebelum memberi
native policy tindakan kapasitas adaptif. Bila menguji lebih dari 40 fitur,
daftarkan budget lesson dan holdout baru terlebih dahulu; jangan memperpanjang
jadwal ini karena hasilnya buruk. Pertimbangkan lingkungan native dengan sebab
temporal persisten yang benar-benar diobservasi, tanpa memasukkan jawaban ke
fitur atau memakai label sintetis sebagai kemampuan S2.

Tidak ada perubahan `main`, merge #74/#75/#76, promosi Brain penelitian,
controller alternatif, penghapusan state historis, atau pelatihan S3.
