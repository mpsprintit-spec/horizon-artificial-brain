# Canonical integration dan continual learning

## Tujuan

Menyiapkan unit yang layak untuk main dan melanjutkan Brain PR68, tanpa
mengulang A–D sebagai training utama atau membangun learning engine baru.

**Hasil:** koreksi startup63 READY secara independen; stack native coexistence
tetap BLOCKED oleh G8. Brain PR68 berhasil menerima E1 pada source yang sama,
dengan retensi A–D dan state akhir yang dapat dilanjutkan.

- [Main readiness matrix](MAIN_READINESS.md)
- [Proven learning inventory / compatibility & consolidation](PROVEN_LEARNING_INVENTORY.md)
- [Protocol E1](PROTOCOL_E1.md)
- [Performance dan CI](PERFORMANCE.md)
- [Continuation record](CONTINUATION.md)

## Implementasi / perubahan minimal

Main tetap `3a9e7ce3ffadff86c792050056ebea4eda48da0b`. PR63 yang sudah ada
diperbarui terhadap main, bukan dibuat ulang atau di-cherry-pick sebagian.
HEAD `cee28ff2431b2a5f5d2ea61c668c4de262c25426`, hanya tiga file berbeda dari
main. `finishProductionStartup` menghormati needsBootstrap: recovered owner
tidak menerima bootstrap kedua dan tidak menulis ulang memory/checkpoint.
Regresi baru membuktikan resume dapat diulang dengan jalur snapshot tidak
dapat ditulis tanpa mengubah byte Brain/journal atau sequence. CI sukses.
Tidak ada merge ke main; tidak ada default native controller baru.

Branch penelitian / PR69 bertumpu pada PR68. Perubahan hanya preparer verifikasi
lineage, harness kelanjutan, CI dan dokumen. **Tidak ada perubahan pada learning
law, encoder, operasi, parameter store, atau sumber hot path runtime** untuk
mengejar skor E. Freeze lokal0687321 / published
`a5d2842f8cc4fa835d32c87bb2008244b0c51768`, tree
`b5943abb9effa1d863b225d2cb33231e69e555fb`. Full tests/race/vet/build/Android
cross-build dan CI43597835457 berhasil.

## Pengujian — lanjut dari Brain yang sama

Parent archive PR68 diverifikasi SHA256 b9cda596..., seluruh selected payload
manifest, checkpoint checksum5b44d516..., canonical hash cfbe7b6f... dan complete
continuation hash d2d9a3a0.... Restore+prefix validation dilakukan sebelum train.
Tidak ada fallback ke Brain kosong, produksi pengguna, V6, atau checkpoint lain.

| State | Parent PR68 | Sesudah E1 | Sesudah restart/continuation |
| --- | ---: | ---: | ---: |
| Sequence | 10817 | 75329 | 75331 |
| Native updates | 1536 | 10752 | 10752 |
| Neuron / sinaps / PatternSynapse | 57 /94 /24 | 57 /94 /24 | 57 /94 /24 |
| Policy edges berubah vs parent | — | 72 | 72 |
| World weight edges berubah vs parent | — | 0 | 22 |

L1 perubahan bobot bertanda policy selama E1 =8.462188477183945. Seluruh bobot
tetap sinaps kanonik, tanpa model reset. World changes22 terjadi pada observasi
dan thought sesudah restart melalui hukum existing; bukan pembekuan semua
bobot untuk mempertahankan daftar jawaban. Model config dan role addresses sama.
Seluruh12,295 receipt lama dipertahankan, dan journal baru mempertahankan byte
prefix journal PR68 yang identik.

### Direct learning E1 — pemilihan operasi tahan gangguan

Sinyal dan nuisance co-active, memakai populasi yang sudah ada. Reward adalah
aktivitas outcome aktual setelah operasi; dua alternatif benar-benar dijalankan
dengan matched initial state/common stop continuation. Biaya propagate tetap.05.
Tidak ada teacher label relevansi di policy atau forecast yang menjadi reward.

Jadwal terkunci: maksimum empat blok pada Brain yang sama,48 ronde per blok;
validation36 kasus menentukan stop. Berhenti setelah blok ketiga: **9216
pasangan /18432 trajectory** baru. A–D tidak dilatih ulang. Block validation
accuracy29/36 ->32/36 ->35/36; seluruh checkpoint blok dan hasil tetap disimpan.

| Metrik holdout36 variasi baru | Sebelum | Setelah |
| --- | ---: | ---: |
| Pilihan greedy optimal | 18/36 (50%) | 35/36 (97.22%) |
| Rerata probabilitas operasi optimal, epsilon-policy | .50 | .925 |
| Regret terhadap dua return faktual | .428330336 | .018610612 |
| Sensitivitas advantage terhadap nuisance | 1.428432056 | .154167124 |
| Perubahan return faktual akibat nuisance pada continuation ini | 0 | 0 |
| Kontrol ablasi sinaps policy setelah E1 | — | 18/36 |

Regret turun95.65%; sensitivitas turun89.21%. Holdout bukan validation yang
dipakai memilih jumlah blok. Ini36 variasi numerik pada dua kondisi lokal,
bukan36 tugas independen atau bukti attention universal. E1 membuktikan
ketergantungan keputusan terhadap nuisance berkurang; tidak membuktikan
penekanan neuron individual atau causal-credit localization eksklusif.

### Retention report A–D

| Probe existing | Sebelum E1 | Setelah E1 | Setelah restart |
| --- | ---: | ---: | ---: |
| A, minimum margin8 probes | 1.377930240 | 1.377859750 | 1.377859747 |
| B, minimum margin8 probes | 1.355518845 | 1.355423740 | 1.355423737 |
| C, pilihan optimal6 probes | 6/6 | 6/6 | 6/6 |
| C, MAE factual advantage | .363485448 | .310903855 | .310850103 |
| D observed-intermediate, minimum8 readings | 1.305220445 | 1.305065894 | 1.305065889 |

Semua margin tetap jauh di atas ambang lokal.1. Perubahan kecil A/B/D dilaporkan,
bukan dibulatkan menjadi state beku. Retensi diukur pada salinan evaluasi; tidak
ada refresh A–D, jawaban cache, atau parameter freezing permanen.

### Derived ability report dan prasyarat

1. **Direct:** E1 di atas memang diajarkan lewat pengalaman konsekuensi.
2. **Kombinasi yang mempunyai dasar:** B menyimpan konteks dan recurrent eligibility
   mempertahankannya melewati frame nuisance. Dua margin sebelum1.142247463 /
   1.142247462 dan sesudah1.142056654 /1.142056653. Kemampuan sudah ada sebelum
   E1 dan tetap bertahan; tidak diklaim sebagai kemampuan baru yang muncul dari E1.
3. **E2 calon direct curriculum, belum dilatih:** konteks p/q memilih goal r/s,
   tetapi seluruh basis9 input policy pada dua state yang diuji identik:
   [6.85,4.5,0,0,.65,.38,0,0,1]. Return aktual membutuhkan stop untuk satu kasus
   (.8 vs .309991), propagate untuk lainnya (.309991 vs0). Menambah readout
   training pada input identik tidak cukup. Ini batas handoff current-state
   proposal ini, bukan bukti semua jalur recurrent tidak bisa memisahkan konteks.
4. Tidak ada vonis tentang kemampuan lain yang belum diajarkan. D hanya ditargetkan
   memakai intermediate yang diamati. Probe ThinkAt(1) lama bukan latihan ataupun
   pengujian lengkap komposisi mandiri; hasil tersebut tidak dipakai menggagalkan D.

Langkah E2 berikut harus memeriksa apakah pengalaman dan jalur recurrent existing
dapat membentuk state yang berbeda sebelum meminta encoder/migrasi baru.
Tidak dipasang algoritma khusus tugas untuk memaksa E2 berhasil.

### Compatibility & consolidation report

Kode ordered/context/plasticity/feedback/composition/spelling/V6 sudah tersedia
di main dan digunakan ulang sesuai kontraknya. Pengetahuan A–D benar-benar ada
dalam Brain tujuan dan dipertahankan; E1 ditambahkan di N yang sama.

V6 mempunyai Brain terlatih tersendiri, seq126/521 unit, dan bundle-nya diverifikasi
dengan fresh evaluation serta ablation. Itu **belum** menjadi pengetahuan PR68.
Arsip meaning/selector/relational/implant ditemukan dan diaudit; tidak ada mapping
representasi atau migrasi semantic-preserving yang membenarkan copy neuron by ID.
Selector joint lama bahkan mempunyai hasil negatif relevan. Sumber historis
dipertahankan sebagai evidence; metode variasi nuisance yang sesuai boleh dipakai
sebagai rancangan pengalaman, bukan sebagai klaim transfer parameter.

## Persistence evidence / keselamatan

- Complete canonical continuation cocok byte-exact untuk genesis, parent-checkpoint
  + suffix, final checkpoint restore, dan transisi observation/thought berikutnya.
- Full journal75332 record fisik /75331 committed sequence; intent tambahan berasal
  dari sensor PR68. Native pair receipts dan original byte prefix tetap ada.
- Exact retry outcome terakhir idempotent; reward berbeda dengan identitas yang
  sama ditolak tanpa perubahan state. Update count tepat1536+9216.
- Regresi Gate G, old-version compatibility, policy-neuron admission, mixed
  sensor/native/learning, legacy isolation,30 subprocess crash boundaries lulus.
- Counting gateway regression: satu dispatch live yang diotorisasi, nol dispatch
  baru saat replay. E1 tidak menggunakan physical gateway.
- Tidak ada format baru/migrasi diam-diam atau overwrite checkpoint pengguna.
- Android arm64 hanya cross-build; hardware Android dan power-loss durability
  belum diuji dan tidak diklaim lulus.

## Keputusan

**Engineering:** startup63 READY; source kelanjutan dan persistence E1 teruji,
native stack tetap menunggu G8. **Training:** E1 memenuhi target lokal dan A–D
bertahan. **Generalisasi:** hasil hanya pada variasi in-scope yang dicatat.
**Main integration:** tidak ada merge; hanya kandidat63 yang disiapkan terhadap
main. **Consolidation:** mekanisme reused, foreign learned checkpoints tidak digabung.

## Next canonical checkpoint

Paket: [Horizon_Canonical_Continuation_E1_20261114.tar.xz](evidence/Horizon_Canonical_Continuation_E1_20261114.tar.xz).

Di dalam paket:

- `e1-main-continuation/07-E1-restarted.json`
- `e1-main-continuation/events.jsonl`

Checkpoint checksum:
`68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82`.
Journal SHA256:
`ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede`.
Archive SHA256:
`7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3`.
203 payload diverifikasi terhadap manifest. Lanjutkan pasangan ini; jangan
memulai kembali A–D atau menggantinya dengan V6/Brain kosong.
