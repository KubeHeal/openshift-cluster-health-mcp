package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KubeHeal/openshift-cluster-health-mcp/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleADRReadme = `# Architectural Decision Records (ADRs)

## ADR Index

### Implementation ADRs (Local)

| ADR | Title | Status | Description |
|-----|-------|--------|-------------|
| [001](001-go-project-architecture.md) | Go Project Architecture and Standards | ACCEPTED | Go version, project layout |
| [002](002-deployment-detection.md) | Deployment Detection Implementation | ACCEPTED | Deployment method detection |
| [003](003-multi-layer-coordination.md) | Multi-Layer Coordination | IMPLEMENTED | Layer detection, planner |
| 007-010 | *(Reserved - see note below)* | - | Reserved for future use |
| [021](021-deep-rca-v2.md) | Deep RCA v2 Multi-Signal Correlation | IMPLEMENTED | Three parallel correlators |
| [023](023-future-feature.md) | Future Feature Proposal | PROPOSED | A feature that is still under review |
`

func githubContentResponse(raw string) []byte {
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	resp := struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}{Content: encoded, Encoding: "base64"}
	b, _ := json.Marshal(resp)
	return b
}

func TestListADRsTool_Name(t *testing.T) {
	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	assert.Equal(t, "list-adrs", tool.Name())
}

func TestListADRsTool_Description(t *testing.T) {
	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	desc := tool.Description()
	assert.Contains(t, desc, "ADR")
	assert.Contains(t, desc, "Coordination Engine")
}

func TestListADRsTool_InputSchema(t *testing.T) {
	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	schema := tool.InputSchema()

	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	_, hasStatus := props["status"]
	assert.True(t, hasStatus)

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required)
}

func TestParseADRTable(t *testing.T) {
	entries := parseADRTable(sampleADRReadme)

	require.Len(t, entries, 5)

	assert.Equal(t, "ADR-001", entries[0].ID)
	assert.Equal(t, "Go Project Architecture and Standards", entries[0].Title)
	assert.Equal(t, "ACCEPTED", entries[0].Status)

	assert.Equal(t, "ADR-003", entries[2].ID)
	assert.Equal(t, "IMPLEMENTED", entries[2].Status)

	assert.Equal(t, "ADR-023", entries[4].ID)
	assert.Equal(t, "PROPOSED", entries[4].Status)
}

func TestParseADRTable_SkipsReserved(t *testing.T) {
	entries := parseADRTable(sampleADRReadme)
	for _, e := range entries {
		assert.NotContains(t, e.Title, "Reserved")
	}
}

func TestListADRsTool_Execute_NoFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(githubContentResponse(sampleADRReadme))
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c,
		WithADRHTTPClient(srv.Client()),
		WithADRRepo("test", "repo", "docs/adrs/README.md"),
	)

	// Override the fetchReadme URL by replacing the httpClient with one that routes to our test server
	tool.httpClient = srv.Client()
	// We need to override the URL, so let's set up a custom transport
	tool.httpClient.Transport = rewriteTransport{base: http.DefaultTransport, target: srv.URL}

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)

	out, ok := result.(ListADRsOutput)
	require.True(t, ok)
	assert.Equal(t, 5, out.TotalCount)
	assert.Equal(t, 5, out.FilteredCount)
	assert.Equal(t, "github", out.Source)
	assert.Contains(t, out.Message, "5 ADR(s)")
}

func TestListADRsTool_Execute_StatusFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(githubContentResponse(sampleADRReadme))
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	tool.httpClient = &http.Client{Transport: rewriteTransport{base: http.DefaultTransport, target: srv.URL}}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"status": "proposed",
	})
	require.NoError(t, err)

	out, ok := result.(ListADRsOutput)
	require.True(t, ok)
	assert.Equal(t, 5, out.TotalCount)
	assert.Equal(t, 1, out.FilteredCount)
	require.Len(t, out.ADRs, 1)
	assert.Equal(t, "ADR-023", out.ADRs[0].ID)
	assert.Contains(t, out.Message, "PROPOSED")
}

func TestListADRsTool_Execute_CacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(githubContentResponse(sampleADRReadme))
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	tool.httpClient = &http.Client{Transport: rewriteTransport{base: http.DefaultTransport, target: srv.URL}}

	// First call fetches from GitHub
	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call should hit cache
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "second call should use cache")

	out, ok := result.(ListADRsOutput)
	require.True(t, ok)
	assert.Equal(t, "cache", out.Source)
}

func TestListADRsTool_Execute_StatusFilterImplemented(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(githubContentResponse(sampleADRReadme))
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(30 * time.Second)
	tool := NewListADRsTool(c)
	tool.httpClient = &http.Client{Transport: rewriteTransport{base: http.DefaultTransport, target: srv.URL}}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"status": "IMPLEMENTED",
	})
	require.NoError(t, err)

	out, ok := result.(ListADRsOutput)
	require.True(t, ok)
	assert.Equal(t, 2, out.FilteredCount)
	for _, a := range out.ADRs {
		assert.Equal(t, "IMPLEMENTED", a.Status)
	}
}

// rewriteTransport rewrites request URLs to point at the test server.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = ""
	req.URL.Path = "/"
	fullURL := t.target + "/"
	newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, fullURL, req.Body)
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return t.base.RoundTrip(newReq)
}
