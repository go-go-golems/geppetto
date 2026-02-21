package session

import (
	"context"
	"testing"
)

func TestWithSessionMetaRoundTrip(t *testing.T) {
	ctx := WithSessionMeta(context.Background(), "s-1", "i-1")
	if got := SessionIDFromContext(ctx); got != "s-1" {
		t.Fatalf("SessionIDFromContext() = %q, want %q", got, "s-1")
	}
	if got := InferenceIDFromContext(ctx); got != "i-1" {
		t.Fatalf("InferenceIDFromContext() = %q, want %q", got, "i-1")
	}
}

func TestWithSessionMetaHandlesTodoContext(t *testing.T) {
	ctx := WithSessionMeta(context.TODO(), "s-2", "i-2")
	if got := SessionIDFromContext(ctx); got != "s-2" {
		t.Fatalf("SessionIDFromContext() = %q, want %q", got, "s-2")
	}
	if got := InferenceIDFromContext(ctx); got != "i-2" {
		t.Fatalf("InferenceIDFromContext() = %q, want %q", got, "i-2")
	}
}

func TestSessionMetaAccessorsMissing(t *testing.T) {
	if got := SessionIDFromContext(context.Background()); got != "" {
		t.Fatalf("SessionIDFromContext() = %q, want empty", got)
	}
	if got := InferenceIDFromContext(context.Background()); got != "" {
		t.Fatalf("InferenceIDFromContext() = %q, want empty", got)
	}
}

func TestWithRunTagsRoundTrip(t *testing.T) {
	ctx := WithRunTags(context.Background(), map[string]any{"request_id": "r-1", "attempt": 2})
	tags := RunTagsFromContext(ctx)
	if tags["request_id"] != "r-1" {
		t.Fatalf("RunTagsFromContext()[request_id] = %v, want r-1", tags["request_id"])
	}
	if tags["attempt"] != 2 {
		t.Fatalf("RunTagsFromContext()[attempt] = %v, want 2", tags["attempt"])
	}

	// Ensure caller mutation does not alter stored context tags.
	tags["request_id"] = "changed"
	tags2 := RunTagsFromContext(ctx)
	if tags2["request_id"] != "r-1" {
		t.Fatalf("RunTagsFromContext() should return cloned tags, got %v", tags2["request_id"])
	}
}
