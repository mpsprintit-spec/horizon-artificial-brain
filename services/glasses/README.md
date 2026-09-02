# HUD AI / Glasses AI

HUD AI / Glasses AI is the HC-002 scaffold for information prioritization, augmented-reality placement, target highlighting, alert notification, user attention management, and HUD rendering.

The service reuses shared HC-002 domain contracts from `services/ai/domain` through the AI module dependency. It defines architecture, interfaces, repositories, handlers, HTTP contracts, and gRPC contracts only; it does not implement rendering engines, AR business logic, or AI frameworks.

## HTTP

- `POST /glasses/hud:render`
- `GET /glasses/hud:data`
- `POST /glasses/alerts`

## gRPC

- `HUDAIService.RenderHUD`
- `HUDAIService.PrioritizeInformation`
- `HUDAIService.SendAlert`
