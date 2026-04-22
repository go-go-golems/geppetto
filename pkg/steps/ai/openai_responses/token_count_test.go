package openai_responses

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tokencount"
	inferencetools "github.com/go-go-golems/geppetto/pkg/inference/tools"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	types2 "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/invopop/jsonschema"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
	host   string
	scheme string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = rt.target.Scheme
	req2.URL.Host = rt.target.Host
	req2.Host = rt.target.Host
	if rt.scheme != "" {
		req2.Header.Set("X-Original-Scheme", rt.scheme)
	}
	if rt.host != "" {
		req2.Header.Set("X-Original-Host", rt.host)
	}
	return rt.base.RoundTrip(req2)
}

func TestTokenCounterCountTurn(t *testing.T) {
	model := "gpt-4o-mini"
	apiType := types2.ApiTypeOpenAIResponses
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses/input_tokens" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		payload := string(body)
		if !strings.Contains(payload, `"model":"gpt-4o-mini"`) {
			t.Fatalf("request missing model: %s", payload)
		}
		if !strings.Contains(payload, `"input"`) {
			t.Fatalf("request missing input: %s", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens":42}`))
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	httpClient := server.Client()
	httpClient.Transport = &rewriteTransport{
		base:   httpClient.Transport,
		target: targetURL,
		host:   "api.openai.com",
		scheme: "https",
	}

	ss := &aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys: map[string]string{"openai-api-key": "test-key"},
			BaseUrls: map[string]string{
				"openai-base-url": "https://api.openai.com/v1",
			},
		},
		Client: &aisettings.ClientSettings{
			HTTPClient: httpClient,
		},
		Chat: &aisettings.ChatSettings{
			Engine:  &model,
			ApiType: &apiType,
			Stream:  true,
		},
		OpenAI: &openaisettings.Settings{},
	}

	counter := NewTokenCounter(ss)
	res, err := counter.CountTurn(context.Background(), &turns.Turn{
		Blocks: []turns.Block{turns.NewUserTextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("CountTurn returned error: %v", err)
	}
	if res.InputTokens != 42 {
		t.Fatalf("expected 42 input tokens, got %d", res.InputTokens)
	}
	if res.Provider != "open-responses" {
		t.Fatalf("expected canonical provider open-responses, got %q", res.Provider)
	}
	if res.Source != tokencount.SourceProviderAPI {
		t.Fatalf("expected provider_api source, got %q", res.Source)
	}
}

func TestTokenCounterCountTurn_IncludesImageInputs(t *testing.T) {
	model := "gpt-4o-mini"
	apiType := types2.ApiTypeOpenAIResponses
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		payload := string(body)
		if !strings.Contains(payload, `"type":"input_image"`) {
			t.Fatalf("request missing input_image content part: %s", payload)
		}
		if !strings.Contains(payload, `"image_url":"https://example.com/evidence.png"`) {
			t.Fatalf("request missing image URL: %s", payload)
		}
		if !strings.Contains(payload, `"detail":"high"`) {
			t.Fatalf("request missing image detail: %s", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens":11}`))
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	httpClient := server.Client()
	httpClient.Transport = &rewriteTransport{
		base:   httpClient.Transport,
		target: targetURL,
		host:   "api.openai.com",
		scheme: "https",
	}

	ss := &aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys: map[string]string{"openai-api-key": "test-key"},
			BaseUrls: map[string]string{
				"openai-base-url": "https://api.openai.com/v1",
			},
		},
		Client: &aisettings.ClientSettings{
			HTTPClient: httpClient,
		},
		Chat: &aisettings.ChatSettings{
			Engine:  &model,
			ApiType: &apiType,
			Stream:  true,
		},
		OpenAI: &openaisettings.Settings{},
	}

	counter := NewTokenCounter(ss)
	res, err := counter.CountTurn(context.Background(), &turns.Turn{
		Blocks: []turns.Block{turns.NewUserMultimodalBlock("Inspect this screenshot", []map[string]any{{
			"media_type": "image/png",
			"url":        "https://example.com/evidence.png",
			"detail":     "high",
		}})},
	})
	if err != nil {
		t.Fatalf("CountTurn returned error: %v", err)
	}
	if res.InputTokens != 11 {
		t.Fatalf("expected 11 input tokens, got %d", res.InputTokens)
	}
}

func TestParseOpenAIInputTokensNestedResponse(t *testing.T) {
	got, err := parseOpenAIInputTokens([]byte(`{"response":{"input_tokens":17}}`))
	if err != nil {
		t.Fatalf("parseOpenAIInputTokens error: %v", err)
	}
	if got != 17 {
		t.Fatalf("expected 17, got %d", got)
	}
}

func TestParseOpenAIInputTokensZeroResponse(t *testing.T) {
	got, err := parseOpenAIInputTokens([]byte(`{"input_tokens":0}`))
	if err != nil {
		t.Fatalf("parseOpenAIInputTokens error: %v", err)
	}
	if got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestTokenCounterCountTurn_AttachesTools(t *testing.T) {
	model := "gpt-4o-mini"
	apiType := types2.ApiTypeOpenAIResponses
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		payload := string(body)
		if !strings.Contains(payload, `"name":"lookup_weather"`) {
			t.Fatalf("request missing tool definition: %s", payload)
		}
		if !strings.Contains(payload, `"parallel_tool_calls":false`) {
			t.Fatalf("request missing parallel tool setting: %s", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens":8}`))
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	httpClient := server.Client()
	httpClient.Transport = &rewriteTransport{
		base:   httpClient.Transport,
		target: targetURL,
		host:   "api.openai.com",
		scheme: "https",
	}

	ss := &aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys: map[string]string{"openai-api-key": "test-key"},
			BaseUrls: map[string]string{
				"openai-base-url": "https://api.openai.com/v1",
			},
		},
		Client: &aisettings.ClientSettings{
			HTTPClient: httpClient,
		},
		Chat: &aisettings.ChatSettings{
			Engine:  &model,
			ApiType: &apiType,
			Stream:  true,
		},
		OpenAI: &openaisettings.Settings{},
	}

	registry := inferencetools.NewInMemoryToolRegistry()
	if err := registry.RegisterTool("lookup_weather", inferencetools.ToolDefinition{
		Name:        "lookup_weather",
		Description: "Look up weather",
		Parameters:  &jsonschema.Schema{Type: "object"},
	}); err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	turn := &turns.Turn{
		Blocks: []turns.Block{turns.NewUserTextBlock("hello")},
	}
	if err := engine.KeyToolConfig.Set(&turn.Data, engine.ToolConfig{
		Enabled:          true,
		MaxParallelTools: 1,
	}); err != nil {
		t.Fatalf("set tool config: %v", err)
	}

	counter := NewTokenCounter(ss)
	res, err := counter.CountTurn(inferencetools.WithRegistry(context.Background(), registry), turn)
	if err != nil {
		t.Fatalf("CountTurn returned error: %v", err)
	}
	if res.InputTokens != 8 {
		t.Fatalf("expected 8 input tokens, got %d", res.InputTokens)
	}
	if res.Provider != "open-responses" {
		t.Fatalf("expected canonical provider open-responses, got %q", res.Provider)
	}
}
