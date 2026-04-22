package streamhelpers

import "testing"

func TestEnsureReasoningItemBoundary(t *testing.T) {
	t.Run("empty current text", func(t *testing.T) {
		if got := EnsureReasoningItemBoundary(""); got != "" {
			t.Fatalf("expected no separator for empty current text, got %q", got)
		}
	})

	t.Run("newline already present", func(t *testing.T) {
		if got := EnsureReasoningItemBoundary("First item.\n"); got != "" {
			t.Fatalf("expected no separator when newline already present, got %q", got)
		}
	})

	t.Run("paragraph break needed", func(t *testing.T) {
		if got := EnsureReasoningItemBoundary("First item."); got != "\n\n" {
			t.Fatalf("expected paragraph separator, got %q", got)
		}
	})
}

func TestNormalizeReasoningDelta(t *testing.T) {
	t.Run("plain continuation stays unchanged", func(t *testing.T) {
		got := NormalizeReasoningDelta("Some thought.", " More detail.")
		if got != " More detail." {
			t.Fatalf("expected unchanged continuation, got %q", got)
		}
	})

	t.Run("bold heading gets paragraph break", func(t *testing.T) {
		got := NormalizeReasoningDelta("Some thought.", "**Crafting a response**\n\nMore text.")
		if got != "\n\n**Crafting a response**\n\nMore text." {
			t.Fatalf("expected paragraph break before bold heading, got %q", got)
		}
	})

	t.Run("list item gets paragraph break", func(t *testing.T) {
		got := NormalizeReasoningDelta("Some thought.", "- first item")
		if got != "\n\n- first item" {
			t.Fatalf("expected paragraph break before list item, got %q", got)
		}
	})

	t.Run("existing newline avoids duplication", func(t *testing.T) {
		got := NormalizeReasoningDelta("Some thought.\n", "**Crafting a response**\n\nMore text.")
		if got != "**Crafting a response**\n\nMore text." {
			t.Fatalf("expected no extra paragraph break, got %q", got)
		}
	})
}

func TestNormalizeReasoningSummaryDelta(t *testing.T) {
	t.Run("sentence boundary gets space", func(t *testing.T) {
		got := NormalizeReasoningSummaryDelta("A prior category.", "Creating an analysis plan")
		if got != " Creating an analysis plan" {
			t.Fatalf("expected sentence boundary space, got %q", got)
		}
	})

	t.Run("existing leading whitespace is preserved", func(t *testing.T) {
		got := NormalizeReasoningSummaryDelta("A prior category.", " Creating an analysis plan")
		if got != " Creating an analysis plan" {
			t.Fatalf("expected unchanged delta when leading space already present, got %q", got)
		}
	})

	t.Run("markdown heading still gets paragraph break", func(t *testing.T) {
		got := NormalizeReasoningSummaryDelta("Some thought.", "**Crafting a response**\n\nMore text.")
		if got != "\n\n**Crafting a response**\n\nMore text." {
			t.Fatalf("expected paragraph break before bold heading, got %q", got)
		}
	})

	t.Run("lowercase continuation stays unchanged", func(t *testing.T) {
		got := NormalizeReasoningSummaryDelta("Some thought.", "more detail")
		if got != "more detail" {
			t.Fatalf("expected unchanged lowercase continuation, got %q", got)
		}
	})
}
