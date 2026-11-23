package initialize

import (
	"strings"

	officialtypes "aurora/typings/official"
)

// prepareMCPMessages puts all system-level instructions in the same user
// message as the research task. Duck.ai has no system role, and mapping a
// system message to a separate user turn weakens the instruction boundary.
func prepareMCPMessages(source []officialtypes.ApiMessage, protocol string) []officialtypes.ApiMessage {
	messages := make([]officialtypes.ApiMessage, 0, len(source)+1)
	var systemInstructions []string

	for _, message := range source {
		if strings.EqualFold(message.Role, "system") {
			if text := contentText(message.Content); strings.TrimSpace(text) != "" {
				systemInstructions = append(systemInstructions, strings.TrimSpace(text))
			}
			continue
		}
		messages = append(messages, message)
	}

	instructions := strings.Join(systemInstructions, "\n\n")
	if strings.TrimSpace(protocol) != "" {
		if instructions != "" {
			instructions += "\n\n"
		}
		instructions += strings.TrimSpace(protocol)
	}
	if instructions == "" {
		return messages
	}

	// Attach the protocol to the latest user message so it is adjacent to the
	// request. Preserve multimodal content by appending a text content part.
	for i := len(messages) - 1; i >= 0; i-- {
		if !strings.EqualFold(messages[i].Role, "user") {
			continue
		}
		switch value := messages[i].Content.(type) {
		case string:
			if strings.TrimSpace(value) == "" {
				messages[i].Content = instructions
			} else {
				messages[i].Content = value + "\n\n" + instructions
			}
			return messages
		case []interface{}:
			parts := append([]interface{}(nil), value...)
			parts = append(parts, map[string]interface{}{"type": "text", "text": instructions})
			messages[i].Content = parts
			return messages
		}
	}

	return append(messages, officialtypes.ApiMessage{Role: "user", Content: instructions})
}

func contentText(content interface{}) string {
	switch value := content.(type) {
	case string:
		return value
	case []interface{}:
		var parts []string
		for _, item := range value {
			part, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}
