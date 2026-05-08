package factory

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type captureFactoryObserver struct {
	mu      sync.Mutex
	records []geppettoobs.Record
}

func (o *captureFactoryObserver) OnGeppettoRecord(_ context.Context, rec geppettoobs.Record) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records = append(o.records, rec)
}

func (o *captureFactoryObserver) snapshot() []geppettoobs.Record {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]geppettoobs.Record, len(o.records))
	copy(out, o.records)
	return out
}

type factoryRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f factoryRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type factoryRewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t *factoryRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = t.target.Scheme
	req2.URL.Host = t.target.Host
	req2.Host = t.target.Host
	req2.Header = req.Header.Clone()
	return t.base.RoundTrip(req2)
}

func TestStandardEngineFactory_WithOpenAIOptionsPassesObservabilityOptions(t *testing.T) {
	obs := &captureFactoryObserver{}
	factory := NewStandardEngineFactory(
		WithOpenAIOptions(
			openai.WithObserver(obs),
			openai.WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceEvents}),
		),
	)
	settings := createValidOpenAISettings()
	engineName := "gpt-test"
	settings.Chat.Engine = &engineName
	settings.Client.HTTPClient = &http.Client{Transport: factoryRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body := strings.Join([]string{
			`data: {"id":"chatcmpl-factory","object":"chat.completion.chunk","model":"gpt-test","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}

	eng, err := factory.CreateEngine(settings)
	if err != nil {
		t.Fatalf("CreateEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_factory", Blocks: []turns.Block{turns.NewUserTextBlock("say ok")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	var sawStarted bool
	for _, rec := range records {
		if rec.Stage == geppettoobs.StageProviderRoutedEvent {
			t.Fatalf("TraceEvents should not emit provider records: %#v", rec)
		}
		if rec.Stage == geppettoobs.StageGeppettoPublishStarted && rec.EventType == "final" && rec.TurnID == "turn_factory" {
			sawStarted = true
		}
		if rec.Stage == geppettoobs.StageGeppettoPublishDone {
			t.Fatalf("did not expect publish-done records: %#v", rec)
		}
	}
	if !sawStarted {
		t.Fatalf("factory-created OpenAI engine did not emit final publish-started record: %#v", records)
	}
}

func TestStandardEngineFactory_WithClaudeOptionsPassesObservabilityOptions(t *testing.T) {
	obs := &captureFactoryObserver{}
	factory := NewStandardEngineFactory(
		WithClaudeOptions(
			claude.WithObserver(obs),
			claude.WithObservabilityConfig(geppettoobs.Config{Level: geppettoobs.TraceEvents}),
		),
	)
	settings := createValidClaudeSettings()
	engineName := "claude-test"
	settings.Chat.Engine = &engineName
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := strings.Join([]string{
			"event: message_start",
			`data: {"type":"message_start","message":{"id":"msg_factory","type":"message","role":"assistant","content":[],"model":"claude-test","stop_reason":"","stop_sequence":"","usage":{"input_tokens":1,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
			"",
			"event: content_block_start",
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			"",
			"event: content_block_delta",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`,
			"",
			"event: content_block_stop",
			`data: {"type":"content_block_stop","index":0}`,
			"",
			"event: message_stop",
			`data: {"type":"message_stop"}`,
			"",
		}, "\n") + "\n"
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	httpClient := server.Client()
	httpClient.Transport = &factoryRewriteTransport{base: httpClient.Transport, target: targetURL}
	settings.Client.HTTPClient = httpClient

	eng, err := factory.CreateEngine(settings)
	if err != nil {
		t.Fatalf("CreateEngine: %v", err)
	}
	turn := &turns.Turn{ID: "turn_claude_factory", Blocks: []turns.Block{turns.NewUserTextBlock("say ok")}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}

	records := obs.snapshot()
	var sawStarted bool
	for _, rec := range records {
		if rec.Stage == geppettoobs.StageProviderRoutedEvent {
			t.Fatalf("TraceEvents should not emit provider records: %#v", rec)
		}
		if rec.Stage == geppettoobs.StageGeppettoPublishStarted && rec.EventType == string(events.EventTypeProviderCallFinished) && rec.TurnID == "turn_claude_factory" {
			sawStarted = true
		}
	}
	if !sawStarted {
		t.Fatalf("factory-created Claude engine did not emit provider-call-finished publish-started record: %#v", records)
	}
}
