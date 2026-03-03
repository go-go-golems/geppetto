package runtimeattrib

import (
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// AttachToExtra copies runtime/profile attribution from Turn metadata into an EventMetadata.Extra map.
//
// Expected Turn.Metadata source:
// - turns.KeyTurnMetaRuntime may be either:
//   - string (runtime key), or
//   - map[string]any (recommended) with keys like:
//   - runtime_key
//   - runtime_fingerprint
//   - profile_slug
//   - registry_slug
//   - profile_version
//
// Target keys written into extra (when available):
// - runtime_key
// - runtime_fingerprint
// - profile.slug
// - profile.registry
// - profile.version
func AttachToExtra(extra map[string]any, t *turns.Turn) {
	if extra == nil || t == nil {
		return
	}

	v, ok, err := turns.KeyTurnMetaRuntime.Get(t.Metadata)
	if err != nil || !ok || v == nil {
		return
	}

	switch rt := v.(type) {
	case string:
		if s := strings.TrimSpace(rt); s != "" {
			extra["runtime_key"] = s
		}
	case map[string]any:
		if s := trimString(rt["runtime_key"]); s != "" {
			extra["runtime_key"] = s
		} else if s := trimString(rt["key"]); s != "" {
			extra["runtime_key"] = s
		} else if s := trimString(rt["slug"]); s != "" {
			extra["runtime_key"] = s
		}
		if s := trimString(rt["runtime_fingerprint"]); s != "" {
			extra["runtime_fingerprint"] = s
		}

		// Normalize profile/registry naming into dotted keys used by persistence.
		if s := trimString(rt["profile.slug"]); s != "" {
			extra["profile.slug"] = s
		} else if s := trimString(rt["profile_slug"]); s != "" {
			extra["profile.slug"] = s
		}
		if s := trimString(rt["profile.registry"]); s != "" {
			extra["profile.registry"] = s
		} else if s := trimString(rt["registry_slug"]); s != "" {
			extra["profile.registry"] = s
		}
		if n, ok := rt["profile.version"].(uint64); ok && n > 0 {
			extra["profile.version"] = n
		} else if n, ok := rt["profile_version"].(uint64); ok && n > 0 {
			extra["profile.version"] = n
		} else if n, ok := rt["profile_version"].(int64); ok && n > 0 {
			extra["profile.version"] = uint64(n)
		} else if n, ok := rt["profile_version"].(int); ok && n > 0 {
			extra["profile.version"] = uint64(n)
		}
	}
}

func trimString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
