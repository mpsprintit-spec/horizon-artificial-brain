# Continuation record

## Source, branch, keputusan

- Main autentik tetap `3a9e7ce3ffadff86c792050056ebea4eda48da0b`.
- Kandidat main: PR63, branch `fix/bootstrap-idempotent-startup-android`, HEAD
  `cee28ff2431b2a5f5d2ea61c668c4de262c25426`, CI43597512222 SUCCESS, READY.
- Penelitian: PR69, branch `research/canonical-integration-continual-20261114`.
  Source freeze published `a5d2842f8cc4fa835d32c87bb2008244b0c51768`, tree
  `b5943abb9effa1d863b225d2cb33231e69e555fb`; source CI43597835457 SUCCESS.
- Local source commit yang dieksekusi0687321241a392c4c1049bf909fc40ac5153c294
  mempunyai tree identik. Publication sesudah freeze hanya dokumen/evidence.
- Tidak ada merge ke main, force push, controller default atau Brain produksi
  yang diganti. PR63 digunakan kembali, bukan diduplikasi.

## Brain yang wajib dilanjutkan

Archive di repository:
`docs/research/canonical_integration_20261114/evidence/Horizon_Canonical_Continuation_E1_20261114.tar.xz`.

SHA256 `7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3`.
203 payload, seluruh hash diperiksa terhadap MANIFEST.json setelah compression.

Setelah ekstraksi, gunakan pasangan:

- `e1-main-continuation/07-E1-restarted.json`
- `e1-main-continuation/events.jsonl`

Identity `horizon-primary-brain`, sequence75331, native updates10752,
57 neuron,94 sinaps,24 PatternSynapse. Checkpoint checksum
`68a1518ebf1eea887041a339851754880d7638866a135361693205bab4bf4b82`.
Checkpoint file SHA256
`e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537`.
Journal SHA256
`ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede`.
Journal-prefix hash
`9cb342a9419e187a8a333a75018debd8fde39d7a40c150f92fb72e3cf23039a4`.

Restore dengan API existing RebuildRuntimeFromCheckpointAndJournal pada owner
yang detached, lalu attach writer setelah verifikasi. Jangan menjalankan E1
lagi dari10817 sebagai pelatihan utama. Parent PR68, checkpoint setiap blok,
serta full journal tetap dipertahankan. Jangan campur dengan checkpoint V6,
production memory, atau neuron hasil overlay penelitian lain.

## Pekerjaan yang sudah selesai

- Source/integration audit: MAIN_READINESS.md; learning inventory dan compatibility
  consolidation: PROVEN_LEARNING_INVENTORY.md.
- Main63: koreksi bootstrap + regresi repeat-resume dengan snapshot path tidak
  dapat ditulis; full tests/race/vet/build/Android cross-build/CI lulus.
- E1: 3 blok/9216 pasangan; holdout18/36->35/36, regret turun95.65%, nuisance
  sensitivity turun89.21%; policy-synapse ablation kembali18/36.
- Retensi A/B/D semua margin>=.1; C6/6, termasuk setelah restart. Tidak ada refresh
  A–D atau model reset.72 edge policy existing diperbarui, role/config sama.
- Complete genesis/suffix/final-checkpoint recovery dan next recurrent transition
  identik; exact outcome retry idempotent, reward berubah ditolak.
- Mixed admission/ownership/legacy isolation,30 subprocess crash cases dan
  physical replay safety lulus;0 dispatch baru saat replay.
- V6 bundle diverifikasi sebagai learned lineage terpisah, bukan dipindahkan.

## Yang tetap terbuka

1. G8 BLOCKED81/90,9 gagal. Diagnosis hanya failure families; belum ada patch
   runtime tervalidasi/full90-cell confirmation baru. Raw attempt/profiling lama
   dan corrected-baseline staging tetap ada. Jangan mengubah budget/durability.
2. E2 goal-context langsung pada frozen current-state policy mempunyai exact
   feature collision. Belum dilatih dan bukan kegagalan E1. Pertama periksa apakah
   pengalaman/recurrence existing dapat membentuk aktivitas yang terpisah;
   jangan langsung mengganti encoder atau mem-port selector heuristic lama.
3. Tidak ada semantic-preserving migration antar-Brain V6/meaning/selector/implant.
   Source evidence ada, ID overlap bukan mapping semantik yang sah.
4. Komposisi tanpa observed intermediate tidak dinilai dengan satu ThinkAt(1)
   sebagai kriteria gagal D. Tetapkan hipotesis dan jalur internal yang benar
   sebelum menilainya.
5. Real Android hardware/power-loss belum diuji. Main63 tidak menambah native20
   compatibility; jangan downgrade state native untuk memudahkan deployment.

## Reproduksi teknis

Gunakan Go1.22+ (lokal memakai pinned Go1.23.2), bukan executable board-game `go`
yang ada di system PATH managed environment.

```sh
python3 scripts/prepare_continual_parent.py --destination /tmp/horizon-parent-new
HORIZON_CONTINUAL_PARENT=/tmp/horizon-parent-new \
  HORIZON_CONTINUAL_TRAINING_DIR=/tmp/horizon-e1-regression-new \
  go test ./services/ai/runtime -run '^TestContinual(EPrerequisites|E1)$' -count=1 -timeout=20m -v
go test ./...
go test -race -timeout=40m ./services/ai/knowledge ./services/ai/learning ./services/ai/runtime ./services/ai/teaching ./services/ai/bridge ./services/ai/core ./services/ai/cmd/horizon-cli
go vet ./...
go build ./...
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/horizon-cli-android ./services/ai/cmd/horizon-cli
```

Perintah E1 di atas adalah reproduksi/regresi dari parent PR68 pada salinan,
**bukan** cara melanjutkan pelatihan utama sesudahE1. Untuk tahap baru, pulihkan
checkpoint75331 dengan journal pasangannya, kunci tujuan/data evaluasi lokal,
dan berikan pengalaman baru melalui API runtime yang sudah ada. Jangan reset
model/native weights dan jangan menimpa direktori artefak ini.
