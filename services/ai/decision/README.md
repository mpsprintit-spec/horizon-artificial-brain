# Decision AI

Decision AI is the HC-002 core brain scaffold for situation assessment, risk analysis, goal planning, behavior selection, multi-objective optimization, and context-aware decision creation.

It reuses shared contracts from `services/ai/domain` and does not implement AI or business logic.

## HTTP

- `POST /ai/decision:create`
- `POST /ai/decision/assess`
- `POST /ai/decision/behavior:select`
- `GET /ai/decision/{id}`

## gRPC

- `DecisionAIService.CreateDecision`
- `DecisionAIService.AssessSituation`
- `DecisionAIService.SelectBehavior`
