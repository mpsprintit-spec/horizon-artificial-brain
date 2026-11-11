# Catatan eksperimen Horizon — komposisi dua pengetahuan neural terpisah

**Tanggal:** 11 November 2026  
**Status:** hasil eksperimen terverifikasi dalam lingkungan uji terkontrol; belum menjadi kemampuan otonom produksi  
**Repository:** `mpsprintit-spec/horizon-artificial-brain`  
**Branch penelitian:** `research/neural-factor-composition-continued-20261111`  
**Perlindungan integrasi:** tidak mengubah `main`, tidak ada PR atau merge.

## 1. Pertanyaan dan hipotesis

**Pertanyaan:** Dapatkah Brain Horizon menggabungkan representasi konteks dan struktur relasional yang dipelajari **secara terpisah**, lalu menghasilkan jawaban untuk pasangan konteks–struktur yang **tidak pernah dijadikan contoh latihan langsung**?

**Hipotesis yang diuji:** jika dua jalur faktor dan satu jalur penggabungan berbagi substrat neural kanonik, keluaran keadaan neural antara dapat dipakai sebagai masukan operasi berikutnya, sehingga kombinasi yang tidak dilatih langsung dapat ditangani.

Ini bukan pengujian pemahaman bahasa atau pengetahuan filsafat; pelajaran dan evaluasi memakai populasi neural numerik.

## 2. Substrat, data awal, dan isolasi

- Melanjutkan **checkpoint asli hasil eksperimen sebelumnya** dari workflow [43037032979](https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43037032979), bukan membuat brain kosong.
- Keadaan awal terverifikasi: **488 pola neural**. SHA-256 berkas checkpoint induk: `a32cda6cc0cc9a010aec1d5f224513d7188bee8ec639f0f68fcf507c98722a84`.
- Semua pelajaran menggunakan `BrainRuntime`, `LearnExperience`, dan `PatternIndex` dari Horizon yang sama. Populasi neural baru direkrut ke Brain yang dipulihkan, bukan model terpisah.
- Tidak mengubah jalur inferensi produksi. Pengurutan operasi dan pembacaan populasi dilakukan di **penguji internal khusus eksperimen**.
- Hanya perubahan berkas penelitian/workflow pada branch yang tercantum di atas; branch `main` tidak digunakan sebagai target penulisan.

## 3. Protokol pelatihan

Jumlah tambahan: **184 pengalaman**, 8 putaran. Setelah pelatihan, Brain berisi **580 pola neural** (bertambah 92).

Tiga jenis pelajaran **tidak berisi pasangan mentah konteks × struktur relasional**:

1. **Faktor konteks:** sebuah pengamatan numerik dan konteks dipetakan ke satu dari dua populasi keadaan internal yang dipelajari.
2. **Faktor relasional:** tiga frame numerik berurutan dan satu sinyal bukti dipetakan ke satu dari tiga populasi relasional internal yang dipelajari.
3. **Komposisi:** kombinasi dua keadaan internal (2 × 3) dipetakan ke satu dari tiga populasi hasil. Keluaran akhir tidak diberikan saat inferensi.

Penggabungan dilakukan berurutan: baca faktor konteks → baca faktor relasional → berikan kedua hasil neural sebagai input ke pola komposisi. **Pemilihan urutan ini masih dibuat oleh penguji**, belum merupakan general neural policy yang dipilih Horizon secara otonom.

Pembacaan keluaran awalnya gagal (0/33) akibat *shared projection anchor*: beberapa populasi faktor berbagi neuron jangkar umum sehingga skor maksimum seri. Evaluator kemudian diperbaiki untuk hanya menghitung neuron yang **diskriminatif** antar-populasi. Perbaikan ini tidak mengubah brain atau inferensi produksi. Hasil tervalidasi di bawah menggunakan evaluator yang telah diperbaiki.

## 4. Hasil pengujian yang dapat ditelusuri

| Kategori uji | Sebelum latihan komposisi | Sesudah | Keterangan |
|---|---:|---:|---|
| Struktur terbalik dengan konteks yang belum pernah dipasangkan | 0/6 | **6/6** | Pasangan mentah tidak disupervisi |
| Struktur alternatif dengan konteks yang faktor-faktornya dipelajari terpisah | 0/3 | **3/3** | Transfer komposisional terkendali |
| Struktur relasional baru yang faktor strukturnya telah dilatih terpisah × berbagai konteks | 0/15 | **15/15** | Struktur baru bagi pasangan, **bukan** belum pernah dipelajari sebagai faktor |
| Kontrol struktur sama sekali belum dipelajari | 0/3 | 0/3 | Tidak ada keluaran yang memenuhi kriteria |
| Kontrol konteks sama sekali belum dipelajari | 0/3 | 0/3 | Tidak ada keluaran yang memenuhi kriteria |
| Kontrol konteks dan struktur sama-sama belum dipelajari | 0/3 | 0/3 | Tidak ada keluaran yang memenuhi kriteria |
| **Total 33 kasus** | **0/33** | **24/33** | 24/24 kombinasi faktor yang telah diajarkan secara terpisah; 0/9 kontrol OOD |

Keberhasilan dinilai berdasarkan identitas populasi keluaran dan margin dukungan learned-pattern dalam evaluator. **24/24 bukan ukuran kemampuan penalaran umum**, dan sembilan kontrol tanpa keluaran tidak dengan sendirinya membuktikan penangguhan keputusan yang terkalibrasi secara epistemik.

**Retensi pengetahuan sebelumnya:** margin relasional awal `[+1.361564, -1.361564]`, sesudah `[+1.369935, -1.368625]`. Jalur lama tidak hilang pada probe retensi yang digunakan.

**Kontrol kausal dalam laboratorium:**
- Pembalikan target pengajaran untuk faktor konteks mengubah hasil prediksi pada kasus yang bergantung pada konteks.
- Ablasi jalur learned-pattern menghilangkan hasil yang didukung evaluator.
- Kedua kontrol menunjukkan ketergantungan terhadap pembelajaran yang dilakukan, tetapi tidak menggantikan replikasi independen atau ablation lengkap atas setiap mekanisme neural.

## 5. Sumber verifikasi dan reproduksi

- **Uji implementasi:** [`services/ai/runtime/neural_factor_composition_continuation_test.go`](../../services/ai/runtime/neural_factor_composition_continuation_test.go)
- **Workflow:** [`.github/workflows/neural-factor-composition-continued.yml`](../../.github/workflows/neural-factor-composition-continued.yml)
- **Run GitHub Actions yang berhasil:** [43040477001](https://github.com/mpsprintit-spec/horizon-artificial-brain/actions/runs/43040477001), job `130590601634`, status **completed/success**.
- **Artifact terlatih:** `horizon-learned-factor-composition-brain`, artifact ID `13864987050`, berisi `horizon-brain-factor-composition.json` dan `factor-composition-metrics.json`. Artifact GitHub Actions memiliki masa retensi dan dijadwalkan kedaluwarsa **11 Desember 2026**; tautan run/log tidak menjamin artifact masih dapat diunduh setelah masa itu.
- **SHA commit source eksperimen yang teruji:** `5c181cbc56ec0191ce8761a1a60319de0736b33f`.

## 6. Interpretasi dan batas klaim

**Terbukti dalam batas eksperimen:** ketika dua faktor numerik dan satu operator neural telah dilatih secara terpisah, Horizon mampu menggunakan jalur learned-pattern pada Brain yang sama untuk menangani 24 kombinasi konteks–struktur yang tidak diberikan sebagai pasangan mentah saat pelatihan. Checkpoint hasil lanjutannya dapat dipulihkan dan tersimpan sebagai artifact.

**Belum terbukti:**
- pemilihan dan pengurutan operasi neural secara mandiri tanpa evaluator yang mengendalikan jalurnya;
- penemuan sendiri faktor abstrak dari input mentah tanpa pemisahan kanal dan supervisi;
- generalisasi pada konteks **atau** struktur yang belum pernah dipelajari sedikit pun;
- transfer pada domain berbeda, ketahanan terhadap gangguan dunia nyata, kalibrasi probabilitas, serta penalaran filosofis umum.

**Keputusan eksperimen:** *lulus sebagai demonstrasi komposisi neural terarah dan transfer antarpasangan pada faktor yang telah dipelajari*, **bukan** bukti berpikir umum/otonom.

**Target eksperimen berikutnya:** latih kebijakan internal untuk memilih urutan operasi, gunakan pembelajaran kausal atas manfaat komposisi, dan evaluasi kasus pasangan baru lintas domain tanpa penguji memilihkan jalur pemrosesan.

---
Catatan ini dibuat sebagai **rekaman penelitian**; bukan izin untuk menggabungkan kode/hasil ke `main`.
