package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	claudesettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type captureClaudeObserver struct {
	mu      sync.Mutex
	records []geppettoobs.Record
}

func (o *captureClaudeObserver) OnGeppettoRecord(_ context.Context, rec geppettoobs.Record) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records = append(o.records, rec)
}

func (o *captureClaudeObserver) snapshot() []geppettoobs.Record {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]geppettoobs.Record, len(o.records))
	copy(out, o.records)
	return out
}

type panicClaudeObserver struct{}

func (panicClaudeObserver) OnGeppettoRecord(context.Context, geppettoobs.Record) {
	panic("observer should not affect inference")
}

func newObservableClaudeSettings(t *testing.T, body string) *aisettings.InferenceSettings {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	httpClient := server.Client()
	httpClient.Transport = &claudeHeaderTransport{
		base:   httpClient.Transport,
		target: targetURL,
		host:   "api.anthropic.com",
		scheme: "https",
		header: "X-Test-Transport",
		value:  "claude-observability",
	}
	engine := "claude-test"
	apiType := ai_types.ApiTypeClaude
	return &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{HTTPClient: httpClient},
		Claude: &claudesettings.Settings{},
		API: &aisettings.APISettings{
			APIKeys:  map[string]string{"claude-api-key": "test"},
			BaseUrls: map[string]string{"claude-base-url": "https://api.anthropic.com"},
		},
		Chat: &aisettings.ChatSettings{
			Engine:  &engine,
			ApiType: &apiType,
			Stream:  true,
		},
	}
}

func claudeSSE(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func minimalClaudeSSE() string {
	return claudeSSE(
		"event: message_start",
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-test","stop_reason":"","stop_sequence":"","usage":{"input_tokens":1,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"pong"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":""},"usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	)
}

func toolUseClaudeSSE() string {
	return claudeSSE(
		"event: message_start",
		`data: {"type":"message_start","message":{"id":"msg_tool","type":"message","role":"assistant","content":[],"model":"claude-test","stop_reason":"","stop_sequence":"","usage":{"input_tokens":7,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I'll inspect first."}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool_1","name":"sql_doc","input":{}}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"topic\":\"inventory\"}"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":1}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":""},"usage":{"input_tokens":0,"output_tokens":13,"cache_creation_input_tokens":2,"cache_read_input_tokens":5}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	)
}

func TestClaudeRunInferenceToolUsePreservesMessageDeltaMetadata(t *testing.T) {
	eng, err := NewClaudeEngine(newObservableClaudeSettings(t, toolUseClaudeSSE()))
	if err != nil {
		t.Fatalf("NewClaudeEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_tool", Blocks: []turns.Block{turns.NewUserTextBlock("inspect inventory")}}
	out, err := eng.RunInference(context.Background(), turn)
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if out == nil {
		t.Fatalf("expected output turn")
	}

	stored, ok, err := turns.KeyTurnMetaInferenceResult.Get(out.Metadata)
	if err != nil {
		t.Fatalf("get inference_result: %v", err)
	}
	if !ok {
		t.Fatalf("expected inference_result metadata")
	}
	if stored.StopReason != "tool_use" {
		t.Fatalf("expected stop_reason=tool_use, got %+v", stored)
	}
	if stored.Usage == nil {
		t.Fatalf("expected usage metadata, got %+v", stored)
	}
	if stored.Usage.InputTokens != 7 || stored.Usage.OutputTokens != 13 || stored.Usage.CacheCreationInputTokens != 2 || stored.Usage.CacheReadInputTokens != 5 {
		t.Fatalf("unexpected usage metadata: %+v", stored.Usage)
	}
	if stored.DurationMs == nil {
		t.Fatalf("expected duration metadata, got %+v", stored)
	}
}

func TestClaudeObservabilityOffEmitsNoRecords(t *testing.T) {
	obs := &captureClaudeObserver{}
	eng, err := NewClaudeEngine(newObservableClaudeSettings(t, minimalClaudeSSE()), WithObserver(obs), WithObservabilityConfig(geppettoobs.DefaultConfig()))
	if err != nil {
		t.Fatalf("NewClaudeEngine: %v", err)
	}
	eng.observe(context.Background(), geppettoobs.Record{Stage: geppettoobs.StageProviderRoutedEvent})
	if got := len(obs.snapshot()); got != 0 {
		t.Fatalf("expected no records with trace off, got %d", got)
	}
}

func TestClaudeObservabilityCapturesPublishStartedAndProviderRecords(t *testing.T) {
	obs := &captureClaudeObserver{}
	eng, err := NewClaudeEngine(newObservableClaudeSettings(t, minimalClaudeSSE()), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewClaudeEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say pong")}}
	out, err := eng.RunInference(context.Background(), turn)
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if out == nil || len(out.Blocks) == 0 {
		t.Fatalf("expected response block")
	}

	records := obs.snapshot()
	providerRec := findClaudeRecord(records, geppettoobs.StageProviderRoutedEvent, "message_start")
	if providerRec == nil {
		t.Fatalf("missing message_start provider record in %#v", records)
	}
	if providerRec.Provider != "claude" || providerRec.Model != "claude-test" || providerRec.ResponseID != "msg_1" || providerRec.TurnID != "turn_1" {
		t.Fatalf("provider scalar fields not captured: %#v", providerRec)
	}
	var object map[string]any
	if err := json.Unmarshal(providerRec.ObjectJSON, &object); err != nil {
		t.Fatalf("provider ObjectJSON invalid: %s: %v", string(providerRec.ObjectJSON), err)
	}
	if object["type"] != "message_start" {
		t.Fatalf("provider ObjectJSON missing event type: %s", string(providerRec.ObjectJSON))
	}

	deltaRec := findClaudeRecord(records, geppettoobs.StageProviderRoutedEvent, "content_block_delta")
	if deltaRec == nil || deltaRec.OutputIndex == nil || *deltaRec.OutputIndex != 0 || deltaRec.DeltaLen != len("pong") {
		t.Fatalf("delta provider record missing index or length: %#v", deltaRec)
	}
	started := findClaudeRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeProviderCallFinished))
	if started == nil {
		t.Fatalf("missing provider-call-finished publish-started record in %#v", records)
	}
	if started.ProviderCallID == "" {
		t.Fatalf("provider-call-finished record missing provider call id: %#v", started)
	}
	if len(started.EventJSON) != 0 || len(started.MetadataJSON) != 0 {
		t.Fatalf("publish-started should be compact: %#v", started)
	}
	if done := findClaudeRecord(records, geppettoobs.StageGeppettoPublishDone, ""); done != nil {
		t.Fatalf("did not expect publish-done record: %#v", done)
	}
}

func TestClaudeObservabilityEventsLevelSkipsProviderRecords(t *testing.T) {
	obs := &captureClaudeObserver{}
	eng, err := NewClaudeEngine(newObservableClaudeSettings(t, minimalClaudeSSE()), WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceEvents}))
	if err != nil {
		t.Fatalf("NewClaudeEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say pong")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	records := obs.snapshot()
	if rec := findClaudeRecord(records, geppettoobs.StageProviderRoutedEvent, "message_start"); rec != nil {
		t.Fatalf("did not expect provider record at events trace level: %#v", rec)
	}
	if rec := findClaudeRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeProviderCallFinished)); rec == nil {
		t.Fatalf("expected provider-call-finished publish-started record in %#v", records)
	}
}

func TestClaudeObservabilityObserverPanicDoesNotFailInference(t *testing.T) {
	eng, err := NewClaudeEngine(newObservableClaudeSettings(t, minimalClaudeSSE()), WithObserver(panicClaudeObserver{}), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewClaudeEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("say pong")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference should ignore observer panic: %v", err)
	}
}

func findClaudeRecord(records []geppettoobs.Record, stage geppettoobs.Stage, eventType string) *geppettoobs.Record {
	for i := range records {
		rec := &records[i]
		if rec.Stage != stage {
			continue
		}
		if eventType != "" && rec.EventType != eventType {
			continue
		}
		return rec
	}
	return nil
}
