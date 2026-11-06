package settings

import (
	"net/http"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	"github.com/pkg/errors"
)

// ConnectionSpec describes how one settings slot declares an API connection.
// Defaults and token carriers are chosen at this boundary, never by encoders.
type ConnectionSpec struct {
	Slot           string
	DefaultBaseURL string
	Routes         connection.Routes
	TokenHeaders   connection.TokenHeaders
	TokenSource    credentials.BearerTokenSource
	Headers        http.Header
}

// ResolveAPIConnection reads one slot's key and endpoint together, then seals
// them in a connection. An explicitly blank URL is an error, not a fallback.
func ResolveAPIConnection(api *APISettings, client *ClientSettings, spec ConnectionSpec) (*connection.Connection, error) {
	baseURL, key := spec.DefaultBaseURL, ""
	if api != nil {
		if declared, ok := api.BaseUrls[spec.Slot+"-base-url"]; ok {
			baseURL = declared
		}
		key = api.APIKeys[spec.Slot+"-api-key"]
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.Errorf("no base URL for %s", spec.Slot)
	}
	carriers := spec.TokenHeaders
	if len(carriers) == 0 {
		carriers = connection.TokenHeaders{"Authorization": "Bearer "}
	}
	if spec.TokenSource == nil && strings.TrimSpace(key) == "" {
		return nil, errors.Errorf("missing API key %s-api-key", spec.Slot)
	}
	auth := connection.StaticToken(key, carriers)
	if spec.TokenSource != nil {
		auth = connection.Token(spec.TokenSource, carriers)
	}
	return BindConnection(client, connection.Config{
		Provider: spec.Slot, BaseURL: baseURL, Routes: spec.Routes,
		Auth: auth, Headers: spec.Headers, OutboundURL: OutboundURLOptions(api, spec.Slot),
	})
}

// BindConnection applies host timeout/proxy/TLS settings at a source boundary.
func BindConnection(client *ClientSettings, config connection.Config) (*connection.Connection, error) {
	httpClient, err := EnsureHTTPClient(client)
	if err != nil {
		return nil, err
	}
	config.HTTPClient = httpClient
	return connection.New(config)
}

// PayloadSettings snapshots model/encoding settings without carrying credential
// maps or host clients into an engine. The bound connection owns API authority.
func PayloadSettings(s *InferenceSettings) *InferenceSettings {
	if s == nil {
		return nil
	}
	payload := *s
	payload.Client = nil
	if s.API != nil {
		payload.API = s.API.Clone()
		payload.API.APIKeys = nil
	}
	if s.Chat != nil {
		payload.Chat = s.Chat.Clone()
		payload.Chat.APIKeys = nil
	}
	if s.Embeddings != nil {
		payload.Embeddings = s.Embeddings.Clone()
		payload.Embeddings.APIKeys = nil
	}
	return &payload
}
