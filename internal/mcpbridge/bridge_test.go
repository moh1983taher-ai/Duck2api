package mcpbridge

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseAllowedToolsNames(t *testing.T) {
	got, err := parseAllowedTools([]any{"search_books", "fetch_book"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.NamesSpecified || got.ReadOnly != nil || len(got.Names) != 2 || got.Names[0] != "search_books" {
		t.Fatalf("unexpected names filter: %+v", got)
	}
}

func TestParseAllowedToolsReadOnlyFilter(t *testing.T) {
	got, err := parseAllowedTools(map[string]any{"read_only": true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ReadOnly == nil || !*got.ReadOnly || got.NamesSpecified {
		t.Fatalf("unexpected read-only filter: %+v", got)
	}
}

func TestParseAllowedToolsCombinedFilter(t *testing.T) {
	got, err := parseAllowedTools(map[string]any{
		"read_only":  true,
		"tool_names": []any{"search_books", "fetch_book"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ReadOnly == nil || !*got.ReadOnly || !got.NamesSpecified || len(got.Names) != 2 {
		t.Fatalf("unexpected combined filter: %+v", got)
	}
}

func TestParseAllowedToolsRejectsInvalidFilter(t *testing.T) {
	for _, value := range []any{
		map[string]any{"read_only": "true"},
		map[string]any{"unknown": true},
		map[string]any{"tool_names": "search_books"},
	} {
		if _, err := parseAllowedTools(value); err == nil {
			t.Errorf("expected invalid allowed_tools filter to be rejected: %#v", value)
		}
	}
}

func TestReadOnlyPolicyMatchesResearchHost(t *testing.T) {
	readOnly := true
	destructive := true
	tests := []struct {
		name string
		tool *mcp.Tool
		want bool
	}{
		{name: "Aljam3 search tool", tool: &mcp.Tool{Name: "search_books"}, want: true},
		{name: "Shamela text retrieval", tool: &mcp.Tool{Name: "get_book_page"}, want: true},
		{name: "Arabic search tool", tool: &mcp.Tool{Name: "بحث_الكتب"}, want: true},
		{name: "write tool", tool: &mcp.Tool{Name: "update_book"}, want: false},
		{name: "delete tool", tool: &mcp.Tool{Name: "delete_book"}, want: false},
		{name: "unrelated tool", tool: &mcp.Tool{Name: "calculate_hash"}, want: false},
		{name: "explicit read-only annotation", tool: &mcp.Tool{Name: "search_books", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}}, want: true},
		{name: "non-read-only annotation", tool: &mcp.Tool{Name: "search_books", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false}}, want: false},
		{name: "destructive annotation", tool: &mcp.Tool{Name: "search_books", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &destructive}}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAllowedReadOnlyTool(test.tool); got != test.want {
				t.Fatalf("isAllowedReadOnlyTool(%q) = %v, want %v", test.tool.Name, got, test.want)
			}
		})
	}
}

func TestRegistryReadOnlyPolicyAndRequestFilterIntersect(t *testing.T) {
	registry := ServerConfig{AllowReadOnlyTools: true}
	search := &mcp.Tool{Name: "search_books"}
	fetch := &mcp.Tool{Name: "fetch_book_page"}
	write := &mcp.Tool{Name: "delete_book"}

	requested, err := parseAllowedTools(map[string]any{
		"read_only":  true,
		"tool_names": []any{"search_books"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !toolAllowedByPolicy(registry, requested, search) {
		t.Fatal("explicit read-only search tool should be allowed")
	}
	if toolAllowedByPolicy(registry, requested, fetch) {
		t.Fatal("request tool_names must narrow the server's read-only catalog")
	}
	if toolAllowedByPolicy(registry, requested, write) {
		t.Fatal("write tools must never pass the research read-only policy")
	}
}

func TestReadOnlyRegistryMustBeExplicit(t *testing.T) {
	cfg := ServerConfig{
		ServerLabel: "research",
		ServerURL:   "https://example.invalid/mcp",
	}
	if err := validateServerConfig(cfg); err == nil {
		t.Fatal("registry without explicit tool names or allow_read_only_tools must be rejected")
	}
	cfg.AllowReadOnlyTools = true
	if err := validateServerConfig(cfg); err != nil {
		t.Fatalf("explicit read-only policy should be accepted: %v", err)
	}
}

func TestParseRequestToolsDefaultsToApprovalRequired(t *testing.T) {
	tools := []any{
		map[string]any{
			"type":         "mcp",
			"server_label": "research",
			"server_url":   "https://example.invalid/mcp",
			"allowed_tools": []any{
				map[string]any{"name": "search"},
				"fetch",
			},
		},
	}
	requests, enabled, err := parseRequestTools(tools)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || len(requests) != 1 {
		t.Fatalf("unexpected parsed tools: enabled=%v requests=%+v", enabled, requests)
	}
	if requests[0].RequireApproval != "always" {
		t.Fatalf("omitted require_approval must default to always, got %q", requests[0].RequireApproval)
	}
	if !requests[0].AllowedTools.NamesSpecified || len(requests[0].AllowedTools.Names) != 2 {
		t.Fatalf("allowed_tools were not parsed: %+v", requests[0].AllowedTools)
	}
}

func TestParseRequestToolsExplicitNeverApproval(t *testing.T) {
	tools := []any{
		map[string]any{
			"type":             "mcp",
			"server_label":     "research",
			"server_url":       "https://example.invalid/mcp",
			"allowed_tools":    []any{"search"},
			"require_approval": "never",
		},
	}
	requests, enabled, err := parseRequestTools(tools)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || requests[0].RequireApproval != "never" {
		t.Fatalf("explicit never approval was not preserved: %+v", requests)
	}
}

func TestParseRequireApprovalDoesNotTrustFilterObjects(t *testing.T) {
	if got := parseRequireApproval(map[string]any{"never": map[string]any{"tool_names": []any{"search"}}}); got != "always" {
		t.Fatalf("unsupported approval value must remain approval-required, got %q", got)
	}
}

func TestParseToolChoice(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want ToolChoicePolicy
	}{
		{name: "auto", raw: "auto", want: ToolChoicePolicy{}},
		{name: "none", raw: "none", want: ToolChoicePolicy{Disabled: true}},
		{name: "required", raw: "required", want: ToolChoicePolicy{Required: true}},
		{name: "forced MCP tool", raw: map[string]any{
			"type": "mcp", "server_label": "research", "name": "search",
		}, want: ToolChoicePolicy{Required: true, ServerLabel: "research", ToolName: "search"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ParseToolChoice(test.raw)
			if got != test.want {
				t.Fatalf("ParseToolChoice(%#v) = %+v, want %+v", test.raw, got, test.want)
			}
		})
	}
}

func TestIntersectToolsOnlyReturnsConfiguredAllowList(t *testing.T) {
	got := intersectTools(
		[]string{"search", "fetch"},
		[]string{"search", "delete"},
	)
	if len(got) != 1 || got[0] != "search" {
		t.Fatalf("unexpected tool intersection: %#v", got)
	}
}

func TestExplicitEmptyAllowedToolsDisablesAllTools(t *testing.T) {
	requested := make([]string, 0)
	got := intersectTools([]string{"search", "fetch"}, requested)
	if len(got) != 0 {
		t.Fatalf("an explicit empty allowed_tools list must not grant access: %#v", got)
	}
}

func TestOmittedAllowedToolsUsesRegistryAllowList(t *testing.T) {
	got := intersectTools([]string{"search", "fetch"}, nil)
	if len(got) != 2 || got[0] != "search" || got[1] != "fetch" {
		t.Fatalf("omitted allowed_tools should preserve configured tools: %#v", got)
	}
}

func TestMCPHTTPClientDoesNotFollowRedirects(t *testing.T) {
	client := newServerHTTPClient(map[string]string{"Authorization": "Bearer test"})
	if client.CheckRedirect == nil {
		t.Fatal("MCP HTTP client must explicitly control redirect behavior")
	}
	err := client.CheckRedirect(&http.Request{}, nil)
	if err != http.ErrUseLastResponse {
		t.Fatalf("expected redirects to be blocked, got %v", err)
	}
}

// This integration test is intentionally opt-in. CI enables it to discover
// live public catalogs without executing a single research tool.
func TestLiveResearchServerDiscovery(t *testing.T) {
	if os.Getenv("MCP_LIVE_DISCOVERY") != "1" {
		t.Skip("set MCP_LIVE_DISCOVERY=1 to query the official MCP source catalogs")
	}
	raw := []any{
		map[string]any{
			"type": "mcp", "server_label": "aljam3",
			"server_url": "https://aljam3.com/mcp",
			"allowed_tools": map[string]any{"read_only": true},
			"require_approval": "always",
		},
		map[string]any{
			"type": "mcp", "server_label": "shamela",
			"server_url": "https://mcp.shamela.ws",
			"allowed_tools": map[string]any{"read_only": true},
			"require_approval": "always",
		},
	}
	bridge, enabled, err := Open(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || bridge == nil {
		t.Fatal("expected the live MCP bridge to be enabled")
	}
	defer bridge.Close()

	minimums := map[string]int{"aljam3": 7, "shamela": 4}
	for _, descriptor := range bridge.Descriptors() {
		names := make([]string, 0, len(descriptor.Tools))
		for _, tool := range descriptor.Tools {
			names = append(names, tool.Name)
		}
		t.Logf("%s: %d permitted read-only tools: %s", descriptor.ServerLabel, len(names), strings.Join(names, ", "))
		if len(names) < minimums[descriptor.ServerLabel] {
			t.Errorf("%s exposed %d permitted tools; expected at least %d", descriptor.ServerLabel, len(names), minimums[descriptor.ServerLabel])
		}
	}
	if len(bridge.Descriptors()) != 2 {
		t.Fatalf("both official MCP sources must connect; got %d source catalogs", len(bridge.Descriptors()))
	}
	encoded, _ := json.Marshal(bridge.Descriptors())
	t.Logf("discovered catalogs: %s", encoded)
}
