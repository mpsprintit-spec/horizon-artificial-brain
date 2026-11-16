# Canonical A–E1 production activation

## State and startup

`services/ai` ships the authenticated A–E1 checkpoint **and matching journal**
inside its executable. No Python, research directory, fixture environment
variable, extraction command, network download or training is needed at startup.

```sh
go build -o bin/horizon-ai ./services/ai
HORIZON_STATE_DIR=/absolute/persistent/horizon-state ./bin/horizon-ai
```

The default, when no existing deployment or path setting is present, is
`./horizon-state` relative to the service working directory. Keep the working
directory stable, or set an absolute persistent `HORIZON_STATE_DIR`.

| File in the state directory | Purpose |
| --- | --- |
| `brain_memory.json.checkpoint.json` | Complete authenticated native checkpoint |
| `horizon_events.jsonl` | Original committed A–E1 journal plus new WAL transactions |
| `brain_memory.json` | Complete serving checkpoint, written by normal persistence after new experience/cognition |
| `ae1-origin.json` | Immutable provenance of the original archive, checkpoint and journal prefix |
| `horizon_events.jsonl.writer.lock` | Existing exclusive journal-writer lock |

A sibling `.<directory-name>.startup.lock` is held throughout the process.
It serializes first-install publication as well as normal startup. A new or
empty directory receives the complete state via a temporary sibling directory,
file/directory fsync and atomic rename. Nonempty unrelated directories,
state-directory symlinks, ambiguous path settings and invalid state are refused.
Stale staging directories left by a killed first installation are never selected
as serving state; their names begin `.ae1-staging-`.

Before publication, `initializeProductionHorizon` calls
`RuntimeSupervisor.RecoverFromCheckpoint`, which calls the existing
`RebuildRuntimeFromCheckpointAndJournal`. Schema, checksum, committed sequence,
processed receipts and checkpoint/journal anchor must agree. The restored Brain
has identity `horizon-primary-brain`, sequence **75,331**, **10,752** native
learning updates, **57** neurons, **94** synapses and **24** PatternSynapses.
`finishProductionStartup` receives `needsBootstrap=false` and emits no bootstrap
event. Temporary validation supervisors are closed before the state is published.
Only the serving supervisor owns the active neural substrate; bridge and
continuous cognition resolve it through that owner.

Each restart verifies the original A–E1 journal byte prefix and the current
authenticated checkpoint against the complete committed journal. Later native
learning and recurrent thought append to that journal; the initial checkpoint
is never reinstalled over an evolved state. WAL suffix replay uses the existing
neural recovery path and does not invoke the physical execution gateway.

## Existing installations

Without a new directory selection, the old default paths in the working
directory take priority when any existing state is found:
`brain_memory.json`, `horizon_events.jsonl`, and
`brain_memory.json.checkpoint.json`. Existing individual path settings also keep
their previous meaning:

| Setting | Meaning |
| --- | --- |
| `HORIZON_BRAIN_MEMORY_PATH` | Existing serving memory/checkpoint |
| `HORIZON_EVENT_LOG_PATH` | Its matching committed journal |
| `HORIZON_CHECKPOINT_PATH` | Its authenticated checkpoint; defaults to memory path plus `.checkpoint.json` |

`HORIZON_STATE_DIR` cannot be combined with these individual settings. Empty
individual paths are refused; use `HORIZON_STATE_DIR` to initialize a new A–E1
deployment. An existing standalone legacy memory can still be anchored with an
empty journal by the established startup migration. Invalid or mismatched state
fails closed; it never triggers fallback to A–E1 or an empty Brain.
Invalid standalone memory is rejected before a missing journal is created;
complete serving memory requires its matching journal even if the separate
checkpoint file is absent.

## Explicit switch, backup and rollback

1. Stop the current service and all writers. Record its working directory and
   exact path environment settings. Copy the **whole stopped state directory**
   to a new backup location, preserving checkpoint, serving memory and journal
   together. If using individual paths in different directories, back up all
   three files together and retain their original mapping. Do not copy a live WAL.
2. Verify the backup bytes (for example with `sha256sum`) against the stopped
   original files. Retain the original directory too; do not rename, reset,
   truncate or splice its files.
3. Prepare an entirely **new, nonexistent** destination:

   ```sh
   ./bin/horizon-ai -prepare-ae1 /absolute/persistent/horizon-ae1
   ```

   This validates and stages the original trained state, then exits without
   opening a listener. It refuses every existing destination. Validation failure
   leaves the old deployment untouched and publishes no new serving directory.
4. Change only the stopped service configuration: remove individual path
   settings and set `HORIZON_STATE_DIR=/absolute/persistent/horizon-ae1`.
   Start normally. The readiness log reports identity, sequence and all serving
   paths. After the first immediate recurrent cycle the live revision can
   already exceed 75,331; the readiness record identifies the restored revision.
5. To roll back, stop all writers, restore the **previous configuration** and
   start from the preserved old directory (or its complete verified backup).
   Never copy an A–E1 journal into the old directory or combine neurons by ID.
   Keep the separate A–E1 directory if it acquired new experience; rollback
   selects a lineage and does not delete or merge either lineage's experiences.

Preparation is optional for clean installations and runs only once for explicit
switches. Routine service restarts use normal checkpoint/WAL recovery.

## Artifact provenance and validation

| Item | SHA-256 |
| --- | --- |
| Canonical source archive | `7452a46b6231b5af409848bf928b1fd486fbe8e563ee44baeffbfefeeaf0c4d3` |
| `e1-main-continuation/07-E1-restarted.json` | `e73f8d34e80957dd442b1b5be81811270989e30ee50d03c783fab4dfeb137537` |
| `e1-main-continuation/events.jsonl` | `ebe2ff1ed74c3adf2c0176b2aa5ab8ead09c8fbdf10c07243e724de9b8354ede` |

The embedded gzip tar is a packaging derivative of the exact original member
bytes. The existing preparer verifies source archive/manifest, payload hashes,
schema, inventory and lineage. It supports `--production-bundle` for maintainers
and `--check-production-bundle` for CI. No neural state is transformed.

```sh
python3 scripts/prepare_native_continuation.py --check-production-bundle services/ai/canonical-ae1.tar.gz
python3 scripts/prepare_native_continuation.py --destination /tmp/horizon-ae1-oracle
HORIZON_NATIVE_CONTINUATION_DIR=/tmp/horizon-ae1-oracle \
  go test ./services/ai/runtime -run '^TestNativeIntegrationSavedContinuation$' -count=1 -v
go test -race ./services/ai -run '^TestCanonicalProduction' -count=1 -v
python3 scripts/canonical_production_smoke.py --binary bin/horizon-ai
```

The unchanged saved-state oracle checks A/B/D readouts, C **6/6**, E1 **35/36**,
genesis and suffix replay, exact checksum and recurrent continuation. Production
tests additionally check real startup/HTTP input/restart, single writer, one new
native learning update, WAL crash recovery, existing-state protection and
explicit switch/rollback. They do not repeat acquisition experiments or training.

## Entry point scope

This activation is the production service built from `./services/ai`, including
its managed Bridge, sensor adapters and continuous cognition. A clean service
started from the repository root uses `./horizon-state`. Starting in a directory
with existing default state deliberately selects that state; use the explicit
switch procedure above to select A–E1 there.

`cmd/horizon-chat` still serves its existing `grounded-dialogue-v1` checkpoint,
journal and codec profile; the historical `services/ai/cmd/horizon-cli` also
retains its existing memory loader. Neither is evidence of A–E1 activation and
neither should be pointed at an A–E1 directory as if its profile formats matched.
Changing the chat lineage or training a language adapter is outside this
production activation. No new frontend or Android application is introduced.

CI cross-builds the production service and existing terminal for Android arm64.
This verifies build compatibility; execution on a physical Android/Termux device
and hardware power-loss behavior have not been tested here.
