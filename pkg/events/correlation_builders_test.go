package events

import "testing"

func TestBuildProviderCallCorrelation(t *testing.T) {
	corr := BuildProviderCallCorrelation("claude", "inference-1", "", 3, "msg_late")
	if corr.ProviderCallID != "claude:inference-1:provider-call:3" {
		t.Fatalf("provider call id mismatch: %+v", corr)
	}
	if corr.ResponseID != "msg_late" {
		t.Fatalf("response id should remain provider-native metadata: %+v", corr)
	}
	if corr.CorrelationKey != corr.ProviderCallID {
		t.Fatalf("provider call key should be independent from response id: %+v", corr)
	}
}

func TestBuildSegmentCorrelation(t *testing.T) {
	parent := BuildProviderCallCorrelation("openai", "inference-1", "run-1", 1, "resp_1")
	segment := BuildSegmentCorrelation(parent, "item_1", 2, SegmentTypeText)
	if segment.SegmentID == "" || segment.CorrelationKey == "" {
		t.Fatalf("segment identity missing: %+v", segment)
	}
	if segment.ParentCorrelationKey != parent.CorrelationKey {
		t.Fatalf("parent key mismatch: got %q want %q", segment.ParentCorrelationKey, parent.CorrelationKey)
	}
	if segment.SegmentType != SegmentTypeText || segment.StreamKind != StreamKindContent {
		t.Fatalf("segment type/stream kind mismatch: %+v", segment)
	}
}

func TestBuildChatCompletionsCorrelation(t *testing.T) {
	choice := 1
	toolIndex := 2

	tests := []struct {
		name string
		corr Correlation
		key  string
	}{
		{
			name: "content",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindContent, "", nil),
			key:  "openai-chat:chatcmpl_1:choice:1:content",
		},
		{
			name: "reasoning",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindReasoning, "", nil),
			key:  "openai-chat:chatcmpl_1:choice:1:reasoning",
		},
		{
			name: "tool id",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindToolCall, "call_1", &toolIndex),
			key:  "openai-chat:chatcmpl_1:choice:1:tool:call_1",
		},
		{
			name: "tool index fallback",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindToolCall, "", &toolIndex),
			key:  "openai-chat:chatcmpl_1:choice:1:tool-index:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.corr.CorrelationKey != tt.key {
				t.Fatalf("key mismatch: got %q want %q", tt.corr.CorrelationKey, tt.key)
			}
			if tt.corr.ChoiceIndex == nil || *tt.corr.ChoiceIndex != 1 {
				t.Fatalf("choice index not preserved: %+v", tt.corr.ChoiceIndex)
			}
		})
	}
}

func TestBuildResponsesCorrelation(t *testing.T) {
	output := 3
	summary := 4

	if got := BuildResponsesCorrelation("openai", "resp_1", "item_1", &output, &summary).CorrelationKey; got != "openai:resp_1:item:item_1" {
		t.Fatalf("item key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", &output, &summary).CorrelationKey; got != "openai:resp_1:output:3:summary:4" {
		t.Fatalf("summary key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", &output, nil).CorrelationKey; got != "openai:resp_1:output:3" {
		t.Fatalf("output key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", nil, nil).CorrelationKey; got != "openai:resp_1" {
		t.Fatalf("response key mismatch: %q", got)
	}
}

func TestBuildClaudeCorrelation(t *testing.T) {
	providerCall := BuildClaudeProviderCallCorrelation("claude", "msg_1", 7)
	if providerCall.ProviderCallID != "msg_1" {
		t.Fatalf("provider call id mismatch: %+v", providerCall)
	}
	if providerCall.CorrelationKey != "claude:msg_1:provider-call" {
		t.Fatalf("provider key mismatch: %q", providerCall.CorrelationKey)
	}

	segment := BuildClaudeSegmentCorrelation("claude", providerCall.ProviderCallID, 2, SegmentTypeText)
	if segment.SegmentID != "msg_1:block:2:text" {
		t.Fatalf("segment id mismatch: %q", segment.SegmentID)
	}
	if segment.CorrelationKey != "claude:msg_1:block:2:text" {
		t.Fatalf("segment key mismatch: %q", segment.CorrelationKey)
	}
	if segment.ParentCorrelationKey != providerCall.CorrelationKey {
		t.Fatalf("parent key mismatch: got %q want %q", segment.ParentCorrelationKey, providerCall.CorrelationKey)
	}
}
