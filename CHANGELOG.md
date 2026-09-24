# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- ADR documentation compliance gaps (9/10 score items) — [#116](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/116) `good first issue`
- Dev container for one-click Codespaces development — [#117](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/117) `good first issue`

## [1.2.0] - 2026-09-24

### Changed — OCP 4.22 GA Release
This release marks the production-ready v1.x line for OpenShift 4.22.

### Dependencies
- Bumped `github.com/modelcontextprotocol/go-sdk` from 1.6.0 to 1.8.0 — security hardening (JSON depth limits, SSE event size caps, request body bounds), session leak fixes, per-request cache control
- Bumped `github.com/stretchr/testify` from 1.11.1 to 1.11.2
- Bumped `actions/setup-go` from 6 to 7

### Fixed
- Release branch CI: `NodesResource` tests now skip gracefully when no Kubernetes cluster is available (replaced `require.NoError` with `t.Skipf`)
- Release branch CI: OpenShift `oc login` step now uses `continue-on-error` so stale cluster credentials don't block the entire CI pipeline

### Documentation
- Linked `RELEASE.md` from `.github/CONTRIBUTING.md` (completing [#108](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/108) acceptance criteria)
- Audited and remediated stale documentation across 6 files ([#180](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/180))

## [1.1.0] - 2026-09-23

### Changed — Infrastructure & Maintenance

### Infrastructure
- Upgraded Go from 1.24 to 1.26 across all CI workflows and Docker images
- Added OCP 4.22 support: `release-4.22` branch, Kubernetes 1.35 compatibility, updated CI matrix. Closes [#121](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/121)
- Added Apache 2.0 LICENSE file
- Added Dependabot auto-merge workflow and k8s.io dependency grouping
- Mirrored coordination-engine release process (RELEASE.md, RELEASE-CHECKLIST.md, VERSION-STRATEGY.md, playbooks)

### Dependencies
- Bumped `actions/checkout` from 6 to 7
- Bumped `codecov/codecov-action` from 5 to 7
- Bumped `dependabot/fetch-metadata` from 2 to 3

### Documentation
- Added `DESIGN_DOC.md` (arc42 software design document). Closes [#131](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/131)
- Added `SECURITY.md` with supported version matrix and vulnerability reporting policy. Closes [#129](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/129)

## [0.2.0] - 2026-09-23

### Added — AIOps Use Case Tools

#### CE v1.1.0 Integration — New Tools
- **`get-throttled-pods`**: Identifies pods with high CPU throttling using CFS-based metrics from CE ADR-020. Surfaces `cpu_throttle_rate` from `enriched_signals`; fallback to per-pattern `throttle_rate_pct` metadata. Configurable threshold (default 25%). Closes [#109](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/109)
- **`predict-disk-exhaustion`**: Forecasts filesystem full dates via CE ADR-018 `deriv()` analysis. Returns days-until-full, urgency classification (`critical`/`warning`/`info`/`stable`), projected full date, and daily fill rate per filesystem. Closes [#110](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/110)
- **`get-rightsizing-recommendations`**: Per-container CPU/memory right-sizing via CE ADR-019. Compares P95 usage against current requests/limits with 20%/50% headroom. Classifies containers as over-provisioned, under-provisioned, or right-sized. Configurable analysis window (7d/14d/30d/90d). Closes [#111](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/111)
- **`list-adrs`**: Fetches the CE ADR index from GitHub Contents API, parses the markdown table, and returns structured ADR metadata. Supports optional status filter. 5-minute cache TTL. Closes [#115](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/115)

#### CE v1.2.0 Integration — New Tools
- **`investigate-rca`**: Deep root-cause analysis correlating pod events, NetworkPolicy logs, and Istio traffic via CE `POST /api/v1/investigate/rca`. Returns root causes with confidence scores and correlated signals. Closes [#175](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/175)

#### Updated Tools
- **`analyze-anomalies`**: Now surfaces `enriched_signals` object when CE v1.1.0 returns application-level signals (ADR-017). Includes `cpu_throttle_rate_pct`, `http_error_rate_pct`, `http_response_time_p99_ms`, `throttling_detected`, `http_degraded`. Closes [#112](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/112)
- **`predict-resource-usage`**: Enriched with `capacity_forecast` block containing `forecasted_exhaustion_days` and `recommended_replica_increase` from CE `/api/v1/capacity/trends` (use case 5). Enrichment is best-effort; prediction still returned if CE endpoint unavailable. Closes [#113](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/113)
- **`trigger-remediation`**: OOMKill memory patching context added (CE v1.2.0). When `issue_type` is `oom_kill`, the response includes `OOMKillDetected` flag and `OOMKillAdvice` with memory patch guidance. Closes [#176](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/176)

#### Client Updates (`pkg/clients/coordination_engine.go`)
- Added `EnrichedSignals` struct mirroring CE ADR-017 response fields
- Added `CapacityTrendingResponse` struct with `ForecastedExhaustionDays` and `RecommendedReplicaIncrease`
- Added `GetCapacityTrends(ctx, namespace)` method calling `GET /api/v1/capacity/trends`
- Added RCA types (`RCARootCause`, `RCACorrelatorStat`, `RCATimeRange`, `RCAResponse`, `RCARequest`) and `InvestigateRCA()` method

### Documentation
- `CHANGELOG.md` v0.2.0 section (this entry). Closes [#114](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/114)
- `RELEASE.md` added with release runbook and branch protection references. Closes [#108](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/108)
- CE Version Compatibility section added to ADR-010 with tool-to-CE-version mapping table. Closes [#177](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/177)

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

[Unreleased]: https://github.com/KubeHeal/openshift-cluster-health-mcp/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/KubeHeal/openshift-cluster-health-mcp/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/KubeHeal/openshift-cluster-health-mcp/compare/v0.2.0...v1.1.0
[0.2.0]: https://github.com/KubeHeal/openshift-cluster-health-mcp/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/KubeHeal/openshift-cluster-health-mcp/releases/tag/v0.1.0
