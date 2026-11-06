package pi

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
	providerdefaults "github.com/go-go-golems/geppetto/pkg/steps/ai/providers/defaults"
)

// NoEndpointError: a model has no usable baseUrl and its provider has no
// default endpoint, so there is nowhere its credential may be sent.
type NoEndpointError struct {
	Model    string
	Provider string
	Reason   string
}

func (e *NoEndpointError) Error() string {
	return "pi: " + e.Model + " has no baseUrl (" + e.Reason + "); set baseUrl for provider " + e.Provider + " in models.json"
}

// endpoint returns the base URL a model's requests go to and the origin its
// credential is scoped to. Both come from the same place, the model's
// provider: its normalized baseUrl, else the canonical provider endpoint table. Resolve
// and Decider use the returned baseURL verbatim, so the value checked here
// is the value used.
func (a *AgentDir) endpoint(m Model) (string, security.Origin, error) {
	baseURL := normalizeBaseURL(m.BaseURL)
	reason := "and provider " + m.Provider + " has no default endpoint"
	if baseURL == "" {
		baseURL, _ = providerdefaults.BaseURL(m.Provider)
	}
	if baseURL == "" {
		return "", security.Origin{}, &NoEndpointError{Model: m.Ref(), Provider: m.Provider, Reason: reason}
	}
	origin, err := security.ParseOrigin(baseURL)
	if err != nil {
		return "", security.Origin{}, &NoEndpointError{Model: m.Ref(), Provider: m.Provider, Reason: "baseUrl is not an absolute http(s) URL"}
	}
	return baseURL, origin, nil
}

// normalizeBaseURL drops surrounding space and trailing slashes; " " and "/"
// become "" (missing).
func normalizeBaseURL(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}
