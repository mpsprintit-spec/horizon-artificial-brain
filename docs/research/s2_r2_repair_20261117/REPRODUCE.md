# Reproduksi S2-R2

Kebutuhan: Go 1.22+ (diuji 1.22.12 linux/amd64), Python 3.10+, repository branch
S2-R2, dan source asli terverifikasi. Harness memakai standard library; tidak
memerlukan API model, regressor eksternal, kamera, GPU atau Brain kosong.

## Sumber

Arsip S2 asli memiliki SHA-256
`661d007850d3026ed22636fdc4d96e9113115fb2b9e2d66856ad88e4ba60a49f`.
Gunakan hasil ekstraksinya, atau direktori `source/` dari bundle R2 lengkap.
Runner memverifikasi 75 payload. Semua output harus terpisah dari source.

## Akuisisi verifikasi dari original sequence 76617

```sh
python3 scripts/run_s2r2_repair.py \
  --source /absolute/original-s2 \
  --work /absolute/new-r2-verification \
  --go /absolute/go/bin/go
```

Runner mengompilasi tag `horizon_s2r2`, menjalankan prepare, mechanism, delapan
blok train, retry engineering, retention/freeze, satu final holdout, persistence
dan continuation. Source dan produksi tidak ditulis. Work harus baru. Hasil
ilmiah PARTIAL tetap dapat keluar dengan test exit 0 bila evidence valid.
Tidak ada retry otomatis untuk mengubah hasil negatif menjadi PASS.

Reproduksi baru mengumpulkan development native pada seed yang sama. Pada run
yang dipublikasikan, setelah kegagalan retensi engineering awal, development
dipakai ulang menggunakan `HORIZON_S2R2_DEVELOPMENT_REUSE` yang menunjuk run awal:
512 observasi aktual, ledger pra-operasi dan WAL observer yang sama. Tidak ada
parameter hasil pelatihan run awal yang disalin ke lineage baru. Lihat amendment
dan failed-engineering-revision dalam bundle; jangan menghilangkan hasil gagal.

Untuk satu fase gunakan `--phase`; eksekusi fase bergantung pada output fase
sebelumnya. Jangan mengulang `final` pada run yang sama: marker FINAL_STARTED
dibuat eksklusif sebelum pengambilan kasus. Verifikasi restart terhadap input
holdout tersimpan bukan pengambilan kasus holdout baru atau pemilihan model.

## Audit checkpoint/journal yang sudah ada

```sh
HORIZON_S2R2_SOURCE=/absolute/original-s2 \
HORIZON_S2R2_WORK=/absolute/restored/s2r2 \
/absolute/go/bin/go test -tags horizon_s2r2 ./services/ai/runtime \
  -run '^TestS2R2SavedReadoutAndS1Attribution$' -count=1 -v -timeout 10m
```

Audit membaca final dengan journal final, membandingkan 13 intervensi,
memeriksa S1 restart/continuation dan ablasi pada salinan disposable. Ia tidak
mengubah checkpoint atau melakukan pembelajaran. JournalReproduction mengharapkan
journal yang berakhir tepat pada s2r2-final; untuk journal terbaru gunakan native
recovery dari pasangan continuation sebagaimana RESTORE.md.

Uji generik tanpa source khusus:

```sh
/absolute/go/bin/go test -race ./services/ai/knowledge ./services/ai/runtime \
  -run 'Test(NMetric|MetricBounded|Metric|NativeNumeric)' -count=1 -timeout 20m
/absolute/go/bin/go vet ./...
/absolute/go/bin/go build ./...
```

`legacy_equivalence_test.go.txt` dapat disalin sebagai file test sementara ke
checkout S2-R1 asli untuk mencocokkan legacy receipts/state dengan kontrol R2.
Ia tidak mengubah learner lama. Source-backed test yang skip tanpa environment
tidak dihitung sebagai eksperimen yang dieksekusi.

## Analisis tanpa fitting

```sh
python3 scripts/analyze_s2r2_repair.py \
  --work /absolute/restored/s2r2 \
  --failed /absolute/restored/failed-engineering-revision \
  --output /absolute/new-analysis
```

Analisis mengaudit witness/prefix, signed paths, receipt dan metrik tersimpan;
tidak mengubah model, threshold, checkpoint atau holdout. Semua payload bundle
terautentikasi oleh MANIFEST.json. Lihat ARTIFACT.md untuk byte count dan SHA-256.
