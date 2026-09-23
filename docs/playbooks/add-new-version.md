# Playbook: Adding New OpenShift Version

## Overview

This playbook guides you through adding support for a new OpenShift version
(e.g., 4.23) when it is released.

## Prerequisites

- [ ] New OpenShift version released
- [ ] Kubernetes version mapping confirmed (e.g., 4.23 = k8s 1.36)
- [ ] client-go version available (e.g., v0.36.0)
- [ ] Write access to the repository

## Steps

### 1. Create New Release Branch

```bash
git checkout main
git pull origin main
git checkout -b release-4.23

go get k8s.io/client-go@v0.36.0
go get k8s.io/api@v0.36.0
go get k8s.io/apimachinery@v0.36.0
go mod tidy

git add go.mod go.sum
git commit -s -m "chore: initialize release-4.23 for OpenShift 4.23 (k8s 1.36)"
git push -u origin release-4.23
```

### 2. Update GitHub Actions Workflows

#### 2.1 Update CI Workflow

Edit `.github/workflows/ci.yml`:

```yaml
on:
  push:
    branches: [ main, release-4.21, release-4.22, release-4.23 ]
  pull_request:
    branches: [ main, release-4.21, release-4.22, release-4.23 ]
```

Add Kind cluster version mapping:

```yaml
release-4.23)
  KIND_IMAGE="kindest/node:v1.36.0"
  ;;
```

#### 2.2 Update Release Workflow

Edit `.github/workflows/release-quay.yml`:

```yaml
on:
  push:
    branches:
      - 'release-4.21'
      - 'release-4.22'
      - 'release-4.23'
```

#### 2.3 Update Container Workflow

Edit `.github/workflows/container.yml`:
- Add `release-4.23` to push/PR branches
- Add `release-4.23` case in OCP version map

#### 2.4 Update Sync Workflow

Edit `.github/workflows/sync-release-branch.yml`:
- Change default sync target from `release-4.22` to `release-4.23`

#### 2.5 Update E2E Workflow

Edit `.github/workflows/e2e-openshift.yml`:
- Add `values-ocp-4.23.yaml` to the values file choices

### 3. Create Helm Values Overlay

```bash
cp charts/openshift-cluster-health-mcp/values-ocp-4.22.yaml \
   charts/openshift-cluster-health-mcp/values-ocp-4.23.yaml
```

Edit the new file:
- `image.tag: "ocp-4.23-latest"`
- `openshiftVersion.target: "4.23"`
- `openshiftVersion.kubernetes: "1.36"`

### 4. Update Helm Chart Metadata

Edit `charts/openshift-cluster-health-mcp/Chart.yaml`:

```yaml
annotations:
  openshift.io/supported-versions: "4.21,4.22,4.23"
  kubernetes.io/supported-versions: "1.34,1.35,1.36"
```

### 5. Update Dependabot

Edit `.github/dependabot.yml`:
- Add `gomod` entry for `release-4.23` with `k8s.io/*` ignore >= 0.37

### 6. Update Documentation

- `docs/VERSION-STRATEGY.md` — update support matrix
- `SECURITY.md` — update supported versions table
- `RELEASE.md` — update compatibility matrix
- `.github/CONTRIBUTING.md` — add `release-4.23` to branch list
- `.github/PULL_REQUEST_TEMPLATE.md` — add `release-4.23` to target branch options
- `docs/BRANCH_PROTECTION.md` — add `release-4.23` to protected branches

### 7. Deprecate Old Version (4.20)

Remove `release-4.20` from active workflow triggers (keep branch as archive).

### 8. Testing

```bash
git checkout release-4.23
make test
make build
make helm-lint
```

### 9. Commit Workflow Changes

```bash
git checkout main
git add .github/workflows/ charts/ docs/ .github/dependabot.yml
git commit -s -m "chore: add support for OpenShift 4.23, deprecate 4.20"
git push origin main
```

### 10. Verify

- [ ] CI passes on `release-4.23`
- [ ] Container image builds: `ocp-4.23-latest`
- [ ] Helm lint passes with `values-ocp-4.23.yaml`

## Checklist

- [ ] New release branch created with correct client-go version
- [ ] All workflow files updated
- [ ] Helm values overlay created
- [ ] Chart.yaml annotations updated
- [ ] Dependabot configured
- [ ] Documentation updated
- [ ] Old version deprecated
- [ ] Container images build successfully
