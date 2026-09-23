# Security Policy

## Supported Versions

The following table shows which versions of the OpenShift Cluster Health MCP Server and its dependencies are currently supported with security updates.

### MCP Server Releases

| Version | Supported          | Notes |
| ------- | ------------------ | ----- |
| v1.1.0  | :construction: In development | OCP 4.22 readiness |
| v0.2.0  | :white_check_mark: | Current development (unreleased) |
| v0.1.0  | :white_check_mark: | Initial release |

### OpenShift / Kubernetes Compatibility

| OpenShift | Kubernetes | client-go    | Supported          |
| --------- | ---------- | ------------ | ------------------ |
| 4.22      | 1.35       | v0.35.x      | :white_check_mark: Current |
| 4.21      | 1.34       | v0.34.x      | :white_check_mark: Supported |
| 4.20      | 1.33       | v0.33.x      | :white_check_mark: Supported |
| 4.19      | 1.32       | v0.32.x      | :warning: Maintenance only |
| 4.18      | 1.31       | v0.31.x      | :x: End of life |
| < 4.18    | < 1.31     | < v0.31.x    | :x: Not supported |

See [ADR-010](docs/adrs/010-version-compatibility-upgrade-roadmap.md) for the full version compatibility matrix and upgrade roadmap.

### Go Toolchain

| Go Version | Supported          |
| ---------- | ------------------ |
| 1.26       | :construction: Target (see [#128](https://github.com/KubeHeal/openshift-cluster-health-mcp/issues/128)) |
| 1.24 – 1.25 | :white_check_mark: Current |
| < 1.24     | :x: Not supported |

### MCP Go SDK

| MCP Go SDK | Supported          |
| ---------- | ------------------ |
| v1.8.0     | :white_check_mark: Current |
| v1.6.0     | :white_check_mark: Supported |
| v1.2.0     | :white_check_mark: Current |
| < v1.0.0   | :x: Not supported |

### Helm Chart

| Helm | Supported          |
| ---- | ------------------ |
| 3.12+ | :white_check_mark: |
| < 3.12 | :x: Not supported |

## Reporting a Vulnerability

If you discover a security vulnerability in this project, please report it responsibly.

### How to Report

1. **Do NOT open a public GitHub issue** for security vulnerabilities
2. **Use GitHub's private vulnerability reporting**: Navigate to the [Security tab](https://github.com/KubeHeal/openshift-cluster-health-mcp/security/advisories/new) and click "Report a vulnerability"
3. Alternatively, email the maintainers at the address listed in the repository's GitHub profile

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected versions (see supported versions above)
- Potential impact assessment
- Suggested fix (if you have one)

### What to Expect

| Stage | Timeline |
| ----- | -------- |
| **Acknowledgment** | Within 48 hours of report |
| **Initial assessment** | Within 5 business days |
| **Fix development** | Depends on severity (Critical: 7 days, High: 14 days, Medium: 30 days) |
| **Disclosure** | Coordinated with reporter after fix is available |

### If Accepted

- A fix will be developed and tested
- A security advisory will be published on GitHub
- The fix will be backported to all supported release branches (currently release-4.20, release-4.21, and release-4.22)
- Credit will be given to the reporter (unless they prefer anonymity)

### If Declined

- You will receive an explanation of why the report was not classified as a vulnerability
- The issue may be reclassified as a bug or enhancement if appropriate

## Security Scanning

This project uses automated security scanning in CI/CD:

| Tool | Purpose | When |
| ---- | ------- | ---- |
| **[gosec](https://github.com/securego/gosec)** | Go source code security analysis | Every PR and push (`make security-gosec`) |
| **[Trivy](https://github.com/aquasecurity/trivy)** | Container image vulnerability scanning | Every PR and push (`make security-scan`) |
| **[Dependabot](https://docs.github.com/en/code-security/dependabot)** | Dependency vulnerability alerts and updates | Continuous |

### Running Security Scans Locally

```bash
# Go source code analysis
make security-gosec

# Container image scan (requires built image)
make docker-build
make security-scan

# Check for known vulnerabilities in dependencies
go list -m -json all | docker run --rm -i sonatypecommunity/nancy:latest sleuth
```

## Security Architecture

The MCP server follows a defense-in-depth approach documented in [ADR-007: RBAC-Based Security Model](docs/adrs/007-rbac-based-security-model.md):

- **Principle of Least Privilege**: Read-only ClusterRole with minimal permissions
- **Non-root execution**: Runs as UID 1000 with `runAsNonRoot: true`
- **Read-only filesystem**: Container uses `readOnlyRootFilesystem: true`
- **No privilege escalation**: All capabilities dropped, `allowPrivilegeEscalation: false`
- **Network policies**: Ingress restricted to OpenShift Lightspeed, egress to required services only
- **OpenShift SCC compliance**: Uses default `restricted-v2` Security Context Constraint
- **Distroless base image**: Minimal attack surface ([ADR-008](docs/adrs/008-distroless-container-images.md))
- **No hardcoded credentials**: All secrets via Kubernetes Secrets or ServiceAccount tokens

## Related Documentation

- [ADR-007: RBAC-Based Security Model](docs/adrs/007-rbac-based-security-model.md)
- [ADR-008: Distroless Container Images](docs/adrs/008-distroless-container-images.md)
- [ADR-010: Version Compatibility and Upgrade Roadmap](docs/adrs/010-version-compatibility-upgrade-roadmap.md)
- [Branch Protection Policy](docs/BRANCH_PROTECTION.md)
