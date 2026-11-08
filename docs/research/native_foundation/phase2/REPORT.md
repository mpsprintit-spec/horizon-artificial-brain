# Phase II — policy relasional dan kredit dari percobaan berpasangan

Status: perubahan penelitian opt-in; **belum memenuhi seluruh gate fondasi**.
PR60 tetap menjadi production baseline. PR61 tidak dihapus atau ditulis ulang.
Hasil Phase I tetap disimpan di direktori induk; hasil di sini adalah eksperimen baru.

## Source autentik dan keputusan

Main diverifikasi pada `32df59b9285d829ae73429c7f277621e63de7ce3`, tree
`6d7bfa05f3bec5e60489eeb3ab3dc4e58773649c`, CI42355859550 success.
PR61 diverifikasi pada `0ef994a1d8842b36dbc6b994ecafcb0bf9951174`, tree
`58ca1592cf9f60907a334346170d8138d018b920`, PR CI42252885318 success.
Branch terisolasi dibuat langsung dari remote PR61, kemudian main di-merge
(`f33b5bb`). Tidak ada rebase paksa, penghapusan checkpoint, atau migrasi data.
`FINAL_ARCHITECTURE_LOCK.md` dibaca; SHA256 ada di PROTOCOL.md.

## Akar masalah dan implementasi

1. **Gate C:** Q lama mempunyai parameter berbeda untuk setiap alamat neuron.
   Aktivitas di alamat baru tidak menggunakan bobot yang sudah dipelajari.
   Encoder baru memakai empat momen lokal yang dijumlahkan: aktivitas dan
   kuadrat aktivitas, masing-masing dibobot konektivitas resiprokal dan
   nonresiprokal. Normalisasi fan-out mengurangi ketergantungan lebar populasi.
   Tidak ada angka kedalaman, label goal, identitas task, atau daftar urutan
   operasi dalam encoder. Readout yang dipelajari terdiri dari 80 sinaps biasa
   dalam canonical N. Feature units adalah alamat parameter, bukan Brain lain.
   Ini **encoding numerik tetap dengan proyeksi neural yang dipelajari**;
   representasi relasional end-to-end belum diimplementasikan.

2. **Adaptasi:** softmax .12 masih sering memilih operasi yang merusak sinyal
   saat menjalani chain panjang. Return aktual untuk melanjutkan turun, lalu
   policy menguatkan stop. Mengganti lambda1 dengan TD0 pada pilot tidak
   memperbaikinya. Opsi eksplorasi epsilon-greedy .1 memperbaiki kegagalan
   tersebut pada pilot dan holdout. Evaluasi greedy tetap terpisah. Tidak ada
   pengaturan hardcoded bahwa sebuah task harus propagate atau harus stop.

3. **Gate D:** satu error outcome memperbarui seluruh fitur yang aktif pada
   readout. Aktivitas kausal dan pengganggu yang berkorelasi sama-sama berubah.
   Kontrak baru menunggu dua percobaan nyata dengan urutan tindakan sama dan
   parameter yang belum berubah. Pembelajaran menggunakan selisih return dan
   selisih fitur. Fitur yang sama di kedua percobaan membatalkan satu sama lain.
   Tidak ada mask relevansi dari fixture. Perbandingan ini memerlukan asumsi
   bahwa kondisi eksternal benar-benar dipasangkan oleh adapter. Label pair
   sendiri tidak membuktikan asumsi tersebut.

   Loss: `0.5 * ((G_A-G_B) - (Q_A-Q_B))^2`. Lambda harus 1 pada jalur ini.
   Bias/fitur bersama tidak menerima update. Trial nol yang benar-benar
   dijalankan mengkalibrasi nilai absolut/biaya; selisih saja tidak dapat
   mengidentifikasi intercept. Jalur Sarsa lama tetap tersedia. Ini bukan
   pengganti umum untuk identifikasi kausal dari observasi tak terkontrol.

4. **Biaya runtime:** snapshot metadata sebelumnya melakukan marshal/unmarshal
   JSON berulang pada setiap perintah. Diganti deep copy bertipe yang menjaga
   isolasi snapshot dan bentuk serialisasi. Traversal relasional mengikuti
   state aktif dan edge lokal, bukan semua neuron. Readout melaporkan jumlah
   pembacaan parameter dan edge struktural. Versi2 membatasi jumlah keduanya
   sebelum menulis WAL; versi1 mempertahankan batas historis untuk operasi.

## Jalur neural dan konsekuensi

State aktivasi saat ini dan edge canonical menghasilkan fitur lokal; parameter
readout canonical menghasilkan nilai operasi. Sampling pelatihan atau greedy
memilih salah satu `propagate`, `focus`, `relax`, `stop`. Preflight menghitung
transisi, WAL merekam perintah, commit mempertahankan state untuk operasi
berikutnya. Outcome teramati mengikat digest trajectory dan ID unik. Pembaruan
mengubah sinaps readout pada Brain yang sama. Thought tidak membuat empirical
evidence, dan native command tidak mempunyai jalur untuk aktuator fisik.
Outcome eksperimen di sini adalah skor lingkungan Go setelah operasi benar-benar
dieksekusi, bukan observasi dunia fisik atau hasil prediksi yang dianggap fakta.

Operasi yang sudah ada dipertahankan. Tidak ada pipeline kognitif wajib.
Urutan propagate→stop pada probe adalah intervensi evaluator yang dicatat
secara eksplisit, bukan strategi bawaan policy. Pada episode intervensi,
probabilitas policy yang dicatat bukan propensity penugasan eksperimen.

## Hasil konfirmatori

Lima seed baru,2500 episode/model,256 kasus/distribusi. Protokol dibekukan
sebelum holdout dijalankan. Kandidat dan kontrol identitas memakai epsilon .1
serta lambda1; ablation softmax memakai parameter lain yang sama.

| Evaluasi greedy | Per-NodeID | Relasional |
|---|---:|---:|
| Distribusi tugas training, sampel baru |0.514357|0.513248|
| Chain4–5, training hanya0–3 |−0.106517|0.388639|
| NodeID dan posisi graph baru |0.008409|0.513248|
| Populasi dua kali lebih lebar |0.083479|0.510743|
| Graph baru, chain6–7 |−0.050125|0.308639|
| Kombinasi populasi baru |−0.005175|0.545302|
| Noise baru pada graph baru |−0.048814|0.230313|
| Amplitude baru pada graph baru |−0.023529|0.249786|

Hasil ini menunjukkan penggunaan ulang pembelajaran pada keluarga graph yang
memiliki struktur lokal yang sama. Kebijakan tidak mengungguli kontrol per-ID
pada distribusi training; manfaat utama adalah transfer. Data lengkap, confidence
interval dan perbandingan berpasangan tersedia di `results.json`. Nilai ini
bukan persentase kecerdasan atau bukti kemampuan reasoning umum.

Return training stochastic relasional adalah 0.441193, CI95% [0.426763,0.455623].
Evaluasi greedy pada sampel baru adalah 0.513248, CI95% [0.506082,0.520414],
dibanding static3 0.452717; rata-rata operasi nonstop 1.486719. Seluruh 80 parameter
readout berubah. Bobot dunia tidak dilatih dalam eksperimen ini. Pada chain 4–5,
selisih berpasangan terhadap kontrol identitas adalah +0.495156,
CI95% [0.190943,0.799370]; pada chain 6–7, +0.358764,
CI95% [0.225129,0.492400].

Setelah 512 pengalaman baru pada chain panjang:

- Relasional epsilon: return tugas baru 0.384454, retensi tugas lama 0.508927.
  Checkpoint sebelum adaptasi diuji ulang pada **kasus retensi yang sama**:
  0.509130 sebelum, 0.508927 sesudah; delta −0.000203,
  CI95% [−0.000768,0.000361]. Ini bukti retensi dalam distribusi tersebut,
  bukan bukti bahwa seluruh pengetahuan lama terlindungi.
- Relasional softmax: return tugas baru 0.005677, retensi 0.143220;
  delta retensi pada kasus yang sama −0.365911, CI95% [−0.390825,−0.340996].
- Reversal reward tanpa penanda konteks: retensi masih runtuh ke 0.006632.
  Dua reward berlawanan pada state identik tidak dapat dipilih kembali sesuai
  konteks jika konteks itu tidak teramati. Continual learning umum belum lulus.

Semua arm dan interval retensi berpasangan ada di `retention_results.json`.

## Kredit kausal: kemajuan dan hasil nol

Lima seed berbeda dari Phase I;1088 episode faktual per arm (512 pasangan +64
anchor), dengan data eksekusi yang sama untuk paired dan unpaired learning.
Probe parameter memakai readout identitas agar parameter relevan dan tidak
relevan bisa dibedakan; precision parameter tidak berlaku langsung untuk bobot
bersama yang memengaruhi kedua populasi.

| Metrik | Unpaired | Paired |
|---|---:|---:|
| Parameter relevan berubah |32|32|
| Pengganggu aktif berubah |32|0|
| Parameter tak aktif berubah |0|0|
| Precision non-bias |50%|100%|
| Sensitivitas selisih Q terhadap nuisance |0.0000218644|0.00000001095|
| Return greedy setelah belajar |0.611430|0.611430|

Paired learning memperbaiki atribusi dan invariansi readout pada probe ini.
**Keunggulan perilaku terhadap pembanding belum terbukti:** selisih return
akhir persis0 pada semua seed. Jangan menyebut Gate D lulus penuh. Precision
single-outcome dan paired-outcome juga mempunyai anggaran informasi berbeda;
perbandingan perilaku menggunakan jumlah eksekusi yang sama.
Reward tertunda melintasi beberapa operasi, reward negatif, pembatalan fitur
bersama, dan perlindungan fitur tak aktif diuji secara numerik/unit. Pembelajaran
kausal umum, gangguan berkorelasi dengan confounder yang tidak dikontrol,
intervensi edge yang luas, dan end-to-end policy relasional dengan ketepatan
kredit kausal teridentifikasi masih belum terbukti.

## Uji lanjutan: batas biaya dan kalibrasi nilai

Probe pertama hampir selalu memiliki satu operasi optimal karena amplitude
minimum0.3 jauh di atas biaya0.04. Evaluasi baru memakai checkpoint beku,
amplitude0–0.08, dan nuisance0–1.8. Paired loss saja menghasilkan keputusan
buruk pada sebagian besar seed: melindungi fitur bersama tidak otomatis
mengkalibrasi nilai absolut tindakan.

Kandidat kedua `contrast_mean=true` melatih sinaps bias dari mean return faktual
kedua trial, sementara bobot fitur tetap diperbarui dari perbedaan kedua trial.
Tidak ada label causal fixture atau reward prediktif yang ditambahkan. Pilot
512 pasangan membaik, tetapi pilot32 pasangan masih gagal. Karena itu kedua
anggaran diuji pada lima seed konfirmatori **baru**, terpisah dari holdout pertama.

| Pasangan training | Unpaired | Paired awal | Paired + kalibrasi mean |
|---|---:|---:|---:|
|32|−0.137026|−0.300098|−0.106227|
|512|0.009868|−0.001666|0.008139|

Pada512 pasangan, kalibrasi memperbaiki paired awal sebesar0.009806,
CI95%[0.003300,0.016311]. Terhadap unpaired, selisihnya−0.001729,
CI95%[−0.004919,0.001462]: **belum unggul**. Pada32 pasangan, semua arm
masih memiliki kegagalan serius dan interval lebar. Hasil lengkap ada di
`calibration_results.json`.

Ini menunjukkan kekurangan bukan hanya jumlah data: relative credit dan
kalibrasi nilai absolut merupakan masalah berbeda. Encoder/value linear serta
baseline global belum cukup untuk semua state dan operasi. `focus` juga bisa
membuat nuisance berpengaruh melalui inhibisi global; klaim zero-effect hanya
berlaku pada intervensi dengan urutan propagate→stop yang dikendalikan.
Tidak ada alasan ilmiah untuk menaikkan status Gate D hanya dari precision100%.
Eksperimen dihentikan setelah diagnosis ini; tidak ada training besar tambahan
untuk mengejar angka positif pada holdout yang sama.

## Persistence dan kompatibilitas

Container canonical7/checkpoint20/WAL14 dipertahankan. Konfigurasi model baru
secara eksplisit memakai `native-operation-control-v2`; binary lama menolaknya.
Tidak ada migrasi otomatis dari model v1 atau V5. Parameter tetap pada sinaps
canonical. Satu pending contrast adalah pengalaman sementara terbatas, bukan
model bobot lain atau database pengetahuan paralel.

Checkpoint menyimpan pending trial, recurrent activation, konfigurasi, trace dan
parameter. Trial kedua harus mengikat episode, sumber, ID outcome, urutan tindakan,
dan parameter yang masih sama. Learning lain diblokir selama pending; pasangan
bisa dibatalkan secara eksplisit melalui perintah journaled tanpa menghapus
pengalaman lama. Outcome dan contrast yang sudah selesai tidak dapat dilatih ulang.

Uji baru mencakup restart di antara dua trial, replay suffix, retry outcome,
penolakan contrast terpakai, discard salah/benar, batas read sebelum mutasi,
serta subprocess exit sebelum append dan sesudah fsync sebelum apply. Recovery
pada kedua batas menghasilkan state yang diharapkan. Jalur lama untuk checksum,
checkpoint korup, journal terpotong, dan replay tetap menjalani regression tests.

## Gate dan batas klaim

| Gate | Status Phase II |
|---|---|
| A Architecture | Lulus untuk perubahan opt-in: satu Brain/N, tanpa V5 atau engine baru |
| B Operations | Lulus dalam primitive dan komposisi yang diuji; tanpa input baru per step |
| C Policy | Belum seluruh cakupan: transfer keluarga graph lulus; lintas struktur/biaya belum terbukti |
| D Causal learning | Belum lulus penuh: precision membaik, keuntungan perilaku belum ada |
| E Performance | Belum lulus: readout/controller membaik, p95 whole-thought masih regresi sekitar 10% pada uji terakhir; target 3× belum tercapai |
| F Persistence | Lulus pengujian baru dan regression; tidak menyatakan semua pola crash/storage telah dicakup |
| G Integration | Belum lulus untuk production foundation; branch/PR penelitian tetap draft |

**Implemented:** readout relasional, eksplorasi opsional, kontrak observed pairs,
intervensi eksperimen eksplisit, kalibrasi mean opsional, discard, deep-copy
snapshot, buffer reuse dan telemetry reads.
**Tested:** lihat VALIDATION.md dan log autentik.
**Functionally demonstrated:** transfer antaralamat dan panjang, learned stop,
retensi setelah adaptasi panjang, pembatalan kredit pengganggu pada paired probe.
**Generalized:** terbatas pada keluarga topologi/momen yang diuji.
**Performance verified:** hanya workload yang benar-benar dilaporkan.
**Integrated:** PR60 berada di main; perubahan Phase II masih penelitian.
**Not yet proven:** causal behavioral advantage, learned representation umum,
retensi lintas konteks/reversal, sensitivitas biaya yang dipelajari, pengenalan
reward-role pada graph isomorfik, dan penggunaan bersama pipeline inquiry/sensor
produksi. Isomorfisme lokal dapat menyamarkan peran reward yang berbeda: encoder
empat momen tidak mempunyai informasi untuk membedakan semua kasus tersebut.

## Reproduksi

Gunakan Go1.22+; penelitian lokal memakai Go1.23.2. Contoh:

```sh
go run ./cmd/horizon-native-foundation -phase2 relational -epsilon .1 -seeds 2027,2039,2053,2063,2081 -train 2500 -test 256 -stream 9300000 -out evidence/confirm-relational
go run ./cmd/horizon-native-foundation -phase2 identity -epsilon .1 -seeds 2027,2039,2053,2063,2081 -train 2500 -test 256 -stream 9300000 -out evidence/confirm-identity
go run ./cmd/horizon-native-foundation -phase2 relational -seeds 2027,2039,2053,2063,2081 -train 2500 -test 256 -stream 9300000 -out evidence/confirm-softmax
go run ./cmd/horizon-native-foundation -credit paired -seeds 2027,2039,2053,2063,2081 -train 512 -test 256 -stream 9300000 -out evidence/confirm-credit-paired
go run ./cmd/horizon-native-foundation -credit unpaired -seeds 2027,2039,2053,2063,2081 -train 512 -test 256 -stream 9300000 -out evidence/confirm-credit-unpaired
python scripts/summarize_native_phase2.py evidence
```

Follow-up kalibrasi memakai seed 3041,3061,3083,3109,3121, stream 10700000,
`-credit unpaired|paired|paired-centered`, `-train 32|512`, dan `-test 512`.
Untuk setiap checkpoint, jalankan `-credit-boundary PATH -stream 10800000+SEED`
(jumlahkan angkanya sebelum memanggil CLI), dengan direktori output baru.
`scripts/summarize_native_phase2_followup.py evidence` mereproduksi ringkasan
kalibrasi dan retensi. Evaluasi retensi sebelum adaptasi menggunakan
`-retention-baseline PATH -seeds SEED -stream 9300000 -test 256`.

Recovery yang dipertahankan dapat dibuat ulang dengan
`-recovery-audit -phase2 relational -out DIREKTORI_BARU`. Pipeline tidak menimpa
direktori eksperimen lama. Biaya eksperimen konfirmatori: 37.500 episode utama,
15.360 episode adaptasi, 10.880 episode causal awal, serta 17.340 episode
kalibrasi baru; evaluasi greedy, pilot, probe parameter dan reproduksi dihitung
terpisah. Anggaran ini tidak mencakup pelatihan model besar.

Landasan, bukan klaim implementasi penuh makalah:
[Battaglia dkk.2018](https://arxiv.org/abs/1806.01261) untuk parameter sharing dan
inductive bias relasional;
[Varici dkk.2024](https://proceedings.mlr.press/v238/varici24a.html) untuk pentingnya
asumsi identifiability pada causal representation learning. Derivasi paired loss
di atas adalah kontrak regresi terbatas, bukan reproduksi algoritma makalah tersebut.
