package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/clients"
)

// InvestigateRCATool performs deep root-cause analysis via CE v1.2.0 (ADR-021).
type InvestigateRCATool struct {
	ceClient *clients.CoordinationEngineClient
}

// NewInvestigateRCATool creates a new investigate-rca tool.
func NewInvestigateRCATool(ceClient *clients.CoordinationEngineClient) *InvestigateRCATool {
	return &InvestigateRCATool{ceClient: ceClient}
}

func (t *InvestigateRCATool) Name() string { return "investigate-rca" }

func (t *InvestigateRCATool) Description() string {
	return `Perform deep root-cause analysis by correlating pod events, NetworkPolicy changes, and Istio VirtualService misconfigurations.

WHAT THIS TOOL DOES:
- Calls CE POST /api/v1/investigate/rca (ADR-021, Deep RCA v2)
- Runs three parallel correlators: pod event timeline, NetworkPolicy audit, Istio VS check
- Returns ranked root causes with confidence scores and remediation steps

RESPONSE INTERPRETATION:
- root_causes[]: Array of findings from correlators, ordered by confidence
- root_causes[].signal_type: "pod_event" | "network_policy" | "istio_virtual_service"
- root_causes[].confidence: 0.0–1.0 confidence for this specific finding
- confidence_score: Overall investigation confidence (0.0–1.0)
- affected_components[]: Kubernetes resources implicated
- istio_available: Whether Istio CRDs were accessible for analysis
- correlator_stats[]: Per-correlator execution time and finding count

PRESENTATION TO USER:
- Lead with the highest-confidence root cause
- Group findings by signal_type for clarity
- Include remediation_steps verbatim — they are specific to the finding
- If istio_available=false, note that Istio analysis was skipped (cluster may not have Istio)
- If no root_causes found, suggest broadening the time range

Example questions this tool answers:
- "Why is my pod crashing?"
- "Investigate the root cause of failures in the payments namespace"
- "What caused the outage for order-service in the last hour?"`
}

func (t *InvestigateRCATool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"service": map[string]interface{}{
				"type":        "string",
				"description": "Service or deployment name to investigate (e.g., 'order-service', 'api-gateway')",
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace where the service runs",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "How far back to investigate (e.g., '1h', '6h', '24h', '3d'). Defaults to '1h'.",
				"enum":        []string{"1h", "6h", "24h", "3d", "7d"},
				"default":     "1h",
			},
		},
		"required": []string{"service", "namespace"},
	}
}

// RCAFindingOutput represents a single root-cause finding.
type RCAFindingOutput struct {
	SignalType       string                 `json:"signal_type"`
	Description      string                 `json:"description"`
	Confidence       float64                `json:"confidence"`
	Evidence         map[string]interface{} `json:"evidence,omitempty"`
	RemediationSteps []string               `json:"remediation_steps"`
}

// InvestigateRCAOutput is the tool response.
type InvestigateRCAOutput struct {
	Status             string             `json:"status"`
	Service            string             `json:"service"`
	Namespace          string             `json:"namespace"`
	TimeRange          string             `json:"time_range"`
	RootCauses         []RCAFindingOutput `json:"root_causes"`
	RootCauseCount     int                `json:"root_cause_count"`
	ConfidenceScore    float64            `json:"confidence_score"`
	AffectedComponents []string           `json:"affected_components"`
	IstioAvailable     bool               `json:"istio_available"`
	Message            string             `json:"message"`
	Recommendation     string             `json:"recommendation"`
}

type investigateRCAInput struct {
	Service   string `json:"service"`
	Namespace string `json:"namespace"`
	TimeRange string `json:"time_range"`
}

func (t *InvestigateRCATool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input := investigateRCAInput{TimeRange: "1h"}
	if b, err := json.Marshal(args); err == nil {
		_ = json.Unmarshal(b, &input) //nolint:errcheck
	}

	if input.Service == "" {
		return nil, fmt.Errorf("service is required")
	}
	if input.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	now := time.Now().UTC()
	start, err := subtractTimeRange(now, input.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("invalid time_range %q: %w", input.TimeRange, err)
	}

	ceReq := &clients.RCARequest{
		Service:   input.Service,
		Namespace: input.Namespace,
		StartTime: start.Format(time.RFC3339),
		EndTime:   now.Format(time.RFC3339),
	}

	ceResp, err := t.ceClient.InvestigateRCA(ctx, ceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to investigate root cause: %w", err)
	}

	findings := make([]RCAFindingOutput, 0, len(ceResp.RootCauses))
	for _, rc := range ceResp.RootCauses {
		findings = append(findings, RCAFindingOutput{
			SignalType:       rc.SignalType,
			Description:      rc.Description,
			Confidence:       rc.Confidence,
			Evidence:         rc.Evidence,
			RemediationSteps: rc.RemediationSteps,
		})
	}

	out := InvestigateRCAOutput{
		Status:             ceResp.Status,
		Service:            input.Service,
		Namespace:          input.Namespace,
		TimeRange:          input.TimeRange,
		RootCauses:         findings,
		RootCauseCount:     len(findings),
		ConfidenceScore:    ceResp.ConfidenceScore,
		AffectedComponents: ceResp.AffectedComponents,
		IstioAvailable:     ceResp.IstioAvailable,
	}

	out.Message, out.Recommendation = rcaSummary(input, findings, ceResp.IstioAvailable)
	return out, nil
}

// rcaSummary builds natural-language message and recommendation.
func rcaSummary(input investigateRCAInput, findings []RCAFindingOutput, istioAvailable bool) (string, string) {
	if len(findings) == 0 {
		msg := fmt.Sprintf("No root causes identified for service %q in namespace %q over the last %s.",
			input.Service, input.Namespace, input.TimeRange)
		rec := "Try broadening the time range (e.g., '6h' or '24h') or check if the service name is correct."
		if !istioAvailable {
			rec += " Note: Istio was not available — network-layer analysis was limited."
		}
		return msg, rec
	}

	// Categorise by signal type
	byType := map[string]int{}
	var topDesc string
	var topConf float64
	for _, f := range findings {
		byType[f.SignalType]++
		if f.Confidence > topConf {
			topConf = f.Confidence
			topDesc = f.Description
		}
	}

	parts := []string{}
	for st, n := range byType {
		parts = append(parts, fmt.Sprintf("%d %s", n, signalTypeLabel(st)))
	}

	msg := fmt.Sprintf("Found %d root cause(s) for %s/%s over the last %s: %s. Top finding (%.0f%% confidence): %s",
		len(findings), input.Namespace, input.Service, input.TimeRange,
		strings.Join(parts, ", "), topConf*100, topDesc)

	if !istioAvailable {
		msg += " (Istio analysis unavailable)"
	}

	rec := "Review the remediation_steps for each root cause. "
	if byType["pod_event"] > 0 {
		rec += "Pod event findings may indicate crashloops or OOMKills — check container logs and resource limits. "
	}
	if byType["network_policy"] > 0 {
		rec += "NetworkPolicy findings suggest traffic is being blocked — verify policy selectors and ports. "
	}
	if byType["istio_virtual_service"] > 0 {
		rec += "Istio VirtualService findings may indicate routing misconfiguration — check destination rules and weights. "
	}

	return msg, strings.TrimSpace(rec)
}

// signalTypeLabel returns a human-friendly label for signal types.
func signalTypeLabel(st string) string {
	switch st {
	case "pod_event":
		return "pod event(s)"
	case "network_policy":
		return "NetworkPolicy finding(s)"
	case "istio_virtual_service":
		return "Istio VirtualService finding(s)"
	default:
		return st + " finding(s)"
	}
}

// subtractTimeRange computes start = now - range.
func subtractTimeRange(now time.Time, tr string) (time.Time, error) {
	switch tr {
	case "1h":
		return now.Add(-1 * time.Hour), nil
	case "6h":
		return now.Add(-6 * time.Hour), nil
	case "24h":
		return now.Add(-24 * time.Hour), nil
	case "3d":
		return now.Add(-3 * 24 * time.Hour), nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time range: %s", tr)
	}
}
