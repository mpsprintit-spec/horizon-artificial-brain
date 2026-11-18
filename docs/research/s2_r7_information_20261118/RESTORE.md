# Horizon S2-R7 — complete research recovery

Verdict: PARTIAL; H1 failed. No canonical successor or production replacement.

The numbered `.tar.zst.partNN` files are consecutive binary chunks of one
tar+zstd archive. Concatenate in numeric order, verify the combined SHA256 in
the separately supplied owner recovery note, then decompress/extract into a
new research directory. `reproduction/package_s2r7_complete.py --verify-parts`
can authenticate every payload without extracting. `MANIFEST.json` maps every
archived payload to its byte count and SHA256. The R6 archive is included in full
under `prior/`, with its original authenticated manifest and download proof.

Layout:

- `r7/`: all raw acquisitions, forecasts, outcomes, checkpoints, WAL, receipts,
  rejected candidates, development/final metrics and statistical locks.
- `source/R2-active/`: unmodified sequence80735 source checkpoint and historical
  WAL. Native checksum9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3.
- `scientific/`: full frozen source tar and Git bundle, original scientific
  source metadata, postfreeze audit companions and standalone diagnostic code.
- `bin/`: original frozen Linux amd64 test binary, audit companions and Android
  arm64 cross-builds. Do not install these as a production Brain.
- `reports/`, `evidence/`, `execution/`, `r7-execution/`: report, figures,
  cryptographic provenance and all retained successes/failed invocations.

Large closed payloads may be individually encoded as `name.json.zst` or
`events.jsonl.zst`. All bytes are preserved. `r7/maintenance/*ENCODING*.json` and
`evidence/LOSSLESS_INTERMEDIATE_CHECKPOINTS.json` record original and encoded
checksums. A `.zst` file is restored with `zstd -d --keep PATH.zst`; do not
overwrite an existing raw file. Decompress those encodings before rerunning
the original frozen analysis/training scripts. The recovery companion handles
compressed WAL and frozen-hash checks directly.

Extract `scientific/Horizon_S2_R7_Frozen_Source.tar.gz` into a new `horizon/`
directory. Overlay `scientific/companions/` preserving relative paths. The
original 823-file scientific freeze and 944-file confirmatory freeze remain
unchanged; audit files are additional companions, not revised learner code.
The Git bundle contains local scientific commit5e83ebfe85208ff1d2b82e37a137cdbad10369cc
and requires R6 parenta902b06fc87652858049f12b7e05cbf471fb74d2. The published
scientific commit4b15d7b11c58633b5f7716aa8b1c23ae8fb919b7 has the exact same
tree7b3cb56f239475dc9d5eab7d753775c2f52e1fd3; author metadata explains the
different commit hash. The source tar needs no Git history for recovery.

From the extracted archive root, verify frozen bytes:

```bash
python3 horizon/scripts/recover_s2r7_research.py \
  --source source/R2-active --work r7 --repo horizon \
  --binary bin/r7.test --seed 37 --in-place --verify-only
```

Recover all ten final models for one seed to a **new**, disjoint directory:

```bash
python3 horizon/scripts/recover_s2r7_research.py \
  --source source/R2-active --work r7 --repo horizon \
  --binary bin/r7.test --audit-binary bin/r7-audit.test \
  --revision-audit-binary bin/r7-revision-audit.test \
  --seed 37 --destination ../r7-recovered-seed37
```

Repeat independently for83,151,211,307,419. Never pass a production directory.
Recovery checks all944 frozen payload hashes, copies final checkpoints/full
WAL, replays the full80735→final suffix and runs historical oracles from the
required `horizon/services/ai/runtime` working directory. The source is never
written. Existing result files are not overwritten. On this host all60 models
passed restart, canonical/signed reconstruction, full replay, rollback and
historical WAL prefix. Median full-suffix replay29.936s excludes oracle overhead.

Interpret the original unmatched A–E1 result separately: raw source and final
candidate have different LastSequence, and `flexAt` derives probe time from
that revision. Original unmatched controls0/60 and physical-time-only0/30 are
retained. The existing R6 revision/time-matched nonlearning observer control
passes60/60 at the unchanged1e-7 criterion, C6/6 and E1≥35/36. New normal paired
retention still fails1/6; H1/history/calibration also fail. An audit program
returning success does not replace these scientific gate decisions.

Go1.22.12 and zstd are required to rebuild; Python3 with numpy/scipy is used for
analysis, matplotlib for figures. Original build tags are
`horizon_s2r7,horizon_s2r6,horizon_s2r5,horizon_s2r4,horizon_s2r3,horizon_s2r2`.
Add `horizon_s2r7_audit` for companion tests. Build from `horizon/`:

```bash
go test -c -tags 'horizon_s2r7,horizon_s2r6,horizon_s2r5,horizon_s2r4,horizon_s2r3,horizon_s2r2' \
  -o ../rebuilt-r7.test ./services/ai/runtime
go test ./...
go vet ./...
```

A rebuilt executable can differ in build paths/metadata; retain the archived
original binary for locked byte verification. For fresh independent research,
use a new work directory and the frozen protocol/scripts rather than replacing
archived outcomes. H2/H3 remain closed until independent H1 passes.
