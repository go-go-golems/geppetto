package settings

import (
	"maps"
	"strings"

	"github.com/pkg/errors"
)

// CredentialSlots is a configuration layer, before it becomes a connection.
// Its maps use the APISettings suffix convention; never pass it to request code.
type CredentialSlots struct {
	APIKeys  map[string]string
	BaseURLs map[string]string
}

// OverlayCredentialSlots replaces credential pairs as a unit. Blank overlay
// values do not erase bindings. A first declaration may use its provider's
// documented endpoint default; changing an existing authenticated binding needs
// both fields. Anonymous endpoint overrides remain possible.
func OverlayCredentialSlots(base, overlay CredentialSlots) (CredentialSlots, error) {
	result := CredentialSlots{APIKeys: maps.Clone(base.APIKeys), BaseURLs: maps.Clone(base.BaseURLs)}
	if result.APIKeys == nil {
		result.APIKeys = map[string]string{}
	}
	if result.BaseURLs == nil {
		result.BaseURLs = map[string]string{}
	}
	providers := map[string]struct{}{}
	for name, value := range overlay.APIKeys {
		if provider, ok := strings.CutSuffix(name, "-api-key"); ok && strings.TrimSpace(value) != "" {
			providers[provider] = struct{}{}
		}
	}
	for name, value := range overlay.BaseURLs {
		if provider, ok := strings.CutSuffix(name, "-base-url"); ok && strings.TrimSpace(value) != "" {
			providers[provider] = struct{}{}
		}
	}
	for provider := range providers {
		keyName, urlName := provider+"-api-key", provider+"-base-url"
		key, endpoint := strings.TrimSpace(overlay.APIKeys[keyName]), strings.TrimSpace(overlay.BaseURLs[urlName])
		priorKey, priorURL := strings.TrimSpace(base.APIKeys[keyName]), strings.TrimSpace(base.BaseURLs[urlName])
		existing := priorKey != "" || priorURL != ""
		authenticated := priorKey != "" || key != ""
		if existing && authenticated && (key == "" || endpoint == "") {
			return CredentialSlots{}, errors.Errorf("credential override for %s requires both %s and %s", provider, keyName, urlName)
		}
		delete(result.APIKeys, keyName)
		delete(result.BaseURLs, urlName)
		if key != "" {
			result.APIKeys[keyName] = key
		}
		if endpoint != "" {
			result.BaseURLs[urlName] = endpoint
		}
	}
	return result, nil
}
