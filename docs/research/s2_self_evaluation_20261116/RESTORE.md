# Pemulihan S2 tanpa pelatihan ulang

Paket lengkap ditautkan melalui `ARTIFACT.md` pada PR penelitian. Unduh byte
asli, cocokkan ukuran dan SHA-256 arsip yang dipublikasikan, lalu verifikasi
semua payload sebelum ekstraksi:

```sh
python3 scripts/package_s2_self_evaluation.py --verify PATH/Horizon_S2_Native_Self_Evaluation_20261116.tar.xz
mkdir NEW_EXTRACTED_DIRECTORY
tar -xJf PATH/Horizon_S2_Native_Self_Evaluation_20261116.tar.xz -C NEW_EXTRACTED_DIRECTORY
```

Gunakan checkout PR penelitian (baseline main `11c6e0a` ditambah file S2), Go
1.22.12 atau kompatibel. Verifier memakai jalur produksi/native schema20 dan
tidak menjalankan acquisition, Evaluate, atau pelajaran S1/S2:

```sh
go test -c ./services/ai/runtime -o /tmp/horizon-s2-verify.test
cd services/ai/runtime
HORIZON_S2_SOURCE=ABSOLUTE/NEW_EXTRACTED_DIRECTORY/source \
HORIZON_S2_WORK=ABSOLUTE/NEW_EXTRACTED_DIRECTORY/s2 \
/tmp/horizon-s2-verify.test -test.run '^TestS2Verify$' -test.v -test.timeout 8m
```

Bagian `source/` berisi byte continuation/WAL asli S1 dan populasi sumber yang
dibutuhkan verifier. `s2/lineage/s2-continuation.json` bersama
`s2/lineage/events.jsonl` adalah pasangan state penelitian yang dilanjutkan.
Sequence yang diharapkan **76617**, native checksum
`b87285e38a321d45e4da920f219b388d1a2d061554042d2dd0f109883b38ab9d`.
Counter native policy **10752**; numeric lesson S2 **16 commands / 4 commits**.

Untuk rollback penelitian, gunakan `s2/rollback-observed.json` dengan
`s2/rollback-observed.json.events.jsonl`. Memasangkan checkpoint ini dengan
full `s2/lineage/events.jsonl` akan mereplay suffix S2 yang tersimpan, bukan
rollback. Pasangan checkpoint/journal dan lineage harus tetap cocok.

Jangan menjalankan `Prepare`, `Train` atau `Evaluate` pada artefak yang sudah
selesai. Jangan mengganti checkpoint produksi. `run_s2_self_evaluation.py`
memiliki mode `--verify-only` untuk workflow yang juga memegang arsip S1 asli;
verifier langsung di atas cukup untuk bundle S2 yang berdiri sendiri.

Arsip S1 asal tetap tersedia melalui
[Horizon_S1_Native_Dynamics_20261116.tar.xz](https://chatgpt.com/library?file_id=libfile_8028c0b7b70c8191a35dbb9168ef040c&account_id=personal),
SHA-256 `7ca53ea11b2a56a149f466d65a1d31733581d893ea810b3003ba3a3895fb12e7`.
Hasil ilmiah S1 dan S2 tetap PARTIAL. Recovery yang sukses tidak menaikkan
kemampuan menjadi PASS.
