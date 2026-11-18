# S2-R7 — pemulihan artefak penelitian

**Verdict: PARTIAL — H1 utility belum lulus.** H2/H3 tidak dibuka; tidak ada penggantian Brain produksi atau perubahan main.

[Laporan lengkap](https://chatgpt.com/library?file_id=libfile_04768c38543481919d9aa59883e0e016&account_id=personal) · [Hasil utility](https://chatgpt.com/library?file_id=libfile_d8209036fd448191983d87fff440b768&account_id=personal) · [Learning curves](https://chatgpt.com/library?file_id=libfile_e528796820f8819188fae67504cb559d&account_id=personal)

Arsip lengkap: **3,577,492,032 bytes**, 12 bagian; **2,424 payload**, 10,254,275,266 bytes setelah dekompresi.

SHA-256 gabungan:

`67cc764a946d1e664ccb53593b2d3dbacc35375da83f1a6613539f4847e13cc1`

Seluruh bagian diunduh ulang secara independen setelah disimpan. Checksum masing-masing bagian, ukuran/gabungan SHA-256 dan seluruh payload manifest cocok. Pemeriksaan ini membaca ulang hasil unduhan, bukan hanya file asal.

## Bagian arsip

Unduh semua bagian berikut, lalu gabungkan **01→12** sebagai byte biner. Jangan ekstrak setiap bagian sendiri.

| Bagian | Bytes | SHA-256 |
|---|---:|---|
| [01](https://chatgpt.com/library?file_id=libfile_319dd8f14d88819186dc56ac72fe7209&account_id=personal) | 314572800 | `825517522f368762e1618797bb17630c0e6ab6d4210aad667bc7352a0d0f007c` |
| [02](https://chatgpt.com/library?file_id=libfile_c870ddc3a9648191b73b01d209f9d904&account_id=personal) | 314572800 | `141e004a1f8afd65788887f60899a5d278d860468107fc4d0a0f905a3fab08ac` |
| [03](https://chatgpt.com/library?file_id=libfile_d8ded0862a148191887cf5728eb9d83c&account_id=personal) | 314572800 | `57b477cdbeeed99af11bf20d565a68c954080f95b686205c70fb74bb6adc4a81` |
| [04](https://chatgpt.com/library?file_id=libfile_45bcda556b8c8191a8e9bf810635484d&account_id=personal) | 314572800 | `e32b386a9178e6bac6b020b1a7637fe90c99a3a17e1e91ef446e7b4f8791a6d3` |
| [05](https://chatgpt.com/library?file_id=libfile_f00c8637c2f481919d08f4d1f5d97967&account_id=personal) | 314572800 | `6c1462354c06a44938d4059e019ee48023a3f39320c9acea135327131fb1829c` |
| [06](https://chatgpt.com/library?file_id=libfile_49dc02aa11a481919abed37a32bbdf57&account_id=personal) | 314572800 | `8b643f9ee6d71bfae7232554797ed47f63a9418d6f5f94c08c73b42bf75134d0` |
| [07](https://chatgpt.com/library?file_id=libfile_ac4d2d4034a481919b5092b46312ca47&account_id=personal) | 314572800 | `150332bdf03a52c53f3827e2db1594174f90b630025d340d2aff8666923c7c19` |
| [08](https://chatgpt.com/library?file_id=libfile_391e660dde2081919dded041d0668070&account_id=personal) | 314572800 | `ca993c572ea78d435b73ae77572dda3ad699e1dc21db2b70b6b1e037f06a2b00` |
| [09](https://chatgpt.com/library?file_id=libfile_5689dbaec4388191a260e69e706e7bd4&account_id=personal) | 314572800 | `968bd9f905bb55761f091bc00d2f6a7188fb8fcee37edb8be88958bcc68570c3` |
| [10](https://chatgpt.com/library?file_id=libfile_d7d11e7d85848191800da54724830a3a&account_id=personal) | 314572800 | `30d23150df506d986014d40041b8a73e04f9053609d7eac0c276395478ba3e66` |
| [11](https://chatgpt.com/library?file_id=libfile_69f12e103e4c8191aec714adb7e26c01&account_id=personal) | 314572800 | `b0019135fc68fc61b9f691ea626d786439cb97125400c2c7bd2b46fc248770ea` |
| [12](https://chatgpt.com/library?file_id=libfile_d2ceb285f4a881919d59f0549f81ec95&account_id=personal) | 117191232 | `d83bda2724ac446570135350186ff0cdf549e95ea2e0defa03fafdf006bc927c` |

Nama gabungan: `Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst`.

```bash
cat Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst.part[0-9][0-9] > Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst
sha256sum Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst
mkdir horizon-r7-restored
zstd -dc Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst | tar -xf - -C horizon-r7-restored
```

Periksa SHA-256 sebelum ekstraksi, lalu ikuti `RESTORE.md` di dalam arsip. Verifier `scientific/package_s2r7_complete.py` dapat memeriksa seluruh manifest langsung dari bagian arsip tanpa ekstraksi.

## Isi dan sumber

Arsip memuat R6 lengkap beserta source R2, full WAL, source/intermediate/final checkpoints, seluruh forecast sebelum operasi, actual outcomes, intervention records, alias analyses, Fit/Check receipts, rejected candidates, per-seed metrics, learning curves, negative controls, recovery scripts, binaries dan source code. Payload besar yang dikompresi secara individual memiliki pemetaan checksum byte asli dan dapat dipulihkan persis.

[Full frozen source tar](https://chatgpt.com/library?file_id=libfile_3ae4abd225148191993113e948342ede&account_id=personal) · [Git scientific commits bundle](https://chatgpt.com/library?file_id=libfile_c31499e60960819187433030215b917f&account_id=personal)

R7 frozen local commit `5e83ebfe85208ff1d2b82e37a137cdbad10369cc`; published scientific commit `4b15d7b11c58633b5f7716aa8b1c23ae8fb919b7`; exact tree `7b3cb56f239475dc9d5eab7d753775c2f52e1fd3`. Bundle mempertahankan commit lokal asli. Keduanya memakai code parent R6 `a902b06fc87652858049f12b7e05cbf471fb74d2`.

Brain parent tetap sequence80735, native checksum `9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`, bukan checkpoint R6 yang dipilih. Seluruh arm melanjutkan Brain fase sebelumnya, 32 numerical features, satu substrate N per arm.

## Hasil yang harus dipertahankan saat recovery

- H1 gagal pada seen/unseen/shifted; seluruh enam seed termasuk211 dipertahankan. Tidak ada pembelajaran interference/adaptive plasticity yang diklaim.
- Exact restart, full suffix replay, signed reconstruction, historical WAL prefix dan rollback:60/60.
- A–E1 dengan control revision/time prosedur R6:60/60. Perbandingan raw source yang tidak menyamakan revision:0/60; physical-time-only:0/30; seluruh hasil awal itu disimpan sebagai diagnosis confound pengukuran.
- Historical S2 dan strict S1:6/6; paired new-normal retention:5/6; history gain/calibration gagal. Overall integration belum PASS.

Branch penelitian: [research/s2-r7-native-utility-information-repair-20261118](https://github.com/mpsprintit-spec/horizon-artificial-brain/tree/research/s2-r7-native-utility-information-repair-20261118). Main `11c6e0a78547d68a0be8f78b3c801e912722608d` tetap; PR80 tidak di-merge.
