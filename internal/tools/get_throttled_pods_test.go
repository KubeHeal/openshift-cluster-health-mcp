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

func TestGetThrottledPodsTool_Name(t *testing.T) {
	tool := &GetThrottledPodsTool{}
	assert.Equal(t, "get-throttled-pods", tool.Name())
}

func TestGetThrottledPodsTool_Description(t *testing.T) {
	tool := &GetThrottledPodsTool{}
	desc := tool.Description()
	assert.Contains(t, desc, "throttl")
	assert.Contains(t, desc, "CFS")
	assert.Contains(t, desc, "cpu_throttle_rate")
}

func TestGetThrottledPodsTool_InputSchema(t *testing.T) {
	tool := &GetThrottledPodsTool{}
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	for _, prop := range []string{"namespace", "deployment", "pod", "threshold_pct"} {
		_, exists := properties[prop]
		assert.True(t, exists, "property %s should exist in schema", prop)
	}

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required, "no required fields")
}

func TestThrottleSeverity(t *testing.T) {
	tests := []struct {
		rate     float64
		expected string
	}{
		{80.0, "critical"},
		{75.0, "critical"},
		{60.0, "high"},
		{50.0, "high"},
		{30.0, "medium"},
		{25.0, "medium"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, throttleSeverity(tt.rate), "rate=%.0f%%", tt.rate)
	}
}

func TestThrottleRecommendation(t *testing.T) {
	rec := throttleRecommendation(80.0, "my-deploy", "")
	assert.Contains(t, rec, "CRITICAL")
	assert.Contains(t, rec, "deployment my-deploy")

	rec = throttleRecommendation(40.0, "", "my-pod")
	assert.Contains(t, rec, "WARNING")
	assert.Contains(t, rec, "my-pod")

	rec = throttleRecommendation(30.0, "", "")
	assert.Contains(t, rec, "this workload")
}

func TestGetThrottledPodsTool_Execute_NoThrottling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status":             "ok",
			"anomalies_detected": 0,
			"anomalies":          []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetThrottledPodsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace": "default",
	})
	require.NoError(t, err)

	out, ok := result.(GetThrottledPodsOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.TotalThrottled)
	assert.Contains(t, out.Message, "No pods")
	assert.Empty(t, out.ThrottledPods)
}

func TestGetThrottledPodsTool_Execute_WithEnrichedSignals(t *testing.T) {
	throttleRate := 0.65
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status":             "ok",
			"anomalies_detected": 1,
			"anomalies":          []interface{}{},
			"enriched_signals": map[string]interface{}{
				"cpu_throttle_rate":    throttleRate,
				"throttling_detected":  true,
				"http_error_rate":      nil,
				"http_response_time_p99_ms": nil,
				"http_degraded":        false,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetThrottledPodsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace":     "openshift-aiops",
		"pod":           "model-server-0",
		"threshold_pct": 25.0,
	})
	require.NoError(t, err)

	out, ok := result.(GetThrottledPodsOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.TotalThrottled)
	assert.Equal(t, "openshift-aiops", out.Namespace)
	require.Len(t, out.ThrottledPods, 1)
	assert.InDelta(t, 65.0, out.ThrottledPods[0].ThrottleRatePct, 0.1)
	assert.Equal(t, "high", out.ThrottledPods[0].Severity)
	assert.Contains(t, out.Message, "1 pod(s)")
	assert.Contains(t, out.Recommendation, "get-rightsizing-recommendations")
}

func TestGetThrottledPodsTool_Execute_PatternFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status":             "ok",
			"anomalies_detected": 2,
			"anomalies": []interface{}{
				map[string]interface{}{
					"explanation": "monitoring/prometheus-0",
					"metrics":    map[string]interface{}{"throttle_rate_pct": 0.40},
				},
				map[string]interface{}{
					"explanation": "monitoring/grafana-1",
					"metrics":    map[string]interface{}{"throttle_rate_pct": 0.10},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetThrottledPodsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"threshold_pct": 25.0,
	})
	require.NoError(t, err)

	out, ok := result.(GetThrottledPodsOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.TotalThrottled, "only the 40%% pod should exceed 25%% threshold")
	require.Len(t, out.ThrottledPods, 1)
	assert.Equal(t, "prometheus-0", out.ThrottledPods[0].Pod)
	assert.Equal(t, "monitoring", out.ThrottledPods[0].Namespace)
	assert.InDelta(t, 40.0, out.ThrottledPods[0].ThrottleRatePct, 0.1)
}

func TestGetThrottledPodsTool_Execute_CustomThreshold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status":             "ok",
			"anomalies_detected": 0,
			"anomalies":          []interface{}{},
			"enriched_signals": map[string]interface{}{
				"cpu_throttle_rate":   0.30,
				"throttling_detected": true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetThrottledPodsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"threshold_pct": 50.0,
	})
	require.NoError(t, err)

	out, ok := result.(GetThrottledPodsOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.TotalThrottled, "30%% rate should not exceed 50%% threshold")
}
