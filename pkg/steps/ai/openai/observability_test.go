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

func TestOpenAIObservabilityCapturesPublishAndProviderRecords(t *testing.T) {
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

	partialDone := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishDone, string(events.EventTypePartialCompletion), "")
	if partialDone == nil {
		t.Fatalf("missing partial publish-done record in %#v", records)
	}
	if len(partialDone.EventJSON) == 0 || len(partialDone.MetadataJSON) == 0 {
		t.Fatalf("publish-done should carry event and metadata JSON: %#v", partialDone)
	}
	if partialDone.Provider != "openai" || partialDone.Model != "gpt-test" || partialDone.TurnID != "turn_1" {
		t.Fatalf("publish record did not capture expected scalar fields: %#v", partialDone)
	}

	startStarted := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeStart), "")
	if startStarted == nil {
		t.Fatalf("missing start publish-started record in %#v", records)
	}
	if len(startStarted.EventJSON) != 0 || len(startStarted.MetadataJSON) != 0 {
		t.Fatalf("publish-started should not carry full JSON payloads: %#v", startStarted)
	}
}

func TestOpenAIObservabilityEventsLevelSkipsProviderRecords(t *testing.T) {
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
	if rec := findGeppettoRecord(records, geppettoobs.StageGeppettoPublishDone, string(events.EventTypeFinal), ""); rec == nil {
		t.Fatalf("expected final publish record at events trace level in %#v", records)
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
