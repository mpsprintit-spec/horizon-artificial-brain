# Structured browser observations (H16)

This increment closes the measured browser-ingress gaps on PR38
(`828a3aa7a992e8e4732970a8cf5a35b1590b7203`). The controlling contract is
[FINAL_ARCHITECTURE_LOCK.md](../../../FINAL_ARCHITECTURE_LOCK.md): one persistent
distributed Brain, with sensors supplying measurements rather than cognition.

The neural law remains `evidence-provenance-v1`, checkpoint v12 and canonical
Brain v3. Production Go processing and historical replay are unchanged. The
browser now supplies the existing structured interfaces correctly; no semantic
parser, action classifier, LLM, answer store, or second Brain is introduced.

## Reproduced gaps and resulting paths

Executing actual PR38 browser handlers with deterministic device mocks showed
two listeners after two ENABLE MOTION clicks. A motion sample went to
`/v1/observations` with zero numeric fields, and the text box also supplied no
tokens. Both therefore reached the documented envelope/hash fallback.
Camera and microphone already used numeric field ingress before this increment.

| Browser control | Current input to the existing runtime |
| --- | --- |
| Camera | Local light/position/delta/correspondence fields through `/v1/sensor-observations` |
| Microphone | Eight timed RMS/spectral windows through the numeric endpoint |
| Motion | Attributed native IMU channels through the numeric endpoint |
| Main text box and text simulator | Ordered whitespace-delimited written tokens through `/v1/observations` |
| Non-text simulator | Explicit normalized fields/channel arrays through the numeric endpoint |

Whitespace segmentation preserves spelling and order without deciding meaning,
grammar, semantic roles or answers. Written-surface normalization remains in the
existing runtime contract. Text payload hashes authenticate the original payload;
explicit nonempty tokens supply cognition. Physical capture sends no hash-token
replacement, media file, class label, velocity, distance, or adapter-made answer.

The non-text simulator requires numeric JSON such as
`{"channel_states":[0.2,0.8]}` or `{"fields":[...]}`. Arbitrary labels/invalid JSON
are rejected rather than silently represented as sensor hashes. Its source,
adapter and normalization profile explicitly identify a synthetic input path.

## IMU coordinates, values and missing channels

The adapter follows the units reported by
[Device Orientation and Motion](https://www.w3.org/TR/orientation-event/).
Linear acceleration and acceleration including gravity are separate channels.
Rotation retains native `alpha`, `beta`, `gamma` names because their Cartesian
mapping differs across browser/spec revisions; it does not assert world attitude.

| Measurement | Native channel labels | Encoding scale |
| --- | --- | ---: |
| Linear acceleration | x, y, z; m/s² | 16 |
| Acceleration including gravity | x, y, z; m/s² | 16 |
| Rotation rate | alpha, beta, gamma; degrees/s | 180 |
| Reported hardware sampling interval | milliseconds | 1000 |

Each scalar uses the monotonic softsign transform `value / (scale + abs(value))`.
These are fixed numerical encoding scales, **not measured device calibration**.
The profile explicitly remains uncalibrated. The numerical law is continuous
and preserves sign and ordering; it does not use arbitrary hashes or ordinary
hard clipping. For encoded magnitudes below one, the inverse is
`scale * encoded / (1 - abs(encoded))` within floating-point precision.

Every physical axis has its own measurement-coordinate `kind` and stable field
ID within the `device-imu` group. This prevents exchanging X/Y magnitudes from
activating the same unordered population set. Acceleration bindings also retain
device-axis basis coordinates. Equal factors can reuse their populations across
captures without erasing their axis or mixing gravity/rotation coordinate systems.

Null/undefined axes are omitted. A measured zero remains present with value zero.
An entirely unavailable IMU sample is not sent, even when an interval exists.
Non-finite measurements and invalid intervals are rejected before HTTP. Reported
interval alone does not fabricate an integration window or continuous recording.

## Acquisition, timestamps and ownership

Camera/audio/motion source IDs include both browser-session and acquisition IDs.
Re-enabling a stopped capture starts a new physical stream for field-local
matching, while compatible measured factors still reuse the same Brain. Parent
sample IDs refer only to acknowledged samples of that acquisition.

The timestamp is captured at the browser read/delivery boundary, before feature
analysis or waiting for HTTP. It is a browser wall-clock estimate, not an exact
hardware exposure timestamp. Audio offsets retain the eight analyser windows;
motion's reported hardware interval remains a separate measured field. Actual
application-level sample gaps follow observation timestamps and can be much
larger than the hardware interval.

Text, simulator and all physical captures share one FIFO mutation queue. Text
hashing runs inside that queue, so it cannot overtake earlier physical captures
and cause their timestamps to be rejected. Camera/audio retain at most one
outstanding capture each; motion also permits one and sends at most twice per
second. Busy motion samples are dropped rather than queued indefinitely. This
rate accommodates the existing developer monitor's HTTP budget; it is not a
continuous control-loop or full H20 backpressure guarantee.

ENABLE replaces the old motion listener. STOP removes it, invalidates its
generation, and cancels unsent queued samples. Pending permission resolution and
late acknowledgments cannot reactivate a stopped capture or rewrite new-capture
state. An HTTP request already sent can still complete, as with camera/audio.

## Acceptance evidence

Node tests execute production browser handlers with fake devices and clocks.
They cover ordered text and numeric simulator ingress, axis/missing-data handling,
local perturbations, permission denial/cancellation, repeated enable/stop, bounded
sends, failed acknowledgments, capture restarts, timestamp-before-analysis, and
FIFO order across physical/text inputs. The complete monitor suite has 27 tests.

`testdata/browser_sensors_v1.json` contains actual production adapter measurements:
four IMU captures, two acoustic captures and two image captures. A Node contract
checks it against current extraction (1e-12 numeric tolerance for platform audio
math, exact metadata). Go tests consume the same fixture through real processing.
They verify retained values/coordinates/timing/provenance, unchanged factors,
small-change reuse with zero new neurons, sign-reversal separation, distinct
neural input after X/Y exchange, and absence of invented missing/lexical data.

Live execution, genesis replay, checkpoint suffix and standalone checkpoint
recovery preserve complete state and the next physical input/readout and thought.
No historical fixture or encoding is regenerated.

A separate one-axis continuity experiment trains 16 repetitions of acceleration
8 → -8 m/s². An unseen 8.1 m/s² cue recovers the original cue population and
predicts the **first** outcome population over an untrained zero control.
The required raw prediction margin was 0.02; observed margin was 1.105444.
Removing only learned cue/outcome synapses reduced it to 0.000000. This is raw
neural activation evidence, not confidence, accuracy, semantic understanding,
statistical generalization or proof of physical causation.

`make bridge-smoke` now also runs actual production text/camera/audio/IMU/simulator
handlers against the live executable and authenticated HTTP endpoints. It checks
reuse/separation, restart lineage, written stimulus tokens, persisted numeric
traces, calibration metadata and the journal view. Mock devices do not substitute
for a real phone acceptance session; no user hardware is accessed by these tests.

Local root-module tests, vet/build, runtime/Bridge race checks, monitor check/test/
build and both Bridge smoke paths are recorded in the PR. Nested Go modules are
unchanged and outside root `go test ./...`.

## Remaining limits

Real-device calibration, exposure timing, orientation convention verification,
continuous high-rate acquisition, aliasing, sensor fusion and robust world-model
generalization still need their own experiments. Image fields retain local
bindings; full spatial/temporal understanding is not established by this patch.
The legacy envelope API remains available for compatibility and old history;
past hash representations cannot be reconstructed into missing measurements.
The current browser physical controls no longer select that fallback.
