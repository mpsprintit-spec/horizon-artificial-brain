# Proven learning inventory dan konsolidasi

## Tujuan

Pisahkan A: mekanisme kode yang dapat dipakai lagi; B: pengetahuan yang memang
ada pada lineage PR68; C: pengetahuan/struktur dari Brain terpisah. Identitas
global `horizon-primary-brain` tidak membuktikan bahwa dua checkpoint mempunyai
alamat, representasi atau provenance yang dapat digabung.

## Inventaris terverifikasi

Path di bawah relatif terhadap services/ai kecuali disebut lain.

| Hasil | Kemampuan dan bukti aktual | Kontrak neural / persistence | Reuse dan keputusan |
| --- | --- | --- | --- |
| Ordered observations | Urutan/pengulangan dan conditional prediction berubah karena paparan; kontrol shuffled dan pattern ablation existing | learning/ordered_experience.go; knowledge/ordered_pattern.go; runtime/ordered_*_test.go; canonical PatternSynapse, tidak ada vocabulary relasi semantik baru | A: sudah main. B: PR68 A terbukti dan dipertahankan sesudah E1 |
| Recurrent context | Prefix/context yang diamati membedakan outcome; histori dapat melewati gangguan dalam batas eligibility | activation/ordered.go, knowledge/recurrent_context.go; runtime/recurrent_context_test.go, ordered_recovery_test.go; context/activation ikut checkpoint | A: sudah main. B: PR68 B dan probe context-through-noise tetap tersedia; bukan hasil transplantasi |
| Local plasticity / retention | Aktivitas teramati memperbarui dynamic synapses; no empirical learning from thought-only; retensi parameter khusus dilindungi | activation/activity_plasticity.go; runtime/activity_retention_test.go; knowledge/memory_time.go; native confidence protection ada pada stack67 | A: reuse. Jangan membekukan semua bobot; perlindungan native harus ikut closure saat promosi |
| Observed feedback / prediction error | Feedback aktual memperbarui asosiasi/pola yang relevan; provenance/dedup existing | activation/observed_feedback.go dan runtime/outcome_provenance_test.go; evidence ledger/journal versioned | A: sudah main. E1 menggunakan outcome aktual dan law native yang sama, bukan reward forecast |
| Contradiction/reconsolidation | Bukti berlawanan mengubah support sambil menjaga history; bukan penghapusan paksa pengetahuan lama | knowledge/reconsolidation.go, reconsolidation*_test.go; signed channels/evidence canonical | A: sudah main. Tidak diklaim seluruh variasi contradiction sudah diajarkan pada PR68 |
| Neural pattern / prediction | Population traces, conductance/confidence/frequency/evidence dipelajari; ablasi menghilangkan conditional advantage | knowledge/ordered_pattern.go; runtime/ordered_prediction_test.go | A+B: 24 pola PR68 tetap24 selama E1; tidak ada tabel jawaban tambahan |
| Spelling / continuation | Bentuk Unicode diajarkan lewat signed synapses ke bit/end cells; tanpa edge spelling decoding gagal | knowledge/spelling_readout.go; runtime/spelling_readout_test.go; testdata/learned_spelling_v1; learned sequence continuation existing | A: sudah main, I/O bounded. B: PR68 belum dilatih spelling/V6; jangan menyatakan kode tersedia berarti Brain PR68 sudah tahu kata |
| Numerical composition / N-resolution | Learned directions, products, centers, resolution, query heads; numerical checks dan ablation existing | knowledge/neural_resolution.go; runtime/N_RESOLUTION.md; knowledge/neural_composition*.go; model/witness/resource contracts explicit | A: reuse existing opt-in API. B: belum diaktifkan/dilatih pada PR68; tidak auto-enable bank saat native sudah aktif |
| Native operation selection | Phase I–III menyediakan ordinary synaptic readout; PR68 C3/6->6/6 pada scope lokal | knowledge/native_control.go; activation/native_operation.go; runtime/native_control.go; native schema7/checkpoint20/journal14 | A: research source. B: C tetap6/6 setelah E1/restart |
| Causal action contrasts | Dua trajectory dieksekusi pada initial state/block/suffix yang cocok; common offset dan biaya diuji; duplicate outcomes ditolak | knowledge/native_action_contrast.go; runtime/native_action_contrast_test.go; pending pair dan receipt persisted | A+B: dipakai E1 tanpa law baru. Tidak sama dengan credit per-neuron eksklusif |
| Signed relational encoder | Kasus tandingan sign Phase III terpisah setelah penambahan signed moments; old basis tetap kompatibel | knowledge/native_relational.go; native_relational_v2_test.go; explicit encoder string | A+B: encoderV2 sudah ada pada parent; tidak diubah untuk E1. Temporal/role collisions masih batas representasi |
| PR68 A->B->C->D | Pembelajaran urutan, konteks, operasi berdasarkan konsekuensi, dan reuse dengan intermediate yang diamati | Archive SHA b9cda596..., checksum parent5b44d516..., full journal; seven genesis/suffix matches pada evidence lama | B: benar-benar di dalam Brain tujuan, dipertahankan, tidak diulang sebagai training |
| E1 baru | Pilihan operasi tahan nuisance: 18/36->35/36, sensitivitas1.428432->.154167; ablation kembali18/36 | 9216 pasangan faktual baru, 72 edge policy berubah dalam N, world weights tidak berubah selama E1; sequence75331 setelah restart | B: benar-benar dipelajari pada lineage yang sama. Bukan bukti selector neuron individual atau attention universal |
| V6 dialogue bundle | Manifest27 N circuits/29 heads/521 units/seq126; fresh evaluation, persistence/reference/spelling synaptic ablation dijalankan ulang tanpa training | models/grounded-dialogue-v1; archive SHA diverifikasi preparer; script menolak mencampur journal deployment; 480 substantive+16 clarification pada scope room yang dicatat | A: kode/bundle sudah main. C: learned state terpisah; tidak dipindahkan ke PR68 |
| Meaning12–13 November | Mixed nuisance-dropout report24/24 lokal, baseline/ablation/reversed feedback; hipotesis transfer sinonim dari ko-okurensi tetap2/8 | Horizon_Meaning_Laporan_20261113.md dibaca; checkpoint source244 units/507 synapses/60 patterns/seq221, schema13; basis c7bffe + overlay, bukan PR64/68 | C: reuse prinsip pemberian variasi nuisance, bukan copy bobot atau klaim seluruh pengetahuan dipindah. Validasi penuh/race dalam laporan lama belum lengkap |
| Causal selector / joint prototype12 November | Selector F1 95.31% pada holdout120; joint91.67% vs98.33% baseline, shuffled salience menyamai baseline | Archive9.25MB ditemukan, 24 payload manifest verified; 36 ordinary canonical Q edges,16 engineered features, source c7bffe + overlay; bukan native journal14 replay | C: pertahankan evidence dan source; tolak promosi heuristic joint yang merusak keputusan. Tidak menggunakan F1 selector sebagai presisi causal credit |
| Relational/philosophical continuation historis | Arsip memuat thinking-metrics: 72 lessons baru, 16->40 patterns; local6/6 dan transfer0/6 sesuai scope kasus yang diuji | Parent action runs43029895990/43032495188; source checkpoint dalam archive implant; checksum/provenance historis terpisah | C: hasil lokal dipertahankan. Tidak mengganti Brain PR68 dengan parent ini atau menganggap transfer luas sudah diajarkan |
| Structural implant46 | 184 unit/488 connections baru, zero learning exposures; proksi aktivasi46/46 dengan engineered links | Horizon_Internal_Cognitive_Implant_Experiment.zip ditemukan; sumber mengakui belum menjalankan CognitiveProcess pada checkpoint hasil saat itu | C: kandidat struktur, **bukan46 kemampuan dipelajari**; tidak dikonsolidasikan sebagai pengetahuan terbukti |
| PR66 numerical reasoning lab | Source/lab smoke tersedia, acquisition memakai pengalaman neural dan policy existing; promoted research gate belum dinyatakan lulus | HEAD22f5de7, base64; output metrics dibuat saat run, tidak dijalankan ulang dari Brain kosong dalam tugas ini | C/research only. Jangan menyamakan dengan selector overlay lama atau menyalin lab sebagai engine baru |
| MUSE/BMPS/V5 | Mekanisme/hasil historis mempunyai batas dan frozen-controller status sendiri | Architecture Lock, core/gate_g_legacy_test.go dan dependency guard; compatibility/migration/diagnostic use dibedakan | Isolasi dari recurring native path; tidak dipromosikan sebagai dependency baru |

## Yang benar-benar terkonsolidasi

Dalam Brain tujuan: A/B/C/D dipulihkan dari parent yang autentik, lalu E1
memperbarui72 sinaps policy existing. Konfigurasi model, role addresses, 57 neuron,
94 sinaps dan24 pola dipertahankan. Seluruh12,295 receipt dedup lama tetap ada;
journal baru mempunyai byte-prefix journal PR68 yang identik.

Dalam kode: reuse activation, ordered learning, context, native policy, observed
action contrasts, journal/checkpoint/recovery yang sudah ada. Tidak ada engine
belajar generik kedua atau algorithma produksi khusus nuisance.

Belum terkonsolidasi sebagai pengetahuan PR68: V6, meaning archive, selector
archive, implant dan hasil lab/transfer lain. Alamat neuron saling bertumpang
tindih, basis/encoding/config berbeda, dan kesetaraan semantik belum dibuktikan.
Tidak ada migrasi tervalidasi yang dipaksakan. Menyimpan sumber terpisah sebagai
evidence bukan menjalankan dua intelligence dalam satu runtime.

## Keputusan dan langkah berikutnya

Pertahankan yang sudah ada, lanjutkan data pengalaman pada lineage E1, dan
pakai sumber historis sebagai evidence/kurikulum bila sesuai kontrak. Migrasi
learned state memerlukan pembuktian mapping representasi dan makna serta
transaksi/versioning/rollback; nama yang sama atau identity string yang sama
tidak cukup. Ketiadaan pengujian suatu kemampuan tidak disebut kegagalan belajar.
