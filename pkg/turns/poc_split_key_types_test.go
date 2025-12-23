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
// This file intentionally only proves the language/API shape is viable against
// the current implementation (which today uses Key[T] + DataGet/DataSet functions).

type DataKey[T any] struct{ id TurnDataKey }
type TurnMetaKey[T any] struct{ id TurnMetadataKey }
type BlockMetaKey[T any] struct{ id BlockMetadataKey }

func DataK[T any](namespace, value string, version uint16) DataKey[T] {
	return DataKey[T]{id: NewTurnDataKey(namespace, value, version)}
}
func TurnMetaK[T any](namespace, value string, version uint16) TurnMetaKey[T] {
	return TurnMetaKey[T]{id: TurnMetadataKey(NewTurnDataKey(namespace, value, version))}
}
func BlockMetaK[T any](namespace, value string, version uint16) BlockMetaKey[T] {
	return BlockMetaKey[T]{id: BlockMetadataKey(NewTurnDataKey(namespace, value, version))}
}

func (k DataKey[T]) String() string     { return k.id.String() }
func (k TurnMetaKey[T]) String() string { return k.id.String() }
func (k BlockMetaKey[T]) String() string {
	return k.id.String()
}

func (k DataKey[T]) Get(d Data) (T, bool, error) {
	return DataGet(d, Key[T](k))
}
func (k DataKey[T]) Set(d *Data, v T) error {
	return DataSet(d, Key[T](k), v)
}

func (k TurnMetaKey[T]) Get(m Metadata) (T, bool, error) {
	return MetadataGet(m, Key[T]{id: TurnDataKey(k.id)})
}
func (k TurnMetaKey[T]) Set(m *Metadata, v T) error {
	return MetadataSet(m, Key[T]{id: TurnDataKey(k.id)}, v)
}

func (k BlockMetaKey[T]) Get(bm BlockMetadata) (T, bool, error) {
	return BlockMetadataGet(bm, Key[T]{id: TurnDataKey(k.id)})
}
func (k BlockMetaKey[T]) Set(bm *BlockMetadata, v T) error {
	return BlockMetadataSet(bm, Key[T]{id: TurnDataKey(k.id)}, v)
}

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
