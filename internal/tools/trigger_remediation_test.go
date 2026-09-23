package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerRemediationTool_Name(t *testing.T) {
	tool := &TriggerRemediationTool{}
	assert.Equal(t, "trigger-remediation", tool.Name())
}

func TestTriggerRemediationTool_Description(t *testing.T) {
	tool := &TriggerRemediationTool{}
	desc := tool.Description()
	assert.Contains(t, desc, "remediation")
	assert.Contains(t, desc, "OOMKill")
	assert.Contains(t, desc, "memory limit")
}

func TestTriggerRemediationTool_InputSchema(t *testing.T) {
	tool := &TriggerRemediationTool{}
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	for _, p := range []string{"incident_id", "namespace", "resource_name", "resource_kind", "issue_type", "severity", "dry_run"} {
		_, exists := props[p]
		assert.True(t, exists, "property %s should exist", p)
	}

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "incident_id")
	assert.Contains(t, required, "namespace")
}

func TestTriggerRemediationTool_Execute_Validation(t *testing.T) {
	tool := &TriggerRemediationTool{}

	tests := []struct {
		name   string
		args   map[string]interface{}
		errMsg string
	}{
		{"missing incident_id", map[string]interface{}{"namespace": "x", "resource_name": "y", "resource_kind": "Pod", "issue_type": "pod_crash", "severity": "high"}, "incident_id is required"},
		{"missing namespace", map[string]interface{}{"incident_id": "1", "resource_name": "y", "resource_kind": "Pod", "issue_type": "pod_crash", "severity": "high"}, "namespace is required"},
		{"missing resource_name", map[string]interface{}{"incident_id": "1", "namespace": "x", "resource_kind": "Pod", "issue_type": "pod_crash", "severity": "high"}, "resource_name is required"},
		{"missing resource_kind", map[string]interface{}{"incident_id": "1", "namespace": "x", "resource_name": "y", "issue_type": "pod_crash", "severity": "high"}, "resource_kind is required"},
		{"missing issue_type", map[string]interface{}{"incident_id": "1", "namespace": "x", "resource_name": "y", "resource_kind": "Pod", "severity": "high"}, "issue_type is required"},
		{"missing severity", map[string]interface{}{"incident_id": "1", "namespace": "x", "resource_name": "y", "resource_kind": "Pod", "issue_type": "pod_crash"}, "severity is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), tt.args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func ceRemediationServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.TriggerRemediationResponse{
			WorkflowID:        "wf-abc-123",
			Status:            "triggered",
			DeploymentMethod:  "Helm",
			EstimatedDuration: "30s",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestTriggerRemediationTool_Execute_StandardRemediation(t *testing.T) {
	srv := ceRemediationServer()
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewTriggerRemediationTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"incident_id":   "INC-001",
		"namespace":     "production",
		"resource_name": "api-server",
		"resource_kind": "Deployment",
		"issue_type":    "high_cpu",
		"severity":      "high",
		"description":   "CPU at 95% for 10 minutes",
	})
	require.NoError(t, err)

	out, ok := result.(TriggerRemediationOutput)
	require.True(t, ok)
	assert.Equal(t, "triggered", out.Status)
	assert.Equal(t, "wf-abc-123", out.WorkflowID)
	assert.Equal(t, "INC-001", out.IncidentID)
	assert.Contains(t, out.Message, "Remediation triggered successfully")
	assert.False(t, out.OOMKillDetected)
	assert.Empty(t, out.OOMKillAdvice)
}

func TestTriggerRemediationTool_Execute_OOMKillRemediation(t *testing.T) {
	srv := ceRemediationServer()
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewTriggerRemediationTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"incident_id":   "INC-002",
		"namespace":     "ml-serving",
		"resource_name": "model-server",
		"resource_kind": "Deployment",
		"issue_type":    "oom_kill",
		"severity":      "critical",
		"description":   "Container OOMKilled 3 times",
	})
	require.NoError(t, err)

	out, ok := result.(TriggerRemediationOutput)
	require.True(t, ok)
	assert.True(t, out.OOMKillDetected)
	assert.Contains(t, out.OOMKillAdvice, "OOMKill detected")
	assert.Contains(t, out.OOMKillAdvice, "memory limit patching")
	assert.Contains(t, out.OOMKillAdvice, "get-rightsizing-recommendations")
	assert.Contains(t, out.OOMKillAdvice, "Deployment/model-server")
	assert.Contains(t, out.OOMKillAdvice, "ml-serving")
	assert.Contains(t, out.Message, "OOMKill")
	assert.Contains(t, out.Message, "get-rightsizing-recommendations")
}

func TestTriggerRemediationTool_Execute_DryRun(t *testing.T) {
	srv := ceRemediationServer()
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewTriggerRemediationTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"incident_id":   "INC-003",
		"namespace":     "default",
		"resource_name": "web-app",
		"resource_kind": "Deployment",
		"issue_type":    "pod_crash",
		"severity":      "medium",
		"dry_run":       true,
	})
	require.NoError(t, err)

	out, ok := result.(TriggerRemediationOutput)
	require.True(t, ok)
	assert.True(t, out.DryRun)
	assert.Contains(t, out.Message, "DRY RUN")
}

func TestTriggerRemediationTool_Execute_OOMKillDryRun(t *testing.T) {
	srv := ceRemediationServer()
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewTriggerRemediationTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"incident_id":   "INC-004",
		"namespace":     "staging",
		"resource_name": "worker",
		"resource_kind": "StatefulSet",
		"issue_type":    "oom_kill",
		"severity":      "high",
		"dry_run":       true,
	})
	require.NoError(t, err)

	out, ok := result.(TriggerRemediationOutput)
	require.True(t, ok)
	assert.True(t, out.DryRun)
	assert.True(t, out.OOMKillDetected)
	assert.Contains(t, out.OOMKillAdvice, "StatefulSet/worker")
}
