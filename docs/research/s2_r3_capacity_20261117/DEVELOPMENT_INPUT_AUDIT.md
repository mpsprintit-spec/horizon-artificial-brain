# Supplementary stored-development input audit

The final protocol's `no-extras` condition explicitly masks channels added by
S2: forecast, history magnitude, history presence and history operation. It
retains inherited S1 temporal context, while the separate preregistered
`no-previous-operation` mask tests S1's previous-operation coordinate.

To also audit the broad interpretation of **all information beyond current
neural state and current operation**, the separate development audit removes
all those S2 extras plus nine S1 previous-state projections, S1 previous
operation and its presence flag. Current neural state and operation stay.

This supplementary diagnosis was specified after final acquisition. Therefore
it uses **only 128 already stored development Check observations**, never the
final holdout. No new acquisition, fitting, checkpoint selection, capacity
decision rule or confirmatory intervention changes. Results are descriptive
development findings in `s2r3/development-input-audit.json`; they cannot be
presented as a preregistered final-holdout test. The original confirmatory code,
analysis and checkpoint hashes remain intact.
