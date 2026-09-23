#!/usr/bin/env bash
# setup-e2e.sh — Deploy MCP server to a cluster and run E2E smoke tests
#
# Usage:
#   ./scripts/setup-e2e.sh --deploy              Deploy MCP server + smoke test
#   ./scripts/setup-e2e.sh --audit               Audit cluster readiness
#   ./scripts/setup-e2e.sh --secrets             Print GitHub secrets needed for CI
#   ./scripts/setup-e2e.sh --set-secrets         Auto-set GitHub secrets via gh CLI
#   ./scripts/setup-e2e.sh --destroy             Uninstall chart + delete namespace
#   ./scripts/setup-e2e.sh --status              Check deployment status
#
# Environment variables (optional overrides):
#   NAMESPACE          — Target namespace (default: self-healing-platform)
#   HELM_RELEASE       — Helm release name (default: mcp-server)
#   IMAGE_TAG          — Image tag to deploy (default: ocp-4.22-latest)
#   IMAGE_REPO         — Image repository (default: quay.io/takinosh/openshift-cluster-health-mcp)
#   VALUES_FILE        — Helm values file (default: values-ocp-4.22.yaml)
#   SKIP_SMOKE_TEST    — Set to "true" to skip the post-deploy smoke test

set -euo pipefail

# Config
NAMESPACE="${NAMESPACE:-self-healing-platform}"
HELM_RELEASE="${HELM_RELEASE:-mcp-server}"
IMAGE_REPO="${IMAGE_REPO:-quay.io/takinosh/openshift-cluster-health-mcp}"
IMAGE_TAG="${IMAGE_TAG:-ocp-4.22-latest}"
VALUES_FILE="${VALUES_FILE:-values-ocp-4.22.yaml}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CHART_DIR="$REPO_DIR/charts/openshift-cluster-health-mcp"
GH_REPO="KubeHeal/openshift-cluster-health-mcp"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${BLUE}INFO${NC}  $*"; }
ok()      { echo -e "${GREEN}PASS${NC}  $*"; }
warn()    { echo -e "${YELLOW}WARN${NC}  $*"; }
fail()    { echo -e "${RED}FAIL${NC}  $*"; exit 1; }
fail_soft() { echo -e "${RED}FAIL${NC}  $*"; }
section() { echo -e "\n${CYAN}=== $* ===${NC}\n"; }

# ──────────────────────────────────────────────────────────────────────
# Preflight checks
# ──────────────────────────────────────────────────────────────────────
preflight() {
    info "Running preflight checks..."

    for tool in kubectl helm; do
        if ! command -v "$tool" &>/dev/null; then
            fail "$tool not found. Install it first."
        fi
        ok "$tool available: $(command -v "$tool")"
    done

    if command -v oc &>/dev/null; then
        KUBECTL="oc"
        ok "oc CLI available"
    else
        KUBECTL="kubectl"
        warn "oc CLI not found, using kubectl"
    fi

    if ! $KUBECTL cluster-info &>/dev/null; then
        fail "Not connected to a cluster. Run: oc login or export KUBECONFIG"
    fi

    CLUSTER_VERSION=$($KUBECTL version -o json 2>/dev/null | python3 -c "import sys,json; print('Server: ' + json.load(sys.stdin).get('serverVersion',{}).get('gitVersion','unknown'))" 2>/dev/null || echo "unknown")
    ok "Connected to cluster: $CLUSTER_VERSION"

    if [ ! -f "$CHART_DIR/Chart.yaml" ]; then
        fail "Helm chart not found at $CHART_DIR"
    fi
    ok "Helm chart found: $CHART_DIR"
}

# ──────────────────────────────────────────────────────────────────────
# Deploy MCP server
# ──────────────────────────────────────────────────────────────────────
deploy_server() {
    section "Deploying MCP server"

    info "Release:   $HELM_RELEASE"
    info "Namespace: $NAMESPACE"
    info "Image:     $IMAGE_REPO:$IMAGE_TAG"
    info "Values:    $VALUES_FILE"
    echo ""

    helm uninstall "$HELM_RELEASE" -n "$NAMESPACE" 2>/dev/null || true

    helm install "$HELM_RELEASE" "$CHART_DIR" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --values "$CHART_DIR/$VALUES_FILE" \
        --set "image.repository=$IMAGE_REPO" \
        --set "image.tag=$IMAGE_TAG" \
        --set "image.pullPolicy=Always" \
        --wait \
        --timeout 5m

    ok "Helm install complete"

    info "Waiting for deployment rollout..."
    $KUBECTL rollout status deployment/"$HELM_RELEASE" \
        -n "$NAMESPACE" --timeout=3m 2>/dev/null || \
    $KUBECTL rollout status deployment/mcp-server \
        -n "$NAMESPACE" --timeout=3m 2>/dev/null || true

    echo ""
    $KUBECTL get pods -n "$NAMESPACE"
    ok "Deployment complete"
}

# ──────────────────────────────────────────────────────────────────────
# Smoke test
# ──────────────────────────────────────────────────────────────────────
run_smoke_test() {
    section "Running MCP smoke test"

    if [ "${SKIP_SMOKE_TEST:-false}" = "true" ]; then
        warn "Smoke test skipped (SKIP_SMOKE_TEST=true)"
        return 0
    fi

    POD_NAME=$($KUBECTL get pods -n "$NAMESPACE" \
        -l "app.kubernetes.io/name=openshift-cluster-health-mcp" \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$POD_NAME" ]; then
        fail "No MCP server pod found in $NAMESPACE"
    fi
    info "Pod: $POD_NAME"

    $KUBECTL port-forward "pod/$POD_NAME" 18080:8080 -n "$NAMESPACE" &
    PF_PID=$!
    sleep 3

    # Test 1: Health endpoint
    info "Testing GET /health..."
    RESPONSE=$(curl -sf http://localhost:18080/health 2>/dev/null || echo "")
    if [ -z "$RESPONSE" ]; then
        kill $PF_PID 2>/dev/null || true
        fail "No response from /health endpoint"
    fi
    STATUS=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
    if [ "$STATUS" = "ok" ]; then
        ok "Health check passed: $RESPONSE"
    else
        kill $PF_PID 2>/dev/null || true
        fail "Health check failed. Response: $RESPONSE"
    fi

    # Test 2: MCP tools
    info "Testing GET /mcp/tools..."
    TOOLS=$(curl -sf http://localhost:18080/mcp/tools 2>/dev/null || echo "[]")
    TOOL_COUNT=$(echo "$TOOLS" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
    if [ "$TOOL_COUNT" -ge 1 ]; then
        ok "MCP tools endpoint: $TOOL_COUNT tools available"
    else
        warn "MCP tools endpoint returned 0 tools (may need cluster connectivity)"
    fi

    # Test 3: MCP resources
    info "Testing GET /mcp/resources..."
    RESOURCES=$(curl -sf http://localhost:18080/mcp/resources 2>/dev/null || echo "[]")
    RESOURCE_COUNT=$(echo "$RESOURCES" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
    if [ "$RESOURCE_COUNT" -ge 1 ]; then
        ok "MCP resources endpoint: $RESOURCE_COUNT resources available"
    else
        warn "MCP resources endpoint returned 0 resources"
    fi

    # Test 4: MCP session
    info "Testing POST /mcp/session..."
    SESSION=$(curl -sf -X POST http://localhost:18080/mcp/session \
        -H 'Content-Type: application/json' \
        -d '{"client":"e2e-test"}' 2>/dev/null || echo "")
    if echo "$SESSION" | grep -q "session_id"; then
        SESSION_ID=$(echo "$SESSION" | python3 -c "import sys,json; print(json.load(sys.stdin).get('session_id',''))" 2>/dev/null || echo "")
        ok "Session created: $SESSION_ID"

        # Test 5: Execute tool via session
        info "Testing POST /mcp/tools/get-cluster-health/call..."
        HEALTH=$(curl -sf -X POST "http://localhost:18080/mcp/tools/get-cluster-health/call?sessionid=$SESSION_ID" \
            -H 'Content-Type: application/json' \
            -d '{}' 2>/dev/null || echo "")
        if [ -n "$HEALTH" ]; then
            ok "Tool execution succeeded"
        else
            warn "Tool execution returned empty (may need cluster connectivity)"
        fi
    else
        warn "Session creation returned unexpected response"
    fi

    kill $PF_PID 2>/dev/null || true
    wait $PF_PID 2>/dev/null || true
}

# ──────────────────────────────────────────────────────────────────────
# Audit cluster readiness
# ──────────────────────────────────────────────────────────────────────
audit_cluster() {
    section "E2E Cluster Readiness Audit"

    local PASS=0
    local FAIL_COUNT=0

    info "[1/6] Cluster connection"
    if $KUBECTL cluster-info &>/dev/null; then
        ok "  Connected to cluster"
        PASS=$((PASS + 1))
    else
        fail_soft "  Not connected to a cluster"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    info "[2/6] Helm chart"
    if [ -f "$CHART_DIR/Chart.yaml" ]; then
        CHART_VER=$(grep "^version:" "$CHART_DIR/Chart.yaml" | awk '{print $2}')
        ok "  Chart found: version $CHART_VER"
        PASS=$((PASS + 1))
    else
        fail_soft "  Chart not found at $CHART_DIR"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    info "[3/6] Target namespace"
    if $KUBECTL get namespace "$NAMESPACE" &>/dev/null; then
        ok "  Namespace $NAMESPACE exists"
        PASS=$((PASS + 1))
    else
        warn "  Namespace $NAMESPACE does not exist (will be created by Helm)"
        PASS=$((PASS + 1))
    fi

    info "[4/6] MCP server deployment"
    DEP_NAME=$($KUBECTL get deployments -n "$NAMESPACE" -o name 2>/dev/null | grep mcp-server | head -1 || echo "")
    if [ -n "$DEP_NAME" ]; then
        READY=$($KUBECTL get "$DEP_NAME" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "${READY:-0}" -gt 0 ]; then
            ok "  Deployment running ($READY replicas ready)"
            PASS=$((PASS + 1))
        else
            fail_soft "  Deployment exists but not ready"
            FAIL_COUNT=$((FAIL_COUNT + 1))
        fi
    else
        warn "  No deployment found (run --deploy first)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    info "[5/6] Health endpoint"
    POD_NAME=$($KUBECTL get pods -n "$NAMESPACE" \
        -l "app.kubernetes.io/name=openshift-cluster-health-mcp" \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$POD_NAME" ]; then
        $KUBECTL port-forward "pod/$POD_NAME" 18080:8080 -n "$NAMESPACE" &>/dev/null &
        PF_PID=$!
        sleep 2
        RESPONSE=$(curl -sf http://localhost:18080/health 2>/dev/null || echo "")
        kill $PF_PID 2>/dev/null || true
        wait $PF_PID 2>/dev/null || true

        if echo "$RESPONSE" | grep -q '"ok"'; then
            ok "  Health endpoint returns 200"
            PASS=$((PASS + 1))
        else
            fail_soft "  Health endpoint not responding"
            FAIL_COUNT=$((FAIL_COUNT + 1))
        fi
    else
        fail_soft "  No pod found to check health"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    info "[6/6] Container image"
    if [ -n "$POD_NAME" ]; then
        CURRENT_IMAGE=$($KUBECTL get pod "$POD_NAME" -n "$NAMESPACE" \
            -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "unknown")
        ok "  Image: $CURRENT_IMAGE"
        PASS=$((PASS + 1))
    else
        warn "  Cannot check image (no pod running)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi

    echo ""
    echo "========================================================"
    echo "  Audit Summary: $PASS passed, $FAIL_COUNT failed"
    echo "========================================================"

    if [ "$FAIL_COUNT" -gt 0 ]; then
        warn "Cluster is not fully ready for E2E testing"
        return 1
    else
        ok "Cluster is ready for E2E testing!"
        return 0
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Print / set GitHub secrets
# ──────────────────────────────────────────────────────────────────────
print_github_secrets() {
    echo ""
    echo "========================================================"
    echo "  GitHub Secrets for E2E OpenShift Workflow"
    echo "========================================================"
    echo ""
    echo "  Required:"
    echo "    OPENSHIFT_SERVER=<cluster API URL>"
    echo "    OPENSHIFT_TOKEN=<login token>          # from: oc whoami -t"
    echo ""
    echo "  Optional (for image push):"
    echo "    QUAY_USERNAME=<quay.io username>"
    echo "    QUAY_PASSWORD=<quay.io password/token>"
    echo ""

    if command -v oc &>/dev/null && oc whoami &>/dev/null; then
        echo "  Current cluster values:"
        echo "    OPENSHIFT_SERVER=$(oc whoami --show-server 2>/dev/null || echo 'N/A')"
        echo "    OPENSHIFT_TOKEN=$(oc whoami -t 2>/dev/null || echo 'N/A')"
        echo ""
    fi

    echo "  Set them with:"
    echo "    gh secret set OPENSHIFT_SERVER --repo $GH_REPO"
    echo "    gh secret set OPENSHIFT_TOKEN --repo $GH_REPO"
    echo ""
    echo "========================================================"
}

set_github_secrets() {
    info "Setting GitHub secrets for E2E workflow..."

    if ! command -v gh &>/dev/null; then
        fail "gh CLI not installed"
    fi
    if ! gh auth status &>/dev/null; then
        fail "gh CLI not authenticated. Run: gh auth login"
    fi

    if command -v oc &>/dev/null && oc whoami &>/dev/null; then
        SERVER=$(oc whoami --show-server 2>/dev/null || echo "")
        TOKEN=$(oc whoami -t 2>/dev/null || echo "")

        if [ -n "$SERVER" ]; then
            gh secret set OPENSHIFT_SERVER --repo "$GH_REPO" --body "$SERVER"
            ok "OPENSHIFT_SERVER set"
        fi
        if [ -n "$TOKEN" ]; then
            gh secret set OPENSHIFT_TOKEN --repo "$GH_REPO" --body "$TOKEN"
            ok "OPENSHIFT_TOKEN set"
        fi
    else
        warn "Not logged into OpenShift. Set secrets manually."
        print_github_secrets
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Status check
# ──────────────────────────────────────────────────────────────────────
check_status() {
    section "MCP Server Status"

    echo "  Namespace: $NAMESPACE"
    echo "  Release:   $HELM_RELEASE"
    echo ""

    info "Helm releases:"
    helm list -n "$NAMESPACE" 2>/dev/null || echo "  (none)"
    echo ""

    info "Pods:"
    $KUBECTL get pods -n "$NAMESPACE" 2>/dev/null || echo "  (none)"
    echo ""

    info "Services:"
    $KUBECTL get svc -n "$NAMESPACE" 2>/dev/null || echo "  (none)"
}

# ──────────────────────────────────────────────────────────────────────
# Destroy
# ──────────────────────────────────────────────────────────────────────
destroy() {
    section "Destroying MCP server deployment"

    helm uninstall "$HELM_RELEASE" -n "$NAMESPACE" 2>/dev/null && ok "Helm release uninstalled" || warn "No release to uninstall"
    $KUBECTL delete namespace "$NAMESPACE" --wait=false 2>/dev/null && ok "Namespace deletion initiated" || warn "Namespace not found"

    ok "Cleanup complete"
}

# ──────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────
main() {
    echo ""
    echo "  MCP Server E2E Setup"
    echo "  ===================="
    echo ""

    case "${1:-}" in
        --deploy)
            preflight
            deploy_server
            run_smoke_test
            echo ""
            ok "Deploy + smoke test complete!"
            ;;
        --audit)
            preflight
            audit_cluster
            ;;
        --secrets)
            print_github_secrets
            ;;
        --set-secrets)
            set_github_secrets
            ;;
        --status)
            preflight
            check_status
            ;;
        --destroy)
            preflight
            destroy
            ;;
        --help|-h)
            cat <<HELPEOF
MCP Server E2E Setup

Usage:
  $(basename "$0") --deploy              Deploy MCP server + smoke test
  $(basename "$0") --audit               Audit cluster readiness
  $(basename "$0") --secrets             Print GitHub secrets needed for CI
  $(basename "$0") --set-secrets         Auto-set GitHub secrets via gh CLI
  $(basename "$0") --status              Check deployment status
  $(basename "$0") --destroy             Uninstall chart + delete namespace

Environment variables:
  NAMESPACE       Target namespace (default: self-healing-platform)
  HELM_RELEASE    Helm release name (default: mcp-server)
  IMAGE_REPO      Image repository (default: quay.io/takinosh/openshift-cluster-health-mcp)
  IMAGE_TAG       Image tag (default: ocp-4.22-latest)
  VALUES_FILE     Helm values file (default: values-ocp-4.22.yaml)
  SKIP_SMOKE_TEST Set to "true" to skip post-deploy smoke test
HELPEOF
            ;;
        "")
            echo "No mode specified. Use --help for usage."
            exit 1
            ;;
        *)
            echo "Unknown option: $1. Use --help for usage."
            exit 1
            ;;
    esac
}

main "$@"
