package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
)

// PredictDiskExhaustionTool forecasts filesystem full dates via CE ADR-018.
type PredictDiskExhaustionTool struct {
	ceClient *clients.CoordinationEngineClient
}

// NewPredictDiskExhaustionTool creates a new predict-disk-exhaustion tool.
func NewPredictDiskExhaustionTool(ceClient *clients.CoordinationEngineClient) *PredictDiskExhaustionTool {
	return &PredictDiskExhaustionTool{ceClient: ceClient}
}

func (t *PredictDiskExhaustionTool) Name() string { return "predict-disk-exhaustion" }

func (t *PredictDiskExhaustionTool) Description() string {
	return `Predict when node filesystems will run out of disk space using 7-day fill-rate trends.

WHAT THIS TOOL DOES:
- Calls CE GET /api/v1/predict/disk-exhaustion (ADR-018)
- Uses deriv() on node_filesystem_avail_bytes over a 7-day window
- Returns days-until-full per filesystem with urgency classification

RESPONSE INTERPRETATION:
- results[].days_until_full: Estimated days before filesystem reaches 100% capacity
- results[].urgency: "critical" (<7 days), "warning" (<30 days), "info" (<90 days), "stable"
- results[].projected_full_date: ISO date string of projected exhaustion
- results[].daily_fill_rate_bytes: Average bytes consumed per day
- critical_count / warning_count: Summary counts for quick triage

PRESENTATION TO USER:
- If critical_count > 0: Immediately highlight with node, mountpoint, and days remaining
- If warning_count > 0: Show projected full date so teams can plan capacity expansion
- If all stable: "All monitored filesystems have sufficient capacity."
- Always include the daily fill rate to help teams understand growth trends

Example questions this tool answers:
- "When will my disks run out of space?"
- "Are any nodes close to running out of disk space?"
- "Check /var/lib disk exhaustion on node worker-1"`
}

func (t *PredictDiskExhaustionTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"node": map[string]interface{}{
				"type":        "string",
				"description": "Node hostname to scope the prediction (optional, defaults to all nodes)",
			},
			"mountpoint": map[string]interface{}{
				"type":        "string",
				"description": "Filesystem mountpoint to filter (e.g., '/var/lib', '/', optional)",
			},
		},
		"required": []string{},
	}
}

// DiskExhaustionOutput is the tool response.
type DiskExhaustionOutput struct {
	Status        string                        `json:"status"`
	Node          string                        `json:"node,omitempty"`
	Mountpoint    string                        `json:"mountpoint,omitempty"`
	Results       []clients.DiskExhaustionResult `json:"results"`
	CriticalCount int                            `json:"critical_count"`
	WarningCount  int                            `json:"warning_count"`
	Message       string                         `json:"message"`
	Recommendation string                        `json:"recommendation"`
}

type predictDiskInput struct {
	Node       string `json:"node"`
	Mountpoint string `json:"mountpoint"`
}

func (t *PredictDiskExhaustionTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input := predictDiskInput{}
	if b, err := json.Marshal(args); err == nil {
		_ = json.Unmarshal(b, &input) //nolint:errcheck
	}

	ceResp, err := t.ceClient.PredictDiskExhaustion(ctx, input.Node, input.Mountpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk exhaustion prediction: %w", err)
	}

	out := DiskExhaustionOutput{
		Status:        ceResp.Status,
		Node:          input.Node,
		Mountpoint:    input.Mountpoint,
		Results:       ceResp.Results,
		CriticalCount: ceResp.CriticalCount,
		WarningCount:  ceResp.WarningCount,
	}

	switch {
	case ceResp.CriticalCount > 0:
		out.Message = fmt.Sprintf(
			"%d filesystem(s) will reach capacity within 7 days. Immediate action required.",
			ceResp.CriticalCount,
		)
		out.Recommendation = "Expand storage or free disk space on critical nodes immediately. " +
			"Consider enabling log rotation, cleaning up container images (podman system prune), " +
			"or adding persistent volume capacity."
	case ceResp.WarningCount > 0:
		out.Message = fmt.Sprintf(
			"%d filesystem(s) are projected to reach capacity within 30 days.",
			ceResp.WarningCount,
		)
		out.Recommendation = "Plan storage expansion for affected nodes. " +
			"Review application log retention policies and consider capacity forecasting with predict-resource-usage."
	default:
		out.Message = "All monitored filesystems have sufficient capacity based on current fill rates."
		out.Recommendation = "Continue monitoring. Re-run after significant workload changes."
	}

	return out, nil
}
