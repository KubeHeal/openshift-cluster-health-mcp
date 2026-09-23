# ADR Documentation Enhancement Todo List

**Generated:** 2026-01-25
**Priority:** Low (Non-blocking)
**Status:** Completed (2026-09-23, issue #116)

## Overview

This document tracks minor documentation enhancements identified during the ADR synchronization review. All ADRs are fully implemented with 9/10 compliance scores. These items improve documentation clarity but are not blocking issues.

---

## ADR-001: Go Language Selection for MCP Server

**Current Compliance:** 9/10
**Gap:** Missing Kubernetes references in documentation

### Tasks
- [x] Add explicit Kubernetes client-go references to Context section
  - Location: `docs/adrs/001-go-language-selection.md` - Context section
  - Detail: Reference `pkg/clients/kubernetes.go` as evidence of Go + Kubernetes integration
  - Completed: 2026-09-23

- [x] Document Go version compatibility with Kubernetes API versions
  - Location: `docs/adrs/001-go-language-selection.md` - Technical Considerations section
  - Detail: Added Go version compatibility table with k8s.io client-go version mapping
  - Completed: 2026-09-23

---

## ADR-005: Stateless Design (No Database)

**Current Compliance:** 9/10
**Gap:** Missing Kubernetes stateless pattern references

### Tasks
- [x] Enhance Consequences section with Kubernetes StatefulSet comparison
  - Location: `docs/adrs/005-stateless-design.md` - Consequences section
  - Detail: Added "Why Deployment Instead of StatefulSet" subsection explaining why StatefulSet is not needed
  - Completed: 2026-09-23

- [x] Document caching strategy as stateless pattern implementation
  - Location: `docs/adrs/005-stateless-design.md` - Implementation section
  - Detail: Added "In-Memory Caching as a Stateless Pattern" subsection referencing `pkg/cache/memory_cache.go`
  - Completed: 2026-09-23

---

## ADR-007: RBAC-Based Security Model

**Current Compliance:** 9/10
**Gap:** Missing Kubernetes RBAC documentation cross-references

### Tasks
- [x] Cross-reference Kubernetes RBAC documentation
  - Location: `docs/adrs/007-rbac-based-security-model.md` - Decision section
  - Detail: Added links to official Kubernetes RBAC docs and OpenShift RBAC best practices
  - Completed: 2026-09-23

- [x] Add examples of ClusterRole and ServiceAccount YAML from charts/
  - Location: `docs/adrs/007-rbac-based-security-model.md` - Implementation section
  - Detail: ClusterRole YAML snippets already present; added explicit reference to `charts/openshift-cluster-health-mcp/templates/clusterrole.yaml`
  - Completed: 2026-09-23

---

## ADR-009: Architecture Evolution Roadmap

**Current Compliance:** 9/10
**Gap:** Missing PostgreSQL and Kubernetes future planning details

### Tasks
- [x] Update Phase 3 PostgreSQL planning section with current timeline
  - Location: `docs/adrs/009-architecture-evolution-roadmap.md` - Roadmap section
  - Detail: Added "PostgreSQL and Persistent Storage Decision Criteria" section, marked as deferred with no active plans
  - Completed: 2026-09-23

- [x] Document decision criteria for when to implement persistent storage
  - Location: `docs/adrs/009-architecture-evolution-roadmap.md` - Decision Criteria section
  - Detail: Added 4 specific triggers (incident history capacity, audit trail, offline mode, request volume)
  - Completed: 2026-09-23

---

## ADR-010: Version Compatibility and Upgrade Roadmap

**Current Compliance:** 9/10
**Gap:** Missing Kubernetes version compatibility matrix

### Tasks
- [x] Add Kubernetes API version compatibility matrix
  - Location: `docs/adrs/010-version-compatibility-upgrade-roadmap.md` - Compatibility section
  - Detail: Extended compatibility matrix to cover OCP 4.18 through 4.22 with k8s.io version mapping
  - Completed: 2026-09-23 (via issue #123 ADR-010 amendment)

- [x] Document tested Kubernetes versions (1.24, 1.25, 1.26, etc.)
  - Location: `docs/adrs/010-version-compatibility-upgrade-roadmap.md` - Testing section
  - Detail: Updated CI matrix to test against K8s 1.33 (OCP 4.20), 1.34 (OCP 4.21), 1.35 (OCP 4.22)
  - Completed: 2026-09-23 (via issue #123 ADR-010 amendment)

---

## Security Notice Review

**Current Status:** Informational (not a gap)
**Confidence:** 60% (likely false positive)

### Tasks
- [x] Review GitHub workflow token usage in `.github/workflows/ci.yml:86`
  - Detail: Confirmed false positive. Line 86 is a kubectl config command. No `secrets.GITHUB_TOKEN` reference exists in ci.yml at all — the flagged lines use `env.OPENSHIFT_TOKEN` (environment variable, not a secret reference).
  - Completed: 2026-09-23

- [x] Review GitHub workflow token usage in `.github/workflows/ci.yml:98`
  - Detail: Confirmed false positive. Line 98 is part of the healthcheck test block using `env.OPENSHIFT_TOKEN`. Only `container.yml` uses `secrets.*` (for standard Quay registry credentials).
  - Completed: 2026-09-23

---

## Implementation Notes

### Process
1. Create GitHub Issues for each ADR enhancement (optional)
2. Tag issues with `documentation`, `adr`, and `enhancement` labels
3. Assign to appropriate team member during sprint planning
4. Target completion: Q2 2026 (no urgency)

### Acceptance Criteria
- ADR documents updated with new content
- Cross-references to code files include line numbers
- All links to external documentation are valid
- Changes reviewed by at least one team member
- ADR compliance score improves to 9.5+/10 after changes

### Estimated Total Effort
**Total:** ~4 hours (distributed across multiple team members)

---

## Related Documents
- [ADR Sync Report](./ADR_SYNC_REPORT.md)
- [ADR Directory](./adrs/)
- [Branch Protection](./BRANCH_PROTECTION.md)
- [CLAUDE.md](../CLAUDE.md)
