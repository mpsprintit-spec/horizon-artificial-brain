# S2-R4 — registered native temporal-error experiment

Source branch #77, verified `6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c`.
Main remains `11c6e0a78547d68a0be8f78b3c801e912722608d`.
Every research arm starts at the original R2 continuation 80735, checksum
`9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3`.
No R3 trained arm is a source; no production replacement or S3.

## Development and information gate

Three environment revisions are exploratory and all remain archived. V1
conditional history benefit 1.896% (CI crosses zero); v2 6.786% (positive,
below the prospectively recorded 10% information gate). V3 reduces the amount
of last-poststate mixed into fresh sensory drive from .10 to .02, so current
input alone does not nearly reveal the persistent gain. It retains real
sequential state continuity. V3 independent audit streams give 26.279%
conditional RMSE benefit, CI95 [19.584%,33.040%], satisfying the unchanged gate.

Each development condition has 24 independent streams of 32 actual operations.
Streams 0–7 Fit, 8–15 model-selection Check, 16–23 information audit; first two
steps excluded from information analysis. Auxiliary polynomial and RBF ridge
comparators condition on current state, operation, previous state/operation,
S1 forecast, history operation and presence. Only raw history magnitudes are
added in the nested comparator. Check selects diagnostic model configuration;
audit streams never fit it. These comparators never generate Horizon runtime
predictions or supply synaptic weights.

V1 timestamps are held constant as a controlled variable; WAL revisions provide
strict logical order. V2/v3 and every training/final experience use strictly
increasing event times. V1's expensive full-state read guard was replaced by
activation/sequence read guards after the read-only property was tested; target
generation is unchanged. One v2 development learning preflight rejected future
dated evidence before mutation. Replaying the complete observation suffix into
the learner fixes provenance, without changing admission or thresholds.

## Environment and observed target

Actual native propagate operations receive an observed experimental gain;
the selected operation, canonical graph and policy do not change. IID gain is
the negative condition; persistent gain uses a two-state process that redraws
with probability .07. Gain = .10 or .97 plus uniform jitter of width .025.
Stop probability .18. Shifted gain centers .17/.92 and jitter .035; independent
fresh drive is widened. Population 5 receives .40–.62 drive and downstream
populations 6/7 .02–.06; other populations .06–.62 (shifted .08–.68).
Prestate = .98 fresh drive + .02 previous actual poststate. This is an external
experience generator, not a second Brain or answer function.

Gain and latent regime are stored only in environment evidence/native WAL.
Neither enters S1/S2 encoded inputs. Pre-operation input/forecast ledger is
fsynced before the target native command. S1 predicts three post-operation
state projections; actual S2 target is mean absolute difference between those
forecasts and actual projections. Equivalently it is change-prediction error
after subtracting the same prestate projection. No target is hand assigned.
Each stream resets history only at its boundary. Native episodes are closed
per operation because the preserved policy permits two steps; the outer
stream carries real poststate, previous prestate and observed residual.

## Native learning arms and equal opportunity

Seeds **37,83,151** are data/environment seeds; native fitting is deterministic.
All start from identical 80735. Per seed, 16 fresh streams ×16 steps in each
condition. Four lessons per arm, each two complete Fit streams (32 rows) and
two disjoint Check streams (32 rows). Stream sets advance in fixed order.
Training seed `2500000+1000*seed`; IID/persistent use common random sensory
drives/actions but actual states differ after their actual gain interventions.
All observations are replayed before batch learning; there is no online
claim. Complete Fit/Check and receipts remain in each arm's full journal.

| Arm | Capacity | Data | Native mechanism/input |
|---|---:|---|---|
| A32 | 32 | Persistent | Existing S2-R2/R3 learner, audited only |
| B32 | 32 | Same as A32 | Generic task-local leaf-direction refinement search |
| C32 | 32 | Same as A32 | A32 with only history magnitude coordinates masked |
| B0-32 | 32 | Same as A32 | B32 with identical magnitude mask |
| D32 | 32 | IID | Existing learner, history present |
| D0-32 | 32 | Same as D32 | Identical learner, magnitude masked |
| A64 | 64 | Same as A32 | Prespecified supportive capacity diagnosis |
| C64 | 64 | Same as A32 | Supportive no-history control |

The audit-only protocol executes the exact old solver/search/admission law
and adds detailed candidate/solver receipts. Its explicit new protocol makes
old readers reject unsupported research WAL rather than silently misreplay.
The proposed refinement changes only fit-derived kind-1 leaf directions unused
by every other known head and parent circuit. It searches normalized residual
directions at fractions .25/.5/1, never names history coordinates, never assigns
readout/history weights, and does not add features. Native Check improvement,
signed bounds, precision, historical witnesses and shared protection remain.
Development found its trials rejected by retention; no successful repair is
assumed. B32 is an attempted repair, never an assertion of repaired ability.

All arms have the same four learning opportunities. Final = lesson 4, no
checkpoint selection. Capacity 64 never changes default or supplies a
canonical successor. B32 is the prespecified attempted-repair contrast even
if identical to A32; every negative result is reported.

## Confirmatory lock, measurements and decisions

Before final acquisition, freeze scientific source commit, code hashes,
all 24 final checkpoints, seeds, protocol, sample counts and analysis files.
Three new conditions: persistent seed 2600037, IID 2700037, shifted 2800037,
each 32 new independent streams ×24 operations. None used for Fit/Check,
repair choice or checkpoint choice. One-shot acquisition guard prevents rerun.
All 768 cases/condition enter prediction metrics; report history-available
cases separately for intervention analyses. No fixed/learned model selected
from final outcomes.

Primary paired contrasts: B32 vs B0-32, A32 vs C32; supportive A64 vs C64;
D32 vs D0-32 on IID. RMSE/MAE/correlation, fixed calibration bins
[.01,.03,.06,.10,.20,.30], high/low error discrimination (development Fit
tertiles frozen), per-stream improvement and shifted generalization.

Magnitude ablation zeros only the three history-magnitude input channels,
preserving presence/previous operation. Ten target-blind conditional history
permutations: exact current/previous/history operation and presence; donors
from other streams among eight nearest standardized current-state/previous-
state/forecast neighbors. Fit-only scaling; fixed RNG seeds 3100031..3100040.
Nearest-neighbor conditional permutation is approximate, not an exact
conditional randomization test; report distance and balance diagnostics.
No-history learners retain their registered mask at inference. Full feature
traces/individual feature clamps use the first eight whole final streams,
without fitting or checkpoint mutation.

2000 paired hierarchical bootstrap replicates resample training seeds and
whole independent final streams, preserving within-stream order and common
test cases across paired arms. Also report each seed and stream; three
training seeds give limited population-level precision. Confirmatory benefit
requires >=10% relative RMSE gain with lower CI >0, same direction in all three
seeds, history ablation and conditional permutation harm with lower CI >0,
and shifted gain. No harmful IID reliance: <=5% relative RMSE harm compared
with registered no-history control, with uncertainty reported. Failure of
any requirement prevents HISTORY LEARNING DEMONSTRATED. Multiple exploratory
subanalyses are labeled; no significance fishing or universal no-information
claim from a failed comparator.

## Retention, persistence and publication

Native old-witness gates are unchanged. Separately, S1 maximum prediction
change <=1e-7; A–E1 oracle delta <=1e-7 with C=6/6, E1>=35/36; historical S2
RMSE <=1.05×the actual 80735 source RMSE +1e-7 on stored R2 cases. Policy and
witness bank exact. Any old-ability drift blocks promotion irrespective of
new scores. Historical 0.00957043 is the R2 report reference, not assumed to
be the exact 80735 continuation's score on every stored evaluation.

Every arm retains full historical WAL bytes, source checkpoint, four block
checkpoints and fixed final checkpoint. Exact native replay from 80735,
restart prediction equality, rollback, signed parameters and source hashes
are required. Additional legacy replay from 76617 proves reader compatibility.
Source Brain remains unchanged. Report engineering acceptance separately from
information, representation, learned utilization and benefit. S2 remains
PARTIAL unless all historical capability gates actually pass.

One non-draft research PR targets the verified S2-R3 branch; main is untouched.
