package geppetto

import (
	"fmt"
	"strings"

	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
)

func (m *moduleRuntime) requireEngineProfileRegistryReader(method string) (profiles.RegistryReader, error) {
	if m.profileRegistry == nil {
		return nil, fmt.Errorf("%s requires a configured profile registry", method)
	}
	return m.profileRegistry, nil
}

func parseOptionalRegistrySlug(raw any) (profiles.RegistrySlug, error) {
	rawSlug := strings.TrimSpace(toString(raw, ""))
	if rawSlug == "" {
		return "", nil
	}
	return profiles.ParseRegistrySlug(rawSlug)
}

func decodeEngineProfileRegistrySources(raw any) ([]string, error) {
	if raw == nil {
		return nil, fmt.Errorf("profile registry sources are required")
	}
	switch v := raw.(type) {
	case string:
		return profiles.ParseEngineProfileRegistrySourceEntries(v)
	case []string:
		ret := make([]string, 0, len(v))
		for i, entry := range v {
			s := strings.TrimSpace(entry)
			if s == "" {
				return nil, fmt.Errorf("profile registry source entry %d is empty", i)
			}
			ret = append(ret, s)
		}
		return ret, nil
	case []any:
		ret := make([]string, 0, len(v))
		for i, rawEntry := range v {
			s := strings.TrimSpace(toString(rawEntry, ""))
			if s == "" {
				return nil, fmt.Errorf("profile registry source entry %d must be a non-empty string", i)
			}
			ret = append(ret, s)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("profile registry sources must be a comma-separated string or string array")
	}
}
