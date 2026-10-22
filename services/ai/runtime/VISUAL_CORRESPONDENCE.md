# Local visual correspondence evidence

This increment extends the numeric camera path from factorized sensor ingress.
It supplies image-plane motion and measurement-support evidence to the same
Brain. It does not construct a depth ordering, object identity, camera pose,
or an occlusion fact in the adapter. The developer brief's learned relative-depth
hypothesis and revision tests remain an outstanding milestone.

## Measurements

The browser retains one acknowledged downsampled luminance grid for the next
capture, alongside the existing patch means. The grid stays in browser memory;
only derived numeric fields are submitted. Camera provenance now identifies
`browser-light-motion-v2`. Capture cadence remains once per second with the
existing per-source backpressure and serialized request queue.

For each of the 16 fixed patch centers, `visual-correspondence.js` compares a
7×7 luminance template with integer displacements within ±6 pixels in the next
frame. Matching minimizes the mean absolute difference after removing each
patch's mean brightness. This tolerates additive brightness changes without
inventing image motion. It does not compensate for arbitrary gain, lighting,
deformation, compression, blur, or exposure changes.

Measurements require texture standard deviation at least 0.015 in `[0,1]`
luminance, residual at most 0.08, and uniqueness margin at least 0.10 against
the best competing minimum outside a one-pixel neighborhood. A reverse search
must also have support and return within one pixel of the starting point.
Search extrema and cropped search neighborhoods remain unsupported. These
explicit engineering thresholds are uncalibrated measurement gates, not
semantic rules or probabilities of correctness. Integer displacement has
one-pixel resolution; subpixel motion is unresolved.

Each attempted patch adds a quality field. Supported patches additionally
add displacement, and may add motion relative to a shared image reference.
Kinds include the search radius and image geometry, for example
`light.flow.displacement.r6px.160x120-v1`. Different image resolutions therefore
do not silently share the same pixel-coordinate projection scope.

| Factor | Numeric values | Interpretation |
| --- | --- | --- |
| `displacement` | `[dx/6, dy/6]` | Apparent movement from the prior patch center to the current image; positive x is right, positive y is down. |
| `relative` | `[(dx-referenceX)/12, (dy-referenceY)/12]` | Difference from the componentwise median supported image displacement; denominator 12 covers the difference of two ±6 searches. |
| `quality` | `[texture, residual, uniqueness, cycle, range, supported, referenceSupport]` | Measurement evidence as defined below, including when displacement is unavailable. |

Quality dimensions are:

1. `texture`: twice the prior template's standard deviation, capped at one.
2. `residual`: mean-centered matching error, capped at one.
3. `uniqueness`: relative separation of the best and competing minima.
4. `cycle`: forward/reverse disagreement divided by two pixels, capped at one.
5. `range`: maximum absolute displacement divided by six; one also indicates
   unavailable/cropped search support.
6. `supported`: zero or one indicating whether the measurement gates passed.
7. `referenceSupport`: supported-patch count divided by 16 when a shared
   image reference is available; otherwise zero.

Unavailable residual/cycle measurements use penalty one; unavailable uniqueness
uses zero. These penalties must be read with the support dimension. They are
not measured photometric or cycle errors when no search could be completed.
Texture-poor and ambiguous patches omit displacement rather than emitting a
false zero-motion observation. A textured stationary patch can emit supported
zero displacement. Quality remains available to distinguish those cases.

A shared reference requires at least four supported patches spanning at least
two grid rows and columns. Its median describes apparent image motion. It is
not an ego-motion estimate: a moving foreground, camera rotation, perspective,
or several independently moving regions can affect it. Differential flow is
evidence a Brain could combine with further experience; its sign or magnitude
does not automatically mean closer or farther.

Coordinates name the prior fixed patch bounds. The end point can leave those
bounds. The group is an observation location, not a tracked object. Exact
capture offsets/durations and acknowledged parent sample IDs preserve the
temporal and provenance context. A resolution change or missing/invalid time
interval resets correspondence. The first frame remains 64 fields. With valid
history, 80 existing fields plus up to 48 correspondence fields fit the current
128-field contract; fully unsupported matching produces 96 fields.

## Brain, persistence, and inspection

These fields use the existing scoped neural projection, sensory traces, and
sensor intent/commit transaction. No second visual memory, metric-depth store,
new journal event type, or semantic dictionary is introduced. Unsupported
correspondence cannot reinforce an invented zero-displacement population.
After a missing motion observation, the next supported event does not create
an uninterrupted transition from stale motion in an earlier event. An earlier
compatible factor can still be reused when support returns.

The camera acknowledgment displays the number of supported patches. The
monitor's sensor trace inspector exposes the values, support dimensions,
geometry, timing, neural population bindings, and provenance. Existing aggregate
prediction error is retained; field-specific causal credit is future work.

## Evidence and remaining work

Tests exercise known positive/negative translations, independent region
motions whose differential sign reverses, common image motion, stationary
texture, additive light changes, periodic ambiguity, missing texture, search
boundaries, insufficient shared-reference support, and geometry/time resets.
The production capture-handler test verifies two acknowledged frames and that
the submitted body carries numeric fields rather than the retained luminance
grid. These are analytical fixtures and mock media-device tests.

Runtime regression tests verify neural reuse/separation, unknown correspondence
without fabricated continuity, complete canonical equality after genesis replay
and checkpoint suffix recovery, and reuse after recovery. The live executable
smoke feeds analytical pixels through the production JavaScript adapter, typed
HTTP ingress, neural projection, canonical memory and monitor/event inspection.
It uses an isolated temporary executable and port, then stops that executable.

No real phone camera session, calibration, large-motion tracking, subpixel
accuracy, or physical relative-depth test has been completed. At the existing
one-second cadence, typical handheld motion may exceed the search range and
correctly produce unsupported measurements. Further work needs learned
temporal/world predictions and revisable relative-depth hypotheses, with tests
that distinguish camera translation, rotation, independent motion and actual
occlusion. Neither match failure nor differential flow is sufficient by itself.

Background on apparent image motion and its assumptions:
[OpenCV optical-flow documentation](https://docs.opencv.org/4.13.0/d4/dee/tutorial_optical_flow.html).
This implementation uses bounded mean-centered block matching with a reverse
consistency check; it adds no OpenCV dependency.
