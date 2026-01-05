package turns

import (
	"encoding/json"

	pkgerrors "github.com/pkg/errors"
)

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
	return TurnMetaKey[T]{id: TurnMetadataKey(NewTurnDataKey(namespace, value, version))}
}

// BlockMetaK constructs a typed key for Block.Metadata.
func BlockMetaK[T any](namespace, value string, version uint16) BlockMetaKey[T] {
	return BlockMetaKey[T]{id: BlockMetadataKey(NewTurnDataKey(namespace, value, version))}
}

func (k DataKey[T]) String() string     { return k.id.String() }
func (k TurnMetaKey[T]) String() string { return k.id.String() }
func (k BlockMetaKey[T]) String() string {
	return k.id.String()
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
	typed, ok := value.(T)
	if !ok {
		return zero, true, pkgerrors.Errorf("Turn.Data[%q]: expected %T, got %T", k.id.String(), zero, value)
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
	typed, ok := value.(T)
	if !ok {
		return zero, true, pkgerrors.Errorf("Turn.Metadata[%q]: expected %T, got %T", k.id.String(), zero, value)
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
	typed, ok := value.(T)
	if !ok {
		return zero, true, pkgerrors.Errorf("Block.Metadata[%q]: expected %T, got %T", k.id.String(), zero, value)
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
