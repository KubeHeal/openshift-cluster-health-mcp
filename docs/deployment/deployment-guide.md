# MCP Server Deployment Guide

> **Version:** 0.2.0 | **Last updated:** 2026-09-23

This guide describes how to deploy and operate the OpenShift Cluster Health MCP Server
on OpenShift. It covers Helm installation, RBAC, ROSA compatibility, MCP endpoint
verification, upgrades, and troubleshooting.

## Prerequisites

| Requirement | Minimum Version |
|---|---|
| OpenShift | 4.20 |
| Kubernetes | 1.33 |
| Helm | 3.0 |

The MCP server supports OpenShift 4.20, 4.21, and 4.22. Each version has a dedicated
values overlay file.

## Step 1: Create the Namespace

```bash
kubectl create namespace self-healing-platform
```

The MCP server runs in the `self-healing-platform` namespace by default.

## Step 2: Select the Values File

The chart ships with version-specific values overlays:

| OpenShift Version | Values File | Image Tag |
|---|---|---|
| 4.20 | `values-ocp-4.20.yaml` | `ocp-4.20-latest` |
| 4.21 | `values-ocp-4.21.yaml` | `ocp-4.21-latest` |
| 4.22 | `values-ocp-4.22.yaml` | `ocp-4.22-latest` |

Identify your cluster version:

```bash
oc version
```

## Step 3: Install the Helm Chart

Install using the values overlay that matches your cluster version:

```bash
helm install mcp-server ./charts/openshift-cluster-health-mcp \
  -n self-healing-platform \
  -f ./charts/openshift-cluster-health-mcp/values-ocp-4.22.yaml
```

Helm creates the following resources:

- Deployment (2 replicas by default)
- Service (ClusterIP on port 8080)
- ServiceAccount
- ClusterRole and ClusterRoleBinding (read access to nodes, pods, namespaces)
- NetworkPolicy (optional, disabled by default)

## Step 4: Verify the Deployment

```bash
# Check pod status
kubectl get pods -n self-healing-platform -l app.kubernetes.io/name=openshift-cluster-health-mcp

# Check the health endpoint
kubectl exec -n self-healing-platform deploy/mcp-server -- \
  curl -s http://localhost:8080/health | jq .

# Verify MCP tools are available
kubectl exec -n self-healing-platform deploy/mcp-server -- \
  curl -s http://localhost:8080/mcp/tools | jq .

# Verify MCP resources are available
kubectl exec -n self-healing-platform deploy/mcp-server -- \
  curl -s http://localhost:8080/mcp/resources | jq .
```

## RBAC Resources

The chart creates a ClusterRole with read permissions:

| API Group | Resources | Verbs |
|---|---|---|
| `""` (core) | nodes | get, list, watch |
| `""` (core) | pods | get, list, watch |
| `""` (core) | namespaces | get, list |

Optional integrations (enabled via values) may require additional RBAC for KServe
InferenceServices.

To verify RBAC permissions after installation:

```bash
kubectl auth can-i list pods \
  --as=system:serviceaccount:self-healing-platform:mcp-server \
  -n self-healing-platform
```

## Pod Security Context (ROSA Compatibility)

The chart runs with OpenShift **restricted-v2** SCC compatibility:

- `runAsNonRoot: true`
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `seccompProfile.type: RuntimeDefault`
- All capabilities are dropped

The `runAsUser` and `fsGroup` fields are **null** by default so that OpenShift assigns
UIDs from the namespace range. This is required for ROSA clusters.

**Do not set `runAsUser` on ROSA clusters.** Let OpenShift assign the UID.

The deployment mounts two writable `emptyDir` volumes at `/tmp` and `/cache` for
temporary files and cache data.

## Optional Integrations

### Coordination Engine

Enable communication with the Coordination Engine for incident management:

```yaml
integrations:
  coordinationEngine:
    enabled: true
    url: http://coordination-engine.self-healing-platform.svc:8080/api/v1
```

### KServe ML Models

Enable anomaly detection via KServe InferenceServices:

```yaml
integrations:
  kserve:
    enabled: true
    namespace: self-healing-platform
    predictorPort: 8080
```

### Prometheus

Enable Prometheus metrics querying:

```yaml
integrations:
  prometheus:
    enabled: true
    url: https://thanos-querier.openshift-monitoring.svc:9091
```

## Upgrade Procedure

### Helm Upgrade

```bash
helm upgrade mcp-server ./charts/openshift-cluster-health-mcp \
  -n self-healing-platform \
  -f ./charts/openshift-cluster-health-mcp/values-ocp-4.22.yaml
```

Verify the new pod is running:

```bash
kubectl rollout status deployment/mcp-server -n self-healing-platform
```

### Rollback

```bash
helm rollback mcp-server -n self-healing-platform
```

## Troubleshooting

### CrashLoopBackOff

**Symptom:** The pod repeatedly restarts.

1. Check the logs:
   ```bash
   kubectl logs -n self-healing-platform deploy/mcp-server --previous
   ```
2. Common causes:
   - Missing kubeconfig (when running outside the cluster)
   - Port conflict (`MCP_HTTP_PORT` already in use)

### RBAC Errors

**Symptom:** Log messages contain `"forbidden"` or `"cannot list resource"`.

1. Verify the ClusterRole and ClusterRoleBinding exist:
   ```bash
   kubectl get clusterrole,clusterrolebinding | grep mcp-server
   ```
2. Check that `rbac.create` is `true` in your values file.
3. Test a specific permission:
   ```bash
   kubectl auth can-i list nodes \
     --as=system:serviceaccount:self-healing-platform:mcp-server
   ```

### Image Pull Errors

**Symptom:** Pod status shows `ImagePullBackOff` or `ErrImagePull`.

1. Verify the image exists:
   ```bash
   podman pull quay.io/takinosh/openshift-cluster-health-mcp:ocp-4.22-latest
   ```
2. If the registry requires authentication, create an image pull secret:
   ```bash
   kubectl create secret docker-registry regcred \
     --docker-server=quay.io \
     --docker-username=<user> \
     --docker-password=<token> \
     -n self-healing-platform
   ```
   Then set `imagePullSecrets` in your values file.

### SCC Issues on OpenShift

**Symptom:** Pod fails to start with `"unable to validate against any security context constraint"`.

1. Check the pod events:
   ```bash
   kubectl describe pod -n self-healing-platform -l app.kubernetes.io/name=openshift-cluster-health-mcp
   ```
2. The chart defaults are compatible with `restricted-v2`. If you override
   `podSecurityContext.runAsUser`, verify the UID is within the namespace range:
   ```bash
   kubectl get namespace self-healing-platform \
     -o jsonpath='{.metadata.annotations.openshift\.io/sa\.scc\.uid-range}'
   ```
3. Do not set `runAsUser` on ROSA clusters. Let OpenShift assign the UID.

### MCP Endpoints Not Responding

**Symptom:** `/mcp/tools` returns empty or errors.

1. Verify the MCP transport is set to HTTP:
   ```bash
   kubectl get deploy mcp-server -n self-healing-platform \
     -o jsonpath='{.spec.template.spec.containers[0].env}' | jq .
   ```
2. Check for `MCP_TRANSPORT=http` in the environment.
3. Verify the pod is ready:
   ```bash
   kubectl get pods -n self-healing-platform -l app.kubernetes.io/name=openshift-cluster-health-mcp
   ```

## Related Documentation

- [RELEASE.md](../../RELEASE.md) for versioning and release procedures
- [VERSION-STRATEGY.md](../VERSION-STRATEGY.md) for multi-version support strategy
- [Helm Chart values.yaml](../../charts/openshift-cluster-health-mcp/values.yaml) for all configuration options
