package pi

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/systemone"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

// LunaRouteProvider is LunaRoute's Pi provider name. Its models default to
// LunaRoute's gateway (providers/defaults) when the catalog has no
// baseUrl for them.
const LunaRouteProvider = "lunaroute"

// DecisionOptions shape a decision provider.
type DecisionOptions struct {
	// MaxRequestBytes and MaxResponseBytes default (8 MiB, 1 MiB) when 0.
	MaxRequestBytes  int64
	MaxResponseBytes int64
}

// Decider returns a decision provider for "provider/id" (for example
// "lunaroute/djev"), using the provider's base URL, models.json headers and
// Pi's credential, with the same precedence as Resolve (auth.json, then
// models.json apiKey).
//
// Pi's catalog does not record which models are decision models (it lists
// djev as openai-completions), so the API and Supported fields are not
// checked here; a chat model id is rejected by the server with
// decision.ErrInvalidRequest.
//
// The endpoint comes from the model's provider (see endpoint): its baseUrl,
// else that provider's own default, else a NoEndpointError that also wraps
// decision.ErrInvalidRequest. The credential is scoped and the client pinned
// to that origin.
func (a *AgentDir) Decider(ctx context.Context, ref string, o DecisionOptions) (decision.Provider, error) {
	m, ok := a.Catalog.Lookup(ref)
	if !ok {
		return nil, errors.Wrap(ErrUnknownModel, ref)
	}
	baseURL, origin, err := a.endpoint(m)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", err, decision.ErrInvalidRequest)
	}
	src, _, err := a.credential(ctx, m.Provider)
	if err != nil {
		return nil, err
	}
	headers, err := a.headers(ctx, m)
	if err != nil {
		return nil, err
	}
	c, err := a.bindConnection(m, baseURL, connection.Bearer(credentials.Scoped(src, origin)), systemone.Routes(), headers)
	if err != nil {
		return nil, err
	}
	return systemone.New(systemone.Options{
		Connection:       c,
		Model:            m.ID,
		MaxRequestBytes:  o.MaxRequestBytes,
		MaxResponseBytes: o.MaxResponseBytes,
	})
}
