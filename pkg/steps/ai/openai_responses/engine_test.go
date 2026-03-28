package openai_responses

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	inferencetools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

type capturingEventSink struct {
	mu     sync.Mutex
	events []events.Event
}

func (s *capturingEventSink) PublishEvent(event events.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *capturingEventSink) snapshot() []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, len(s.events))
	copy(out, s.events)
	return out
}

func ptr[T any](v T) *T { return &v }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type responseHeaderTransport struct {
	base   http.RoundTripper
	target *url.URL
	host   string
	scheme string
	header string
	value  string
}

func (t *responseHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = t.target.Scheme
	req2.URL.Host = t.target.Host
	req2.Host = t.target.Host
	req2.Header = req.Header.Clone()
	if t.scheme != "" {
		req2.Header.Set("X-Original-Scheme", t.scheme)
	}
	if t.host != "" {
		req2.Header.Set("X-Original-Host", t.host)
	}
	req2.Header.Set(t.header, t.value)
	return t.base.RoundTrip(req2)
}

func TestRunInference_StreamingErrorReturnsFailureAndNoFinalEvent(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := "event: error\n" +
				"data: {\"error\":{\"message\":\"stream broke\",\"code\":\"upstream_failure\"}}\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-4o-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err == nil {
		t.Fatalf("expected streaming error to be returned")
	}
	if !strings.Contains(err.Error(), "stream") {
		t.Fatalf("expected stream-related error, got %q", err.Error())
	}

	var errorEvents int
	var finalEvents int
	for _, event := range sink.snapshot() {
		if event.Type() == events.EventTypeError {
			errorEvents++
		}
		if event.Type() == events.EventTypeFinal {
			finalEvents++
		}
	}
	if errorEvents == 0 {
		t.Fatalf("expected at least one error event")
	}
	if finalEvents != 0 {
		t.Fatalf("did not expect final success event on streaming failure, got %d", finalEvents)
	}
}

func TestRunInference_UsesConfiguredHTTPClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Transport"); got != "responses-proxy" {
			t.Fatalf("expected custom transport header, got %q", got)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			"event: response.output_item.added",
			`data: {"item":{"type":"message","id":"msg_1"}}`,
			"",
			"event: response.output_text.delta",
			`data: {"delta":"hello"}`,
			"",
			"event: response.output_item.done",
			`data: {"item":{"type":"message","id":"msg_1"}}`,
			"",
			"event: response.completed",
			`data: {"response":{"usage":{"input_tokens":1,"output_tokens":1}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	httpClient := server.Client()
	httpClient.Transport = &responseHeaderTransport{
		base:   httpClient.Transport,
		target: targetURL,
		host:   "api.openai.com",
		scheme: "https",
		header: "X-Test-Transport",
		value:  "responses-proxy",
	}

	eng, err := NewEngine(&settings.InferenceSettings{
		Client: &settings.ClientSettings{
			HTTPClient: httpClient,
		},
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-4o-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("Hello"),
	}}
	if _, err := eng.RunInference(context.Background(), turn); err != nil {
		t.Fatalf("RunInference: %v", err)
	}
}

func TestAttachToolsToResponsesRequest_IgnoresPersistedToolDefinitionsWithoutRuntimeRegistry(t *testing.T) {
	eng := &Engine{}
	turn := &turns.Turn{}
	if err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true}); err != nil {
		t.Fatalf("set tool config: %v", err)
	}
	if err := engine.KeyToolDefinitions.Set(&turn.Data, engine.ToolDefinitions{
		{
			Name:        "persisted_only",
			Description: "Should not be advertised from turn data",
			Parameters: map[string]any{
				"type": "object",
			},
		},
	}); err != nil {
		t.Fatalf("set tool definitions: %v", err)
	}

	reqBody := &responsesRequest{}
	if err := eng.attachToolsToResponsesRequest(context.Background(), turn, reqBody); err != nil {
		t.Fatalf("attachToolsToResponsesRequest: %v", err)
	}
	if len(reqBody.Tools) != 0 {
		t.Fatalf("expected no advertised tools without a live registry, got %#v", reqBody.Tools)
	}
}

func TestAttachToolsToResponsesRequest_UsesRuntimeRegistryInsteadOfPersistedSnapshots(t *testing.T) {
	eng := &Engine{}
	turn := &turns.Turn{}
	if err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{Enabled: true}); err != nil {
		t.Fatalf("set tool config: %v", err)
	}
	if err := engine.KeyToolDefinitions.Set(&turn.Data, engine.ToolDefinitions{
		{
			Name:        "persisted_only",
			Description: "Should not be advertised from turn data",
			Parameters: map[string]any{
				"type": "object",
			},
		},
	}); err != nil {
		t.Fatalf("set tool definitions: %v", err)
	}

	reg := inferencetools.NewInMemoryToolRegistry()
	type runtimeIn struct {
		Text string `json:"text"`
	}
	runtimeTool, err := inferencetools.NewToolFromFunc("runtime_echo", "Echo runtime text", func(in runtimeIn) (map[string]any, error) {
		return map[string]any{"echo": in.Text}, nil
	})
	if err != nil {
		t.Fatalf("NewToolFromFunc: %v", err)
	}
	if err := reg.RegisterTool("runtime_echo", *runtimeTool); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	reqBody := &responsesRequest{}
	ctx := inferencetools.WithRegistry(context.Background(), reg)
	if err := eng.attachToolsToResponsesRequest(ctx, turn, reqBody); err != nil {
		t.Fatalf("attachToolsToResponsesRequest: %v", err)
	}
	if len(reqBody.Tools) != 1 {
		t.Fatalf("expected one advertised runtime tool, got %#v", reqBody.Tools)
	}

	toolMap, ok := reqBody.Tools[0].(map[string]any)
	if !ok {
		t.Fatalf("expected tool payload as map, got %T", reqBody.Tools[0])
	}
	if toolMap["name"] != "runtime_echo" {
		t.Fatalf("expected runtime registry tool name, got %#v", toolMap["name"])
	}
	if toolMap["name"] == "persisted_only" {
		t.Fatalf("unexpected persisted turn snapshot in advertised tools")
	}
}

func TestRunInference_StreamingReasoningTextEventsArePublished(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.reasoning_summary_text.delta",
				`data: {"delta":"Short summary."}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"Thinking "}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"hard."}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":"Thinking hard."}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_1","encrypted_content":"enc_1"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"42"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":10,"output_tokens":5,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":3}}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	out, err := eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasoningDeltaEvents int
	var reasoningDoneEvents int
	var thinkingPartialEvents int
	var finalEvents int
	var finalThinkingText string
	var finalUsage *events.Usage
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventReasoningTextDelta:
			reasoningDeltaEvents++
		case *events.EventReasoningTextDone:
			reasoningDoneEvents++
		case *events.EventThinkingPartial:
			thinkingPartialEvents++
		case *events.EventFinal:
			finalEvents++
			finalUsage = e.Metadata().Usage
			if e.Metadata().Extra != nil {
				if s, ok := e.Metadata().Extra["thinking_text"].(string); ok {
					finalThinkingText = s
				}
			}
		}
	}

	if reasoningDeltaEvents == 0 {
		t.Fatalf("expected reasoning delta events")
	}
	if reasoningDoneEvents == 0 {
		t.Fatalf("expected reasoning done events")
	}
	if thinkingPartialEvents == 0 {
		t.Fatalf("expected mirrored partial-thinking events for reasoning text")
	}
	if finalEvents != 1 {
		t.Fatalf("expected exactly one final event, got %d", finalEvents)
	}
	if finalThinkingText != "Thinking hard." {
		t.Fatalf("expected final metadata thinking_text to be propagated, got %q", finalThinkingText)
	}
	if finalUsage == nil {
		t.Fatalf("expected final usage metadata")
	}
	if finalUsage.InputTokens != 10 || finalUsage.OutputTokens != 5 || finalUsage.CachedTokens != 4 {
		t.Fatalf("expected usage input=10 output=5 cached=4, got input=%d output=%d cached=%d", finalUsage.InputTokens, finalUsage.OutputTokens, finalUsage.CachedTokens)
	}
	if out == nil || len(out.Blocks) < 1 {
		t.Fatalf("expected output turn blocks")
	}
	var reasoningBlock *turns.Block
	for i := range out.Blocks {
		if out.Blocks[i].Kind == turns.BlockKindReasoning {
			reasoningBlock = &out.Blocks[i]
			break
		}
	}
	if reasoningBlock == nil {
		t.Fatalf("expected persisted reasoning block")
	}
	if got, _ := reasoningBlock.Payload[turns.PayloadKeyText].(string); got != "Thinking hard." {
		t.Fatalf("expected persisted reasoning text, got %q", got)
	}
	if got, _ := reasoningBlock.Payload[turns.PayloadKeyEncryptedContent].(string); got != "enc_1" {
		t.Fatalf("expected persisted encrypted reasoning content, got %q", got)
	}
	summary, ok := reasoningBlock.Payload[turns.PayloadKeySummary].([]any)
	if !ok || len(summary) != 1 {
		t.Fatalf("expected persisted reasoning summary payload, got %#v", reasoningBlock.Payload[turns.PayloadKeySummary])
	}
}

func TestRunInference_StreamingReasoningAliasEventsAreNormalized(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_alias"}}`,
				"",
				"event: response.reasoning.delta",
				`data: {"delta":"Thinking "}`,
				"",
				"event: response.reasoning.delta",
				`data: {"delta":"alias."}`,
				"",
				"event: response.reasoning.done",
				`data: {"text":"Thinking alias."}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_alias"}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":4,"output_tokens":2}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("Hello"),
	}}

	out, err := eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasoningDeltaEvents int
	var reasoningDoneEvents int
	for _, event := range sink.snapshot() {
		switch event.(type) {
		case *events.EventReasoningTextDelta:
			reasoningDeltaEvents++
		case *events.EventReasoningTextDone:
			reasoningDoneEvents++
		}
	}
	if reasoningDeltaEvents != 2 {
		t.Fatalf("expected normalized reasoning delta events, got %d", reasoningDeltaEvents)
	}
	if reasoningDoneEvents != 1 {
		t.Fatalf("expected normalized reasoning done events, got %d", reasoningDoneEvents)
	}

	var reasoningBlock *turns.Block
	for i := range out.Blocks {
		if out.Blocks[i].Kind == turns.BlockKindReasoning {
			reasoningBlock = &out.Blocks[i]
			break
		}
	}
	if reasoningBlock == nil {
		t.Fatalf("expected persisted reasoning block")
	}
	if got, _ := reasoningBlock.Payload[turns.PayloadKeyText].(string); got != "Thinking alias." {
		t.Fatalf("expected normalized reasoning text persistence, got %q", got)
	}
}

func TestRunInference_StreamingReasoningTextDonePreservesAccumulatedThinking(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"First "}`,
				"",
				"event: response.reasoning_text.delta",
				`data: {"delta":"thought."}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":"First thought.","item_id":"rs_1"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_1"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"reasoning","id":"rs_2"}}`,
				"",
				"event: response.reasoning_text.done",
				`data: {"text":" Second thought.","item_id":"rs_2"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"reasoning","id":"rs_2"}}`,
				"",
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"42"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":10,"output_tokens":5,"output_tokens_details":{"reasoning_tokens":3}}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasoningDoneEvents int
	var finalThinkingText string
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventReasoningTextDone:
			reasoningDoneEvents++
		case *events.EventFinal:
			if e.Metadata().Extra != nil {
				if s, ok := e.Metadata().Extra["thinking_text"].(string); ok {
					finalThinkingText = s
				}
			}
		}
	}

	if reasoningDoneEvents != 2 {
		t.Fatalf("expected two reasoning done events, got %d", reasoningDoneEvents)
	}
	if finalThinkingText != "First thought. Second thought." {
		t.Fatalf("expected combined thinking_text, got %q", finalThinkingText)
	}
}

func TestRunInference_StreamingOutputItemDoneDoesNotDuplicateStreamedText(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"Hel","item_id":"msg_1"}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"lo","item_id":"msg_1"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello"}]}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":1,"output_tokens":1}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var finalText string
	var partialDeltas []string
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventPartialCompletion:
			partialDeltas = append(partialDeltas, e.Delta)
		case *events.EventFinal:
			finalText = e.Text
		}
	}

	if finalText != "Hello" {
		t.Fatalf("expected final text %q, got %q", "Hello", finalText)
	}
	if strings.Join(partialDeltas, "") != "Hello" {
		t.Fatalf("expected partial deltas to compose %q, got %q", "Hello", strings.Join(partialDeltas, ""))
	}
	if len(partialDeltas) != 2 {
		t.Fatalf("expected exactly two streamed partial deltas, got %d (%v)", len(partialDeltas), partialDeltas)
	}
}

func TestRunInference_StreamingOutputItemDoneBackfillsMissingTail(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"Hel","item_id":"msg_1"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello"}]}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":1,"output_tokens":1}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var finalText string
	var partialDeltas []string
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventPartialCompletion:
			partialDeltas = append(partialDeltas, e.Delta)
		case *events.EventFinal:
			finalText = e.Text
		}
	}

	if finalText != "Hello" {
		t.Fatalf("expected final text %q, got %q", "Hello", finalText)
	}
	if strings.Join(partialDeltas, "") != "Hello" {
		t.Fatalf("expected partial deltas to compose %q, got %q", "Hello", strings.Join(partialDeltas, ""))
	}
	if len(partialDeltas) != 2 {
		t.Fatalf("expected two partial deltas (stream + backfill), got %d (%v)", len(partialDeltas), partialDeltas)
	}
	if partialDeltas[1] != "lo" {
		t.Fatalf("expected done backfill delta %q, got %q", "lo", partialDeltas[1])
	}
}

func TestRunInference_StreamingPreservesWhitespaceOnlyDelta(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"Hello","item_id":"msg_1"}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":" ","item_id":"msg_1"}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"world","item_id":"msg_1"}`,
				"",
				"event: response.output_item.done",
				`data: {"item":{"type":"message","id":"msg_1","content":[{"type":"output_text","text":"Hello world"}]}}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":1,"output_tokens":1}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var finalText string
	var partialDeltas []string
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventPartialCompletion:
			partialDeltas = append(partialDeltas, e.Delta)
		case *events.EventFinal:
			finalText = e.Text
		}
	}

	if finalText != "Hello world" {
		t.Fatalf("expected final text %q, got %q", "Hello world", finalText)
	}
	if strings.Join(partialDeltas, "") != "Hello world" {
		t.Fatalf("expected partial deltas to compose %q, got %q", "Hello world", strings.Join(partialDeltas, ""))
	}
	if len(partialDeltas) != 3 {
		t.Fatalf("expected exactly three streamed partial deltas, got %d (%v)", len(partialDeltas), partialDeltas)
	}
	if partialDeltas[1] != " " {
		t.Fatalf("expected preserved whitespace-only delta %q, got %q", " ", partialDeltas[1])
	}
}

func TestRunInference_NonStreamingUsageIncludesCachedTokens(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := `{
  "output": [
    {
      "type": "message",
      "id": "msg_1",
      "content": [
        {"type": "output_text", "text": "Done"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 12,
    "output_tokens": 7,
    "input_tokens_details": {"cached_tokens": 5},
    "output_tokens_details": {"reasoning_tokens": 2}
  }
}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-5-mini"),
			Stream: false,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	sink := &capturingEventSink{}
	ctx := events.WithEventSinks(context.Background(), sink)
	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}

	_, err = eng.RunInference(ctx, turn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var finalEvents int
	var finalUsage *events.Usage
	var reasoningTokens int
	for _, event := range sink.snapshot() {
		switch e := event.(type) {
		case *events.EventFinal:
			finalEvents++
			finalUsage = e.Metadata().Usage
			if e.Metadata().Extra != nil {
				if v, ok := e.Metadata().Extra["reasoning_tokens"].(int); ok {
					reasoningTokens = v
				}
			}
		}
	}

	if finalEvents != 1 {
		t.Fatalf("expected exactly one final event, got %d", finalEvents)
	}
	if finalUsage == nil {
		t.Fatalf("expected final usage metadata")
	}
	if finalUsage.InputTokens != 12 || finalUsage.OutputTokens != 7 || finalUsage.CachedTokens != 5 {
		t.Fatalf("expected usage input=12 output=7 cached=5, got input=%d output=%d cached=%d", finalUsage.InputTokens, finalUsage.OutputTokens, finalUsage.CachedTokens)
	}
	if reasoningTokens != 2 {
		t.Fatalf("expected reasoning_tokens=2 in metadata extra, got %d", reasoningTokens)
	}
}

func TestParseUsageTotalsFromResponse_NestedResponseUsage(t *testing.T) {
	rr := responsesResponse{
		Response: &responsesResponseNested{
			Usage: json.RawMessage(`{
  "input_tokens": 9,
  "output_tokens": 4,
  "input_tokens_details": {"cached_tokens": 3},
  "output_tokens_details": {"reasoning_tokens": 1}
}`),
		},
	}

	totals, ok := parseUsageTotalsFromResponse(rr)
	if !ok {
		t.Fatalf("expected usage totals from nested response.usage")
	}
	if totals.inputTokens != 9 || totals.outputTokens != 4 || totals.cachedTokens != 3 || totals.reasoningTokens != 1 {
		t.Fatalf(
			"unexpected totals: input=%d output=%d cached=%d reasoning=%d",
			totals.inputTokens,
			totals.outputTokens,
			totals.cachedTokens,
			totals.reasoningTokens,
		)
	}
}

func TestRunInference_StreamingPersistsCanonicalInferenceResultMetadata(t *testing.T) {
	origClient := http.DefaultClient
	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body := strings.Join([]string{
				"event: response.output_item.added",
				`data: {"item":{"type":"message","id":"msg_1"}}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"Hello"}`,
				"",
				"event: response.completed",
				`data: {"response":{"stop_reason":"max_tokens","usage":{"input_tokens":9,"output_tokens":2}}}`,
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	defer func() { http.DefaultClient = origClient }()

	eng, err := NewEngine(&settings.InferenceSettings{
		API: &settings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": "https://example.test/v1"},
		},
		Chat: &settings.ChatSettings{
			Engine: ptr("gpt-4o-mini"),
			Stream: true,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewSystemTextBlock("You are a LLM."),
		turns.NewUserTextBlock("Hello"),
	}}
	out, err := eng.RunInference(context.Background(), turn)
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if out == nil {
		t.Fatalf("expected output turn")
	}

	res, ok, err := turns.KeyTurnMetaInferenceResult.Get(out.Metadata)
	if err != nil {
		t.Fatalf("get inference_result metadata: %v", err)
	}
	if !ok {
		t.Fatalf("expected canonical inference_result metadata")
	}
	if res.Provider != "open_responses" {
		t.Fatalf("expected provider=open_responses, got %q", res.Provider)
	}
	if res.Model != "gpt-4o-mini" {
		t.Fatalf("expected model=gpt-4o-mini, got %q", res.Model)
	}
	if res.StopReason != "max_tokens" {
		t.Fatalf("expected stop_reason=max_tokens, got %q", res.StopReason)
	}
	if !res.Truncated {
		t.Fatalf("expected truncated=true")
	}
	if res.FinishClass != turns.InferenceFinishClassMaxTokens {
		t.Fatalf("expected finish_class=%q, got %q", turns.InferenceFinishClassMaxTokens, res.FinishClass)
	}
	if res.Usage == nil || res.Usage.InputTokens != 9 || res.Usage.OutputTokens != 2 {
		t.Fatalf("expected usage 9/2, got %+v", res.Usage)
	}

}
