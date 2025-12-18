---
Title: Turn and Block Serialization (YAML)
Slug: turns-blocks-serialization
Short: Human-readable YAML serialization for turns and blocks used across tests, runners, and analysis.
Topics:
- turns
- blocks
- serialization
- yaml
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Turn and Block Serialization (YAML)

## Overview

The turns/blocks YAML format provides a stable, human‑readable way to persist chat histories, tool calls, and provider reasoning for tests, E2E runners, and offline analysis. It encodes `turns.Turn` and `turns.Block` with string‑based kinds, snake_case field names, and minimal constraints so fixtures are easy to write, diff, and round‑trip.

This document explains the data model, the YAML shape, recommended conventions, and how to use the serde helpers to load/save turns.

## Data Model

`turns.Turn` contains ordered `blocks` plus optional `metadata` and `data`. A `turns.Block` is a single unit (system/user/assistant text, tool call or result, or provider reasoning).

- Turn fields:
  - `id` (string), `run_id` (string)
  - `blocks` ([]Block, ordered)
  - `metadata` (map), `data` (map)
- Block fields:
  - `id` (string), `turn_id` (string)
  - `kind` (enum), `role` (string), `payload` (map), `metadata` (map)

**Note:** In Go, `metadata` and `data` maps use typed keys (`TurnMetadataKey`, `TurnDataKey`, `BlockMetadataKey`, `RunMetadataKey`) for compile-time safety. When serialized to YAML, these keys appear as strings. The serializer handles the conversion automatically.

## Block kinds (string enums)

Kinds are serialized as lowercase strings. Unknown kinds are accepted and treated as `other`.

- `user` → user text
- `llm_text` → assistant text
- `tool_call` → a function/tool invocation request
- `tool_use` → a function/tool execution result
- `system` → system directive
- `reasoning` → provider reasoning item (e.g., encrypted)
- `other` → fallback for unknown kinds

## YAML field names and roles

All fields are snake_case. Roles are meaningful for text:

- `system` → role must be `system`
- `user` → role must be `user`
- `llm_text` → role is typically `assistant` (defaulted if omitted)
- `tool_call`, `tool_use`, `reasoning` → role is ignored

## Payload conventions

Payload is intentionally flexible. Common keys:

- Text blocks: `text` (string)
- Multimodal: `images` (array)
- Tool call: `id` (string), `name` (string), `args` (any or string)
- Tool use: `id` (string), `result` (any or string)
- Reasoning: `encrypted_content` (string), `item_id` (string, optional)

The serializer is permissive; strict validation lives in higher layers.

## Determinism and ordering

- Blocks remain in their original order.
- Maps are written with stable key order to produce clean diffs.

## Unknowns and defaults

- Unknown `kind` is accepted; the block is treated as `other`.
- Missing `role` on `llm_text` defaults to `assistant` when loading.
- Missing `payload`/`metadata` are treated as empty maps.

## Examples

### Plain chat

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

### Reasoning + assistant output

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

### Tool call and result

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

## Using the serde helpers

The serde helpers live in `github.com/go-go-golems/geppetto/pkg/turns/serde`. They normalize sensible defaults (e.g., assistant role for `llm_text`) and perform YAML load/save using the string‑based `kind` values.

```go
import (
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/turns/serde"
)

// Construct a small turn
t := &turns.Turn{Blocks: []turns.Block{
    turns.NewSystemTextBlock("You are a LLM."),
    turns.NewUserTextBlock("Say hi."),
}}

// Save to YAML
if err := serde.SaveTurnYAML("turn.yaml", t, serde.Options{OmitData: false}); err != nil {
    panic(err)
}

// Load from YAML
loaded, err := serde.LoadTurnYAML("turn.yaml")
if err != nil { panic(err) }

// Or work with bytes directly
yb, _ := serde.ToYAML(t, serde.Options{})
tt, _ := serde.FromYAML(yb)

// Normalize is applied during (de)serialization; can be called explicitly if needed
serde.NormalizeTurn(tt)
```

## Test fixtures and runners

Store curated input turns as YAML under a `testdata/` directory or a dedicated `fixtures/` directory. For E2E recordings, pair each turn YAML with a VCR cassette and capture emitted events for analysis. This produces reproducible, debuggable artifacts without coupling to provider uptime.

## Migration notes

- Existing code using `turns.Block`/`turns.Turn` can adopt YAML serde immediately.
- No redaction is performed by default; reasoning ciphertext (`encrypted_content`) is preserved as provided by the model.
- Unknown kinds remain compatible via the `other` fallback.

## See also

- `glaze help how-to-write-good-documentation-pages`
- `geppetto/pkg/turns/serde`
- Unified streaming events and Responses integration docs


