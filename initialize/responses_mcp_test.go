package initialize

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	officialtypes "aurora/typings/official"

	"github.com/gin-gonic/gin"
)

func TestWriteResponsesMCPStreamEmitsMCPEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	completed := officialtypes.NewResponseAPIFull("Final answer", "gpt-6-luna", 0, 0, 0, 0, 0, "")
	completed.Output = []officialtypes.ResponseOutput{
		{
			ID:          "mcpl_test",
			Type:        "mcp_list_tools",
			Status:      "completed",
			ServerLabel: "research",
		},
		{
			ID:          "mcp_test",
			Type:        "mcp_call",
			Status:      "completed",
			ServerLabel: "research",
			Name:        "search",
			Arguments:   `{"query":"test"}`,
			Output:      `{"items":[]}`,
		},
		officialtypes.NewResponseOutput("Final answer"),
	}

	writeResponsesMCPStream(ctx, completed)
	body := recorder.Body.String()
	for _, eventType := range []string{
		"response.created",
		"response.output_item.added",
		"response.mcp_list_tools.in_progress",
		"response.mcp_list_tools.completed",
		"response.mcp_call_arguments.delta",
		"response.mcp_call_arguments.done",
		"response.mcp_call.in_progress",
		"response.mcp_call.completed",
		"response.output_item.done",
		"response.output_text.delta",
		"response.completed",
	} {
		if !strings.Contains(body, "event: "+eventType+"\n") {
			t.Errorf("missing SSE event %q in stream:\n%s", eventType, body)
		}
	}
	if !strings.Contains(body, `"arguments":"{\"query\":\"test\"}"`) {
		t.Errorf("MCP arguments were not included in arguments.done event:\n%s", body)
	}
	if !strings.Contains(body, `"output_index":0`) {
		t.Errorf("output_index zero must be serialized, not omitted:\n%s", body)
	}

	lastSequence := 0
	for _, block := range strings.Split(body, "\n\n") {
		dataLine := ""
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "data: ") {
				dataLine = strings.TrimPrefix(line, "data: ")
				break
			}
		}
		if dataLine == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
			t.Fatalf("invalid SSE event JSON: %v", err)
		}
		sequence, ok := event["sequence_number"].(float64)
		if !ok || int(sequence) != lastSequence+1 {
			t.Fatalf("SSE sequence numbers must be contiguous: previous=%d current=%v", lastSequence, event["sequence_number"])
		}
		lastSequence = int(sequence)
	}
}

func TestWriteResponsesMCPStreamEmitsFailedCallEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	completed := officialtypes.NewResponseAPIFull("Partial answer", "gpt-6-luna", 0, 0, 0, 0, 0, "")
	completed.Output = []officialtypes.ResponseOutput{
		{
			ID:        "mcp_failed",
			Type:      "mcp_call",
			Status:    "failed",
			Name:      "search",
			Arguments: `{}`,
			Output:    `{"error":"not approved"}`,
		},
	}

	writeResponsesMCPStream(ctx, completed)
	body := recorder.Body.String()
	if !strings.Contains(body, "event: response.mcp_call.failed\n") {
		t.Fatalf("failed MCP calls must emit response.mcp_call.failed:\n%s", body)
	}
	if strings.Contains(body, "event: response.mcp_call.completed\n") {
		t.Fatalf("failed MCP calls must not emit completed event:\n%s", body)
	}
}
