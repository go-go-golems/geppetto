package factory

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/stretchr/testify/require"
)

func TestEngineFactoryWithConnectionNeedsNoCredentialMaps(t *testing.T) {
	for _, protocol := range []types.ApiType{types.ApiTypeOpenAI, types.ApiTypeOpenResponses, types.ApiTypeClaude, types.ApiTypeGemini} {
		t.Run(string(protocol), func(t *testing.T) {
			calls := 0
			routes := openai.Routes()
			carrier := connection.TokenHeaders{"Authorization": "Bearer "}
			if protocol == types.ApiTypeOpenResponses {
				routes = openai_responses.Routes()
			}
			if protocol == types.ApiTypeClaude {
				routes = api.Routes()
				carrier = connection.TokenHeaders{"x-api-key": ""}
			}
			if protocol == types.ApiTypeGemini {
				carrier = connection.TokenHeaders{"x-goog-api-key": ""}
			}
			c, err := connection.New(connection.Config{Provider: "issuer", BaseURL: "https://gateway.example/v1", Routes: routes, Auth: connection.StaticToken("FACTORY-SECRET", carrier), HTTPClient: &http.Client{Transport: factoryRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "gateway.example", req.URL.Host)
				for name, prefix := range carrier {
					require.Equal(t, prefix+"FACTORY-SECRET", req.Header.Get(name))
				}
				return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"test"}}`)), Request: req}, nil
			})}})
			require.NoError(t, err)
			s, err := settings.NewInferenceSettings()
			require.NoError(t, err)
			model := "test"
			s.Chat.Engine = &model
			s.Chat.ApiType = &protocol
			s.API = nil
			s.Client = nil
			e, err := NewStandardEngineFactory(WithConnection(c)).CreateEngine(s)
			require.NoError(t, err)
			_, _ = e.RunInference(t.Context(), &turns.Turn{Blocks: []turns.Block{turns.NewUserTextBlock("test")}})
			require.Equal(t, 1, calls)
		})
	}
}
