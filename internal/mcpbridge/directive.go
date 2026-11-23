package mcpbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Directive struct {
	Kind        string
	ServerLabel string
	ToolName    string
	Arguments   map[string]any
	Text        string
	Raw         string
}

func ParseDirective(output string) (Directive, error) {
	trimmed := strings.TrimSpace(output)
	const callOpen = "<mcp_call>"
	const callClose = "</mcp_call>"
	const finalOpen = "<mcp_final>"
	const finalClose = "</mcp_final>"

	if strings.HasPrefix(trimmed, callOpen) {
		end := strings.Index(trimmed, callClose)
		if end < 0 || strings.TrimSpace(trimmed[end+len(callClose):]) != "" {
			return Directive{}, fmt.Errorf("malformed MCP call control block")
		}
		payload := strings.TrimSpace(trimmed[len(callOpen):end])
		var call struct {
			ServerLabel string         `json:"server_label"`
			ToolName    string         `json:"tool_name"`
			Arguments   map[string]any `json:"arguments"`
		}
		decoder := json.NewDecoder(strings.NewReader(payload))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&call); err != nil {
			return Directive{}, fmt.Errorf("invalid MCP call JSON: %w", err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return Directive{}, fmt.Errorf("MCP call JSON must contain exactly one object")
		}
		if strings.TrimSpace(call.ServerLabel) == "" || strings.TrimSpace(call.ToolName) == "" {
			return Directive{}, fmt.Errorf("MCP call requires server_label and tool_name")
		}
		if call.Arguments == nil {
			call.Arguments = map[string]any{}
		}
		return Directive{
			Kind:        "call",
			ServerLabel: call.ServerLabel,
			ToolName:    call.ToolName,
			Arguments:   call.Arguments,
			Raw:         trimmed,
		}, nil
	}

	if strings.HasPrefix(trimmed, finalOpen) {
		end := strings.Index(trimmed, finalClose)
		if end < 0 || strings.TrimSpace(trimmed[end+len(finalClose):]) != "" {
			return Directive{}, fmt.Errorf("malformed MCP final control block")
		}
		return Directive{
			Kind: "final",
			Text: strings.TrimSpace(trimmed[len(finalOpen):end]),
			Raw:  trimmed,
		}, nil
	}

	// Models that do not emit the optional wrapper remain usable: plain text is
	// treated as a final answer, never as an executable tool request.
	return Directive{Kind: "final", Text: trimmed, Raw: trimmed}, nil
}
