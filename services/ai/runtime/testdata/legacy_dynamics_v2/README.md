# Immutable pre-H11 dynamics reference

Generated using Go 1.22.12 against actual PR30 commit
`2f1537a957b15e400566dc5ec7dba9cd67312a44`, before ordered dynamics exist.
`commands.jsonl` contains schema-2 written-text-v1 commands; the anchored prefix
checkpoint is v7. `reference.json` records full-state checksums and the next
cognitive thought and post-thought state from that original code.

Run `generate.go` with the working directory and VCS environment pointing at
that original checkout and a new empty output directory. Never regenerate
these references against the candidate. Tests normalize only the new envelope
version to v7 when comparing checksums, preserving all neural/runtime fields.
