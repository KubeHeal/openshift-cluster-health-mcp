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

func TestPredictDiskExhaustionTool_Name(t *testing.T) {
	tool := &PredictDiskExhaustionTool{}
	assert.Equal(t, "predict-disk-exhaustion", tool.Name())
}

func TestPredictDiskExhaustionTool_Description(t *testing.T) {
	tool := &PredictDiskExhaustionTool{}
	desc := tool.Description()
	assert.Contains(t, desc, "disk")
	assert.Contains(t, desc, "days_until_full")
	assert.Contains(t, desc, "urgency")
}

func TestPredictDiskExhaustionTool_InputSchema(t *testing.T) {
	tool := &PredictDiskExhaustionTool{}
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	_, hasNode := properties["node"]
	_, hasMount := properties["mountpoint"]
	assert.True(t, hasNode)
	assert.True(t, hasMount)

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required)
}

func TestPredictDiskExhaustionTool_Execute_Critical(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.DiskExhaustionResponse{
			Status:        "ok",
			CriticalCount: 1,
			WarningCount:  0,
			Results: []clients.DiskExhaustionResult{
				{
					Node:               "worker-1",
					Mountpoint:         "/var/lib",
					AvailableBytes:     5e9,
					TotalBytes:         100e9,
					UsedPercent:        95.0,
					DailyFillRateBytes: 1e9,
					DaysUntilFull:      5,
					Urgency:            "critical",
					ProjectedFullDate:  "2026-09-28",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewPredictDiskExhaustionTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"node":       "worker-1",
		"mountpoint": "/var/lib",
	})
	require.NoError(t, err)

	out, ok := result.(DiskExhaustionOutput)
	require.True(t, ok)
	assert.Equal(t, 1, out.CriticalCount)
	assert.Equal(t, 0, out.WarningCount)
	assert.Contains(t, out.Message, "7 days")
	assert.Contains(t, out.Message, "Immediate action")
	assert.Contains(t, out.Recommendation, "Expand storage")
	require.Len(t, out.Results, 1)
	assert.Equal(t, 5, out.Results[0].DaysUntilFull)
	assert.Equal(t, "critical", out.Results[0].Urgency)
}

func TestPredictDiskExhaustionTool_Execute_Warning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.DiskExhaustionResponse{
			Status:        "ok",
			CriticalCount: 0,
			WarningCount:  1,
			Results: []clients.DiskExhaustionResult{
				{
					Node:               "worker-2",
					Mountpoint:         "/",
					AvailableBytes:     20e9,
					TotalBytes:         100e9,
					UsedPercent:        80.0,
					DailyFillRateBytes: 1e9,
					DaysUntilFull:      20,
					Urgency:            "warning",
					ProjectedFullDate:  "2026-10-13",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewPredictDiskExhaustionTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)

	out, ok := result.(DiskExhaustionOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.CriticalCount)
	assert.Equal(t, 1, out.WarningCount)
	assert.Contains(t, out.Message, "30 days")
	assert.Contains(t, out.Recommendation, "Plan storage expansion")
}

func TestPredictDiskExhaustionTool_Execute_Stable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := clients.DiskExhaustionResponse{
			Status:        "ok",
			CriticalCount: 0,
			WarningCount:  0,
			Results: []clients.DiskExhaustionResult{
				{
					Node:          "worker-1",
					Mountpoint:    "/",
					UsedPercent:   40.0,
					DaysUntilFull: 365,
					Urgency:       "stable",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	ceClient := clients.NewCoordinationEngineClient(srv.URL)
	tool := NewPredictDiskExhaustionTool(ceClient)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)

	out, ok := result.(DiskExhaustionOutput)
	require.True(t, ok)
	assert.Equal(t, 0, out.CriticalCount)
	assert.Equal(t, 0, out.WarningCount)
	assert.Contains(t, out.Message, "sufficient capacity")
	assert.Contains(t, out.Recommendation, "Continue monitoring")
}
