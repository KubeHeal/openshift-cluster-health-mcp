# Playbook: Cherry-Picking Critical Fixes

## Overview

This playbook guides you through cherry-picking critical bug fixes or security
patches from `main` to older release branches.

## When to Cherry-Pick

**DO cherry-pick**:
- Critical security vulnerabilities
- Data corruption bugs
- Production outages
- Critical performance issues

**DO NOT cherry-pick**:
- New features
- Refactoring
- Non-critical improvements
- Documentation-only changes

## Prerequisites

- [ ] Fix has been merged to `main` and tested
- [ ] Issue affects the target release branches
- [ ] Fix is compatible with older Kubernetes API versions

## Steps

### 1. Identify the Commit

```bash
git checkout main
git pull origin main
git log --oneline | grep "fix:"
```

### 2. Determine Target Branches

```bash
# Check which branches need the fix
git checkout release-4.21
git log --oneline | grep "<fix description>"  # Should not exist
```

### 3. Cherry-Pick to Each Branch

```bash
git checkout release-4.21
git pull origin release-4.21
git cherry-pick <commit-sha>

# If conflicts occur:
git status
# Resolve conflicts...
git add <resolved-files>
git cherry-pick --continue

# Test
make test
make build

# Push (triggers container build automatically)
git push origin release-4.21
```

Repeat for each target branch (e.g., `release-4.20`).

### 4. Verify Container Builds

After pushing to each branch, the `release-quay.yml` workflow builds new images:

```bash
# Check workflow status
gh run list --repo KubeHeal/openshift-cluster-health-mcp \
  --workflow release-quay.yml --limit 5

# Verify images
podman pull quay.io/takinosh/openshift-cluster-health-mcp:ocp-4.21-latest
```

### 5. Update Tracking

Document the cherry-pick in the PR or issue:

```markdown
## Cherry-Pick Status

- [x] `main` (original fix)
- [x] `release-4.21` (cherry-picked: <sha>)
- [x] `release-4.20` (cherry-picked: <sha>)
```

### 6. Notify Users (if security patch)

- [ ] Update GitHub Security Advisory
- [ ] Recommend users update deployments

## Handling Conflicts

### API Version Differences

Older branches may use different Kubernetes API versions. Adjust imports to
match the branch's `go.mod` dependencies.

### Aborting a Cherry-Pick

```bash
git cherry-pick --abort
```

Document why the cherry-pick was skipped.

## Checklist

- [ ] Commit identified from main
- [ ] Target branches determined
- [ ] Cherry-pick applied to each branch
- [ ] Conflicts resolved (if any)
- [ ] Tests passed on all branches
- [ ] Container images built successfully
- [ ] Tracking updated in PR/issue
