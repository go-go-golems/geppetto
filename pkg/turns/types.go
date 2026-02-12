package turns

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// Block represents a single atomic unit within a Turn.
type Block struct {
	ID      string         `yaml:"id,omitempty"`
	Kind    BlockKind      `yaml:"kind"`
	Role    string         `yaml:"role,omitempty"`
	Payload map[string]any `yaml:"payload,omitempty"`
	// Metadata stores arbitrary metadata about the block
	Metadata BlockMetadata `yaml:"metadata,omitempty"`
}

// Turn contains an ordered list of Blocks and associated metadata.
type Turn struct {
	ID     string  `yaml:"id,omitempty"`
	Blocks []Block `yaml:"blocks"`
	// Metadata stores arbitrary metadata about the turn
	Metadata Metadata `yaml:"metadata,omitempty"`
	// Data stores the application data payload associated with this turn
	Data Data `yaml:"data,omitempty"`
}

// Clone returns a deep copy of the Turn suitable for mutation without affecting the original.
//
// It copies:
// - Turn.ID
// - Turn.Metadata (shallow copy of the underlying map)
// - Turn.Data (shallow copy of the underlying map)
// - Blocks slice (new slice) and, for each block:
//   - Payload map (new map)
//   - Block.Metadata (shallow copy of the underlying map)
func (t *Turn) Clone() *Turn {
	if t == nil {
		return nil
	}
	out := &Turn{
		ID:       t.ID,
		Metadata: t.Metadata.Clone(),
		Data:     t.Data.Clone(),
	}
	if len(t.Blocks) == 0 {
		return out
	}
	out.Blocks = make([]Block, len(t.Blocks))
	for i := range t.Blocks {
		b := t.Blocks[i]
		if b.Payload != nil {
			cp := make(map[string]any, len(b.Payload))
			for k, v := range b.Payload {
				cp[k] = v
			}
			b.Payload = cp
		}
		b.Metadata = b.Metadata.Clone()
		out.Blocks[i] = b
	}
	return out
}

// NewKeyString constructs a canonical key identity string in the form "namespace.value@vN".
//
// This panics on invalid input (per design doc): empty namespace/value or version < 1.
func NewKeyString(namespace, value string, version uint16) string {
	if namespace == "" || value == "" || version < 1 {
		panic(fmt.Errorf("invalid key: namespace=%q value=%q version=%d", namespace, value, version))
	}
	return fmt.Sprintf("%s.%s@v%d", namespace, value, version)
}

func NewTurnDataKey(namespace, value string, version uint16) TurnDataKey {
	return TurnDataKey(NewKeyString(namespace, value, version))
}

func NewTurnMetadataKey(namespace, value string, version uint16) TurnMetadataKey {
	return TurnMetadataKey(NewKeyString(namespace, value, version))
}

func NewBlockMetadataKey(namespace, value string, version uint16) BlockMetadataKey {
	return BlockMetadataKey(NewKeyString(namespace, value, version))
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

// Clone returns a shallow copy of the underlying map (keys and values are copied,
// but any reference-typed values inside remain shared).
func (d Data) Clone() Data {
	if len(d.m) == 0 {
		return Data{}
	}
	out := make(map[TurnDataKey]any, len(d.m))
	for k, v := range d.m {
		out[k] = v
	}
	return Data{m: out}
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

// Clone returns a shallow copy of the underlying map (keys and values are copied,
// but any reference-typed values inside remain shared).
func (m Metadata) Clone() Metadata {
	if len(m.m) == 0 {
		return Metadata{}
	}
	out := make(map[TurnMetadataKey]any, len(m.m))
	for k, v := range m.m {
		out[k] = v
	}
	return Metadata{m: out}
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

// Clone returns a shallow copy of the underlying map (keys and values are copied,
// but any reference-typed values inside remain shared).
func (bm BlockMetadata) Clone() BlockMetadata {
	if len(bm.m) == 0 {
		return BlockMetadata{}
	}
	out := make(map[BlockMetadataKey]any, len(bm.m))
	for k, v := range bm.m {
		out[k] = v
	}
	return BlockMetadata{m: out}
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
