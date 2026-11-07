package factory

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"

	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestNewFromSettings_OpenResponsesSupported(t *testing.T) {
	apiType := types.ApiTypeOpenResponses
	ss := &settings.InferenceSettings{
		API: &settings.APISettings{APIKeys: map[string]string{"open-responses-api-key": "test"}, BaseUrls: map[string]string{"open-responses-base-url": "https://responses.example/v1"}},
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromSettings(ss)
	if err != nil {
		t.Fatalf("NewFromSettings returned error: %v", err)
	}
	if counter == nil {
		t.Fatalf("expected counter, got nil")
	}
}

func TestNewFromSettings_RejectsLegacyResponsesAlias(t *testing.T) {
	apiType := types.ApiType("openai-responses")
	ss := &settings.InferenceSettings{Chat: &settings.ChatSettings{ApiType: &apiType}}
	counter, err := NewFromSettings(ss)
	require.Error(t, err)
	require.Nil(t, counter)
}

func TestNewFromSettings_OpenAINotSupported(t *testing.T) {
	apiType := types.ApiTypeOpenAI
	ss := &settings.InferenceSettings{
		API: &settings.APISettings{APIKeys: map[string]string{"open-responses-api-key": "test"}, BaseUrls: map[string]string{"open-responses-base-url": "https://responses.example/v1"}},
		Chat: &settings.ChatSettings{
			ApiType: &apiType,
		},
	}

	counter, err := NewFromSettings(ss)
	if err == nil {
		t.Fatalf("expected error, got counter=%v", counter)
	}
}

func TestCounterFactoryWithConnectionNeedsNoCredentialMaps(t *testing.T) {
	for _, protocol := range []types.ApiType{types.ApiTypeOpenResponses, types.ApiTypeClaude} {
		t.Run(string(protocol), func(t *testing.T) {
			calls := 0
			routes := openai_responses.Routes()
			if protocol == types.ApiTypeClaude {
				routes = api.Routes()
			}
			c, err := connection.New(connection.Config{Provider: "issuer", BaseURL: "https://gateway.example", Routes: routes, Auth: connection.StaticBearer("TOKEN"), HTTPClient: &http.Client{Transport: counterTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "gateway.example", req.URL.Host)
				require.Equal(t, "Bearer TOKEN", req.Header.Get("Authorization"))
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"input_tokens":4}`)), Request: req}, nil
			})}})
			require.NoError(t, err)
			s, err := settings.NewInferenceSettings()
			require.NoError(t, err)
			model := "test"
			s.Chat.Engine = &model
			s.Chat.ApiType = &protocol
			s.API = nil
			s.Client = nil
			counter, err := NewFromSettings(s, WithConnection(c))
			require.NoError(t, err)
			result, err := counter.CountTurn(t.Context(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("test")}})
			require.NoError(t, err)
			require.Equal(t, 4, result.InputTokens)
			require.Equal(t, 1, calls)
		})
	}
}

type counterTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = counterTransport(nil)

func (f counterTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
