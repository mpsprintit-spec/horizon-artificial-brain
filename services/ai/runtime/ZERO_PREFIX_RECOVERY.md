# Revision-zero checkpoint recovery (A10)

## Problem and resulting behavior

An initial checkpoint attached to an empty journal could be recovered only
while the journal remained empty. After the first committed lesson, recovery
failed with `checkpoint journal prefix hash mismatch`. The same checkpoint
now remains a valid base for replaying that committed suffix.

The audit `Horizon_Audit_Ulang_PR43_2026-10-21.zip` reported this on PR #42 and
#43. Its unchanged public-API reproducer also failed on the PR #44 parent
`56afe5dd449eeebc6cb69e0e13561589fb54002f`. The failure is a recovery boundary
bug, not evidence that every checkpoint or every startup path was broken.

## Canonical representation and compatibility

`ReadEventLog` returns a nil slice for an empty committed journal. Historical
checkpoint writers therefore authenticated SHA256 of JSON `null`. Recovery
from a populated journal takes `events[:0]`, an allocated empty slice encoded
as JSON `[]`. The two byte sequences gave different digests for the same empty
logical prefix.

`journalPrefixHash` now normalizes **every zero-length prefix to nil** before
encoding it. Choosing the existing writer's `null` representation preserves
historical checkpoint bytes, anchors, and their enclosing checksums. There is
no new snapshot schema, checksum version, migration, fallback hash, or rewrite
of old files. Nonempty prefixes retain their exact existing hash law.

| Prefix | Hash behavior |
|---|---|
| nil, allocated empty, or zero-length prefix of a populated journal | Existing SHA256(`null`), `74234e98…2b90b` |
| One or more logical committed records | Existing SHA256 of the JSON event slice |
| Wrong digest, including SHA256(`[]`) at revision zero | Rejected by exact comparison |
| Valid anchor but corrupt enclosing checkpoint checksum | Rejected by checkpoint verification |

All committed journal records are still decoded and validated before replay.
Recovery still checks the checkpoint's sequence, complete Brain format, dedup
index, checksum, and replay contract. An empty prefix contains no prior records
to authenticate; it does not turn a malformed suffix into a valid history.

## Evidence

The new tests first failed on the unchanged parent: allocated-empty hashes
disagreed, zero→one replay failed for both empty and bootstrap Brains, and
supervisor handover failed after numeric spelling training. They pass with
the normalization in place.

The focused regression coverage includes:

- empty→empty, empty→one, and empty→two replay from the same immutable checkpoint;
- repeated recovery with identical full neural/runtime continuation and no
  checkpoint or journal rewrites;
- an actual frozen PR #44 checkpoint, unchanged checksum, original complete
  continuation, and identical next internal thought;
- byte-identical output from old and new revision-zero checkpoint writers;
- wrong anchors and corrupt checksums against empty and appended journals;
- changed nonempty committed history and an empty digest forged at a nonzero
  sequence, both rejected even with a recomputed valid snapshot checksum;
- supervisor recovery after grounding and spelling supervision, preserving the
  learned output, retiring the old writer, and continuing at the next sequence.

The frozen files and reproduction instructions are in
`testdata/zero_journal_prefix_v1/`. The original audit reproducer and local
validation results are recorded in `testdata/zero_prefix_diagnostics/`.

## Scope

The patch changes only empty-prefix canonicalization and updates the recovery
API's schema-range comment to v7–v13. Existing architecture and neural learning
laws are unchanged. It repairs a prerequisite for durable new training runs;
it does not implement productive sentence generation, role generalization,
larger language training, reduced checkpoint cost, or physical devices.

The latest expression-training report demonstrated seven whole taught phrases,
18/18 taught QA cases, 4/6 new-condition cases, and 0/18 new-actor cases. Those
are source-report results, not new training performed by this patch. P0 semantic
knowledge and productive language remain **RED / OPEN**.

No user Brain was loaded, trained, or migrated, and no service was deployed.
