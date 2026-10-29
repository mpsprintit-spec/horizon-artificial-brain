# Horizon Core

Horizon Core is the foundation repository for Project Horizon cloud intelligence and control center. The repository now contains the HC-001 foundation layer: shared configuration, logging, middleware, domain primitives, infrastructure interfaces, API gateway health endpoint, deployment assets, CI/CD, and tests.

Business logic, AI processing, and HC-002 features are intentionally outside this foundation scope.

## HC-001 Foundation Components

| Component | Status | Location |
| --- | --- | --- |
| Configuration Management | Complete foundation | `internal/config` |
| Structured Logger | Complete foundation | `internal/logger` |
| Middleware | Complete foundation | `internal/middleware` |
| Shared Domain Model | Complete foundation | `internal/model` |
| gRPC / Protobuf Foundation | Complete foundation | `api/proto`, `internal/proto` |
| Database Layer | Interface + SQL foundation | `internal/db` |
| Cache Layer | Interface + memory foundation | `internal/cache` |
| Message Queue Layer | Interface + memory foundation | `internal/mq` |
| Common Utilities | Complete foundation | `internal/utils` |
| API Gateway Foundation | Health endpoint foundation | `cmd/api-gateway` |
| Docker Configuration | API gateway image foundation | `deployment/docker` |
| Kubernetes Foundation | Base deployment/service | `deployment/kubernetes/base` |
| CI/CD | Go validation workflow | `.github/workflows/ci-cd.yml` |
| Testing Foundation | Unit tests + Make targets | `internal/*`, `Makefile` |

See `docs/HC-001-foundation.md` for the completion analysis and component map.

## Trained contextual dialogue

The optional [grounded dialogue bundle](models/grounded-dialogue-v1/README.md)
includes an authenticated V6 trained Brain, matching journal, neutral I/O codec,
and frozen evaluation. It answers bounded Indonesian questions about an observed
simulated room through native numerical composition and learned spelling.
See the bundle instructions to prepare a new state directory and run the CLI or
select this Brain at production startup. Existing serving memory is never
automatically replaced. This is a research profile, not unrestricted conversation.

## Development

```bash
make fmt
make test
make validate
make build
```
