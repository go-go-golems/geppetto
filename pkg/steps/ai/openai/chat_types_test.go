package openai

import (
	"encoding/json"
	"testing"
)

func TestChatCompletionMessageMarshalJSON_UsesMultiContentArrayWhenPresent(t *testing.T) {
	msg := ChatCompletionMessage{
		Role: "user",
		MultiContent: []ChatMessagePart{
			{Type: chatMessagePartTypeText, Text: "look"},
			{Type: chatMessagePartTypeImageURL, ImageURL: &ChatMessageImageURL{
				URL:    "data:image/png;base64,abc",
				Detail: chatImageURLDetailAuto,
			}},
		},
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	content, ok := decoded["content"].([]any)
	if !ok {
		t.Fatalf("expected content array, got %#v", decoded["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected two content parts, got %d", len(content))
	}
}

func TestChatCompletionRequestMarshalJSON_PreservesExplicitFalseParallelToolCalls(t *testing.T) {
	req := ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "hello"},
		},
		N:                 1,
		Stream:            true,
		ParallelToolCalls: boolRef(false),
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if got, ok := decoded["parallel_tool_calls"].(bool); !ok || got {
		t.Fatalf("expected explicit parallel_tool_calls=false, got %#v", decoded["parallel_tool_calls"])
	}
}
