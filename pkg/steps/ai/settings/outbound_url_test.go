package settings

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOutboundURLOptionsDefaultDeny(t *testing.T) {
	opts := OutboundURLOptions(NewAPISettings(), "openai")
	if opts.AllowHTTP || opts.AllowLocalNetworks {
		t.Fatalf("default outbound options = %#v, want deny", opts)
	}
}

func TestOutboundURLOptionsUsesRawProviderKeys(t *testing.T) {
	api := NewAPISettings()
	api.AllowHTTP["openai"] = true
	api.AllowLocalNetworks["openai"] = true

	opts := OutboundURLOptions(api, "openai")
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("outbound options = %#v, want both true", opts)
	}
}

func TestOutboundURLOptionsUsesSuffixedProviderKeys(t *testing.T) {
	api := NewAPISettings()
	api.AllowHTTP["openai-allow-http"] = true
	api.AllowLocalNetworks["openai-allow-local-networks"] = true

	opts := OutboundURLOptions(api, "openai")
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("outbound options = %#v, want both true", opts)
	}
}

func TestInferenceSettingsUnmarshalOutboundURLOptions(t *testing.T) {
	ss, err := NewInferenceSettings()
	if err != nil {
		t.Fatalf("NewInferenceSettings: %v", err)
	}

	input := `api:
  allow_http:
    openai: true
  allow_local_networks:
    openai: true
chat:
  api_type: openai
  engine: gpt-4o-mini
`
	if err := yaml.NewDecoder(strings.NewReader(input)).Decode(ss); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	opts := OutboundURLOptions(ss.API, "openai")
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("outbound options = %#v, want both true", opts)
	}
}

func TestOutboundURLOptionsForKeysFallsBackAcrossAliases(t *testing.T) {
	api := NewAPISettings()
	api.AllowHTTP["openai"] = true
	api.AllowLocalNetworks["openai"] = true

	opts := OutboundURLOptionsForKeys(api, "open-responses", "openai-responses", "openai")
	if !opts.AllowHTTP || !opts.AllowLocalNetworks {
		t.Fatalf("outbound options = %#v, want fallback to openai", opts)
	}
}
