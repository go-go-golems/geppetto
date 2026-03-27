package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infengine "github.com/go-go-golems/geppetto/pkg/inference/engine"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aisettingsopenai "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	ai_types "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

func boolPtr(v bool) *bool { return &v }

func newTestEngine(st *aisettings.InferenceSettings) *OpenAIEngine {
	return &OpenAIEngine{settings: st}
}

func TestMakeCompletionRequestFromTurnStructuredOutput(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                 &engine,
			StructuredOutputMode:   aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:   "person",
			StructuredOutputSchema: `{"type":"object","properties":{"name":{"type":"string"}}}`,
			StructuredOutputStrict: boolPtr(true),
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.JSONSchema == nil {
		t.Fatalf("expected structured response_format to be set")
	}
	if req.ResponseFormat.Type != "json_schema" {
		t.Fatalf("expected response format type json_schema, got %q", req.ResponseFormat.Type)
	}
	if req.ResponseFormat.JSONSchema.Name != "person" {
		t.Fatalf("expected schema name person, got %q", req.ResponseFormat.JSONSchema.Name)
	}
}

func TestMakeCompletionRequestFromTurnStructuredOutputInvalidSchemaIgnoredWhenNotRequired(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:                       &engine,
			StructuredOutputMode:         aisettings.StructuredOutputModeJSONSchema,
			StructuredOutputName:         "person",
			StructuredOutputSchema:       `{"type":"object",`,
			StructuredOutputRequireValid: false,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("return JSON"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.ResponseFormat != nil {
		t.Fatalf("expected invalid schema to be ignored when require_valid=false")
	}
}

func TestMakeCompletionRequestFromTurnReasoningModelSanitizesPenalties(t *testing.T) {
	engine := "o3-mini"
	pp := 0.5
	fp := 0.3
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{
			PresencePenalty:  &pp,
			FrequencyPenalty: &fp,
		},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Reasoning models should have penalties zeroed
	if req.PresencePenalty != 0 {
		t.Errorf("expected PresencePenalty=0 for reasoning model, got %v", req.PresencePenalty)
	}
	if req.FrequencyPenalty != 0 {
		t.Errorf("expected FrequencyPenalty=0 for reasoning model, got %v", req.FrequencyPenalty)
	}
	if req.Temperature != 0 {
		t.Errorf("expected Temperature=0 for reasoning model, got %v", req.Temperature)
	}
}

func TestMakeCompletionRequestFromTurnInferenceEmptyStopClearsChatStop(t *testing.T) {
	engine := "gpt-4o-mini"
	st := &aisettings.InferenceSettings{
		Client: &aisettings.ClientSettings{},
		OpenAI: &aisettingsopenai.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine: &engine,
			Stop:   []string{"<END>"},
		},
	}
	tu := &turns.Turn{Blocks: []turns.Block{
		turns.NewUserTextBlock("hello"),
	}}
	if err := infengine.KeyInferenceConfig.Set(&tu.Data, infengine.InferenceConfig{Stop: []string{}}); err != nil {
		t.Fatalf("failed to set inference config: %v", err)
	}

	e := newTestEngine(st)
	req, err := e.MakeCompletionRequestFromTurn(tu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Stop == nil {
		t.Fatalf("expected explicit empty stop override to produce non-nil empty stop")
	}
	if len(req.Stop) != 0 {
		t.Fatalf("expected stop override to clear chat stop, got %v", req.Stop)
	}
}

type headerTransport struct {
	base   http.RoundTripper
	header string
	value  string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header = req.Header.Clone()
	req2.Header.Set(t.header, t.value)
	return t.base.RoundTrip(req2)
}

func TestMakeClient_UsesConfiguredHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Transport"); got != "openai-proxy" {
			t.Fatalf("expected custom transport header, got %q", got)
		}
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data":   []map[string]any{},
		})
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = &headerTransport{
		base:   httpClient.Transport,
		header: "X-Test-Transport",
		value:  "openai-proxy",
	}

	client, err := MakeClient(
		&aisettings.APISettings{
			APIKeys:  map[string]string{"openai-api-key": "test"},
			BaseUrls: map[string]string{"openai-base-url": server.URL + "/v1"},
		},
		&aisettings.ClientSettings{
			HTTPClient: httpClient,
		},
		ai_types.ApiTypeOpenAI,
	)
	if err != nil {
		t.Fatalf("MakeClient: %v", err)
	}

	_, err = client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
}
