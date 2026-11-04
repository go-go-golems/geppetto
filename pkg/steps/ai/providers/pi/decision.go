package pi

import (
	"context"

	"github.com/pkg/errors"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/systemone"
	"github.com/go-go-golems/geppetto/pkg/security"
)

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
func (a *AgentDir) Decider(ctx context.Context, ref string, o DecisionOptions) (decision.Provider, error) {
	m, ok := a.Catalog.Lookup(ref)
	if !ok {
		return nil, errors.Wrap(ErrUnknownModel, ref)
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
		BaseURL:          first(m.BaseURL, systemone.DefaultBaseURL),
		TokenSource:      src,
		Model:            m.ID,
		HTTPClient:       withHeaders(a.opts.httpClient, headers),
		OutboundURL:      outbound,
		MaxRequestBytes:  o.MaxRequestBytes,
		MaxResponseBytes: o.MaxResponseBytes,
	})
}
