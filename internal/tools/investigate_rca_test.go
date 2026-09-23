package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvestigateRCATool_Name(t *testing.T) {
	tool := &InvestigateRCATool{}
	assert.Equal(t, "investigate-rca", tool.Name())
}

func TestInvestigateRCATool_Description(t *testing.T) {
	tool := &InvestigateRCATool{}
	desc := tool.Description()
	assert.Contains(t, desc, "root-cause")
	assert.Contains(t, desc, "pod_event")
	assert.Contains(t, desc, "network_policy")
	assert.Contains(t, desc, "istio_virtual_service")
}

func TestInvestigateRCATool_InputSchema(t *testing.T) {
	tool := &InvestigateRCATool{}
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	for _, p := range []string{"service", "namespace", "time_range"} {
		_, exists := props[p]
		assert.True(t, exists, "property %s should exist", p)
	}

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "service")
	assert.Contains(t, required, "namespace")
}

func TestInvestigateRCATool_Execute_MissingService(t *testing.T) {
	tool := &InvestigateRCATool{}
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace": "default",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service is required")
}

func TestInvestigateRCATool_Execute_MissingNamespace(t *testing.T) {
	tool := &InvestigateRCATool{}
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"service": "api-gateway",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")
}

func TestInvestigateRCATool_Execute_SingleCorrelatorHit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RCAResponse{
			Status:    "ok",
			Service:   "order-service",
			Namespace: "payments",
			TimeRange: clients.RCATimeRange{Start: "2026-09-23T14:00:00Z", End: "2026-09-23T15:00:00Z"},
			RootCauses: []clients.RCARootCause{
				{
					SignalType:       "pod_event",
					Description:      "Container order-service OOMKilled 3 times in the last hour",
					Confidence:       0.92,
					Evidence:         map[string]interface{}{"event_count": 3, "reason": "OOMKilled"},
					RemediationSteps: []string{"Increase memory limit to 512Mi", "Check for memory leaks"},
				},
			},
			ConfidenceScore:    0.92,
			AffectedComponents: []string{"pod/order-service-abc-123"},
			IstioAvailable:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewInvestigateRCATool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"service":   "order-service",
		"namespace": "payments",
	})
	require.NoError(t, err)

	out, ok := result.(InvestigateRCAOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.RootCauseCount)
	assert.InDelta(t, 0.92, out.ConfidenceScore, 0.01)
	assert.Contains(t, out.Message, "1 root cause")
	assert.Contains(t, out.Message, "OOMKilled")
	assert.Contains(t, out.Recommendation, "Pod event findings")
	require.Len(t, out.RootCauses, 1)
	assert.Equal(t, "pod_event", out.RootCauses[0].SignalType)
	assert.Len(t, out.RootCauses[0].RemediationSteps, 2)
}

func TestInvestigateRCATool_Execute_MultipleCorrelators(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RCAResponse{
			Status:    "ok",
			Service:   "api-gateway",
			Namespace: "production",
			TimeRange: clients.RCATimeRange{Start: "2026-09-23T09:00:00Z", End: "2026-09-23T15:00:00Z"},
			RootCauses: []clients.RCARootCause{
				{
					SignalType:       "network_policy",
					Description:      "NetworkPolicy default-deny blocks ingress to api-gateway on port 8080",
					Confidence:       0.88,
					RemediationSteps: []string{"Add ingress rule for port 8080 from frontend namespace"},
				},
				{
					SignalType:       "istio_virtual_service",
					Description:      "VirtualService routes 100% traffic to non-existent subset v3",
					Confidence:       0.95,
					RemediationSteps: []string{"Update VirtualService to route to subset v2", "Create DestinationRule for v3"},
				},
				{
					SignalType:       "pod_event",
					Description:      "CrashLoopBackOff on container api-gateway",
					Confidence:       0.75,
					RemediationSteps: []string{"Check container logs"},
				},
			},
			ConfidenceScore:    0.95,
			AffectedComponents: []string{"vs/api-gateway", "netpol/default-deny", "pod/api-gateway-xyz"},
			IstioAvailable:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewInvestigateRCATool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"service":    "api-gateway",
		"namespace":  "production",
		"time_range": "6h",
	})
	require.NoError(t, err)

	out, ok := result.(InvestigateRCAOutput)
	require.True(t, ok)
	assert.Equal(t, 3, out.RootCauseCount)
	assert.InDelta(t, 0.95, out.ConfidenceScore, 0.01)
	assert.Contains(t, out.Message, "3 root cause")
	assert.Contains(t, out.Message, "95% confidence")
	assert.Contains(t, out.Message, "VirtualService routes") // top finding
	assert.Contains(t, out.Recommendation, "NetworkPolicy")
	assert.Contains(t, out.Recommendation, "Istio VirtualService")
	assert.Contains(t, out.Recommendation, "Pod event")
}

func TestInvestigateRCATool_Execute_NoFindings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RCAResponse{
			Status:             "ok",
			Service:            "healthy-service",
			Namespace:          "default",
			RootCauses:         []clients.RCARootCause{},
			ConfidenceScore:    0.0,
			AffectedComponents: []string{},
			IstioAvailable:     false,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewInvestigateRCATool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"service":   "healthy-service",
		"namespace": "default",
	})
	require.NoError(t, err)

	out, ok := result.(InvestigateRCAOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.RootCauseCount)
	assert.Contains(t, out.Message, "No root causes identified")
	assert.Contains(t, out.Recommendation, "broadening the time range")
	assert.Contains(t, out.Recommendation, "Istio was not available")
	assert.False(t, out.IstioAvailable)
}

func TestSubtractTimeRange(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"1h", true},
		{"6h", true},
		{"24h", true},
		{"3d", true},
		{"7d", true},
		{"30d", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		_, err := subtractTimeRange(timeNow(), tt.input)
		if tt.valid {
			assert.NoError(t, err, "time range %q should be valid", tt.input)
		} else {
			assert.Error(t, err, "time range %q should be invalid", tt.input)
		}
	}
}

func TestSignalTypeLabel(t *testing.T) {
	assert.Equal(t, "pod event(s)", signalTypeLabel("pod_event"))
	assert.Equal(t, "NetworkPolicy finding(s)", signalTypeLabel("network_policy"))
	assert.Equal(t, "Istio VirtualService finding(s)", signalTypeLabel("istio_virtual_service"))
	assert.Equal(t, "custom finding(s)", signalTypeLabel("custom"))
}

func TestRCASummary_NoFindings(t *testing.T) {
	input := investigateRCAInput{Service: "svc", Namespace: "ns", TimeRange: "1h"}
	msg, rec := rcaSummary(input, nil, true)
	assert.Contains(t, msg, "No root causes")
	assert.Contains(t, rec, "broadening")
	assert.NotContains(t, rec, "Istio was not available")
}

func TestRCASummary_NoFindingsNoIstio(t *testing.T) {
	input := investigateRCAInput{Service: "svc", Namespace: "ns", TimeRange: "1h"}
	msg, rec := rcaSummary(input, nil, false)
	assert.Contains(t, msg, "No root causes")
	assert.Contains(t, rec, "Istio was not available")
}

// timeNow is a helper for tests that need a stable time.
func timeNow() time.Time {
	return time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
}
