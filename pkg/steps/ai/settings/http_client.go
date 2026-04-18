package settings

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const defaultClientSettingsTimeout = 60 * time.Second

type HTTPClientDecision struct {
	Mode                 string   `yaml:"mode"`
	Reasons              []string `yaml:"reasons,omitempty"`
	EffectiveTimeout     string   `yaml:"effective_timeout,omitempty"`
	ProxyURL             string   `yaml:"proxy_url,omitempty"`
	ProxyFromEnvironment bool     `yaml:"proxy_from_environment"`
	ProxyMode            string   `yaml:"proxy_mode,omitempty"`
}

// ExplainHTTPClientDecision reports why EnsureHTTPClient(...) will reuse the
// default client, reuse an injected client, or build a dedicated transport-aware
// client.
func ExplainHTTPClientDecision(cs *ClientSettings) HTTPClientDecision {
	if cs == nil {
		return HTTPClientDecision{
			Mode:                 "default-client",
			Reasons:              []string{"client settings absent; reusing http.DefaultClient"},
			EffectiveTimeout:     defaultClientSettingsTimeout.String(),
			ProxyFromEnvironment: true,
			ProxyMode:            "environment",
		}
	}
	if cs.HTTPClient != nil {
		effectiveTimeout := cs.HTTPClient.Timeout
		if effectiveTimeout == 0 {
			effectiveTimeout = effectiveTimeoutOrDefault(cs)
		}
		return HTTPClientDecision{
			Mode:                 "injected-client",
			Reasons:              []string{"client.HTTPClient was injected explicitly"},
			EffectiveTimeout:     effectiveTimeout.String(),
			ProxyURL:             redactedClientProxyURL(cs),
			ProxyFromEnvironment: proxyFromEnvironmentOrDefault(cs),
			ProxyMode:            proxyModeForSettings(cs),
		}
	}

	timeout := effectiveTimeout(cs)
	proxyURL := strings.TrimSpace(stringValue(cs.ProxyURL))
	useEnv := proxyFromEnvironmentOrDefault(cs)
	reasons := []string{}
	if proxyURL != "" {
		reasons = append(reasons, fmt.Sprintf("client.proxy_url is set (%s)", RedactedProxyURL(proxyURL)))
	}
	if !useEnv {
		reasons = append(reasons, "client.proxy_from_environment=false disables ambient proxy reuse")
	}
	if timeout != defaultClientSettingsTimeout {
		switch {
		case timeout == 0:
			reasons = append(reasons, fmt.Sprintf("effective timeout is %s instead of the default %s", timeout, defaultClientSettingsTimeout))
		case cs.TimeoutSeconds != nil:
			reasons = append(reasons, fmt.Sprintf("client.timeout_second=%d overrides the default %s timeout", *cs.TimeoutSeconds, defaultClientSettingsTimeout))
		default:
			reasons = append(reasons, fmt.Sprintf("client.timeout overrides the default %s timeout", defaultClientSettingsTimeout))
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "default timeout + environment proxy behavior + no explicit proxy URL; reusing http.DefaultClient")
	}

	mode := "custom-client"
	if proxyURL == "" && useEnv && timeout == defaultClientSettingsTimeout {
		mode = "default-client"
	}

	return HTTPClientDecision{
		Mode:                 mode,
		Reasons:              reasons,
		EffectiveTimeout:     timeout.String(),
		ProxyURL:             RedactedProxyURL(proxyURL),
		ProxyFromEnvironment: useEnv,
		ProxyMode:            proxyModeForSettings(cs),
	}
}

// EnsureHTTPClient returns the effective HTTP client for the given client settings.
//
// Behavior:
//   - reuse cs.HTTPClient when explicitly injected,
//   - preserve current default-client behavior when no proxy override is requested
//     and timeout remains at the default settings value,
//   - otherwise build and cache a transport-aware client derived from http.DefaultTransport.
func EnsureHTTPClient(cs *ClientSettings) (*http.Client, error) {
	if cs == nil {
		return http.DefaultClient, nil
	}
	if cs.HTTPClient != nil {
		return cs.HTTPClient, nil
	}

	timeout := effectiveTimeout(cs)
	proxyURL := ""
	if cs.ProxyURL != nil {
		proxyURL = strings.TrimSpace(*cs.ProxyURL)
	}
	useEnv := true
	if cs.ProxyFromEnvironment != nil {
		useEnv = *cs.ProxyFromEnvironment
	}

	if proxyURL == "" && useEnv && timeout == defaultClientSettingsTimeout {
		return http.DefaultClient, nil
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.Errorf("default transport is %T, expected *http.Transport", http.DefaultTransport)
	}
	transport := baseTransport.Clone()

	switch {
	case proxyURL != "":
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, errors.Wrap(err, "parse proxy URL")
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, errors.Errorf("proxy URL must include scheme and host: %q", proxyURL)
		}
		transport.Proxy = http.ProxyURL(u)
	case useEnv:
		transport.Proxy = http.ProxyFromEnvironment
	default:
		transport.Proxy = nil
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
	cs.HTTPClient = client
	return client, nil
}

func effectiveTimeout(cs *ClientSettings) time.Duration {
	if cs == nil {
		return 0
	}
	if cs.TimeoutSeconds != nil {
		return time.Duration(*cs.TimeoutSeconds) * time.Second
	}
	if cs.Timeout != nil {
		return *cs.Timeout
	}
	return 0
}

func effectiveTimeoutOrDefault(cs *ClientSettings) time.Duration {
	if timeout := effectiveTimeout(cs); timeout != 0 {
		return timeout
	}
	return defaultClientSettingsTimeout
}

func proxyFromEnvironmentOrDefault(cs *ClientSettings) bool {
	if cs == nil || cs.ProxyFromEnvironment == nil {
		return true
	}
	return *cs.ProxyFromEnvironment
}

func proxyModeForSettings(cs *ClientSettings) string {
	if cs == nil {
		return "environment"
	}
	if proxyURL := strings.TrimSpace(stringValue(cs.ProxyURL)); proxyURL != "" {
		return "explicit"
	}
	if proxyFromEnvironmentOrDefault(cs) {
		return "environment"
	}
	return "disabled"
}

func redactedClientProxyURL(cs *ClientSettings) string {
	if cs == nil {
		return ""
	}
	return RedactedProxyURL(strings.TrimSpace(stringValue(cs.ProxyURL)))
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func RedactedProxyURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Redacted()
}
