# S2-R6 — Pemulihan artefak penelitian

Tanggal: 18 November 2026, Asia/Jakarta. **PARTIAL — UTILITY PREREQUISITE NOT MET.**

[Laporan lengkap](https://chatgpt.com/library?file_id=libfile_ae675cd298f48191bd8939433afa07a8&account_id=personal), [grafik hasil](https://chatgpt.com/library?file_id=libfile_d6b206e9939881918ec623103081caa6&account_id=personal), [grafik retensi](https://chatgpt.com/library?file_id=libfile_7cd2b799d2208191bde0642f8c02de17&account_id=personal), dan [original Git bundle](https://chatgpt.com/library?file_id=libfile_3fc3a5d95f308191887484b0cb976c9d&account_id=personal).

Utility belum mengalahkan pembanding sederhana pada holdout independen. Interference learning, adaptive plasticity dan consolidation tidak dijalankan karena prasyarat utility gagal pada development. Enam seed, 48 model final, seluruhnya lulus exact restart/replay/rollback dan oracle A–E1. Retensi distribusi lama baru, target manfaat history dan kalibrasi belum lulus. Tidak ada arm yang menjadi Brain kanonik. Main dan Brain produksi tidak diubah; fixed 32 numerical features.

## Unduh seluruh bagian

Enam bagian berikut membentuk satu arsip lengkap. Lima bagian pertama masing-masing 314572800 byte; bagian keenam 17541149 byte. Nama dan urutan wajib dipertahankan.

| Bagian | Bytes | SHA-256 |
|---|---:|---|
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part1](https://chatgpt.com/library?file_id=libfile_469545a608b881919ff1b84738e87144&account_id=personal) | 314572800 | `615dfc7ea36763445b119d854fa14e67438e9f1c60196ff08e9741cc33f030eb` |
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part2](https://chatgpt.com/library?file_id=libfile_bae1660b9eb08191a6157f2e19ce643d&account_id=personal) | 314572800 | `903d2fab798e6a93c83bf0c50415039a12a8c320131c8a8f8ad6df2dbb80e3f6` |
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part3](https://chatgpt.com/library?file_id=libfile_05be0cf8cdfc81919d75d7c520f82495&account_id=personal) | 314572800 | `d63a9dbf9533e0f1ff610999ecc1f2b476b62a9288f34c2f7459a4baba557dc1` |
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part4](https://chatgpt.com/library?file_id=libfile_f4f58e7a3ccc81918c98504edc23769d&account_id=personal) | 314572800 | `2ef749ce2d747527b39e764ca822635676739ffe3e98d952cd931677bf92f42c` |
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part5](https://chatgpt.com/library?file_id=libfile_a6894b589a1c8191ae1622874bb0e487&account_id=personal) | 314572800 | `9638e0caa15646b6be54b87dcebc3d181a19c4b37401aec1f6c6262f41a5b00e` |
| [Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part6](https://chatgpt.com/library?file_id=libfile_e8359057768c8191897f96c959bfd1be&account_id=personal) | 17541149 | `748425fe386baa265395b80f9d548e219d5eb57ce21dcebceffbc71a66716af3` |

**Gabungan:** `Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst`.

- Ukuran: **1590405149 byte**.
- SHA-256: `059fa4665ef3f1b5822801c212f27b9494d9b806d0ccb0892c62465cff89cf0c`.
- Payload terautentikasi: **1914/1914**, termasuk manifest sumber historis.
- Total payload sebelum kompresi: **14295582771 byte**.
- Semua enam bagian diunduh kembali secara independen; SHA-256 setiap bagian dan gabungan sama persis dengan arsip asli.

## Gabungkan dan periksa

Jalankan Python di direktori berisi keenam bagian. Kode berikut membuat berkas baru; jangan menggunakan nama target yang sudah ada.

```python
import hashlib
import shutil
from pathlib import Path

parts = [
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part1',
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part2',
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part3',
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part4',
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part5',
    'Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst.part6',
]
target = Path('Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst')
with target.open('xb') as out:
    for name in parts:
        with open(name, 'rb') as src:
            shutil.copyfileobj(src, out)
h = hashlib.sha256()
with target.open('rb') as src:
    for block in iter(lambda: src.read(8 * 1024 * 1024), b''):
        h.update(block)
assert target.stat().st_size == 1590405149
assert h.hexdigest() == '059fa4665ef3f1b5822801c212f27b9494d9b806d0ccb0892c62465cff89cf0c'
print('Arsip utuh:', h.hexdigest())
```

Ekstrak ke direktori baru, dengan `tar` dan `zstd` tersedia:

```bash
mkdir restored-s2r6
tar --zstd -xf Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst -C restored-s2r6
python3 restored-s2r6/reproduce/package_s2r6_fast.py --verify Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst
```

Ikuti `RESTORE.md` di hasil ekstraksi. Arsip mempertahankan source R5 lengkap, source R2 lengkap, checkpoint sumber/intermediate/final, full WAL, seluruh pengalaman/prediksi/label aktual, receipts, candidates rejected, causal interventions, kontrol negatif, locks, biaya performa, hasil per-seed, log kegagalan dan skrip reproduksi. Kompresi Zstandard bersifat lossless; tidak ada pengurangan evidence.

## Pulihkan satu seed tanpa mengubah aslinya

Ekstrak `scientific-source.tar.gz` ke checkout baru. Gunakan Go 1.22.12 dari source tersebut:

```bash
go test -tags 'horizon_s2r2 horizon_s2r3 horizon_s2r4 horizon_s2r5 horizon_s2r6' -c ./services/ai/runtime -o /absolute/s2r6-owner.test
python3 scripts/recover_s2r6_utility.py --work /absolute/restored-s2r6/r6 --source /absolute/restored-s2r6/source/R2-active --binary /absolute/s2r6-owner.test --destination /absolute/recovered-seed-37 --seed 37
```

Ulangi seed `83, 151, 211, 307, 419` dengan tujuan berbeda. Pemulihan meliputi delapan arm per seed: A, A0, B, B0, BI, BI0, C dan F. Historical WAL tidak dipotong. Cross-build Android arm64 tersedia sebagai bukti kompatibilitas build; exact replay lintas arsitektur dan benchmark perangkat fisik tidak diklaim.

## Identitas sumber dan publikasi

Brain sumber: S2-R2 sequence `80735`, native checksum `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`, 274 units, 6408 synapses, 32 features, 10752 native policy updates. Tidak ada checkpoint terbaik R3/R4/R5 yang dipilih.

Main baseline `11c6e0a78547d68a0be8f78b3c801e912722608d`; basis kode R5 `099ae285737cdc62fc121791ba4e6f4e463a1cb3`. Scientific source dibekukan di local commit `c068d827b1b8c2cd6edb701b8e9b16dc747ea120`, tree `6f09340835532cda247495d0550a7a6af55d48ac`. Published scientific commit `b82b07f530bff9c72fc48b0b4be7b74f6d6a84a8` mempunyai tree identik. Metadata commit berbeda; original commits dipertahankan di Git bundle.

Untuk mengimpor original commits dari clone yang sudah memuat R5:

```bash
git fetch /absolute/Horizon_S2_R6_Scientific_Commits_20261118.bundle HEAD:refs/heads/recovered-s2-r6-original
```

Branch penelitian: `research/s2-r6-native-utility-retention-20261118`. Seluruh hasil negatif dipertahankan. Arsip ini adalah artefak penelitian, bukan pengganti Brain produksi.
