# Codec pengukuran dan readout S2

Ini adalah format adapter penelitian, bukan controller atau fungsi jawaban.
Implementasi rujukan `s2Input`/`s2Predict` berada di
`services/ai/runtime/s2_self_evaluation_test.go`. Pemanggilan produksi/native
yang mendasari pembacaan tetap `BrainRuntime.ReadComposition`.

## Target yang dipelajari

Untuk setiap operasi, S1 memprediksi tiga nilai. Nilai aktual masing-masing
adalah rata-rata aktivitas empat neuron populasi5,6,7 pada `populations.json`,
dibagi dua. Sesudah operasi selesai, hitung residual aktual minus forecast,
kemudian mean absolute error ketiga residual. Itulah label kontinu S2.
Tidak ada rumus error aktual pada forward readout S2.

Reliability di sini berarti perkiraan besarnya error, dalam satuan proyeksi S1.
Nilai lebih kecil dimaksudkan sebagai prediksi lebih akurat. Tidak ada
probabilitas benar/salah, interval coverage, skor keyakinan terkalibrasi,
ambang trading atau tindakan yang diturunkan dari angka ini. Dua keluaran
negatif mentah pada holdout memperlihatkan keterbatasan readout magnitude;
keluaran tersebut dipertahankan dalam metrik dan data.

## Input sebelum operasi

`CompositionInput.Words` berisi satu vektor16: indeks3 = 1, indeks15 = 0,125,
sisanya nol. Ini query numerik error, berbeda dari query0,1,2 milik S1.
`RequiredPrecision` = 0,002. Scene adalah envelope inert yang sama dengan S1;
scene tidak menyimpan hasil target atau label reliabilitas.

`Sequence` mempunyai 10 baris, setiap baris enam nilai dan gate ketujuh = 1:

| Baris | Enam nilai |
| --- | --- |
| 0 | Ada history (0/1); operasi trial sebelumnya (-1 propagate, +1 stop, 0 jika tiada); empat nol. |
| 1 | Tiga forecast S1 saat ini; tiga absolute error dari trial sebelumnya. |
| 2–9 | Delapan baris codec S1 yang tidak diubah. |

Bagian S1 terdiri dari 36 aktivitas neuron pada sembilan populasi dibagi dua,
sembilan rata-rata aktivitas dari keadaan sebelumnya (nol jika tidak ada),
operasi saat ini (-1/+1), operasi internal sebelumnya (-1/+1 atau nol), dan
indikator keberadaan konteks internal sebelumnya. Total48 nilai disusun enam
nilai per baris. Populasi dan urutan neuron mengikuti `populations.json`.

Input numerik native mengindeks sequence dari lag terakhir ke pertama. Karena
baris baru **diletakkan di depan**, 56 koordinat numerik lama S1 tetap pada
posisi yang sama. Forecast tambahan berada di koordinat56–58, error history
59–61, dan indikator/operasinya63–64. ID trial, seed, after-state dan error
trial saat ini tidak masuk input forward.

`ReadComposition(...).NResolution.Value` dibaca langsung sebagai estimasi
magnitude, tanpa clipping atau normalisasi berdasarkan target yang belum
terjadi. Fitur, arah, resolusi, signed readout dan bias diambil dari sinaps
kanonik checkpoint. Menghapus witnesses tidak mengubah pembacaan.

## Apa yang benar-benar diperoleh

Pada kandidat S2 ini tidak terdapat source aktif di koordinat56 ke atas.
Keempat fitur baru merupakan produk fitur lama. Dengan demikian kanal forecast
tambahan dan history ditawarkan oleh codec tetapi belum direkrut oleh learner.
Penghapusan history memberi delta RMSE tepat nol. Hubungan kondisi neural
lama dengan magnitude error sudah dipelajari sebagian; penggunaan history
dan pembaruan reliabilitas setelah blok pengalaman baru belum terbukti.

Keterbatasan ini melekat pada hasil PARTIAL yang disimpan dan tidak ditutupi
dengan tabel operasi, regressor tambahan, atau pengendali kepercayaan.
