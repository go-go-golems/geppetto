package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type captureGeppettoObserver struct {
	mu      sync.Mutex
	records []geppettoobs.Record
}

func (o *captureGeppettoObserver) OnGeppettoRecord(_ context.Context, rec geppettoobs.Record) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records = append(o.records, rec)
}

func (o *captureGeppettoObserver) snapshot() []geppettoobs.Record {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]geppettoobs.Record, len(o.records))
	copy(out, o.records)
	return out
}

type panicGeppettoObserver struct{}

func (panicGeppettoObserver) OnGeppettoRecord(context.Context, geppettoobs.Record) {
	panic("observer should not affect inference")
}

func newObservableOpenAITestSettings(body string) *settings.InferenceSettings {
	engine := "gpt-test"
	apiType := ai_types.ApiTypeOpenAI
	return &settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test-key"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Client: &settings.ClientSettings{
			HTTPClient: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"missing auth"}`)), Request: r}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    r,
				}, nil
			})},
		},
		OpenAI: &openaisettings.Settings{},
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
			Engine:  &engine,
			Stream:  true,
		},
	}
}

func chatCompletionSSE(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func TestOpenAIObservabilityOffEmitsNoRecords(t *testing.T) {
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(""), WithObserver(obs), WithObservabilityConfig(geppettoobs.DefaultConfig()))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	eng.observe(context.Background(), geppettoobs.Record{Stage: geppettoobs.StageProviderRoutedEvent})
	if got := len(obs.snapshot()); got != 0 {
		t.Fatalf("expected no records with trace off, got %d", got)
	}
}

func TestOpenAIObservabilityCapturesPublishStartedAndProviderRecords(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"hel"}}]}`,
		``,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"prompt_tokens_details":{"cached_tokens":1}}}`,
		``,
		`data: [DONE]`,
		``,
	)
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}

	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say hello")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	providerRec := findGeppettoRecord(records, geppettoobs.StageProviderRoutedEvent, "chat.completion.chunk", "")
	if providerRec == nil {
		t.Fatalf("missing provider routed record in %#v", records)
	}
	if providerRec.Provider != "openai" || providerRec.Model != "gpt-test" || providerRec.ResponseID != "chatcmpl-1" || providerRec.TurnID != "turn_1" {
		t.Fatalf("provider record did not capture expected scalar fields: %#v", providerRec)
	}
	var providerObject map[string]any
	if err := json.Unmarshal(providerRec.ObjectJSON, &providerObject); err != nil {
		t.Fatalf("provider ObjectJSON invalid: %s: %v", string(providerRec.ObjectJSON), err)
	}
	if providerObject["id"] != "chatcmpl-1" || providerObject["object"] != "chat.completion.chunk" {
		t.Fatalf("provider ObjectJSON missing raw fields: %s", string(providerRec.ObjectJSON))
	}

	partialStarted := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypePartialCompletion), "")
	if partialStarted == nil {
		t.Fatalf("missing partial publish-started record in %#v", records)
	}
	if len(partialStarted.EventJSON) != 0 || len(partialStarted.MetadataJSON) != 0 {
		t.Fatalf("publish-started should stay compact and omit full JSON payloads: %#v", partialStarted)
	}
	if partialStarted.Provider != "openai" || partialStarted.Model != "gpt-test" || partialStarted.TurnID != "turn_1" {
		t.Fatalf("publish-started record did not capture expected scalar fields: %#v", partialStarted)
	}
	if done := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishDone, "", ""); done != nil {
		t.Fatalf("did not expect publish-done records: %#v", done)
	}
}

func TestOpenAIObservabilityEventsLevelEmitsPublishStartedOnly(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"hi"},"finish_reason":"stop"}]}`,
		``,
		`data: [DONE]`,
		``,
	)
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceEvents}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say hi")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	if rec := findGeppettoRecord(records, geppettoobs.StageProviderRoutedEvent, "chat.completion.chunk", ""); rec != nil {
		t.Fatalf("did not expect provider record at events trace level: %#v", rec)
	}
	if rec := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeFinal), ""); rec == nil {
		t.Fatalf("expected final publish-started record at events trace level in %#v", records)
	}
	if rec := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishDone, "", ""); rec != nil {
		t.Fatalf("did not expect publish-done records: %#v", rec)
	}
}

func TestOpenAIObservabilityAddsChatCorrelationFields(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-corr","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"reasoning_content":"thinking"}}]}`,
		``,
		`data: {"id":"chatcmpl-corr","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"content":"answer"},"finish_reason":"stop"}]}`,
		``,
		`data: [DONE]`,
		``,
	)
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("think")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	reasoningRec := findGeppettoRecord(obs.snapshot(), geppettoobs.StageProviderRoutedEvent, "chat.completion.chunk", "")
	if reasoningRec == nil {
		t.Fatalf("missing provider record")
	}
	if reasoningRec.StreamKind != "reasoning" || reasoningRec.ChoiceIndex == nil || *reasoningRec.ChoiceIndex != 0 {
		t.Fatalf("unexpected reasoning correlation fields: %#v", reasoningRec)
	}
	if reasoningRec.CorrelationKey != "openai-chat:chatcmpl-corr:choice:0:reasoning" {
		t.Fatalf("unexpected reasoning correlation key: %q", reasoningRec.CorrelationKey)
	}
	partialThinking := findGeppettoRecord(obs.snapshot(), geppettoobs.StageGeppettoPublishStarted, string(events.EventTypePartialThinking), "")
	if partialThinking == nil {
		t.Fatalf("missing partial thinking publish record")
	}
	if partialThinking.CorrelationKey != reasoningRec.CorrelationKey || partialThinking.StreamKind != "reasoning" {
		t.Fatalf("thinking publish did not preserve correlation data: %#v", partialThinking)
	}
}

func TestOpenAIObservabilityAddsToolCorrelationFields(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-tool","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"q\""}}]}}]}`,
		``,
		`data: {"id":"chatcmpl-tool","object":"chat.completion.chunk","model":"gpt-test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"cats\"}"}}]},"finish_reason":"tool_calls"}]}`,
		``,
		`data: [DONE]`,
		``,
	)
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("use tool")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	var toolProvider *geppettoobs.Record
	for _, rec := range obs.snapshot() {
		if rec.Stage == geppettoobs.StageProviderRoutedEvent && rec.StreamKind == "tool_call" {
			toolProvider = &rec
			break
		}
	}
	if toolProvider == nil {
		t.Fatalf("missing tool provider record in %#v", obs.snapshot())
	}
	if toolProvider.ToolCallID != "call_1" || toolProvider.ToolCallIndex == nil || *toolProvider.ToolCallIndex != 0 {
		t.Fatalf("unexpected tool provider fields: %#v", toolProvider)
	}
	if toolProvider.CorrelationKey != "openai-chat:chatcmpl-tool:choice:0:tool:call_1" {
		t.Fatalf("unexpected tool correlation key: %q", toolProvider.CorrelationKey)
	}
	toolPublish := findGeppettoRecord(obs.snapshot(), geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeToolCall), "")
	if toolPublish == nil {
		t.Fatalf("missing tool publish record")
	}
	if toolPublish.CorrelationKey != toolProvider.CorrelationKey || toolPublish.ToolCallID != "call_1" {
		t.Fatalf("tool publish did not preserve correlation data: %#v", toolPublish)
	}
}

func TestOpenAIObservabilityCapturesReasoningNormalization(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"reasoning_content":"thinking"}}]}`,
		``,
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"answer"},"finish_reason":"stop"}]}`,
		``,
		`data: [DONE]`,
		``,
	)
	obs := &captureGeppettoObserver{}
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("think")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	rec := findGeppettoRecord(obs.snapshot(), geppettoobs.StageProviderNormalizeDelta, "chat.completion.chunk", "")
	if rec == nil {
		t.Fatalf("missing reasoning normalization record")
	}
	if rec.DeltaLen != len("thinking") || rec.NormalizedDeltaLen == 0 || rec.BufferLen == 0 {
		t.Fatalf("unexpected normalization lengths: %#v", rec)
	}
}

func TestOpenAIObservabilityObserverPanicDoesNotFailInference(t *testing.T) {
	body := chatCompletionSSE(
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"hi"},"finish_reason":"stop"}]}`,
		``,
		`data: [DONE]`,
		``,
	)
	eng, err := NewOpenAIEngine(newObservableOpenAITestSettings(body), WithObserver(panicGeppettoObserver{}), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewOpenAIEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say hi")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference should ignore observer panic: %v", err)
	}
}

func findGeppettoRecord(records []geppettoobs.Record, stage geppettoobs.Stage, eventType string, infoMessage string) *geppettoobs.Record {
	for i := range records {
		rec := &records[i]
		if rec.Stage != stage {
			continue
		}
		if eventType != "" && rec.EventType != eventType {
			continue
		}
		if infoMessage != "" && rec.InfoMessage != infoMessage {
			continue
		}
		return rec
	}
	return nil
}
