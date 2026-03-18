package runtimeattrib

import (
	"math"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// AddRuntimeAttributionToExtra copies runtime/profile attribution from Turn metadata
// into an EventMetadata.Extra map.
//
// Expected Turn.Metadata source:
// - turns.KeyTurnMetaRuntime as map[string]any with canonical keys:
//   - runtime_key
//   - runtime_fingerprint
//   - profile.slug
//   - profile.registry
//   - profile.version
//
// Target keys written into extra (when available):
// - runtime_key
// - runtime_fingerprint
// - profile.slug
// - profile.registry
// - profile.version
func AddRuntimeAttributionToExtra(extra map[string]any, t *turns.Turn) {
	if extra == nil || t == nil {
		return
	}

	v, ok, err := turns.KeyTurnMetaRuntime.Get(t.Metadata)
	if err != nil || !ok || v == nil {
		return
	}

	rt, ok := v.(map[string]any)
	if !ok {
		return
	}
	if s := trimString(rt["runtime_key"]); s != "" {
		extra["runtime_key"] = s
	}
	if s := trimString(rt["runtime_fingerprint"]); s != "" {
		extra["runtime_fingerprint"] = s
	}
	if s := trimString(rt["profile.slug"]); s != "" {
		extra["profile.slug"] = s
	}
	if s := trimString(rt["profile.registry"]); s != "" {
		extra["profile.registry"] = s
	}
	if n, ok := positiveUint64(rt["profile.version"]); ok {
		extra["profile.version"] = n
	}
}

func trimString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func positiveUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		return n, n > 0
	case uint32:
		return uint64(n), n > 0
	case uint16:
		return uint64(n), n > 0
	case uint8:
		return uint64(n), n > 0
	case uint:
		return uint64(n), n > 0
	case int64:
		if n > 0 {
			return uint64(n), true
		}
	case int32:
		if n > 0 {
			return uint64(n), true
		}
	case int16:
		if n > 0 {
			return uint64(n), true
		}
	case int8:
		if n > 0 {
			return uint64(n), true
		}
	case int:
		if n > 0 {
			return uint64(n), true
		}
	case float64:
		if n > 0 && n == math.Trunc(n) && n <= float64(^uint64(0)) {
			return uint64(n), true
		}
	case float32:
		f := float64(n)
		if f > 0 && f == math.Trunc(f) && f <= float64(^uint64(0)) {
			return uint64(f), true
		}
	}
	return 0, false
}
