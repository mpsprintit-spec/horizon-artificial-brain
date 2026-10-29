# Metric and context composition V6

Internal opt-in contract: synaptic-metric-context-v6. The default remains V1.
V5 numerical behavior and checkpoint interpretation remain unchanged. There is
no implicit migration of an already initialized V5 Brain.

## Canonical learned representation

Each projection retains its quantized metric value and bounded firing activity:
metric = center + delta * round((raw-center)/delta); firing = tanh(code).
Queries choose a fitted activation or metric readout using independent check
evidence. Binary evidence with no metric goal retains categorical semantics.

Directions, centers, resolution, weights, bias, mode and empirical error are
signed canonical dynamic synapses. Bank27 encodes the chosen readout mode. One
additional structural control unit carries per-query observed maximum errors.
No fitted parameter table or answer cache lives outside the Brain.

Projection, activation-product and metric-product circuits are supported.
Resolution can refine all three circuit types. Precision is bounded by observed
numerical scale and representable floating point codes. This is not a claim of
unlimited sensor accuracy or exact mathematical identity.

Growth must improve a task-relevant distinction not already solved by the
existing basis. Retention protects previously accepted binary distinctions and
the numerical resolution envelope of retained experience. Witness selection
covers active feature dependencies, including parents of relational N.

## Precision and diagnosis

CompositionInput.RequiredPrecision specifies an absolute output error target.
MeasurementError optionally supplies six nonnegative error estimates per frame.
NObservation.OutcomeError supplies reported outcome measurement uncertainty.
These fields require V6; V5 rejects them before mutation.

For a precision request, acceptance compares the propagated numerical estimate
plus observed maximum residual with the requested tolerance. Confidence is not
reported as a probability. The error estimate is an empirical envelope and
quantization/supplied-error calculation; unseen dynamics can still violate it.

Without an explicit precision request, a metric head returns its value with
status metric-precision-required and does not claim accepted accuracy.

NDiagnosis records candidate mode, fit validity, rejection reasons, retained
regressions, aliases, hypotheses, requested precision and estimated error.
CompositionDiagnosis returns a detached copy under the active Brain owner.
Evidence requests remain pending even when a small average error would otherwise
hide a precision problem. No hypothesis name proves physical causality.

## Native numerical imagination

NumericalDomain defines 1..6 state coordinates, 0..6 action and context
coordinates, an explicit numeric domain ID and a fixed step duration.
Coordinates must be normalized to [-1,1] by the adapter.

LearnNumericalTransition teaches ONE target coordinate as one atomic existing
composition event. Each target has a separate durable receipt. Independent fit
and check episodes and matured after-states are required. Hypothetical frames
are rejected as observation evidence.

ImagineNumericalTrajectory is native, read-only and uses only learned heads.
Predicted values and their error envelopes feed the next step. It stops at
unknown transitions, unmet precision, numerical range or encoded-domain limits.
Actions are supplied planning candidates, never executed commands.

A rollout is marked hypothetical both globally and on predicted frames.
The engine contains no body plant, market rule or teacher after-state evaluator.
Maximum per-call rollout is16 steps. The domain codec, action candidates and
coordinate meaning are supplied; autonomous variable discovery is not claimed.

## Managed API

Initialize a blank research Brain through the existing backed-up migration:

~~~go
cfg := knowledge.DefaultCompositionConfig()
cfg.Version = knowledge.NeuralCompositionMetricV6
err := engine.InitializeNeuralComposition(cfg, backupPath, observedAt)
~~~

The active canonical owner exposes:

~~~go
sequence, receipt, err := engine.LearnNumericalTransition(targetLesson, observedAt)
trajectory, err := engine.ImagineNumericalTrajectory(request)
diagnosis, err := engine.CompositionDiagnosis()
~~~

HNL/1 and HNL/2 wire formats remain unchanged. They can teach V6 numerical
examples but do not carry metric tolerances; use the typed API for those goals.
V6 readouts never authorize motor execution.

## Current limits

32 learned circuits,32 query heads,128 retained witnesses; bounded codec and
global resource ceilings, not per-object neuron allocations. General causal
discovery, arbitrary body complexity, long-term retention beyond the budget,
calibrated probabilities and hardware control remain unproven.
