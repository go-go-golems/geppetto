package streamhelpers

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

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

// NormalizeReasoningSummaryDelta preserves readable boundaries in public
// reasoning summaries. It reuses markdown-block detection and also inserts a
// plain space when one streamed chunk ends at sentence punctuation and the next
// chunk starts with an uppercase continuation without any leading whitespace.
func NormalizeReasoningSummaryDelta(current, delta string) string {
	if current == "" || delta == "" {
		return delta
	}
	if normalized := NormalizeReasoningDelta(current, delta); normalized != delta {
		return normalized
	}
	trimmedCurrent := strings.TrimRight(current, " \t")
	if trimmedCurrent == "" || strings.HasSuffix(trimmedCurrent, "\n") {
		return delta
	}
	trimmedDelta := strings.TrimLeft(delta, " \t")
	if trimmedDelta == "" || trimmedDelta != delta {
		return delta
	}
	if !shouldInsertSentenceBoundarySpace(trimmedCurrent, trimmedDelta) {
		return delta
	}
	return " " + delta
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

func shouldInsertSentenceBoundarySpace(current, delta string) bool {
	last, ok := lastRune(current)
	if !ok || !isSentenceBoundaryRune(last) {
		return false
	}
	first, ok := firstRune(delta)
	if !ok {
		return false
	}
	return unicode.IsUpper(first)
}

func isSentenceBoundaryRune(r rune) bool {
	switch r {
	case '.', '!', '?', ':':
		return true
	default:
		return false
	}
}

func firstRune(s string) (rune, bool) {
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && s == "" {
		return 0, false
	}
	return r, true
}

func lastRune(s string) (rune, bool) {
	r, _ := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && s == "" {
		return 0, false
	}
	return r, true
}
