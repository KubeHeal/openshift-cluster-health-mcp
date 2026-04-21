package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
)

// GetThrottledPodsTool surfaces CPU throttle rates for pods via CE ADR-020 CFS metrics.
type GetThrottledPodsTool struct {
	ceClient *clients.CoordinationEngineClient
}

// NewGetThrottledPodsTool creates a new get-throttled-pods tool.
func NewGetThrottledPodsTool(ceClient *clients.CoordinationEngineClient) *GetThrottledPodsTool {
	return &GetThrottledPodsTool{ceClient: ceClient}
}

func (t *GetThrottledPodsTool) Name() string { return "get-throttled-pods" }

func (t *GetThrottledPodsTool) Description() string {
	return `Identify pods with high CPU throttling using CFS (Completely Fair Scheduler) metrics.

WHAT THIS TOOL DOES:
- Queries the Coordination Engine anomaly endpoint with CPU metrics
- Surfaces per-pod cpu_throttle_rate from enriched_signals (ADR-020 CFS metrics)
- A pod is considered throttled when cpu_throttle_rate > 25% of CFS periods

RESPONSE INTERPRETATION:
- throttled_pods[].throttle_rate_pct: Fraction of CFS periods throttled (0-100%)
- throttled_pods[].severity: "critical" (>75%), "high" (>50%), "medium" (>25%)
- throttled_pods[].recommendation: Specific action (increase CPU limit, rightsizing)
- total_throttled: Count of pods exceeding the threshold

PRESENTATION TO USER:
- If total_throttled=0: "No pods are experiencing CPU throttling above threshold."
- If total_throttled>0: List each pod with its throttle rate and recommendation
- Always suggest using get-rightsizing-recommendations for affected pods

Example questions this tool answers:
- "Which pods are being CPU throttled?"
- "Are any pods in the openshift-aiops namespace being throttled?"
- "Show me throttling issues across the cluster"`
}

func (t *GetThrottledPodsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace to scope the analysis (optional, defaults to cluster-wide)",
			},
			"deployment": map[string]interface{}{
				"type":        "string",
				"description": "Specific deployment name to filter (mutually exclusive with pod)",
			},
			"pod": map[string]interface{}{
				"type":        "string",
				"description": "Specific pod name to check throttling for (mutually exclusive with deployment)",
			},
			"threshold_pct": map[string]interface{}{
				"type":        "number",
				"description": "Throttle rate percentage threshold; pods above this are reported (default: 25.0)",
				"default":     25.0,
				"minimum":     0.0,
				"maximum":     100.0,
			},
		},
		"required": []string{},
	}
}

// ThrottledPodResult describes one pod exceeding the throttle threshold.
type ThrottledPodResult struct {
	Namespace      string   `json:"namespace,omitempty"`
	Pod            string   `json:"pod,omitempty"`
	Deployment     string   `json:"deployment,omitempty"`
	ThrottleRatePct float64 `json:"throttle_rate_pct"`
	Severity       string   `json:"severity"`
	Recommendation string   `json:"recommendation"`
}

// GetThrottledPodsOutput is the tool response.
type GetThrottledPodsOutput struct {
	Status         string               `json:"status"`
	Namespace      string               `json:"namespace,omitempty"`
	ThresholdPct   float64              `json:"threshold_pct"`
	ThrottledPods  []ThrottledPodResult `json:"throttled_pods"`
	TotalThrottled int                  `json:"total_throttled"`
	Message        string               `json:"message"`
	Recommendation string               `json:"recommendation"`
}

type getThrottledPodsInput struct {
	Namespace    string  `json:"namespace"`
	Deployment   string  `json:"deployment"`
	Pod          string  `json:"pod"`
	ThresholdPct float64 `json:"threshold_pct"`
}

func (t *GetThrottledPodsTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input := getThrottledPodsInput{ThresholdPct: 25.0}
	if b, err := json.Marshal(args); err == nil {
		_ = json.Unmarshal(b, &input) //nolint:errcheck
	}

	threshold := input.ThresholdPct / 100.0
	ceReq := &clients.AnalyzeAnomaliesRequest{
		TimeRange:     "1h",
		Metrics:       []interface{}{"cpu_usage"},
		Threshold:     &threshold,
		Namespace:     input.Namespace,
		Deployment:    input.Deployment,
		Pod:           input.Pod,
		LabelSelector: "",
	}

	ceResp, err := t.ceClient.AnalyzeAnomalies(ctx, ceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to query coordination engine: %w", err)
	}

	var throttled []ThrottledPodResult

	// Primary path: enriched_signals carries CFS-based throttle rate (CE v1.1.0 ADR-020).
	if ceResp.EnrichedSignals != nil && ceResp.EnrichedSignals.ThrottlingDetected {
		rate := 0.0
		if ceResp.EnrichedSignals.CPUThrottleRate != nil {
			rate = *ceResp.EnrichedSignals.CPUThrottleRate * 100.0
		}
		if rate >= input.ThresholdPct {
			throttled = append(throttled, ThrottledPodResult{
				Namespace:       input.Namespace,
				Pod:             input.Pod,
				Deployment:      input.Deployment,
				ThrottleRatePct: rate,
				Severity:        throttleSeverity(rate),
				Recommendation:  throttleRecommendation(rate, input.Deployment, input.Pod),
			})
		}
	}

	// Fallback path: scan anomaly patterns for per-pod throttle_rate metadata.
	for _, p := range ceResp.Patterns {
		if p.Metrics == nil {
			continue
		}
		rawRate, ok := p.Metrics["throttle_rate_pct"]
		if !ok {
			continue
		}
		ratePct := rawRate * 100.0
		if ratePct < input.ThresholdPct {
			continue
		}
		pod := ""
		ns := input.Namespace
		if parts := strings.SplitN(p.Explanation, "/", 2); len(parts) == 2 {
			ns = parts[0]
			pod = parts[1]
		}
		throttled = append(throttled, ThrottledPodResult{
			Namespace:       ns,
			Pod:             pod,
			ThrottleRatePct: ratePct,
			Severity:        throttleSeverity(ratePct),
			Recommendation:  throttleRecommendation(ratePct, "", pod),
		})
	}

	out := GetThrottledPodsOutput{
		Status:         ceResp.Status,
		Namespace:      input.Namespace,
		ThresholdPct:   input.ThresholdPct,
		ThrottledPods:  throttled,
		TotalThrottled: len(throttled),
	}

	if len(throttled) == 0 {
		out.Message = fmt.Sprintf("No pods are experiencing CPU throttling above %.0f%% threshold", input.ThresholdPct)
		out.Recommendation = "CPU throttling is within acceptable limits. Continue monitoring."
	} else {
		out.Message = fmt.Sprintf("%d pod(s) exceed the %.0f%% CPU throttle threshold", len(throttled), input.ThresholdPct)
		out.Recommendation = "Run get-rightsizing-recommendations for affected pods to compute recommended CPU limit increases."
	}

	return out, nil
}

func throttleSeverity(ratePct float64) string {
	switch {
	case ratePct >= 75:
		return "critical"
	case ratePct >= 50:
		return "high"
	default:
		return "medium"
	}
}

func throttleRecommendation(ratePct float64, deployment, pod string) string {
	target := pod
	if deployment != "" {
		target = "deployment " + deployment
	}
	if target == "" {
		target = "this workload"
	}
	if ratePct >= 75 {
		return fmt.Sprintf("CRITICAL: %s is throttled %.0f%% of the time. Increase CPU limit immediately or scale out.", target, ratePct)
	}
	return fmt.Sprintf("WARNING: %s CPU throttle rate is %.0f%%. Consider increasing CPU limit or running get-rightsizing-recommendations.", target, ratePct)
}
