# Release Guide — openshift-cluster-health-mcp

This document is the canonical reference for cutting a release of the OpenShift Cluster Health MCP server.

---

## Versioning Policy

The MCP server uses **Semantic Versioning** (`MAJOR.MINOR.PATCH`):

| Increment | When |
|-----------|------|
| **MAJOR** | Breaking MCP tool API changes (tool removed, input schema incompatible) |
| **MINOR** | New MCP tools, new tool output fields, new CE endpoint integrations |
| **PATCH** | Bug fixes, dependency bumps, documentation corrections |

### OpenShift Compatibility Matrix

OCP 4.21 is now GA (April 2026) — active window is 4.19 / 4.20 / 4.21.

| OCP Version | Kubernetes | Status |
|-------------|------------|--------|
| 4.21        | 1.34       | Active (current) |
| 4.20        | 1.33       | Active |
| 4.19        | 1.32       | Active |
| 4.18        | 1.31       | Maintenance — dropping when 4.22 releases |

---

## Branch Strategy

```
main          ← integration branch (all required checks must pass before merge)
release-4.21  ← patch backports for OCP 4.21 train (current)
release-4.20  ← patch backports for OCP 4.20 train
release-4.19  ← patch backports for OCP 4.19 train
release-4.18  ← maintenance only (no new features)
```

---

## Required Checks Before Merge (Branch Protection)

All PRs targeting `main` must pass all **6 required checks** defined in
[`docs/BRANCH_PROTECTION.md`](./docs/BRANCH_PROTECTION.md) and [ADR-014](./docs/adrs/014-branch-protection-strategy.md):

| Check | Workflow | Required |
|-------|----------|----------|
| `Test` | `ci.yml` | ✅ |
| `Lint` | `ci.yml` | ✅ |
| `Build` | `ci.yml` | ✅ |
| `Security` | `ci.yml` | ✅ |
| `Helm` | `ci.yml` | ✅ |
| `build-and-push` | `container.yml` | ✅ |

At least **1 approving review** is required.

---

## Developer Certificate of Origin (DCO)

All commits **must** include a DCO sign-off:

```bash
git commit -s -m "feat: your commit message"
# Adds: Signed-off-by: Your Name <your@email.com>
```

---

## Release Checklist

### 1. Pre-Release

- [ ] All issues in the target milestone are closed or moved to next milestone
- [ ] `CHANGELOG.md` `[Unreleased]` section accurately describes all changes
- [ ] All new MCP tools documented in `README.md` tool table
- [ ] CI green on `main` — all 6 required checks pass
- [ ] `go.mod` Go version matches `ci.yml` `go-version:`
- [ ] Coordination Engine dependency version documented in README

### 2. CHANGELOG Update

Move the `[Unreleased]` content to a dated version section:

```markdown
## [0.2.0] - YYYY-MM-DD
```

Commit with sign-off:
```bash
git add CHANGELOG.md
git commit -s -m "release: prepare v0.2.0 — update CHANGELOG"
```

### 3. Tag and Push

```bash
VERSION=v0.2.0
git tag -a "$VERSION" -m "Release $VERSION"
git push origin main --tags
```

### 4. Container Image Publish

The `container.yml` workflow triggers automatically on `v*` tags and publishes
to the configured Quay registry. Monitor the run:

```bash
gh run watch --repo KubeHeal/openshift-cluster-health-mcp
```

Verify the image: `quay.io/kubeheal/openshift-cluster-health-mcp:<tag>`

### 5. GitHub Release Draft

```bash
gh release create "$VERSION" \
  --repo KubeHeal/openshift-cluster-health-mcp \
  --title "openshift-cluster-health-mcp $VERSION" \
  --notes-file <(sed -n "/^## \[$VERSION\]/,/^## \[/p" CHANGELOG.md | head -n -1) \
  --draft
```

### 6. Close Milestone

After verifying the release, close the milestone via GitHub UI or:
```bash
gh api repos/KubeHeal/openshift-cluster-health-mcp/milestones \
  --jq '.[] | select(.title=="'$VERSION'") | .number'
```

### 7. Deploy to OpenShift (Optional Manual Step)

Use the `openshift-deploy.yml` workflow for manual deployment:
```bash
gh workflow run openshift-deploy.yml \
  --repo KubeHeal/openshift-cluster-health-mcp \
  -f image_tag="$VERSION"
```

---

## Coordination Engine Compatibility

This MCP server depends on the Coordination Engine REST API. Before releasing,
verify that:

- [ ] CE version compatibility is documented in `README.md`
- [ ] All new client methods in `pkg/clients/coordination_engine.go` have a corresponding
  CE endpoint in the target CE release
- [ ] CE endpoints are reachable in the target OCP environment

---

## Related Documentation

- [CHANGELOG.md](./CHANGELOG.md) — full version history
- [docs/BRANCH_PROTECTION.md](./docs/BRANCH_PROTECTION.md) — required checks
- [docs/adrs/](./docs/adrs/) — Architectural Decision Records
- [.github/CONTRIBUTING.md](./.github/CONTRIBUTING.md) — development workflow
- [GitHub Issues](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues)
