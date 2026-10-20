# Immutable PR32 recurrent-dynamics reference

Generated with Go 1.22.12 against actual commit
`b2ea1f7a4395602fa9e1a067a5daf3ad2a458018`, before physical-memory-v1.
The schema-3 journal uses recurrent-context-v1; its anchored prefix is v9.
The fixture includes repeated thought at one physical timestamp after an hour.
Reference checksums cover the complete logical state and the next thought.

Run generate.go only against the original checkout with a fresh output folder.
Historical next thoughts are explicitly pinned to their recorded dynamics;
new serving operations intentionally use physical-memory-v1.
