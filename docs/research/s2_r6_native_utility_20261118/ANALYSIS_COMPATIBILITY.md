# Analisis receipt tanpa candidate

Source ilmiah dan raw data tetap sesuai996 hashes pada CONFIRMATORY_LOCK. Initial analysis gagal pada `len(null)` untuk receipt yang ditolak sebelum candidate generation. Ini terutama relevan pada conflicting observations; menghapus receipt tersebut akan menyembunyikan hasil negatif.

Companion `scripts/analyze_s2r6_results_compat.py` membaca `trials:null` sebagai list kosong **hanya dalam memori**, lalu menjalankan analyzer, seeds, thresholds, bootstrap dan keputusan yang sudah disegel. Opportunity tetap dihitung dan seluruh raw receipt/WAL asli tidak berubah. Frozen analyzer asli tidak diedit; traceback dipertahankan dalam execution/final-analysis.log. Hasil definitif dihasilkan dengan companion ini.

Untuk reproduksi statistik, salin companion dari folder reproduce/ ke scripts/ pada source ilmiah yang diekstrak, kemudian jalankan:

`python3 scripts/analyze_s2r6_results_compat.py --work /absolute/restored/r6 --phase final`

Perbaikan ini bukan fitting ulang, perubahan acceptance criteria, seed replacement atau final checkpoint selection. Seluruh48 native restart/replay/A–E1 proofs selesai sebelum perbaikan parser.
