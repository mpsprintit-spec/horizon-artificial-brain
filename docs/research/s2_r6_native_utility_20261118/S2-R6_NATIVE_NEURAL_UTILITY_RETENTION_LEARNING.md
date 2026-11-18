# S2-R6 — Native Neural Utility, Retention & Adaptive Plasticity Learning

**Tanggal:** 18 November 2026 (Asia/Jakarta). **Verdict: PARTIAL — UTILITY PREREQUISITE NOT MET.**
Repository: `mpsprintit-spec/horizon-artificial-brain`; branch `research/s2-r6-native-utility-retention-20261118`.
Main baseline: `11c6e0a78547d68a0be8f78b3c801e912722608d`.
Basis kode R5: `099ae285737cdc62fc121791ba4e6f4e463a1cb3`.
Source training: `a2391109eecddf1b13f6f9c8ae45bdf208609b09`; source/analysis confirmatory: `c068d827b1b8c2cd6edb701b8e9b16dc747ea120`.
Main dan Brain produksi tidak diubah. Tidak ada merge PR sebelumnya atau arm penelitian yang dipromosikan.

## 1. Executive scientific verdict

Tujuan yang diuji adalah kemampuan N mempelajari manfaat suatu perubahan neural dari intervensi aktual. Implementasi menyediakan pengalaman kausal dan readout utility pada 32 fitur N yang sama. Seluruh enam seed menghasilkan 48 model final penelitian; semua merupakan salinan dari sumber R2 yang sah.

**H1: FAIL; H2: NOT RUN; H3: NOT RUN; H4: NOT RUN.** H1 tidak lulus development independen sehingga tahap interference/adaptive plasticity/consolidation tetap ditahan. Hasil final tidak dipakai untuk membalik keputusan prasyarat tersebut. Tidak ada overall PASS.

Pada final unseen pathways, native utility RMSE 0.02463850 versus baseline terbaik 0.00205881, manfaat -1096.736%, CI95 [-2439.408%, -1.189%]. Perbaikan Check selama training bukan bukti generalisasi.

Manfaat history C/B terhadap matched B0 adalah 2.401%, CI95 [0.100%, 5.342%]; calibration mean-ratio 0.60976. Retensi seluruh kontrak lulus 0/6 untuk C.

Keputusan: **perlu perbaikan**. Utility belum boleh menjadi pengendali plasticity. Hasil safeguard eksternal dipisahkan dari kemampuan yang dipelajari.

## 2. Brain source verification

Arsip R5 lengkap: 1.127.417.584 byte, SHA256 `ee6cadaae0fa122beb40da1908c776262385f8c7e758d814e2bc4e411d9e2531`; 3.272/3.272 payload terautentikasi. Arsip R2 di dalamnya mempunyai SHA256 `e51b61bf41107f4f74e7d72d6360b22cd4671d0f6ce84e867212090290e0aa46`; 443/443 payload diverifikasi.

Checkpoint dan journal R2 byte-identik dengan `r5/00-source.json` dan `r5/source-events.jsonl` terautentikasi. Sequence 80735; native checksum `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`; 274 unit, 6.408 sinaps, 32 fitur dan 10.752 native policy updates. Checkpoint SHA256 `445991d177dc955ec58e7f9a3244a3f27da98fb90de584b09d4cc05c5c82d29d`; journal SHA256 `05659f8f3ea4bbe29fbcdd90a4f1c00ee8fbbe488234ee4af762730926c083d8`. Native historical suffix 80681→80735 exact; activation/learning/DNF memakai pemilik Brain yang sama.

Lineage: Canonical A–E1 → S1 → S2 → S2-R2 → S2-R6. Tidak ada checkpoint final R3/R4/R5 yang dipilih. Kemampuan history dibentuk ulang memakai prosedur D32 R5 pada training baru. C/F meneruskan common prefix B per seed; tidak ada reset di antara blok utility.

## 3. Architecture audit

Audit lengkap tersedia dalam `ARCHITECTURE_AUDIT.md`. Aktivitas rekuren, komposisi numerik, signed parameters, eligibility, policy temporal credit dan paired factual contrast sudah ada. Jalur pengalaman konsekuensi numerical-path intervention ke query utility belum tersedia; adapter riset menyediakan pengukuran tersebut.

Generic shared-basis reuse membentuk query numerik baru pada fitur dan signed substrate N yang sama. V1 memakai fitter lama; V2 memakai solver berbatas R5 pada zero-initialized query. Versi dibedakan agar replay prototype lama tetap exact. Seluruh hasil belajar berada pada ordinary canonical synapses, bukan objek estimator eksternal. Jumlah fitur tetap 32. Query baru dapat menambah empat query units; itu tidak menambah feature cap atau substrate.

Tidak ada Utility Engine, utility threshold, ranking neuron hardcoded, candidate-selection policy baru, permanently frozen general synapses atau controller plasticity. Fixed basis adalah ruang pencarian lesson ini, bukan bukti consolidation. General policy tetap 10.752 updates.

## 4. Native utility experience

Setiap seed mempunyai 512 pengalaman training utility: 256 Fit dan 256 Check, empat blok 64/64. Dua intervensi dari satu operasi asal tetap pada split yang sama. Semua label berasal dari squared-loss difference aktual setelah native numerical counterfactual forwards; actual S2 error berasal dari operasi propagate/stop yang benar-benar dijalankan.

Utility = `(loss_before − loss_after)/4`; faktor empat hanya normalisasi rentang. Tidak ada pemilihan contoh hanya karena bermanfaat. Single/pair pathways menerima signed changes −0.08/−0.02/0/+0.02/+0.08 dan ablation. Semua source/intermediate checkpoints, actual outcomes, source IDs, eligibility dan deltas dipertahankan.

Untuk ablation, gain positif berarti menghapus pathway membantu, sehingga pathway itu mengganggu pada konteks tersebut. Gain negatif berarti penghapusan merusak prediksi. Ini utility perubahan, bukan ranking intrinsik neuron.

## 5. Causal intervention design

Urutan: pre-intervention descriptor → utility forecast native → ledger di-sync → paired counterfactual → actual loss/utility. Input N tidak memuat target, outcome aktual, ID neuron, ID kandidat atau keputusan gate. Descriptor memuat bounded preactivation/metric/weight/delta/eligibility/dependency summaries dan konteks terbatas.

Paired forwards memakai initial signed parameters dan input identik; hanya pathway yang dideklarasikan berubah. Workspace dibuang sesudah pengukuran. Tidak ada parameter berisiko dipasang pada sumber kanonik. Ini uji computational numerical-readout pathways, bukan ablation seluruh aktivitas rekuren atau seluruh fungsi sebuah neuron. Pasangan pathway tidak membuktikan causal decomposition seluruh jaringan.

## 6. Utility learning results

| Holdout | N | Native RMSE | Best baseline | Baseline RMSE | Benefit | CI95 | Macro-F1 |
|---|---|---|---|---|---|---|---|
| seen | 6144 | 0.00603445 | weight_only | 0.00184932 | -226.307% | [-518.730%, -5.157%] | 0.312 |
| shifted | 6144 | 0.02670304 | weight_only | 0.00174323 | -1431.813% | [-2958.758%, -2.348%] | 0.318 |
| unseen | 6144 | 0.02463850 | weight_only | 0.00205881 | -1096.736% | [-2439.408%, -1.189%] | 0.325 |

Gate memerlukan ≥10% manfaat, CI95 lower>0, macro-F1≥0.40, ketiga kelas ≥10 contoh, generalisasi seen/unseen/shifted dan shuffled control tidak memberi manfaat sebanding. Baseline constant, zero, activation-magnitude OLS dan weight-magnitude OLS dilatih hanya pada Fit. Final bersama memakai bootstrap seed dan stream yang dipertahankan lintas model, 10.000 replika.

Neural pathway ordinals mod 3=0 tidak pernah diintervensi pada Fit/Check: 11 unseen pathways versus 21 seen. ID-nya tidak masuk codec. Per-seed/stream/class counts ada dalam `SCIENTIFIC_RESULTS.json` dan seluruh prediksi mentah.

| Seed | Seen utility RMSE | Unseen utility RMSE | Shifted utility RMSE |
|---|---|---|---|
| 37 | 0.00207969 | 0.00214290 | 0.00184075 |
| 83 | 0.00199421 | 0.00206592 | 0.00175442 |
| 151 | 0.00271606 | 0.00216745 | 0.00185652 |
| 211 | 0.01397709 | 0.06015717 | 0.06527624 |
| 307 | 0.00197263 | 0.00225100 | 0.00195522 |
| 419 | 0.00188621 | 0.00219673 | 0.00189450 |

Seed 211 mendominasi kegagalan generalisasi; tidak ada seed replacement, checkpoint retry setelah final atau penghapusan outlier. Hasil ini menunjukkan bahwa admission improvement pada Check tidak menjamin forecast utility reliable di konteks baru.

## 7. Neural interference learning

**NOT RUN.** H1 development gagal sebelum final acquisition. Karena itu tidak ada training/prediksi H2 yang diklaim. Tidak ada label interferensi sintetis dari asumsi bahwa jenis perubahan tertentu selalu merusak. Arm D/E belum dijalankan. Kegagalan prasyarat disimpan sebagai hasil penelitian, bukan disembunyikan sebagai keberhasilan implisit.

## 8. Retention prediction

**Belum terbukti.** C memprediksi manfaat intervensi numerical-path lokal; belum memprediksi perubahan kemampuan lama oleh candidate native learner yang belum dijalankan. Net-benefit multi-kemampuan, delayed effects, novel update magnitudes dan ID-independent interference transfer tidak diuji karena tahap prasyarat belum lulus.

## 9. Native adaptive plasticity

**NOT RUN.** Utility C tidak memengaruhi candidate search, ranking, gate, solver atau installed update. Karena itu tidak ada attribution improvement terhadap native policy adaptation. Arm G nonlearned candidate-selection control juga tidak dijalankan: tahap selection yang hendak dibandingkan belum dibuka.

Pada stage yang berjalan, solver membentuk candidate, native admission memeriksa Check dan retention banks, lalu writer memasang candidate yang lolos. Semua candidate/trials/rejections tercatat. Tidak ada utility selector yang diam-diam ditambahkan. C/F mempunyai S2 predictions persis sama dengan common prefix B pada 512 training contexts, diverifikasi sebelum final.

## 10. Knowledge consolidation

**NOT PROVEN.** Empat utility lesson meneruskan Brain yang sama, tetapi bukan rangkaian empat kemampuan berbeda A→B→C→D. Tidak ada bukti bahwa N telah belajar memilih pengetahuan valid untuk dipertahankan atau salah untuk diperbaiki. Witness gates/shared-dependency guards tetap safeguard eksternal.

## 11. Continual learning sequence

| Arm | Serial blocks/seed | Committed | Opportunities | Changed signed parameter-pairs/seed |
|---|---|---|---|---|
| A | 4 | 23 | 24 | 2962–3126 |
| B | 4 | 22 | 24 | 2884–3140 |
| C | 4 | 8 | 24 | 2556–2728 |
| F | 4 | 4 | 24 | 2392–2592 |

Semua intermediate checkpoints tersedia. Parameter-pairs menjumlahkan excitatory/inhibitory canonical values pada pasangan source/target; bukan menghitung dua sign branches sebagai satu nilai yang saling menimpa. Jumlah ini adalah perubahan struktural terukur, bukan skor consolidation. Forward/backward transfer antar empat kemampuan dan corrective learning belum diuji; matriks tersebut tidak diisi dengan angka rekaan.

## 12. Error-history retention

| Condition | A RMSE | B/C RMSE | A vs A0 | B vs B0 | B CI95 |
|---|---|---|---|---|---|
| persistent | 0.07532283 | 0.07783500 | 5.579% | 2.401% | [0.100%, 5.342%] |
| independent | 0.08676459 | 0.09019372 | -3.049% | -1.212% | [-3.311%, 0.368%] |
| shifted | 0.07060397 | 0.07585088 | 5.112% | 2.239% | [-0.241%, 5.842%] |

Magnitude ablation C: 7.761%, CI95 [1.676%, 15.314%]. Conditional permutation C: 4.629%, CI95 [0.837%, 10.067%].

IID BI vs BI0: -1.173%, CI90 [-3.907%, 0.646%]; praktis equivalence memerlukan seluruh interval di±3%. Nearest-neighbor permutation target-blind memakai 36 observed neurons+previous state+forecast, exact discrete context dan 12 donor neighbors dari stream lain. Conditioning ini aproksimatif. Source R5 benefit 21.45% adalah referensi, bukan nilai yang dipaksakan ke R6.

## 13. Contextual calibration

| Arm | Mean actual | Mean prediction | Ratio | Native RMSE | Fit-constant RMSE |
|---|---|---|---|---|---|
| A | 0.08555161 | 0.06663306 | 0.77886 | 0.07532283 | 0.07760781 |
| B | 0.08555161 | 0.05216609 | 0.60976 | 0.07783500 | 0.07760781 |
| C | 0.08555161 | 0.05216609 | 0.60976 | 0.07783500 | 0.07760781 |
| F | 0.08555161 | 0.05216609 | 0.60976 | 0.07783500 | 0.07760781 |

Tidak ada global scaling/bias yang dipasang ke runtime. Mean ratio bukan seluruh definisi calibration; operation groups, calibration bins, residual quantiles, MAE dan per-seed measurements juga disimpan. Calibration gate tetap memerlukan historical retention. Utility tidak dianggap memperbaiki calibration hanya karena query baru dapat dilatih.

## 14. Historical knowledge retention

| Arm | Historical S2 | New normal | New shifted | Strict S1 | A–E1 | All |
|---|---|---|---|---|---|---|
| A | 5/6 | 0/6 | 5/6 | 6/6 | 6/6 | 0/6 |
| A0 | 5/6 | 0/6 | 5/6 | 6/6 | 6/6 | 0/6 |
| B | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |
| B0 | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |
| BI | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |
| BI0 | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |
| C | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |
| F | 6/6 | 0/6 | 6/6 | 6/6 | 6/6 | 0/6 |

Historical S2 limit 0.01766457. Frozen independent-development source limits: normal 0.00918627, shifted 0.01889719; masing-masing source RMSE+1e-7 pada 128 fresh development operations, sebelum training. Source witnesses 128 tidak dihapus; original anchor 64, normal 64 dan shifted 64 dipertahankan sebagai bank terpisah. Per-bank veto counts ada dalam receipts.

Sampling variation source sendiri pada final dilaporkan pada per-seed retention data; threshold tidak dinaikkan setelah melihat hasil. Coverage berbeda tidak dirata-ratakan sehingga satu kelompok menutupi regresi lainnya. Perbaikan bank adalah safeguard, bukan learned retention.

## 15. Generalization

Utility seen/unseen/shifted dinilai pada 32 independent streams×16 actual operations×dua interventions per model. Holdout tidak digunakan untuk checkpoint selection, fitting atau training lanjut. Transfer di luar numerical-readout pathways, novel magnitudes dan novel continual-learning tasks belum diuji.

Descriptor pasangan memakai mean summaries sehingga dapat kehilangan correspondence antara activity/weight/delta. Sembilan raw coordinates tidak menjamin seluruh konteks/history tersedia. Native activation-basis alias counts di receipts merupakan bukti batas representasi yang diuji, bukan bukti mathematical capacity insufficiency. Feature cap 64 tidak dicoba.

## 16. Causal ablation

H1 menggunakan paired signed interventions, pathway ablation, held-out pathways, untrained-zero baseline dan F shuffled-label control. Hasil aktual tidak diberikan sebelum utility forecast. History mempunyai magnitude masking dan target-blind conditional permutation.

Tidak ada causal ablation H3/H4 karena fungsi selection belum ada. S2 C dan B identik by design; hasil history/retention tidak boleh diatribusikan kepada utility. Solver, coverage dan representation refinement tetap penjelasan terpisah. Interference-information removal belum menjadi uji yang valid karena H2 belum dipelajari.

## 17. Negative controls

| Holdout | C RMSE | F shuffled RMSE | F benefit | C class counts −/0/+ |
|---|---|---|---|---|
| seen | 0.00603445 | 0.00188610 | -1.989% | [2716, 1711, 1717] |
| shifted | 0.02670304 | 0.00177256 | -1.682% | [2767, 1431, 1946] |
| unseen | 0.02463850 | 0.00209429 | -1.724% | [2705, 1628, 1811] |

F merandomisasi relasi input→utility dengan permutasi label aktual terpisah pada Fit dan Check, menjaga marginal labels. Random labels/shuffled outcomes merupakan satu operationalization, bukan dua eksperimen independen. Konflik akibat random labels tetap dilaporkan dan dapat mengurangi admission; tidak dianggap bukti causal regulation. Activation-only/weight-only/constant/zero controls tidak dipasang ke N.

## 18. Performance and resource costs

| Arm | Trials | Median learning s/lesson | S2 p 50 ms | S2 p 95 ms | Utility p 50 ms | Median WAL growth MiB |
|---|---|---|---|---|---|---|
| A | 1560 | 6.732 | 0.1877 | 0.4781 | 0.0000 | 2.107 |
| B | 1560 | 8.343 | 0.1856 | 0.4681 | 0.0000 | 2.694 |
| BI | 1560 | 8.735 | 0.1796 | 0.3895 | 0.0000 | 2.699 |
| C | 624 | 1.252 | 0.1916 | 0.4173 | 0.2101 | 2.989 |
| F | 494 | 1.045 | 0.1792 | 0.3877 | 0.1802 | 2.989 |

Learning time mencakup native search/admission/journal transaction. Utility latency hanya native utility read, bukan keseluruhan counterfactual acquisition atau ledger sync. Seluruh per-call inference durations, memory snapshots VmRSS/VmHWM, heap counters, actual inventories, checkpoint sizes, full WAL bytes dan recovery costs disimpan. Host dipakai bersama tiga research processes, GOMAXPROCS2 per proses; angka bukan CPU-isolated benchmark atau Android speed.

Exact neural operation counts untuk seluruh instrumentation tidak diukur; candidate counts dan circuit/synapse deltas bukan pengganti FLOP count. Active-neuron runtime work tercatat pada environment/WAL untuk operasi akuisisi. Android arm 64 service/chat cross-build lulus; perangkat fisik dan cross-architecture exact replay tidak diuji.

Brain knowledge berada pada canonical neurons/signed synapses/topology. Transient optimization evidence berada pada workspace. Retention witnesses/banks terpisah dari knowledge. Raw experiences dan research logs dipertahankan untuk audit, full WAL untuk recovery; inference utility tidak membaca tabel seluruh pengalaman mentah.

## 19. Persistence and recovery

**48/48** final models lolos exact restart, full suffix replay 80735→final, canonical/signed reconstruction, original rollback dan unchanged historical WAL prefix. Median full-suffix recovery 33.669 s. Suffix replay mengeksekusi ulang native learning; jangan menyamakannya dengan checkpoint-only restoration.

Prototype V1 dan V2 masing-masing empat C/F development models juga diverifikasi exact restart/full suffix. Kedua protocol tetap didukung. Tests/build/vet, tagged metric/native regressions, targeted race dan Android arm 64 cross-build lulus. Unit baru menguji new-query fixed-basis recovery pada kedua protocol, old-function exactness dan checkpoint/full-suffix reconstruction.

Arsip recovery menyimpan source/intermediate/final checkpoints, full historical+new WAL, pre-operation/ intervention ledgers, actual poststates, utility predictions, eligibility, signed deltas, candidates/rejections, witness banks, locks, source hashes, negative development dan complete R5 source archive. `RESTORE.md`, `BUNDLE.json` dan manifest SHA256 menjelaskan recovery. Pengunduhan kembali/hash luar disajikan dalam owner delivery note.

## 20. Failed experiments

1. V1 development: unconstrained metric fitting mencapai coefficient di luar signed bound; activation fitting tidak membaik pada independent Check. Semua receipts/checkpoints/WAL tetap ada.
2. V2 development: bounded solver menghilangkan ketergantungan pada satu unconstrained solution, tetapi utility belum mengalahkan baseline independent seen/unseen. Tidak ada threshold yang dilonggarkan.
3. Beberapa lesson C/F diterima pada Check; final evaluation diperlukan untuk menguji apakah gain bertahan. Conflicting/shuffled labels dan no-check-improvement tetap dihitung.
4. H2/H3/H4 serta G dihentikan oleh preregistered development prerequisite, bukan oleh source corruption. Tidak ada hasil yang direka untuk tahap yang tidak dijalankan.
5. Descriptor summaries, fixed existing basis, finite optimizer budget, approximate permutation dan limited new-distribution contract tetap batas inferensi. Tidak satu pun membuktikan cap 32 mustahil secara matematis.

6. Analyzer semula terhenti pada `trials:null`. Companion compatibility membaca nilai itu sebagai nol kandidat dalam memori, tetap menghitung rejected opportunity, tanpa mengubah source analyzer yang disegel, raw data, kriteria atau hasil native. Traceback dan penjelasan disimpan pada `ANALYSIS_COMPATIBILITY.md`.

## 21. Scientific decision per gate

| Gate | Status | Makna |
|---|---|---|
| A | PASS | authenticated R2/R5 source, source hashes, six frozen seeds |
| B | FAIL | independent utility forecasts vs Fit-only baselines, unseen pathways and shifted contexts |
| C | NOT RUN | H1 development prerequisite failed |
| D | NOT RUN | H1/H2 development prerequisites unavailable; no adaptive selector installed |
| E | FAIL | Lihat metrik lengkap pada SCIENTIFIC_RESULTS.json |
| F | FAIL | Lihat metrik lengkap pada SCIENTIFIC_RESULTS.json |
| G | FAIL | Lihat metrik lengkap pada SCIENTIFIC_RESULTS.json |
| H | NOT PROVEN | four serial utility blocks executed; distinct-ability consolidation/correction protocol not reached |
| I | PASS | one N, 32 shared features, all learned coefficients canonical signed synapses; no utility controller |
| J | PASS | 48 final checkpoints: exact restart/full suffix replay/signed reconstruction/rollback, historical prefix intact |

Repository standing evaluation: tujuan utility diidentifikasi; implementasi native berjalan; hasil dibandingkan dengan ambang/kontrol yang telah dibekukan; keputusan perlu perbaikan. Keberhasilan build/recovery tidak diubah menjadi keberhasilan learned utility. Tidak diperlukan klaim AGI untuk menerima mekanisme yang benar-benar lulus, tetapi gate wajib yang gagal tetap menahan overall PASS.

## 22. Next research recommendation

Pertahankan 32 fitur dan sumber R2. Prioritas berikutnya adalah menguji sufficiency/access descriptor: pertahankan pasangan activity–weight–delta secara lokal, sertakan konteks/history yang benar-benar tersedia, lalu ukur alias raw/activation/metric secara terpisah pada development baru. Bedakan informasi hilang, akses hilang, basis yang tidak memadai, finite optimizer failure dan admission yang berlebihan; jangan langsung memperbesar kapasitas.

Setelah representasi memadai, uji generic bounded/regularized fitting dengan Fit/Check/dev baru dan pertahankan semua hasil negatif. Utility baru boleh memengaruhi plasticity sesudah H1 dan H2 lolos development independen. Jangan mengubah source/witness limits berdasarkan final R6, jangan memilih seed terbaik, dan jangan mempromosikan salah satu arm ini.

Laporan dan arsip ini mempertahankan kemajuan implementasi sekaligus batasnya: pengalaman causal numerical utility dapat disalurkan ke N dan direcover persis; kemampuan reliable learned retention/adaptive plasticity belum terbukti.
