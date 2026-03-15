package scopeddb

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func NormalizeNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	ret := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		ret = append(ret, trimmed)
	}
	return ret
}

func ShortID(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])[:16]
}

func AllowedObjectList(allowedObjects map[string]struct{}) []string {
	out := make([]string, 0, len(allowedObjects))
	for k := range allowedObjects {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func NormalizeObjectName(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func AllowedObjectMap(allowedObjects []string) map[string]struct{} {
	ret := make(map[string]struct{}, len(allowedObjects))
	for _, value := range allowedObjects {
		normalized := NormalizeObjectName(value)
		if normalized == "" {
			continue
		}
		ret[normalized] = struct{}{}
	}
	return ret
}

// TrimOptionalTrailingSemicolon allows a single harmless statement terminator
// while still letting validators reject any remaining semicolons as multi-statement SQL.
func TrimOptionalTrailingSemicolon(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	if strings.HasSuffix(trimmed, ";") {
		return strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return trimmed
}
