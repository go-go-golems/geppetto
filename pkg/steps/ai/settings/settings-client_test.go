package settings

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewClientSettings_DefaultsProxyFromEnvironment(t *testing.T) {
	s := NewClientSettings()
	if s.ProxyFromEnvironment == nil {
		t.Fatalf("expected ProxyFromEnvironment default")
	}
	if !*s.ProxyFromEnvironment {
		t.Fatalf("expected ProxyFromEnvironment default true")
	}
}

func TestClientSettingsUnmarshalYAMLParsesProxyFields(t *testing.T) {
	var s ClientSettings
	err := yaml.Unmarshal([]byte(`
timeout: 15
proxy_url: http://proxy.internal:8080
proxy_from_environment: false
`), &s)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if s.TimeoutSeconds == nil || *s.TimeoutSeconds != 15 {
		t.Fatalf("expected TimeoutSeconds=15, got %#v", s.TimeoutSeconds)
	}
	if s.ProxyURL == nil || *s.ProxyURL != "http://proxy.internal:8080" {
		t.Fatalf("expected ProxyURL to be parsed, got %#v", s.ProxyURL)
	}
	if s.ProxyFromEnvironment == nil {
		t.Fatalf("expected ProxyFromEnvironment to be parsed")
	}
	if *s.ProxyFromEnvironment {
		t.Fatalf("expected ProxyFromEnvironment=false")
	}
}

func TestClientValueSection_ExposesProxyFlags(t *testing.T) {
	section, err := NewClientValueSection()
	if err != nil {
		t.Fatalf("NewClientValueSection: %v", err)
	}

	defs := section.GetDefinitions()
	proxyURLDef, ok := defs.Get("proxy-url")
	if !ok {
		t.Fatalf("expected proxy-url definition")
	}
	if !strings.Contains(proxyURLDef.Help, "proxy URL") {
		t.Fatalf("expected proxy-url help to mention proxy URL, got %q", proxyURLDef.Help)
	}

	proxyEnvDef, ok := defs.Get("proxy-from-environment")
	if !ok {
		t.Fatalf("expected proxy-from-environment definition")
	}
	if proxyEnvDef.Default == nil {
		t.Fatalf("expected proxy-from-environment default=true, got nil")
	}
	defaultValue, ok := (*proxyEnvDef.Default).(bool)
	if !ok || !defaultValue {
		t.Fatalf("expected proxy-from-environment default=true, got %#v", proxyEnvDef.Default)
	}
}
