package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	infengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	claudesettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func newTestEngine(st *aisettings.InferenceSettings) *ClaudeEngine {
	return &ClaudeEngine{settings: st}
}

func TestMakeMessageRequestFromTurnStructuredOutput(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                 &engine,
			StructuredOutputMode:   aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:   "person",
			StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
			Stream:                 true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeMessageRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.OutputFormat == nil {
		t.Fatalf("expected output_format to be set")
	}
	if req.OutputFormat.Type != "json_schema" {
		t.Fatalf("expected output_format.type=json_schema, got %q", req.OutputFormat.Type)
	}
	if req.OutputFormat.Name != "person" {
		t.Fatalf("expected output_format.name=person, got %q", req.OutputFormat.Name)
	}
}

func TestMakeMessageRequestFromTurnStructuredOutputInvalidSchemaRequireValid(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                       &engine,
			StructuredOutputMode:         aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: true,
			Stream:                       true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	if _, err := e.MakeMessageRequestFromTurn(tu); err == nil {
		t.Fatalf("expected error when require_valid=true and schema JSON is invalid")
	}
}

func TestMakeMessageRequestFromTurnStructuredOutputInvalidSchemaIgnoredWhenNotRequired(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                       &engine,
			StructuredOutputMode:         aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: false,
			Stream:                       true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeMessageRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.OutputFormat != nil {
		t.Fatalf("expected invalid schema to be ignored when require_valid=false")
	}
}

func TestMakeMessageRequestFromTurnInferenceEmptyStopClearsChatStop(t *testing.T) {
	engine := "claude-sonnet-4-20250514"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
			Stop:   []string{"<END>"},
			Stream: true,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	if err := infengine.KeyInferenceConfig.Set(&tu.Data, infengine.InferenceConfig{Stop: []string{}}); err != nil {
		t.Fatalf("failed to set inference config: %v", err)
	}

	e := newTestEngine(st)
	req, err := e.MakeMessageRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.StopSequences == nil {
		t.Fatalf("expected explicit empty stop override to produce non-nil empty stop list")
	}
	if len(req.StopSequences) != 0 {
		t.Fatalf("expected stop override to clear chat stop, got %v", req.StopSequences)
	}
}

type claudeHeaderTransport struct {
	base   http.RoundTripper
	target *url.URL
	host   string
	scheme string
	header string
	value  string
}

func (t *claudeHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

func TestClaudeRunInference_UsesConfiguredHTTPClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Transport"); got != "claude-proxy" {
			t.Fatalf("expected custom transport header, got %q", got)
		}
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			"event: message_start",
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","stop_sequence":"","usage":{"input_tokens":1,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`,
			"",
			"event: content_block_start",
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			"",
			"event: content_block_delta",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
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
		}, "\n")))
	}))
	defer server.Close()
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
		value:  "claude-proxy",
	}

	engine := "claude-sonnet-4-20250514"
	apiType := ai_types.ApiTypeClaude
	e := newTestEngine(&aisettings.InferenceSettings{
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
	})

	turn := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	out, err := e.RunInference(context.Background(), turn)
	if err != nil {
		t.Fatalf("RunInference: %v", err)
	}
	if out == nil || len(out.Blocks) == 0 {
		t.Fatalf("expected response blocks to be appended")
	}
}
