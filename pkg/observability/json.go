package observability

import (
	"encoding/json"
	"fmt"
	"strings"
)

var sensitiveKeys = map[string]struct{}{
	"api_key":           {},
	"apikey":            {},
	"api-key":           {},
	"authorization":     {},
	"access_token":      {},
	"refresh_token":     {},
	"bearer":            {},
	"token":             {},
	"encrypted_content": {},
}

// MarshalEvidenceJSON returns valid JSON for observability payload fields after
// recursively cloning, redacting, and capping string-heavy values. It never
// returns raw stream text; callers pass decoded provider objects, Geppetto
// events, or metadata values.
func MarshalEvidenceJSON(v any, cfg Config) json.RawMessage {
	cfg = cfg.Normalized()
	clean := sanitizeValue(v, cfg)
	b, err := json.Marshal(clean)
	if err != nil {
		b, _ = json.Marshal(map[string]any{
			"_marshal_error": err.Error(),
		})
		return b
	}
	return b
}

func sanitizeValue(v any, cfg Config) any {
	switch tv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(tv))
		for k, val := range tv {
			if shouldRedactKey(k, cfg) {
				out[k] = redactedMarker(val)
				continue
			}
			out[k] = sanitizeValue(val, cfg)
		}
		return out
	case []any:
		out := make([]any, len(tv))
		for i, val := range tv {
			out[i] = sanitizeValue(val, cfg)
		}
		return out
	case string:
		return capString(tv, cfg.MaxPayloadBytes)
	default:
		return v
	}
}

func shouldRedactKey(k string, cfg Config) bool {
	if !cfg.RedactProviderData {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(k), "-", "_"))
	_, ok := sensitiveKeys[normalized]
	return ok
}

func redactedMarker(v any) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("<redacted:%d bytes>", len(s))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "<redacted>"
	}
	return fmt.Sprintf("<redacted:%d bytes>", len(b))
}

func capString(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return fmt.Sprintf("%s<truncated:%d bytes>", s[:limit], len(s)-limit)
}
