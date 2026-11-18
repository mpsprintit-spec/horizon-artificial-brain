# Native learning capability audit

| Jalur yang diaudit | Temuan kode aktual | Implikasi penelitian |
|---|---|---|
| Aktivitas rekuren | `activation` dan native runtime memakai Brain registry yang sama; command journal merekam begin/step/outcome dan poststate. | Residual S1/S2 berasal dari operasi yang benar-benar dieksekusi. |
| Aktivitas numerik | `nMetricEvaluate` menghitung projection/product, quantized code, tanh activation, metric reconstruction dan uncertainty dari32 circuit pada signed N. | Aktivasi numerik tidak sama dengan keseluruhan aktivitas rekuren. |
| Signed parameters | `loadNModel`, `compositionParameter`, `writeNParameter`; positive/inhibitory canonical synapses, population averaging, explicit signed bounds. | Workspace solver hanya sementara; inference membaca koefisien N. |
| Eligibility umum | Activity plasticity memiliki elapsed-time decay; native ownership menghindari legacy mutator mengubah owned parameters. | Aktivasi besar sendiri bukan causal credit. |
| Eligibility numerik | Writer mengubah eligibility berdasarkan delta parameter, frequency dan timestamp pada canonical synapses. | Eligibility ini bukan utility label dan belum membuktikan prediksi manfaat. |
| Credit operasi | `LearnNativeOutcome`: Sarsa(lambda) dengan feature/action yang benar-benar dieksekusi. | Credit temporal statistik, tidak otomatis merupakan identifikasi kausal seluruh neuron. |
| Credit berpasangan | `native_contrast.go` dan `native_action_contrast.go`: matched factual trajectories, action continuation contracts, signed updates di N. | Causal validity masih memerlukan matched exogenous conditions, bukan hanya pair ID. |
| General neural policy | `NativeControlState` menyimpan topology references; learned action values dibaca dari sinaps. Available operations adalah propagate/focus/relax/stop sesuai config. | Sumber R2 mempunyai10.752 updates. Tidak ada operasi candidate-selection/plasticity yang sudah terbukti. |
| Numeric learning | Fit independen; Check/admission; existing witnesses; finite readout/leaf candidate search; parameter installation melalui native transaction. | Belajar hubungan angka tersedia, tetapi belum ada pengalaman utility yang dikodekan kembali ke N. |
| Solver R5 | Incumbent-relative, box bounded, dyadic coefficient grid, explicit witnesses/anchors, original native gate setelah fitting. | Optimizer tidak boleh disebut learned retention. |
| Retention coverage baru | `RetentionBanks` maksimal4×64, original128 witnesses dan original64 anchors tetap terpisah; setiap tambahan mempunyai veto regresinya sendiri. | Safeguard eksternal. Perbaikan bank ini tidak dihitung sebagai H1–H4. Solver masih membentuk kandidat berdasarkan bank R5, bukan sertifikat seluruh feasible space. |
| Pengetahuan lama | Policy values dan numeric readouts/feature topology pada Brain; checkpoint/WAL merekonstruksi signed graph. | Tidak ada salinan model kedua yang menyimpan hasil utility di luar N. |

## Penghubung pengalaman minimal

Adapter riset memilih intervensi tanpa membaca target: single/pair signed readout-path perturbations dan pathway ablation. `S2R6Describe` memuat signed parameters dari N, mengukur preactivation/metric, eligibility, parent/fanout summaries dan konteks input terbatas. Descriptor tidak berisi neuron ID, actual error, hasil counterfactual atau keputusan gate. Query4 adalah pertanyaan yang diminta pada komposisi native, bukan ID tersembunyi yang memberikan jawaban.

Urutan tiap pengalaman: descriptor → native utility prediction → durable ledger → dua forward counterfactual pada state/input/parameter awal yang sama → actual squared-loss difference. Actual target S2 adalah residual dari operasi native yang telah dicatat. Intervensi tidak memasang parameter berisiko pada lineage; delta, source checkpoint dan outcome cukup untuk merekonstruksi workspace persis.

Kedua forward menjalankan evaluator numerik native pada salinan parameter N yang sama. Ini uji kontribusi **numerical readout pathways**. Ia tidak mengablasi seluruh neuron rekuren, tidak mengukur semua peran sebuah neuron, dan tidak memberi bukti interaksi representasional multi-jalur di seluruh Brain. Group intervention memakai pasangan pathway; interaksi loss/clipping dapat muncul, tetapi bukan bukti causal decomposition seluruh jaringan.

Pelajaran utility memakai generic shared-basis reuse. V1 memakai fitter lama; V2 menggunakan constrained solver R5 untuk menghindari penolakan menyeluruh akibat unconstrained coefficients. Keduanya ditandai protocol berbeda untuk replay. Hanya bobot/bias query baru dipelajari;32 fitur bersama tidak bertambah. S2 tetap head aslinya. State N menyimpan readout baru pada ordinary canonical synapses; tidak ada tabel pengalaman utility yang dipakai langsung saat inference.

## Keterbatasan yang harus dipisahkan

1. Descriptor memakai ringkasan mean pada intervensi pasangan. Dua pasangan berbeda dapat mempunyai ringkasan sama tetapi efek berbeda. Kehilangan informasi ini harus dibedakan dari learning-rule failure atau feature capacity.
2. Descriptor hanya membawa sembilan raw coordinates awal, bukan seluruh36-neuron state dan history. Prediksi S2 incumbent membawa sebagian konteks, tetapi sufficiency descriptor tidak dijamin.
3. Reuse fixed basis membuktikan atau menggagalkan metode pada representasi yang diuji; kegagalan bukan bukti matematis bahwa32 fitur tidak cukup.
4. Berbagi fitur dengan readout baru dapat memperketat shared-dependency guard untuk pelajaran berikutnya. Itu perlindungan eksternal, bukan consolidation yang sudah dipelajari.
5. Utility query tidak memengaruhi kandidat yang dipilih atau dipasang. Tidak ada argmax utility, threshold utility, heuristic ranking atau plasticity controller yang diam-diam dipasang.
6. Native utility exposure Check tidak masuk Fit; beberapa prototype awal memakai seluruh acquisition untuk offline baseline, diperbaiki dan disimpan sebelum confirmatory. Label-shuffle bisa menimbulkan conflict rejection; seluruh rejection tetap dilaporkan.

H1 harus lulus development sebelum H2. Kegagalan H1 menahan H2/H3/H4 dan arm selection G. C/F setelah empat blok masih menguji learning utility prediction, bukan native adaptive plasticity. Tidak ada alasan mengubah main, mengganti Brain produksi atau memperbesar kapasitas.
