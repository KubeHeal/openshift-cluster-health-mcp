# Multi-Version OpenShift Release Strategy

## Overview

This document describes the multi-version release strategy for the OpenShift Cluster
Health MCP Server, supporting a rolling 3-version window across OpenShift releases.

## Version Support Matrix

| Component | OCP 4.20 | OCP 4.21 | OCP 4.22 |
|-----------|----------|----------|----------|
| **Kubernetes** | 1.33 | 1.34 | 1.35 |
| **client-go** | v0.33.x | v0.34.x | v0.35.x |
| **Release Branch** | `release-4.20` | `release-4.21` | `release-4.22` |
| **Container Tag** | `ocp-4.20-*` | `ocp-4.21-*` | `ocp-4.22-*` |
| **Status** | Supported | Supported | Current |

## Branch Strategy

### Branch Roles

1. **main**: Primary development branch
   - All new features developed here
   - Auto-syncs to `release-4.22` (current version) via GitHub Actions
   - No direct container builds from main

2. **release-4.22** (Current Release)
   - Matches latest stable OpenShift (4.22)
   - Auto-synced from `main` via `.github/workflows/sync-release-branch.yml`
   - Triggers container builds: `ocp-4.22-latest`, `ocp-4.22-<sha>`

3. **release-4.21** (Stable)
   - Cherry-pick bug fixes from `main`
   - Manual updates for critical security patches
   - Triggers container builds: `ocp-4.21-latest`, `ocp-4.21-<sha>`

4. **release-4.20** (Legacy Support)
   - Critical bug fixes only
   - Security patches only
   - Triggers container builds: `ocp-4.20-latest`, `ocp-4.20-<sha>`

### Automated Sync Workflow

**Trigger**: Every push to `main`

**Action**: Automatically merge `main` into `release-4.22`

**Conflict Resolution**:
- If merge succeeds: Changes pushed automatically
- If conflicts occur: PR created for manual resolution

## Container Image Strategy

### Image Naming Convention

```
quay.io/takinosh/openshift-cluster-health-mcp:<tag>
```

**Tags**:
- `ocp-4.20-latest`: Latest build for OCP 4.20 (from `release-4.20`)
- `ocp-4.20-<sha>`: Specific commit SHA on `release-4.20`
- `ocp-4.21-latest`: Latest build for OCP 4.21 (from `release-4.21`)
- `ocp-4.22-latest`: Latest build for OCP 4.22 (from `release-4.22`)

### Build Triggers

**Workflow**: `.github/workflows/release-quay.yml`

**Trigger Branches**: `release-4.20`, `release-4.21`, `release-4.22` (NOT `main`)

**Build Process**:
1. Extract OpenShift version from branch name
2. Run tests
3. Build container image (linux/amd64)
4. Tag with both `-latest` and `-<sha>`
5. Push to Quay.io registry
6. Scan with Trivy
7. Generate deployment summary

## Maintenance Procedures

### Adding Support for New OpenShift Version (e.g., 4.23)

See [docs/playbooks/add-new-version.md](playbooks/add-new-version.md) for the full playbook.

Summary:
1. Create `release-4.23` branch from `main` with bumped k8s.io deps
2. Update all workflow files to include `release-4.23`
3. Create `values-ocp-4.23.yaml` Helm overlay
4. Update Chart.yaml annotations
5. Update documentation
6. Archive old version (4.20)

### Cherry-Picking Bug Fixes to Older Versions

See [docs/playbooks/cherry-pick-fixes.md](playbooks/cherry-pick-fixes.md) for the full playbook.

## Deployment Guidelines

### Production Deployment

Always match container image version to OpenShift cluster version:

```bash
# Check cluster version
oc version

# Deploy matching image
helm install mcp-server ./charts/openshift-cluster-health-mcp \
  -f ./charts/openshift-cluster-health-mcp/values-ocp-4.22.yaml \
  --namespace self-healing-platform
```

### Upgrading Deployments (OCP 4.21 to 4.22)

```bash
helm upgrade mcp-server ./charts/openshift-cluster-health-mcp \
  -f ./charts/openshift-cluster-health-mcp/values-ocp-4.22.yaml \
  --namespace self-healing-platform
```

### Rollback

```bash
helm rollback mcp-server -n self-healing-platform
```

## E2E Testing

### Testing Tiers

| Tier | Environment | Trigger | Required |
|------|-------------|---------|----------|
| Tier 0 | Unit tests | Every PR | Yes |
| Tier 1 | Kind cluster | Every PR | Yes |
| Tier 2 | Live OpenShift/ROSA | Release branches, `e2e-test` label | Major releases only |

### E2E Gating Policy

- **Major releases**: Tier 2 E2E must pass before release
- **Minor/patch releases**: Tier 2 is optional (skip if cluster unavailable)
- **If no ROSA cluster configured**: Tier 2 skips cleanly, never blocks CI

See [scripts/setup-e2e.sh](../scripts/setup-e2e.sh) for local E2E setup.

## References

- [Kubernetes API Deprecation Guide](https://kubernetes.io/docs/reference/using-api/deprecation-guide/)
- [client-go Compatibility Matrix](https://github.com/kubernetes/client-go#compatibility-matrix)
- [OpenShift Release Schedule](https://access.redhat.com/support/policy/updates/openshift)
