# Factorized sensor ingress

Camera, microphone and motion adapters can submit local physical measurements to the
one canonical Brain through `POST /v1/sensor-observations`. This increment
implements the sensor-state and factor-reuse boundary in the Horizon developer
brief. It does not establish object recognition, depth perception, language
understanding, or a completed artificial brain.

## Contract and ownership

The endpoint accepts `SensorObservation` directly, under the Bridge's existing
authentication, origin, rate, and body-size limits. Its JSON decoder rejects
unknown properties and multiple JSON objects. The default Bridge limit is now
600 requests per minute, allowing two one-second capture streams and their
monitor snapshot refreshes. Explicit configured limits still take precedence.

```json
{
  "modality": "vision",
  "source": "browser-camera",
  "timestamp": "2026-10-19T08:00:00Z",
  "sample_id": "frame-1",
  "confidence": 0.5,
  "provenance": {
    "adapter": "browser-numeric-v1",
    "calibration_id": "uncalibrated-relative-v1"
  },
  "fields": [
    {
      "field_id": "patch-0:light.rgb-v1",
      "group_id": "patch-0",
      "kind": "light.rgb-v1",
      "values": [0.8, -0.4, 0.2],
      "coordinates": [-1, -1, 0, 1]
    }
  ]
}
```

Each field has a stable local ID, group ID, numeric coordinate-system kind,
1–64 values, and up to four spatial coordinates. Values and coordinates must
be finite within `[-1,1]`. At most 128 fields are accepted. A group denotes a
patch or capture window; it does not assert an object identity. `kind` describes
measurement coordinates, such as RGB or RMS, rather than a semantic class.

`temporal_offset_ns` is the capture-window start relative to the observation
timestamp. `temporal_scale_ns` is its duration. A nonzero window must end at or
before the observation time and fit within the preceding minute. Omitted
timing denotes an instantaneous measurement. Each observation retains the
existing monotonic timestamp and duplicate-sample checks. Browser sends are
serialized through one queue, with at most one outstanding capture per source.
Physical capture sources include the browser session and acquisition IDs, so separate
sessions or capture restarts do not masquerade as the same physical stream for transition
learning. Compatible factors can still reuse the same substrate populations.

The complete multi-field observation acquires one supervisor mutation lease
and one runtime transaction: validate, append sensor intent, project factors,
run cognition, learn field transitions, record sensory trace, append commit.
Only the committed observation is published to telemetry. Uncertain journal
writes retire availability until recovery. Checkpoint persistence remains inside
the supervisor lease. The success response includes revision and local neural
bindings; it does not contain a semantic answer.

## Reuse, bindings, and time

Projection searches the existing neural substrate using measurement scope:
modality, field kind, and receptive temporal scale. Receptive durations round
to the nearest power of two in nanoseconds, so minor camera scheduling jitter
does not allocate duplicate populations. Exact offsets and durations remain
unchanged in bindings and traces, including during replay. The scale bands
have boundaries; they are a measurement-resolution policy, not learned timing.

A compatible population with prototype similarity at least `0.90` is reused.
A distinct factor may share a compatible unit while recruiting new members.
Incompatible scopes and the legacy unscoped projector cannot accidentally use
these units as anonymous candidates. These are receptive-field constraints
within the same Brain, not separate modality brains or semantic rule engines.

Identical color in two patches can reuse a population while retaining separate
group IDs and spatial bounds. Altering a temporal delta does not recreate an
unchanged RGB factor. Temporal reinforcement matches a field's own preceding
source/group/scope. Windows within a capture are ordered by their measured
offsets; simultaneous spatial patches do not acquire an invented scan order.

The Brain retains up to 64 `sensory_traces`, including measured values,
coordinates, exact timing, population IDs, provenance, confidence, and the
observation's prediction error. They are canonical Brain state saved alongside
the existing network and learning traces. The monitor exposes and inspects
these traces and scoped populations. Prediction error remains an aggregate
runtime signal; this increment does not add field-level causal attribution.

## Browser measurements and limits

Camera capture downsamples to a maximum 160×120 canvas and measures 16 fixed
patches: luma intensity, RGB, signed local brightness gradients, spatial
position, and subsequent RGB differences. The first frame has 64 fields;
subsequent frames can have up to 128, including local correspondence quality,
supported image displacement, and differences from shared apparent image
motion. It sends numerical measurements, not JPEGs, base64 media, hashes,
class labels, or adapter-produced distances. See
[VISUAL_CORRESPONDENCE.md](VISUAL_CORRESPONDENCE.md) for coordinates, support
criteria, ambiguity, and the boundary before learned relative depth.

Microphone capture samples a 16,384-sample analyser buffer once per second and
divides it into eight timed windows. Each carries RMS and 16 Hann-windowed
Goertzel bands at 125–2000 Hz; seven windows also carry RMS onset differences.
Spectral shape is normalized relative to RMS, allowing equal-frequency signals
with different amplitude envelopes to share that factor while preserving
envelope and timing differences. These are physical bands, not pitch or word
labels. The fixed band range and eight windows limit frequency/time resolution.

Analyser windows can overlap or have gaps depending on sample rate and
processing delay. Capture is not a continuous recording or a speech decoder.
Confidence `0.5` is nominal metadata and the adapter explicitly reports
uncalibrated relative measurements. Real-device calibration and an end-to-end
phone camera/microphone acceptance session remain outstanding.

Camera and microphone require the user's START action and browser permission.
STOP closes media tracks and prevents further queued captures for that
generation. It does not revoke an HTTP request already sent. H16 additionally
migrates motion to native-axis numerical fields, supplies ordered written tokens
from text controls, and requires numeric JSON for the non-text simulator. See
[BROWSER_SENSOR_STRUCTURE.md](BROWSER_SENSOR_STRUCTURE.md) for encoding, acquisition
lifecycle, FIFO timing, the continuity probe and limits. Public Brain encapsulation
remains a separate contract.

## Compatibility and evidence

Flat `channel_states` observations retain their old projection and replay path.
They cannot be mixed with `fields`; factorized input cannot call `NeuralVector`
and silently flatten its bindings. New canonical properties are optional, so
existing state without sensory fields retains its prior JSON encoding.

Coverage includes shared factors with distinct bindings, unchanged static
factors after a temporal change, capture jitter versus different time scales,
capture-order learning, malformed requests before mutation, uncertain commits,
full canonical equality after genesis replay and checkpoint suffix recovery,
continued learning after recovery, and managed Bridge handover. Browser tests
use analytical pixels/tones and mock media devices executing the production
capture handlers. The live Bridge smoke exercises the executable, typed HTTP
ingress, shared neural factors, monitor traces, persisted memory, and event
retrieval. The smoke's event retrieval is not a substitute for reconstruction;
reconstruction equality is asserted in runtime and Bridge tests.

Run `go test ./...`, `go vet ./...`, `go build ./...`,
`go test -race ./services/ai/knowledge ./services/ai/runtime ./services/ai/bridge ./services/ai/core`,
and `make bridge-smoke`. In `web/monitor`, run `pnpm check`, `pnpm test`, and
`pnpm build`.

Local image motion and correspondence quality now provide one part of the
brief's required evidence. Learned relative-depth hypotheses, camera/body
motion disambiguation, additional occlusion/perspective evidence, strategy
traces, and causal attribution still require their own behavioral evidence.
