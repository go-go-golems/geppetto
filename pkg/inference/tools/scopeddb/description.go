package scopeddb

import (
	"fmt"
	"strings"
)

func BuildDescription(desc ToolDescription, allowedObjects []string, opts QueryOptions) string {
	parts := make([]string, 0, 4+len(desc.Notes))

	summary := strings.TrimSpace(desc.Summary)
	if summary == "" {
		summary = "Query a scoped read-only SQLite database."
	}
	parts = append(parts, ensureSentence(summary))

	allowed := NormalizeNonEmptyStrings(allowedObjects)
	if len(allowed) > 0 {
		parts = append(parts, "Allowed tables/views: "+strings.Join(allowed, ", ")+".")
	}

	for _, note := range desc.Notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		parts = append(parts, ensureSentence(note))
	}

	if opts.RequireOrderBy && !containsOrderByHint(parts) {
		parts = append(parts, "Use ORDER BY for deterministic row ordering.")
	}

	if len(desc.StarterQueries) > 0 {
		queries := NormalizeNonEmptyStrings(desc.StarterQueries)
		if len(queries) > 0 {
			parts = append(parts, fmt.Sprintf("Starter queries: %s.", strings.Join(queries, " | ")))
		}
	}

	return strings.Join(parts, " ")
}

func ensureSentence(v string) string {
	if v == "" {
		return v
	}
	if strings.HasSuffix(v, ".") || strings.HasSuffix(v, "!") || strings.HasSuffix(v, "?") {
		return v
	}
	return v + "."
}

func containsOrderByHint(parts []string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(part), "order by") {
			return true
		}
	}
	return false
}
