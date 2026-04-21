# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-21

### Added
- Go-based standalone MCP server for OpenShift cluster health ([ADR-001](docs/adrs/001-go-language-selection.md))
- Official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk v1.2.0`) adoption ([ADR-002](docs/adrs/002-official-mcp-go-sdk-adoption.md))
- Standalone server architecture decoupled from openshift-aiops-platform ([ADR-003](docs/adrs/003-standalone-mcp-server-architecture.md))
- HTTP/SSE transport for OpenShift Lightspeed compatibility ([ADR-004](docs/adrs/004-transport-layer-strategy.md))
- Stateless design — no database, cluster state served from Kubernetes API on demand ([ADR-005](docs/adrs/005-stateless-design.md))
- Integration architecture with Coordination Engine, KServe, and Prometheus backends ([ADR-006](docs/adrs/006-integration-architecture.md))
- RBAC-based security model with minimal cluster permissions ([ADR-007](docs/adrs/007-rbac-based-security-model.md))
- Distroless container images for minimal attack surface ([ADR-008](docs/adrs/008-distroless-container-images.md))
- Architecture evolution roadmap for v0.1 → v1.0 ([ADR-009](docs/adrs/009-architecture-evolution-roadmap.md))
- Version compatibility and upgrade roadmap for OCP 4.18–4.20 ([ADR-010](docs/adrs/010-version-compatibility-upgrade-roadmap.md))
- ArgoCD and MCO integration boundary definitions ([ADR-011](docs/adrs/011-argocd-mco-integration-boundaries.md))
- Non-ArgoCD application remediation strategy ([ADR-012](docs/adrs/012-non-argocd-application-remediation.md))
- Multi-layer Coordination Engine design integration ([ADR-013](docs/adrs/013-multi-layer-coordination-engine.md))
- Branch protection strategy for `main` and `release-*` branches ([ADR-014](docs/adrs/014-branch-protection-strategy.md))
- MCP tools: `get-cluster-health`, `list-pods`, `list-incidents`, `trigger-remediation`, `analyze-anomalies`, `predict-resource-usage`, `calculate-pod-capacity`, `analyze-scaling-impact`, `get-remediation-recommendations`
- MCP resources: `cluster://health`, `cluster://nodes`, `cluster://incidents`
- Helm chart `openshift-cluster-health-mcp` v0.1.0 supporting OCP 4.18–4.20
- Kubernetes manifests under `deploy/kubernetes/` (namespace, RBAC, deployment, route, BuildConfig)
- Optional Coordination Engine integration via `ENABLE_COORDINATION_ENGINE` environment variable
- Optional KServe integration via `ENABLE_KSERVE` environment variable
- OCP-version image tagging strategy: `quay.io/takinosh/openshift-cluster-health-mcp:4.x-latest`

[Unreleased]: https://github.com/KubeHeal/openshift-cluster-health-mcp/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KubeHeal/openshift-cluster-health-mcp/releases/tag/v0.1.0
