# Kelanjutan Brain PR68 — protokol E1

## Tujuan

Lanjutkan Brain sequence10817, 1536 update native, 57 neuron/94 sinaps/24 pola.
Tidak ada inisialisasi ulang atau latihan A–D. Validasi archive SHA256
b9cda5961c5d59f5ec47fd436b39e9e4ed4cbd24f7ef93c3f0faa75a98c31dc0,
manifest, checkpoint checksum dan journal prefix sebelum pembelajaran.

## Implementasi dan prasyarat

Gunakan policy native v3 + encoder signed v2 yang sudah terlatih dan outcome
meter C yang sama: aktivitas aktual pada populasi outcome r/s, biaya propagate
.05. Operasi tetap propagate/stop; learning rate .5 dan semua konfigurasi tetap.
Representasi berasal dari Brain PR68, bukan hasil pemetaan neuron Brain lain.

E1 melatih pilihan operasi ketika sinyal dan aktivitas nuisance bersamaan.
Sinyal memakai cue yang sudah memiliki hubungan hasil dari B, atau aktivitas
yang sudah mencapai hasil. Nuisance memakai populasi lama 0 atau8; kedua
intervensi benar-benar dijalankan. Verifikasi perubahan return akibat nuisance
adalah nol pada continuation satu propagate lalu stop yang diuji.
"Tidak berguna" berlaku untuk konsekuensi dan continuation ini, bukan semua masa depan.

E2, pergantian tujuan menurut konteks, tidak dipaksakan: B membedakan p->r dan
q->s, tetapi basis policy current-state terbukti identik untuk dua input dengan
konteks berbeda yang membutuhkan pilihan operasi berlawanan. Ini prasyarat
representasi yang belum terpenuhi pada handoff langsung, bukan kegagalan belajar.
Tidak ditambahkan encoder/controller/migrasi agar skor E2 terlihat berhasil.

## Pengujian dan target lokal, dikunci sebelum latihan utama

Train: sinyal .3/.5/.7/.9, nuisance 0/.25/.6/1, dua populasi nuisance, dua kondisi.
Satu blok 48 ronde =3072 pasangan faktual. Urutan dua aksi dibalik tiap ronde.
Maksimum empat blok pada Brain yang SAMA. Setelah tiap blok hanya validation
yang mengendalikan berhenti; simpan checkpoint, journal dan semua hasil tiap blok.

Validation: sinyal .35/.55/.75, nuisance .2/.4/.8 (36 kasus).
Holdout: sinyal .4/.6/.8, nuisance .1/.5/.9 (36 kasus), hanya dinilai sebelum
dan sesudah seluruh latihan. Tidak dipakai memilih jumlah blok.
Probe dijalankan pada salinan evaluasi tanpa pembaruan policy.

Berhenti setelah blok jika accuracy validation >=.90, regret <=50% baseline
validation, dan sensitivitas nuisance <=70% baseline validation. Jika batas
empat blok tercapai, simpan pembelajaran dan laporkan kegagalan lokal; jangan
reset Brain atau mengganti ambang setelah melihat holdout.

Target holdout: accuracy >=.90 dan naik >=.10 absolut, regret turun >=50%,
sensitivitas nuisance turun >=30%. Sensitivitas = rerata |advantage noisy - clean|
dengan sinyal tetap; regret menggunakan dua return yang benar-benar dieksekusi.
Kontrol: ancestor tanpa paparan baru dan ablasi sinaps policy pada salinan.

Retensi: probe A/B/D existing mempunyai semua margin >=.1; C minimal5/6 sesuai
target lokal PR68, tetap laporkan setiap penurunan dari6/6. Periksa sesudah
latihan dan sesudah restart. Tidak ada refresh A–D kecuali diagnosis membuktikan
perlunya; tidak ada bobot yang dibekukan demi daftar jawaban.

Hipotesis turunan yang mempunyai dasar: B mempertahankan konteks melewati frame
gangguan melalui mekanisme recurrent-context/eligibility existing. Nilai sebelum
dan sesudah E1; keberhasilan yang sudah ada sebelum E1 bukan kemampuan baru E1.
Tidak menggunakan probe ThinkAt(1) lama sebagai vonis tentang komposisi mandiri.

Engineering: satu owner Brain, exact continuation+checksum pada genesis/suffix
replay, update native tepat1536+pasangan baru, ID/provenance existing, tidak ada
physical gateway dalam latihan. Checkpoint07 dan full journal menjadi parent
berikutnya. Parent lama/production/V6 tidak diubah. Tidak mengubah source di
tengah kurikulum; perubahan pengalaman antarblok mengikuti jadwal di atas.

## Keputusan

Pisahkan engineering, direct learning E1, E2 yang terblokir, retensi, dan hipotesis
turunan. G8 tetap BLOCKED sampai konfirmasi90 perbandingan asli lulus. Profil
diagnostik tidak menggantikan gate dan tidak mengubah threshold/durability.
