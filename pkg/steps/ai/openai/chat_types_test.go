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

func TestChatCompletionMessageMarshalJSON_IncludesReasoningContentWhenSet(t *testing.T) {
	msg := ChatCompletionMessage{
		Role:             "assistant",
		ReasoningContent: "need a tool",
		ToolCalls: []ChatToolCall{{
			ID:   "call_1",
			Type: chatToolTypeFunction,
			Function: ChatFunctionCall{
				Name:      "lookup",
				Arguments: `{"q":"cats"}`,
			},
		}},
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if decoded["reasoning_content"] != "need a tool" {
		t.Fatalf("expected reasoning_content, got %#v", decoded["reasoning_content"])
	}
}

func TestChatCompletionRequestMarshalJSON_IncludesThinkingControlsWhenSet(t *testing.T) {
	req := ChatCompletionRequest{
		Model: "DeepSeek-V4-Pro",
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: "hello"},
		},
		N:               1,
		Stream:          true,
		ReasoningEffort: "max",
		Thinking:        &ChatThinkingControl{Type: "enabled"},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if decoded["reasoning_effort"] != "max" {
		t.Fatalf("expected reasoning_effort=max, got %#v", decoded["reasoning_effort"])
	}
	thinking, ok := decoded["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("expected thinking object, got %#v", decoded["thinking"])
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("expected thinking.type=enabled, got %#v", thinking["type"])
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
