package events

import "testing"

func TestBuildProviderCallCorrelation(t *testing.T) {
	corr := BuildProviderCallCorrelation("claude", "inference-1", "", 3, "msg_late")
	if corr.RunID != "inference-1" {
		t.Fatalf("run id mismatch: %+v", corr)
	}
	if corr.ProviderCallID != "claude:inference-1:provider-call:3" {
		t.Fatalf("provider call id mismatch: %+v", corr)
	}
}

func TestBuildSegmentCorrelation(t *testing.T) {
	parent := BuildProviderCallCorrelation("openai", "inference-1", "run-1", 1, "resp_1")
	segment := BuildSegmentCorrelation(parent, "item_1", 2, SegmentTypeText)
	if segment.RunID != parent.RunID || segment.ProviderCallID != parent.ProviderCallID {
		t.Fatalf("parent identity not preserved: segment=%+v parent=%+v", segment, parent)
	}
	if segment.SegmentID != "openai:inference-1:provider-call:1:item_1:2:text" {
		t.Fatalf("segment id mismatch: got %q", segment.SegmentID)
	}
}

func TestBuildChatCompletionsCorrelation(t *testing.T) {
	choice := 1
	toolIndex := 2

	tests := []struct {
		name string
		corr Correlation
		id   string
	}{
		{
			name: "content",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindContent, "", nil),
			id:   "openai-chat:chatcmpl_1:choice:1:content",
		},
		{
			name: "reasoning",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindReasoning, "", nil),
			id:   "openai-chat:chatcmpl_1:choice:1:reasoning",
		},
		{
			name: "tool id",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindToolCall, "call_1", &toolIndex),
			id:   "openai-chat:chatcmpl_1:choice:1:tool:call_1",
		},
		{
			name: "tool index fallback",
			corr: BuildChatCompletionsCorrelation("openai", "chatcmpl_1", &choice, StreamKindToolCall, "", &toolIndex),
			id:   "openai-chat:chatcmpl_1:choice:1:tool-index:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.corr.SegmentID != tt.id {
				t.Fatalf("segment id mismatch: got %q want %q", tt.corr.SegmentID, tt.id)
			}
		})
	}
}

func TestBuildResponsesCorrelation(t *testing.T) {
	output := 3
	summary := 4

	if got := BuildResponsesCorrelation("openai", "resp_1", "item_1", &output, &summary).SegmentID; got != "openai:resp_1:item:item_1" {
		t.Fatalf("item key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", &output, &summary).SegmentID; got != "openai:resp_1:output:3:summary:4" {
		t.Fatalf("summary key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", &output, nil).SegmentID; got != "openai:resp_1:output:3" {
		t.Fatalf("output key mismatch: %q", got)
	}
	if got := BuildResponsesCorrelation("openai", "resp_1", "", nil, nil).SegmentID; got != "openai:resp_1" {
		t.Fatalf("response key mismatch: %q", got)
	}
}

func TestBuildClaudeCorrelation(t *testing.T) {
	providerCall := BuildClaudeProviderCallCorrelation("claude", "msg_1", 7)
	if providerCall.ProviderCallID != "claude:msg_1" {
		t.Fatalf("provider call id mismatch: %+v", providerCall)
	}

	segment := BuildClaudeSegmentCorrelation("claude", providerCall.ProviderCallID, 2, SegmentTypeText)
	if segment.ProviderCallID != providerCall.ProviderCallID {
		t.Fatalf("provider call id not preserved: %+v", segment)
	}
	if segment.SegmentID != "claude:claude:msg_1:block:2:text" {
		t.Fatalf("segment id mismatch: %q", segment.SegmentID)
	}
}
