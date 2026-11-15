# Main readiness matrix — 14 November 2026

## Tujuan dan baseline autentik

Menilai unit engineering secara terpisah, menyiapkan integrasi yang aman tanpa
merge, dan mempertahankan seluruh dependensi/provenance penelitian.

Main terverifikasi: `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.

| PR | HEAD saat audit awal | Dependensi | Status |
| --- | --- | --- | --- |
| 61 | 0ef994a1d8842b36dbc6b994ecafcb0bf9951174 | main historis d7cb140 | Draft/open, belum merged |
| 62 | ea69b2101004dbf10c07068e42436170c2adcf8d | 61 | Draft/open, belum merged |
| 63 | 1837968ce0ba5707e3f15a36b14a7c87b1cc357f | main | Kandidat independen; diperbarui di bawah |
| 64 | 0df3f19a33bd7de69ed496293591d734331c5e51 | 62 | Draft/open, belum merged |
| 66 | 22f5de7017f538778cf5eb460a63d43351848c76 | 64 | Lab penelitian terpisah; bukan dependensi 67/68 |
| 67 | e9ae2331c9c020f3d905a79cc0a303599ac83889 | 64 + main + versi awal63 | G8 masih BLOCKED |
| 68 | 801ca9ae116ef5fb91a2a51cddfcafa594fc8a8d | 67 | CI43588974833 sukses; source freeze daf040a |
| 69 | a5d2842f8cc4fa835d32c87bb2008244b0c51768 | 68 | Kelanjutan E1 / audit ini, research non-default |

PR67 sudah membawa koreksi awal63 melalui merge8f3a817 dan main melalui03206ee.
Head63 baru di bawah menambah sinkronisasi main, format dan regresi; bukan
dependensi kognitif baru. Tidak ada cherry-pick parsial atau force push.

## Implementasi dan keputusan per unit

READY berarti kandidat engineering yang siap ditinjau untuk merge oleh pengguna,
bukan klaim bahwa sudah masuk main. ALREADY IN MAIN juga bukan otomatis seluruh
perilakunya production-ready. Kolom promosi menjelaskan dependency closure.

| Unit | Canonical N / bukti | Kompatibilitas dan risiko | Status unit | Keputusan promosi |
| --- | --- | --- | --- | --- |
| Ordered observations, recurrent context, observed feedback | Runtime production + fixture recovery/checksum historis | Versi dinamika/checkpoint existing; jangan menganggap prediksi sebagai observasi | ALREADY IN MAIN | Reuse, jangan implementasi ulang |
| Plasticity, retensi, reconsolidation | Sinaps dinamis/pola kanonik; tests activity_retention, observed_feedback, reconsolidation | Adaptasi historis harus tetap versioned; decay bukan penghapusan | ALREADY IN MAIN | Pertahankan |
| Optimasi PR60/65 | Main sudah memiliki stable registry traversal dan curiosity winner selection | Baseline performa autentik; tidak dipasang ulang | ALREADY IN MAIN | Tidak ada duplikasi |
| Startup recovery63 | needsBootstrap mengendalikan bootstrap, rujuk owner supervisor yang sama | Tidak mengubah format/model; hanya format yang didukung main dapat dipulihkan | READY | PR63 terhadap main, CI sukses, belum merge |
| Native compute primitives / policy parameter law61–64 | Bounded operations dan readout sinaps N, observed action contrasts | Belum default; parameter protection, schema dan admission harus ikut sebagai closure, tidak boleh diekstrak sembarang | READY WITH OPT-IN untuk fungsi yang diuji | Paket menuju main tetap BLOCKED oleh closure67/G8; bukan penolakan fungsi lokal |
| Default behavior/controller promotion | Gate C/D dan keterbatasan credit/transfer penelitian masih relevan | Keberhasilan API tidak mengesahkan controller default | RESEARCH ONLY | Jangan aktifkan default |
| Coexistence/admission/unified journal67 | G1–G7/G9/G10 regressions lulus | Default/uninitialized thought juga terkena hot-path changes; opt-in saja tidak menghapus biaya itu | BLOCKED | Sembilan pelanggaran G8 harus diselesaikan sebelum promosi paket |
| Signed encoder V2 + validator68 | Sign collision terpisah, dimensi/version fail-closed, replay/ownership/crash lulus | Hanya init model eksplisit; migrasi learned policy encoder lama belum ada | READY WITH OPT-IN untuk model baru pada runtime kompatibel | Tidak di-port tanpa native persistence/admission closure; bukan upgrade otomatis |
| Spelling I/O, learned continuation | Signed synapses + Unicode motor coordinates; ablation dan replay existing | Vocabulary/format terikat kontrak; tidak sama dengan pemahaman bahasa umum | ALREADY IN MAIN | Reuse sesuai scope, codec tetap eksperimental |
| Composition/N-resolution | Parameter directions/centers/resolution/readouts dalam N | Opt-in version, witness/resource bounds; migrasi trained V4/V5 bukan otomatis | ALREADY IN MAIN | Reuse API; jangan menyalin bobot antarbank |
| V6 dialogue code dan model bundle | Bundle manifest + fresh evaluation/persistence/synaptic ablation | Learned Brain tersendiri, bukan pengetahuan PR68 | ALREADY IN MAIN | Pertahankan bundle; tidak digabung ke PR68 |
| Brain PR68 A–D / E1 | Lineage dan journal lengkap, direct learning dan retensi tercatat | Research training state, bukan pengganti memory pengguna | RESEARCH ONLY sebagai distribusi state | Lanjutkan lineage, jangan auto-install ke main |
| Context-goal E2 langsung ke policy | Exact collision seluruh9 feature termasuk bias, factual optimum berlawanan | B sudah menyimpan konteks, tetapi projection current-state ini menghapus perbedaannya | BLOCKED | Diagnosis representasi, bukan kegagalan latihan; tidak diberi patch khusus tugas |
| Selector/joint prototype12 November | Source bundle/manifest ditemukan; joint91.67% vs baseline98.33% | Code overlay lama, 16 fitur/metadata berbeda, bukan native journal14 | RESEARCH ONLY | Jangan port heuristic credit gabungan yang ditolak hasilnya |
| Implant46 / lab66 / transfer historis | Ada struktur/proxy/artefak atau source terpisah, scope bukti terbatas | Tidak ada identity/representation migration terverifikasi | RESEARCH ONLY | Simpan evidence, jangan merge neuron berdasarkan ID |

## Kandidat integrasi bersih yang disiapkan

PR63 dipakai kembali, tanpa membuat PR duplikat. HEAD baru:
`cee28ff2431b2a5f5d2ea61c668c4de262c25426`.
Parent adalah head63 asli dan main3a9e7ce. Diff terhadap main hanya:

- services/ai/main.go;
- services/ai/production_bootstrap.go;
- services/ai/production_bootstrap_test.go.

Regresi baru memanggil resume tiga kali dengan jalur snapshot yang sengaja
tidak dapat ditulis, lalu membandingkan byte Brain/journal dan sequence.
Full tests, race, vet, build, Android arm64 cross-build serta CI43597512222
berhasil pada tree ini. Clone biasa mempunyai vcs.revision=cee28ff dan
vcs.modified=false. Error VCS stamping pada worktree lokal dipertahankan sebagai
diagnostik lingkungan; tidak disamarkan sebagai keberhasilan build.

PR masih draft/open untuk ditinjau; **tidak ada merge ke main**. Pengguna tetap
memegang keputusan merge. Tidak ada controller eksperimental diaktifkan.
Perbaikan ini tidak menambah dukungan checkpoint native20 ke binary main yang
belum memiliki stack native, dan tidak mengizinkan downgrade Brain penelitian.

## Pengujian performa / G8

Lihat [PERFORMANCE.md](PERFORMANCE.md). Artefak81/90 lama dipertahankan. Profil
diagnostik hanya menyentuh kasus thought, learning4k, native-idle dan checkpoint4k
yang gagal. Tidak ada patch runtime/perubahan threshold yang dipromosikan dari
profil sesaat. G8 tetap BLOCKED sampai konfirmasi penuh memakai anggaran asli.

## Langkah berikutnya

Tinjau kandidat63 secara independen. Untuk stack native, selesaikan bottleneck
yang dapat direproduksi serta compatibility closure terlebih dulu. Jangan
menyetujui seluruh PR stack sekaligus hanya karena unit startup siap.
