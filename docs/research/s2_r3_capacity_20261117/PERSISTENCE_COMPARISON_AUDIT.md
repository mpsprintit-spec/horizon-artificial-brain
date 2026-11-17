# Persistence comparison correction

The original six complete-state 80735 suffix replay comparisons and restart
checks passed byte for byte. A separate `reflect.DeepEqual` over the entire
`NResolutionState` returned false and made the first analysis mark persistence
false. That whole structure includes diagnostic trial annotations, not only
feature metadata.

A targeted independent native replay of the already recorded last block checks
every feature, parent, head topology, source, control and capacity field by
structural equality, the entire serialized NResolution and complete state by
byte equality, and enumerates every raw struct difference. The correction is
allowed only if all differences are `Last.Trials[].RejectedBy`: nil versus
allocated **empty** slices (length 0/0). Both serialize identically because the
diagnostic JSON field is `omitempty`. No neural parameter, metadata value,
witness, command, policy, data or prediction is changed.

`s2r3_metadata_audit_test.go` writes the exact paths and matching SHA-256 values.
`finalize_s2r3_metadata_audit.py` fails closed on any other difference. It retains
`persistence-initial-struct-comparison.json`, records a hashed amendment, and
sets the feature-metadata gate using the independently verified feature and
serialization contract. Initial analysis output is preserved separately. The
frozen analysis script is rerun without modification, new data or final-holdout
re-evaluation. Scientific decisions and historical R2 results are unchanged.
