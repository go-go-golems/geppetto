package settings

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const defaultClientSettingsTimeout = 60 * time.Second

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
