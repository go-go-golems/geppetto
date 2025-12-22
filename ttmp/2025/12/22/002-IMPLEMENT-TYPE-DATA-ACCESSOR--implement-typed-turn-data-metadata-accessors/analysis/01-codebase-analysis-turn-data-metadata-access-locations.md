---
Title: 'Codebase analysis: Turn.Data/Metadata access locations'
Ticket: 002-IMPLEMENT-TYPE-DATA-ACCESSOR
Status: active
Topics:
    - geppetto
    - turns
    - go
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go
      Note: Compression middleware that needs refactoring for typed API
    - Path: ../../../../../../../moments/backend/pkg/inference/middleware/current_user_middleware.go
      Note: Example middleware with Turn.Data access patterns
    - Path: ../../../../../../../moments/backend/pkg/turnkeys/block_meta_keys.go
      Note: Moments-specific Block.Metadata keys (to migrate)
    - Path: ../../../../../../../moments/backend/pkg/turnkeys/data_keys.go
      Note: |-
        Moments-specific Turn.Data keys (to migrate)
        Moments keys cataloged
    - Path: ../../../../../../../moments/backend/pkg/turnkeys/turn_meta_keys.go
      Note: Moments-specific Turn.Metadata keys (to migrate)
    - Path: ../../../../../../../pinocchio/pkg/middlewares/agentmode/middleware.go
      Note: Pinocchio middleware with Turn.Data access patterns
    - Path: pkg/analysis/turnsdatalint/analyzer.go
      Note: Linter that will need enhancement for new API
    - Path: pkg/inference/toolhelpers/helpers.go
      Note: Tool helpers that initialize and write to Turn.Data
    - Path: pkg/turns/keys.go
      Note: |-
        Current canonical key definitions (to be migrated to typed Key[T] pattern)
        Key definitions cataloged
    - Path: pkg/turns/serde/serde.go
      Note: Serialization normalization that initializes nil maps
    - Path: pkg/turns/types.go
      Note: |-
        Core Turn/Block type definitions with current map-based Data/Metadata fields
        Core type definitions analyzed
ExternalSources: []
Summary: Comprehensive mapping of all Turn.Data, Turn.Metadata, and Block.Metadata access locations across geppetto, moments, pinocchio, and bobatea codebases. Identifies migration targets, helper functions, serialization points, and linter integration needs.
LastUpdated: 2025-12-22T16:00:00-05:00
WhatFor: 'Implementation planning: identify all code locations that must migrate from map access to typed wrapper API.'
WhenToUse: Reference when planning migration phases, estimating effort, and validating completeness.
---


# Codebase Analysis: Turn.Data/Metadata Access Locations

## Executive Summary

This analysis maps all locations across **geppetto**, **moments**, **pinocchio**, and **bobatea** where `Turn.Data`, `Turn.Metadata`, and `Block.Metadata` are accessed. The design doc (`001-REVIEW-TYPED-DATA-ACCESS`) specifies replacing `map[TurnDataKey]any` with an opaque wrapper providing type-safe `Get[T]`/`Set[T]` accessors.

**Scope:** All direct map access (`t.Data[key]`, `t.Metadata[key]`, `b.Metadata[key]`) must migrate to wrapper API (`t.Data.Get(key)`, `t.Data.Set(key, value)`).

**Total locations identified:** ~150+ access sites across 4 repositories.

---

## Analysis Methodology

1. **Grep searches** for `.Data[`, `.Metadata[` patterns
2. **Codebase semantic search** for Turn creation/initialization
3. **File-by-file review** of key middleware and helper files
4. **Helper function analysis** (`SetTurnMetadata`, `SetBlockMetadata`, `HasBlockMetadata`)
5. **Serialization point identification** (YAML marshal/unmarshal)

---

## Repository Breakdown

### Geppetto (`geppetto/`)

**Core types and keys:**
- `pkg/turns/types.go`: `Turn.Data`, `Turn.Metadata`, `Block.Metadata` definitions
- `pkg/turns/keys.go`: Canonical key constants (`DataKeyToolConfig`, `TurnMetaKeyModel`, etc.)
- `pkg/turns/key_types.go`: Type definitions (`TurnDataKey`, `TurnMetadataKey`, `BlockMetadataKey`)

**Access patterns found:**

#### Turn.Data Access (Geppetto)

1. **Initialization sites:**
   - `pkg/turns/serde/serde.go:25`: `t.Data = map[turns.TurnDataKey]any{}` (normalization)
   - `pkg/inference/toolhelpers/helpers.go:293,297`: `t.Data = map[turns.TurnDataKey]any{}` (tool loop initialization)

2. **Write operations:**
   - `pkg/inference/toolhelpers/helpers.go:304`: `t.Data[turns.DataKeyToolConfig] = engine.ToolConfig{...}` (tool config)

3. **Read operations:**
   - No direct reads found in geppetto (likely accessed via middleware in moments/pinocchio)

#### Turn.Metadata Access (Geppetto)

1. **Helper functions:**
   - `pkg/turns/types.go:147`: `SetTurnMetadata(t *Turn, key TurnMetadataKey, value interface{})` - initializes map if nil, writes directly

2. **Read operations:**
   - `pkg/turns/serde/serde_test.go:57-59`: Test assertions reading `turn.Metadata[turns.TurnMetaKeyModel]`

#### Block.Metadata Access (Geppetto)

1. **Helper functions:**
   - `pkg/turns/types.go:155`: `SetBlockMetadata(b *Block, key BlockMetadataKey, value interface{})` - initializes map if nil, writes directly
   - `pkg/turns/helpers_blocks.go:111`: `HasBlockMetadata(b Block, key BlockMetadataKey, value string) bool` - reads `b.Metadata[key]`
   - `pkg/turns/helpers_blocks.go:127`: `RemoveBlocksByMetadata` - iterates blocks, reads `b.Metadata[key]`

2. **Direct access:**
   - `pkg/inference/middleware/systemprompt_middleware.go:51`: `t.Blocks[firstSystemIdx].Metadata[turns.BlockMetaKeyMiddleware] = "systemprompt"`
   - `pkg/steps/ai/claude/helpers.go:78,129`: Reads `b.Metadata[turns.BlockMetaKeyClaudeOriginalContent]`
   - `pkg/inference/middleware/tool_middleware.go:258`: Reads `message.Metadata["tool_calls"]` (raw string - should be caught by linter)
   - `pkg/inference/toolhelpers/helpers.go:114`: Reads `lastMessage.Metadata["claude_original_content"]` (raw string - should be caught by linter)

#### Serialization (Geppetto)

- `pkg/turns/serde/serde.go:24-28`: `NormalizeTurn` initializes nil maps before serialization
- YAML marshal/unmarshal handled by `gopkg.in/yaml.v3` via struct tags

---

### Moments (`moments/backend/`)

**Key definitions:**
- `pkg/turnkeys/data_keys.go`: 20+ Turn.Data keys (`PersonID`, `ProfileSlug`, `ThinkingMode`, etc.)
- `pkg/turnkeys/turn_meta_keys.go`: Turn.Metadata keys (`TurnMetaKeyAPIType`, `TurnMetaKeyTemperature`, etc.)
- `pkg/turnkeys/block_meta_keys.go`: Block.Metadata keys (`BlockMetaKeyMemoryContext`, `BlockMetaKeyMemoryExtraction`)

**Access patterns found:**

#### Turn.Data Access (Moments)

1. **Initialization sites:**
   - `pkg/webchat/router.go:551,554`: `conv.Turn.Data = map[turns.TurnDataKey]any{}` (profile metadata setup)
   - `pkg/inference/middleware/current_user_middleware.go:42`: `t.Data = map[turns.TurnDataKey]any{}` (middleware init)

2. **Write operations (middleware):**
   - `pkg/inference/middleware/current_user_middleware.go:71,74,77`: Writes `PersonID`, `UserPrimaryEmail`, `UserDisplayName`
   - `pkg/webchat/router.go:559,565`: Writes `PromptSlugPrefix`, `ProfileSlug`
   - `pkg/inference/middleware/thinkingmode/middleware.go`: Reads/writes `ThinkingMode` (via helper pattern)
   - `pkg/inference/middleware/team_suggestions_middleware.go`: Writes `TeamSuggestions`
   - `pkg/inference/middleware/relationships_middleware.go`: Writes relationship keys

3. **Read operations:**
   - `pkg/inference/middleware/current_user_middleware.go:104`: Reads `t.Data[turnkeys.PersonID].(string)` (type assertion)
   - `pkg/artifact/metadata_extractor.go:85`: Reads `turn.Data[turnkeys.ProfileSlug].(string)` (type assertion)
   - `pkg/promptutil/resolve.go`: Reads `PromptSlugPrefix`, `ProfileSlug` for prompt resolution
   - Multiple middleware read `ThinkingMode`, `PersonID`, `ProfileSlug` for conditional logic

4. **Compression middleware:**
   - `pkg/inference/middleware/compression/turn_data_compressor.go:33`: **Takes `map[string]any`** (not `Turn.Data` directly) - needs refactoring
   - Called from `pkg/inference/middleware/conversation_compression_middleware.go` which converts `Turn.Data` to `map[string]any`

#### Turn.Metadata Access (Moments)

1. **Write operations:**
   - `pkg/inference/middleware/conversation_compression_middleware.go:101,104`: `turns.SetTurnMetadata` calls (uses helper)
   - `pkg/webchat/langfuse_middleware.go:180-335`: Multiple reads of `t.Metadata[turnkeys.TurnMetaKeyModel]`, `TurnMetaKeyTemperature`, etc. (type assertions)

2. **Read operations:**
   - `pkg/inference/middleware/conversation_compression_middleware.go:86`: Reads `t.Metadata[turns.TurnMetadataKey(key)]` (type assertion)
   - `pkg/webchat/langfuse_middleware.go`: Extensive reads for logging/tracing

#### Block.Metadata Access (Moments)

1. **Helper usage (common pattern):**
   - `pkg/inference/middleware/current_user_middleware.go:86-87`: `turns.HasBlockMetadata` calls
   - `pkg/memory/middleware.go:28`: `turns.HasBlockMetadata` for memory extraction detection
   - `pkg/memory/context_middleware.go:53`: `turns.HasBlockMetadata` for memory context detection
   - `pkg/inference/middleware/relationships_middleware.go:195-196,227-228,243-244`: Multiple `HasBlockMetadata` calls
   - `pkg/inference/middleware/team_member_blocks_middleware.go:50`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/team_suggestions_middleware.go:129`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/thinkingmode/middleware.go:86`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/debate/middleware.go:41`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/doc_blocks_middleware.go:44`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/coachingconversationsummary/middleware.go:42`: `HasBlockMetadata` for block identification
   - `pkg/inference/middleware/coachingguidelines/middleware.go:42`: `HasBlockMetadata` for block identification
   - `pkg/drive1on1/chat/chat.go:96`: `HasBlockMetadata` for block identification
   - `pkg/doclens/chat/chat.go:114`: `HasBlockMetadata` for block identification
   - `pkg/doclens/chat/docs_suggestions_prompt_middleware.go:75`: `HasBlockMetadata` for block identification
   - `pkg/webchat/system_prompt_middleware.go:44`: `HasBlockMetadata` for block identification
   - `pkg/webchat/moments_global_prompt_middleware.go:50`: `HasBlockMetadata` for block identification
   - `pkg/webchat/ordering_middleware.go:65`: Reads `b.Metadata[turns.BlockMetadataKey(metadata.MetadataKeySection)]` (direct access)

2. **Write operations:**
   - `pkg/inference/middleware/conversation_compression_middleware.go:114`: `turns.SetBlockMetadata` call (uses helper)
   - `pkg/inference/middleware/compression/block_compressor.go:297`: Direct write `block.Metadata[turns.BlockMetadataKey(...)] = value`

---

### Pinocchio (`pinocchio/`)

**Access patterns found:**

#### Turn.Data Access (Pinocchio)

1. **Initialization:**
   - `pkg/middlewares/agentmode/middleware.go:81`: `t.Data = map[turns.TurnDataKey]interface{}{}` (middleware init)

2. **Write operations:**
   - `pkg/middlewares/agentmode/middleware.go:87,95,157,179`: Reads/writes `DataKeyAgentMode`, `DataKeyAgentModeAllowedTools`
   - `cmd/agents/simple-chat-agent/main.go`: Turn creation with Data initialization
   - `cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go`: Tool loop that may access Turn.Data

3. **Read operations:**
   - `pkg/middlewares/agentmode/middleware.go:87`: Reads `t.Data[DataKeyAgentMode].(string)` (type assertion)

---

### Bobatea (`bobatea/`)

**Access patterns found:**

- **No direct Turn.Data/Metadata access found** in bobatea codebase
- Bobatea appears to be a TUI library (bubbletea components) that doesn't interact with Turn structures directly

---

## Helper Functions Analysis

### Current Helpers (to be updated)

1. **`SetTurnMetadata(t *Turn, key TurnMetadataKey, value interface{})`**
   - **Location:** `geppetto/pkg/turns/types.go:147`
   - **Current behavior:** Initializes map if nil, writes `t.Metadata[key] = value`
   - **Migration:** Replace with `t.Metadata.Set(key, value)` (wrapper API)

2. **`SetBlockMetadata(b *Block, key BlockMetadataKey, value interface{})`**
   - **Location:** `geppetto/pkg/turns/types.go:155`
   - **Current behavior:** Initializes map if nil, writes `b.Metadata[key] = value`
   - **Migration:** Replace with `b.Metadata.Set(key, value)` (wrapper API)

3. **`HasBlockMetadata(b Block, key BlockMetadataKey, value string) bool`**
   - **Location:** `geppetto/pkg/turns/helpers_blocks.go:111`
   - **Current behavior:** Reads `b.Metadata[key]`, type-asserts to string, compares
   - **Migration:** Use `b.Metadata.Get(key)` with typed key, compare result

4. **`RemoveBlocksByMetadata(t *Turn, key BlockMetadataKey, values ...string)`**
   - **Location:** `geppetto/pkg/turns/helpers_blocks.go:127`
   - **Current behavior:** Iterates blocks, reads `b.Metadata[key]`, compares values
   - **Migration:** Use `b.Metadata.Get(key)` with typed key, compare results

5. **`WithBlockMetadata(b Block, kvs map[BlockMetadataKey]interface{}) Block`**
   - **Location:** `geppetto/pkg/turns/helpers_blocks.go:94`
   - **Current behavior:** Clones metadata map, merges kvs
   - **Migration:** Use wrapper API `Set` methods on cloned wrapper

---

## Serialization Points

### YAML Serialization

1. **Marshal:**
   - Handled by `gopkg.in/yaml.v3` via struct tags (`yaml:"data,omitempty"`)
   - **Migration:** Implement `MarshalYAML()` on `Data`, `Metadata`, `BlockMetadata` wrappers
   - **Design doc specifies:** Convert `map[TurnDataKey]any` to `map[string]any` for YAML (keys are strings)

2. **Unmarshal:**
   - Handled by `gopkg.in/yaml.v3` via struct tags
   - **Migration:** Implement `UnmarshalYAML(*yaml.Node)` on wrapper types
   - **Design doc specifies:** Parse string keys, validate format `"namespace.slug@vN"`, convert to `TurnDataKey`

3. **Normalization:**
   - `geppetto/pkg/turns/serde/serde.go:NormalizeTurn` initializes nil maps
   - **Migration:** Wrapper API handles nil initialization internally, normalization may become no-op or simplified

---

## Linter Integration

### Current Linter (`turnsdatalint`)

**Location:** `geppetto/pkg/analysis/turnsdatalint/analyzer.go`

**Current rules:**
- Prevents raw string literals: `t.Data["foo"]` ❌
- Allows typed conversions: `t.Data[turns.TurnDataKey("foo")]` ⚠️ (loophole)
- Allows const keys: `t.Data[turns.DataKeyToolConfig]` ✅

**Enhancement needed:**
1. **Ban ad-hoc key construction:** `NewTurnDataKey(...)` or `TurnDataKey("...")` only allowed in `*/keys.go` or `*/turnkeys/*.go`
2. **Enforce canonical keys:** Key expressions must use `K[T](namespaceConst, valueConst, version)` with const namespace/value keys
3. **Enforce namespace/value consts:** Namespace and value must be string consts (not variables or literals)
4. **Enforce key format:** Resulting key must match `^[a-z]+\.[a-z_]+@v\d+$` format
5. **Deprecation warnings:** Parse `// Deprecated:` comments, warn at usage sites
6. **Ban direct map access:** After migration, ban `t.Data[key]` entirely (must use wrapper API)

---

## Special Cases

### Compression Middleware

**Location:** `moments/backend/pkg/inference/middleware/compression/turn_data_compressor.go`

**Current pattern:**
```go
func (tdc *TurnDataCompressor) Compress(ctx context.Context, data map[string]any) TurnDataCompressionOutcome
```

**Problem:** Takes `map[string]any` (not `Turn.Data`), iterates with `for key := range data`, modifies in-place.

**Migration options (from design doc):**
1. **Option A:** Use `Range` method: `turn.Data.Range(func(key TurnDataKey, value any) bool { ... })`
2. **Option B:** Work with known typed keys: `CompressKnownKeys(ctx, turn *Turn, keys []Key[any])`

**Design doc recommends Option B** (compression works with specific typed keys rather than generic iteration).

**Call site:** `moments/backend/pkg/inference/middleware/conversation_compression_middleware.go` converts `Turn.Data` to `map[string]any` before calling `Compress`. This conversion must be removed.

---

## Migration Complexity Assessment

### High Complexity (Requires Refactoring)

1. **Compression middleware** (`turn_data_compressor.go`): Needs API redesign
2. **Serialization** (`serde.go`): Must implement `MarshalYAML`/`UnmarshalYAML` on wrappers
3. **Linter** (`turnsdatalint`): Needs new rules for wrapper API enforcement

### Medium Complexity (Pattern Updates)

1. **Helper functions** (`SetTurnMetadata`, `SetBlockMetadata`, `HasBlockMetadata`): Update to use wrapper API internally
2. **Middleware initialization**: Replace `t.Data = map[...]any{}` with wrapper initialization (may be automatic)
3. **Type assertions**: Replace `t.Data[key].(string)` with `t.Data.Get(key)` returning typed value

### Low Complexity (Direct Replacements)

1. **Direct writes**: `t.Data[key] = value` → `t.Data.Set(key, value)`
2. **Direct reads**: `value, ok := t.Data[key]` → `value, ok, err := t.Data.Get(key)`
3. **Nil checks**: `if t.Data == nil` → handled by wrapper (no-op check)

---

## Key Migration Patterns

### Pattern 1: Write with Default

**Before:**
```go
if t.Data == nil {
    t.Data = map[turns.TurnDataKey]any{}
}
t.Data[turnkeys.ThinkingMode] = modeName
```

**After:**
```go
if err := t.Data.Set(turnkeys.KeyThinkingMode, modeName); err != nil {
    return nil, fmt.Errorf("set thinking mode: %w", err)
}
```

### Pattern 2: Read with Type Assertion

**Before:**
```go
modeName, _ := t.Data[turnkeys.ThinkingMode].(string)
if modeName == "" {
    modeName = ModeExploring
    t.Data[turnkeys.ThinkingMode] = modeName
}
```

**After:**
```go
mode, ok, err := t.Data.Get(turnkeys.KeyThinkingMode)
if err != nil {
    return nil, fmt.Errorf("decode error: %w", err)
}
if !ok || mode == "" {
    mode = ModeExploring
    if err := t.Data.Set(turnkeys.KeyThinkingMode, mode); err != nil {
        return nil, fmt.Errorf("set thinking mode: %w", err)
    }
}
```

### Pattern 3: Helper Function Update

**Before:**
```go
func SetTurnMetadata(t *Turn, key TurnMetadataKey, value interface{}) {
    if t.Metadata == nil {
        t.Metadata = make(map[TurnMetadataKey]interface{})
    }
    t.Metadata[key] = value
}
```

**After:**
```go
func SetTurnMetadata(t *Turn, key Key[T], value T) error {
    return t.Metadata.Set(key, value)
}
```

---

## Test Coverage

### Existing Tests

1. **`geppetto/pkg/turns/serde/serde_test.go`:** Tests YAML round-trip with Metadata
2. **`moments/backend/pkg/inference/middleware/compression/turn_data_compressor_test.go`:** Tests compression on `map[string]any`
3. **`moments/backend/pkg/memory/context_middleware_test.go`:** Tests Block.Metadata access via `HasBlockMetadata`

### Tests Needing Updates

- All tests that directly access `t.Data[key]` or `t.Metadata[key]`
- Compression tests that use `map[string]any` (need refactoring)
- Serialization tests (verify wrapper marshal/unmarshal)

---

## Summary Statistics

| Repository | Turn.Data Sites | Turn.Metadata Sites | Block.Metadata Sites | Total |
|------------|----------------|---------------------|---------------------|-------|
| **geppetto** | ~5 | ~3 | ~8 | ~16 |
| **moments** | ~40 | ~15 | ~60 | ~115 |
| **pinocchio** | ~5 | ~0 | ~0 | ~5 |
| **bobatea** | 0 | 0 | 0 | 0 |
| **Total** | ~50 | ~18 | ~68 | **~136** |

**Note:** Counts are approximate based on grep results. Actual counts may vary due to:
- Helper function calls (counted once, but used many times)
- Test files (included in counts)
- Documentation/examples (excluded from counts)

---

## Next Steps

1. **Implement wrapper types** (`Data`, `Metadata`, `BlockMetadata`) in `geppetto/pkg/turns/`
2. **Implement API methods** (`Get`, `Set`, `Range`, `Delete`, `Len`)
3. **Implement YAML marshal/unmarshal** for wrappers
4. **Convert canonical keys** to typed keys with namespace/version
5. **Enhance linter** (ban ad-hoc keys, enforce format, deprecation warnings)
6. **Migrate helper functions** to use wrapper API
7. **Migrate middleware** (start with high-traffic ones: `current_user_middleware`, `agentmode`)
8. **Refactor compression middleware** to work with typed API
9. **Update tests** to use new API and verify error handling

---

## Related Documents

- **Design doc:** `geppetto/ttmp/2025/12/22/001-REVIEW-TYPED-DATA-ACCESS--review-typed-turn-data-metadata-design-debate-synthesis/design-doc/03-final-design-typed-turn-data-metadata-accessors.md`
- **Previous analysis:** `geppetto/ttmp/2025/12/17/001-TYPED-ACCESSOR-TURN-DATA--typed-accessor-for-turn-data-opaque-wrapper/analysis/01-opaque-turn-data-typed-get-t-accessors.md`
