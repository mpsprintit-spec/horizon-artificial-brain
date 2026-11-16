# Integrasi produksi Brain kanonik A–E1 — 16 November 2026

## Tujuan

Baseline: `57fa1a6b8abf8d100ee58ecbd62da095d14a95aa`, setelah merge PR #71.
Instalasi layanan produksi baru harus memakai hasil pembelajaran A–E1 yang
telah tersimpan, mempertahankan satu Brain aktif, dan melanjutkan pengalaman
melalui runtime, supervisor, checkpoint dan WAL yang sudah ada.

## Implementasi

`services/ai/production_state.go` memasang checkpoint dan jurnal asli dari
arsip E1 yang sudah diverifikasi oleh `scripts/prepare_native_continuation.py`.
Paket `canonical-ae1.tar.gz` hanya mengompresi ulang byte anggota asli untuk
`go:embed`; tidak mengubah representasi, menjalankan akuisisi ulang, atau
membuat model pembelajaran pengganti.

Pemasangan memakai direktori staging, fsync berkas/direktori, validasi melalui
`initializeProductionHorizon` → `RuntimeSupervisor.RecoverFromCheckpoint` →
`RebuildRuntimeFromCheckpointAndJournal`, lalu atomic rename. Supervisor
validasi ditutup sebelum runtime yang melayani input dibuka. Bridge, adapter
sensor dan continuous cognition memakai supervisor dari runtime aktif itu.

State awal yang wajib cocok:

| Item | Nilai |
| --- | --- |
| Brain identity | `horizon-primary-brain` |
| Sequence | 75.331 |
| Native learning updates | 10.752 |
| Neuron / sinaps / PatternSynapses | 57 / 94 / 24 |
| Checksum runtime | `68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82` |
| SHA-256 arsip E1 | `7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3` |
| SHA-256 checkpoint asli | `e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537` |
| SHA-256 jurnal asli | `ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede` |

Sequence, processed receipts, state neural, dinamika recurrent, native policy,
causal credit, eligibility dan plastisitas dipulihkan oleh jalur autentikasi
yang sudah ada. Restart tidak memasang ulang snapshot awal di atas pengalaman
baru. Prefix jurnal A–E1 diverifikasi terhadap hash aslinya; pengalaman baru
menjadi suffix jurnal yang sama.

## Direktori produksi dan Brain lama

Tanpa konfigurasi atau state lama di working directory, layanan memakai
`./horizon-state`. `HORIZON_STATE_DIR=/absolute/persistent/path` memilih lokasi
persisten secara eksplisit. Di dalamnya ada checkpoint autentik,
`horizon_events.jsonl`, serving memory, dan metadata asal `ae1-origin.json`.

State lama di working directory dan pengaturan individual yang sudah ada tetap
diutamakan. Aktivasi A–E1 pada instalasi lama memakai direktori baru melalui
`horizon-ai -prepare-ae1 /new/path`, setelah penghentian writer dan backup utuh
yang diperiksa byte/hash-nya. Rollback mengembalikan konfigurasi ke direktori
lama; tidak menggabungkan checkpoint, jurnal atau neuron dua lineage.
Petunjuk operasional lengkap ada di
[CANONICAL_AE1_ACTIVATION.md](../../CANONICAL_AE1_ACTIVATION.md).

Peninjauan lanjutan menemukan penolakan memori lama yang rusak masih sempat
membuat jurnal kosong. `openProductionState` sekarang memakai loader standalone
yang sudah ada untuk memvalidasi memori sebelum membuat jurnal yang hilang;
serving-memory berbentuk checkpoint juga ditolak bila jurnal pasangannya hilang.
Uji negatif mencakup JSON rusak, checkpoint tanpa jurnal, dan state native yang
tidak dapat dimuat sebagai standalone. Penolakan menjaga byte memori dan tidak
menambah berkas di direktori state tersebut.

## Pengujian dan bukti

Hasil akhir lokal dan perintah yang benar-benar dijalankan dicatat dalam
`evidence/verification.json`; log mentah ada di direktori yang sama. Manifest
source mengikat bukti ke byte implementasi yang diuji. Tiga berkas bukti dari
pekerjaan awal tetap dipertahankan; bukti penutupan memakai nama `final-*`.

Penutupan lokal **PASS: 27/27 pemeriksaan**, termasuk race aktivasi, seluruh
15 nested modules, Bridge, startup/input/restart executable, oracle retensi,
vet/build, dan dua cross-build Android arm64. Manifest source masih cocok
setelah seluruh pemeriksaan selesai; **115/115** hash yang dilindungi cocok.

Pengujian aktivasi memeriksa:

1. Startup produksi memuat sequence 75.331 / updates 10.752 dengan nol bootstrap.
2. Satu pengalaman native baru pada direktori uji sementara melanjutkan
   sequence ke 75.335 / updates 10.753 pada substrate yang sama.
3. Restart idle, suffix WAL tanpa checkpoint baru, dan restart checkpoint
   mempertahankan seluruh state hasil pengalaman tersebut secara identik.
4. Pemilik jurnal kedua ditolak; Brain lama, konfigurasi aktivasi eksplisit,
   backup dan rollback diperiksa tanpa mengganti state asli.
5. Pasangan tidak cocok, paket rusak, konfigurasi ambigu, direktori tidak
   terkait, symlink direktori, dan memori tidak valid ditolak.
6. Binary produksi dijalankan dari working directory kosong, menerima input
   HTTP melalui Bridge yang sebenarnya, lalu dihentikan dan dijalankan ulang.
   Sequence input dipulihkan dan recurrent cycle berikutnya berlanjut.

Oracle A–E1 lama dipakai tanpa diubah. Verifikasi saved continuation membandingkan
genesis replay, checkpoint restore, suffix replay, checksum, readout A/B/D,
probe C **6/6**, E1 **35/36**, serta next-thought recurrent terhadap oracle PR #69.
Akuisisi A–D/E1 tidak dijalankan kembali. Satu pengalaman untuk uji kelanjutan
hanya memutasi direktori pengujian sementara; bukan penggantian checkpoint asal.

Regresi Go penuh, race aktivasi, vet/build, 15 nested adapter modules, Bridge
dan build Android arm64 dilaksanakan untuk kandidat penutupan. CI mempertahankan
seluruh pemeriksaan lama dan menambahkan validasi paket, race aktivasi,
startup/input/restart executable, serta cross-build layanan produksi Android.
Status CI dan keputusan merge harus dibaca dari checks pada PR untuk commit
final; laporan source ini tidak menggantikan keberhasilan workflow tersebut.

## Arsitektur, keputusan dan batas cakupan

Seluruh **115/115** sumber/artifact yang dilindungi pada retirement tetap cocok
dengan hash yang dipertahankan di baseline. Termasuk Brain produksi lama,
seluruh state/paket penelitian asli dan implementasi neural yang dilindungi.
SHA-256 `services/ai/brain_memory.json` tetap
`c7749827a5ce1a8799e677f1cff2ad05d2d532ac3715164a35eece5499453cc4`.
Tidak ada perubahan hukum pembelajaran, otorisasi tindakan fisik, atau dispatch
saat replay. Tahap latihan berikutnya/S1 tidak dimulai.

Keputusan fungsional aktivasi ditentukan oleh pengujian di atas. Integrasi ke
main mensyaratkan seluruh pemeriksaan CI commit final berhasil; hasil dan
commit merge dilaporkan pada PR. Hasil performa G8 penuh terakhir pada baseline
tetap **89/90** dengan pengecualian pemilik yang sudah tercatat pada PR #71.
Pekerjaan startup ini tidak mengubah hasil tersebut atau mengklaim G8 90/90.

`cmd/horizon-chat` dan CLI historis tetap memiliki loader/profil lama.
Profil bahasa `grounded-dialogue-v1` bukan checkpoint A–E1 dan tidak digabung
dengan lineage A–E1. Aktivasi ini berlaku pada layanan produksi, Bridge, sensor
dan continuous cognition; tidak mengklaim migrasi chat Termux. Menyalin A–E1
ke folder profil chat yang formatnya berbeda bukan prosedur aktivasi.
Tidak ada frontend/aplikasi Android baru. Build Android berhasil tidak berarti
uji eksekusi pada perangkat Android/Termux fisik atau pemadaman daya perangkat.

## Berkas yang diubah

| Berkas | Perubahan |
| --- | --- |
| `services/ai/production_state.go` | Pemilihan state, pemasangan terverifikasi, lock, provenance, aktivasi eksplisit dan penolakan aman |
| `services/ai/canonical-ae1.tar.gz` | Paket byte checkpoint/jurnal asli untuk executable |
| `services/ai/main.go` | Menggunakan state produksi sebelum input/cognition; path state pada readiness |
| `services/ai/startup.go` | Penjelasan jalur startup A–E1 dan formatting Go; recovery tetap mekanisme yang ada |
| `services/ai/production_state_test.go` | Aktivasi, kelanjutan, WAL/checkpoint restart, perlindungan dan uji penolakan |
| `scripts/prepare_native_continuation.py` | Verifikasi inventory/hash dan pembuatan/pemeriksaan paket produksi |
| `scripts/canonical_production_smoke.py` | Uji executable dari cwd kosong melalui input HTTP dan restart |
| `scripts/bridge_smoke.sh` | Smoke Bridge menggunakan direktori produksi terisolasi |
| `scripts/browser_sensor_smoke.mjs`, `scripts/visual_sensor_smoke.mjs` | Membaca payload Brain di envelope serving checkpoint |
| `.github/workflows/ci-cd.yml` | Pemeriksaan aktivasi dan cross-build produksi Android |
| `.gitignore` | Mengabaikan state deployment lokal dan staging |
| `README.md`, `docs/CANONICAL_AE1_ACTIVATION.md` | Cara memakai layanan, konfigurasi state, backup/rollback dan cakupan entry point |
| Direktori laporan ini | Bukti dan hasil penutupan tanpa menerbitkan state uji yang berevolusi |
