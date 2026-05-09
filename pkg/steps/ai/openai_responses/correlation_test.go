package openai_responses

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
)

func TestResponsesProviderCallCorrelationUsesProviderCallIndex(t *testing.T) {
	metadata := events.EventMetadata{InferenceID: "inference-1", TurnID: "turn-1"}
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
			if corr.ProviderCallIndex != int32(tt.providerCallIndex) {
				t.Fatalf("ProviderCallIndex = %d, want %d", corr.ProviderCallIndex, tt.providerCallIndex)
			}
			if corr.ProviderCallID != tt.wantProviderID {
				t.Fatalf("ProviderCallID = %q, want %q", corr.ProviderCallID, tt.wantProviderID)
			}
			if corr.CorrelationKey != tt.wantProviderID {
				t.Fatalf("CorrelationKey = %q, want %q", corr.CorrelationKey, tt.wantProviderID)
			}
			if corr.Model != reqBody.Model {
				t.Fatalf("Model = %q, want %q", corr.Model, reqBody.Model)
			}
			if corr.TurnID != metadata.TurnID {
				t.Fatalf("TurnID = %q, want %q", corr.TurnID, metadata.TurnID)
			}
		})
	}
}

func TestResponsesSegmentCorrelationKeepsProviderCallIndexAndTypedSegmentIdentity(t *testing.T) {
	metadata := events.EventMetadata{InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := newResponsesProviderCallCorrelation(metadata, responsesRequest{Model: "gpt-test"}, 2)
	state := newResponsesStreamState(responsesRequest{Model: "gpt-test"}, providerCorr, nil)
	state.currentResponseID = "resp-1"
	outputIndex := 0

	for _, tt := range []struct {
		name        string
		segmentType string
		streamKind  string
	}{
		{name: "text", segmentType: events.SegmentTypeText, streamKind: events.StreamKindContent},
		{name: "reasoning", segmentType: events.SegmentTypeReasoning, streamKind: events.StreamKindReasoning},
		{name: "tool", segmentType: events.SegmentTypeTool, streamKind: events.StreamKindToolCall},
	} {
		t.Run(tt.name, func(t *testing.T) {
			corr := state.segmentCorrelation("item-1", &outputIndex, nil, tt.segmentType)
			if corr.ProviderCallIndex != 2 {
				t.Fatalf("ProviderCallIndex = %d, want 2", corr.ProviderCallIndex)
			}
			if corr.ProviderCallID != providerCorr.ProviderCallID {
				t.Fatalf("ProviderCallID = %q, want %q", corr.ProviderCallID, providerCorr.ProviderCallID)
			}
			if corr.ParentCorrelationKey != providerCorr.CorrelationKey {
				t.Fatalf("ParentCorrelationKey = %q, want %q", corr.ParentCorrelationKey, providerCorr.CorrelationKey)
			}
			if corr.SegmentIndex != 0 {
				t.Fatalf("SegmentIndex = %d, want 0; Responses uses provider item/output identity because one provider call may contain multiple same-type segments", corr.SegmentIndex)
			}
			if corr.SegmentType != tt.segmentType {
				t.Fatalf("SegmentType = %q, want %q", corr.SegmentType, tt.segmentType)
			}
			if corr.StreamKind != tt.streamKind {
				t.Fatalf("StreamKind = %q, want %q", corr.StreamKind, tt.streamKind)
			}
			if corr.SegmentID == "" || corr.CorrelationKey == "" {
				t.Fatalf("segment identity missing: %+v", corr)
			}
		})
	}
}

func TestResponsesMultipleReasoningSegmentsInSameProviderCallDoNotShareIdentity(t *testing.T) {
	metadata := events.EventMetadata{InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := newResponsesProviderCallCorrelation(metadata, responsesRequest{Model: "gpt-test"}, 2)
	state := newResponsesStreamState(responsesRequest{Model: "gpt-test"}, providerCorr, nil)
	state.currentResponseID = "resp-1"
	firstOutputIndex := 0
	secondOutputIndex := 2

	first := state.segmentCorrelation("rs_1", &firstOutputIndex, nil, events.SegmentTypeReasoning)
	second := state.segmentCorrelation("rs_2", &secondOutputIndex, nil, events.SegmentTypeReasoning)

	for name, corr := range map[string]events.Correlation{"first": first, "second": second} {
		if corr.ProviderCallIndex != 2 {
			t.Fatalf("%s ProviderCallIndex = %d, want 2", name, corr.ProviderCallIndex)
		}
		if corr.SegmentIndex != 0 {
			t.Fatalf("%s SegmentIndex = %d, want 0; distinct Responses items should route by SegmentID/CorrelationKey", name, corr.SegmentIndex)
		}
		if corr.SegmentType != events.SegmentTypeReasoning || corr.StreamKind != events.StreamKindReasoning {
			t.Fatalf("%s segment type/kind = %q/%q, want reasoning/reasoning", name, corr.SegmentType, corr.StreamKind)
		}
		if corr.SegmentID == "" || corr.CorrelationKey == "" {
			t.Fatalf("%s correlation missing identity: %+v", name, corr)
		}
	}
	if first.SegmentID == second.SegmentID {
		t.Fatalf("SegmentID collision: %q", first.SegmentID)
	}
	if first.CorrelationKey == second.CorrelationKey {
		t.Fatalf("CorrelationKey collision: %q", first.CorrelationKey)
	}
}
