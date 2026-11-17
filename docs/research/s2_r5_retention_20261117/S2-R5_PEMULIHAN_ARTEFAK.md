# Pemulihan artefak S2-R5

**Keputusan penelitian: PARTIAL — RETENTION LIMITED.** Main/Brain produksi tidak berubah. Seluruh 108 model adalah salinan penelitian dari R2; tidak ada arm terpilih sebagai penerus kanonik.

[Laporan lengkap](https://chatgpt.com/library?file_id=libfile_066e1d2b54548191bbed5596a206e401&account_id=personal) · [Grafik hasil](https://chatgpt.com/library?file_id=libfile_a12a1cbc503081919eb73aeafc2b0856&account_id=personal) · [Grafik feasibility](https://chatgpt.com/library?file_id=libfile_f43fe1cf5c48819183ace38ea4d0636e&account_id=personal)

## Arsip utama

Unduh **empat bagian** di bawah. Gabungkan dalam urutan angka 1–4. Ukuran gabungan **1,127,417,584 byte**; SHA-256:

`ee6cadaae0fa122beb40da1908c776262385f8c7e758d814e2bc4e411d9e2531`

**3,272/3,272 payload terautentikasi**, termasuk seluruh 108 checkpoint final, full historical+new WAL, source/checkpoint asli, ledgers sebelum operasi, poststates aktual, solver/rejection/parameter traces, seluruh kegagalan, per-seed/per-stream metrics dan source/reproduction scripts. Payload sebelum kompresi sekitar 17,08 GB. Empat bagian ini merupakan satu arsip utuh.

| Unduhan | Byte | SHA-256 |
|---|---:|---|
| [Bagian 1](https://chatgpt.com/library?file_id=libfile_a487ba6a53fc8191b93b82f5b74efd89&account_id=personal) | 314572800 | `c7df9929238877572c4af497a6f56e51e0de95fa044c21a98d908b4404001f00` |
| [Bagian 2](https://chatgpt.com/library?file_id=libfile_d3cd7ed9be4881919ab6ff448b3d8fab&account_id=personal) | 314572800 | `1f98d6268ed6fe222e4ad56583a4b6e4f852959792307c3daa512409f3a61329` |
| [Bagian 3](https://chatgpt.com/library?file_id=libfile_1dc21c953958819185de482f1d2ec5cd&account_id=personal) | 314572800 | `76e0cc407ffa408d7a2fb601ae74dda8cc29822bec9eaf5860cd7db83723ce93` |
| [Bagian 4](https://chatgpt.com/library?file_id=libfile_9dabeef224f48191bb83089a6d702649&account_id=personal) | 183699184 | `6faa9ee1cb8b13d3890ed59ae0984cbe7a0586889b79d2a4bc7c26be5d26faca` |

Keempat bagian telah diunduh ulang secara independen dan diperiksa byte/SHA-256. Gabungan unduhan mempunyai hash yang sama, dan manifest seluruh payload diperiksa lagi. Tidak ada WAL historis yang dipotong.

## Gabungkan dan verifikasi

Letakkan keempat file pada satu folder, kemudian:

```python
from pathlib import Path
import shutil, hashlib
archive = Path("Horizon_S2_R5_Retention_Constrained_Native_Learning_20261117.tar.xz")
with archive.open("wb") as out:
    for i in range(1, 5):
        with Path(str(archive) + f".part{i}").open("rb") as part:
            shutil.copyfileobj(part, out, 1024 * 1024)
h = hashlib.sha256()
with archive.open("rb") as f:
    while chunk := f.read(1024 * 1024):
        h.update(chunk)
assert h.hexdigest() == "ee6cadaae0fa122beb40da1908c776262385f8c7e758d814e2bc4e411d9e2531"
print("SHA-256 cocok")
```

Dari source R5 yang kompatibel, autentikasi semua payload tanpa mengekstrak 17 GB:

```sh
python3 scripts/package_s2r5_retention.py --verify /absolute/Horizon_S2_R5_Retention_Constrained_Native_Learning_20261117.tar.xz
```

Untuk ekstraksi lengkap gunakan folder baru. Untuk hanya mengambil Brain sumber dan source kode:

```sh
mkdir restored-r5
tar -xJf Horizon_S2_R5_Retention_Constrained_Native_Learning_20261117.tar.xz -C restored-r5 r5/00-source.json r5/source-events.jsonl scientific-source.tar.gz BUNDLE.json MANIFEST.json
```

Untuk arm penelitian tertentu, pilih `r5/training/seed-SEED/ARM/final.json` dan `events.jsonl` pada direktori yang sama. SEED=37/83/151/211/307/419. ARM=A/A0/B/B0/C/C0/Cw/Cw0/D/D0/E/E0/AI/AI0/DI/DI0/EI/EI0. `restarted.json` menyediakan snapshot exact hasil restart. Jangan memilih seed terbaik sebagai Brain produksi.

## Identitas dan native recovery

- Source code ilmiah: `2fac4b8e21cf89ae211aac3f8ee7f52473fda4e6`; tree `25f9ecbc4bcdcaf369a192b06572a26f64e384b7`.
- Basis kode R4: `0069e6eb27dbe9334746dd80cc31e5ed52214867`.
- Sumber Brain: **R2 sequence 80735**, checksum native `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.
- Original rollback: `r5/00-source.json` + `r5/source-events.jsonl`.
- Gunakan `OpenEventLog` dan `RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath)` pada salinan, dengan source/biner kompatibel yang disediakan. Memulai dari 80735 dan journal arm lengkap merekonstruksi state final arm itu persis.
- 108/108 restart, full suffix replay, signed reconstruction dan rollback exact; A–E1 lulus 108/108; strict S1 prediction shift 0.
- `scientific-source.tar.gz` memuat seluruh source kompatibel; `implementation.patch` memakai basis R4 di atas. `RESTORE.md`, `BUNDLE.json`, manifest dan semua konfigurasi menjelaskan reproduksi. Source R2/R4 asli juga ikut disertakan.
- Go 1.22.12 dipakai. Android arm64 hanya diuji cross-build; exact replay bukan klaim pengujian Android fisik.

D32 memperoleh manfaat history 21,455% (CI95 14,907–28,149%), tetapi retensi normal tambahan baru lulus 4/6 seed; E lulus 5/6. Kalibrasi masih gagal target. Karena itu hasil tetap PARTIAL dan tidak dipromosikan.

Berkas dengan nama berakhiran `_delivery.tar.xz.part1` sampai `.part12` adalah salinan alternatif encoding kompresi cepat yang memuat payload sama. Untuk pemulihan utama, gunakan **empat bagian di tabel ini**. Tidak perlu menggabungkan kedua encoding.
