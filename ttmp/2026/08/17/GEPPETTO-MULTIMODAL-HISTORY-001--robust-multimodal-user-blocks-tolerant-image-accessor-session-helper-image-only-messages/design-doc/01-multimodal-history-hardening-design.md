---
Title: Multimodal history hardening design
Ticket: GEPPETTO-MULTIMODAL-HISTORY-001
Status: active
Topics:
    - turns
    - serde
    - inference
    - openai
    - llm
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Make multimodal user blocks survive YAML persistence, add a session helper for text+images, and send image-only user messages on OpenAI chat.
LastUpdated: 2026-08-17T14:48:31.896106813-04:00
WhatFor: ""
WhenToUse: ""
---

# Multimodal history hardening design

## Executive Summary

Geppetto already supports user blocks with images (`turns.NewUserMultimodalBlock`,
`turns.PayloadKeyImages`, `pkg/steps/ai/imageparts`, and image handling in the
OpenAI chat, OpenAI Responses, Claude and Gemini adapters). Three gaps stop a
chat application from relying on it across turns:

1. **Persistence round-trip drops images.** All four adapters assert
   `b.Payload[turns.PayloadKeyImages].([]map[string]any)`. After
   `serde.ToYAML` → `serde.FromYAML` the value is `[]interface{}` (verified with
   a small program in the CoinVault ticket COINVAULT-046), so the assertion
   fails and images are silently omitted from the provider request when a
   session reloads its history from a turn store.
2. **No session-level helper for text + images.** `Session.AppendNewTurnFromUserPrompt`
   hardcodes `NewUserTextBlock`; callers wanting images must hand-build turns
   and lose the "clone latest turn" history semantics.
3. **OpenAI chat drops image-only user blocks.** `MakeCompletionRequestFromTurn`
   skips any block whose text is empty before it looks at images.

This ticket adds `turns.BlockImages(b)` (a tolerant accessor), switches the
four adapters to it, adds `Session.AppendNewTurnFromUserMessage(text, images)`
and `TurnBuilder.WithUserMessage`, fixes the OpenAI chat image-only case, and
adds regression tests for the YAML round-trip.

Downstream consumer: pinocchio `chatapp` (ticket PINOCCHIO-CHAT-ATTACHMENTS-001)
and CoinVault (COINVAULT-046) build on these helpers.

## Problem Statement

Evidence (paths relative to this repo):

- `pkg/turns/helpers_blocks.go:24-40` — `NewUserMultimodalBlock` stores
  `[]map[string]any` under `PayloadKeyImages`.
- `pkg/steps/ai/openai/helpers.go:216-219` — `if text == "" { continue }`
  before the image check at `:239`.
- `pkg/steps/ai/openai/helpers.go:239`, `pkg/steps/ai/openai_responses/helpers.go:604`,
  `pkg/steps/ai/claude/helpers.go:238`, `pkg/steps/ai/gemini/modern_adapter.go:326`
  — the strict type assertion.
- `pkg/turns/serde/serde.go` — YAML (de)serialization used by every turn
  store (`pkg/js/modules/geppetto/provider/sqlite_turn_store.go`, pinocchio
  `chatstore`).
- `pkg/inference/session/session.go:59-103` — `AppendNewTurnFromUserPrompt(s)`.

Round-trip experiment output:

```
before round-trip: type=[]map[string]interface {} assertion_ok=true
after round-trip:  type=[]interface {}            assertion_ok=false
```

## Proposed Solution

### 1. `turns.BlockImages`

```go
// BlockImages returns the images attached to a block as []map[string]any.
// It accepts the shape produced by NewUserMultimodalBlock ([]map[string]any),
// the shape produced by YAML/JSON decoding ([]any of map[string]any) and
// yaml.v2-style maps ([]any of map[any]any). Non-map entries are skipped.
func BlockImages(b Block) []map[string]any
```

Also `HasImages(b Block) bool` for readability.

### 2. Adapters use `BlockImages`

Replace the four assertions with `imgs := turns.BlockImages(b)` (Responses:
`turns.BlockImages(turns.Block{Payload: payload})` or a small payload-level
variant `turns.ImagesFromPayload(payload)`; we add both — `BlockImages` calls
`ImagesFromPayload`).

### 3. OpenAI chat: image-only messages

Move the empty-text guard after image extraction: skip a block only when it
has neither text nor images; when building `MultiContent`, only add the text
part if text is non-empty.

### 4. Session helper and builder

```go
// AppendNewTurnFromUserMessage clones the latest turn (like AppendNewTurnFromUserPrompt)
// and appends one user block carrying text and images. At least one of them must be non-empty.
func (s *Session) AppendNewTurnFromUserMessage(text string, images []map[string]any) (*turns.Turn, error)

func (tb *TurnBuilder) WithUserMessage(text string, images []map[string]any) *TurnBuilder
```

New sentinel: `ErrSessionEmptyUserMessage`.

## Design Decisions

- **Accessor, not serde normalization.** Normalizing in `serde.NormalizeTurn`
  would fix YAML but not JS/JSON-built payloads; the accessor fixes every
  source at the single point of consumption. Both could coexist; we start
  with the accessor because it is strictly additive.
- **Keep the payload shape unchanged.** No new block kind, no typed struct in
  the payload; the documented map keys stay the contract.
- **No behavior change for text-only blocks.**

## Alternatives Considered

- Typed `Images []ImageRef` field on `Block` — larger API change, breaks the
  free-form payload model and JS codec.
- Coercing in every turn store — duplicated logic, misses in-memory sources.

## Implementation Plan

1. `pkg/turns/helpers_blocks.go`: `ImagesFromPayload`, `BlockImages`, `HasImages`.
2. `pkg/turns/builders.go`: `WithUserMessage`.
3. `pkg/inference/session/session.go`: `AppendNewTurnFromUserMessage`, sentinel.
4. Adapters: openai (incl. image-only fix), openai_responses, claude, gemini modern.
5. Tests: `pkg/turns/helpers_blocks_test.go` (shapes), `pkg/turns/serde/serde_test.go`
   (round-trip + BlockImages), `pkg/steps/ai/openai/helpers_test.go`
   (image-only, YAML-shaped images), responses/claude/gemini YAML-shaped tests,
   session helper test.
6. Docs: README multimodal section, `pkg/doc/topics/08-turns.md`.

## Open Questions

- Should `serde.NormalizeTurn` also coerce `images` back to `[]map[string]any`
  so `Turn` values are canonical after load? Deferred; the accessor suffices.

## References

- CoinVault ticket COINVAULT-046 (design + experiment `scripts/imgroundtrip`).
- geppetto ticket `ttmp/2026/06/05/...geppetto-llm-proxy-image-input...` (introduced `imageparts`).
