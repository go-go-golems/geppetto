package turns

import (
	"strings"
	"testing"
)

// Proof-of-concept: split typed keys into 3 families with Get/Set methods:
// - DataKey[T] for Turn.Data
// - TurnMetaKey[T] for Turn.Metadata
// - BlockMetaKey[T] for Block.Metadata
//
// This file originally defined local types to validate the intended API shape.
// Those types now exist in production code; these tests ensure the behavior
// contracts remain correct.

func TestPOC_SplitKeyTypes_Data(t *testing.T) {
	key := DataK[string]("testns", "foo", 1)

	var d Data
	v, ok, err := key.Get(d)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for missing key")
	}
	if v != "" {
		t.Fatalf("expected zero value, got %q", v)
	}

	if err := key.Set(&d, "bar"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	v, ok, err = key.Get(d)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true after set")
	}
	if v != "bar" {
		t.Fatalf("expected %q, got %q", "bar", v)
	}
}

func TestPOC_SplitKeyTypes_TurnMetadata(t *testing.T) {
	key := TurnMetaK[string]("testns", "meta", 1)

	var m Metadata
	if err := key.Set(&m, "baz"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	v, ok, err := key.Get(m)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true after set")
	}
	if v != "baz" {
		t.Fatalf("expected %q, got %q", "baz", v)
	}
}

func TestPOC_SplitKeyTypes_BlockMetadata(t *testing.T) {
	key := BlockMetaK[string]("testns", "block", 1)

	var bm BlockMetadata
	if err := key.Set(&bm, "qux"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	v, ok, err := key.Get(bm)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true after set")
	}
	if v != "qux" {
		t.Fatalf("expected %q, got %q", "qux", v)
	}
}

func TestPOC_SplitKeyTypes_TypeMismatchIsError_Data(t *testing.T) {
	keyString := DataK[string]("testns", "mismatch", 1)

	var d Data
	d.m = map[TurnDataKey]any{
		NewTurnDataKey("testns", "mismatch", 1): 123,
	}

	_, ok, err := keyString.Get(d)
	if !ok {
		t.Fatalf("expected ok=true when key exists (even if wrong type)")
	}
	if err == nil {
		t.Fatalf("expected error on type mismatch")
	}
}

func TestPOC_SplitKeyTypes_SerializabilityValidation(t *testing.T) {
	key := DataK[func()]("testns", "bad", 1)

	var d Data
	err := key.Set(&d, func() {})
	if err == nil {
		t.Fatalf("expected error for non-serializable value")
	}
	if !strings.Contains(err.Error(), "value not serializable") {
		t.Fatalf("expected error to mention serializability, got: %v", err)
	}
	if got := d.Len(); got != 0 {
		t.Fatalf("expected no stored values after failed set, got Len=%d", got)
	}
}
