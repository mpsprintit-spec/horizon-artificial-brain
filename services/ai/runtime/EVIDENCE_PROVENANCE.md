# Observation frequency and independent provenance (H15)

Current serving uses `evidence-provenance-v1`, retaining H14 stable projection,
H13 scoped predictive assessment, H12 observed feedback, H09 local plasticity,
physical memory time, and bounded recurrent context. The controlling contract
is [FINAL_ARCHITECTURE_LOCK.md](../../../FINAL_ARCHITECTURE_LOCK.md).

This change accounts for adapter provenance on the same persistent Brain.
It adds no semantic parser, cognitive rule engine, answer store, or LLM.
The existing bound-outcome promotion boundary remains; a report's provenance
does not authorize a device action or itself establish a fact as true.

## Reproduction and acceptance

On actual PR37 (`b5b697f5570495094c9a2f2db7432708c3d166c1`), two successful
requests from `same-camera` produce `IndependentSources=2`, confidence 0.60,
and promote a target's importance from 0.50 to 0.56. The immutable fixture in
`testdata/stable_projection_v1` was generated against that checkout with Go
1.22.12, before implementing H15.

The acceptance rule is fixed by the H15 audit: request retries contribute
nothing twice; new measurements can increase frequency, but neither request
identity, episode identity, nor changing a source label establishes independent
confirmation. Shared declared causes must prevent pseudo-independence.

| Controlled current-law input | Typed observations | Reported sources | Independent groups | Bound target importance |
| --- | ---: | ---: | ---: | ---: |
| Eight requests, same camera and declared origin | 8 | 1 | 1 | 0.50 |
| Eight distinct sources/declarations, shared cause | 8 | 8 | 1 | 0.50 |
| Either row plus a declared separate source/origin/cause | 9 | 2 or 9 | 2 | 0.56 |
| Three requests with default outcome provenance | 3 | 1 default label | 0 | 0.50 |
| Retry of an accepted outcome identity | unchanged | unchanged | unchanged | unchanged |

The eight initial requests also have eight distinct episode IDs. More episodes
do not create more independent groups. These are bookkeeping and neural
promotion experiments, not measured probabilities of real-world correctness.

## Identities and counts

The ledger retains typed records and exposes `Counts(nodeID)`:

| Count | Meaning |
| --- | --- |
| `observations` | Accepted typed node/request/source identities |
| `episodes` | Distinct nonempty adapter-supplied episode IDs |
| `sources` | Distinct nonempty reported source labels |
| `causal_groups` | Distinct nonempty declared common-cause labels |
| `independent_groups` | Connected provenance components with an explicit independence declaration and attributed source |
| `legacy_observations` | Historical accepted tuples with unclassified provenance; excluded from typed independent support |

Source, episode and group labels are trimmed. Request IDs remain opaque,
including delimiters and whitespace. JSON tuple keys prevent the historical
`request + "|" + source` delimiter collision. The first accepted typed identity
is immutable: retries cannot rewrite its lineage or contradictions.

Sources, explicit independence labels and common-cause labels form separate
namespaces. A shared source, independence label or cause joins components
transitively. A late report bridging two causes can reduce the independent
count. Source identity alone is insufficient: absent an explicit independence
declaration the independent count is zero. One source cannot multiply support
by changing requests, episodes, or declared groups. Adapters must scope labels
consistently across relays and sessions; the runtime does not infer causality.

`Evidence.IndependentSources` retains its API name for compatibility; in this
law it contains the independent **component** count, not raw source count.
Outcome confidence is the maximum reported confidence plus 0.10 per additional
independent component after the first, capped at one. Same-component frequency
adds no such bonus. Contradictions remain recorded separately.

Versioned experience and outcome-trace learning use the same component law
for evidence calibration. `NewExperienceFromObservation` retains experience
and episode identity without inventing an independence group from an event ID.
An inquiry's `CausalLink` identifies its action lineage; it is not a common-cause
or independence declaration. Separate source reports of the same inquiry have
separate evidence identities in the existing neural pattern, preserving both
provenance records while still merging shared causes for calibration.

## Adapter fields and transaction behavior

`OutcomeEvent` accepts optional `episode_id`, `independence_group`, and
`causal_group`. For example, the existing `/v1/outcomes` endpoint accepts:

```json
{
  "RequestID": "capture-request-1",
  "Source": "camera-left",
  "Success": true,
  "Observation": ["sample-observed"],
  "ObservedAt": "2026-10-20T09:00:00Z",
  "Reliability": 0.9,
  "episode_id": "capture-session-1",
  "independence_group": "declared-origin-1",
  "causal_group": "shared-capture-1"
}
```

The request must already have an explicit neural action binding. Defaults such
as `outcome` and `inquiry-outcome` describe an ingress path and do not establish
independence. Empty sources at the raw ledger API are rejected.

`InquiryResult` carries the optional independence and cause fields; its episode
comes from the observation event. Compound commands require the event and
outcome episodes to agree. The lineage reaches both the bound-target ledger
and the existing neural outcome trace. `InquiryConsequenceEvent` and explicit
learning experiences can also preserve source, episode, and causal-group
metadata. Old dynamics reject new-only fields instead of silently ignoring them.

A duplicate standalone `ObserveOutcome` remains an acknowledged journaled
command/revision, with no repeated evidence or bound-target plasticity. Compound
inquiry outcomes reject a repeated node/request/source identity **before WAL
append or grounding**, even if an adapter supplies a different observation
event ID. A genuinely new measurement needs a new request identity. This avoids
repeating observation, consequence, and preference learning on an outcome retry.

## Persistence and historical replay

Checkpoint **v12** authenticates typed observations, their lineage, exact dedup
identities, and any archived legacy ledger values. Restore validates that each
typed record, aggregate, target and dedup set agree, and owns copies of all
records. Earlier envelopes cannot contain typed outcome continuation. Anchored
v7-v11 and v12 remain supported for suffix replay; older journals retain their
original accounting and learning laws. New journal records explicitly encode
`evidence-provenance-v1` under existing journal schema 3.

Canonical Brain v3 remains the same envelope; existing experience provenance
gains optional episode and common-cause metadata. No old fixture is regenerated.
The PR37 fixture verifies full historical state and the next historical outcome
from genesis and a v11 prefix, including the old erroneous count. A serving
runtime adopts the new policy only for subsequent new commands.

Old delimiter keys cannot safely reconstruct independent origins or episodes.
On the first new typed outcome, the ledger archives old values and keys,
preserves old dedup, and computes new policy support from typed observations.
Historical neural adaptations remain in the Brain; this is not retroactive
unlearning. Typed provenance cannot be downgraded to the old ledger law.

## Validation and limits

Tests cover separate source/episode/cause counts, undeclared independence,
transitive bridges, delimiter and opaque-ID collisions, invalid input before
mutation, concurrent retries, immutable duplicate lineage, snapshot ownership,
corrupt/inconsistent lineage, and preservation of historical state.

Mixed old/new journals are compared from genesis, old v11 and new v12 prefixes,
and standalone checkpoints, including the next duplicate/new outcome and
thought. Compound inquiry tests cover provenance in canonical neural traces,
distinct source evidence IDs, retry rejection before append, and restart.
Managed HTTP ingress tests verify actual wire field names and continuation after
supervisor recovery. Existing positive promotion/closed-loop tests now declare
the independent origins they require rather than assuming distinct names prove it.

Local root-module tests, vet/build, race checks, and Bridge smoke validation are
recorded in the PR. Separate nested Go modules are unchanged and are not covered
by root `go test ./...`.

Declared metadata is not statistical proof of independence or hardware source
attestation. Hidden common causes cannot be discovered by bookkeeping. Late
bridges reduce future support but do not undo earlier neural promotions.
Contradictory evidence remains available for the existing adaptation mechanisms.
Ledger storage and component recomputation grow with accepted observations;
bounded retention/indexing and end-to-end sensor semantics remain separate work.
This closes the measured H15 accounting scenarios, not H16/H23 or every audit item.
