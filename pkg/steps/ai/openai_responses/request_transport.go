package openai_responses

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/pkg/errors"
)

const responsesOperation = "responses"

// RequestTransport declares trusted provider routing and authentication at
// construction time. The resulting connection owns these together.
type RequestTransport struct {
	Provider             string
	RouteResolver        aitransport.RouteResolver
	HeaderRules          []aitransport.HeaderRule
	Middlewares          []aitransport.Middleware
	DisableDefaultBearer bool
}

func WithRequestTransport(config RequestTransport) EngineOption {
	return func(e *Engine) {
		copied := config
		copied.HeaderRules = append([]aitransport.HeaderRule(nil), config.HeaderRules...)
		copied.Middlewares = append([]aitransport.Middleware(nil), config.Middlewares...)
		e.requestTransport = &copied
	}
}

func Routes() connection.Routes {
	return connection.Routes{
		responsesOperation: connection.Relative("responses"),
		"input-tokens":     connection.Relative("responses/input_tokens"),
	}
}

func resolveResponsesConnection(s *settings.InferenceSettings, source credentials.BearerTokenSource, adapter *RequestTransport) (*connection.Connection, error) {
	if s == nil {
		return nil, errors.New("Responses settings are required")
	}
	externalAuth := adapter != nil && adapter.DisableDefaultBearer
	slot, err := resolveResponsesSlot(s.API, source != nil || externalAuth)
	if err != nil {
		return nil, err
	}
	provider := string(responsesAPIType(s))
	routes := Routes()
	var auth connection.Authentication
	if !externalAuth {
		auth = connection.StaticBearer(slot.key)
		if source != nil {
			auth = connection.Bearer(source)
		}
	}
	if adapter != nil {
		if adapter.Provider == "" || adapter.RouteResolver == nil {
			return nil, errors.New("Responses adapter needs provider and route")
		}
		provider = adapter.Provider
		routes[responsesOperation] = adapter.RouteResolver
		auth = connection.Combine(auth, connection.Middleware(adapter.HeaderRules, adapter.Middlewares...))
	}
	c, err := settings.BindConnection(s.Client, connection.Config{
		Provider: provider, BaseURL: slot.baseURL, Routes: routes, Auth: auth,
		OutboundURL: responsesOutboundURLOptions(s.API),
	})
	if err != nil {
		return nil, err
	}
	// Validate protocol routing against the declaration, before source access.
	if _, err := c.Resolve(responsesOperation); err != nil {
		return nil, err
	}
	return c, nil
}

func WithConnection(c *connection.Connection) EngineOption {
	return func(e *Engine) { e.connection = c }
}
