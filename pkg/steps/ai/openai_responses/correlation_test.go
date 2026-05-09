package openai_responses

import (
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func TestResponsesProviderCallCorrelationUsesExplicitProviderCallID(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	reqBody := responsesRequest{Model: "gpt-test"}

	for _, tt := range []struct {
		name              string
		providerCallIndex int
		wantProviderID    string
	}{
		{name: "first provider call", providerCallIndex: 0, wantProviderID: "openai_responses:inference-1:provider-call:0"},
		{name: "third provider call", providerCallIndex: 2, wantProviderID: "openai_responses:inference-1:provider-call:2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			corr := newResponsesProviderCallCorrelation(metadata, reqBody, tt.providerCallIndex)
			if corr.ProviderCallID != tt.wantProviderID {
				t.Fatalf("ProviderCallID = %q, want %q", corr.ProviderCallID, tt.wantProviderID)
			}
			if corr.SessionID != metadata.SessionID || corr.RunID != metadata.InferenceID || corr.TurnID != metadata.TurnID {
				t.Fatalf("scope identity mismatch: %+v", corr)
			}
		})
	}
}

func TestResponsesSegmentCorrelationKeepsExplicitSegmentIdentity(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := newResponsesProviderCallCorrelation(metadata, responsesRequest{Model: "gpt-test"}, 2)
	state := newResponsesStreamState(responsesRequest{Model: "gpt-test"}, providerCorr, nil)
	state.currentResponseID = "resp-1"
	outputIndex := 0

	for _, tt := range []struct {
		name        string
		segmentType string
	}{
		{name: "text", segmentType: events.SegmentTypeText},
		{name: "reasoning", segmentType: events.SegmentTypeReasoning},
		{name: "tool", segmentType: events.SegmentTypeTool},
	} {
		t.Run(tt.name, func(t *testing.T) {
			corr := state.segmentCorrelation("item-1", &outputIndex, nil, tt.segmentType)
			if corr.RunID != providerCorr.RunID || corr.ProviderCallID != providerCorr.ProviderCallID {
				t.Fatalf("parent identity mismatch: %+v want %+v", corr, providerCorr)
			}
			if corr.SegmentID == "" || !strings.Contains(corr.SegmentID, "item:item-1") {
				t.Fatalf("segment identity missing provider item: %+v", corr)
			}
		})
	}
}

func TestResponsesMultipleReasoningSegmentsInSameProviderCallDoNotShareIdentity(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := newResponsesProviderCallCorrelation(metadata, responsesRequest{Model: "gpt-test"}, 2)
	state := newResponsesStreamState(responsesRequest{Model: "gpt-test"}, providerCorr, nil)
	state.currentResponseID = "resp-1"
	firstOutputIndex := 0
	secondOutputIndex := 2

	first := state.segmentCorrelation("rs_1", &firstOutputIndex, nil, events.SegmentTypeReasoning)
	second := state.segmentCorrelation("rs_2", &secondOutputIndex, nil, events.SegmentTypeReasoning)

	for name, corr := range map[string]events.Correlation{"first": first, "second": second} {
		if corr.RunID != providerCorr.RunID || corr.ProviderCallID != providerCorr.ProviderCallID {
			t.Fatalf("%s parent identity mismatch: %+v", name, corr)
		}
		if corr.SegmentID == "" {
			t.Fatalf("%s correlation missing SegmentID: %+v", name, corr)
		}
	}
	if first.SegmentID == second.SegmentID {
		t.Fatalf("SegmentID collision: %q", first.SegmentID)
	}
}
