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
	}, WithObserver(obs), WithObservabilityConfig(geppettoobs.Config{
		Level:              geppettoobs.TraceProvider,
		MaxPayloadBytes:    4096,
		RedactProviderData: true,
	}))
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

	publishRec := findRecord(records, geppettoobs.StageGeppettoPublishDone, string(events.EventTypeInfo), "reasoning-summary")
	if publishRec == nil {
		t.Fatalf("missing publish done reasoning-summary record in %#v", records)
	}
	if publishRec.ItemID != "rs_1" || publishRec.ResponseID != "resp_1" {
		t.Fatalf("publish record did not preserve provider IDs: %#v", publishRec)
	}
	if len(publishRec.EventJSON) == 0 || len(publishRec.MetadataJSON) == 0 {
		t.Fatalf("expected event_json and metadata_json: %#v", publishRec)
	}
	var eventJSON map[string]any
	if err := json.Unmarshal(publishRec.EventJSON, &eventJSON); err != nil {
		t.Fatalf("event_json invalid: %s: %v", string(publishRec.EventJSON), err)
	}
	if eventJSON["message"] != "reasoning-summary" {
		t.Fatalf("event_json missing info message: %s", string(publishRec.EventJSON))
	}
	data, ok := eventJSON["data"].(map[string]any)
	if !ok || data["item_id"] != "rs_1" || data["response_id"] != "resp_1" {
		t.Fatalf("event_json missing enriched provider data: %s", string(publishRec.EventJSON))
	}
	var metadataJSON map[string]any
	if err := json.Unmarshal(publishRec.MetadataJSON, &metadataJSON); err != nil {
		t.Fatalf("metadata_json invalid: %s: %v", string(publishRec.MetadataJSON), err)
	}
	if metadataJSON["turn_id"] != "turn_1" {
		t.Fatalf("metadata_json missing turn_id: %s", string(publishRec.MetadataJSON))
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
