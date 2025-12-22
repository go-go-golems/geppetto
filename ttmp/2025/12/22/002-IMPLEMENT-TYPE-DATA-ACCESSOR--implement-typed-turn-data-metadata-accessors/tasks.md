# Tasks

## 1. Core API Implementation

### 1.1 Type Definitions
- [ ] Define `Key[T]` struct in `geppetto/pkg/turns/types.go`
  - Field: `id TurnDataKey` (or `TurnMetadataKey`/`BlockMetadataKey` as appropriate)
- [ ] Define `Data` struct (opaque wrapper for `Turn.Data`)
  - Private field: `m map[TurnDataKey]any`
- [ ] Define `Metadata` struct (opaque wrapper for `Turn.Metadata`)
  - Private field: `m map[TurnMetadataKey]any`
- [ ] Define `BlockMetadata` struct (opaque wrapper for `Block.Metadata`)
  - Private field: `m map[BlockMetadataKey]any`
- [ ] Update `Turn` struct to use `Data` and `Metadata` wrappers
- [ ] Update `Block` struct to use `BlockMetadata` wrapper

### 1.2 Key Constructor Functions
- [ ] Implement `NewTurnDataKey(namespace, value string, version uint16) TurnDataKey`
  - Validates inputs (non-empty namespace/value, version >= 1)
  - Returns `"namespace.slug@vN"` format
  - Panics on invalid input (as per design doc)
- [ ] Implement `K[T any](namespace, value string, version uint16) Key[T]`
  - Helper to create typed keys from namespace/value consts
  - Calls `NewTurnDataKey` internally
- [ ] Add `String()` method to `Key[T]` for debugging/logging

### 1.3 Data API Methods
- [ ] Implement `(d *Data) Set[T any](key Key[T], value T) error`
  - Initialize map if nil
  - Validate serializability via `json.Marshal(value)`
  - Store value in map
  - Return error on validation failure
- [ ] Implement `(d Data) Get[T any](key Key[T]) (T, bool, error)`
  - Return zero value, false, nil if map is nil
  - Return zero value, false, nil if key not found
  - Type assert value to T
  - Return error if type assertion fails
- [ ] Implement `(d Data) Len() int`
- [ ] Implement `(d Data) Range(fn func(TurnDataKey, any) bool)`
- [ ] Implement `(d *Data) Delete(key TurnDataKey)`

### 1.4 Metadata API Methods
- [ ] Implement `(m *Metadata) Set[T any](key Key[T], value T) error` (same pattern as Data)
- [ ] Implement `(m Metadata) Get[T any](key Key[T]) (T, bool, error)` (same pattern as Data)
- [ ] Implement `(m Metadata) Len() int`
- [ ] Implement `(m Metadata) Range(fn func(TurnMetadataKey, any) bool)`
- [ ] Implement `(m *Metadata) Delete(key TurnMetadataKey)`

### 1.5 BlockMetadata API Methods
- [ ] Implement `(bm *BlockMetadata) Set[T any](key Key[T], value T) error` (same pattern as Data)
- [ ] Implement `(bm BlockMetadata) Get[T any](key Key[T]) (T, bool, error)` (same pattern as Data)
- [ ] Implement `(bm BlockMetadata) Len() int`
- [ ] Implement `(bm BlockMetadata) Range(fn func(BlockMetadataKey, any) bool)`
- [ ] Implement `(bm *BlockMetadata) Delete(key BlockMetadataKey)`

### 1.6 YAML Serialization
- [ ] Implement `(d Data) MarshalYAML() (interface{}, error)`
  - Return `nil` if map is empty
  - Convert `map[TurnDataKey]any` to `map[string]any` for YAML
- [ ] Implement `(d *Data) UnmarshalYAML(value *yaml.Node) error`
  - Parse `map[string]any` from YAML
  - Convert string keys to `TurnDataKey`
  - Validate key format `"namespace.slug@vN"` (accept as-is, linter enforces canonical keys)
- [ ] Implement `(m Metadata) MarshalYAML() (interface{}, error)` (same pattern)
- [ ] Implement `(m *Metadata) UnmarshalYAML(value *yaml.Node) error` (same pattern)
- [ ] Implement `(bm BlockMetadata) MarshalYAML() (interface{}, error)` (same pattern)
- [ ] Implement `(bm *BlockMetadata) UnmarshalYAML(value *yaml.Node) error` (same pattern)

### 1.7 Remove Helper Functions
- [ ] Delete `SetTurnMetadata` function from `geppetto/pkg/turns/types.go`
- [ ] Delete `SetBlockMetadata` function from `geppetto/pkg/turns/types.go`
- [ ] Delete `HasBlockMetadata` function from `geppetto/pkg/turns/helpers_blocks.go`
- [ ] Delete `RemoveBlocksByMetadata` function from `geppetto/pkg/turns/helpers_blocks.go`
- [ ] Delete `WithBlockMetadata` function from `geppetto/pkg/turns/helpers_blocks.go`

### 1.8 Tests
- [ ] Unit tests for `Set` with valid values
- [ ] Unit tests for `Set` with non-serializable values (should error)
- [ ] Unit tests for `Get` with existing keys (correct type)
- [ ] Unit tests for `Get` with existing keys (wrong type, should error)
- [ ] Unit tests for `Get` with missing keys (should return false, nil)
- [ ] Unit tests for `Get` with nil map (should return false, nil)
- [ ] Unit tests for `Len()` (empty, non-empty maps)
- [ ] Unit tests for `Range()` (iteration, early termination)
- [ ] Unit tests for `Delete()` (existing key, missing key)
- [ ] YAML marshal/unmarshal round-trip tests
- [ ] YAML marshal with empty maps (should be nil/omitted)
- [ ] YAML unmarshal with invalid key formats (should accept but linter will catch)

---

## 2. Linting Rules (with Tests)

### 2.1 Ban Ad-Hoc Key Construction
- [ ] Add linter rule: `NewTurnDataKey(...)` calls only allowed in `*/keys.go` or `*/turnkeys/*.go`
- [ ] Add linter rule: `TurnDataKey("...")` conversions only allowed in `*/keys.go` or `*/turnkeys/*.go`
- [ ] Add linter rule: `TurnMetadataKey("...")` conversions only allowed in canonical key files
- [ ] Add linter rule: `BlockMetadataKey("...")` conversions only allowed in canonical key files
- [ ] Test: Linter flags `NewTurnDataKey` call outside canonical key files
- [ ] Test: Linter flags `TurnDataKey("foo")` conversion outside canonical key files
- [ ] Test: Linter allows `NewTurnDataKey` in `pkg/turns/keys.go`
- [ ] Test: Linter allows `TurnDataKey("...")` in `moments/backend/pkg/turnkeys/data_keys.go`

### 2.2 Enforce Canonical Key Format
- [ ] Add linter rule: Key expressions must use `K[T](namespaceConst, valueConst, version)` pattern
- [ ] Add linter rule: Namespace parameter must be string const (not variable or literal)
- [ ] Add linter rule: Value parameter must be string const (not variable or literal)
- [ ] Add linter rule: Version must be numeric literal or const
- [ ] Test: Linter flags `K[string]("namespace", "value", 1)` with string literals
- [ ] Test: Linter flags `K[string](namespaceVar, valueConst, 1)` with variable namespace
- [ ] Test: Linter allows `K[string](NamespaceConst, ValueConst, 1)` with consts

### 2.3 Enforce Key Format Regex
- [ ] Add linter rule: Resulting key string must match `^[a-z]+\.[a-z_]+@v\d+$` format
- [ ] Test: Linter flags keys that don't match format (e.g., `"Namespace.value@v1"`, `"namespace.value"`, `"namespace.value@v"`)
- [ ] Test: Linter allows keys matching format (e.g., `"geppetto.tool_config@v1"`)

### 2.4 Ban Direct Map Access
- [ ] Add linter rule: Ban `t.Data[key]` (must use `t.Data.Get(key)` or `t.Data.Set(key, value)`)
- [ ] Add linter rule: Ban `t.Metadata[key]` (must use wrapper API)
- [ ] Add linter rule: Ban `b.Metadata[key]` (must use wrapper API)
- [ ] Add linter rule: Ban `t.Data[key] = value` (must use `t.Data.Set(key, value)`)
- [ ] Add linter rule: Ban `t.Metadata[key] = value` (must use wrapper API)
- [ ] Add linter rule: Ban `b.Metadata[key] = value` (must use wrapper API)
- [ ] Test: Linter flags `t.Data[key]` read access
- [ ] Test: Linter flags `t.Data[key] = value` write access
- [ ] Test: Linter allows `t.Data.Get(key)` wrapper API
- [ ] Test: Linter allows `t.Data.Set(key, value)` wrapper API

### 2.5 Deprecation Warnings
- [ ] Add linter rule: Parse `// Deprecated:` comments on key variables
- [ ] Add linter rule: Warn at usage sites of deprecated keys
- [ ] Test: Linter warns when using deprecated key
- [ ] Test: Linter doesn't warn for non-deprecated keys

### 2.6 Configurable Strictness
- [ ] Add linter flag: `--strict` (errors) vs default (warnings)
- [ ] Test: Linter emits errors in strict mode
- [ ] Test: Linter emits warnings in default mode

### 2.7 Test Infrastructure
- [ ] Create testdata directory structure for linter tests
- [ ] Add test cases for each linter rule
- [ ] Add test cases for edge cases (nested calls, function parameters, etc.)
- [ ] Verify linter test framework can detect violations correctly

---

## 3. Clean Up Geppetto

### 3.1 Migrate Core Types
- [ ] Update `geppetto/pkg/turns/types.go` to use wrapper types
- [ ] Remove all direct map access in `geppetto/pkg/turns/types.go`
- [ ] Update `geppetto/pkg/turns/helpers_blocks.go` to use wrapper API
- [ ] Remove helper functions: `SetTurnMetadata`, `SetBlockMetadata`, `HasBlockMetadata`, `RemoveBlocksByMetadata`, `WithBlockMetadata`

### 3.2 Migrate Key Definitions
- [ ] Update `geppetto/pkg/turns/keys.go` to use `K[T]` constructor
- [ ] Convert all `const` keys to typed `var` keys with namespace/version
- [ ] Add namespace consts: `GeppettoNamespaceKey = "geppetto"`
- [ ] Add value key consts for each key (e.g., `ToolConfigValueKey = "tool_config"`)
- [ ] Create typed keys: `KeyToolConfig = K[engine.ToolConfig](GeppettoNamespaceKey, ToolConfigValueKey, 1)`
- [ ] Update all key references to use typed keys

### 3.3 Migrate Serialization
- [ ] Update `geppetto/pkg/turns/serde/serde.go` to work with wrapper types
- [ ] Remove nil map initialization from `NormalizeTurn` (wrapper handles this)
- [ ] Verify YAML round-trip tests still pass

### 3.4 Migrate Tool Helpers
- [ ] Update `geppetto/pkg/inference/toolhelpers/helpers.go` to use wrapper API
- [ ] Replace `t.Data = map[...]any{}` with wrapper initialization (may be automatic)
- [ ] Replace `t.Data[key] = value` with `t.Data.Set(key, value)`
- [ ] Add error handling for `Set` calls

### 3.5 Migrate Middleware
- [ ] Update `geppetto/pkg/inference/middleware/systemprompt_middleware.go`
  - Replace `t.Blocks[i].Metadata[key] = value` with `b.Metadata.Set(key, value)`
- [ ] Update `geppetto/pkg/inference/middleware/tool_middleware.go`
  - Replace `message.Metadata["tool_calls"]` with wrapper API (fix raw string literal)
- [ ] Update any other geppetto middleware files

### 3.6 Migrate Step Helpers
- [ ] Update `geppetto/pkg/steps/ai/claude/helpers.go`
  - Replace `b.Metadata[turns.BlockMetaKeyClaudeOriginalContent]` with `b.Metadata.Get(key)`

### 3.7 Update Tests
- [ ] Update `geppetto/pkg/turns/serde/serde_test.go` to use wrapper API
- [ ] Update all geppetto tests that access `Turn.Data` or `Turn.Metadata` or `Block.Metadata`
- [ ] Verify all tests pass with new API

### 3.8 Documentation
- [ ] Update `geppetto/pkg/doc/topics/08-turns.md` to document new wrapper API
- [ ] Remove documentation references to removed helper functions
- [ ] Add examples showing wrapper API usage

---

## Notes

- **Order of implementation:** Core API → Linting → Geppetto cleanup
- **Breaking changes:** Helper functions are removed immediately (no deprecation)
- **Test coverage:** Each component must have comprehensive tests before moving to next phase
- **Review checkpoints:** Review after Core API, after Linting, and after Geppetto cleanup
