package mcpbridge

import "testing"

func TestParseDirectiveToolCall(t *testing.T) {
	got, err := ParseDirective(`<mcp_call>{"server_label":"aljam3","tool_name":"search","arguments":{"q":"الأمر الوارد بعد الحظر"}}</mcp_call>`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "call" || got.ServerLabel != "aljam3" || got.ToolName != "search" {
		t.Fatalf("unexpected directive: %+v", got)
	}
	if got.Arguments["q"] != "الأمر الوارد بعد الحظر" {
		t.Fatalf("arguments were not decoded: %#v", got.Arguments)
	}
}

func TestParseDirectiveFinal(t *testing.T) {
	got, err := ParseDirective("<mcp_final>تمت مراجعة المصادر.</mcp_final>")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "final" || got.Text != "تمت مراجعة المصادر." {
		t.Fatalf("unexpected directive: %+v", got)
	}
}

func TestParseDirectivePlainTextIsFinal(t *testing.T) {
	got, err := ParseDirective("إجابة مباشرة من النموذج")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "final" || got.Text != "إجابة مباشرة من النموذج" {
		t.Fatalf("unexpected directive: %+v", got)
	}
}

func TestParseDirectiveRejectsUnknownCallFields(t *testing.T) {
	_, err := ParseDirective(`<mcp_call>{"server_label":"aljam3","tool_name":"search","arguments":{},"url":"https://evil.example"}</mcp_call>`)
	if err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestParseDirectiveRejectsTrailingJSON(t *testing.T) {
	_, err := ParseDirective(`<mcp_call>{"server_label":"research","tool_name":"search","arguments":{}} {"another":"object"}</mcp_call>`)
	if err == nil {
		t.Fatal("expected trailing JSON data to be rejected")
	}
}


func TestParseDirectiveCallAllowsSurroundingModelChatter(t *testing.T) {
	got, err := ParseDirective("I will search the source first.\n```xml\n<mcp_call>{\"server_label\":\"aljam3\",\"tool_name\":\"search\",\"arguments\":{\"q\":\"test\"}}</mcp_call>\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "call" || got.ServerLabel != "aljam3" || got.ToolName != "search" {
		t.Fatalf("unexpected directive: %+v", got)
	}
}

func TestParseDirectiveRejectsMultipleControlBlocks(t *testing.T) {
	_, err := ParseDirective(
		`<mcp_call>{"server_label":"aljam3","tool_name":"search","arguments":{}}</mcp_call>
<mcp_call>{"server_label":"shamela","tool_name":"shamela_find","arguments":{}}</mcp_call>`,
	)
	if err == nil {
		t.Fatal("expected duplicate call blocks to be rejected")
	}
}

func TestParseDirectiveRejectsUnclosedControlBlock(t *testing.T) {
	_, err := ParseDirective(`<mcp_call>{"server_label":"aljam3","tool_name":"search","arguments":{}}`)
	if err == nil {
		t.Fatal("expected unclosed call block to be rejected")
	}
}
