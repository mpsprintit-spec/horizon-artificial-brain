# HORIZON Gate G — integrasi runtime neural kanonik

## Tujuan

Mengganti pembatasan global native-versus-production dengan kontrak transaksi
berdasarkan konsistensi episode, sehingga sensor, observasi tertulis, inquiry,
recurrent thought, operasi native, dan pembelajaran konsekuensi memakai satu
Brain/N serta satu journal yang dapat dipulihkan secara deterministik.

**Keputusan: Gate G BELUM LULUS; G8 perlu perbaikan.**
Konfirmasi perbaikan lengkap memenuhi81/90 perbandingan anggaran;9 masih gagal.
G1–G7/G9/G10 lulus pada kasus terdaftar, tetapi keseluruhan Gate G tidak dipromosikan.
Controller native tetap opt-in. Main, checkpoint pengguna dan brain memory
pengguna tidak diubah. Review ada di [PR67](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/67),
masih draft dan bertumpu pada PR64.

## Implementasi

### Baseline autentik dan commit

Main diverifikasi ulang pada14 November2026 sebagai
`3a9e7ce3ffadff86c792050056ebea4eda48da0b`.
Metadata awal lengkap beserta hasil ilmiah yang diwarisi ada di
[BASELINE.json](BASELINE.json).

| Sumber | HEAD yang digunakan | Kondisi |
|---|---|---|
| PR60 | `87fc532c3d14fb0e9b7ee37523ad308aba503de3` | merged `32df59b9285d829ae73429c7f277621e63de7ce3` |
| PR61 | `0ef994a1d8842b36dbc6b994ecafcb0bf9951174` | draft; ancestor Phase III |
| PR62 | `ea69b2101004dbf10c07068e42436170c2adcf8d` | draft; ancestor Phase III |
| PR63 | `1837968ce0ba5707e3f15a36b14a7c87b1cc357f` | perbaikan startup diikutkan sebagai merge parent terpisah |
| PR64 | `0df3f19a33bd7de69ed496293591d734331c5e51` | draft; baseline penelitian; CI42567905600 sukses |
| PR65 | `e6c86c9d5c2d96288428ad4eec42b114c40b8e64` | merged `45ecb989b67da690b9bbc1b7a942bb8c55ca6f8e`, sudah tercakup |

Baseline pembanding sebelum implementasi adalah lokal `1e12332dcf2dc3f5ed974959ebc656b980ee87ae`
(published `609ee918aa8533e0aca5706dea6046043bdc09bc`): Phase III, main terbaru,
PR63, dan protokol awal; belum ada perubahan Gate G.

| Unit kerja | Commit lokal | Commit published |
|---|---|---|
| Implementasi awal dan kepemilikan parameter | `ca4994366b01637d3ba9346af14053921f114d30` | `af3b946197e3d7636d5aab5e2544a2003f31eb62` |
| Grounding otomatis menolak neuron kebijakan | `6de2fe868aeb94c5327aeffa1f44a5f59d74af18` | `3d74282f33d0c69d03413db3668ac46ada0309cc` |
| Recovery menolak alias populasi yang tidak aman | `a9e4ed75827fe328e259f47b058a42f0a1162b8b` | `d87fb5a3ed246376f68c55d100f3ac55f0c4417e` |
| Perbaikan sorting/aktivitas kosong | `b5b70faaf47d54992832d346546d66a1df7f0b00` | `af1217f2a04656bce72b6df5fa3d93081914d3f8` |
| Source freeze dan runner konfirmasi | `9cca3e645203ceff84de3f214923b210f478a66e` | `45b13f2e46c515f0ec5f67a49f2063c9217935a2` |

Tree konfirmasi terakhir adalah `99c1648594661cffe1a5f00ba974fedffd915e66`.
Tree lokal/published tiap unit diverifikasi sama. Git CLI tidak mempunyai
kredensial push; publication melalui koneksi repository mempertahankan seluruh
merge parent dan history. Perbedaan author/waktu menghasilkan SHA commit berbeda,
sedangkan tree source identik. Tidak ada force push atau merge ke main.

### Akar masalah dan perbaikan

Guard lama memblokir command produksi ketika native diinisialisasi, termasuk saat
idle. Jalur sensor/vector/import tertentu justru melewati guard tersebut. Menghapus
guard saja akan membiarkan aktivitas, eligibility dan parameter berubah di tengah
trajectory atau factual pair.

Audit dan uji negatif menemukan masalah tambahan yang diperbaiki:

1. Generic memory/plasticity/observed feedback mengubah confidence atau bobot
   parameter kebijakan native; checkpoint kemudian gagal validasi.
2. Receipt factual contrast yang sudah dipelajari tidak direkonstruksi oleh indeks
   prefix checkpoint; recovery setelah outcome kedua gagal atau kehilangan dedup.
3. Ketidakcocokan input/scope trial kedua baru ditolak saat outcome; kini ditolak
   sebelum Begin mengubah aktivitas atau menulis journal.
4. Replay gagal sebagian dapat meninggalkan runtime yang berpotensi dipublikasikan;
   kini recovery memakai runtime privat dan menandai kegagalan untuk recovery.
5. Experience, consequence dan spelling dapat menyebut ID neuron kebijakan langsung;
   kini referensinya diperiksa sebelum append.
6. Similarity matching otomatis dapat merekrut prototipe bias sebagai populasi
   observasi. Matcher kini mengecualikan role native dalam canonical N.
7. Checkpoint dengan checksum benar dapat tetap menyimpan alias populasi tersebut.
   Validasi semantik kini menolaknya sebelum Brain tujuan dimutasi; berkas sumber
   dipertahankan. Tidak ada penghapusan pengetahuan atau migrasi diam-diam.

[AUDIT.md](AUDIT.md) memuat inventaris efek baca/tulis tiap jenis transaksi,
provenance, dedup, sequence, schema dan batas perangkat.

| Area kode | Perubahan konkret |
|---|---|
| `runtime/command_admission.go` dan jalur input | admission sebelum append/projection/mutation; pemeriksaan ID internal |
| `knowledge/native_admission.go` dan kernel neural | metadata episode serta ownership role; seluruh parameter tetap di N |
| `runtime/native_control.go` | Begin membaca activation produksi yang sebenarnya; scope factual pair diperiksa lebih awal |
| journal/index/rebuild | urutan campuran, receipt pasangan, replay privat, failure poisoning |
| persistence/supervisor | temporary file restore unik, snapshot konsisten, publication setelah validasi |
| projection/native validation | input otomatis dan checkpoint tidak boleh mengalias role kebijakan sebagai populasi dunia |
| activation | typed sorting dan jalur coactivity nol dengan hasil kanonik identik |
| tests runtime/core/execution/production | replay lengkap, startup nyata, crash subprocess, gateway actuator dan isolasi legacy |

### Admission matrix

| Keadaan | Mutasi produksi | Command native | Inspeksi/checkpoint |
|---|---|---|---|
| Belum diinisialisasi | kontrak lama berlaku | explicit initialize | kontrak lama |
| Idle / episode sudah diselesaikan | sensor, teks, inquiry, thought dan world learning tervalidasi diterima | preflight native | di bawah lock pemilik |
| Episode aktif | ditolak sebelum WAL/mutasi | operasi episode yang sah | snapshot/read-only konsisten diizinkan |
| Stopped, menunggu outcome | ditolak | outcome/discard tervalidasi | diizinkan |
| Factual contrast pending | ditolak | trial kedua yang cocok / settlement / discard | pending state dipertahankan |
| Recovery diperlukan | ditolak | ditolak | tidak membuat checkpoint baru; supervisor rebuild privat |
| Replay | tidak mempunyai serving owner bersamaan | kontrak episode tetap diperiksa | kegagalan tidak dipublikasikan |
| Checkpoint/migrasi berjalan | diserialkan oleh lock pemilik | diserialkan | tanpa antrean persisten baru |

V5 controller, migrasi serving contract, surface rebinding dan tipe tak dikenal
ditolak setelah native aktif, termasuk saat idle. Composition yang sudah diaktifkan
melalui migrasi eksplisit sebelumnya dapat dipakai. Ini kontrak konsistensi
transaksi; tidak menentukan urutan berpikir atau jawaban kognitif.

## Pengujian

### Satu Brain, kontinuitas, journal dan replay

`TestGateGMixedReplay` menjalankan9 urutan ×3 seed (37,83,151), yaitu27 kasus:
sensor→native; native→sensor; sensor→inquiry→native; inquiry→native→outcome;
written observation→native; recurrent thought→native; learning→checkpoint→sensor;
checkpoint→restart→native continuation; native→thought→native.

Input sensor numerik benar-benar membentuk aktivitas awal. Begin tidak menggantinya
dengan observasi fixture. State tiap operasi berikutnya dibandingkan dengan hasil
operasi sebelumnya. Feedback teramati mengubah sinaps kanonik. Input baru dan episode
belajar berikutnya berjalan identik setelah restore. Pointer Brain, activation.Memory,
learning.Kb dan DNF.Brain tetap menunjuk Brain yang sama sepanjang eksekusi hidup.
Brain baru pada recovery adalah pengganti privat yang belum dipublikasikan.

Perbandingan mencakup seluruh canonical bytes: bobot/dinamika sinaps, policy,
aktivitas/context/prediction, episode/pair pending, receipt, evidence/action bindings,
composition, predecessor, waktu dan sequence. Genesis replay, checkpoint+suffix,
checksum schema20 dan kelanjutan belajar dibandingkan secara eksak. Tidak cukup
hanya membandingkan teks atau jumlah neuron.

Waktu penulisan envelope checkpoint, telemetry, lock, inspection fence dan cache
turunan dikecualikan dari checksum karena bukan input neural. Waktu event/observasi,
parameter belajar, pending state dan dedup tetap dicakup.

Satu logical sequence berlaku untuk semua command. Sensor intent+commit tetap dua
record fisik untuk satu revision dengan payload hash yang sesuai. Payload lengkap
tervalidasi sebelum append; fsync mendahului apply. Kegagalan setelah append membuat
runtime membutuhkan recovery. Satu outcome merekonstruksi seluruh update sinaptik
secara deterministik, tanpa event terpisah per sinaps. Tidak ada reward hipotetis
yang diubah menjadi bukti teramati.

### Crash, dedup dan perangkat fisik

Ada15 child-process crash menggunakan `os.Exit(86)` di boundary nyata:

| Boundary | State yang harus pulih |
|---|---|
| Persiapan payload / sebelum append | commit sebelumnya |
| Setelah write, sebelum fsync | record lengkap yang bertahan diputar satu kali |
| Setelah fsync, sebelum apply | transaksi durable diterapkan satu kali |
| Setelah apply, sebelum acknowledgment | retry tidak menggandakan belajar |
| Sensor intent / partial apply tanpa commit | ditolak; repair eksplisit membuat backup byte-identik terlebih dulu |
| Sensor commit durable | observasi dipulihkan lengkap |
| Episode aktif | trajectory yang sama dapat dilanjutkan |
| Outcome pertama / pending contrast | pasangan tetap pending; mutasi eksternal diblokir |
| Outcome kedua durable | satu update pasangan dan receipt konsumsi |
| Sebelum rename checkpoint | checkpoint lama + suffix durable |
| Setelah rename dan directory sync | checkpoint baru + suffix |
| Rebuild supervisor | tidak mempublikasikan runtime sebagian |

Exact retry outcome kedua tidak mengubah state; perubahan reward di bawah identitas
outcome yang sama ditolak. Uji gateway asli dengan counting mock mencatat satu
dispatch berizin saat live dan nol dispatch baru saat genesis/checkpoint recovery.
Pilihan operasi internal tidak memberikan izin perangkat.

Ini bukti crash proses; bukan uji pemadaman daya. Go TempDir berada pada tmpfs di
host ini, dan seluruh pemanggilan Sync tetap dijalankan. Uji append tambahan pada
overlayfs mengukur filesystem yang berbeda. Tidak ada klaim exactly-once lintas
perangkat di luar kontrak gateway yang sudah ada.

### Kompatibilitas, rollback dan arsitektur

Format tetap canonical7, checkpoint20, native WAL14, production WAL3 dan
composition WAL4. Model native1/2/3 diuji. Format container lama tidak dinaikkan
hanya karena command kini bercampur. Schema/identity/payload/sequence yang rusak
atau tidak lengkap ditolak. Format valid historis mempertahankan reader/law-nya;
interleaving/alias yang tidak aman ditolak, bukan dihapus atau dibetulkan diam-diam.

Tidak ada migrasi schema baru atau konversi otomatis V5→native. Downgrade yang tidak
sesuai ditolak. Rollback memakai checkpoint pra-opt-in dan prefix journal yang cocok;
journal lengkap sesudahnya dipertahankan. Restore tidak menimpa runtime native yang
sudah dikonfigurasi. Jalur startup PR63 diuji melalui startup produksi sebenarnya:
resume tidak menambahkan bootstrap kedua atau mengganti memory, dan episode native
serta pembelajaran konsekuensi dapat dilanjutkan.

Tes dependency closure runtime mengecualikan thinking/understanding/decision/reasoning
dan dispatch execution. Pulse/Think facade diuji tanpa pointer Decision legacy.
Compatibility/migration/diagnostic imports yang sah tidak dilarang secara global.
Tidak ada Brain pengetahuan kedua, controller simbolik baru atau LLM sebagai
kecerdasan Horizon.

### Performa dan verifikasi teknis

[PROTOCOL.md](PROTOCOL.md) mengunci fixture1k/4k/40k, sparse/dense, seed, boundary,
metrik dan anggaran sebelum konfirmasi. ABBA berurutan menggunakan Go1.23.2,
GOMAXPROCS1/GOGC100 dan harness identik. Batas tetap: thought p95 ≤baseline×1,15+0,10ms;
bytes/op ≤baseline×1,05+4096; stage native/persistence ≤baseline×1,20+0,10ms dan
bytes/op ≤baseline×1,10+4096; admission p95 ≤10µs dan ≤512B/op.

Konfirmasi awal pada sourceaf3b946 gagal9/90 perbandingan. Semua log, profil dan
hasil negatif dipertahankan di `evidence/performance-final/`. Probe penolakan sensor
baseline juga ternyata salah asumsi: jalur sensor lama melewati guard. Probe
Process yang diperbaiki menolak dengan error asli. Hasil awal tidak dilabel ulang
sebagai lulus. Profil diagnosis tidak menggantikan konfirmasi tersebut.

Perbaikan typed sorting/aktivitas kosong diuji terhadap general activity law lama
pada384 transisi state kanonik dan27 kasus urutan target/tie. Semua identik eksak,
termasuk eligibility, timestamp, rollback dan negative zero. Validasi dense4k
p9537,69→32,39ms; allocations41041→25026/op. Angka validasi ini bukan hasil akhir.

**Hasil konfirmatori baru: lengkap,81/90 memenuhi anggaran;9 gagal.**
Lihat [PERFORMANCE.md](PERFORMANCE.md) serta [matriks lengkap](evidence/performance-repaired-confirmation-20261114/SUMMARY.md)
untuk p50/p95/p99, B/op, allocations/op, GC dan peak RSS. Satu sesi persistence
terputus tanpa hasil akhir; log parsial dipertahankan dan command lengkap dijalankan
ulang tanpa perubahan source/sampel. Command yang telah selesai tidak diulang/dibuang. Percentile
adalah rerata statistik per-run, bukan pooled percentile. Native stage memakai20
pasangan/run; persistence3 operasi/run bersifat deskriptif. Peak RSS bersifat
kumulatif per proses; GC per-cell juga mencakup reseeding di luar timer thought.

Append overlayfs sebelumnya lulus anggaran tambahan yang dikunci:
p951,1455→1,1565ms,2084B/op tetap. Ini berbeda dari latency tmpfs dan tidak menjadi
klaim performa storage Android. Pembanding sensor bypass yang tidak tervalidasi
tidak dipakai untuk klaim kecepatan mixed runtime yang aman.

| Pemeriksaan pada source perbaikan | Hasil saat ini |
|---|---|
| Suite Go penuh | lulus; runtime168,975s |
| Race Gate G/activation/native ownership terarah | lulus |
| Race62 paket di luar cakupan CI | lulus; union62+7 paket dicatat eksplisit |
| Race7 paket CI pada source freeze terakhir | lulus; CI43550972852; bersama62 paket lokal mencakup69 paket |
| Vet dan plain build | lulus dari checkout terisolasi |
| Android arm64 CGO0 | CLI, native foundation, chat dan production lulus cross-build |
| Raw Gate G capture |83 record pass, nol failure;27 mixed +15 crash |
| Law/tie preservation |384 transisi dan27 urutan lulus |
| CI source freeze45b13f2 | CI43550972852 lulus validate dan monitor, termasuk native/recovery, nested adapters, trained dialogue, Android dan Bridge |

Full race lokal awal mencapai timeout20m pada curriculum lama, tanpa laporan race;
log tetap disimpan. Deadline dinaikkan40m tanpa menghapus tes atau mengubah anggaran
runtime. Cakupan race seluruh paket dilaporkan sebagai gabungan paket CI dan paket
lokal lainnya, bukan klaim bahwa perintah monolitik20m itu sukses. Error VCS dari
workspace luar diatasi dengan checkout terisolasi tanpa menonaktifkan VCS stamping.
Tidak tersedia perangkat Android fisik untuk pengukuran on-device.

## Keputusan per subgate

| Subgate | Status final iterasi ini | Bukti |
|---|---|---|
| G1 Unified Brain | fungsi terintegrasi opt-in; uji lulus | pointer bersama dan canonical bytes |
| G2 Unified journal | uji lulus |27 mixed order/seed dan recovery |
| G3 Admission | uji lulus | active/stopped/pending, alias ID/projection, V5/migration guard |
| G4 Replay deterministik |100% pada kasus valid terdaftar | genesis, suffix, checksum, continuation |
| G5 Crash safety |15 boundary proses lulus | subprocess dan expected durable state |
| G6 Physical safety | uji lulus | nol dispatch baru saat recovery |
| G7 Compatibility | kasus terdaftar lulus | native1/2/3, komposisi, downgrade dan rollback prefix |
| G8 Performance | **GAGAL / perlu perbaikan** |9/90 perbandingan gagal;81 memenuhi anggaran |
| G9 End-to-end | fungsi terbukti pada fixture integrasi | sensor→native→observed learning→restart→continuation |
| G10 Architecture | uji batas jalur aktif lulus | satuN, dependency closure dan facade |

Implemented, Tested, Functionally demonstrated, Integrated dan Production-ready
merupakan status berbeda. Hasil integrasi tidak mengaktifkan native sebagai default.
Keterbatasan PR64 pada signed encoder, exclusive first-pair credit precision50%,
dan observable-context retention tetap diwarisi; eksperimen Gate C/D/H tidak diulang
untuk menghasilkan klaim baru di pekerjaan ini.

## Langkah berikutnya, bukti dan reproduksi

ABBA baru dan seluruh verifikasi teknis sudah selesai. Langkah berikutnya adalah
profiling diferensial kasus1k sparse/dense, learning4k dan checkpoint4k; penyebab
seluruh delta tail latency belum terisolasi. Jangan menyimpulkan host scheduling
sebagai satu-satunya sebab. Perbaikan selanjutnya harus mempertahankan full-state
equivalence, kemudian mendapat source freeze dan konfirmasi baru dengan anggaran
yang sama. Jangan merge atau aktifkan default saat G8 gagal. Semua negative logs
serta hasil sebelum/sesudah perbaikan dipertahankan.

Arsip teknis awal: `evidence/technical-evidence.zip`,264423bytes,
SHA256 `f6dc4294ce112901e0abb8642be9e4931e031d210b9a1271a78be08e2b97362c`.
Arsip final: `evidence/Gate_G_Evidence_20261114.zip`; identitas/manifest ada di
`evidence/ARCHIVE.json`. [STATUS.md](STATUS.md) dan
[INTEGRATION_FREEZE_20261114.json](INTEGRATION_FREEZE_20261114.json) mencatat source,
provenance dan langkah lanjutan.

```sh
go test ./... -count=1
go vet ./...
go build ./...
HORIZON_GATE_G_EVIDENCE_DIR=NEW_RAW_DIRECTORY go test -json ./services/ai/activation ./services/ai/runtime ./services/ai/knowledge ./services/ai/execution ./services/ai/core ./services/ai -run '^TestGateG' -count=1
python3 scripts/run_gate_g_confirmation.py --baseline BASELINE_CHECKOUT --candidate CANDIDATE_CHECKOUT --output NEW_PERFORMANCE_DIRECTORY
python3 scripts/summarize_gate_g_confirmation.py NEW_PERFORMANCE_DIRECTORY
```

Gunakan directory baru; harness pada baseline harus identik dan hanya merupakan
file tes untracked. Jangan menyalin kode integrasi ke baseline atau menjalankan
fault/recovery eksperimen pada checkpoint/journal pengguna.
