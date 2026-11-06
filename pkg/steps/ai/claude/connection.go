package claude

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/pkg/errors"
	"net/http"
)

// Authentication selects Anthropic's API-key, gateway dual-auth or subscription
// OAuth carriers at the source boundary. Request encoders know none of these.
func Authentication(source credentials.BearerTokenSource, oauth bool) (connection.Authentication, http.Header) {
	carriers := connection.TokenHeaders{"x-api-key": ""}
	headers := http.Header{}
	if oauth {
		carriers = connection.TokenHeaders{"Authorization": "Bearer "}
		headers.Set("anthropic-beta", "claude-code-20250219,oauth-2025-04-20")
		headers.Set("user-agent", "claude-cli/geppetto")
		headers.Set("x-app", "cli")
	} else {
		carriers["Authorization"] = "Bearer "
	}
	return connection.Token(source, carriers), headers
}

func resolveClaudeConnection(s *settings.InferenceSettings, source credentials.BearerTokenSource, oauth bool, bearer string) (*connection.Connection, error) {
	if s == nil {
		return nil, errors.New("Claude settings are required")
	}
	carriers := connection.TokenHeaders{"x-api-key": ""}
	headers := http.Header{}
	if source != nil {
		if oauth {
			carriers = connection.TokenHeaders{"Authorization": "Bearer "}
			_, headers = Authentication(source, true)
		} else if bearer == "" {
			carriers["Authorization"] = "Bearer "
		}
	}
	if bearer != "" {
		headers.Set("Authorization", "Bearer "+bearer)
	}
	return settings.ResolveAPIConnection(s.API, s.Client, settings.ConnectionSpec{
		Slot: "claude", DefaultBaseURL: providerdefaults.AnthropicBaseURL, Routes: api.Routes(),
		TokenHeaders: carriers, TokenSource: source, Headers: headers,
	})
}
