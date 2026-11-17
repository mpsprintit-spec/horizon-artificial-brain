# Preserved engineering failures before confirmatory evaluation

Initial fixed-basis development: A completed four lessons; the first B
bias-only lesson was aborted by the new strict post-install comparison:
`signed reconstruction changed admitted prediction`. All initial WAL,
receipts and failure logs remain under fixed-development and execution logs.
No final R5 holdout existed. The transaction requires native recovery.

Cause: replicating a float64 coefficient across 4/16 population edges and
averaging can change the last bit; `writeNParameter` does not make arbitrary
float coefficients bit-exact under repeated summation. Old protocols remain
unchanged. R5 candidates now use a 40-bit dyadic significand grid BEFORE
fitting/evaluation and exact admission. This leaves summation headroom and
permits bit-exact signed reconstruction. The solver still checks original
retention after this quantization; no tolerance was relaxed. New/changed
circuit parameters use the same grid, unchanged source circuits are untouched.

Rerun fixed-basis development from fresh source copies as v2. Preserve original
negative evidence; no source/checkpoint/holdout selection from failed data.
The initial implementation scaffold preceded completion of calibration
computation, but no solver experiment was executed before source verification
and the calibration diagnosis. Scientific phase ordering was preserved.
