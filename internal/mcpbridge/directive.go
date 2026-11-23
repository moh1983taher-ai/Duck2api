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

	callPayload, hasCall, err := extractControlPayload(trimmed, callOpen, callClose)
	if err != nil {
		return Directive{}, err
	}
	finalPayload, hasFinal, err := extractControlPayload(trimmed, finalOpen, finalClose)
	if err != nil {
		return Directive{}, err
	}
	if hasCall && hasFinal {
		return Directive{}, fmt.Errorf("model output contains both MCP call and final control blocks")
	}

	if hasCall {
		var call struct {
			ServerLabel string         `json:"server_label"`
			ToolName    string         `json:"tool_name"`
			Arguments   map[string]any `json:"arguments"`
		}
		decoder := json.NewDecoder(strings.NewReader(callPayload))
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

	if hasFinal {
		return Directive{
			Kind: "final",
			Text: strings.TrimSpace(finalPayload),
			Raw:  trimmed,
		}, nil
	}

	// Models that do not emit the optional wrapper remain usable: plain text is
	// treated as a final answer, never as an executable tool request.
	return Directive{Kind: "final", Text: trimmed, Raw: trimmed}, nil
}

// extractControlPayload accepts harmless prose or Markdown surrounding one
// explicit control block. The payload itself remains strictly validated by
// ParseDirective; duplicate, mismatched, or unclosed markers are rejected.
func extractControlPayload(output, open, close string) (string, bool, error) {
	openCount := strings.Count(output, open)
	closeCount := strings.Count(output, close)
	if openCount == 0 && closeCount == 0 {
		return "", false, nil
	}
	if openCount != 1 || closeCount != 1 {
		return "", false, fmt.Errorf("malformed MCP control block markers")
	}
	start := strings.Index(output, open)
	payloadStart := start + len(open)
	relativeEnd := strings.Index(output[payloadStart:], close)
	if relativeEnd < 0 {
		return "", false, fmt.Errorf("MCP control block is missing its closing marker")
	}
	return strings.TrimSpace(output[payloadStart : payloadStart+relativeEnd]), true, nil
}
