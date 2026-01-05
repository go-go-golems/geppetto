package turns

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

// BlockKind represents the kind of a block within a Turn.
type BlockKind int

const (
	BlockKindUser BlockKind = iota
	BlockKindLLMText
	BlockKindToolCall
	BlockKindToolUse
	BlockKindSystem
	// BlockKindReasoning represents provider reasoning items (e.g., OpenAI encrypted reasoning).
	BlockKindReasoning
	BlockKindOther
)

// String returns a human-readable identifier for the BlockKind.
func (k BlockKind) String() string {
	switch k {
	case BlockKindUser:
		return "user"
	case BlockKindLLMText:
		return "llm_text"
	case BlockKindToolCall:
		return "tool_call"
	case BlockKindToolUse:
		return "tool_use"
	case BlockKindSystem:
		return "system"
	case BlockKindReasoning:
		return "reasoning"
	case BlockKindOther:
		return "other"
	default:
		return "other"
	}
}

// YAML serialization for BlockKind using stable string names
func (k BlockKind) MarshalYAML() (interface{}, error) {
	return k.String(), nil
}

func (k *BlockKind) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*k = BlockKindOther
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "user":
		*k = BlockKindUser
	case "llm_text":
		*k = BlockKindLLMText
	case "tool_call":
		*k = BlockKindToolCall
	case "tool_use":
		*k = BlockKindToolUse
	case "system":
		*k = BlockKindSystem
	case "reasoning":
		*k = BlockKindReasoning
	case "other", "":
		*k = BlockKindOther
	default:
		// Unknown kind – map to Other but keep original for visibility
		*k = BlockKindOther
	}
	return nil
}

// Block represents a single atomic unit within a Turn.
type Block struct {
	ID      string         `yaml:"id,omitempty"`
	TurnID  string         `yaml:"turn_id,omitempty"`
	Kind    BlockKind      `yaml:"kind"`
	Role    string         `yaml:"role,omitempty"`
	Payload map[string]any `yaml:"payload,omitempty"`
	// Metadata stores arbitrary metadata about the block
	Metadata BlockMetadata `yaml:"metadata,omitempty"`
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
	ID     string  `yaml:"id,omitempty"`
	RunID  string  `yaml:"run_id,omitempty"`
	Blocks []Block `yaml:"blocks"`
	// Metadata stores arbitrary metadata about the turn
	Metadata Metadata `yaml:"metadata,omitempty"`
	// Data stores the application data payload associated with this turn
	Data Data `yaml:"data,omitempty"`
}

// Key is the legacy typed key previously used to access Turn.Data, Turn.Metadata, and Block.Metadata.
// The underlying ID is an encoded string key of form: "namespace.value@vN".
//
// Note: We intentionally keep this wrapper opaque: callers get a typed Key[T] and can only read/write via wrapper APIs.
type Key[T any] struct {
	id TurnDataKey
}

// NewTurnDataKey constructs a canonical key identity string in the form "namespace.value@vN".
//
// This panics on invalid input (per design doc): empty namespace/value or version < 1.
func NewTurnDataKey(namespace, value string, version uint16) TurnDataKey {
	if namespace == "" || value == "" || version < 1 {
		panic(fmt.Errorf("invalid key: namespace=%q value=%q version=%d", namespace, value, version))
	}
	return TurnDataKey(fmt.Sprintf("%s.%s@v%d", namespace, value, version))
}

// K creates a typed key from namespace/value consts and an explicit version.
func K[T any](namespace, value string, version uint16) Key[T] {
	return Key[T]{id: NewTurnDataKey(namespace, value, version)}
}

func (k Key[T]) String() string {
	return k.id.String()
}

// Data is an opaque wrapper for Turn.Data.
type Data struct {
	m map[TurnDataKey]any
}

// IsZero allows YAML omitempty to work with an opaque wrapper that has no exported fields.
// gopkg.in/yaml.v3 treats structs with no exported fields as empty unless IsZero is provided.
func (d Data) IsZero() bool {
	return len(d.m) == 0
}

// DataSet is the legacy function API for Turn.Data; prefer DataKey.Set.
func DataSet[T any](d *Data, key DataKey[T], value T) error {
	return key.Set(d, value)
}

// DataGet is the legacy function API for Turn.Data; prefer DataKey.Get.
func DataGet[T any](d Data, key DataKey[T]) (T, bool, error) {
	return key.Get(d)
}

func (d Data) Len() int {
	if d.m == nil {
		return 0
	}
	return len(d.m)
}

func (d Data) Range(fn func(TurnDataKey, any) bool) {
	if d.m == nil {
		return
	}
	for k, v := range d.m {
		if !fn(k, v) {
			return
		}
	}
}

func (d *Data) Delete(key TurnDataKey) {
	if d.m == nil {
		return
	}
	delete(d.m, key)
	if len(d.m) == 0 {
		d.m = nil
	}
}

func (d Data) MarshalYAML() (interface{}, error) {
	if len(d.m) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(d.m))
	for k, v := range d.m {
		out[k.String()] = v
	}
	return out, nil
}

func (d *Data) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		d.m = nil
		return nil
	}
	var raw map[string]any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		d.m = nil
		return nil
	}
	d.m = make(map[TurnDataKey]any, len(raw))
	for kStr, v := range raw {
		// Accept key strings as-is; canonical/format enforcement is handled by turnsdatalint.
		d.m[TurnDataKey(kStr)] = v
	}
	return nil
}

// Metadata is an opaque wrapper for Turn.Metadata.
type Metadata struct {
	m map[TurnMetadataKey]any
}

// IsZero allows YAML omitempty to work with an opaque wrapper that has no exported fields.
func (m Metadata) IsZero() bool {
	return len(m.m) == 0
}

// MetadataSet is the legacy function API for Turn.Metadata; prefer TurnMetaKey.Set.
func MetadataSet[T any](m *Metadata, key TurnMetaKey[T], value T) error {
	return key.Set(m, value)
}

// MetadataGet is the legacy function API for Turn.Metadata; prefer TurnMetaKey.Get.
func MetadataGet[T any](m Metadata, key TurnMetaKey[T]) (T, bool, error) {
	return key.Get(m)
}

func (m Metadata) Len() int {
	if m.m == nil {
		return 0
	}
	return len(m.m)
}

func (m Metadata) Range(fn func(TurnMetadataKey, any) bool) {
	if m.m == nil {
		return
	}
	for k, v := range m.m {
		if !fn(k, v) {
			return
		}
	}
}

func (m *Metadata) Delete(key TurnMetadataKey) {
	if m.m == nil {
		return
	}
	delete(m.m, key)
	if len(m.m) == 0 {
		m.m = nil
	}
}

func (m Metadata) MarshalYAML() (interface{}, error) {
	if len(m.m) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(m.m))
	for k, v := range m.m {
		out[k.String()] = v
	}
	return out, nil
}

func (m *Metadata) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		m.m = nil
		return nil
	}
	var raw map[string]any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		m.m = nil
		return nil
	}
	m.m = make(map[TurnMetadataKey]any, len(raw))
	for kStr, v := range raw {
		// Accept key strings as-is; canonical/format enforcement is handled by turnsdatalint.
		m.m[TurnMetadataKey(kStr)] = v
	}
	return nil
}

// BlockMetadata is an opaque wrapper for Block.Metadata.
type BlockMetadata struct {
	m map[BlockMetadataKey]any
}

// IsZero allows YAML omitempty to work with an opaque wrapper that has no exported fields.
func (bm BlockMetadata) IsZero() bool {
	return len(bm.m) == 0
}

// BlockMetadataSet is the legacy function API for Block.Metadata; prefer BlockMetaKey.Set.
func BlockMetadataSet[T any](bm *BlockMetadata, key BlockMetaKey[T], value T) error {
	return key.Set(bm, value)
}

// BlockMetadataGet is the legacy function API for Block.Metadata; prefer BlockMetaKey.Get.
func BlockMetadataGet[T any](bm BlockMetadata, key BlockMetaKey[T]) (T, bool, error) {
	return key.Get(bm)
}

func (bm BlockMetadata) Len() int {
	if bm.m == nil {
		return 0
	}
	return len(bm.m)
}

func (bm BlockMetadata) Range(fn func(BlockMetadataKey, any) bool) {
	if bm.m == nil {
		return
	}
	for k, v := range bm.m {
		if !fn(k, v) {
			return
		}
	}
}

func (bm *BlockMetadata) Delete(key BlockMetadataKey) {
	if bm.m == nil {
		return
	}
	delete(bm.m, key)
	if len(bm.m) == 0 {
		bm.m = nil
	}
}

func (bm BlockMetadata) MarshalYAML() (interface{}, error) {
	if len(bm.m) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(bm.m))
	for k, v := range bm.m {
		out[k.String()] = v
	}
	return out, nil
}

func (bm *BlockMetadata) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		bm.m = nil
		return nil
	}
	var raw map[string]any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		bm.m = nil
		return nil
	}
	bm.m = make(map[BlockMetadataKey]any, len(raw))
	for kStr, v := range raw {
		// Accept key strings as-is; canonical/format enforcement is handled by turnsdatalint.
		bm.m[BlockMetadataKey(kStr)] = v
	}
	return nil
}

// PrependBlock inserts a block at the beginning of the Turn's block slice.
func PrependBlock(t *Turn, b Block) {
	if t == nil {
		return
	}
	t.Blocks = append([]Block{b}, t.Blocks...)
}

// Run captures a multi-turn session.
type Run struct {
	ID    string
	Name  string
	Turns []Turn
	// Metadata stores arbitrary metadata about the run
	Metadata map[RunMetadataKey]interface{}
}

// AppendBlock appends a Block to a Turn, assigning an order if missing.
func AppendBlock(t *Turn, b Block) {
	t.Blocks = append(t.Blocks, b)
}

// AppendBlocks appends multiple Blocks ensuring increasing order.
func AppendBlocks(t *Turn, blocks ...Block) {
	for _, b := range blocks {
		AppendBlock(t, b)
	}
}

// FindLastBlocksByKind returns blocks of the requested kinds from the Turn ordered by Order asc.
func FindLastBlocksByKind(t Turn, kinds ...BlockKind) []Block {
	lookup := map[BlockKind]bool{}
	for _, k := range kinds {
		lookup[k] = true
	}
	ret := make([]Block, 0, len(t.Blocks))
	for _, b := range t.Blocks {
		if lookup[b.Kind] {
			ret = append(ret, b)
		}
	}
	return ret
}
