# AI Service

Shared AI boundary for the Horizon Core AI cluster.

## HC-002 Step 1: Shared AI Contracts and Documentation

The HC-002 roadmap starts with shared contracts under the AI service boundary. The `domain` package contains API-neutral contract definitions for:

- shared domain models used across HC-002 AI modules;
- canonical domain event names and the transport-neutral event envelope;
- public interfaces for AI cluster lifecycle, inference routing, event publishing and subscription, model lookup, context lookup, Vision AI, Sensor Fusion AI, Decision AI, Learning Engine, Behavior, Drone AI, and HUD AI.

Step 1 does not implement Vision, Sensor Fusion, Decision, Learning, Drone, HUD, or Behavior business logic. Later roadmap steps must implement these contracts in order without replacing HC-001 behavior or introducing a new architecture.
