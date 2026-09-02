# gRPC / Protobuf Foundation

Canonical protobuf definitions live under `api/proto`. Generated Go files should target this package subtree when protobuf tooling is added to the build image. HC-001 currently defines shared request context and health-check service contracts only; service-specific business RPCs belong to later HC documents.
