package turns

import (
	"encoding/json"

	pkgerrors "github.com/pkg/errors"
)

func decodeViaJSON[T any](raw any) (T, error) {
	var out T
	if raw == nil {
		return out, pkgerrors.Errorf("cannot decode <nil> into %T", out)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return out, pkgerrors.Wrapf(err, "json marshal %T", raw)
	}
	ptr := new(T)
	if err := json.Unmarshal(b, ptr); err != nil {
		return out, pkgerrors.Wrapf(err, "json unmarshal into %T", out)
	}
	return *ptr, nil
}

// DataKey is a typed key used to access Turn.Data.
//
// The underlying id string is a canonical key identity of the form:
// "namespace.value@vN".
type DataKey[T any] struct {
	id TurnDataKey
}

// TurnMetaKey is a typed key used to access Turn.Metadata.
//
// The underlying id string is a canonical key identity of the form:
// "namespace.value@vN".
type TurnMetaKey[T any] struct {
	id TurnMetadataKey
}

// BlockMetaKey is a typed key used to access Block.Metadata.
//
// The underlying id string is a canonical key identity of the form:
// "namespace.value@vN".
type BlockMetaKey[T any] struct {
	id BlockMetadataKey
}

// DataK constructs a typed key for Turn.Data.
func DataK[T any](namespace, value string, version uint16) DataKey[T] {
	return DataKey[T]{id: NewTurnDataKey(namespace, value, version)}
}

// TurnMetaK constructs a typed key for Turn.Metadata.
func TurnMetaK[T any](namespace, value string, version uint16) TurnMetaKey[T] {
	return TurnMetaKey[T]{id: NewTurnMetadataKey(namespace, value, version)}
}

// BlockMetaK constructs a typed key for Block.Metadata.
func BlockMetaK[T any](namespace, value string, version uint16) BlockMetaKey[T] {
	return BlockMetaKey[T]{id: NewBlockMetadataKey(namespace, value, version)}
}

// DataKeyFromID constructs a typed key for Turn.Data from an already-encoded id string.
//
// This is intended for advanced/internal use cases where the key id is already known
// (e.g. cloning/transforms over untyped values, YAML/JSON-derived keys).
func DataKeyFromID[T any](id TurnDataKey) DataKey[T] {
	return DataKey[T]{id: id}
}

// TurnMetaKeyFromID constructs a typed key for Turn.Metadata from an already-encoded id string.
//
// This is intended for advanced/internal use cases where the key id is already known
// (e.g. cloning/transforms over untyped values, YAML/JSON-derived keys).
func TurnMetaKeyFromID[T any](id TurnMetadataKey) TurnMetaKey[T] {
	return TurnMetaKey[T]{id: id}
}

// BlockMetaKeyFromID constructs a typed key for Block.Metadata from an already-encoded id string.
//
// This is intended for advanced/internal use cases where the key id is already known
// (e.g. cloning/transforms over untyped values, YAML/JSON-derived keys).
func BlockMetaKeyFromID[T any](id BlockMetadataKey) BlockMetaKey[T] {
	return BlockMetaKey[T]{id: id}
}

func (k DataKey[T]) String() string     { return k.id.String() }
func (k TurnMetaKey[T]) String() string { return k.id.String() }
func (k BlockMetaKey[T]) String() string {
	return k.id.String()
}

func (k DataKey[T]) Decode(raw any) (T, error) {
	typed, ok := raw.(T)
	if ok {
		return typed, nil
	}
	ret, err := decodeViaJSON[T](raw)
	if err != nil {
		var zero T
		return zero, pkgerrors.Wrapf(err, "Turn.Data[%q]: decode %T into %T", k.id.String(), raw, zero)
	}
	return ret, nil
}

func (k TurnMetaKey[T]) Decode(raw any) (T, error) {
	typed, ok := raw.(T)
	if ok {
		return typed, nil
	}
	ret, err := decodeViaJSON[T](raw)
	if err != nil {
		var zero T
		return zero, pkgerrors.Wrapf(err, "Turn.Metadata[%q]: decode %T into %T", k.id.String(), raw, zero)
	}
	return ret, nil
}

func (k BlockMetaKey[T]) Decode(raw any) (T, error) {
	typed, ok := raw.(T)
	if ok {
		return typed, nil
	}
	ret, err := decodeViaJSON[T](raw)
	if err != nil {
		var zero T
		return zero, pkgerrors.Wrapf(err, "Block.Metadata[%q]: decode %T into %T", k.id.String(), raw, zero)
	}
	return ret, nil
}

// Get returns (value, ok, error). Missing keys are (zero, false, nil).
// Type mismatches are (zero, true, error).
func (k DataKey[T]) Get(d Data) (T, bool, error) {
	var zero T
	if d.m == nil {
		return zero, false, nil
	}
	value, ok := d.m[k.id]
	if !ok {
		return zero, false, nil
	}
	typed, err := k.Decode(value)
	if err != nil {
		return zero, true, err
	}
	return typed, true, nil
}

// Set stores value for the key after validating JSON serializability.
func (k DataKey[T]) Set(d *Data, value T) error {
	if d.m == nil {
		d.m = make(map[TurnDataKey]any)
	}
	if _, err := json.Marshal(value); err != nil {
		return pkgerrors.Wrapf(err, "Turn.Data[%q]: value not serializable", k.id.String())
	}
	d.m[k.id] = value
	return nil
}

// Get returns (value, ok, error). Missing keys are (zero, false, nil).
// Type mismatches are (zero, true, error).
func (k TurnMetaKey[T]) Get(m Metadata) (T, bool, error) {
	var zero T
	if m.m == nil {
		return zero, false, nil
	}
	value, ok := m.m[k.id]
	if !ok {
		return zero, false, nil
	}
	typed, err := k.Decode(value)
	if err != nil {
		return zero, true, err
	}
	return typed, true, nil
}

// Set stores value for the key after validating JSON serializability.
func (k TurnMetaKey[T]) Set(m *Metadata, value T) error {
	if m.m == nil {
		m.m = make(map[TurnMetadataKey]any)
	}
	if _, err := json.Marshal(value); err != nil {
		return pkgerrors.Wrapf(err, "Turn.Metadata[%q]: value not serializable", k.id.String())
	}
	m.m[k.id] = value
	return nil
}

// Get returns (value, ok, error). Missing keys are (zero, false, nil).
// Type mismatches are (zero, true, error).
func (k BlockMetaKey[T]) Get(bm BlockMetadata) (T, bool, error) {
	var zero T
	if bm.m == nil {
		return zero, false, nil
	}
	value, ok := bm.m[k.id]
	if !ok {
		return zero, false, nil
	}
	typed, err := k.Decode(value)
	if err != nil {
		return zero, true, err
	}
	return typed, true, nil
}

// Set stores value for the key after validating JSON serializability.
func (k BlockMetaKey[T]) Set(bm *BlockMetadata, value T) error {
	if bm.m == nil {
		bm.m = make(map[BlockMetadataKey]any)
	}
	if _, err := json.Marshal(value); err != nil {
		return pkgerrors.Wrapf(err, "Block.Metadata[%q]: value not serializable", k.id.String())
	}
	bm.m[k.id] = value
	return nil
}
