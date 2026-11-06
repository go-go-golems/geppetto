// Package umans binds the verified Umans Anthropic-Messages authentication to
// its issuer endpoint. The resulting connection can be shared with token counting.
package umans

import (
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
)

// NewConnection binds the Umans key, dual-auth headers and routes together.
// Model settings cannot redirect this key to Anthropic's default endpoint.
func NewConnection(apiKey string, client *http.Client) (*connection.Connection, error) {
	return connection.New(connection.Config{
		Provider: "umans", BaseURL: providerdefaults.UmansBaseURL, Routes: api.Routes(), HTTPClient: client,
		Auth: connection.StaticToken(apiKey, connection.TokenHeaders{"x-api-key": "", "Authorization": "Bearer "}),
	})
}
