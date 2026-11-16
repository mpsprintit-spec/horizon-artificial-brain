# Reproduksi S2-R1 tanpa pelatihan baru

Kode harus berada pada branch diagnosis berbasis S2 research HEAD
`70dbf8a36a13abdd8c763b73566d64b7b3025b96`. PR diagnosis ditumpuk di atas
branch PR #74 agar diff hanya memuat diagnosis; PR #74 dan main tetap utuh.

## 1. Sumber dan compiler

Gunakan arsip asli yang ditautkan pada
[ARTIFACT.md](../s2_self_evaluation_20261116/ARTIFACT.md):
`Horizon_S2_Native_Self_Evaluation_20261116.tar.xz`.
SHA-256 wajib:

```text
661d007850d3026ed22636fdc4d96e9113115fb2b9e2d66856ad88e4ba60a49f
```

Jika sumber hilang atau hash berbeda, hentikan dengan BLOCKED. Jangan menjalankan
S2 Prepare/Train/Evaluate atau merekonstruksi checkpoint melalui pelatihan.

Jalankan dari root repository, dengan direktori kerja penelitian di luar
checkout. Berikut contoh path yang dapat disesuaikan:

```sh
sha256sum /absolute/path/Horizon_S2_Native_Self_Evaluation_20261116.tar.xz
python3 scripts/package_s2_self_evaluation.py --verify /absolute/path/Horizon_S2_Native_Self_Evaluation_20261116.tar.xz
mkdir -p /absolute/diagnostic-source
tar -xJf /absolute/path/Horizon_S2_Native_Self_Evaluation_20261116.tar.xz -C /absolute/diagnostic-source
```

Go **1.22.12 linux/amd64** diuji; diperlukan compiler Go 1.22+ dan Python 3.10+.
Repository memakai standard library untuk harness ini. Tidak memerlukan GPU,
layanan AI, API key atau akses kamera. Sediakan beberapa ratus MB ruang sementara.

## 2. Satu perintah reproduksi

Pastikan `--work` belum ada, terpisah dari source, dan source tidak sedang dipakai
sebagai runtime produksi:

```sh
python3 scripts/run_s2r1_diagnostic.py \
  --source /absolute/diagnostic-source \
  --work /absolute/s2r1-new-run \
  --go /absolute/path/to/go
```

Runner memverifikasi 75 payload sebelum/sesudah; mengunci hash learner;
menghasilkan tracer build-tag sementara; mengompilasi test binary; memeriksa
source dan input; mereplay hanya command historis; menjalankan semua kontrol;
dan membuat data JSON/CSV. Tracer sementara dihapus saat runner selesai.

Output utama: `analysis/summary.json`. Engineering test exit 0 bukan scientific
PASS S2. Lihat status dan batas validitas di laporan.

## 3. Menjalankan satu diagnosis

Untuk inspeksi manual, buat tracer terlebih dahulu:

```sh
python3 scripts/prepare_s2r1_diagnostic.py
```

Lalu dari root checkout:

```sh
HORIZON_S2R1_SOURCE=/absolute/diagnostic-source \
HORIZON_S2R1_WORK=/absolute/s2r1-manual-run \
/absolute/path/to/go test -tags horizon_s2r1 ./services/ai/runtime \
  -run '^TestS2R1(Source|InputAndCausal|HistoricalAdmission)$' \
  -count=1 -v -timeout 20m
```

`Source` membuat salinan checkpoint/WAL di work; test lain memakai salinan
tersebut. `Followup`, `RankedTail` dan `RecruitmentCounterfactual` dapat dijalankan
setelah source tersedia dengan env yang sama. Tidak ada `LearnComposition`
baru pada intervensi: fungsi asli hanya dipanggil untuk replay command yang
sudah committed ke WAL historis.

Setelah inspeksi manual, hapus hanya file yang dibuat generator:

```sh
rm services/ai/knowledge/zz_s2r1_trace_generated.go
```

Build biasa tanpa tag tidak menyertakan support diagnostik ataupun harness ini.

## 4. Payload dan cara membaca

| File pada `evidence/` | Isi |
| --- | --- |
| `summary.json` | Ringkasan terstruktur, source, admission, input, kontrol, partial steps |
| `code-map.json` | Revision, hash source dan lokasi deklarasi fungsi |
| `source-validation.json`, `source-after.json` | Keaslian source dan tidak adanya learning command baru |
| `input-coordinate-map.json` / `.csv` | Coordinate → source → signed synapse → fitur → efek readout |
| `feature-contributions.json` | Parent, sumber, bobot, variasi metric/activation, ablation per operasi |
| `admission-summary.csv` | Satu baris per command |
| `candidate-trials.csv` | 1.260 trial historis |
| `admission-traces.json.xz` | Full model/activation/parameter/retention traces |
| `input-path.json.xz` | Input, source graph, numerical activations dan lima kondisi A–E |
| `positive-controls.json.xz` | Dua kanal × dua kondisi variasi × tiga seeds |
| `basis-probes.json.xz` | 144 proposal basis yang tidak dipasang |
| `causal-interventions.json.xz` | Ablation edge, perubahan bobot dan intervensi koordinat |
| `learning-input-interventions.json.xz` | Pencarian dengan forecast/history removal, permutation dan scale ×4 |
| `witness-controls.json.xz` | Perbandingan kontrol witness dan retensi asli |
| `partial-step-interventions.json.xz` | Seluruh 140 partial-step parameter evaluations |
| `ranked-tail-probes.json.xz` | 224 produk ranking 9–64, termasuk semua kegagalan |
| `direction-step-contributions.json` | Pemisahan old readout/new feature dan komponen history |
| `direction-step-contributions.json.xz` | Full parameter trace counterfactual direction sebelum ringkasan |
| `direction-step-before-component-split.json.xz` | Hasil awal sebelum pemisahan history-action/error-magnitude |
| `*PROTOCOL.json` pada direktori induk | Rencana dan seeds sebelum tiap tahap |
| `*.log` | Hasil verifikasi, kegagalan comparator awal, kontrol dan regresi |

`trial.index` mulai **nol**; koordinat juga mulai nol. Nilai `check_mse` adalah
MSE. Neuron ID nol pada candidate feature berarti fitur belum dipasang, bukan
neuron kanonik tambahan. `Sources[j] == 0` berarti belum direkrut.

Untuk membaca JSON yang dikompresi tanpa mengubah payload:

```sh
python3 - <<'PY'
import json, lzma
from pathlib import Path
p = Path('docs/research/s2_r1_diagnostic_20261116/evidence/admission-traces.json.xz')
data = json.loads(lzma.decompress(p.read_bytes()))
print(data['exact_16_receipt_matches'])
print(len(data['lessons']))
PY
```

`MANIFEST.json` pada direktori laporan mengautentikasi seluruh payload laporan,
protokol, evidence dan source harness. Berkas checkpoint/WAL asli tidak
diduplikasi dalam hasil diagnosis; tetap gunakan arsip S2 yang sudah diverifikasi.

## 5. Batas interpretasi

Model counterfactual tidak menjadi checkpoint pembelajaran. Metrik check
historis tidak independen penuh: 32 episode check masuk fit melalui witness
setelah commit pertama. Angka improvement dan ablation tidak boleh digunakan
sebagai skor generalisasi baru. Dataset sintetis hanya menguji keterjangkauan
dan representabilitas kanal. S2-R2 dan S3 tidak dijalankan.
