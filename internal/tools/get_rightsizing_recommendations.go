package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
)

// GetRightSizingRecommendationsTool provides CPU/memory right-sizing deltas via CE ADR-019.
type GetRightSizingRecommendationsTool struct {
	ceClient *clients.CoordinationEngineClient
}

// NewGetRightSizingRecommendationsTool creates a new get-rightsizing-recommendations tool.
func NewGetRightSizingRecommendationsTool(ceClient *clients.CoordinationEngineClient) *GetRightSizingRecommendationsTool {
	return &GetRightSizingRecommendationsTool{ceClient: ceClient}
}

func (t *GetRightSizingRecommendationsTool) Name() string {
	return "get-rightsizing-recommendations"
}

func (t *GetRightSizingRecommendationsTool) Description() string {
	return `Recommend CPU and memory request/limit adjustments based on P95 historical usage.

WHAT THIS TOOL DOES:
- Calls CE GET /api/v1/recommendations/rightsizing (ADR-019)
- Compares P95 CPU/memory usage over a configurable window against current requests/limits
- Applies 20% headroom for requests, 50% headroom for limits
- Classifies each container as: over-provisioned, under-provisioned, or right-sized

RESPONSE INTERPRETATION:
- recommendations[].cpu_sizing: "over-provisioned" | "under-provisioned" | "right-sized"
- recommendations[].memory_sizing: same classification for memory
- recommendations[].recommended_cpu_request / recommended_cpu_limit: Suggested values
- recommendations[].recommended_memory_request / recommended_memory_limit: Suggested values
- recommendations[].p95_cpu_usage_cores / p95_memory_usage_bytes: Observed P95 values
- over_provisioned_count / under_provisioned_count / right_sized_count: Summary

PRESENTATION TO USER:
- For over-provisioned: Highlight cost savings from reducing requests
- For under-provisioned: Flag risk of OOMKills or throttling; suggest increasing limits
- For right-sized: Confirm current config is appropriate
- Always show the analysis_window so users understand the data age

Example questions this tool answers:
- "Are my pods over- or under-provisioned?"
- "What should I set for CPU/memory requests in the openshift-aiops namespace?"
- "Check right-sizing for the model-serving deployment"`
}

func (t *GetRightSizingRecommendationsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace to scope recommendations (optional, defaults to all namespaces)",
			},
			"pod": map[string]interface{}{
				"type":        "string",
				"description": "Specific pod name to analyse (optional)",
			},
			"window": map[string]interface{}{
				"type":        "string",
				"description": "Analysis window for historical data (e.g., '7d', '30d'). Defaults to '30d'.",
				"enum":        []string{"7d", "14d", "30d", "90d"},
				"default":     "30d",
			},
		},
		"required": []string{},
	}
}

// RightSizingOutput is the tool response.
type RightSizingOutput struct {
	Status           string                         `json:"status"`
	Namespace        string                         `json:"namespace,omitempty"`
	Pod              string                         `json:"pod,omitempty"`
	AnalysisWindow   string                         `json:"analysis_window"`
	Recommendations  []clients.ContainerRightSizingRec `json:"recommendations"`
	OverProvisioned  int                            `json:"over_provisioned_count"`
	UnderProvisioned int                            `json:"under_provisioned_count"`
	RightSized       int                            `json:"right_sized_count"`
	Message          string                         `json:"message"`
	Recommendation   string                         `json:"recommendation"`
}

type getRightSizingInput struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Window    string `json:"window"`
}

func (t *GetRightSizingRecommendationsTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input := getRightSizingInput{Window: "30d"}
	if b, err := json.Marshal(args); err == nil {
		_ = json.Unmarshal(b, &input) //nolint:errcheck
	}

	ceResp, err := t.ceClient.GetRightSizingRecommendations(ctx, input.Namespace, input.Pod, input.Window)
	if err != nil {
		return nil, fmt.Errorf("failed to get right-sizing recommendations: %w", err)
	}

	out := RightSizingOutput{
		Status:           ceResp.Status,
		Namespace:        input.Namespace,
		Pod:              input.Pod,
		AnalysisWindow:   ceResp.AnalysisWindow,
		Recommendations:  ceResp.Recommendations,
		OverProvisioned:  ceResp.OverProvisioned,
		UnderProvisioned: ceResp.UnderProvisioned,
		RightSized:       ceResp.RightSized,
	}

	total := ceResp.OverProvisioned + ceResp.UnderProvisioned + ceResp.RightSized
	if total == 0 {
		out.Message = "No containers found matching the filter criteria."
		out.Recommendation = "Verify the namespace/pod filter and ensure Prometheus metrics are available."
		return out, nil
	}

	parts := []string{}
	if ceResp.UnderProvisioned > 0 {
		parts = append(parts, fmt.Sprintf("%d under-provisioned (risk of OOMKill/throttling)", ceResp.UnderProvisioned))
	}
	if ceResp.OverProvisioned > 0 {
		parts = append(parts, fmt.Sprintf("%d over-provisioned (wasted resources)", ceResp.OverProvisioned))
	}
	if ceResp.RightSized > 0 {
		parts = append(parts, fmt.Sprintf("%d right-sized", ceResp.RightSized))
	}

	out.Message = fmt.Sprintf("Analysed %d container(s) over %s: %s.", total, ceResp.AnalysisWindow,
		joinStrings(parts, ", "))

	switch {
	case ceResp.UnderProvisioned > 0 && ceResp.OverProvisioned > 0:
		out.Recommendation = "Apply recommended limits to under-provisioned containers first to prevent OOMKills. " +
			"Then reduce over-provisioned requests to reclaim cluster capacity."
	case ceResp.UnderProvisioned > 0:
		out.Recommendation = "Increase CPU limits and memory limits for under-provisioned containers " +
			"to prevent throttling and OOMKills. Run get-throttled-pods to correlate with active throttling."
	case ceResp.OverProvisioned > 0:
		out.Recommendation = "Reduce CPU/memory requests for over-provisioned containers to improve bin-packing " +
			"and reduce cluster costs. Validate with load testing before applying to production."
	default:
		out.Recommendation = "All analysed containers are right-sized. No changes recommended at this time."
	}

	return out, nil
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
