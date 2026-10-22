# Immutable PR37 fixture

Generated with Go 1.22.12 on actual commit
`b5b697f5570495094c9a2f2db7432708c3d166c1`, before H15 implementation.
Do not regenerate using candidate source.

The v3 journal contains stable projection dynamics, numeric grounding, action
bindings, two requests from `same-camera`, and alternating tolerant sensors.
The historical ledger incorrectly reports two independent sources. The v11
checkpoint precedes the second outcome. `reference.json` seals complete state
and continuation after a third same-camera outcome under that historical law.

Reproduction from the named checkout:

```sh
go run /path/to/generate.go /path/to/empty-output-directory
```

The output directory must exist and contain no journal. Fixture generation is
an audit operation; it is not part of candidate tests or production startup.
