# HORIZON — S2-R4 TEMPORAL ERROR-HISTORY LEARNING & CAUSAL DIAGNOSIS

**Diagnosis: keterbatasan representasi serta recruitment/admission pada kapasitas 32. History learning pada arm utama belum memenuhi gate. Kemampuan S2 tetap PARTIAL.**

Horizon belum terbukti belajar menilai history tidak berguna. Pada generator lama, error memang ditentukan oleh state lengkap saat ini dan operasi. Pada kontrol temporal baru, history terbukti menambah informasi setelah seluruh state neuron diperhitungkan, tetapi arm 32 belum memanfaatkannya. Arm pendukung 64 memperoleh manfaat history yang terukur; seluruh gate kemampuan/retensi tetap belum terpenuhi. Main dan Brain produksi tidak diubah.

## 1. Source verification

Repository `mpsprintit-spec/horizon-artificial-brain`; main `11c6e0a78547d68a0be8f78b3c801e912722608d`; HEAD R3 `6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c`. Arsip R3 SHA-256 `0137ba463fd78e85b130cba5213d37747582cf32ddf4acf6757c9d5d99a9d6f5` cocok. **653/653 payload R3 dan 443/443 payload R2** terautentikasi. Native source checkpoint `s2r2/s2r2-continuation.json`, journal lengkap `s2r2/training-combined/events.jsonl`, sequence **80735**, checksum **9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3**.

Recovery sumber exact, tidak menambah sequence. Inventory: **274 unit neural, 6.408 sinaps, 32 fitur numerik**, policy updates 10.752. Sumber tidak diubah. Source ilmiah primary: **cee0fa80dfa1667ee2738c3467463eb1fb8313ab**. Angka 32 adalah ceiling fitur, bukan total neuron.

## 2. Brain lineage dan tujuan

**Canonical A–E1 → S1 → S2 → S2-R2 → S2-R4.** Setiap arm dan revisi eksperimen dimulai dari sumber R2 yang sama. Tidak memilih arm R3/R4 terbaik, mencampur parameter, membuat Brain baru untuk menggantikan sumber, atau melatih ulang A–E1/S1/S2/R2. Tujuan lokal: membedakan informasi yang tersedia, direpresentasikan, dipelajari dan bermanfaat untuk prediksi error berikutnya.

## 3. Error-history signal trace

Native operation aktual → proyeksi poststate aktual → selisih dengan forecast S1 → magnitude tiga residual → buffer pengalaman → input berikutnya → fitur N → readout S2. S1 memprediksi tiga proyeksi poststate; setelah mengurangkan prestate yang sama, residual identik dengan error prediksi perubahan.

Magnitude pada koordinat **59/60/61** memakai source neuron **270/271/272**; presence/action **63/64** memakai **273/274**. Query S2 = **257–260**. Fitur sumber **266** satu-satunya jalur langsung history: axis `.00290446446/.00035468987/.00216034615`, center `.343707303`, delta `3.61199957e-6`, readout metrik `−.0296251977`. Axis bercampur dengan state/operasi; keberadaan kanal tidak membuktikan akses yang memadai.

History berisi tiga komponen error **satu langkah sebelumnya**, bukan tiga lag. Magnitude menghilangkan tanda residual. Adapter membawa history/previous state antaroperasi; ledger dan data menyimpan provenance. Penyimpanan/pemulihan history secara otonom dari memori neural belum diuji. Tidak dibuat Error-History Engine.

## 4. Temporal information audit dan koreksi full-state

Reeksekusi **256 kasus historis** native menghasilkan max delta proyeksi/error **0**, tanpa neuron aktif yang tidak tercakup prestate. Dengan graph/gain/waktu tetap dan forecast S1 diberikan, current full state + operation menentukan error; raw history kondisionalnya redundan. History masih mungkin menjadi proxy bila representasi learned kehilangan state yang cukup. Ini bukan learned rejection.

Audit awal memakai sembilan rata-rata populasi. Manfaat 26,28% pada revisi lingkungan ke-3 **tidak sah sebagai bukti informasi melampaui seluruh state**. Audit tambahan development dengan 36 nilai dan spatial contrasts memberi hanya 0,276%, CI95 [−1,568%,1,372%]. Seluruh data, skor dan koreksi tetap diarsipkan.

Revisi primary menambah stimulus independen per neuron agar selisih spasial tidak langsung membocorkan gain lampau. Pembanding mengondisikan seluruh 36 nilai secara invertibel melalui means+contrasts, previous state, forecast dan flags. Fit-only scaling; 8 stream Fit, 8 Check, 8 audit; dua langkah awal dikeluarkan. Polynomial/RBF comparators hanya offline, tidak masuk Brain atau menyuntikkan sinaps.

Pada audit independen primary, RMSE current-only **.07628616** menjadi **.05720055**, manfaat **25.01844%**, CI95 **[19.28215%,30.53301%]**. IID: **1.90429% [-1.13014%, 4.44850%]**, belum menunjukkan manfaat. Gate development ≥10% tidak diturunkan.

## 5. Native experience generation

Lingkungan mengubah gain propagasi **aktual**: persistent centers `.10/.97`, jitter `.025`, redraw probability `.07`; IID redraw independen. Stop probability `.18`. Fresh population drive ditambah independent per-neuron perturbation lebar `.08`, clipped [.001,.9]. Prestate meneruskan `.02` poststate aktual dan `.98` fresh drive. Shifted gain `.17/.92`, jitter `.035` dan rentang drive diperlebar. Gain/regime tidak menjadi fitur, target bantu, atau jawaban prediksi.

Source S1 tetap sama. Native episode ditutup per operasi sesuai max-step policy; outer stream mempertahankan poststate, previous prestate dan history target operation. Closing stop tidak mengubah state. Training per seed: 16 stream ×16 langkah pada masing-masing kondisi persisten/IID. Split selalu seluruh stream berbeda, tidak random antar-langkah.

## 6. Actual-error provenance dan leakage

Pre-operation input/forecast di-fsync sebelum target command. Ledger mengikat ID, stream, previous ID, before-sequence dan input SHA-256. Label hanya mean absolute difference forecast S1 dan proyeksi hasil native Step. Target/history linkage development: **0 kegagalan**. Actual poststates, work counts, environment parameters dan journal disimpan. History direset hanya pada batas stream.

Waktu simulasi primary memakai tick **1 ns yang meningkat**, terpisah dari pengukuran wall latency. Ini mengendalikan passive ageing tanpa mengubah decay law atau threshold. Tidak ada klaim operasi sungguhan berjalan satu nanodetik.

## 7. Representation sensitivity dan accessibility

Pada history-available development, penghilangan magnitude dengan presence/action tetap menghasilkan max activation change **0**, sedangkan metric change **0.0015242638**. Activation fitur 266 jenuh; basis metrik masih membawa variasi lemah. Max intervensi yang mencakup langkah pertama pada audit lama tercampur perubahan presence; angka tersebut tidak dipakai sebagai sensitivitas magnitude-only.

Selama axis sumber tetap, hanya fitur 266 bergantung history. Pada basis metrik berlaku finite-change bound `|Δy| ≤ 7,99 × Σ|a_h Δh| + 7,99 × delta`. Untuk perubahan tiap komponen ≤1, bagian axis-nya sekitar **0.04330181**, ditambah toleransi quantization sekitar `2,89e-5`; readout aktual jauh lebih kecil. Resolution refinement saja tidak menambah arah history independen. Ini batas representasi pada bentuk tersebut, tanpa menyatakan semua informasi hilang. Accessibility probes, activation/code/metric arrays dan kontribusi readout tersedia lengkap.

## 8. Feature recruitment diagnosis

Di **32/32**, learner melewati new-direction/product recruitment; readout dan resolution masih dicoba. Di bawah ceiling, residual-direction serta product candidates memakai Fit, parent bervariasi, centered/normalized covariance dan top-eight ranking. Traces menyimpan kandidat yang benar-benar diuji. Tidak ada widened-top-eight ablation; cutoff itu tidak ditetapkan sebagai sebab tunggal.

Refinement penelitian menawarkan kind-1 leaf yang tidak dipakai head lain maupun child. Pada B32: **72** trial, **54** Check-improving terhadap error awal, **51** di antaranya mencatat penolakan retensi (**51** dengan alasan retensi saja). Command yang memilih refinemen: **0**. Peluang proposal bukan bukti utility telah dipelajari.

## 9. Numerical admission diagnosis

| Alasan rejection (dapat bertumpang tindih) | Trial |
|---|---|
| retention-regression | 7817 |
| cross-query-representation-change | 4992 |
| no-check-improvement | 2997 |
| fit-invalid-or-parameter-bound | 16 |
| functionally-duplicate | 15 |


Alasan bertumpang tindih; trial bukan sampel independen. Solver mencatat **72** signed-bound failures dan **9192** valid fits. Nilai pivot, peak coefficients, basis, axis/parents, fraksi, parameter hashes dan failed proposals tersimpan. Tidak menurunkan precision/retention, mematikan shared protection atau mengubah signed bounds.

## 10. Empat hipotesis dan root cause

| Hipotesis | Temuan |
|---|---|
| H1 information limitation | Didukung untuk generator historis lengkap/deterministik. Tidak cukup menjelaskan primary persistent: raw history memberi manfaat 25,02% setelah full-state conditioning. |
| H2 representation failure | Activation tidak berubah akibat magnitude-only; metric channel lemah/tercampur dan readout berbatas. Informasi direpresentasikan sebagian, belum dalam bentuk yang memadai pada 32. |
| H3 recruitment/admission failure | Ceiling menghentikan recruitment; kandidat refinemen dapat terhalang retensi. Kapasitas pendukung memperoleh fitur history dan manfaat causal. Gate yang melindungi kemampuan lama tidak dinyatakan salah. |
| H4 learned low utility | Tidak terbukti untuk kegagalan utama. Prasyarat akses serta kesempatan pembelajaran yang aman belum terpenuhi. Tidak memakai history bukan bukti telah belajar menganggapnya tidak berguna. |

Keterbatasan H2/H3 diterima pada cakupan mekanisme ini; sebab tunggal semua kegagalan tidak diisolasi. Dataset memakai one-lag history; learned multi-lag storage dan per-case utility selection otonom belum ditunjukkan.

## 11. Minimal repair dan controlled revisions

Opt-in leaf-direction refinement menormalkan arah residual Fit pada fraksi .25/.5/1, tidak menyebut kanal history atau menetapkan bobot jawaban. Search tetap 32, shared directions/parents tidak diganti, native gates tetap. Seluruh kanal axis lama termasuk nol ditulis ulang bila kandidat diterima. **Repair ini belum memperbaiki history ability utama.**

Validator protokol diagnostic baru dan mask evaluasi pada absent-history boundary diperbaiki dengan regresi. Sumber/model tidak dipilih ulang; unscored preflight data dipisahkan dan source refreeze dilakukan sebelum data pengganti. Full-state correction memulai tahap primary baru dari sumber R2 yang sama, mempertahankan seluruh eksperimen sebelumnya. Lihat tiga protocol documents dan FULL_STATE_PROTOCOL untuk kronologi lengkap.

## 12. Native training dan confirmatory results

Seed **37/83/151**, delapan arm, empat command/arm, masing-masing 32 Fit +32 Check. **96 kesempatan belajar; 88 native commits.** Pembaruan melalui LearnCompositionWithDiagnosis pada signed N, bukan Python. A32 baseline; B32 attempted repair; C32/B0-32 magnitude masked; D32/D0-32 IID controls; A64/C64 supportive capacity controls. Magnitude mask mempertahankan presence/operation. Old witness bank dan incumbent knowledge tetap identik; control membatasi history pengalaman baru, bukan menghapus semua pengetahuan history sepanjang hidup.

Source/checkpoints/seed/sample/threshold/mask/analysis dibekukan sebelum sole primary holdout. Final seeds **3500037/3600037/3700037**, 32×24 per kondisi: **768 kasus/kondisi, 2.304 pengalaman aktual**. Pengulangan prediksi tiga model seed tidak dianggap tambahan pengalaman independen. Final selalu lesson 4, tanpa memilih checkpoint terbaik.

| Arm | RMSE persisten | RMSE IID | RMSE bergeser | Fitur seed 37/83/151 |
|---|---|---|---|---|
| A32 | 0.11256269 | 0.10356005 | 0.09196549 | 32,32,32 |
| B32 | 0.11256269 | 0.10356005 | 0.09196549 | 32,32,32 |
| C32 | 0.11255389 | 0.10355492 | 0.09195822 | 32,32,32 |
| B0-32 | 0.11255389 | 0.10355492 | 0.09195822 | 32,32,32 |
| D32 | 0.11206677 | 0.10346452 | 0.09140514 | 32,32,32 |
| D0-32 | 0.11205353 | 0.10345609 | 0.09139410 | 32,32,32 |
| A64 | 0.09498660 | 0.09140326 | 0.07625386 | 36,36,36 |
| C64 | 0.10265526 | 0.09520358 | 0.08322445 | 36,36,36 |


| Kontras (efek positif = manfaat) | Persisten: efek [CI95] | IID | Bergeser |
|---|---|---|---|
| A32-vs-C32 | -0.00781% [-0.01225%, 0.00000%] | -0.00495% [-0.00801%, 0.00000%] | -0.00791% [-0.01243%, 0.00000%] |
| B32-vs-B0-32 | -0.00781% [-0.01225%, 0.00000%] | -0.00495% [-0.00801%, 0.00000%] | -0.00791% [-0.01243%, 0.00000%] |
| A64-vs-C64 | 7.47031% [0.50306%, 16.03693%] | 3.99179% [-0.62174%, 10.80395%] | 8.37565% [0.56718%, 17.88446%] |
| D32-vs-D0-32 | -0.01181% [-0.01208%, -0.01150%] | -0.00815% [-0.00867%, -0.00757%] | -0.01208% [-0.01246%, -0.01162%] |


CI: 2.000 paired hierarchical bootstrap atas training seeds dan seluruh stream independen, preserving order/common test cases. Tiga seed membatasi presisi. MAE, correlation, calibration, error-discrimination AUROC serta setiap stream/seed tersimpan; RMSE bukan satu-satunya metrik.

## 13. History ablation dan learned attribution

| Arm; history tersedia | Magnitude ablation: manfaat full [CI95] | Conditional permutation: manfaat full [CI95] |
|---|---|---|
| A32 | -0.00795% [-0.01230%, 0.00000%] | -0.00220% [-0.00352%, 0.00000%] |
| B32 | -0.00795% [-0.01230%, 0.00000%] | -0.00220% [-0.00352%, 0.00000%] |
| A64 | 6.47288% [1.80987%, 11.01462%] | 2.58560% [0.56508%, 4.44280%] |
| D32 | -0.01024% [-0.01111%, -0.00949%] | -0.00287% [-0.00334%, -0.00232%] |


History-available subset **736 kasus/kondisi**. Efek positif membandingkan full terhadap RMSE intervensi. Primary B32 tidak memperoleh manfaat. Pada A64, penghilangan magnitude merugikan dan efek berulang pada tiga seed: bukti pemanfaatan yang bermanfaat pada arm pendukung, bukan kelulusan semua gate.

## 14. Conditional permutation

Sepuluh target-blind donor maps memakai exact operation/previous/history operation/presence, donor dari stream lain, delapan nearest neighbors pada **57 continuous coordinates** yang mencakup full current state, previous state dan forecast; scaling dari Fit. Seeds 3100031–3100040. Distance/balance dan semua arrays tersimpan.

A64 persistent conditional effect **2.58560% [0.56508%, 4.44280%]**; pada IID interval mencakup nol. Matching adalah aproksimasi, bukan exact conditional-randomization test. Zero ablation juga menggeser distribusi, sehingga tidak ditafsirkan sendirian. Clamp fitur hanya pada evaluation copies, tanpa mengubah checkpoint.

## 15. Informative versus noninformative control

Hidden IID gain independen dari history setelah current full state diketahui; audit finite-sample tidak menunjukkan manfaat yang meyakinkan. D32 versus D0-32 pada IID **-0.00815% [-0.00867%, -0.00757%]**. Tidak dipaksakan ketergantungan history. A64 permutation membantu pada persistent dan bukan IID; itu mendukung nilai informasi yang berbeda, tetapi belum membuktikan kebijakan perhatian/utility yang otonom pada semua kondisi.

## 16. Generalization dan gate kemampuan

B32 versus B0-32 shifted: **-0.00791% [-0.01243%, 0.00000%]**. A64 versus C64 shifted: **8.37565% [0.56718%, 17.88446%]**. Sebagian pemanfaatan history bergeneralisasi pada arm pendukung. Overall persistent gain A64 **7.47031%** masih di bawah gate terdaftar ≥10%, dan retensi S2 tidak lolos semua seed. Default 32 tidak diubah; tidak memilih arm 64 terbaik sebagai penerus kanonik.

## 17. A–E1 / S1 / S2 retention

S1 maximum prediction shift **0**. A–E1 maximum oracle delta **4.8978599e-12**, **24/24** lolos batas 1e-7; C dan E1 counts ada pada setiap oracle. Policy exact **True**, old witnesses exact **True**, policy updates tetap 10.752. Native owner, signed parameters, eligibility/credit infrastructure dipertahankan. Ini numerical S2 learning; policy tidak dilatih ulang.

Sumber 80735 diukur langsung RMSE **0.01682331** pada 256 kasus R2, limit eksternal **0.01766457**. Referensi laporan R2 `.00957043` tidak disamakan dengan score continuation sumber. Internal witness gate tidak menjamin seluruh kasus lama: sebagian arm gagal external S2 retention, dan semua kegagalan disimpan.

| Arm | RMSE S2 historis seed 37/83/151 | S1/S2 lolos | A–E1 lolos |
|---|---|---|---|
| A32 | 0.01644600, 0.01912813, 0.01559954 | 2/3 | 3/3 |
| B32 | 0.01644600, 0.01912813, 0.01559954 | 2/3 | 3/3 |
| C32 | 0.01645220, 0.01912813, 0.01560645 | 2/3 | 3/3 |
| B0-32 | 0.01645220, 0.01912813, 0.01560645 | 2/3 | 3/3 |
| D32 | 0.01500362, 0.01769861, 0.01581237 | 2/3 | 3/3 |
| D0-32 | 0.01500301, 0.01769940, 0.01581166 | 2/3 | 3/3 |
| A64 | 0.03094278, 0.03489841, 0.01544725 | 1/3 | 3/3 |
| C64 | 0.01648432, 0.03566748, 0.01534397 | 2/3 | 3/3 |


**15/24** final arm lolos S1/S2 eksternal. Tidak ada promosi, walau beberapa kontrol membaik. Revisi sebelumnya dengan millisecond ticks mengalami drift A–E1 ~4.9e-6 akibat passive memory time; raw failure tetap ada. Tiga baseline exemplar elapsed-time controls mencapai delta sekitar 2.2–4.4e-16. Audit luas awal timeout 20 menit dan partial control copies tetap disimpan. Primary memakai controlled nanosecond simulation agar ageing tidak menjadi confound; threshold/law tidak diubah.

## 18. Computational costs

| Arm | Native commits /12 | Median p50 / p95 inference (ms) |
|---|---|---|
| A32 | 10 | 0.139116 / 0.274777 |
| B32 | 10 | 0.135300 / 0.291562 |
| C32 | 10 | 0.135611 / 0.321726 |
| B0-32 | 10 | 0.142030 / 0.338432 |
| D32 | 12 | 0.143022 / 0.321666 |
| D0-32 | 12 | 0.135862 / 0.316679 |
| A64 | 12 | 0.160718 / 0.328627 |
| C64 | 12 | 0.162040 / 0.352942 |


GOMAXPROCS=2/proses. Training seed dan recovery per-model independen dapat paralel; latency adalah wall time teramati, bukan isolated-host throughput. Cost JSON memuat allocations, heap/RSS, recovery dan inventory; journal/checkpoint lengkap tetap disimpan. Ceiling 64 hanya mencapai 35–36 fitur, bukan jaringan yang sudah mencapai 64.

## 19. Persistence, engineering checks dan artefak

**24/24** full checkpoints: restart predictions exact, full suffix replay dari 80735 exact, complete canonical/serialized state exact, source rollback exact dan historical WAL prefix utuh. Reader juga mereplay R2 76617→80681 exact. Tidak ada pemotongan journal. Fresh behavioral evaluation images hanya menghilangkan old adapter dedup index/anchor pada salinan, mempertahankan seluruh Brain/activation/policy/sequence; parity terhadap full oracle = **0**. Images tidak untuk recovery; original full checkpoints/WAL selalu dipakai recovery.

Go tests/vet/build, tagged vet, protocol/mask/recovery regressions dan Android arm64 builds lulus. Full knowledge/activation race lulus; full runtime race timeout global 10 menit, bukan PASS. Targeted Native/Metric/NMetric runtime race lulus. Dependency preflight sebelum prediction files siap, unknown-protocol/mask failures, full-state correction dan seluruh hasil negatif disimpan; tidak ada skor digunakan untuk memilih model.

Arsip lengkap memuat primary serta semua revisi, checkpoints, full WAL, receipts, observations/poststates, pre-operation ledgers, feature/synapse traces, masks/donors, failed proposals, negative controls, per-seed/per-stream scores, source/config hashes, original authenticated R3 archive, scripts/patch dan MANIFEST SHA-256. RESTORE/ARTIFACT memberi pemulihan serta file persisten, tidak hanya hash atau path sementara.

## 20. Scientific decision dan next step

| Penilaian | Keputusan |
|---|---|
| Tujuan | Memisahkan kandungan informasi, representasi, recruitment/admission dan learned low utility. |
| Implementasi | Actual temporal neural experiences dan native signed-synaptic learning berjalan pada satu Brain/N; full recovery tervalidasi. |
| Pengujian | Full-state conditional information tersedia. Default32 gagal memperoleh manfaat; supportive64 memperoleh manfaat causal tetapi tidak memenuhi semua threshold/retensi. |
| Keputusan | **H2/H3 sebagai keterbatasan mekanisme didukung. Primary history gate belum lulus; S2 PARTIAL. Learned low utility belum terbukti.** |
| Next step | Cari representasi temporal/contextual yang retention-safe pada budget tetap, audit kandidat Check-improving yang tertahan retensi, dan perkuat cakupan anchor S2 lama. Jangan menurunkan threshold, memaksakan history, memilih checkpoint64 terbaik, atau memulai S3. |

Pengetahuan lama, informasi yang tersedia dan kemampuan memakai informasi adalah tiga persoalan berbeda. Hasil ini tidak membenarkan menyebut history tidak berguna secara umum, maupun menyatakan Horizon sudah belajar menolaknya. Main dan Brain produksi tetap tidak berubah; satu PR penelitian non-draft ditujukan ke branch S2-R3 tanpa merge otomatis.
