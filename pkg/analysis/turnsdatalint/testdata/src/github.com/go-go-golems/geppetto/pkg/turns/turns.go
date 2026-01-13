package turns

// NOTE: This is a minimal stub package for analysistest.
// It exists so the analyzer tests are stable and do not depend on the real module API.

// RunMetadataKey is a typed string key for Run.Metadata map.
type RunMetadataKey string

const (
	RunMetaKeyTraceID RunMetadataKey = "trace_id"
)

type Run struct {
	Metadata map[RunMetadataKey]any
}

type Block struct {
	Payload map[string]any
}

const (
	PayloadKeyText = "text"
)

type TurnDataKey string
type TurnMetadataKey string
type BlockMetadataKey string

// Key families + constructors (minimal signatures only).
type DataKey[T any] struct{}
type TurnMetaKey[T any] struct{}
type BlockMetaKey[T any] struct{}

func DataK[T any](namespace, value string, version uint16) DataKey[T] { return DataKey[T]{} }
func TurnMetaK[T any](namespace, value string, version uint16) TurnMetaKey[T] {
	return TurnMetaKey[T]{}
}
func BlockMetaK[T any](namespace, value string, version uint16) BlockMetaKey[T] {
	return BlockMetaKey[T]{}
}
