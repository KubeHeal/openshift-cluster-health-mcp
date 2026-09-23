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

func TestGetRightSizingRecommendationsTool_Name(t *testing.T) {
	tool := &GetRightSizingRecommendationsTool{}
	assert.Equal(t, "get-rightsizing-recommendations", tool.Name())
}

func TestGetRightSizingRecommendationsTool_Description(t *testing.T) {
	tool := &GetRightSizingRecommendationsTool{}
	desc := tool.Description()
	assert.Contains(t, desc, "right-siz")
	assert.Contains(t, desc, "P95")
	assert.Contains(t, desc, "over-provisioned")
}

func TestGetRightSizingRecommendationsTool_InputSchema(t *testing.T) {
	tool := &GetRightSizingRecommendationsTool{}
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	for _, prop := range []string{"namespace", "pod", "window"} {
		_, exists := properties[prop]
		assert.True(t, exists, "property %s should exist", prop)
	}

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required)
}

func TestJoinStrings(t *testing.T) {
	assert.Equal(t, "", joinStrings(nil, ", "))
	assert.Equal(t, "a", joinStrings([]string{"a"}, ", "))
	assert.Equal(t, "a, b, c", joinStrings([]string{"a", "b", "c"}, ", "))
}

func TestGetRightSizingRecommendationsTool_Execute_OverProvisioned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RightSizingResponse{
			Status:         "ok",
			AnalysisWindow: "30d",
			Recommendations: []clients.ContainerRightSizingRec{
				{
					Namespace:            "default",
					Pod:                  "web-app-1",
					Container:            "nginx",
					CurrentCPURequest:    "500m",
					CurrentCPULimit:      "1000m",
					P95CPUUsageCores:     0.05,
					RecommendedCPUReq:    "60m",
					RecommendedCPULimit:  "100m",
					CurrentMemoryRequest: "512Mi",
					CurrentMemoryLimit:   "1Gi",
					P95MemoryUsageBytes:  50e6,
					RecommendedMemoryReq: "60Mi",
					RecommendedMemoryLimit: "100Mi",
					CPUSizing:            "over-provisioned",
					MemorySizing:         "over-provisioned",
				},
			},
			OverProvisioned:  1,
			UnderProvisioned: 0,
			RightSized:       0,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetRightSizingRecommendationsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace": "default",
		"window":    "30d",
	})
	require.NoError(t, err)

	out, ok := result.(RightSizingOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.OverProvisioned)
	assert.Equal(t, 0, out.UnderProvisioned)
	assert.Contains(t, out.Message, "1 over-provisioned")
	assert.Contains(t, out.Recommendation, "Reduce CPU/memory requests")
	require.Len(t, out.Recommendations, 1)
	assert.Equal(t, "over-provisioned", out.Recommendations[0].CPUSizing)
}

func TestGetRightSizingRecommendationsTool_Execute_UnderProvisioned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RightSizingResponse{
			Status:         "ok",
			AnalysisWindow: "7d",
			Recommendations: []clients.ContainerRightSizingRec{
				{
					Namespace:            "ml",
					Pod:                  "model-server-0",
					Container:            "inference",
					CurrentCPURequest:    "100m",
					CurrentCPULimit:      "200m",
					P95CPUUsageCores:     0.95,
					RecommendedCPUReq:    "1140m",
					RecommendedCPULimit:  "1900m",
					CurrentMemoryRequest: "128Mi",
					CurrentMemoryLimit:   "256Mi",
					P95MemoryUsageBytes:  250e6,
					RecommendedMemoryReq: "300Mi",
					RecommendedMemoryLimit: "500Mi",
					CPUSizing:            "under-provisioned",
					MemorySizing:         "under-provisioned",
				},
			},
			OverProvisioned:  0,
			UnderProvisioned: 1,
			RightSized:       0,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetRightSizingRecommendationsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace": "ml",
		"window":    "7d",
	})
	require.NoError(t, err)

	out, ok := result.(RightSizingOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.OverProvisioned)
	assert.Equal(t, 1, out.UnderProvisioned)
	assert.Contains(t, out.Message, "1 under-provisioned")
	assert.Contains(t, out.Recommendation, "Increase CPU limits")
	assert.Contains(t, out.Recommendation, "get-throttled-pods")
}

func TestGetRightSizingRecommendationsTool_Execute_AllRightSized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RightSizingResponse{
			Status:         "ok",
			AnalysisWindow: "30d",
			Recommendations: []clients.ContainerRightSizingRec{
				{
					Namespace: "default",
					Pod:       "api-server",
					Container: "api",
					CPUSizing: "right-sized",
					MemorySizing: "right-sized",
				},
			},
			OverProvisioned:  0,
			UnderProvisioned: 0,
			RightSized:       1,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetRightSizingRecommendationsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)

	out, ok := result.(RightSizingOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.RightSized)
	assert.Contains(t, out.Message, "1 right-sized")
	assert.Contains(t, out.Recommendation, "No changes recommended")
}

func TestGetRightSizingRecommendationsTool_Execute_NoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RightSizingResponse{
			Status:          "ok",
			AnalysisWindow:  "30d",
			Recommendations: []clients.ContainerRightSizingRec{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetRightSizingRecommendationsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"namespace": "nonexistent",
	})
	require.NoError(t, err)

	out, ok := result.(RightSizingOutput)
	require.True(t, ok)
	assert.Contains(t, out.Message, "No containers found")
}

func TestGetRightSizingRecommendationsTool_Execute_Mixed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.RightSizingResponse{
			Status:         "ok",
			AnalysisWindow: "14d",
			Recommendations: []clients.ContainerRightSizingRec{
				{CPUSizing: "under-provisioned", MemorySizing: "under-provisioned"},
				{CPUSizing: "over-provisioned", MemorySizing: "over-provisioned"},
			},
			OverProvisioned:  1,
			UnderProvisioned: 1,
			RightSized:       0,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewGetRightSizingRecommendationsTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)

	out, ok := result.(RightSizingOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.OverProvisioned)
	assert.Equal(t, 1, out.UnderProvisioned)
	assert.Contains(t, out.Recommendation, "under-provisioned containers first")
}
