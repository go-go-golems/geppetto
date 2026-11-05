package pi

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/systemone"
	"github.com/go-go-golems/geppetto/pkg/security"
)

// LunaRouteProvider is the Pi provider name whose models may fall back to
// systemone.DefaultBaseURL when the catalog has no baseUrl for them.
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
// A model without a base URL gets LunaRoute's endpoint only when its
// provider is LunaRoute. For any other provider that default would send
// that provider's credential to LunaRoute, so an explicit baseUrl is
// required instead.
func (a *AgentDir) Decider(ctx context.Context, ref string, o DecisionOptions) (decision.Provider, error) {
	m, ok := a.Catalog.Lookup(ref)
	if !ok {
		return nil, errors.Wrap(ErrUnknownModel, ref)
	}
	baseURL := m.BaseURL
	if baseURL == "" {
		if m.Provider != LunaRouteProvider {
			return nil, fmt.Errorf("pi: %s has no baseUrl; a System One endpoint must be configured for provider %q: %w",
				ref, m.Provider, decision.ErrInvalidRequest)
		}
		baseURL = systemone.DefaultBaseURL
	}
	src, _, err := a.credential(ctx, m.Provider)
	if err != nil {
		return nil, err
	}
	headers, err := a.headers(ctx, m)
	if err != nil {
		return nil, err
	}
	outbound := security.OutboundURLOptions{}
	if a.opts.insecure {
		outbound = security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true}
	}
	return systemone.New(systemone.Options{
		BaseURL:          baseURL,
		TokenSource:      src,
		Model:            m.ID,
		HTTPClient:       withHeaders(a.opts.httpClient, headers),
		OutboundURL:      outbound,
		MaxRequestBytes:  o.MaxRequestBytes,
		MaxResponseBytes: o.MaxResponseBytes,
	})
}
