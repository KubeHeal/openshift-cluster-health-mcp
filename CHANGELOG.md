# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned — v0.3.0 (Tracked Issues)
- CI gate: verify v0.1.0 tag passes all required checks, enforce branch protection — [#107](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/107)
- `list-adrs` tool for AI reasoning about architectural decisions — [#115](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/115) `good first issue`
- ADR documentation compliance gaps (9/10 score items) — [#116](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/116) `good first issue`
- Dev container for one-click Codespaces development — [#117](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/117) `good first issue`
- OCP 4.21 support: add `release-4.21` branch, update CI matrix (Kubernetes 1.34), move 4.18 to maintenance-only

## [0.2.0] - 2026-04-21

### Added — AIOps Use Case Tools (CE v1.1.0 integration)

#### New Tools
- **`get-throttled-pods`**: Identifies pods with high CPU throttling using CFS-based metrics from CE ADR-020. Surfaces `cpu_throttle_rate` from `enriched_signals`; fallback to per-pattern `throttle_rate_pct` metadata. Configurable threshold (default 25%). Closes [#109](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/109)
- **`predict-disk-exhaustion`**: Forecasts filesystem full dates via CE ADR-018 `deriv()` analysis. Returns days-until-full, urgency classification (`critical`/`warning`/`info`/`stable`), projected full date, and daily fill rate per filesystem. Closes [#110](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/110)
- **`get-rightsizing-recommendations`**: Per-container CPU/memory right-sizing via CE ADR-019. Compares P95 usage against current requests/limits with 20%/50% headroom. Classifies containers as over-provisioned, under-provisioned, or right-sized. Configurable analysis window (7d/14d/30d/90d). Closes [#111](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/111)

#### Updated Tools
- **`analyze-anomalies`**: Now surfaces `enriched_signals` object when CE v1.1.0 returns application-level signals (ADR-017). Includes `cpu_throttle_rate_pct`, `http_error_rate_pct`, `http_response_time_p99_ms`, `throttling_detected`, `http_degraded`. Closes [#112](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/112)
- **`predict-resource-usage`**: Enriched with `capacity_forecast` block containing `forecasted_exhaustion_days` and `recommended_replica_increase` from CE `/api/v1/capacity/trends` (use case 5). Enrichment is best-effort; prediction still returned if CE endpoint unavailable. Closes [#113](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/113)

#### Client Updates (`pkg/clients/coordination_engine.go`)
- Added `EnrichedSignals` struct mirroring CE ADR-017 response fields
- Added `CapacityTrendingResponse` struct with `ForecastedExhaustionDays` and `RecommendedReplicaIncrease`
- Added `GetCapacityTrends(ctx, namespace)` method calling `GET /api/v1/capacity/trends`

### Documentation
- `CHANGELOG.md` v0.2.0 section (this entry). Closes [#114](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/114)
- `RELEASE.md` added with release runbook and branch protection references. Closes [#108](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/108)

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
