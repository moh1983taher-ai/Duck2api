package mcpbridge

import "testing"

func TestParseRequestToolsDefaultsToApprovalRequired(t *testing.T) {
	tools := []any{
		map[string]any{
			"type":        "mcp",
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
	if len(requests[0].AllowedTools) != 2 || requests[0].AllowedTools[0] != "search" || requests[0].AllowedTools[1] != "fetch" {
		t.Fatalf("allowed_tools were not parsed: %#v", requests[0].AllowedTools)
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
		t.Fatalf("unsupported filter object must remain approval-required, got %q", got)
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
