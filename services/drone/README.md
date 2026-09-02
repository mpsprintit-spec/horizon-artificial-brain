# Drone AI

Drone AI is the HC-002 scaffold for path planning, obstacle avoidance, auto tracking, flight stabilization, energy management, and command dispatch.

The service reuses shared HC-002 domain contracts from `services/ai/domain` through the AI module dependency. It defines architecture, interfaces, repositories, handlers, HTTP contracts, and gRPC contracts only; it does not implement control algorithms, flight autonomy business logic, or AI frameworks.

## HTTP

- `POST /drone/path:plan`
- `POST /drone/commands`
- `GET /drone/state/{droneId}`

## gRPC

- `DroneAIService.PlanPath`
- `DroneAIService.DispatchCommand`
- `DroneAIService.GetFlightState`
