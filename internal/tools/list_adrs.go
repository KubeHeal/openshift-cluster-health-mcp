package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/cache"
)

const (
	adrCacheKey = "ce-adr-index"
	adrCacheTTL = 5 * time.Minute

	// GitHub API endpoint for the CE ADR README.
	defaultADRRepoOwner = "KubeHeal"
	defaultADRRepoName  = "openshift-coordination-engine"
	defaultADRFilePath  = "docs/adrs/README.md"
)

// adrTableRow matches a markdown table row: | [NNN](...) | Title | STATUS | Desc |
var adrTableRow = regexp.MustCompile(
	`\|\s*\[?(\d{3})\]?(?:\([^)]*\))?\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]*?)\s*\|`,
)

// ListADRsTool exposes the Coordination Engine's ADR index to AI agents.
type ListADRsTool struct {
	cache      *cache.MemoryCache
	httpClient *http.Client
	repoOwner  string
	repoName   string
	filePath   string
}

// ListADRsOption allows overriding defaults in tests.
type ListADRsOption func(*ListADRsTool)

// WithADRHTTPClient overrides the HTTP client used for GitHub API requests.
func WithADRHTTPClient(c *http.Client) ListADRsOption {
	return func(t *ListADRsTool) { t.httpClient = c }
}

// WithADRRepo overrides the GitHub owner/repo/path.
func WithADRRepo(owner, repo, path string) ListADRsOption {
	return func(t *ListADRsTool) {
		t.repoOwner = owner
		t.repoName = repo
		t.filePath = path
	}
}

// NewListADRsTool creates a new list-adrs tool.
func NewListADRsTool(c *cache.MemoryCache, opts ...ListADRsOption) *ListADRsTool {
	t := &ListADRsTool{
		cache:      c,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		repoOwner:  defaultADRRepoOwner,
		repoName:   defaultADRRepoName,
		filePath:   defaultADRFilePath,
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

func (t *ListADRsTool) Name() string { return "list-adrs" }

func (t *ListADRsTool) Description() string {
	return `List Architectural Decision Records (ADRs) from the Coordination Engine repository.

WHAT THIS TOOL DOES:
- Fetches the ADR index from the Coordination Engine's docs/adrs/README.md via GitHub API
- Parses the markdown table to extract ADR ID, title, status, and description
- Caches the result for 5 minutes to respect GitHub API rate limits

RESPONSE INTERPRETATION:
- adrs[]: Array of ADR entries with id, title, status, and description
- total_count: Total number of ADRs found
- filtered_count: Number of ADRs after applying the optional status filter

PRESENTATION TO USER:
- Display as a formatted list grouped by status
- Highlight PROPOSED ADRs as open decisions that may affect current work
- Reference ADR IDs when discussing architectural constraints

Example questions this tool answers:
- "What ADRs exist in the coordination engine?"
- "Show me all proposed ADRs"
- "What architectural decisions are implemented?"`
}

func (t *ListADRsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Filter ADRs by status (e.g., 'ACCEPTED', 'PROPOSED', 'IMPLEMENTED', 'DEPRECATED'). Case-insensitive. Omit to return all.",
			},
		},
		"required": []string{},
	}
}

// ADREntry represents a single ADR in the index.
type ADREntry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// ListADRsOutput is the tool response.
type ListADRsOutput struct {
	ADRs          []ADREntry `json:"adrs"`
	TotalCount    int        `json:"total_count"`
	FilteredCount int        `json:"filtered_count"`
	Source        string     `json:"source"`
	Message       string     `json:"message"`
}

func (t *ListADRsTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	statusFilter := ""
	if s, ok := args["status"].(string); ok && s != "" {
		statusFilter = strings.ToUpper(strings.TrimSpace(s))
	}

	adrs, source, err := t.getADRIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ADR index: %w", err)
	}

	filtered := adrs
	if statusFilter != "" {
		filtered = make([]ADREntry, 0, len(adrs))
		for _, a := range adrs {
			if strings.ToUpper(a.Status) == statusFilter {
				filtered = append(filtered, a)
			}
		}
	}

	msg := fmt.Sprintf("Found %d ADR(s) in the Coordination Engine repository.", len(filtered))
	if statusFilter != "" {
		msg = fmt.Sprintf("Found %d ADR(s) with status %q (out of %d total).", len(filtered), statusFilter, len(adrs))
	}

	return ListADRsOutput{
		ADRs:          filtered,
		TotalCount:    len(adrs),
		FilteredCount: len(filtered),
		Source:        source,
		Message:       msg,
	}, nil
}

// getADRIndex returns the parsed ADR list, using cache when available.
func (t *ListADRsTool) getADRIndex(ctx context.Context) ([]ADREntry, string, error) {
	if cached, ok := t.cache.Get(adrCacheKey); ok {
		if entries, ok := cached.([]ADREntry); ok {
			return entries, "cache", nil
		}
	}

	body, err := t.fetchReadme(ctx)
	if err != nil {
		return nil, "", err
	}

	entries := parseADRTable(body)
	if len(entries) == 0 {
		return nil, "github", fmt.Errorf("no ADR entries found in README — format may have changed")
	}

	t.cache.SetWithTTL(adrCacheKey, entries, adrCacheTTL)
	return entries, "github", nil
}

// fetchReadme retrieves the raw README content via the GitHub Contents API.
func (t *ListADRsTool) fetchReadme(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		t.repoOwner, t.repoName, t.filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ghResp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(raw, &ghResp); err != nil {
		return "", fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if ghResp.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(
			strings.ReplaceAll(ghResp.Content, "\n", ""),
		)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return string(decoded), nil
	}

	return ghResp.Content, nil
}

// parseADRTable extracts ADR entries from the markdown table in the README.
func parseADRTable(readme string) []ADREntry {
	var entries []ADREntry
	for _, match := range adrTableRow.FindAllStringSubmatch(readme, -1) {
		id := match[1]
		title := strings.TrimSpace(match[2])
		status := strings.TrimSpace(match[3])
		desc := strings.TrimSpace(match[4])

		// Skip header rows and reserved placeholders
		if strings.EqualFold(id, "ADR") || strings.EqualFold(title, "Title") {
			continue
		}
		if strings.Contains(title, "Reserved") || status == "-" {
			continue
		}

		entries = append(entries, ADREntry{
			ID:          fmt.Sprintf("ADR-%s", id),
			Title:       title,
			Status:      status,
			Description: desc,
		})
	}
	return entries
}
