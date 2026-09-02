# Behavior Service

Behavior Service is the HC-002 scaffold that converts Decision AI outputs into executable behavior plans and dispatchable action execution state.

The service reuses shared HC-002 domain contracts from `services/ai/domain` through the AI module dependency. It defines architecture, interfaces, repositories, handlers, HTTP contracts, and gRPC contracts only; it does not implement autonomous behavior business logic.

## HTTP

- `POST /behavior:execute`
- `GET /behavior/plans/{id}`
- `GET /behavior/executions/{id}`

## gRPC

- `BehaviorService.ExecuteBehavior`
- `BehaviorService.GetBehaviorPlan`
