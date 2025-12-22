---
Title: 'Final design: typed Turn.Data/Metadata accessors'
Ticket: 001-REVIEW-TYPED-DATA-ACCESS
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/analysis/turnsdatalint/analyzer.go
      Note: Linter to be enhanced
    - Path: geppetto/pkg/turns/keys.go
      Note: Current canonical key definitions
    - Path: geppetto/pkg/turns/types.go
      Note: Current Turn.Data structure to be replaced
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/analysis/01-codebase-analysis-turn-data-metadata-access-locations.md
      Note: Inventory of migration sites for this design
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/reference/01-diary.md
      Note: Implementation diary (steps + commit hashes) for this design
    - Path: geppetto/ttmp/2025/12/22/002-IMPLEMENT-TYPE-DATA-ACCESSOR--implement-typed-turn-data-metadata-accessors/sources/go-generic-methods.md
      Note: Generic-method limitation note (why DataGet/DataSet functions)
    - Path: moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Compression middleware that needs refactoring to work with typed API
    - Path: moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Example middleware that will migrate to new API
ExternalSources: []
Summary: 'Final design for typed Turn.Data/Metadata accessors: opaque wrapper with typed keys, encoded string key identity, any storage with validation, and comprehensive linting.'
LastUpdated: 2025-12-22T16:00:00-05:00
WhatFor: 'Implementation guide: concrete API, type definitions, migration patterns, and linting rules.'
WhenToUse: Use when implementing typed Turn.Data/Metadata accessors or migrating existing code.
---


# Final Design: Typed Turn.Data/Metadata Accessors

## Executive Summary

We are replacing the current `map[TurnDataKey]any` access pattern with an **opaque wrapper** that provides type-safe access via **typed keys** `Key[T]`. This design:

- **Eliminates type assertion boilerplate** via typed keys with type inference
- **Centralizes nil-map initialization** in the wrapper
- **Enforces serializability** at write time (fail-fast validation)
- **Prevents bypasses** via opaque API boundary
- **No escape hatches** - all access must go through typed API

**Scope:** `Turn.Data`, `Turn.Metadata`, `Block.Metadata`. `Block.Payload` is unchanged.

**Breaking change:** All direct map access (`t.Data[key] = value`) must migrate to wrapper API (`t.Data.Set(key, value)`).

**Removed helpers:** `SetTurnMetadata`, `SetBlockMetadata`, `HasBlockMetadata`, `RemoveBlocksByMetadata`, `WithBlockMetadata` are removed. All access must use the wrapper API directly. No deprecation period or backward compatibility.

---

## Design Decisions

### Decision 1: Key Identity — Fixed Namespace and Value Keys

**Chosen:** String consts for namespace keys and value keys, combined with version. Similar to current design but with explicit namespace/value separation.

**Rationale:**
- Compile-time consts (familiar Go pattern, prevents typos)
- Clear ownership (namespace keys defined per package, value keys scoped to namespace)
- Linter can enforce canonical keys (consts, not ad-hoc strings)
- Simple serialization format `"namespace.slug@vN"` (constructed from consts)

**Key structure:**
```go
// Namespace keys (defined once per package)
const MentoNamespaceKey = "mento"
const GeppettoNamespaceKey = "geppetto"

// Value keys (scoped to namespace, defined per package)
const UserDisplayNameValueKey = "user_display_name"
const PersonIDValueKey = "person_id"
const ToolConfigValueKey = "tool_config"
```

**Serialization format:** `"namespace.slug@vN"` (e.g., `"mento.user_display_name@v1"`)

**Example canonical keys:**
```go
var KeyUserDisplayName = K[string](MentoNamespaceKey, UserDisplayNameValueKey, 1)
var KeyPersonID = K[string](MentoNamespaceKey, PersonIDValueKey, 1)
var KeyToolConfig = K[engine.ToolConfig](GeppettoNamespaceKey, ToolConfigValueKey, 1)
```

### Decision 2: Value Storage — `any` with Validation

**Chosen:** Store `any`, validate serializability on `Set` by attempting JSON marshal.

**Rationale:**
- Fast reads (no unmarshal cost)
- Fail-fast validation (errors at write time, not persistence time)
- Performance matters for middleware hot paths
- Validation is sufficient guardrail (we can't accidentally store non-serializable values)

**Implementation:** `Set` attempts `json.Marshal(value)` before storing. If marshal fails, `Set` returns error.

### Decision 3: API Boundary — Opaque Wrapper

**Chosen:** Opaque wrapper with private map, public API methods.

**Rationale:**
- Structural enforcement (can't bypass API)
- Centralized nil-map initialization
- Clear API contract
- Consistent behavior across `Turn.Data`, `Turn.Metadata`, `Block.Metadata`

**Trade-off:** Breaking change, but provides strongest guarantees.

**Removed helpers:** Existing helper functions (`SetTurnMetadata`, `SetBlockMetadata`, `HasBlockMetadata`, `RemoveBlocksByMetadata`, `WithBlockMetadata`) are removed. All code must use the wrapper API directly (`t.Metadata.Set(key, value)`, `b.Metadata.Get(key)`, etc.). No deprecation period — these functions are deleted immediately.

### Decision 4: Error Handling — Always Check Errors

**Chosen:** `Set` always returns error (no panic variant). `Get` always returns `(T, bool, error)`.

**Rationale:**
- Consistent error handling across all operations
- No panic variants - all errors are explicit
- Forces explicit error handling at call sites
- Testable error cases

### Decision 5: Versioning — Required from Day One

**Chosen:** All keys must include `@vN` suffix (no default version).

**Rationale:**
- Forces clarity from day one
- Makes version changes explicit
- Linter can enforce format

### Decision 6: Linting — Strong Enforcement

**Chosen:** Ban ad-hoc keys, enforce naming format, warn on deprecation, configurable strictness.

**Rationale:**
- Prevents drift (can't construct `TurnDataKey("oops")` outside canonical packages)
- Enforces namespace conventions
- Makes deprecations visible

---

## API Design

### Type Definitions

```go
package turns

// Key identity: constructed from namespace + value + version
type TurnDataKey string
type TurnMetadataKey string
type BlockMetadataKey string

// Constructor: builds "namespace.slug@vN" format
func NewTurnDataKey(namespace, value string, version uint16) TurnDataKey {
    if namespace == "" || value == "" || version < 1 {
        panic(fmt.Errorf("invalid key: namespace=%q value=%q version=%d", namespace, value, version))
    }
    return TurnDataKey(fmt.Sprintf("%s.%s@v%d", namespace, value, version))
}

// Typed key wrapper (enables type inference)
type Key[T any] struct {
    id TurnDataKey
}

// Helper to create typed keys from namespace/value consts
func K[T any](namespace, value string, version uint16) Key[T] {
    return Key[T]{id: NewTurnDataKey(namespace, value, version)}
}

// Opaque wrapper for Turn.Data
type Data struct {
    m map[TurnDataKey]any  // private
}

// Opaque wrapper for Turn.Metadata
type Metadata struct {
    m map[TurnMetadataKey]any  // private
}

// Opaque wrapper for Block.Metadata
type BlockMetadata struct {
    m map[BlockMetadataKey]any  // private
}
```

### Data API

```go
// IMPORTANT: Go does NOT allow methods to declare their own type parameters.
// The compiler rejects this with: "syntax error: method must have no type parameters".
//
// Therefore the typed API is implemented as generic *functions* taking the wrapper
// as an explicit argument.

// Write operations
func DataSet[T any](d *Data, key Key[T], value T) error

// Read operations
func DataGet[T any](d Data, key Key[T]) (T, bool, error)

// Utility operations
func (d Data) Len() int
func (d Data) Range(fn func(TurnDataKey, any) bool)
func (d *Data) Delete(key TurnDataKey)
```

### Turn Structure

```go
type Turn struct {
    ID       string     `yaml:"id,omitempty"`
    RunID    string     `yaml:"run_id,omitempty"`
    Blocks   []Block    `yaml:"blocks"`
    Metadata Metadata   `yaml:"metadata,omitempty"`
    Data     Data       `yaml:"data,omitempty"`
}
```

### YAML Serialization

```go
// MarshalYAML: TurnDataKey is already string format
func (d Data) MarshalYAML() (interface{}, error) {
    if len(d.m) == 0 {
        return nil, nil
    }
    result := make(map[string]any)
    for k, v := range d.m {
        result[string(k)] = v
    }
    return result, nil
}

// UnmarshalYAML: parse string format to TurnDataKey
func (d *Data) UnmarshalYAML(value *yaml.Node) error {
    if value == nil {
        d.m = nil
        return nil
    }
    var raw map[string]any
    if err := value.Decode(&raw); err != nil {
        return err
    }
    d.m = make(map[TurnDataKey]any, len(raw))
    for kStr, v := range raw {
        // Validate format "namespace.slug@vN" but accept as-is (linter enforces canonical keys)
        d.m[TurnDataKey(kStr)] = v
    }
    return nil
}
```

---

## Implementation Details

### Set Implementation

```go
func DataSet[T any](d *Data, key Key[T], value T) error {
    if d.m == nil {
        d.m = make(map[TurnDataKey]any)
    }

    // Validate serializability
    if _, err := json.Marshal(value); err != nil {
        return fmt.Errorf("Turn.Data[%q]: value not serializable: %w", key.id, err)
    }

    d.m[key.id] = value
    return nil
}
```

### Get Implementation

```go
func DataGet[T any](d Data, key Key[T]) (T, bool, error) {
    var zero T
    
    if d.m == nil {
        return zero, false, nil
    }
    
    value, ok := d.m[key.id]
    if !ok {
        return zero, false, nil
    }
    
    // Type assertion with error
    typed, ok := value.(T)
    if !ok {
        return zero, true, fmt.Errorf("Turn.Data[%q]: expected %T, got %T", key.id, zero, value)
    }
    
    return typed, true, nil
}
```


---

## Canonical Key Definitions

### Geppetto Keys (`geppetto/pkg/turns/keys.go`)

```go
package turns

// Namespace key (defined once)
const GeppettoNamespaceKey = "geppetto"

// Value keys (scoped to Geppetto namespace)
const (
    ToolConfigValueKey = "tool_config"
    AgentModeValueKey = "agent_mode"
    AgentModeAllowedToolsValueKey = "agent_mode_allowed_tools"
    ResponsesServerToolsValueKey = "responses_server_tools"
)

// Typed keys for Turn.Data
var (
    KeyAgentMode = K[string](GeppettoNamespaceKey, AgentModeValueKey, 1)
    KeyAgentModeAllowedTools = K[[]string](GeppettoNamespaceKey, AgentModeAllowedToolsValueKey, 1)
    KeyResponsesServerTools = K[[]any](GeppettoNamespaceKey, ResponsesServerToolsValueKey, 1)
)
```

### Engine-owned keys (`geppetto/pkg/inference/engine/turnkeys.go`)

`ToolConfig` is defined in the `engine` package, and `engine` already depends on `turns` (the Engine interface uses `*turns.Turn`), so `turns` must NOT import `engine` (import cycle). Therefore the typed key for `ToolConfig` is owned by `engine`:

```go
package engine

import "github.com/go-go-golems/geppetto/pkg/turns"

var KeyToolConfig = turns.K[ToolConfig](turns.GeppettoNamespaceKey, turns.ToolConfigValueKey, 1)
```

### Moments Keys (`moments/backend/pkg/turnkeys/keys.go`)

```go
package turnkeys

import "github.com/go-go-golems/geppetto/pkg/turns"

// Namespace key (defined once)
const MentoNamespaceKey = "mento"

// Value keys (scoped to Mento namespace)
const (
    PersonIDValueKey = "person_id"
    UserPrimaryEmailValueKey = "user_primary_email"
    UserDisplayNameValueKey = "user_display_name"
    ThinkingModeValueKey = "thinking_mode"
    // ... more value keys
)

// Typed keys for Turn.Data
var (
    PersonID = turns.K[string](MentoNamespaceKey, PersonIDValueKey, 1)
    UserPrimaryEmail = turns.K[string](MentoNamespaceKey, UserPrimaryEmailValueKey, 1)
    UserDisplayName = turns.K[string](MentoNamespaceKey, UserDisplayNameValueKey, 1)
    ThinkingMode = turns.K[string](MentoNamespaceKey, ThinkingModeValueKey, 1)
    // ... more keys
)
```

---

## Linting Rules

### Enhanced `turnsdatalint` Rules

1. **Ban ad-hoc key construction**: `NewTurnDataKey(...)` or `TurnDataKey("...")` only allowed in `*/keys.go` or `*/turnkeys/*.go`
2. **Enforce canonical keys**: Key expressions must use `K[T](namespaceConst, valueConst, version)` with const namespace/value keys
3. **Enforce namespace/value consts**: Namespace and value must be string consts (not variables or literals)
4. **Enforce key format**: Resulting key must match `^[a-z]+\.[a-z_]+@v\d+$` format
5. **Deprecation warnings**: Parse `// Deprecated:` comments, warn at usage sites
6. **Configurable strictness**: Strict in CI (errors), permissive locally (warnings)

### Test Key Policy

**Chosen:** Require canonical test keys (no `test.*` exemption).

**Rationale:** Keeps tests honest, prevents test-only keys from leaking into production.

---

## Migration Guide

### Before (Current Pattern)

```go
// Middleware pattern
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
modeName, _ := t.Data[turnkeys.ThinkingMode].(string)
if modeName == "" {
    modeName = ModeExploring
    t.Data[turnkeys.ThinkingMode] = modeName
}
```

### After (New Pattern)

```go
// Middleware pattern - always check error
mode, ok, err := turns.DataGet(t.Data, turnkeys.ThinkingMode)
if err != nil {
    return nil, fmt.Errorf("decode error: %w", err)
}
if !ok || mode == "" {
    mode = ModeExploring
    if err := turns.DataSet(&t.Data, turnkeys.ThinkingMode, mode); err != nil {
        return nil, fmt.Errorf("set thinking mode: %w", err)
    }
}
```

### Removed Helper Functions

The following helper functions are **removed** (no deprecation period):

**Before (removed):**
```go
// REMOVED - use wrapper API directly
turns.SetTurnMetadata(t, key, value)
turns.SetBlockMetadata(b, key, value)
turns.HasBlockMetadata(b, key, value)
turns.RemoveBlocksByMetadata(t, key, values...)
turns.WithBlockMetadata(b, kvs)
```

**After (use wrapper API):**
```go
// SetTurnMetadata replacement
if err := turns.MetadataSet(&t.Metadata, key, value); err != nil {
    return fmt.Errorf("set metadata: %w", err)
}

// SetBlockMetadata replacement
if err := turns.BlockMetadataSet(&b.Metadata, key, value); err != nil {
    return fmt.Errorf("set block metadata: %w", err)
}

// HasBlockMetadata replacement
value, ok, err := turns.BlockMetadataGet(b.Metadata, key)
if err != nil {
    return false, fmt.Errorf("get block metadata: %w", err)
}
if ok && value == expectedValue {
    // match found
}

// RemoveBlocksByMetadata replacement
for i := len(t.Blocks) - 1; i >= 0; i-- {
    b := &t.Blocks[i]
    value, ok, err := turns.BlockMetadataGet(b.Metadata, key)
    if err != nil {
        continue // skip on error
    }
    if ok {
        for _, v := range values {
            if value == v {
                t.Blocks = append(t.Blocks[:i], t.Blocks[i+1:]...)
                break
            }
        }
    }
}

// WithBlockMetadata replacement
cloned := b
if err := turns.BlockMetadataSet(&cloned.Metadata, key, value); err != nil {
    return b, fmt.Errorf("set metadata: %w", err)
}
return cloned
```

### Compression Middleware (Refactored)

```go
// Before: direct map access
func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any)

// After: work with typed API via Range
func (tdc *TurnDataCompressor) Compress(ctx context.Context, turn *turns.Turn) error {
        // Iterate over all keys using Range
    turn.Data.Range(func(key turns.TurnDataKey, value any) bool {
        // Compression logic works on value directly
        switch v := value.(type) {
        case string:
            if compressed := tdc.compressString(v); compressed != v {
                // Need typed key to write back - compression must know key types
                // Option: maintain registry of known keys, or refactor compression
                // to work with specific typed keys rather than generic iteration
            }
        }
        return true
    })
    return nil
}

// Better approach: compression works with known typed keys
func (tdc *TurnDataCompressor) CompressKnownKeys(ctx context.Context, turn *turns.Turn, keys []turns.Key[any]) error {
    for _, key := range keys {
        value, ok, err := turn.Data.Get(key)
        if err != nil {
            return fmt.Errorf("get %q: %w", key.id.String(), err)
        }
        if !ok {
            continue
        }
        // Compress based on type
        if compressed := tdc.compressValue(value); compressed != value {
            if err := turn.Data.Set(key, compressed); err != nil {
                return fmt.Errorf("set %q: %w", key.id.String(), err)
            }
        }
    }
    return nil
}
```

---

## Alternatives Considered

### Encoded String Keys Only

**Not chosen:** Would allow ad-hoc key construction, runtime validation only, no compile-time guarantees.

**Why structured keys:** Compile-time enforcement via unexported fields, type-safe key identity, string format only for serialization (human-readable YAML).

### `json.RawMessage` Storage

**Not chosen:** Unmarshal cost on every `Get` would hurt middleware performance.

**Why `any` with validation:** Fast reads, fail-fast validation is sufficient guardrail.

### Public Map + Helpers

**Not chosen:** Can be bypassed, relies on linting for enforcement.

**Why opaque wrapper:** Structural guarantees, centralized initialization, consistent behavior.

---

## Success Criteria

- **Fewer type assertion bugs**: Measure via grep for `.(T)` patterns in Turn.Data access
- **Fewer nil map panics**: Measure via crash reports and test failures
- **Clearer key ownership**: Measure time to find key definition (should be instant via jump-to-definition)
- **Faster code review**: Measure reviewer questions about key types (should decrease)

---

## Implementation Plan

1. **Implement opaque wrapper types** (`Data`, `Metadata`, `BlockMetadata`)
2. **Implement API methods** (`Get`, `Set`, `Range`, `Delete`, `Len`)
3. **Implement YAML marshal/unmarshal** for wrappers
4. **Convert canonical keys** to typed keys with namespace/version
5. **Enhance linter** (ban ad-hoc keys, enforce format, deprecation warnings)
6. **Migrate middleware** to new API (start with high-traffic ones)
7. **Refactor compression middleware** to work with typed API (via `Range` or known keys)
8. **Update tests** to always check errors from `Get`

---

## Related Documents

- **Synthesis:** `design-doc/02-debate-synthesis-v2-concise-rewrite.md` (design space exploration)
- **Review rounds:** `reference/04-review-round-1-initial-review-with-code-research.md`, `reference/05-review-round-2-document-clarity-and-conciseness.md`
