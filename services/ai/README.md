# AI Service

Shared AI boundary for the Horizon Core AI cluster.

## HC-002 Step 1: Shared AI Contracts and Documentation

The HC-002 roadmap starts with shared contracts under the AI service boundary. The `domain` package contains API-neutral contract definitions for:

- shared domain models used across HC-002 AI modules;
- canonical domain event names and the transport-neutral event envelope;
- public interfaces for AI cluster lifecycle, inference routing, event publishing and subscription, model lookup, context lookup, Vision AI, Sensor Fusion AI, Decision AI, Learning Engine, Behavior, Drone AI, and HUD AI.

Step 1 does not implement Vision, Sensor Fusion, Decision, Learning, Drone, HUD, or Behavior business logic. Later roadmap steps must implement these contracts in order without replacing HC-001 behavior or introducing a new architecture.


## Horizon Brain Visual Monitor Bridge

The production monitor is served from `web/monitor` and reads the single `BrainRuntime` through the Horizon Bridge. The Bridge exposes:

- `GET /health`
- `GET /v1/brain/handshake`
- `GET /v1/brain/snapshot`
- `GET /v1/brain/events?after=<revision>`
- `WS /v1/brain/stream`
- `POST /v1/observations`
- `POST /v1/sensor-observations`
- `POST /v1/outcomes`

For a remote GitHub Pages monitor, run the Bridge behind HTTPS/WSS. The runtime supports direct TLS with `HORIZON_BRIDGE_TLS_CERT` and `HORIZON_BRIDGE_TLS_KEY`; set `HORIZON_BRIDGE_ADDRESS` to the externally reachable listener and configure `HORIZON_MONITOR_ORIGINS` for the exact monitor origin. Set `HORIZON_BRIDGE_TOKEN` for non-health endpoints.

The browser sends camera and microphone measurements through typed numeric sensor fields. Other adapters can use `ObservationEnvelope`. Camera/microphone access requires a secure browser context and explicit user permission. The Bridge persists the canonical brain to `HORIZON_BRAIN_MEMORY_PATH` and the runtime event log to `HORIZON_EVENT_LOG_PATH`.

The runtime's complete observation/inquiry transactions, checkpoint v7 and historical recovery limits are documented in [COGNITIVE_JOURNAL.md](runtime/COGNITIVE_JOURNAL.md).

The managed facade uses `Supervisor()` and detached inspection rather than exported neural aliases. See [MANAGED_OWNERSHIP.md](runtime/MANAGED_OWNERSHIP.md) for the API migration and lifetime contract.

Written input, sequence learning, Bridge text and knowledge import share the [written text contract](runtime/WRITTEN_TEXT_CONTRACT.md). New journals use schema 2; schema 1 retains historical replay semantics. The contract documents explicit rebinding of legacy written populations and the remaining language-understanding limits.
