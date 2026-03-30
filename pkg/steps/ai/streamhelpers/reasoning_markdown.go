package streamhelpers

import "strings"

// EnsureReasoningItemBoundary inserts a paragraph break between provider
// reasoning items when the accumulated stream does not already end at a
// newline boundary.
func EnsureReasoningItemBoundary(current string) string {
	if current == "" {
		return ""
	}
	trimmed := strings.TrimRight(current, " \t")
	if strings.HasSuffix(trimmed, "\n") {
		return ""
	}
	return "\n\n"
}

// NormalizeReasoningDelta inserts a paragraph break before a likely markdown
// block opener when the next delta would otherwise be glued directly onto the
// prior text.
func NormalizeReasoningDelta(current, delta string) string {
	if current == "" || delta == "" {
		return delta
	}
	trimmedCurrent := strings.TrimRight(current, " \t")
	if strings.HasSuffix(trimmedCurrent, "\n") {
		return delta
	}
	trimmedDelta := strings.TrimLeft(delta, " \t")
	if trimmedDelta == "" {
		return delta
	}
	if !startsLikelyMarkdownBlock(trimmedDelta) {
		return delta
	}
	return "\n\n" + delta
}

func startsLikelyMarkdownBlock(s string) bool {
	switch {
	case strings.HasPrefix(s, "#"),
		strings.HasPrefix(s, "- "),
		strings.HasPrefix(s, "* "),
		strings.HasPrefix(s, "> "),
		strings.HasPrefix(s, "```"):
		return true
	}

	if len(s) >= 3 && s[0] >= '0' && s[0] <= '9' && strings.HasPrefix(s[1:], ". ") {
		return true
	}

	if !strings.HasPrefix(s, "**") {
		return false
	}
	rest := s[2:]
	idx := strings.Index(rest, "**")
	if idx <= 0 {
		return false
	}
	after := rest[idx+2:]
	return strings.HasPrefix(after, "\n") || strings.HasPrefix(after, "\r\n")
}
