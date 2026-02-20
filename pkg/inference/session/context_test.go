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
