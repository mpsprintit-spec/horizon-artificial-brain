# Exact native retention contract, audited before R5 training

The source is exact R4 `0069e6eb27dbe9334746dd80cc31e5ed52214867`.
The mathematical law below is the unchanged `nMetricRegression` law.

For fixed circuit graph G, quantized features are
`code_j=round((raw_j-center_j)/delta_j)`, `metric_j=center_j+delta_j*code_j`,
and `activation_j=tanh(code_j)`. Kind 1 raw values are signed axes of the
encoded measurement; kinds 2 and 3 multiply parent activation/metric values.
The current query uses either all metric or all activation values, according
to its native mode. Prediction is `clip(b + w^T phi_G(x), -1, 1)`.

Only current-query weights and bias may change in fixed-basis R5. Circuit
axes, centers, deltas, parent graph, all other query weights and all operation
policy parameters stay fixed. No runtime correction head is created.
Readout and bias are stored as excitatory/inhibitory signed synapses, with
coefficient = 8 * mean(exc_dynamic - inh_dynamic). Writing replicates the
signed coefficient across the source/query population; every reconstructed
prediction is checked again. Bounds are finite `[-7.99,7.99]`; they remain
active. Coefficient storage is float64; feature quantization is a different
operation and is never removed. Source witnesses are never deleted.

For each witness r, let a_r be the old *clipped* prediction and b_r the
candidate *clipped* prediction. The incumbent numerical allowance is
`nu_r=roundoff(a_r)+sum_j |w_old,j| feature_error_old,r,j`.
The feature error includes input measurement uncertainty, projection arithmetic
roundoff, half a quantization bin, parent error propagation, and activation
uncertainty when appropriate. `roundoff(v)` is exactly 16 ulps of |v|,
with the native smallest-positive fallback at zero. OutcomeError is NOT
added by `nMetricRegression`; it appears in other precision/quality checks.
For every query group q the gate forbids:

`nImproves(mean_r (|a_r-y_r|+nu_r)^2, mean_r (b_r-y_r)^2)`.

`nImproves(u,v)` is `finite(u) && u+roundoff(max(|u|,|v|)) < v`.
Additionally each historical binary witness with `|y_r|=1`, correct old sign,
and `|a_r|>oldHead.Error+nTolerance` must retain a strictly correct sign:
`b_r*y_r>0`. Counts include both group and individual sign violations.
This is NOT a pointwise prediction-shift contract.

On a fixed nonclipping region, group squared loss is a convex quadratic
constraint, and the binary sign protections are open linear half-spaces.
Coefficient bounds are a box. With clipping the global squared loss is
piecewise quadratic and need not be convex. Changing axes/quantizer bins or
parent graphs also changes the basis discontinuously. Therefore the exact
contract is not generally one linear or convex quadratic program.

R5 solves a finite path of box-constrained weighted least-squares subproblems
on raw affine values in scaled correction coordinates. A multiplier on old
query-group squared error incorporates retention in the search itself.
It evaluates the EXACT original clipped/grouped/sign contract after each
subproblem, performs an incumbent-anchored feasible segment search only when
the candidate basis itself has a feasible zero step, and repeats the native
admission checks. It does not claim global optimality or an infeasibility
certificate. Failure means `NO_BENEFICIAL_FEASIBLE_UPDATE` within this search.

Shared feature protection recursively includes nonzero dependencies of every
other known head. Only query-local leaf axes can be refined; product parents
and circuits used by another known head are excluded. Additional witness
anchors, when used, strengthen the existing contract, with original witnesses
retained in full. Anchors are journaled evidence and are excluded from Fit.
External retention holdouts are never supplied as constraints or fitting data.

Native admission also checks independent Fit/Check evidence, valid feature
numerics (including |code|<=2^52), duplicate/range validity, precision bounds,
Check improvement and shared dependencies. These checks remain authoritative.
Default historical protocols and their replay paths do not invoke the solver.
