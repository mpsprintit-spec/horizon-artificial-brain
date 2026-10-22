# Stable sensory population recovery (H14)

Serving uses `stable-sensory-projection-v1`. The change retains H13 uncertainty,
H12 observed feedback, local activity, physical-time retention and recurrent
context on one Brain. Recovery reads existing ProjectionPopulations; it adds no
semantic identity, answer table, input-hash database or external model.

## Reproduced failure and mechanisms

On actual PR36 (`2b5cca6891f43af8d57eda391b53b8fd300e5edf`), numeric observations
begin with the 128-unit bootstrap Brain. Each measurement is repeated twelve
times in a continuous stream. Seed 17 supplies distinct axes in eight channels.

| Patterns | Units after first cycle | Old final units | New final units | Old repeated membership reuse | New reuse |
|---:|---:|---:|---:|---:|---:|
| 1 | 132 | 132 | 132 | 11/11 | 11/11 |
| 2 | 135 | 201 | 135 | 0/22 | 22/22 |
| 4 | 141 | 273 | 141 | 0/44 | 44/44 |

The four-pattern case reproduces the audit's 273-unit result on the current
integration chain. Prototype drift invalidates nearly exact recovery, creating
132 additional units after the first complete cycle. The single-pattern control
is stable and does not establish resistance to alternating input.

Two code mechanisms contribute. Prediction-error representation learning changes
the population prototype used for sensory retrieval. It also updates cached
inactive members of a partially active population; duplicate population membership
can apply one discrepancy to the same physical neuron more than once.

## Recovery and representation learning

The new law keeps a population's observed numeric prototype separate in purpose
from its members' changing learned representations. Prediction discrepancy
updates directly observed registered members once per physical neuron, preserving
their plasticity, usage history and synaptic learning. It leaves the sensory
prototype unchanged. Explicit AdaptPopulationRepresentationAt learning from an
actual experience retains its existing ability to adapt that prototype.

Recovery checks exact observed prototype coordinates before considering generic
member similarity. Its membership must be intact, have the requested size and
belong to the unscoped measurement coordinate space. Bootstrap domain populations
and scoped sensory fields are not silently repurposed as anonymous measurement
basins. Unseen input retains the existing strongest-compatible-unit sharing and
new-member recruitment law. Learned connectivity and existing populations are
not deleted or merged.

Historical ProjectVectorPopulationAt and prediction-error representation methods
retain their old behavior. Owned numeric sensor input and GroundVectorAt use the
runtime's explicit dynamics. Scoped fields retain their existing coordinate and
receptive-field matching contract; the new discrepancy law also prevents their
sensory prototypes from drifting through global prediction error.

## Declared measurement tolerance

Unscoped SensorObservation accepts optional `noise_tolerance`, a finite maximum
per-measurement-coordinate difference in [0,1]. The default zero requires exact
prototype input. For example, an adapter calibrated to the probe's normalized
measurement uncertainty may submit `"noise_tolerance": 0.02`. It is an explicit
resolution contract, not a confidence probability or inferred semantic equivalence.

Tolerance applies only to channel, spatial and temporal measurement values.
Section markers, coordinate-group lengths and supplied confidence remain exact.
It is not accepted for factorized Fields, whose existing receptive coordinate
contract remains separate. Older dynamics reject nonzero tolerance.

Exact evidence always wins, including when other prototypes lie within tolerance.
Otherwise the whole population is recovered only if exactly one eligible
prototype lies within every declared coordinate bound. Overlapping candidate
basins recruit a distinct population rather than arbitrarily selecting/merging
prior experiences. Two physical states inside an instrument's indistinguishable
measurement range cannot be proven semantically different by this geometric law.
The adapter must supply a justified bound; the default makes no such assumption.

## Behavioral evidence and controls

Twenty axis/sign seeds each test 1/2/4 repeated patterns: 60 runs and 1,540
post-first-cycle comparisons recover all original populations, with zero new
units after warm-up. The zero-growth acceptance bound is fixed before the runs.
Member representations continue to move in alternating/multiple-pattern cases;
seed 17 also inserts internal thought between observations.

For two distinct measurement templates with independent bounded perturbations
of at most 0.01 per coordinate, a declared tolerance 0.02 gives 48/48 reuse and
zero post-warm-up growth. The same noisy samples under zero tolerance give 0/48
reuse and 144 additional units. This ablation isolates tolerant measurement
recovery from protection against prediction-driven prototype drift.

Learned A→B and C→D numeric episodes are evaluated using each outcome's first
population, never a union of later members. On seeds 7/17/47, held-out cue probes
give correct-outcome mean-activation margins 0.499992 and 0.525948. Removing the
learned source/outcome synapses reduces that advantage; population stability alone
does not supply the answer. Numeric processing creates no lexical annotations.

The H13 numeric probe now retains one outcome population instead of eleven:
confidence for the exact cue rises from historical PR36's 0.081374 to 0.794158,
with unknown mass 0.205842. Undeclared perturbations still produce zero confidence
when no exact empirical population is recognized. This is not proof of universal
calibration or noise invariance.

Mechanism tests compare the historical and new discrepancy laws, verify ongoing
active-member learning, no inactive-member drift and exactly one update despite
duplicate memberships. Additional tests cover close-but-distinct inputs,
overlapping noise regions, exact-prior precedence, protected coordinates,
detached responses and invalid tolerance before journal/state mutation.

## Persistence, adoption and limits

Journal schema 3 explicitly records `stable-sensory-projection-v1`; unsupported
readers reject it. Numeric GroundVectorAt now records its actual dynamics and
surface encoding, so replay cannot silently execute a different projection law.
Sensor intents include the declared tolerance in their existing authenticated
payload. Checkpoint v11 and canonical Brain v3 remain unchanged: all persistent
state already exists in population prototypes, members and neural histories.

The immutable evidence_uncertainty_v1 fixture was generated using Go 1.22.12 on
actual PR36. Alternating old-law sensor observations, full state and the next
historical thought match from genesis and a v11 prefix. New-law recovery compares
numeric grounding plus noisy observations from genesis, suffix and standalone
checkpoint, followed by the next thought and sensor observation.

Previously drifted prototypes are not magically reconstructed. Adopting an old
Brain can require one new receptive population for an input that no longer
matches its stored trace; existing history is retained. Ambiguous/unbounded noise,
changing calibration/confidence/layout, long-term retention, sensor drift and
broader perceptual generalization require further evaluation. H15 independence
accounting and H23 task generalization retain their own acceptance criteria.

Local Go 1.22.12 validation passed: root-module `go test ./...`, `go vet ./...`,
`go build ./...`, race tests for knowledge/learning/activation/runtime/Bridge/core/
CLI, and `make bridge-smoke`. The separate nested Go modules are unchanged and
are not covered by the root command.
