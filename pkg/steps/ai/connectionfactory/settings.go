// Package connectionfactory resolves trusted configuration into API authority.
// Payload engines consume its result and never resolve credentials themselves.
package connectionfactory

import (
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	defaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
)

// New selects exactly the protocol's declared settings slot. A supplied source
// owns token renewal; only the source boundary chooses endpoint and carriers.
func New(s *settings.InferenceSettings, source credentials.BearerTokenSource) (*connection.Connection, error) {
	if s == nil || s.Chat == nil {
		return nil, errors.New("connection factory requires chat settings")
	}
	protocol := "openai"
	if s.Chat.ApiType != nil && strings.TrimSpace(string(*s.Chat.ApiType)) != "" {
		protocol = strings.ToLower(strings.TrimSpace(string(*s.Chat.ApiType)))
	}
	spec := settings.ConnectionSpec{Slot: protocol, TokenSource: source}
	switch protocol {
	case "openai", "anyscale", "fireworks":
		spec.Routes = connection.Routes{"chat": connection.Relative("chat/completions"), "transcribe": connection.Relative("audio/transcriptions")}
		if protocol == "openai" {
			spec.DefaultBaseURL = defaults.OpenAIBaseURL
		}
	case "open-responses":
		spec.Routes = connection.Routes{"responses": connection.Relative("responses"), "input-tokens": connection.Relative("responses/input_tokens")}
	case "claude", "anthropic":
		spec.Slot = "claude"
		spec.DefaultBaseURL = defaults.AnthropicBaseURL
		spec.Routes = api.Routes()
		spec.TokenHeaders, spec.Headers = anthropicHeaders(source != nil)

	case "gemini":
		spec.DefaultBaseURL = defaults.GeminiBaseURL
		spec.TokenHeaders = connection.TokenHeaders{"x-goog-api-key": ""}
	default:
		return nil, errors.Errorf("unsupported connection protocol %q", protocol)
	}
	return settings.ResolveAPIConnection(s.API, s.Client, spec)
}

func anthropicHeaders(oauth bool) (connection.TokenHeaders, http.Header) {
	if oauth {
		return connection.TokenHeaders{"Authorization": "Bearer "}, http.Header{
			"Anthropic-Beta": {"claude-code-20250219,oauth-2025-04-20"}, "User-Agent": {"claude-cli/geppetto"}, "X-App": {"cli"},
		}
	}
	return connection.TokenHeaders{"x-api-key": ""}, http.Header{}
}

// AnthropicAuthentication declares a subscription OAuth or gateway credential.
// Gateway credentials use the dual-header carrier; ordinary static settings use
// x-api-key alone through New.
func AnthropicAuthentication(source credentials.BearerTokenSource, oauth bool) (connection.Authentication, http.Header) {
	carriers, headers := anthropicHeaders(oauth)
	if !oauth {
		carriers["Authorization"] = "Bearer "
	}
	return connection.Token(source, carriers), headers
}
