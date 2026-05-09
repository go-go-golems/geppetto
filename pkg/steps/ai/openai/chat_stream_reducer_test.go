package openai

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
)

func TestReduceOpenAIChatStream(t *testing.T) {
	tests := []struct {
		name   string
		inputs []openAIChatStreamInput

		wantEventTypes      []events.EventType
		wantText            string
		wantReasoning       string
		wantTextClosed      bool
		wantReasoningClosed bool
		wantToolCount       int
		wantFinalClass      string
		wantFinalReason     string
	}{
		{
			name: "text delta then eof closes text",
			inputs: []openAIChatStreamInput{
				chunkInput(textDelta("hello")),
				terminalInput(openAIChatTerminalEOF, nil),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeTextSegmentStarted,
				events.EventTypeTextDelta,
				events.EventTypeTextSegmentFinished,
				events.EventTypeProviderCallFinished,
			},
			wantText:       "hello",
			wantTextClosed: true,
			wantFinalClass: "completed",
		},
		{
			name: "eof with no content does not manufacture segment",
			inputs: []openAIChatStreamInput{
				terminalInput(openAIChatTerminalEOF, nil),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeProviderCallFinished,
			},
			wantFinalClass: "completed",
		},
		{
			name: "cancel closes active text but does not request tools",
			inputs: []openAIChatStreamInput{
				chunkInput(textDelta("partial")),
				terminalInput(openAIChatTerminalCancelled, context.Canceled),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeTextSegmentStarted,
				events.EventTypeTextDelta,
				events.EventTypeTextSegmentFinished,
				events.EventTypeInterrupt,
				events.EventTypeProviderCallFinished,
			},
			wantText:        "partial",
			wantTextClosed:  true,
			wantFinalClass:  "cancelled",
			wantFinalReason: "cancelled",
		},
		{
			name: "error closes reasoning and emits error",
			inputs: []openAIChatStreamInput{
				chunkInput(reasoningDelta("thinking")),
				terminalInput(openAIChatTerminalError, errors.New("boom")),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeReasoningSegmentStarted,
				events.EventTypeReasoningDelta,
				events.EventTypeReasoningSegmentFinished,
				events.EventTypeError,
				events.EventTypeProviderCallFinished,
			},
			wantReasoning:       "thinking",
			wantReasoningClosed: true,
			wantFinalClass:      "failed",
			wantFinalReason:     "error",
		},
		{
			name: "tool args accumulate and eof requests tool",
			inputs: []openAIChatStreamInput{
				chunkInput(toolDelta("call_1", 0, "search", `{"q"`)),
				chunkInput(toolDelta("call_1", 0, "", `:"x"}`)),
				terminalInput(openAIChatTerminalEOF, nil),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeToolCallStarted,
				events.EventTypeToolCallArgumentsDelta,
				events.EventTypeToolCallArgumentsDelta,
				events.EventTypeToolCallRequested,
				events.EventTypeProviderCallFinished,
			},
			wantToolCount:  1,
			wantFinalClass: "tool_calls_pending",
		},
		{
			name: "cancel after partial tool does not request tool",
			inputs: []openAIChatStreamInput{
				chunkInput(toolDelta("call_1", 0, "search", `{"q"`)),
				terminalInput(openAIChatTerminalCancelled, context.Canceled),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeToolCallStarted,
				events.EventTypeToolCallArgumentsDelta,
				events.EventTypeInterrupt,
				events.EventTypeProviderCallFinished,
			},
			wantToolCount:   1,
			wantFinalClass:  "cancelled",
			wantFinalReason: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestOpenAIChatStreamState()
			var got []events.Event

			for _, input := range tt.inputs {
				var effects []openAIChatStreamEffect
				state, effects = reduceOpenAIChatStream(state, input)
				got = append(got, eventsFromOpenAIChatEffects(effects)...)
			}

			assertOpenAIChatEventTypes(t, got, tt.wantEventTypes)
			assertOpenAIChatState(t, state, tt.wantText, tt.wantReasoning, tt.wantTextClosed, tt.wantReasoningClosed, tt.wantToolCount)
			assertProviderCallFinished(t, got, tt.wantFinalClass, tt.wantFinalReason)
		})
	}
}

func TestReduceOpenAIChatStreamReviewDerivedScenarios(t *testing.T) {
	tests := []struct {
		name           string
		inputs         []openAIChatStreamInput
		wantEventTypes []events.EventType
		check          func(t *testing.T, state openAIChatStreamState, got []events.Event)
	}{
		{
			name: "metadata-only final chunk does not create text segment",
			inputs: []openAIChatStreamInput{
				chunkInput(metadataOnlyChunk("stop", &chatStreamUsage{promptTokens: 3, completionTokens: 5})),
				terminalInput(openAIChatTerminalEOF, nil),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeProviderCallMetadataUpdated,
				events.EventTypeProviderCallFinished,
			},
			check: func(t *testing.T, state openAIChatStreamState, got []events.Event) {
				t.Helper()
				if state.TextSegmentStarted {
					t.Fatalf("metadata-only chunk started text segment")
				}
				finished := lastOpenAIChatProviderFinished(t, got)
				if finished.StopReason != "stop" {
					t.Fatalf("stop reason = %q, want stop", finished.StopReason)
				}
				if finished.Usage == nil || finished.Usage.InputTokens != 3 || finished.Usage.OutputTokens != 5 {
					t.Fatalf("usage = %#v, want input=3 output=5", finished.Usage)
				}
			},
		},
		{
			name: "sparse tool argument delta preserves tool name for request",
			inputs: []openAIChatStreamInput{
				chunkInput(toolDelta("call_1", 0, "search", `{"q"`)),
				chunkInput(toolDelta("call_1", 0, "", `:"x"}`)),
				terminalInput(openAIChatTerminalEOF, nil),
			},
			wantEventTypes: []events.EventType{
				events.EventTypeToolCallStarted,
				events.EventTypeToolCallArgumentsDelta,
				events.EventTypeToolCallArgumentsDelta,
				events.EventTypeToolCallRequested,
				events.EventTypeProviderCallFinished,
			},
			check: func(t *testing.T, _ openAIChatStreamState, got []events.Event) {
				t.Helper()
				requested := firstOpenAIChatToolRequested(t, got)
				if requested.ToolName != "search" {
					t.Fatalf("tool name = %q, want search", requested.ToolName)
				}
				if requested.Input != `{"q":"x"}` {
					t.Fatalf("tool input = %q, want accumulated JSON", requested.Input)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestOpenAIChatStreamState()
			var got []events.Event
			for _, input := range tt.inputs {
				var effects []openAIChatStreamEffect
				state, effects = reduceOpenAIChatStream(state, input)
				got = append(got, eventsFromOpenAIChatEffects(effects)...)
			}
			assertOpenAIChatEventTypes(t, got, tt.wantEventTypes)
			assertOpenAIChatCanonicalEventsValidate(t, got)
			if tt.check != nil {
				tt.check(t, state, got)
			}
		})
	}
}

func TestOpenAIChatCorrelationUsesProviderCallIndexForTextAndReasoningSegments(t *testing.T) {
	state := newTestOpenAIChatStreamState()
	state.ProviderCallCorr = events.BuildProviderCallCorrelation("openai", "inference-1", "", 2, "")
	state.CurrentResponseID = "chatcmpl-test"

	for _, tt := range []struct {
		name       string
		streamKind string
		wantType   string
	}{
		{name: "text", streamKind: events.StreamKindContent, wantType: events.SegmentTypeText},
		{name: "reasoning", streamKind: events.StreamKindReasoning, wantType: events.SegmentTypeReasoning},
	} {
		t.Run(tt.name, func(t *testing.T) {
			corr := state.chatCorrelation(ptrInt(0), tt.streamKind, "", nil)
			if corr.SegmentIndex != 3 {
				t.Fatalf("SegmentIndex = %d, want providerCallIndex+1 = 3", corr.SegmentIndex)
			}
			if corr.SegmentType != tt.wantType {
				t.Fatalf("SegmentType = %q, want %q", corr.SegmentType, tt.wantType)
			}
			if corr.SegmentID == "" || corr.CorrelationKey == "" {
				t.Fatalf("segment identity missing: %+v", corr)
			}
		})
	}
}

func TestReduceOpenAIChatStreamToolArgumentsAccumulate(t *testing.T) {
	state := newTestOpenAIChatStreamState()
	var got []events.Event

	for _, input := range []openAIChatStreamInput{
		chunkInput(toolDelta("call_1", 0, "search", `{"q"`)),
		chunkInput(toolDelta("call_1", 0, "", `:"x"}`)),
	} {
		var effects []openAIChatStreamEffect
		state, effects = reduceOpenAIChatStream(state, input)
		got = append(got, eventsFromOpenAIChatEffects(effects)...)
	}

	var deltas []*events.EventToolCallArgumentsDelta
	for _, ev := range got {
		if delta, ok := ev.(*events.EventToolCallArgumentsDelta); ok {
			deltas = append(deltas, delta)
		}
	}
	if len(deltas) != 2 {
		t.Fatalf("tool argument delta count = %d, want 2", len(deltas))
	}
	if deltas[0].Delta != `{"q"` || deltas[0].Arguments != `{"q"` || deltas[0].Sequence != 1 {
		t.Fatalf("first delta = (%q, %q, %d), want fragment and accumulated first chunk", deltas[0].Delta, deltas[0].Arguments, deltas[0].Sequence)
	}
	if deltas[1].Delta != `:"x"}` || deltas[1].Arguments != `{"q":"x"}` || deltas[1].Sequence != 2 {
		t.Fatalf("second delta = (%q, %q, %d), want accumulated arguments", deltas[1].Delta, deltas[1].Arguments, deltas[1].Sequence)
	}
}

func newTestOpenAIChatStreamState() openAIChatStreamState {
	metadata := events.EventMetadata{
		ID:          uuid.New(),
		SessionID:   "session-1",
		InferenceID: "inference-1",
		TurnID:      "turn-1",
	}
	providerCallCorr := events.BuildProviderCallCorrelation("openai", metadata.InferenceID, "", 0, "")
	return newOpenAIChatStreamState(metadata, "openai", "gpt-test", providerCallCorr)
}

func ptrInt(v int) *int { return &v }

func chunkInput(chunk chatStreamEvent) openAIChatStreamInput {
	return openAIChatStreamInput{Kind: openAIChatStreamInputChunk, Chunk: chunk}
}

func terminalInput(kind openAIChatTerminalKind, err error) openAIChatStreamInput {
	return openAIChatStreamInput{Kind: openAIChatStreamInputTerminal, Terminal: openAIChatTerminal{Kind: kind, Err: err}}
}

func textDelta(delta string) chatStreamEvent {
	choice := 0
	return chatStreamEvent{
		DeltaText:   delta,
		ChoiceIndex: &choice,
		RawPayload:  map[string]any{"id": "chatcmpl-test"},
	}
}

func reasoningDelta(delta string) chatStreamEvent {
	choice := 0
	return chatStreamEvent{
		DeltaReasoning: delta,
		ChoiceIndex:    &choice,
		RawPayload:     map[string]any{"id": "chatcmpl-test"},
	}
}

func metadataOnlyChunk(finishReason string, usage *chatStreamUsage) chatStreamEvent {
	choice := 0
	return chatStreamEvent{
		ChoiceIndex:  &choice,
		RawPayload:   map[string]any{"id": "chatcmpl-test"},
		FinishReason: &finishReason,
		Usage:        usage,
	}
}

func toolDelta(id string, index int, name string, arguments string) chatStreamEvent {
	choice := 0
	idx := index
	return chatStreamEvent{
		ChoiceIndex: &choice,
		RawPayload:  map[string]any{"id": "chatcmpl-test"},
		ToolCalls: []ChatToolCall{{
			Index: &idx,
			ID:    id,
			Type:  chatToolTypeFunction,
			Function: ChatFunctionCall{
				Name:      name,
				Arguments: arguments,
			},
		}},
	}
}

func eventsFromOpenAIChatEffects(effects []openAIChatStreamEffect) []events.Event {
	out := make([]events.Event, 0, len(effects))
	for _, effect := range effects {
		if effect.Event != nil {
			out = append(out, effect.Event)
		}
	}
	return out
}

func assertOpenAIChatEventTypes(t *testing.T, got []events.Event, want []events.EventType) {
	t.Helper()
	gotTypes := make([]events.EventType, len(got))
	for i, ev := range got {
		gotTypes[i] = ev.Type()
	}
	if !reflect.DeepEqual(gotTypes, want) {
		t.Fatalf("event types = %#v, want %#v", gotTypes, want)
	}
}

func assertOpenAIChatState(
	t *testing.T,
	state openAIChatStreamState,
	wantText string,
	wantReasoning string,
	wantTextClosed bool,
	wantReasoningClosed bool,
	wantToolCount int,
) {
	t.Helper()
	if state.Message != wantText {
		t.Fatalf("text = %q, want %q", state.Message, wantText)
	}
	if state.Reasoning != wantReasoning {
		t.Fatalf("reasoning = %q, want %q", state.Reasoning, wantReasoning)
	}
	if state.TextSegmentFinished != wantTextClosed {
		t.Fatalf("text closed = %v, want %v", state.TextSegmentFinished, wantTextClosed)
	}
	if state.ReasoningSegmentFinished != wantReasoningClosed {
		t.Fatalf("reasoning closed = %v, want %v", state.ReasoningSegmentFinished, wantReasoningClosed)
	}
	if got := len(state.mergedToolCalls()); got != wantToolCount {
		t.Fatalf("tool count = %d, want %d", got, wantToolCount)
	}
}

func assertOpenAIChatCanonicalEventsValidate(t *testing.T, got []events.Event) {
	t.Helper()
	for i, ev := range got {
		if err := events.ValidateCanonicalEvent(ev); err != nil {
			t.Fatalf("event[%d] %s failed canonical validation: %v", i, ev.Type(), err)
		}
	}
}

func lastOpenAIChatProviderFinished(t *testing.T, got []events.Event) *events.EventProviderCallFinished {
	t.Helper()
	for i := len(got) - 1; i >= 0; i-- {
		if finished, ok := got[i].(*events.EventProviderCallFinished); ok {
			return finished
		}
	}
	t.Fatalf("missing provider call finished event")
	return nil
}

func firstOpenAIChatToolRequested(t *testing.T, got []events.Event) *events.EventToolCallRequested {
	t.Helper()
	for _, ev := range got {
		if requested, ok := ev.(*events.EventToolCallRequested); ok {
			return requested
		}
	}
	t.Fatalf("missing tool call requested event")
	return nil
}

func assertProviderCallFinished(t *testing.T, got []events.Event, wantClass string, wantReason string) {
	t.Helper()
	if wantClass == "" && wantReason == "" {
		return
	}
	for i := len(got) - 1; i >= 0; i-- {
		finished, ok := got[i].(*events.EventProviderCallFinished)
		if !ok {
			continue
		}
		if finished.FinishClass != wantClass {
			t.Fatalf("finish class = %q, want %q", finished.FinishClass, wantClass)
		}
		if finished.StopReason != wantReason {
			t.Fatalf("finish reason = %q, want %q", finished.StopReason, wantReason)
		}
		return
	}
	t.Fatalf("missing provider call finished event")
}

func TestAppendOpenAIChatTurnBlocks(t *testing.T) {
	baseState := func() openAIChatStreamState {
		state := newTestOpenAIChatStreamState()
		state.Reasoning = "because"
		state.Message = "partial answer"
		idx := 0
		state.ToolCallMerger.AddToolCalls([]ChatToolCall{{
			Index: &idx,
			ID:    "call_1",
			Type:  chatToolTypeFunction,
			Function: ChatFunctionCall{
				Name:      "search",
				Arguments: `{"q":"x"}`,
			},
		}})
		return state
	}

	tests := []struct {
		name             string
		includeToolCalls bool
		wantKinds        []turns.BlockKind
		wantToolCount    int
	}{
		{
			name:             "cancel path keeps partial transcript but not tool request",
			includeToolCalls: false,
			wantKinds:        []turns.BlockKind{turns.BlockKindReasoning, turns.BlockKindLLMText},
			wantToolCount:    0,
		},
		{
			name:             "eof path appends transcript and tool request",
			includeToolCalls: true,
			wantKinds:        []turns.BlockKind{turns.BlockKindReasoning, turns.BlockKindLLMText, turns.BlockKindToolCall},
			wantToolCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &turns.Turn{ID: "turn-1"}
			gotToolCount := appendOpenAIChatTurnBlocks(turn, baseState(), tt.includeToolCalls)
			if gotToolCount != tt.wantToolCount {
				t.Fatalf("tool count = %d, want %d", gotToolCount, tt.wantToolCount)
			}
			if len(turn.Blocks) != len(tt.wantKinds) {
				t.Fatalf("block count = %d, want %d", len(turn.Blocks), len(tt.wantKinds))
			}
			for i, wantKind := range tt.wantKinds {
				if turn.Blocks[i].Kind != wantKind {
					t.Fatalf("block[%d].kind = %s, want %s", i, turn.Blocks[i].Kind, wantKind)
				}
			}
		})
	}
}

func TestOpenAIChatTerminalStopReason(t *testing.T) {
	tests := []struct {
		name       string
		terminal   openAIChatTerminal
		wantReason string
		wantSet    bool
	}{
		{name: "eof preserves provider stop reason", terminal: openAIChatTerminal{Kind: openAIChatTerminalEOF}},
		{name: "cancel records cancelled", terminal: openAIChatTerminal{Kind: openAIChatTerminalCancelled}, wantReason: "cancelled", wantSet: true},
		{name: "error records error", terminal: openAIChatTerminal{Kind: openAIChatTerminalError}, wantReason: "error", wantSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestOpenAIChatStreamState().withTerminalStopReason(tt.terminal)
			if !tt.wantSet {
				if state.StopReason != nil {
					t.Fatalf("stop reason = %q, want nil", *state.StopReason)
				}
				return
			}
			if state.StopReason == nil || *state.StopReason != tt.wantReason {
				t.Fatalf("stop reason = %v, want %q", state.StopReason, tt.wantReason)
			}
		})
	}
}
