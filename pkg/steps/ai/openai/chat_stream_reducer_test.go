package openai

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
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
