# ADR-010: Version Compatibility and Upgrade Roadmap

## Status

**IMPLEMENTED** - 2026-01-06

**Original**: ACCEPTED - 2025-12-09

**Amendment 1**: Changed from single-branch N-2 compatibility strategy to multi-branch approach with dedicated branches for each OpenShift version (main=4.18, release/4.19, release/4.20). This provides cleaner dependency management for Kubernetes client libraries.

**Amendment 2** (2026-09-23): Extended to cover OpenShift 4.21 and 4.22 (latest GA). Added release-4.21 and release-4.22 branches to the compatibility matrix. Updated Kubernetes version mapping through 4.22. Acknowledged that all branches currently share k8s.io v0.33.x — per-branch pinning is a tracked gap (see issue #125).

## Context

The OpenShift Cluster Health MCP Server must be compatible with the current OpenShift cluster environment while planning for future upgrades. Version compatibility affects:

1. **Kubernetes API compatibility**: client-go version must match K8s API server
2. **Operator compatibility**: Integration with OpenShift AI, GitOps, Pipelines
3. **Go version requirements**: Language features and standard library
4. **Container runtime**: CRI-O and image compatibility
5. **Security features**: Pod Security Standards, SCC evolution

### Current Environment (Baseline)

**Cluster Information** (as of 2025-12-09):

| Component | Version | Details |
|-----------|---------|---------|
| **OpenShift** | 4.18.21 | Channel: stable-4.18 |
| **Kubernetes** | v1.31.10 | API server version |
| **RHEL CoreOS** | 418.94.202507221927-0 | Node operating system |
| **CRI-O** | 1.31.10-4.rhaos4.18 | Container runtime |
| **Kernel** | 5.14.0-427.79.1.el9_4.x86_64 | Linux kernel |

**Key Installed Operators**:

| Operator | Version | Purpose |
|----------|---------|---------|
| **OpenShift AI (RHODS)** | 2.22.3 | ML/AI workloads, KServe |
| **GPU Operator** | (NVIDIA certified) | GPU node management |
| **OpenShift GitOps** | 1.15.4 | ArgoCD for deployments |
| **OpenShift Pipelines** | 1.17.2 | Tekton CI/CD |
| **Serverless** | 1.37.0 | Knative for KServe |
| **Service Mesh** | 2.6.11 | Istio for networking |
| **Cert Manager** | 1.18.0 | Certificate automation |
| **Validated Patterns** | 0.0.63 | Deployment framework |

### Version Constraints

**OpenShift Version Lifecycle** (updated 2026-09-23):
- **4.18 Support**: End of Life — security fixes only
- **4.19**: General Availability (GA)
- **4.20**: General Availability (GA)
- **4.21**: General Availability (GA)
- **4.22**: Latest stable release (GA) — current target

**Kubernetes Version Mapping** (verified from official Red Hat sources):
- OpenShift 4.18 → Kubernetes 1.31
- OpenShift 4.19 → Kubernetes 1.32
- OpenShift 4.20 → Kubernetes 1.33
- OpenShift 4.21 → Kubernetes 1.34
- OpenShift 4.22 → Kubernetes 1.35

### Current Challenges

1. **Not on Latest OpenShift**: 4.18.21 vs 4.20 (2 minor versions behind)
2. **Operator Version Drift**: Some operators may be outdated
3. **Security Updates**: Newer versions include security patches
4. **Feature Gap**: Missing features from 4.19, 4.20
5. **Support Window**: 4.18 support lifecycle considerations

## Decision

We will adopt a **multi-branch version strategy** to manage compatibility with different OpenShift versions:

### Branch Strategy

**Rationale**: Kubernetes client-go libraries have strict version coupling with K8s API versions. Using separate branches allows:
- Clean dependency management per OpenShift version
- Targeted Dependabot updates per branch
- Clear separation of concerns for each OpenShift release
- Easier testing and validation per version

**Branch Structure** (updated 2026-09-23):

| Branch | OpenShift Version | Kubernetes Version | client-go Version | Status |
|--------|------------------|-------------------|------------------|--------|
| **main** | 4.18.x | v1.31.x | v0.31.x (target) | Legacy base ⚠️ |
| **release-4.18** | 4.18.x | v1.31.x | v0.31.x (target) | EOL — security fixes only |
| **release-4.19** | 4.19.x | v1.32.x | v0.32.x (target) | Active ✅ |
| **release-4.20** | 4.20.x | v1.33.x | v0.33.x (target) | Active ✅ |
| **release-4.21** | 4.21.x | v1.34.x | v0.34.x (target) | Active ✅ |
| **release-4.22** | 4.22.x | v1.35.x | v0.35.x (target) | Planned (see #121) |

> **Note (2026-09-23)**: All branches currently use k8s.io v0.33.x. Per-branch version pinning has not been implemented. Issue #125 tracks whether to implement pinning or amend this strategy.

**Branch Lifecycle**:
```
main (4.18) ─→ release-4.18 (EOL)
            ├─→ release-4.19 ─→ Active
            ├─→ release-4.20 ─→ Active
            ├─→ release-4.21 ─→ Active
            └─→ release-4.22 ─→ Planned (#121)
```

### Compatibility Matrix

| Branch | Min OpenShift | Kubernetes | Go Version | client-go | k8s.io/api | k8s.io/apimachinery |
|--------|---------------|------------|------------|-----------|------------|---------------------|
| **main** | 4.18.21+ | v1.31.10+ | 1.24+ | v0.31.x | v0.31.x | v0.31.x |
| **release-4.18** | 4.18.21+ | v1.31.x | 1.24+ | v0.31.x | v0.31.x | v0.31.x |
| **release-4.19** | 4.19.0+ | v1.32.x | 1.24+ | v0.32.x | v0.32.x | v0.32.x |
| **release-4.20** | 4.20.0+ | v1.33.x | 1.24+ | v0.33.x | v0.33.x | v0.33.x |
| **release-4.21** | 4.21.0+ | v1.34.x | 1.24+ | v0.34.x | v0.34.x | v0.34.x |
| **release-4.22** | 4.22.0+ | v1.35.x | 1.24+ | v0.35.x | v0.35.x | v0.35.x |

**Compatibility Policy**:
- Each branch targets a specific OpenShift major.minor version
- Patch updates (4.18.21 → 4.18.22) happen within the same branch
- Minor upgrades (4.18 → 4.19) require branch migration

### Dependabot Configuration

Configure Dependabot to target each branch with appropriate version constraints:

```yaml
# .github/dependabot.yml
version: 2
updates:
  # Main branch - OpenShift 4.18 (Kubernetes 1.31)
  - package-ecosystem: "gomod"
    directory: "/"
    target-branch: "main"
    schedule:
      interval: "weekly"
    ignore:
      # Pin k8s.io dependencies to v0.31.x for OpenShift 4.18
      - dependency-name: "k8s.io/api"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.32.0"]
      - dependency-name: "k8s.io/apimachinery"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.32.0"]
      - dependency-name: "k8s.io/client-go"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.32.0"]
    labels:
      - "dependencies"
      - "openshift-4.18"

  # Release/4.19 branch - OpenShift 4.19 (Kubernetes 1.32)
  - package-ecosystem: "gomod"
    directory: "/"
    target-branch: "release/4.19"
    schedule:
      interval: "weekly"
    ignore:
      # Pin k8s.io dependencies to v0.32.x for OpenShift 4.19
      - dependency-name: "k8s.io/api"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.33.0"]
      - dependency-name: "k8s.io/apimachinery"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.33.0"]
      - dependency-name: "k8s.io/client-go"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.33.0"]
    labels:
      - "dependencies"
      - "openshift-4.19"

  # Release/4.20 branch - OpenShift 4.20 (Kubernetes 1.33)
  - package-ecosystem: "gomod"
    directory: "/"
    target-branch: "release/4.20"
    schedule:
      interval: "weekly"
    ignore:
      # Pin k8s.io dependencies to v0.33.x for OpenShift 4.20
      - dependency-name: "k8s.io/api"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.34.0"]
      - dependency-name: "k8s.io/apimachinery"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.34.0"]
      - dependency-name: "k8s.io/client-go"
        update-types: ["version-update:semver-minor"]
        versions: [">= 0.34.0"]
    labels:
      - "dependencies"
      - "openshift-4.20"

  # GitHub Actions (applies to all branches)
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "github-actions"

  # Docker (applies to all branches)
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "docker"
```

### Branch Migration Workflow

**When upgrading from 4.18 → 4.19**:

```bash
# 1. Create release/4.19 branch from main
git checkout main
git pull origin main
git checkout -b release/4.19

# 2. Update K8s dependencies to v0.32.x
go get k8s.io/api@v0.32.0
go get k8s.io/apimachinery@v0.32.0
go get k8s.io/client-go@v0.32.0
go mod tidy

# 3. Update documentation
sed -i 's/OpenShift 4.18/OpenShift 4.19/g' README.md
sed -i 's/Kubernetes 1.31/Kubernetes 1.32/g' README.md

# 4. Test compatibility
make build
make test
make test-integration

# 5. Push branch
git add .
git commit -m "feat: create release/4.19 branch for OpenShift 4.19 support

- Update k8s.io dependencies to v0.32.x for Kubernetes 1.32
- Update documentation for OpenShift 4.19
- Verify all tests pass"

git push origin release/4.19

# 6. Update main branch to continue 4.18 development
git checkout main
# main remains on k8s.io/* v0.31.x
```

**Feature Backporting**:
- Security fixes: Backport to all active branches
- Bug fixes: Backport to current and previous release
- New features: Only in latest branch (main or newest release branch)

**Example Backport**:
```bash
# Fix merged in release/4.19
git checkout release/4.19
git pull origin release/4.19

# Cherry-pick to main (4.18)
git checkout main
git cherry-pick <commit-sha>
# Resolve conflicts if any (especially in go.mod)
git push origin main
```

## Upgrade Roadmap

### Phase 1: OpenShift 4.18.21 → 4.19 (Months 0-6)

**Timeline**: Q1-Q2 2025

**Pre-Upgrade Assessment**:
```bash
# Check cluster upgrade readiness
oc adm upgrade

# Review operator compatibility
oc get csv -A

# Check deprecated APIs
oc get apirequestcounts -o json | jq '.items[] | select(.status.removedInRelease != null)'

# Verify storage health
oc get pv,pvc -A

# Check node health
oc get nodes
oc adm top nodes
```

**Upgrade Steps**:

1. **Update Operator Subscriptions** (Week 1)
   ```bash
   # Update OpenShift AI operator
   oc patch subscription rhods-operator \
     -n redhat-ods-operator \
     --type merge \
     -p '{"spec":{"channel":"stable-2.24"}}'

   # Update GitOps
   oc patch subscription openshift-gitops-operator \
     -n openshift-operators \
     --type merge \
     -p '{"spec":{"channel":"latest"}}'

   # Update Pipelines
   oc patch subscription openshift-pipelines-operator-rh \
     -n openshift-operators \
     --type merge \
     -p '{"spec":{"channel":"latest"}}'
   ```

2. **Test MCP Server Compatibility** (Week 2)
   ```bash
   # Deploy MCP server to dev cluster running 4.19
   helm install cluster-health-mcp ./charts \
     --namespace self-healing-platform-dev \
     --create-namespace

   # Run integration tests
   make test-integration

   # Verify all MCP tools working
   # - get-cluster-health
   # - list-pods
   # - analyze-anomalies
   # - trigger-remediation
   ```

3. **Upgrade Cluster** (Week 3-4)
   ```bash
   # Pause machine config pools
   oc patch mcp/worker --type merge -p '{"spec":{"paused":true}}'

   # Start upgrade to 4.19
   oc adm upgrade --to=4.19.x

   # Monitor upgrade progress
   watch oc get clusterversion

   # Resume machine config pools (one at a time)
   oc patch mcp/worker --type merge -p '{"spec":{"paused":false}}'
   ```

4. **Post-Upgrade Validation** (Week 4)
   ```bash
   # Verify cluster health
   oc get clusteroperators
   oc get nodes

   # Test MCP server
   curl http://cluster-health-mcp:8080/health

   # Run E2E tests
   make test-e2e

   # Verify OpenShift Lightspeed integration
   ```

**Expected Changes in 4.19**:
- Kubernetes 1.32 features
- Updated security policies (Pod Security Admission)
- Enhanced observability (updated Prometheus)
- Improved OpenShift AI integration

**Risks and Mitigations**:

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **API deprecations** | Medium | High | Test on dev cluster first, review deprecation notices |
| **Operator incompatibility** | Low | Medium | Update operators before cluster upgrade |
| **MCP server breakage** | Low | High | Maintain client-go v0.31 compatibility, test thoroughly |
| **Coordination Engine issues** | Medium | Medium | Python Kubernetes client compatibility check |
| **Downtime during upgrade** | Low | Medium | Schedule maintenance window, use rolling updates |

---

### Phase 2: OpenShift 4.19 → 4.20 (Months 6-12)

**Timeline**: Q3-Q4 2025

**Why Upgrade to 4.20?**

1. **Latest Stable Release**: Current production version as of Dec 2024
2. **Extended Support**: Longer support lifecycle
3. **Security Patches**: Latest CVE fixes
4. **Performance Improvements**: Optimized Kubernetes 1.32
5. **Feature Enhancements**:
   - Enhanced Pod Security
   - Improved observability
   - Better GPU scheduling

**Key 4.20 Features Relevant to MCP Server**:

| Feature | Benefit | Impact on MCP Server |
|---------|---------|---------------------|
| **Kubernetes 1.33** | Latest K8s APIs | **Update client-go to v0.33.x** (major upgrade) |
| **CRI-O 1.33** | Updated container runtime | No impact (transparent) |
| **Pod Security Standards v1.33** | Enhanced security | Review security contexts (ADR-007) |
| **Prometheus 2.48+** | Better metrics | Enhanced Prometheus client features |
| **OpenShift AI 2.26+** | Improved KServe | Better model serving integration |
| **Enhanced RBAC** | Finer-grained permissions | Review ClusterRole definitions |

**Upgrade Process**:

1. **Update MCP Server Dependencies** (Week 1-2)
   ```go
   // go.mod
   require (
       k8s.io/client-go v0.33.0  // Updated from v0.32.x (Kubernetes 1.33)
       k8s.io/api v0.33.0
       k8s.io/apimachinery v0.33.0
   )
   ```

2. **Test Compatibility** (Week 3-4)
   ```bash
   # Run against 4.20 dev cluster
   make test-integration CLUSTER_VERSION=4.20

   # Verify no deprecation warnings
   oc logs -n self-healing-platform cluster-health-mcp-xxx | grep -i deprecated
   ```

3. **Production Upgrade** (Week 5-8)
   - Similar process to 4.19 upgrade
   - Monitor for API changes
   - Validate all integrations

**Expected MCP Server Changes**:
- **Update client-go to v0.33.x** (Kubernetes 1.33)
- Review Kubernetes 1.33 API changes
- Test against new CRI-O 1.33 runtime
- Update Helm chart for 4.20 features
- Verify backward compatibility with 4.19 (K8s 1.32)

---

### Phase 3: OpenShift 4.20 → 4.21 (added 2026-09-23)

**Timeline**: Q1-Q2 2026

**Status**: Branch `release-4.21` created. k8s.io deps not yet aligned to v0.34.x (tracked by #125).

**Key Changes**:
- Kubernetes 1.34 features
- Update client-go to v0.34.x for Kubernetes 1.34
- MCP Go SDK upgrade to v1.6.0+ (tracked by #124)
- Go 1.26 upgrade assessment (tracked by #128)

**MCP Server Dependencies** (target):
```go
// go.mod for release-4.21
require (
    k8s.io/client-go v0.34.x
    k8s.io/api v0.34.x
    k8s.io/apimachinery v0.34.x
    github.com/modelcontextprotocol/go-sdk v1.6.0+
)
```

---

### Phase 4: OpenShift 4.21 → 4.22 (added 2026-09-23)

**Timeline**: Q3-Q4 2026 (v1.1.0 milestone, due 2026-10-31)

**Status**: Branch `release-4.22` not yet created (tracked by #121).

**Key Changes**:
- Kubernetes 1.35 features
- Update client-go to v0.35.x for Kubernetes 1.35
- OpenShift 4.22 is the latest GA release as of Sep 2026
- Drop OCP 4.18 support (EOL)

**MCP Server Dependencies** (target):
```go
// go.mod for release-4.22
require (
    k8s.io/client-go v0.35.x
    k8s.io/api v0.35.x
    k8s.io/apimachinery v0.35.x
    github.com/modelcontextprotocol/go-sdk v1.6.0+
)
```

**Tracking Issues**:
- #121: Add OCP 4.22 support (release branch, chart, CI)
- #123: Update this ADR for 4.21/4.22
- #124: MCP Go SDK v1.6.0 impact assessment
- #125: k8s.io version alignment across branches
- #127: Dependabot config for release-4.21 and release-4.22
- #128: Go 1.26 upgrade assessment

---

### Phase 5: Maintain N-2 Compatibility (Ongoing, updated 2026-09-23)

**Timeline**: Ongoing

**Compatibility Policy** (updated for OCP 4.22 GA):

```
Current Release:   4.22 ✅ Fully supported
Previous Release:  4.21 ✅ Supported
N-1 Release:       4.20 ✅ Supported
N-2 Release:       4.19 ⚠️  Best effort (deprecated)
Older:            4.18- ❌ EOL / Not supported
```

**Testing Strategy**:

```yaml
# .github/workflows/compatibility-test.yml
name: Multi-Version Compatibility

on: [push, pull_request]

jobs:
  test-matrix:
    strategy:
      matrix:
        openshift: ['4.20', '4.21', '4.22']
        include:
          - openshift: '4.20'
            kubernetes: '1.33'
            client-go: 'v0.33.x'
          - openshift: '4.21'
            kubernetes: '1.34'
            client-go: 'v0.34.x'
          - openshift: '4.22'
            kubernetes: '1.35'
            client-go: 'v0.35.x'

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run tests against ${{ matrix.openshift }}
        run: |
          make test-integration \
            OPENSHIFT_VERSION=${{ matrix.openshift }} \
            K8S_VERSION=${{ matrix.kubernetes }}
```

**Version Documentation**:

```markdown
# README.md - Compatibility Section

## Supported Versions

| OpenShift | Kubernetes | Status | Support Level |
|-----------|------------|--------|---------------|
| 4.22      | 1.35       | ✅ Supported | Full support (latest) |
| 4.21      | 1.34       | ✅ Supported | Full support |
| 4.20      | 1.33       | ✅ Supported | Full support |
| 4.19      | 1.32       | ⚠️  Deprecated | Best effort |
| 4.18-     | 1.31-      | ❌ EOL | No support |

## Minimum Requirements
- **OpenShift**: 4.20+
- **Kubernetes**: 1.33+
- **Go**: 1.24+
- **Helm**: 3.12+
```

## Go Version Strategy

### Current: Go 1.24 (updated 2026-09-23)

**Rationale**:
- Kubernetes 1.33+ supports Go 1.24
- MCP Go SDK v1.2.0 requires Go 1.22+
- Stable release with good performance

**Dependencies** (as of 2026-09-23):
```go
// go.mod
module github.com/KubeHeal/openshift-cluster-health-mcp

go 1.24.0
toolchain go1.24.11

require (
    github.com/modelcontextprotocol/go-sdk v1.2.0
    k8s.io/client-go v0.33.7
    k8s.io/api v0.33.7
    k8s.io/apimachinery v0.33.7
)
```

### Target: Go 1.26 (tracked by #128)

**When**: v1.1.0 milestone (OCP 4.22 readiness)

**Assessment needed**:
- Go 1.25 and 1.26 release notes review
- k8s.io and MCP Go SDK compatibility verification
- Dependabot PR #73 (base image bump)

## client-go Version Strategy (updated 2026-09-23)

### Current: v0.33.x (all branches)

**Matches**: Kubernetes 1.33 (OpenShift 4.20)

**Key APIs Used**:
- `k8s.io/client-go/kubernetes`: Core K8s client
- `k8s.io/client-go/rest`: REST client config
- `k8s.io/client-go/tools/clientcmd`: Kubeconfig handling
- `k8s.io/apimachinery/pkg/apis/meta/v1`: Common types

> **Gap**: All branches use v0.33.x despite targeting different OCP versions. Issue #125 tracks alignment.

### Target: v0.34.x (release-4.21)

**Matches**: Kubernetes 1.34 (OpenShift 4.21)

### Target: v0.35.x (release-4.22)

**Matches**: Kubernetes 1.35 (OpenShift 4.22)

**Migration Plan** (per branch):
1. Update go.mod dependencies to target version
2. Run compatibility tests
3. Review API deprecations for target K8s version
4. Update type assertions if needed
5. Test against target OCP version cluster

**Breaking Changes Check**:
```bash
# Check for deprecated APIs
go list -m -u all | grep k8s.io

# Update dependencies for 4.21 (K8s 1.34)
go get k8s.io/client-go@v0.34.0
go get k8s.io/api@v0.34.0
go get k8s.io/apimachinery@v0.34.0

# Update dependencies for 4.22 (K8s 1.35)
go get k8s.io/client-go@v0.35.0
go get k8s.io/api@v0.35.0
go get k8s.io/apimachinery@v0.35.0

# Test
make test
```

## Operator Version Tracking

### Critical Dependencies

**OpenShift AI (RHODS)**:
- **Current**: 2.22.3
- **Target (6 months)**: 2.24+
- **Target (12 months)**: 2.26+
- **Impact**: KServe API compatibility, model serving features

**OpenShift GitOps**:
- **Current**: 1.15.4
- **Target**: Latest stable
- **Impact**: ArgoCD integration (deployment automation)

**OpenShift Pipelines**:
- **Current**: 1.17.2
- **Target**: Latest stable
- **Impact**: Tekton validation pipelines (ADR-021)

**GPU Operator**:
- **Current**: NVIDIA certified for 4.18
- **Target**: Update with OpenShift version
- **Impact**: GPU node scheduling for ML workloads

### Operator Update Strategy

```bash
# Check current operator versions
oc get csv -A | grep -E "(rhods|gitops|pipelines|gpu)"

# Update to latest in channel
oc patch subscription <operator-name> \
  -n <namespace> \
  --type merge \
  -p '{"spec":{"installPlanApproval":"Automatic"}}'

# Monitor upgrade
oc get installplan -n <namespace>
```

## Deprecation and Migration Plan

### API Deprecations to Watch

**Kubernetes 1.32 Removals** (affects OpenShift 4.19+):
- `flowcontrol.apiserver.k8s.io/v1beta2` → v1
- `autoscaling/v2beta2` → v2

**Check for Usage**:
```bash
# Scan codebase for deprecated APIs
grep -r "flowcontrol.apiserver.k8s.io/v1beta2" .
grep -r "autoscaling/v2beta2" .

# Check live cluster
oc get apirequestcounts -o json | \
  jq '.items[] | select(.status.removedInRelease != null) |
      {name: .metadata.name, removedIn: .status.removedInRelease}'
```

**MCP Server Impact**: ✅ Low (we don't use these APIs directly)

### Security Context Constraints (SCC)

**OpenShift 4.18**: restricted-v2 SCC (current)
**OpenShift 4.19+**: Enhanced Pod Security Admission

**Action Required**:
- Review ADR-007 (RBAC-Based Security Model)
- Ensure SecurityContext compliance
- Test with updated SCC policies

```yaml
# Pod Security Standards Label (4.19+)
apiVersion: v1
kind: Namespace
metadata:
  name: self-healing-platform
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## Version Testing Matrix

### CI/CD Integration

```yaml
# .github/workflows/version-matrix.yml
name: Version Compatibility Matrix

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly
  workflow_dispatch:

jobs:
  compatibility-test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        openshift: ['4.18', '4.19', '4.20']
        test-type: ['unit', 'integration', 'e2e']

    steps:
      - name: Test against OpenShift ${{ matrix.openshift }}
        run: |
          make test-${{ matrix.test-type }} \
            OPENSHIFT_VERSION=${{ matrix.openshift }}

      - name: Report results
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: 'Compatibility failure: OpenShift ${{ matrix.openshift }}',
              body: 'Test type: ${{ matrix.test-type }}\nSee workflow run for details.',
              labels: ['compatibility', 'ci-failure']
            })
```

### Manual Testing Checklist

Before each OpenShift upgrade:

- [ ] Update dev/staging cluster to target version
- [ ] Deploy MCP server to updated cluster
- [ ] Verify all MCP tools working:
  - [ ] `get-cluster-health`
  - [ ] `list-pods`
  - [ ] `analyze-anomalies`
  - [ ] `trigger-remediation`
  - [ ] `get-model-status`
- [ ] Test OpenShift Lightspeed integration
- [ ] Test Claude Desktop integration (stdio)
- [ ] Verify Prometheus queries
- [ ] Verify Coordination Engine integration
- [ ] Verify KServe model predictions
- [ ] Check RBAC permissions (no denied requests)
- [ ] Review pod logs for deprecation warnings
- [ ] Run performance benchmarks (<100ms p95)
- [ ] Security scan (Trivy, gosec)
- [ ] Load test (200 req/min for 1 hour)

## Version Announcement Strategy

### Release Notes Template

```markdown
# OpenShift Cluster Health MCP Server v0.2.0

## Compatibility

- **Minimum OpenShift**: 4.19
- **Minimum Kubernetes**: 1.32
- **Tested on**: OpenShift 4.19.5, 4.20.1
- **Go Version**: 1.22
- **client-go**: v0.32.0

## What's New

- Updated Kubernetes API client for 1.32 compatibility
- Enhanced Pod Security Admission support
- Improved Prometheus integration (Prometheus 2.48+)
- Updated OpenShift AI integration (RHODS 2.24+)

## Breaking Changes

- **Minimum OpenShift version**: Now 4.19+ (was 4.18+)
- **Deprecated**: OpenShift 4.18 support (security fixes only)

## Upgrade Instructions

1. Upgrade OpenShift cluster to 4.19+
2. Update MCP server Helm chart:
   ```bash
   helm upgrade cluster-health-mcp ./charts \
     --version 0.2.0
   ```
3. Verify compatibility:
   ```bash
   make test-integration
   ```

## Known Issues

- OpenShift 4.18: Best effort support only
- Some Prometheus metrics renamed in 2.48+

## Deprecation Notices

- OpenShift 4.17 and earlier: No longer supported
- Go 1.21: Will be deprecated in v0.3.0 (6 months)
```

## Success Criteria

### Phase 1 Success (OpenShift 4.18 → 4.19)
- ✅ Cluster upgraded to 4.19 without issues
- ✅ All operators updated and healthy
- ✅ MCP server tests passing on 4.19
- ✅ No deprecated API warnings
- ✅ Performance maintained (<100ms p95)

### Phase 2 Success (OpenShift 4.19 → 4.20)
- ✅ Cluster on latest stable (4.20)
- ✅ client-go updated to v0.32.x
- ✅ All integrations working
- ✅ N-2 compatibility maintained (4.18, 4.19, 4.20)

### Phase 3 Success (OpenShift 4.20 → 4.21)
- ⬜ release-4.21 branch created (done)
- ⬜ client-go updated to v0.34.x on release-4.21
- ⬜ MCP Go SDK updated to v1.6.0+ (#124)
- ⬜ All integrations working on 4.21

### Phase 4 Success (OpenShift 4.21 → 4.22)
- ⬜ release-4.22 branch created (#121)
- ⬜ client-go updated to v0.35.x on release-4.22
- ⬜ Helm chart updated for 4.22
- ⬜ CI matrix covers 4.20, 4.21, 4.22

### Phase 5 Success (Ongoing)
- ⬜ CI/CD testing across 3 supported versions (4.20, 4.21, 4.22)
- ⬜ Clear version documentation
- ⬜ Regular upgrade cadence (every 6 months)
- ⬜ Zero breaking changes for supported versions

## Coordination Engine Version Compatibility

The MCP server integrates with the Coordination Engine (CE) and different tools depend on different CE versions. When the CE is at an older version, tools requiring newer features will return informative errors; the server degrades gracefully.

The CE exposes its version via `GET /api/v1/health` — the `version` field in the response can be used for runtime version detection.

| MCP Tool | Min CE Version | CE Endpoint | Notes |
|----------|---------------|-------------|-------|
| `list-incidents` | v1.0.0 | `GET /api/v1/incidents` | Core incident listing |
| `trigger-remediation` | v1.0.0 | `POST /api/v1/remediation/trigger` | v1.2.0 adds OOMKill memory patching |
| `analyze-anomalies` (enriched signals) | v1.1.0 | `POST /api/v1/anomalies/analyze` | `enriched_signals` field added in v1.1.0 |
| `get-throttled-pods` | v1.1.0 | `POST /api/v1/anomalies/analyze` | Uses `cpu_throttle_rate` from enriched signals (ADR-020) |
| `predict-disk-exhaustion` | v1.1.0 | `GET /api/v1/predict/disk-exhaustion` | ADR-018 |
| `get-rightsizing-recommendations` | v1.1.0 | `GET /api/v1/recommendations/rightsizing` | ADR-019 |
| `predict-resource-usage` (capacity forecast) | v1.1.0 | `GET /api/v1/capacity/trends` | Forecasted exhaustion days |
| `investigate-rca` | v1.2.0 | `POST /api/v1/investigate/rca` | Deep RCA v2 — ADR-021 |
| `trigger-remediation` (OOMKill patching) | v1.2.0 | `POST /api/v1/remediation/trigger` | Memory limit patching for OOMKilled pods |

### Graceful Degradation

- **CE unavailable**: All CE-dependent tools return a clear error ("Coordination Engine not enabled"). Core tools (`get-cluster-health`, `list-pods`, `list-adrs`) continue to function.
- **CE v1.0.x with v1.1.0 tools**: The CE will return HTTP 404 for unknown endpoints. The MCP tool surfaces this as "endpoint not available — CE version may not support this feature".
- **CE v1.1.x with v1.2.0 tools**: Same degradation — `investigate-rca` returns an error pointing to the CE version requirement.

## Related ADRs

- [ADR-001: Go Language Selection](001-go-language-selection.md)
- [ADR-006: Integration Architecture](006-integration-architecture.md)
- [ADR-007: RBAC-Based Security Model](007-rbac-based-security-model.md)
- [ADR-009: Architecture Evolution Roadmap](009-architecture-evolution-roadmap.md)

## References

- [OpenShift 4.18 Release Notes](https://docs.openshift.com/container-platform/4.18/release_notes/ocp-4-18-release-notes.html)
- [OpenShift 4.19 Release Notes](https://docs.okd.io/4.19/updating/preparing_for_updates/updating-cluster-prepare.html) - Uses Kubernetes 1.32
- [OpenShift 4.20 Release](https://www.redhat.com/en/blog/red-hat-openshift-42-what-you-need-to-know) - Uses Kubernetes 1.33
- [What's New in OpenShift 4.20](https://developers.redhat.com/articles/2025/11/11/whats-new-developers-red-hat-openshift-4-20)
- [OpenShift/Kubernetes Version Mapping](https://gist.github.com/jeyaramashok/ebbd25f36338de4422fd584fea841c08)
- [Red Hat Customer Portal - Kubernetes API Versions](https://access.redhat.com/solutions/4870701)
- [Kubernetes 1.31 Release Notes](https://kubernetes.io/blog/2024/08/13/kubernetes-v1-31-release/)
- [Kubernetes 1.32 Release Notes](https://kubernetes.io/blog/2024/12/11/kubernetes-v1-32-release/)
- [client-go Compatibility Matrix](https://github.com/kubernetes/client-go#compatibility-matrix)

## Approval

- **Architect**: Approved
- **Platform Team**: Approved
- **Date**: 2025-12-09
