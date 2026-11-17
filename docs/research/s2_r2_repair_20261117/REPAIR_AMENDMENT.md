# Engineering amendment before the first final holdout

The initial eight-block R2 acquisition is retained as a **failed engineering
revision**, not a scientific success. Its A–E1 delta is zero (C 6/6, E1 35/36),
but S1 predictions move by up to 0.00046141326817925044, violating the frozen
1e-12 preservation gate. Commands 4 and 5 accepted full-refit resolution changes
to shared features 254 and 246. The original quantization-aware witness gate
allows such small changes; a task-specific refit does not establish preservation
of all other S1 readouts. No final holdout was observed or scored.

Correct the proposal scope with an explicit, journaled `preserve_shared: true`.
Build the dependency closure of every other known head's nonzero readout.
Reject changes to a circuit in that closure, recording
`cross-query-representation-change` in the ordinary trial. Query readout weights
remain plastic. New circuits and changes to unshared circuits remain eligible.
All original admission thresholds remain unchanged. This is a minimal proposal
safeguard, not a new learner or a global permanent freeze of old synapses.
The option defaults to false so the initial R2 WAL and every historical WAL
still replay under their exact recorded law.

The revised acquisition must again start from the original authenticated S2
continuation 76617. It must not copy a counterfactual parameter, a best checkpoint,
or the failed revision's learned state. Preserve the complete failed revision.
Reuse its 512 actual development observations and their pre-operation evidence
and native-observer journal, without adding seeds or optimizing on another set.
Re-run the identical fixed eight-block acquisition schedule for verification of
this engineering repair. These are R2 verification trials; A–E1, S1 and the old
16 S2 commands are not retrained. Both revisions and the extra engineering cost
must be disclosed. Do not search for more lessons or change a scientific gate.

All four explicit proposal arms use the safeguard; unchanged v6 remains an
unaltered comparator. Combined remains the predetermined research candidate.
The revised eighth checkpoint must pass the same strict retention audit before
the only final holdout. The initial failed lock is not an authorized final
evaluation. Final sample size, seeds, numerical thresholds, decision rules and
no-tuning rule remain unchanged. If the revised retention gate fails, stop
confirmatory acquisition and report the failure; do not iterate more revisions.
