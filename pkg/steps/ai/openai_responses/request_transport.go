package openai_responses

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

const responsesOperation = "responses"

// RequestTransport configures trusted, Go-only provider behavior for the shared
// OpenAI Responses core. It cannot be represented in profile settings or
// JavaScript values.
type RequestTransport struct {
	Provider             string
	RouteResolver        aitransport.RouteResolver
	HeaderRules          []aitransport.HeaderRule
	Middlewares          []aitransport.Middleware
	DisableDefaultBearer bool
}

type responsesRequestTransport struct {
	request     aitransport.RequestContext
	chain       *aitransport.Chain
	headerRules []aitransport.HeaderRule
}

type responsesRoute struct{}

func (responsesRoute) Resolve(request aitransport.RouteRequest) (*url.URL, error) {
	if request.Operation() != responsesOperation {
		return nil, fmt.Errorf("unsupported Responses operation %q", request.Operation())
	}
	target := request.BaseURL()
	target.Path = strings.TrimRight(target.Path, "/") + "/" + responsesOperation
	target.RawPath = ""
	return &target, nil
}

// WithRequestTransport installs a trusted provider transport adapter for the
// Responses core. The configured route is validated before middleware runs.
// Supplying this option never exposes the adapter to settings or JavaScript.
func WithRequestTransport(config RequestTransport) EngineOption {
	return func(e *Engine) {
		copied := RequestTransport{
			Provider:             config.Provider,
			RouteResolver:        config.RouteResolver,
			HeaderRules:          append([]aitransport.HeaderRule(nil), config.HeaderRules...),
			Middlewares:          append([]aitransport.Middleware(nil), config.Middlewares...),
			DisableDefaultBearer: config.DisableDefaultBearer,
		}
		e.requestTransport = &copied
	}
}

func (e *Engine) newRequestTransport(api *settings.APISettings) (responsesRequestTransport, error) {
	baseURL, err := url.Parse(responsesBaseURL(api))
	if err != nil {
		return responsesRequestTransport{}, errors.New("parse Responses base URL")
	}

	config := RequestTransport{
		Provider:      string(responsesAPIType(e.settings)),
		RouteResolver: responsesRoute{},
	}
	if e.requestTransport != nil {
		config = *e.requestTransport
		config.HeaderRules = append([]aitransport.HeaderRule(nil), e.requestTransport.HeaderRules...)
		config.Middlewares = append([]aitransport.Middleware(nil), e.requestTransport.Middlewares...)
	}
	if strings.TrimSpace(config.Provider) == "" {
		return responsesRequestTransport{}, errors.New("responses request transport provider is required")
	}
	if config.RouteResolver == nil {
		return responsesRequestTransport{}, errors.New("responses request transport route resolver is required")
	}

	request, err := aitransport.ResolveAndValidate(
		config.Provider,
		responsesOperation,
		baseURL,
		config.RouteResolver,
		func(target *url.URL) error {
			return security.ValidateOutboundURL(target.String(), responsesOutboundURLOptions(api))
		},
	)
	if err != nil {
		return responsesRequestTransport{}, fmt.Errorf("invalid Responses URL: %w", err)
	}

	rules := append([]aitransport.HeaderRule(nil), config.HeaderRules...)
	middlewares := append([]aitransport.Middleware(nil), config.Middlewares...)
	if !config.DisableDefaultBearer {
		var err error
		rules, err = appendResponsesHeaderRule(rules, aitransport.HeaderRule{Name: "Authorization", Sensitive: true})
		if err != nil {
			return responsesRequestTransport{}, err
		}
		credentialRequest := credentials.Request{Provider: string(responsesAPIType(e.settings)), BaseURL: responsesBaseURL(api)}
		middlewares = append([]aitransport.Middleware{
			&responsesBearerMiddleware{
				api:               api,
				apiType:           responsesAPIType(e.settings),
				source:            e.bearerTokenSource,
				credentialRequest: credentialRequest,
			},
		}, middlewares...)
	}

	chain, err := aitransport.NewChain(middlewares...)
	if err != nil {
		return responsesRequestTransport{}, fmt.Errorf("build Responses middleware chain: %w", err)
	}
	return responsesRequestTransport{request: request, chain: chain, headerRules: rules}, nil
}

func appendResponsesHeaderRule(rules []aitransport.HeaderRule, rule aitransport.HeaderRule) ([]aitransport.HeaderRule, error) {
	canonicalName := http.CanonicalHeaderKey(strings.TrimSpace(rule.Name))
	for _, existing := range rules {
		if http.CanonicalHeaderKey(strings.TrimSpace(existing.Name)) != canonicalName {
			continue
		}
		if existing.Sensitive != rule.Sensitive {
			return nil, fmt.Errorf("responses request transport header %q has conflicting sensitivity", canonicalName)
		}
		return rules, nil
	}
	return append(rules, rule), nil
}

type responsesBearerMiddleware struct {
	api               *settings.APISettings
	apiType           types.ApiType
	source            credentials.BearerTokenSource
	credentialRequest credentials.Request
}

type responsesBearerAttempt struct {
	token       string
	replacement string
}

func (m *responsesBearerMiddleware) BeforeRequest(ctx context.Context, _ aitransport.RequestContext, previous aitransport.Attempt, headers aitransport.HeaderWriter) (aitransport.Attempt, error) {
	if prior, ok := previous.(*responsesBearerAttempt); ok && strings.TrimSpace(prior.replacement) != "" {
		if err := headers.Set("Authorization", "Bearer "+prior.replacement); err != nil {
			return nil, err
		}
		return &responsesBearerAttempt{token: prior.replacement}, nil
	}
	if previous != nil {
		return nil, errors.New("responses bearer middleware received incompatible prior attempt")
	}

	token, err := resolveResponsesBearerToken(ctx, m.api, m.apiType, m.source)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return &responsesBearerAttempt{}, nil
	}
	if err := headers.Set("Authorization", "Bearer "+token); err != nil {
		return nil, err
	}
	return &responsesBearerAttempt{token: token}, nil
}

func (m *responsesBearerMiddleware) AfterResponse(ctx context.Context, _ aitransport.RequestContext, attempt any, response aitransport.ResponseMetadata) (aitransport.ResponseDecision, error) {
	if response.StatusCode != http.StatusUnauthorized || response.StreamStarted || !response.RetryEligible {
		return aitransport.Continue, nil
	}
	source, ok := m.source.(credentials.UnauthorizedBearerTokenSource)
	if !ok {
		return aitransport.Continue, nil
	}
	bearerAttempt, ok := attempt.(*responsesBearerAttempt)
	if !ok || strings.TrimSpace(bearerAttempt.token) == "" {
		return aitransport.Continue, nil
	}
	replacement, err := source.BearerTokenAfterUnauthorized(ctx, m.credentialRequest, bearerAttempt.token)
	if err != nil {
		return aitransport.Continue, errors.New("refresh bearer credential after provider unauthorized")
	}
	if strings.TrimSpace(replacement) == "" {
		return aitransport.Continue, errors.New("refreshed bearer credential is empty after provider unauthorized")
	}
	bearerAttempt.replacement = replacement
	return aitransport.Retry, nil
}

func redactedResponsesRequestForDebug(request *http.Request, headers *aitransport.HeaderSet) *http.Request {
	if request == nil || headers == nil {
		return request
	}
	redacted := request.Clone(request.Context())
	redacted.Header = headers.RedactedCopy()
	return redacted
}
