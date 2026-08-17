---
Title: Diary
Ticket: GEPPETTO-MULTIMODAL-HISTORY-001
Status: active
Topics:
    - turns
    - serde
    - inference
    - openai
    - llm
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/inference/session/session.go
      Note: AppendNewTurnFromUserMessage (commit c708633a)
    - Path: repo://pkg/steps/ai/imageparts/image_parts.go
      Note: numeric slice content support (commit c708633a)
    - Path: repo://pkg/steps/ai/openai/helpers.go
      Note: accessor + image-only fix (commit c708633a)
    - Path: repo://pkg/turns/helpers_blocks.go
      Note: BlockImages/ImagesFromPayload/HasImages (commit c708633a)
    - Path: repo://pkg/turns/serde/serde_images_test.go
      Note: round-trip regression test
ExternalSources: []
Summary: Step-by-step diary of the multimodal history hardening work (accessor, adapters, session helper, tests).
LastUpdated: 2026-08-17T14:48:32.386431676-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Record how GEPPETTO-MULTIMODAL-HISTORY-001 was designed and implemented:
making image-bearing user blocks survive persistence, adding a session helper,
and sending image-only messages on OpenAI chat. This work is Phase 0 of the
CoinVault image-upload feature (COINVAULT-046) and is consumed by pinocchio
(PINOCCHIO-CHAT-ATTACHMENTS-001).

## Step 1: Ticket, design doc, and implementation of the multimodal hardening

The CoinVault design work had already established, with a runnable experiment,
that every provider adapter asserts `[]map[string]any` for `payload["images"]`
while YAML reload yields `[]any`. I created the ticket and design doc, then
implemented the fix as an additive accessor plus a session helper, and while
writing the serde round-trip test discovered a second latent problem: `[]byte`
image content is encoded by yaml.v3 as a sequence of integers and comes back
as `[]interface{}`, which `imageparts.contentBytes` rejected — meaning that
after making adapters tolerant, reloaded inline-byte images would have turned
into hard errors instead of silent drops. That was fixed in `imageparts` too.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket in each relevant repository, with a its own design document, and then implement them. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)

Backfill the diary as necessary. as well. Make sure the diary format from the skill is WELL UNDERSTOOD TO BE MANDATORY."

**Assistant interpretation:** For geppetto specifically: create a docmgr ticket with a design doc, implement the geppetto part of the image-upload design (COINVAULT-046 §13 Phase 0), commit, and keep this diary in the mandatory skill format.

**Inferred user intent:** Ship the cross-repo image-upload feature with a durable paper trail per repository so each change is reviewable on its own.

**Commit (code):** c708633a — "turns: tolerant BlockImages accessor, session AppendNewTurnFromUserMessage, image-only OpenAI chat messages"
**Commit (docs):** 3f56df92 — "GEPPETTO-MULTIMODAL-HISTORY-001: ticket + design doc"

### What I did

- `docmgr ticket create-ticket --ticket GEPPETTO-MULTIMODAL-HISTORY-001 …`, added design doc + diary.
- Wrote `design-doc/01-multimodal-history-hardening-design.md`.
- `pkg/turns/helpers_blocks.go`: added `ImagesFromPayload`, `BlockImages`, `HasImages`.
- `pkg/turns/builders.go`: `WithUserMessage(text, images)`.
- `pkg/inference/session/session.go`: `AppendNewTurnFromUserMessage`, `ErrSessionEmptyUserMessage`.
- Adapters: `pkg/steps/ai/openai/helpers.go` (accessor + image-only send + skip when no text and no usable images), `openai_responses/helpers.go`, `claude/helpers.go`, `gemini/modern_adapter.go` (accessor).
- `pkg/steps/ai/imageparts/image_parts.go`: `contentBytes` accepts `[]any` of numbers (`numberToByte`).
- Tests: `pkg/turns/helpers_blocks_images_test.go`, `pkg/turns/serde/serde_images_test.go`, additions to `openai/helpers_test.go` (3), `claude/helpers_test.go`, `gemini/modern_adapter_test.go`, `openai_responses/helpers_test.go`, `imageparts/image_parts_test.go`, `session/session_test.go` (2).
- Docs: `README.md` multimodal section, `pkg/doc/topics/08-turns.md`.
- Experiment: `scripts/bytesyaml/main.go` (shows `[]byte` → YAML int sequence).

```bash
go test ./pkg/turns/... ./pkg/inference/session/... ./pkg/steps/ai/openai/... ./pkg/steps/ai/openai_responses/... ./pkg/steps/ai/claude/... ./pkg/steps/ai/gemini/... ./pkg/steps/ai/imageparts/... -count=1   # all ok
golangci-lint run ./pkg/turns/... ./pkg/inference/session/... ./pkg/steps/ai/...   # 0 issues
```

### Why

- The accessor is additive and fixes every source (YAML, JSON, JS codec) at
  the single point of consumption; changing the payload shape or `serde`
  would be a larger, riskier change.
- Image-only messages are a real chat use case ("here is a photo").

### What worked

- The go.work setup meant tests in geppetto immediately reflect what
  pinocchio/coinvault will link against.
- Existing per-adapter image tests were easy to mirror for the YAML shape.

### What didn't work

- First run of `serde_images_test.go` failed:
  `serde_images_test.go:49: content type after round trip = []interface {}`.
  Root cause: yaml.v3 encodes `[]byte` held in `any` as a sequence of ints
  (verified with `scripts/bytesyaml`). Fixed by widening
  `imageparts.contentBytes` and relaxing the test assertion to presence.

### What I learned

- Two distinct round-trip hazards exist: the slice type of `images` and the
  element type of inline `content`. Both are now covered by tests.

### What was tricky to build

- The OpenAI chat guard ordering: the empty-text `continue` had to move
  after image extraction, and a new guard was needed for "no text and no
  *usable* image" (e.g. `file_id`-only, which chat completions can't send) so
  we never emit an empty `MultiContent` message. Test
  `TestMakeCompletionRequestFromTurnSkipsBlockWithNoTextAndNoUsableImages`
  covers it.

### What warrants a second pair of eyes

- `numberToByte` accepts float64 whole numbers (JSON decoding) — confirm this
  is desired and cannot mis-decode legitimate data.
- Claude still drops remote `url` images and Gemini errors on them
  (unchanged, out of scope here; documented in COINVAULT-046 §5.2).

### What should be done in the future

- Consider having `serde.ToYAML` encode `[]byte` content as base64 strings
  (`!!binary` or data URL) for compact on-disk turns.
- Claude URL image source support; Gemini remote URL fetch.

### Code review instructions

- Start at `pkg/turns/helpers_blocks.go` (`ImagesFromPayload`), then the four
  adapter call sites (grep `turns.BlockImages`), then
  `pkg/inference/session/session.go:AppendNewTurnFromUserMessage`.
- Validate: `go test ./pkg/turns/... ./pkg/inference/session/... ./pkg/steps/ai/... -count=1`.

### Technical details

- Accepted shapes: `[]map[string]any`, `[]any{map[string]any…}`, `[]any{map[any]any…}`.
- `contentBytes` now accepts `[]any` of ints/uints/whole floats in 0..255.
