# Horizon Core

Horizon Core contains Project Horizon's native neural Brain and the HC-001 infrastructure foundation: configuration, logging, middleware, domain primitives, API gateway, deployment assets, CI/CD, and tests.

## Current neural architecture

Production cognition runs through one `BrainRuntime` and canonical substrate N.
Perception, presentation, transport and safety boundaries remain adapters around
that Brain. Native operation policy, causal credit, recurrent dynamics,
plasticity, continual learning and authenticated persistence retain their
existing contracts.

MUSE, BMPS, cognitive V1–V5, the historical thinking/decision pipeline and its
research executable have been retired. Their implementation remains available
at [the pre-cleanup revision](https://github.com/mpsprintit-spec/horizon-artificial-brain/tree/6b6db6c9ec0f373fed51f38a415d0a36baf7e812).
Historical controller checkpoints and journals are explicitly refused; see
[retirement and recovery guidance](docs/OBSOLETE_COGNITION_RETIREMENT.md).
This cleanup does not replace the production Brain or promote the A–E1 checkpoint.

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

For an interactive terminal UI on Android, see [Termux chat](docs/TERMUX_CHAT.md).
Run `bash scripts/horizon-chat.sh` after installing Go. The program includes the
trained profile, remembers bounded session context, and answers through the
active canonical Brain without an LLM or API key.
The default view is a communication console; `/brain` identifies the loaded
training artifacts and live neural state, `/kemampuan` reports their evaluated
scope, and `/diagnosis` inspects the last native readout. `/lab` opens simulated
room observations only when needed. After updating code, `-latest` explicitly
selects its bundled training artifacts in a profile named by their checksums;
existing profiles are retained. The console accepts general messages, while the
connected language adapter currently supports only the bounded task below.

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
