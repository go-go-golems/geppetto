package claude

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	claudesettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	types2 "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
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

func TestClaudeTokenCounterCountTurn(t *testing.T) {
	model := "claude-sonnet-4-20250514"
	apiType := types2.ApiTypeClaude
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/count_tokens" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("unexpected api key header: %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		payload := string(body)
		if !strings.Contains(payload, `"model":"claude-sonnet-4-20250514"`) {
			t.Fatalf("request missing model: %s", payload)
		}
		if !strings.Contains(payload, `"messages"`) {
			t.Fatalf("request missing messages: %s", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens":31}`))
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
		host:   "api.anthropic.com",
		scheme: "https",
	}

	ss := &aisettings.InferenceSettings{
		API: &aisettings.APISettings{
			APIKeys: map[string]string{"claude-api-key": "test-key"},
			BaseUrls: map[string]string{
				"claude-base-url": "https://api.anthropic.com",
			},
		},
		Client: &aisettings.ClientSettings{
			HTTPClient: httpClient,
		},
		Claude: &claudesettings.Settings{},
		Chat: &aisettings.ChatSettings{
			Engine:  &model,
			ApiType: &apiType,
			Stream:  true,
		},
	}

	counter := NewTokenCounter(ss)
	res, err := counter.CountTurn(context.Background(), &turns.Turn{
		Blocks: []turns.Block{
			turns.NewSystemTextBlock("you are helpful"),
			turns.NewUserTextBlock("hello"),
		},
	})
	if err != nil {
		t.Fatalf("CountTurn returned error: %v", err)
	}
	if res.InputTokens != 31 {
		t.Fatalf("expected 31 input tokens, got %d", res.InputTokens)
	}
}
