# Immutable PR31 ordered-dynamics reference

Generated with Go 1.22.12 against actual commit
`0f9a4651f1d31e6f1ceeb0ca4d66fddafeafa4dc`, before recurrent-context-v1 exists.
The journal uses schema 3 with ordered-observation-v1. The anchored prefix is
checkpoint v8. Reference checksums cover the complete logical state, next
cognitive thought and resulting continuation state from that original code.

Run `generate.go` from the original checkout (including its VCS environment),
with a fresh empty output directory. Never regenerate against the candidate.
Tests normalize only the new envelope version to v8 for checksum comparison.
