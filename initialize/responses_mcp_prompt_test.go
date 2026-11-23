package initialize

import (
	"strings"
	"testing"

	officialtypes "aurora/typings/official"
)

func TestPrepareMCPMessagesMergesSystemAndProtocolIntoTask(t *testing.T) {
	input := []officialtypes.ApiMessage{
		{Role: "system", Content: "Caller instructions"},
		{Role: "user", Content: "Research question"},
	}
	got := prepareMCPMessages(input, "MCP protocol")
	if len(got) != 1 {
		t.Fatalf("expected one task message, got %d: %#v", len(got), got)
	}
	if got[0].Role != "user" {
		t.Fatalf("expected user role, got %q", got[0].Role)
	}
	text, ok := got[0].Content.(string)
	if !ok {
		t.Fatalf("expected string content, got %T", got[0].Content)
	}
	for _, want := range []string{"Research question", "Caller instructions", "MCP protocol"} {
		if !strings.Contains(text, want) {
			t.Errorf("merged task message does not contain %q: %s", want, text)
		}
	}
}

func TestPrepareMCPMessagesPreservesMultimodalParts(t *testing.T) {
	input := []officialtypes.ApiMessage{
		{Role: "user", Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "Read this image"},
			map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "data:image/png;base64,AA=="}},
		}},
	}
	got := prepareMCPMessages(input, "MCP protocol")
	if len(got) != 1 {
		t.Fatalf("expected one message, got %d", len(got))
	}
	parts, ok := got[0].Content.([]interface{})
	if !ok {
		t.Fatalf("expected multimodal parts, got %T", got[0].Content)
	}
	if len(parts) != 3 {
		t.Fatalf("expected original two parts plus protocol text, got %d", len(parts))
	}
	last, ok := parts[2].(map[string]interface{})
	if !ok || last["type"] != "text" || last["text"] != "MCP protocol" {
		t.Fatalf("protocol text part was not appended correctly: %#v", parts[2])
	}
}

func TestPrepareMCPMessagesAddsUserMessageWhenMissing(t *testing.T) {
	input := []officialtypes.ApiMessage{{Role: "assistant", Content: "Previous assistant turn"}}
	got := prepareMCPMessages(input, "MCP protocol")
	if len(got) != 2 || got[1].Role != "user" || got[1].Content != "MCP protocol" {
		t.Fatalf("expected protocol as final user message, got %#v", got)
	}
}
