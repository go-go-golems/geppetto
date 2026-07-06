package settings

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/security"
)

// OutboundURLOptions returns provider URL validation options for a single API type.
//
// Defaults remain fail-closed: plain HTTP and local-network targets are rejected
// unless the profile explicitly opts in under inference_settings.api.
func OutboundURLOptions(api *APISettings, apiType string) security.OutboundURLOptions {
	return OutboundURLOptionsForKeys(api, apiType)
}

// OutboundURLOptionsForKeys returns provider URL validation options using the
// first matching provider key. It accepts several aliases because some provider
// families (notably OpenAI Responses) support legacy and canonical API type
// names while sharing one endpoint.
//
// The YAML shape is intentionally profile-owned and explicit:
//
//	api:
//	  allow_http:
//	    openai: true
//	  allow_local_networks:
//	    openai: true
//
// Suffix-style keys such as "openai-allow-http" are also accepted so CLI or
// generated-map callers can use the same naming convention as api_keys and
// base_urls.
func OutboundURLOptionsForKeys(api *APISettings, apiTypes ...string) security.OutboundURLOptions {
	if api == nil {
		return security.OutboundURLOptions{}
	}

	return security.OutboundURLOptions{
		AllowHTTP:          lookupOutboundBool(api.AllowHTTP, "allow-http", apiTypes...),
		AllowLocalNetworks: lookupOutboundBool(api.AllowLocalNetworks, "allow-local-networks", apiTypes...),
	}
}

func lookupOutboundBool(values map[string]bool, suffix string, apiTypes ...string) bool {
	if len(values) == 0 {
		return false
	}

	for _, apiType := range apiTypes {
		apiType = strings.TrimSpace(apiType)
		if apiType == "" {
			continue
		}
		for _, key := range []string{apiType, apiType + "-" + suffix} {
			if value, ok := values[key]; ok {
				return value
			}
		}
	}

	return false
}
