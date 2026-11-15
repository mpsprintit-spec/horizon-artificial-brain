# Horizon — audit dan eksperimen fondasi neural native

Tanggal: 6 November 2026. Baseline autentik `d7cb140b5996f1bddffeebbda693ebadfd714841`.

**Status keseluruhan: belum siap sebagai general neural policy production.**
Implementasi menghasilkan pembelajaran pemilihan operasi dan penghentian yang
terukur pada lingkungan numerik kecil. Transfer ke jalur lebih panjang dan
perlindungan pengganggu aktif gagal. Kelulusan kompilasi/CI tidak menggantikan
gate fungsional tersebut. Optimasi substrate yang mempertahankan hukum lama
dipisahkan melalui [PR #60](https://github.com/mpsprintit-spec/horizon-artificial-brain/pull/60).

## 1. Sumber, batas pekerjaan, dan checkpoint

`main` diverifikasi masih pada baseline di atas sebelum perubahan. Tree awal
`0efc6126531ebac1d05043daa46682d1aa3d1396`; CI baseline berhasil. PR #58 telah
merged; PR #59 tentang gradient V5 masih terbuka dan tidak diubah. Architecture
Lock dibaca dari source, bukan laporan percakapan. Semua perubahan dikerjakan
pada branch/worktree terpisah. Checkpoint produksi, bundle bahasa terlatih,
checkpoint penelitian lama, dan jurnal pengguna tidak dimodifikasi/dihapus.

Ada dua unit perubahan:

1. Optimasi hukum plasticity lama: commit remote
   `87fc532c3d14fb0e9b7ee37523ad308aba503de3`, tree
   `6d7bfa05f3bec5e60489eeb3ab3dc4e58773649c`, PR #60.
2. Kontrak operasi, readout neural, pembelajaran temporal, persistence, dan
   lingkungan eksperimen **opt-in**. Tidak diaktifkan oleh default production.

MUSE, BMPS, dan V5 tetap ada sebagai sejarah/pembanding. Jalur baru tidak
menginisialisasi, memanggil, atau menyalin actor V5. Initializer menolak Brain
yang sudah memiliki controller V5; belum ada migrasi gabungan yang tervalidasi.
Artinya hasil ini bukan demonstrasi pada checkpoint produksi V5 yang telah
memiliki seluruh pengetahuan Horizon.

## 2. Audit source aktual dan akar kekurangan

| Jalur | Temuan terverifikasi | Akar kekurangan / tindakan |
|---|---|---|
| `activation/engine.go`, `deterministic.go`, `continuous.go` | Aktivitas recurrent dan propagasi melalui sinaps tersedia; thought dapat berjalan tanpa observasi baru | Jumlah siklus terutama dari caller; state prediction dan propagasi aktual masih memiliki pekerjaan berulang. Urutan ID aktif kini dipakai ulang dalam satu siklus |
| `runtime/continuous_service.go` | Penjadwal melanjutkan komputasi internal | Penjadwal/resource contract bukan bukti strategi berpikir yang dipelajari |
| `runtime/inquiry.go` | Agenda memakai bobot informasi 1, biaya .3, reversibility .2, social fit .2, prior .15 | Formula tetap bukan general neural policy. Jalur historis dan batas izin/risk tidak diubah |
| `runtime/inquiry_preference.go` | Preferensi inquiry dapat berubah akibat konsekuensi | Preferensi tindakan terbatas bukan pemilih komposisi transformasi neural umum |
| `activation/plasticity.go` | Prediction error dan consequence eligibility sudah ada | Consequence memindai graph dan eligibility tidak dengan sendirinya mengidentifikasi kontribusi kausal; local prediction kini mengunjungi union sumber relevan |
| `activation/activity_plasticity.go` | Local coactivity dan decay eligibility tersedia | Iterasi/sorting/alokasi seluruh graph meskipun sedikit aktif. Optimasi konservatif menghapus sorting/alokasi; varian lazy diuji terpisah |
| `activation/observed_feedback.go` | Feedback teramati dibedakan dari prediksi internal | Kontrak ini dipertahankan. Thought baru tidak mengubah frekuensi pengalaman atau menjadi outcome dunia |
| `runtime/inquiry_consequence_event.go`, `inquiry_outcome_command.go` | Provenance dan binding action/outcome tersedia | Belum membuktikan kredit rantai pilihan operasi mental. Jalur baru mengikat episode, transisi, action, dan observed outcome |
| `knowledge/memory_time.go` | Retention berdasarkan waktu fisik tersedia | Per-synapse update independen masih membutuhkan scan; visitor mengurangi overhead, tidak menghilangkan kompleksitas global |

Peta kemampuan operasi:

| Kemampuan | Mengubah state? | Empirical evidence? | Kendali adaptif saat ini |
|---|---|---|---|
| Propagasi, asosiasi, retrieval melalui aktivitas | Ya | Thought sendiri tidak | Native controller dapat memilih propagasi bounded |
| Prediksi/pembacaan uncertainty | Sebagian berupa readout/snapshot | Tidak | Jalur historis; belum semua dipetakan ke menu operasi baru |
| Recurrent state transition | Ya, hasil dipakai berikutnya | Tidak | `propagate`, `focus`, `relax` dapat dipilih berulang/berbeda urutan |
| Rekonstruksi dan hypothetical continuation | Tersedia sebagian lewat dinamika/readout lama | Bukan observasi nyata | Belum dibuktikan sebagai keterampilan general yang dipelajari |
| Perubahan fokus | Ya | Tidak | Lateral inhibition numerik `focus`; bukan aturan jenis masalah |
| Berhenti/melanjutkan | Mengendalikan episode komputasi | Tidak | `stop` dipelajari; batas MaxSteps tetap kontrak runtime |
| Observasi sensor/aksi fisik/inquiry | Melibatkan lingkungan | Hanya outcome teramati dan tervalidasi | Adapter/authorization lama; belum dikendalikan policy baru |

## 3. Implementasi yang dilakukan

### Substrate konservatif

`NeuralRegistry.VisitNodes` mengunjungi indeks alamat node canonical yang
berurutan. Indeks diurutkan saat cardinality berubah; tidak ada bobot di indeks.
Seluruh jalur penambahan membership diaudit append-only, sedangkan restore
membuat registry baru. `Nodes` mengembalikan salinan slice agar caller tidak
bisa mengubah indeks. Hanya update independen memakai visitor; reduction
berurutan dan serialization tetap deterministik. Kandidat awal dengan iterasi
map acak terbukti lebih lambat dan tidak diintegrasikan; urutan stabil menjaga
locality akses memori sekaligus menghindari sorting berulang. Callback berjalan di bawah
Brain lock dan tidak boleh memasuki registry kembali atau mengubah topologi.
`OutboundAll` tidak lagi membuat slice per target. Prediction-error local
membatasi sumber pada union state prediksi dan observasi. Hukum weight,
confidence, timestamp, eligibility, serta format persistence lama tetap sama.

Frozen reference test menyalin hukum baseline d7cb140 sebagai oracle, lalu
membandingkan **seluruh canonical bytes** setelah 100 transisi campuran pada
kedua encoding lama/local. Ini lebih kuat daripada hanya membandingkan reward.

### Kontrak operasi dan state continuity

`activation/native_operation.go` menyediakan preflight dan commit operasi.
Preflight memvalidasi batas active nodes/read synapses sebelum WAL ditulis.
Commit menerima plan hanya sekali, dari engine yang sama, dengan state dan
confidence yang belum berubah. Operasi memakai aktivitas dan sinaps canonical;
hasil menjadi state operasi berikutnya. Tidak ada stage sequence wajib.

Empat affordance komputasi adalah `propagate`, `focus`, `relax`, dan `stop`.
Ini transformasi numerik umum, bukan deduksi/matematika/analogi yang dikodekan.
Operator propagasi baru memakai weight, confidence, temporal attenuation dan
squashing graph native, tetapi **tidak identik** dengan seluruh hukum legacy
`CognitiveThink` (threshold, konteks dan plasticity historisnya). Perbedaan ini
harus diperhitungkan saat membaca pembanding legacy.

### Readout dan pembelajaran melalui N

`knowledge/native_control.go` membuat satu bias dan empat neuron effector per
operasi pada registry Brain yang sama. Sources adalah alamat neuron existing.
Semua parameter Q berada pada sinaps biasa source→effector di canonical N;
tidak ada array bobot persisten, Brain kedua, tabel strategi, GRU tersembunyi,
atau classifier jenis tugas. Readout adalah rata-rata populasi linear atas
aktivitas saat ini. Sign memakai kanal inhibitory, dengan magnitude bounded.

Training memakai softmax; evaluasi greedy tidak memperbarui parameter.
Hanya satu episode pending didukung; banyak episode dengan outcome tertunda
secara bersamaan belum didukung. MaxSteps membatasi episode; penghentian karena budget ditandai `Truncated`,
berbeda dari pilihan `stop`. Overrides hanya untuk evaluasi baseline/ablation;
override pada training ditolak. Budget kerja bukan alasan untuk mengklaim
resource latency sudah terjamin pada Android.

Pembelajaran memakai **offline on-policy Sarsa(lambda), forward lambda return**
dengan Q episode yang dibekukan sampai outcome. Lambda 0 adalah TD0, lambda 1
adalah Monte Carlo. Return mengurangi tariff operasi yang tercatat. Gradient
sementara hanya berisi alamat sinaps canonical untuk action yang dieksekusi
dan sumber nonzero. Ini indeks eligibility komputasional, bukan database
pengetahuan kedua. Tidak mengganti semua aturan plasticity Horizon.

Referensi konsep: Sutton & Barto,
[Reinforcement Learning](http://incompleteideas.net/book/the-book-2nd.html).
Implementasi ini bukan reproduksi e-prop, MUSE, BMPS, atau actor V5.

### Konsekuensi dan keamanan evidence

`ApplyNativeControl` mengikat state sebelum/sesudah, probabilitas dan nilai
action, waktu, biaya, episode, trace digest, source dan outcome ID. Outcome
harus setelah episode berhenti, memiliki provenance observed, serta cocok
dengan trace. ID outcome single-use lintas command/episode/restart; retry yang
identik idempotent. Hypothetical outcome, NaN, stale trace, training dari greedy,
clock rollback dan resource overflow ditolak sebelum mutasi/WAL.

Label provenance bukan bukti fisik kriptografis: adapter tepercaya masih
bertanggung jawab bahwa pengukuran benar-benar terjadi. Tidak ada API actuator
di jalur ini. Replay menjalankan ulang komputasi internal, tidak tindakan fisik.
Authorization/risk/reversibility inquiry produksi tidak dilonggarkan.

`ApplySparseObservedPlasticity` adalah utility eksperimental lazy decay:
effective eligibility dihitung saat digunakan, dengan jalur sparse atau dense
berdasarkan occupancy. Hasil dibandingkan dengan eager law sampai toleransi
2e-15. **Utility ini belum menggantikan pipeline observed learning produksi**;
consequence global historis dan seluruh pembacaan timestamp harus dimigrasikan
secara konsisten sebelum penggantian tersebut aman.

## 4. Desain dan anggaran eksperimen

Protokol ditulis sebelum eksperimen dan keputusan validation dicatat di
[PROTOCOL.md](PROTOCOL.md). Semua training/evaluasi menggunakan Go dan runtime
serta substrate asli. Python hanya meringkas hasil dan mengukur proses Go.

- Fixture: 28 neuron dunia (7 populasi × 4 unit), 112 sinaps dunia; setelah
  controller terdapat 45 neuron dan 576 sinaps, termasuk 464 parameter policy.
- Struktur chain diberikan oleh fixture; **struktur dunia ini tidak dipelajari**.
- Training: 4.000 episode/model, maksimum 8 keputusan/episode, temperatur .12,
  learning rate .15, gamma 1, biaya .04/nonstop operation.
- Validation: seed 37/83, stream 200000; lambda 0/.8/1. Lambda 1 dipilih dari
  validation; static3 dipilih sebagai baseline tetap sebelum final test.
- Final: seed independen 101/211/307/401/503; stream holdout 900000+seed,
  256 kasus/distribusi. Lima model lambda1 dan lima lambda0, 40.000 episode
  training final; tidak menghitung episode validation/adaptation sebagai final.
- Train input chain depth 0–3, amplitude .65–1 dan noise rendah. Policy tidak
  menerima label tugas, jawaban, kedalaman yang benar, atau state masa depan.
- Reward lingkungan: signal terminal dikali exp(-3×noise), dengan regime sign;
  net return mengurangi tariff. “Accuracy” di laporan berarti reward≥.45,
  **bukan akurasi bahasa, matematika, atau kecerdasan umum**.
- Semua arm pada sebuah seed mengevaluasi sampel tugas identik. CI95 memakai
  Student-t df=4 atas lima mean model, paired untuk perbedaan; bukan menganggap
  1.280 episode sebagai 1.280 model independen.

CPU waktu eksperimen tidak dipakai sebagai benchmark latency karena training
dan regression checks pernah berjalan bersamaan. Benchmark latency dijalankan
terpisah dan sequential. Hasil final dihitung ulang setelah perbaikan recovery
dan dedup; return/parameter persis mereproduksi run sebelumnya.

## 5. Hasil fungsional

| Arm, holdout depth 0–3 | Mean net return | Accuracy | Operasi nonstop | Learned stop / budget |
|---|---:|---:|---:|---:|
| Untrained greedy | -0.013172 | 27.03% | 4.9844 | 38.52% / 61.48% |
| Random | 0.018683 | 11.72% | 2.8086 | 89.30% / 10.70% |
| Static3 (validation-selected) | 0.444452 | 83.13% | 3.0000 | Stop diprogram |
| Trained greedy | **0.508751** | **84.84%** | **1.4859** | **100% / 0%** |
| Legacy Think(2) | 0.316406 | 42.66% | 2.0000 | Siklus diprogram |

Trained − static3 = **+0.064300**, paired CI95 **[0.062821, 0.065778]**.
Trained return CI95 [0.494799, 0.522703]. Training stochastic return 0.413436
[0.408583, 0.418289] dilaporkan terpisah dari greedy.

336/464 parameter berubah pada setiap seed; maksimum perubahan signed stored
magnitude .0477–.0528 (nilai readout memakai scale 8). Semua DynamicState sinaps
dunia tetap identik dalam training dan evaluasi controller. Perubahan terjadi
pada parameter policy di N, bukan lewat penggantian jawaban oleh adapter.

Static4/static5 memberi reward yang sama dengan static3 pada tugas ini tetapi
return turun menjadi .404452/.364452 akibat biaya. Trained menghemat siklus
dan memilih stop. Ini bukti manfaat komputasi adaptif pada fixture, bukan bukti
Horizon sudah menguasai seluruh operasi mental atau semua biaya yang berubah.

Lambda1 − lambda0 hanya **+0.000340**, CI95 **[-0.001572, 0.002252]**.
Tidak ada bukti meyakinkan Monte Carlo mengungguli TD0 di sini. Karena itu
tidak ada dasar untuk mengganti plasticity production dengan satu pemenang
berdasarkan hasil ini.

## 6. Transfer, perubahan kondisi, dan retention

| Distribusi baru | Trained | Static3 | Kesimpulan terbatas |
|---|---:|---:|---|
| Kombinasi populasi baru | .534314 | .522487 | Transfer sempit teramati, bukan bukti semua urutan |
| Noise lebih tinggi | .271549 | .158235 | Masih menguntungkan pada rentang uji |
| Amplitude lebih rendah | .224376 | .188417 | Masih menguntungkan pada rentang uji |
| Biaya ×4 | .330439 | .084452 | Return positif; policy tidak menerima sinyal biaya baru, jadi ini bukan bukti cost-awareness |
| Chain depth 4–5 | **-.034219** | -.120000 | **Gagal**; static5 mencapai .366584 |
| Reward sign dibalik | **-.627626** | -.684452 | **Gagal** menghadapi perubahan yang belum teramati |

Akar kegagalan chain panjang: readout berbasis source ID tidak berbagi fungsi
nilai antarposisi graph. Delapan neuron tahap 0/1 tidak pernah aktif saat
training; 128 koneksi policy dari neuron tersebut tidak dilatih. Hubungan graph
yang sama pada posisi baru tidak cukup untuk membuat readout linear memakai
strategi yang sudah dipelajari di posisi lain. Menambah siklus secara paksa
bukan perbaikan generalisasi policy.

Setelah **512 pengalaman tambahan**, bukan zero-shot transfer:

- Longer-chain holdout: mean .267654, CI95 [.069993, .465314]; seed101 tetap
  -.0099. Retention tugas sebelumnya .487098 [.435192, .539004]. Adaptasi belum
  konsisten antar-seed.
- Reversal holdout: -.076129 [-.100340, -.051918], lebih dekat stop; retention
  tugas lama **-.022249** [-.046440, .001942]. Sinaps dunia tidak hilang, tetapi
  perilaku lama hilang setelah objektif yang berlawanan dipelajari. Tidak boleh
  menyebut ini perlindungan continual learning yang sudah selesai.

Regime biaya/sign tidak disertakan sebagai observasi baru. Tidak masuk akal
menuntut policy mengetahui perubahan tersembunyi sebelum feedback; hasil ini
tetap membatasi klaim transfer. Belum diuji: strategi terbuka pada input bahasa,
task baru selain lingkungan numerik ini, komposisi panjang tak terbatas,
identifikasi perubahan regime, dan order-transfer luas.

## 7. Audit kredit kausal

Menghapus sinaps dunia tahap3→4 pada replika terlatih menurunkan return dari
.508751 menjadi **.251537** [.234055, .269019]. Menghapus tahap0→1 yang tidak
terlibat pada holdout ini menghasilkan **.508751 persis**. Bobot policy tidak
diubah pada ablation. Arm bernama `ablated` dalam raw JSON adalah forced relax
8 kali (operation substitution); jangan menyamakannya dengan edge ablation.

Probe terpisah menahan action trajectory tetap (propagate lalu stop), dengan
reward hanya signal terminal. Aktivitas pengganggu self-loop tidak mempunyai
efek reward pada trajectory ini. Setelah feedback positif:

| Ukuran probe, tiap seed | Hasil |
|---|---:|
| Parameter input relevan diperbarui | 32 |
| Parameter pengganggu aktif diperbarui | **32** |
| Parameter input tidak aktif diperbarui | 0 |
| Precision jumlah sinaps berubah berdasarkan causal label fixture, tanpa bias | **50%** |
| Efek reward pengganggu dengan action tetap | 0 |

Eligibility berbasis feature/action derivative membuktikan jalur mana yang
memengaruhi readout, **bukan** feature mana yang menyebabkan keberhasilan dunia.
Pengganggu bisa memengaruhi policy meskipun tidak memengaruhi reward saat action
dikondisikan. Atribusi struktural tersebut belum teridentifikasi. Gate D gagal;
tidak ditutupi oleh test “inactive synapse unchanged”.

Perbaikan berikutnya harus menguji intervensi terkontrol/paired outcomes atau
representasi kausal yang dipelajari, dengan data baru yang dipisahkan dari
holdout ini. Jangan memasukkan causal label fixture ke policy, menghapus
feature berdasarkan jenis masalah, atau menganggap simulasi sebagai evidence.

## 8. Persistence, validasi, dan kompatibilitas

Native opt-in memakai canonical **7**, checkpoint **20**, WAL **14**. Metadata
menyimpan referensi neuron, config, dan episode bounded; parameter tetap di
sinaps canonical. Loader memeriksa topologi, sign/magnitude, Q/propensity,
continuity, stop/budget, checksum dan kecocokan state terakhir. Tanpa native
state, schema/serialization historis dipertahankan. Tidak ada migrasi diam-diam.

Validasi yang selesai:

- Full `go test ./...`, `go vet`, `go build`; local memakai `-buildvcs=false`
  karena stamping VCS worktree sempat gagal di lingkungan ini.
- Race tests activation/knowledge/runtime pada native, lazy law, replay dan
  concurrent canonical read; full regression lama ikut lulus.
- Frozen d7cb140 canonical-byte equivalence untuk perubahan konservatif.
- Finite-difference readout, reward positif/negatif, TD0/Monte Carlo delayed
  update, perlindungan input inactive, rejection false evidence, action override,
  resource overflow tanpa mutasi, plan stale/foreign/reused, idempotency/restart.
- Crash subprocess benar-benar exit setelah `fsync` sebelum apply; recovery
  sama dengan uninterrupted execution.
- Bug kosong `{}` versus `null` pada hash state ditemukan dan diperbaiki;
  normalisasi ini diperlukan untuk checkpoint episode dengan aktivitas kosong.
- Cross-build Android arm64 berhasil; **belum benchmark pada perangkat Termux**.

Audit WAL yang disimpan memakai seed independen, 64 episode training sebelum
checkpoint + 5 episode suffix. Sequence akhir 320; canonical, activation dan
checksum recovery identik. Tidak mengeksekusi aksi fisik. Checkpoint final lima
seed juga berhasil direstore untuk adaptation dan ablation.

Kontrak jurnal lama mensyaratkan prefix lengkap untuk offset recovery.
Checkpoint dari training tanpa WAL tidak boleh dipasangkan dengan log kosong
dan disebut kelanjutan jurnal yang valid. Percobaan itu ditolak sebagaimana
mestinya. `RebuildRuntimeFromCheckpointAndJournal` juga sengaja tidak memasang
writer; writer dipasang lagi sesudah replay sebelum checkpoint baru dibuat.

Recovery cache dedup dibangun dari processed event index yang diautentikasi,
bukan dari database bobot lain. Index eligibility learning hanya sementara dan
direkonstruksi dari episode. Lazy physical eligibility eksperimental belum
dipromosikan menjadi format persisted global.

Rollback: runtime/default produksi tidak mengaktifkan controller ini. Untuk
binary lama, gunakan backup checkpoint schema lama, bukan menurunkan nomor
schema baru atau membuang metadata. Pertahankan checkpoint lengkap beserta WAL
prefix. Revert optimasi konservatif tidak memerlukan migrasi data.

## 9. Performa

Lihat [PERFORMANCE.md](PERFORMANCE.md) untuk paired benchmark, p50/p95/p99,
alokasi, throughput, GC, peak RSS, profiling, journal, checkpoint dan replay.
Benchmark memakai synthetic graph 1k/4k/40k, fanout4; bukan checkpoint 40k neuron
terlatih pengguna. Latency tidak boleh digeneralisasi langsung ke Android.

## 10. Gate dan arti status

| Gate | Status fondasi baru | Bukti / penghalang |
|---|---|---|
| A — Architecture | Lulus pada lingkup eksperimen | Satu Brain/N, sinaps canonical, tanpa V5/LLM/rule skill; belum migrasi Brain V5 existing |
| B — Operations | Lulus untuk empat primitive | State continuity, komposisi, repeat, stop, tanpa input baru; kelengkapan seluruh readout/operasi historis belum dibuktikan |
| C — General policy | **Gagal** | Belajar dan mengalahkan static baseline, tetapi longer-chain transfer gagal |
| D — Causal learning | **Gagal** | Kredit temporal bekerja; precision pengganggu aktif 50%, retention reversal buruk |
| E — Performance | Diukur; target harus dibaca dari benchmark | Percepatan komponen bukan otomatis ≥3× whole thought |
| F — Persistence | Lulus pada kontrak yang diuji | Checksum, replay, fsync crash, stale/duplicate guards, standalone checkpoint restore |
| G — Integration | Terpisah | Optimasi konservatif melalui PR #60; controller eksperimen tetap draft/off |

| Istilah | Yang benar-benar dicapai |
|---|---|
| Implemented | Primitive, readout N, temporal update, provenance, budgets, versioned checkpoint/WAL, harness |
| Tested | Unit/integration/regression/race/recovery/Android cross-build |
| Functionally demonstrated | Learned operation selection, biaya siklus, stop, delayed-return update pada fixture |
| Generalized | Hanya transfer sempit noise/amplitude/kombinasi; generalisasi luas **belum** |
| Performance verified | Ukuran workload dan hasil mentah di PERFORMANCE.md; bukan jaminan semua skala |
| Integrated | Hanya perubahan konservatif yang memenuhi gate masing-masing; policy default tetap historis |
| Not yet proven | General neural policy penuh, identified causal credit, retention luas, learning representasi dunia baru, performa HP |

Prioritas lanjutan adalah memperbaiki representasi value yang tidak berbagi
pengetahuan antarposisi, menguji causal credit dengan intervensi yang benar-benar
teramati, dan menyatukan lazy timestamps dengan semua consumer eligibility.
Ketiganya perlu eksperimen baru dan checkpoint terisolasi; tidak boleh dinyatakan
selesai hanya karena implementation ini sudah berjalan.

## 11. Reproduksi

```sh
go test ./...
go test -race ./services/ai/activation ./services/ai/knowledge ./services/ai/runtime -run '^Test(Native|SparseObserved|HistoricalPlasticity)' -count=1
go run ./cmd/horizon-native-foundation -out results/final-verified-mc -seeds 101,211,307,401,503 -train 4000 -test 256 -lambda 1 -stream 900000
go run ./cmd/horizon-native-foundation -out results/final-verified-td0 -seeds 101,211,307,401,503 -train 4000 -test 256 -lambda 0 -stream 900000
go run ./cmd/horizon-native-foundation -recovery-audit -out results/journal-recovery
python3 scripts/summarize_native_foundation.py results results/summary.json
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build ./cmd/horizon-native-foundation
```

Gunakan direktori baru untuk recovery audit agar WAL lama tidak ditimpa.
Raw seed results, checkpoint, retained WAL, logs, profiles dan checksum disimpan
sebagai arsip evidence. [results.json](results.json) berisi agregat lengkap dan
CI yang dapat dihitung ulang dari raw JSON. PR menyimpan source yang diperlukan
untuk menghasilkan kembali seluruh fixture; tidak perlu server eksternal.
