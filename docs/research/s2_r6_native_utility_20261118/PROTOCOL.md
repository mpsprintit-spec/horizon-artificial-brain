# S2-R6 — preregistered staged research

Tanggal: 18 November 2026 (Asia/Jakarta). Main `11c6e0a78547d68a0be8f78b3c801e912722608d`.
Basis kode: R5 `099ae285737cdc62fc121791ba4e6f4e463a1cb3`; scientific R5 `2fac4b8e21cf89ae211aac3f8ee7f52473fda4e6`.
Sumber seluruh salinan: R2 sequence 80735, native checksum `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.

## Scope dan batas

H1 diuji dahulu melalui konsekuensi intervensi pada kontribusi fitur ke readout S2. Ini adalah intervensi komputasional pada signed numerical pathways, bukan klaim causal credit seluruh aktivitas rekuren. Semua readout yang dipelajari memakai 32 fitur N yang sama. Query baru adalah readout native, bukan jaringan baru. Utility tidak dihitung oleh codec. Codec hanya menyediakan pengukuran praintervensi; loss aktual hanya menjadi label setelah forecast dicatat.

H2 dimulai hanya jika H1 lulus development independen. H3 dimulai hanya jika H1 dan H2 lulus development. H4 memerlukan H3. Kegagalan prasyarat menghasilkan NOT RUN / BLOCKED untuk fase bergantung, bukan hasil nol atau keberhasilan tersirat. Tidak boleh memberi PASS keseluruhan bila gate wajib gagal atau tidak diuji.

## Audit dan primitive

Existing numeric learner memakai least-squares / constrained proposal workspaces dan signed canonical writer. Existing general neural policy mengendalikan propagate/focus/relax/stop, bukan candidate selection. Eligibility numerik ditulis pada transaksi parameter; general policy memberi credit temporal, dengan paired contrast pada versi kausal. Tidak satu pun secara otomatis menyediakan pengalaman kontribusi numerik ke readout utility.

Primitive minimal `native-shared-basis-reuse-v1` membatasi pelajaran pada readout fixed basis. Tidak ada freezing permanen: versi ini hanya ruang pencarian untuk satu lesson. Pelajaran S2 tetap memakai solver R5. Reorganisasi 266 dilakukan sebelum utility bila diperlukan. Descriptor tidak memuat ID neuron/kandidat, hasil aktual, label, atau keputusan gate. Audit-only IDs disimpan terpisah.

## Data dan stopping

Enam seed training dibekukan: 37, 83, 151, 211, 307, 419. Development memakai 37 dan 83 dengan acquisition seeds 17182xxx–17186xxx; seluruh ID berbeda dari final. Pilot tidak memilih seed terbaik. Development dapat menemukan bug; setiap percobaan gagal dipertahankan. Semua perbaikan harus sebelum freeze final.

H1 development: 24 stream ×16 operasi aktual per seed; 16 stream untuk bootstrap D32 empat lesson R5 (Fit32/Check32), delapan stream untuk causal training. Dua intervensi per operasi, 256 contoh utility: Fit128 dan Check128, dipakai bergantian dalam empat transaksi. Evaluasi development: delapan stream baru ×16 operasi ×dua intervensi, masing-masing seen dan unseen ordinals. Semua intervensi dari satu operasi tetap pada satu split. Ordinal mod3=0 tidak pernah diintervensi dalam Fit/Check. ID ordinal tidak masuk codec.

Intervensi: single atau pair pathways, signed delta dari {-0.08,-0.02,0,0.02,0.08}, dan readout-path ablation setiap pola kelima. Ordinal dan delta dipilih RNG sebelum target dibaca. Pada setiap contoh: deskripsi → forecast native dan ledger sync → detached counterfactual forward → loss difference. Kedua forward memakai input dan parameter awal identik. Label adalah `(loss_before-loss_after)/4` dengan loss kuadrat; pembagian empat hanya normalisasi rentang [-1,1]. Tidak ada positive-only sampling.

H1 final: enam seed baru dalam acquisition namespace 181xxxxxx; masing-masing sumber R2, bootstrap empat lesson D32 pada 16×16 operasi baru. Causal Fit/Check adalah 16 stream lain ×16 operasi ×dua intervensi (512), delapan subset disjoint 64 untuk empat blok. Final acquisition baru dilakukan setelah checkpoint, code dan analysis disegel. Holdout utility: 32 stream ×16 operasi ×dua intervensi per seen/unseen/shifted condition. Holdout S2: 32×24 per persistent/IID/shifted condition. Final tidak boleh dipakai untuk melanjutkan training atau memilih arm/checkpoint.

Arm A: D32 R5 reproduction. B: coverage control. C: utility experience, tanpa penggunaan utility untuk selection. F: native reuse yang sama, causal labels diacak deterministik dalam Fit dan Check, mapping dibekukan per subset. G: nonlearned control bila tahap candidate-selection terjangkau. D/E dibuka hanya lewat prerequisite development. Semua arm dimulai R2; penyalinan bootstrap identik sebelum perbedaan arm diperbolehkan sebagai common prefix, bukan pemilihan checkpoint R5 final.

## Metrik dan gates

H1 primer: RMSE utility pada semua contoh, termasuk neutral. Pembanding constant Fit mean, untrained zero, OLS activation-magnitude-only, OLS weight-magnitude-only. Tidak ada fitted baseline yang dipasang ke N. Native harus memperbaiki RMSE ≥10% terhadap pembanding terbaik, CI95 cluster-bootstrap lower benefit >0 pada seen dan unseen. Macro-F1 tiga kelas ≥0.40 dan setiap kelas sekurang-kurangnya 10 contoh; kelas positive>1e-5, negative<-1e-5, neutral sisanya. Random-label control tidak boleh menghasilkan ≥50% gain C. Bootstrap 10.000 replika, RNG 181899991, unit primer seed lalu stream; development dua seed hanya uji prasyarat, final enam seed untuk kesimpulan.

H2: prediksi perubahan loss historis sebelum candidate diterapkan, ≥10% RMSE improvement terhadap constant/no-experience, CI95 positive, AUROC harmful vs beneficial ≥0.65, transfer unseen candidate combinations/magnitudes. Definisi harm dan sample size harus dibekukan sebelum akuisisi H2 setelah H1 development. Jika H1 gagal, H2 belum memiliki prereg confirmatory lengkap dan tidak dijalankan sebagai hasil terselubung.

H3: kandidat native yang sama, solver, gate, witnesses dan budget sama; learned selection vs ablated/shuffled/information-masked utility harus memperbaiki measured net benefit, CI95 positive, sekurang-kurangnya satu beneficial installed update setiap seed, bukan rejection-only. H4: empat blok serial A→B→C→D, matrix backward/forward transfer dan correction changed facts; tidak dilakukan bila prasyarat belum ada.

Retensi: source witnesses tidak dihapus. Historical S2 ≤0.01766457 semua seed; strict S1 shift0; A–E1 seluruh oracles. Old-distribution baru memakai threshold source RMSE+1e-7, dihitung sebelum training dari source pada development, bukan final candidate fitting. Bank berbeda dievaluasi terpisah. Kegagalan berarti Gate E FAIL tanpa post-hoc pelonggaran.

History: gain≥10%, CI95 lower>0; magnitude ablation dan conditional permutation harus mendukung. IID control equivalence CI90 di dalam ±3%, bukan hanya p>0.05. Calibration mean predicted/actual di [0.8,1.2] dan native mengalahkan constant Fit mean; tidak boleh global scaling dipasang sebagai kemampuan kontekstual.

Persistence: exact restart, canonical signed reconstruction, full suffix replay dari80735, original rollback, source byte hashes dan historical WAL prefix. Tidak ada WAL truncation. Android arm64 cross-build hanya compatibility, bukan perangkat fisik atau bukti bit-exact lintas arsitektur.

## Negative results dan artefak

Semua source/intermediate/final checkpoint, full WAL, causal measurements, pre-intervention forecast ledger, Fit/Check mapping, receipts/trials/rejections, costs dan failed development disimpan. Tidak ada arm yang dipromosikan. Arsip ber-manifest SHA256, source kompatibel, reproducibility scripts dan owner recovery diverifikasi setelah upload/download. Laporan memisahkan H1/H2/H3/H4 dan gate A–J, serta sebab tidak dijalankannya tahap bersyarat.
