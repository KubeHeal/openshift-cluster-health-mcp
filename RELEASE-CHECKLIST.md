# Cross-Repository Release Checklist

This checklist documents the end-to-end release sequence across the KubeHeal
ecosystem. Follow every step in order. Each checkbox is a gate.

> **Scope**: This covers coordination between
> [openshift-cluster-health-mcp](https://github.com/KubeHeal/openshift-cluster-health-mcp)
> and [kubeheal-operator](https://github.com/KubeHeal/kubeheal-operator).
> For single-repo release details (tagging, changelog, DCO), see [RELEASE.md](./RELEASE.md).

---

## Prerequisites

- [ ] You have push access to `KubeHeal/openshift-cluster-health-mcp` and `KubeHeal/kubeheal-operator`
- [ ] You have the `gh` CLI authenticated (`gh auth status`)
- [ ] You have `helm`, `make`, and `go` installed locally
- [ ] Quay.io push credentials are configured in repo secrets (`QUAY_USERNAME`, `QUAY_PASSWORD`)

---

## Phase 1 — MCP Server Release

### 1.1 Pre-flight checks

- [ ] All issues in the target milestone are closed or deferred
- [ ] CI is green on `main` — all required checks pass
- [ ] Security scan is clean — no open High/Critical Trivy alerts
- [ ] `go.mod` Go version matches `ci.yml` `go-version:`
- [ ] CHANGELOG.md `[Unreleased]` section is complete

### 1.2 Prepare the release commit

```bash
# Update CHANGELOG.md: move [Unreleased] to [X.Y.Z] - YYYY-MM-DD
git add CHANGELOG.md
git commit -s -m "release: prepare vX.Y.Z — update CHANGELOG"
git push origin main
```

### 1.3 Create release branch (if new OCP version)

```bash
git checkout main && git pull
git checkout -b release-4.XX

go get k8s.io/client-go@v0.XX.0
go get k8s.io/api@v0.XX.0
go get k8s.io/apimachinery@v0.XX.0
go mod tidy

git add go.mod go.sum
git commit -s -m "chore: initialize release-4.XX for OCP 4.XX (k8s 1.XX)"
git push -u origin release-4.XX
```

- [ ] Release branch exists on GitHub
- [ ] `release-quay.yml` workflow triggered on push

### 1.4 Verify image publish

```bash
gh run list --repo KubeHeal/openshift-cluster-health-mcp \
  --workflow release-quay.yml --limit 3

podman pull quay.io/takinosh/openshift-cluster-health-mcp:ocp-4.22-latest
```

- [ ] Image `ocp-4.22-latest` exists on Quay
- [ ] Image `ocp-4.22-<sha>` exists on Quay
- [ ] Trivy scan passed

### 1.5 E2E validation (major releases only)

```bash
# Run E2E against live cluster
gh workflow run e2e-openshift.yml --repo KubeHeal/openshift-cluster-health-mcp

# Or locally:
./scripts/setup-e2e.sh --deploy
```

- [ ] E2E tests pass (or skipped for minor/patch)

### 1.6 Tag and create GitHub Release

```bash
VERSION=vX.Y.Z
git checkout main
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

gh release create "$VERSION" \
  --repo KubeHeal/openshift-cluster-health-mcp \
  --title "openshift-cluster-health-mcp $VERSION" \
  --notes-file <(sed -n "/^## \[$VERSION\]/,/^## \[/p" CHANGELOG.md | head -n -1) \
  --draft
```

- [ ] Tag pushed
- [ ] GitHub Release draft created — review and publish
- [ ] Milestone closed

---

## Phase 2 — Kubeheal Operator Update

**Blocked by**: Phase 1 complete (published image on Quay)

### 2.1 Update operator Helm chart

```bash
cd kubeheal-operator

# Update image reference
vim charts/kubeheal-operator/values.yaml
# image.tag: "ocp-4.22-latest"

# Update Chart.yaml appVersion
vim charts/kubeheal-operator/Chart.yaml

git add -A
git commit -s -m "chore: update MCP server image to vX.Y.Z (ocp-4.22)"
git push origin main
```

- [ ] Helm chart references new MCP server image
- [ ] CI passes on operator main branch

### 2.2 Build and push operator images

```bash
make docker-build docker-push IMG=quay.io/takinosh/kubeheal-operator:vX.Y.Z
make bundle-build bundle-push \
  BUNDLE_IMG=quay.io/takinosh/kubeheal-operator-bundle:vX.Y.Z
```

- [ ] Operator image pushed
- [ ] Bundle image pushed

### 2.3 Smoke test

```bash
kind create cluster --name kubeheal-test
make deploy IMG=quay.io/takinosh/kubeheal-operator:vX.Y.Z
kubectl wait --for=condition=ready pod -l app=mcp-server \
  -n kubeheal-system --timeout=120s
curl -sf http://localhost:8080/health && echo "PASS"
kind delete cluster --name kubeheal-test
```

- [ ] Operator deploys MCP server successfully
- [ ] Health endpoint returns 200

### 2.4 Tag operator release

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
gh release create vX.Y.Z --repo KubeHeal/kubeheal-operator --generate-notes --draft
```

- [ ] Tag pushed and release published

---

## Phase 3 — Distribution (OperatorHub)

**Blocked by**: Phase 2 complete

### 3.1 Submit to community-operators-prod

- [ ] PR opened against `k8s-operatorhub/community-operators-prod`
- [ ] OLM CI pipeline passes
- [ ] PR merged

---

## Post-Release

- [ ] Announce the release
- [ ] Update downstream consumers (Validated Patterns, RHDP catalog items)
- [ ] Open the next milestone and triage deferred issues
- [ ] Archive old release branches past the support window
