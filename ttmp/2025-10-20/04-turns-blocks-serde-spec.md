## Spec: Serialization/Deserialization (Serde) for `turns.Turn` and `turns.Block`

### Purpose & Scope
- Define a stable, human-readable serialization for `turns.Turn` and `turns.Block` suitable for:
  - Test fixtures and golden files
  - E2E/runner inputs and outputs
  - Debugging and offline analysis
- Primary format: YAML (JSON supported as a byproduct of the same tags/logic)
- Non-goals: UI formatting, events serialization (covered elsewhere)

### Goals
- Human-readable: string enums, clear field names
- Round-trippable: load → save without loss (except intentional redactions)
- Backward compatible: tolerate unknown fields/kinds; default behaviors well-defined
- Deterministic: stable field ordering when writing to ease diffing

### Data Model Overview (current types)
- `turns.Block` fields: `ID`, `TurnID`, `Kind`, `Role`, `Payload`, `Metadata`
- `turns.Turn` fields: `ID`, `RunID`, `Blocks []Block`, `Metadata`, `Data`

### Enum: `BlockKind` string mapping
- YAML/JSON MUST use string values (never ints):
  - `user` → BlockKindUser
  - `llm_text` → BlockKindLLMText
  - `tool_call` → BlockKindToolCall
  - `tool_use` → BlockKindToolUse
  - `system` → BlockKindSystem
  - `reasoning` → BlockKindReasoning
  - `other` → BlockKindOther

Implementation:
- Add `(BlockKind) MarshalYAML/UnmarshalYAML` and `(BlockKind) MarshalJSON/UnmarshalJSON` that convert to/from the exact strings above. Unknown string → `BlockKindOther` and preserve raw value in metadata (see Guardrails).

### Field naming (serde)
- Prefer snake_case for YAML/JSON keys:
  - Turn: `id`, `run_id`, `blocks`, `metadata`, `data`
  - Block: `id`, `turn_id`, `kind`, `role`, `payload`, `metadata`
- Add struct tags to `turns.Turn` and `turns.Block` to enforce these names:
  - `json:"id,omitempty" yaml:"id,omitempty"` etc.

### Role semantics
- Required/expected by kind:
  - `system` → role MUST be `system`
  - `user` → role MUST be `user`
  - `llm_text` → role SHOULD be `assistant` (serde will synthesize `assistant` if empty)
  - `reasoning` → role ignored/omitted
  - `tool_call`/`tool_use` → role ignored/omitted

### Payload contract
- Reserved payload keys (documented for clarity, but serde does not enforce provider-specific schemas):
  - `text` (string): message body for user/system/assistant blocks
  - `images` ([]any): optional multimodal metadata on user/system
  - `id` (string): tool or provider call id (for tool_call/tool_use)
  - `name` (string): tool/function name (tool_call)
  - `args` (any or string): tool input (tool_call)
  - `result` (any or string): tool execution result (tool_use)
  - `encrypted_content` (string): provider reasoning ciphertext (reasoning)
  - `item_id` (string): provider-native output item id (e.g., `fc_...`)

Validation guidelines (lightweight):
- `tool_call`: expect `id`, `name`, and `args` present
- `tool_use`: expect `id` and `result` present
- `reasoning`: `encrypted_content` optional
- Serde should not fail if keys are missing; leave strict validation to higher layers/tests

### Redaction & Security
- `encrypted_content` may be sensitive. Serde supports:
  - Pass-through (default)
  - Redaction: replace with a compact placeholder (e.g., `gAAAAA-****-SUFFIX`) preserving length/shape hints when possible
- Redaction controls:
  - Options object: `RedactEncryptedContent bool`
  - Optional file-level note: `metadata.redacted: true`

### Determinism & Ordering
- Maintain block order as-is.
- When marshaling maps for `payload`/`metadata`, write keys sorted alphabetically to produce stable diffs. Implement via a helper encoder or pre-sorted map copy.

### Unknowns & Guardrails
- Unknown `kind` string → map to `other`; store raw kind in `metadata["serde.kind_raw"]` to allow round-trip with minimal loss.
- Missing `role` on `llm_text` → set `assistant` when loading.
- Missing `payload` → treat as empty map.
- Extra/unrecognized fields → preserved when loading (if under `payload`/`metadata`), ignored at top-level.

### Versioning
- Optional top-level doc version:
  - For a single Turn file: `version: 1` at root (above `id`, `blocks`)
  - For a multi-turn suite: `version: 1` under `turns:` collection root
- Default when absent: version 1

### YAML examples

- Plain chat
```yaml
version: 1
id: turn_001
run_id: run_abc
blocks:
  - kind: system
    role: system
    payload: { text: "You are a LLM." }
  - kind: user
    role: user
    payload: { text: "Say hi." }
metadata: {}
data: {}
```

- Reasoning + assistant output
```yaml
version: 1
id: turn_reasoning
blocks:
  - kind: reasoning
    id: rs_123
    payload:
      encrypted_content: gAAAAA...
  - kind: llm_text
    role: assistant
    payload:
      text: "Hello!"
```

- Tool call and result
```yaml
version: 1
id: turn_tools
blocks:
  - kind: tool_call
    payload:
      id: fc_1
      name: search
      args: { q: "golang" }
  - kind: tool_use
    payload:
      id: fc_1
      result: { hits: 10 }
```

### JSON example (identical fields)
```json
{
  "version": 1,
  "id": "turn_001",
  "blocks": [
    { "kind": "system", "role": "system", "payload": { "text": "You are a LLM." } },
    { "kind": "user",   "role": "user",   "payload": { "text": "Say hi." } }
  ],
  "metadata": {},
  "data": {}
}
```

### Public API (helpers)
- Package: `github.com/go-go-golems/geppetto/pkg/turns/serde` (new)
- Types:
  - `type Options struct { RedactEncryptedContent bool; OmitData bool }`
- Functions:
  - `func SaveTurnYAML(path string, t *turns.Turn, opt Options) error`
  - `func LoadTurnYAML(path string) (*turns.Turn, error)`
  - `func ToYAML(t *turns.Turn, opt Options) ([]byte, error)`
  - `func FromYAML(b []byte) (*turns.Turn, error)`
  - `func SaveJSON/LoadJSON` equivalents
  - `func NormalizeTurn(t *turns.Turn)` applies serde defaults (e.g., role synthesis)

Implementation notes:
- Add YAML/JSON struct tags to `turns.Block` and `turns.Turn` (snake_case names)
- Implement kind enum serializers on `BlockKind`
- Implement redaction helper `RedactEncryptedContent(string) string`
- Sorting: when encoding maps, pre-build a sorted key slice and emit in order

### Validation (optional, best-effort)
- Provide `ValidateTurn(t *turns.Turn) error` with checks:
  - Each block has a known `kind` (post-parse, unknown becomes `other`)
  - `llm_text` blocks have `role==assistant`
  - `tool_call` has `id`, `name` present; `tool_use` has `id` present
  - No fatal hard errors for missing fields unless `strict` mode requested

### Round-trip tests (golden)
- Unit suite:
  - Load YAML → Turn → Save YAML: compare semantic equality (normalized object match), and optionally byte-equality when canonicalized
  - JSON parity tests
  - Redaction on/off cases
- Property tests (rapid):
  - Generate random blocks with reserved/unknown keys; ensure FromYAML(ToYAML(x)) preserves semantics

### Runner integration
- Runner loads `turns.Turn` YAML fixtures → runs engine → writes out:
  - `request.json` (exact request to provider)
  - `final_turn.yaml` (post-inference blocks)
  - `events.ndjson` (captured event stream)
  - VCR cassette (optional)
- Filenames share a base (e.g., `case1.*`) to correlate cassette/turn/events

### Backward/Forward Compatibility
- New block kinds: serde accepts unknown strings and maps them to `other`, storing original in `metadata["serde.kind_raw"]`
- Additional fields in `payload`: preserved
- Version field reserved for future breaking changes

### Implementation Tasks
- [ ] Add YAML/JSON tags to `turns.Block` and `turns.Turn` (snake_case)
- [ ] Implement `BlockKind` YAML/JSON marshal/unmarshal (string enums)
- [ ] Add `serde` package with (To|From)(YAML|JSON), Save/Load helpers, Options
- [ ] Implement redaction helpers and normalization
- [ ] Add unit tests: golden round-trips, redaction, unknown-kind handling
- [ ] Add property tests (rapid) for round-trip semantics
- [ ] Wire into E2E runner (optional follow-up): load/save fixtures alongside VCR cassettes

### Notes
- Serde is intentionally permissive. Strict validation belongs in business logic or dedicated validators.
- Keep comments out of emitted YAML to retain determinism; rely on docs/spec for meaning.


