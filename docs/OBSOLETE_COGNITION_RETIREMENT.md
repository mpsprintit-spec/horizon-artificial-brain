# Obsolete cognitive architecture retirement

Baseline: `6b6db6c9ec0f373fed51f38a415d0a36baf7e812` (PR #70).
The archive branch `archive/pre-canonical-cleanup-20261115` points to this
revision. The full immutable source reference is
[available in Git](https://github.com/mpsprintit-spec/horizon-artificial-brain/tree/6b6db6c9ec0f373fed51f38a415d0a36baf7e812).

## Scope

This change retires the MUSE/BMPS cognitive V1–V5 actors, belief/readout models,
optimizers, learning/controller APIs, phase/terminal/coordinated research
commands, and `horizon-metareasoning-v5`. It removes the historical
thinking/decision/FSU/context/understanding/HFCC/language pipeline, unused intent
classifier and task types, compatibility interpreter, and unused weighted
multi-controller cluster. Callers in the facade and CLI no longer import those
packages or expose their result types and disabled switches.

The controller implementation is removed rather than relocated. Historical
research reports describe their original Git revisions; they are not current
production architecture or instructions to retrain those controllers.

## Preserved components

| Component | Concrete reason |
| --- | --- |
| `BrainRuntime.CognitiveProcess`, `CognitiveThink`, `CognitiveOrchestrator`, `cognitive_contract.go` | Active observation, recurrent thought and typed neural-output boundaries; no retired actor dependency. |
| Canonical N, native operation policy and causal credit | Current learned neural operation and credit mechanisms. Their update laws are unchanged. |
| Activation, recurrence, eligibility, plasticity, continual learning | Existing neural dynamics and supported learning paths; no new law, controller or performance optimization. |
| Internal `dnf.Fabric` | A pointer adapter over the same canonical Brain; owns no alternate state. Grounding, direct dynamic-channel inspection and compatibility tests still use it. |
| Public `BrainRuntime.DNF()` | Removed: it exposed a Fabric containing a mutable Brain. Existing neural tests inspect the internal adapter; boundary tests assert the exported escape hatch is absent. |
| Perception, neural interpreter and presentation adapters, sensor ingress, transport and execution safety | Necessary external boundaries; they do not compute retired cognitive decisions. |
| Native persistence, WAL, checkpoint, deterministic replay and recovery | Native and genuinely supported historical neural formats retain their original authenticated state and checksum laws. |
| Written-language/composition versions and trained dialogue bundles | These are active neural capabilities. A version number V5/V6 in a language codec is not a cognitive V5 controller. |
| Production Brain and saved A–E1 fixture/oracle | Preserved byte for byte. Promoting A–E1 to the operational default is a separate task. |

## Historical state handling

| Format | Behavior after retirement |
| --- | --- |
| Canonical Brain schemas 4–6 | Explicit refusal; these authenticate retired cognitive state. |
| Checkpoint schemas 15–19 | Explicit refusal before live state changes. |
| WAL schemas 5–13 or `cognitive_model_command` | Append and replay refused. |
| Non-null `cognitive_model` in a supported envelope or nested `brain_state` | Refused; never silently dropped or interpreted as native state. |
| Reserved cognitive projection population without its controller | Refused rather than importing orphan controller neurons as native state. |
| Canonical native schema 7, checkpoint 20 and WAL14 | Existing validation, replay, recovery and checksum contracts retained. |
| Supported earlier neural formats, composition and language state | Existing neural compatibility remains; retirement is specific to the obsolete controllers. |

There is no verified mapping from historical actor/optimizer state to native
operation-policy state. No automatic migration is offered. On refusal:

1. Keep the original checkpoint and complete journal unchanged.
2. Inspect a copy using the exact pre-cleanup revision in a separate checkout.
3. If native operation is required, select a separately authenticated native
   checkpoint and its matching journal through the existing explicit restore
   procedure. This cleanup does not select or replace a production checkpoint.

Negative tests cover ordinary and escaped JSON keys, nested payloads, orphan
populations, retired schema ranges, and mixed valid-prefix/retired-suffix
journals. They assert unchanged live state and unchanged input bytes on refusal.

## Validation and inventory

The completion evidence, exact deleted-file and API inventory, protected source
hashes and final production dependency closure are recorded in
`docs/research/obsolete_cognition_retirement_20261115/`.
Gate G8 criteria and harnesses, the A–E1 PR #69 oracle, retention fixtures and
production checkpoint are not modified by this change.
