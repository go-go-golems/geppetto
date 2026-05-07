package openai_responses

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

func TestResponsesObservabilityCapturesObjectEventAndMetadataJSON(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body := strings.Join([]string{
			"event: response.output_item.added",
			`data: {"response_id":"resp_1","output_index":0,"item":{"type":"reasoning","id":"rs_1","status":"in_progress"}}`,
			"",
			"event: response.reasoning_summary_part.added",
			`data: {"response_id":"resp_1","item_id":"rs_1","output_index":0,"summary_index":0}`,
			"",
			"event: response.reasoning_summary_text.delta",
			`data: {"response_id":"resp_1","item_id":"rs_1","output_index":0,"summary_index":0,"delta":"thinking"}`,
			"",
			"event: response.reasoning_summary_part.done",
			`data: {"response_id":"resp_1","item_id":"rs_1","output_index":0,"summary_index":0}`,
			"",
			"event: response.output_item.done",
			`data: {"response_id":"resp_1","output_index":0,"item":{"type":"reasoning","id":"rs_1","status":"completed","summary":[{"text":"thinking"}]}}`,
			"",
			"event: response.completed",
			`data: {"response":{"id":"resp_1","usage":{"input_tokens":1,"output_tokens":1}}}`,
			"",
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
	defer func() { http.DefaultClient = origClient }()

	obs := &captureGeppettoObserver{}
	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{Engine: ptr("gpt-5-mini"), Stream: true},
	}, WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceProvider}))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{ID: "turn_1", Blocks: []turns.Block{turns.NewUserTextBlock("think")}}
	if _, err := eng.RunInference(ctx, turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	providerRec := findRecord(records, geppettoobs.StageProviderRoutedEvent, "response.reasoning_summary_text.delta", "")
	if providerRec == nil {
		t.Fatalf("missing provider routed summary delta record in %#v", records)
	}
	if providerRec.ItemID != "rs_1" || providerRec.ResponseID != "resp_1" || providerRec.SummaryIndex == nil || *providerRec.SummaryIndex != 0 {
		t.Fatalf("provider IDs not captured: %#v", providerRec)
	}
	var providerObject map[string]any
	if err := json.Unmarshal(providerRec.ObjectJSON, &providerObject); err != nil {
		t.Fatalf("object_json is invalid: %s: %v", string(providerRec.ObjectJSON), err)
	}
	if providerObject["item_id"] != "rs_1" || providerObject["delta"] != "thinking" {
		t.Fatalf("object_json missing decoded provider fields: %s", string(providerRec.ObjectJSON))
	}

	publishStartedRec := findRecord(records, geppettoobs.StageGeppettoPublishStarted, string(events.EventTypeInfo), "reasoning-summary")
	if publishStartedRec == nil {
		t.Fatalf("missing publish started reasoning-summary record in %#v", records)
	}
	if len(publishStartedRec.EventJSON) != 0 || len(publishStartedRec.MetadataJSON) != 0 {
		t.Fatalf("publish started should not carry full payload JSON: %#v", publishStartedRec)
	}

	if publishDoneRec := findRecord(records, geppettoobs.StageGeppettoPublishDone, string(events.EventTypeInfo), "reasoning-summary"); publishDoneRec != nil {
		t.Fatalf("did not expect publish done reasoning-summary record: %#v", publishDoneRec)
	}

	var finalSummary *events.EventInfo
	for _, ev := range sink.snapshot() {
		if info, ok := ev.(*events.EventInfo); ok && info.Message == "reasoning-summary" {
			finalSummary = info
		}
	}
	if finalSummary == nil {
		t.Fatalf("missing final reasoning-summary event")
	}
	if finalSummary.Data["item_id"] != "rs_1" || finalSummary.Data["response_id"] != "resp_1" {
		t.Fatalf("final reasoning-summary data missing provider IDs: %#v", finalSummary.Data)
	}
}

func TestResponsesObservabilityOffEmitsNoRecords(t *testing.T) {
	obs := &captureGeppettoObserver{}
	eng, err := NewEngine(&settings.InferenceSettings{}, WithObserver(obs), WithObservabilityConfig(geppettoobs.DefaultConfig()))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	eng.observe(context.Background(), geppettoobs.Record{Stage: geppettoobs.StageProviderRoutedEvent})
	if got := len(obs.snapshot()); got != 0 {
		t.Fatalf("expected no records with trace off, got %d", got)
	}
}

func TestIntFromAnyStringParsingRequiresExactInteger(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
		ok   bool
	}{
		{name: "plain", in: "1", want: 1, ok: true},
		{name: "trimmed", in: " 1 ", want: 1, ok: true},
		{name: "trailing junk", in: "1x", ok: false},
		{name: "leading junk", in: "x1", ok: false},
		{name: "empty", in: "", ok: false},
		{name: "spaces", in: "   ", ok: false},
		{name: "uint64 overflow", in: uint64(^uint(0)>>1) + 1, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := intFromAny(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok mismatch for %#v: got %v want %v", tt.in, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("value mismatch for %#v: got %d want %d", tt.in, got, tt.want)
			}
		})
	}
}

func findRecord(records []geppettoobs.Record, stage geppettoobs.Stage, eventType string, infoMessage string) *geppettoobs.Record {
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
