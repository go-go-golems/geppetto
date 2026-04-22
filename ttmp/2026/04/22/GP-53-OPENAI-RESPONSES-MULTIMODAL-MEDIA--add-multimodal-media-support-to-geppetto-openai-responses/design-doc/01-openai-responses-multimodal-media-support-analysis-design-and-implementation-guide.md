---
Title: OpenAI Responses multimodal media support analysis, design, and implementation guide
Ticket: GP-53-OPENAI-RESPONSES-MULTIMODAL-MEDIA
Status: active
Topics:
    - inference
    - open-responses
    - openai-compatibility
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/doc/topics/06-inference-engines.md
      Note: Background on the OpenAI Responses engine within Geppetto's architecture
    - Path: geppetto/pkg/inference/engine/engine.go
      Note: Defines the provider engine contract the Responses engine implements
    - Path: geppetto/pkg/steps/ai/openai/helpers.go
      Note: Existing OpenAI chat image serialization path used as the closest local precedent
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Contains the missing image-support gap and the request-building logic to patch
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers_test.go
    - Path: geppetto/pkg/steps/ai/openai_responses/token_count.go
      Note: Shows that token counting reuses the same request builder
    - Path: geppetto/pkg/turns/helpers_blocks.go
      Note: Shows the existing multimodal block constructor and current image payload contract
    - Path: geppetto/pkg/turns/types.go
      Note: Defines Turn and Block
    - Path: hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/02-openai-images-and-vision-guide.md
      Note: Official OpenAI guide showing Responses API mixed text+image input examples
    - Path: hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/03-openai-create-model-response-reference.md
      Note: Official schema reference for input_image
    - Path: hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/04-openai-input-token-count-reference.md
      Note: Official reference proving the token-count endpoint accepts image-bearing input items
ExternalSources:
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/02-openai-images-and-vision-guide.md
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/03-openai-create-model-response-reference.md
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/04-openai-input-token-count-reference.md
Summary: Detailed, evidence-backed guide for adding OpenAI Responses multimodal media support to Geppetto, with current-state architecture, gap analysis, phased design, pseudocode, and test guidance aimed at a new engineer onboarding to the codebase.
LastUpdated: 2026-04-22T00:30:00-04:00
WhatFor: Explain exactly what is missing in Geppetto's openai_responses request builder and provide a safe implementation plan for image support first, with broader media extensibility clearly scoped.
WhenToUse: Read this before implementing OpenAI Responses image/media support or reviewing request-shape changes in the Geppetto Responses engine.
---


# OpenAI Responses multimodal media support analysis, design, and implementation guide

## Executive summary

Geppetto already has a working OpenAI Responses engine for text, reasoning, tool calls, structured output, streaming, and input-token counting. The engine lives primarily in `pkg/steps/ai/openai_responses`, and its request-building logic is driven by `buildResponsesRequest(...)` and `buildInputItemsFromTurn(...)` in `pkg/steps/ai/openai_responses/helpers.go`. The missing capability is not at the runner level, not at the profile/bootstrap level, and not at the application level. The missing capability is inside the provider request serializer.

The official OpenAI Responses API supports mixed-content inputs that include `input_text`, `input_image`, and `input_file`. The official docs explicitly say that `input_image.image_url` may be either a fully qualified URL or a base64 `data:` URL, and that `detail` may be `low`, `high`, `auto`, or `original`. The same official docs also show that `/responses/input_tokens` accepts the same input-bearing request shape. Evidence for that exists in the downloaded source bundle at:

- `.../02-openai-images-and-vision-guide.md:104-172`
- `.../03-openai-create-model-response-reference.md:15885-15917`
- `.../04-openai-input-token-count-reference.md:23-119`

By contrast, Geppetto's Responses request serializer currently defines a `responsesContentPart` that only carries text-oriented fields and includes the explicit comment `image/audio not supported in first cut` (`pkg/steps/ai/openai_responses/helpers.go:70-80`). The ordinary message path inside `buildInputItemsFromTurn(...)` only appends `input_text` or `output_text` parts and never inspects `turns.PayloadKeyImages` (`pkg/steps/ai/openai_responses/helpers.go:319-333`). This is the core gap.

The recommended first implementation slice is deliberately narrow and safe:

1. Teach `pkg/steps/ai/openai_responses/helpers.go` to serialize image-bearing user messages into Responses API `input_image` parts.
2. Reuse the existing Geppetto turn payload convention already used by the OpenAI chat path and the Claude path: `PayloadKeyImages` with `media_type` plus either `url` or `content` (`pkg/turns/helpers_blocks.go:24-36`, `pkg/steps/ai/openai/helpers.go:190-220`).
3. Support optional `detail` and opportunistic `file_id` passthrough for the Responses API image object.
4. Add focused regression tests in `helpers_test.go` and `token_count_test.go`.
5. Defer broader file/audio/generalized-content modeling to a follow-up phase unless an immediate product requirement justifies expanding the canonical turn schema.

That approach gives Geppetto correct OpenAI Responses image support with minimal churn, preserves compatibility with the current turn model, and keeps the broader “more media” design discussion honest instead of hiding it inside a rushed implementation patch.

---

## Problem statement and scope

### The concrete problem

A Geppetto-powered application can already construct multimodal user turns using `turns.NewUserMultimodalBlock(...)`, and the regular OpenAI chat-completions engine already knows how to serialize those images into provider request parts. However, the OpenAI Responses engine does not. As a result, applications that select `ai-api-type=openai-responses` can appear to be sending multimodal evidence while actually sending only the text summary portion of the prompt.

That mismatch is dangerous for two reasons:

- It can produce misleadingly good results when the prompt contains strong textual evidence, making it easy to assume screenshots or other images were transmitted even when they were silently dropped.
- It creates cross-engine inconsistency inside Geppetto: the same turn payload means one thing for `openai` and a narrower thing for `openai-responses`.

### What this ticket is in scope to solve

This document scopes the work in two layers.

#### In-scope now

- Explain the Geppetto architecture needed to understand the bug.
- Document the official Responses API contract for image-bearing input.
- Identify the exact missing serialization logic.
- Propose a safe, incremental implementation for image support.
- Cover test strategy, token-count implications, and documentation updates.

#### Potentially in scope later, but not required for the first patch

- Canonical support for provider-neutral file inputs in the turn model.
- Canonical support for audio inputs in the turn model.
- A richer typed content-part abstraction shared across all provider engines.
- Replaying assistant-side multimodal outputs into future requests.

### What is explicitly out of scope for the first implementation slice

- Reworking Geppetto's entire turn/block model.
- Changing profile bootstrap or runtime selection logic.
- Replacing the OpenAI Responses engine with another provider path.
- Building product-specific screenshot or artifact orchestration.

---

## Reader orientation: what Geppetto is and where this code lives

This section is for a new intern. Before changing anything, you need a mental map of the system.

### Geppetto's core abstraction: Turns and Blocks

Geppetto's provider engines do not operate on provider-native JSON directly. They operate on a provider-neutral conversation structure built from `Turn` and `Block`.

Relevant files:

- `pkg/turns/types.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/turns/keys_gen.go`

A `Turn` is an ordered collection of `Block` values plus metadata and turn-scoped data (`pkg/turns/types.go:8-26`). A `Block` is a single atomic item such as:

- a user message,
- an assistant text response,
- a tool call,
- a tool result,
- or a reasoning item.

This matters because provider engines are basically translators:

```text
Turn/Blocks  --->  provider request JSON  --->  provider response JSON  --->  Turn/Blocks
```

If the translator ignores one payload field, the feature effectively does not exist for that provider.

### The engine interface

All providers implement the same simple interface:

- `pkg/inference/engine/engine.go:9-14`

```go
type Engine interface {
    RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error)
}
```

That simplicity is intentional. Provider engines handle API I/O and response parsing. They are not responsible for higher-level product logic.

### Where the OpenAI Responses engine fits

The main Geppetto engine architecture guide explains that the OpenAI Responses engine is selected with `ai-api-type=openai-responses` and is responsible for streaming reasoning summary and tool-call arguments in addition to normal output text (`pkg/doc/topics/06-inference-engines.md:713-723`).

So, for this ticket, the relevant conceptual stack is:

```text
Application / CLI / runner
        |
        v
Geppetto Turn + Block model
        |
        v
OpenAI Responses engine (pkg/steps/ai/openai_responses)
        |
        v
OpenAI /v1/responses and /v1/responses/input_tokens
```

### The most important files for this ticket

#### Request/response engine files

- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/openai_responses/helpers.go`
- `pkg/steps/ai/openai_responses/token_count.go`
- `pkg/steps/ai/openai_responses/helpers_test.go`
- `pkg/steps/ai/openai_responses/token_count_test.go`

#### Comparison / precedent files

- `pkg/steps/ai/openai/helpers.go`
- `pkg/steps/ai/openai/chat_types.go`
- `pkg/steps/ai/claude/helpers.go`
- `pkg/turns/helpers_blocks.go`

#### Background docs

- `pkg/doc/topics/06-inference-engines.md`

---

## Official OpenAI Responses API contract: what the provider actually supports

The implementation must be driven by the provider contract, not by local assumptions.

### Official evidence 1: the vision guide says Responses accepts image input

The downloaded OpenAI vision guide states that images can be supplied to generation requests by:

- fully qualified URL,
- base64-encoded data URL,
- or file ID,

and shows a concrete Responses API example with mixed content in one message (`02-openai-images-and-vision-guide.md:104-172`):

```python
input=[{
    "role": "user",
    "content": [
        {"type": "input_text", "text": "what's in this image?"},
        {
            "type": "input_image",
            "image_url": "https://...",
        },
    ],
}]
```

This is the most important behavioral point: **image inputs are content parts inside a message, not a separate top-level request feature**.

### Official evidence 2: the reference defines `input_image`

The create-response reference defines `ResponseInputImage` with these fields (`03-openai-create-model-response-reference.md:15885-15917`):

- `type: "input_image"`
- `detail: low | high | auto | original`
- `file_id: optional string`
- `image_url: optional string`

It also explicitly says that `image_url` may be a fully qualified URL or a base64 data URL.

This matters because a correct serializer must not only emit the right `type`; it must also preserve either remote URL or inline data-URL transport.

### Official evidence 3: `/responses/input_tokens` accepts the same kinds of input

The input-token-count reference also documents `input_text`, `input_image`, and `input_file` as valid request input items (`04-openai-input-token-count-reference.md:23-119`).

That means any feature added to `buildResponsesRequest(...)` also matters to token counting, because Geppetto reuses the request builder for `/responses/input_tokens`.

### Official evidence 4: real example in the create-response reference

The create-response reference includes a full request example (`03-openai-create-model-response-reference.md:47549-47563`) that sends:

- one `input_text` part,
- one `input_image` part,
- inside a single user message.

That is exactly the shape Geppetto should emit for a user block that contains text plus images.

---

## Current Geppetto architecture: how the Responses engine builds requests today

### Step 1: `RunInference` builds a provider request from a `Turn`

The Responses engine entrypoint is `RunInference(...)` in `pkg/steps/ai/openai_responses/engine.go`. Early in that method, the engine builds the provider request using `buildResponsesRequest(t)` (`engine.go:48-60`).

That means this call chain is the core of the ticket:

```text
RunInference(ctx, turn)
  -> buildResponsesRequest(turn)
       -> buildInputItemsFromTurn(turn)
```

### Step 2: `buildResponsesRequest` stores all user/assistant/tool/reasoning inputs in `req.Input`

In `pkg/steps/ai/openai_responses/helpers.go:113-119`, the engine initializes a `responsesRequest` and sets:

```go
req.Input = buildInputItemsFromTurn(t)
```

Everything else in the request builder is important but secondary for this ticket:

- model selection,
- reasoning settings,
- structured output,
- stop sequences,
- streaming,
- per-turn inference overrides.

The media-support bug is entirely about what lands inside `req.Input`.

### Step 3: the local request models currently stop at text-oriented content parts

The current `responsesContentPart` struct is defined in `helpers.go:70-80` as:

```go
type responsesContentPart struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    CallID string `json:"call_id,omitempty"`
    Name string `json:"name,omitempty"`
    Arguments string `json:"arguments,omitempty"`
    ToolCallID string `json:"tool_call_id,omitempty"`
    Content string `json:"content,omitempty"`
    // image/audio not supported in first cut
}
```

The comment is not just documentation. It accurately describes the current limitation: ordinary message content parts have no image-related fields at all.

### Step 4: ordinary messages only serialize text today

Inside `buildInputItemsFromTurn(...)`, the helper `appendMessage` is responsible for message-shaped items (`helpers.go:319-333`). It:

1. computes the message role,
2. looks for `turns.PayloadKeyText`,
3. emits one `input_text` part for non-assistant roles or one `output_text` part for assistant roles,
4. appends the message only if there is at least one text part.

What it does **not** do:

- inspect `turns.PayloadKeyImages`,
- inspect `url`, `content`, or `media_type`,
- emit `input_image`,
- emit `file_id`,
- emit `detail`.

So a block that carries multimodal payloads loses its image content in the Responses path.

### Step 5: reasoning and tool chains already have careful ordering logic

The rest of `buildInputItemsFromTurn(...)` is not broken. In fact, it is careful and valuable. The function preserves special ordering rules for:

- reasoning blocks,
- assistant follower messages,
- function calls,
- function call outputs.

That logic lives in `helpers.go:365-441` and is covered by existing tests in `helpers_test.go`.

This is an important design constraint: the image patch must fit into the existing message path without destabilizing reasoning or tool-chain ordering.

### Step 6: token counting reuses the same request builder

`pkg/steps/ai/openai_responses/token_count.go:34-52` creates an `Engine`, calls `buildResponsesRequest(t)`, and uses that request shape to call `/responses/input_tokens`.

This is good news. It means:

- we do **not** need a second media serializer for token counting,
- but we **do** need tests that prove the new image-bearing request shape is preserved there too.

---

## Existing internal precedent: the OpenAI chat path already knows how to serialize images

The best local precedent is the normal OpenAI chat-completions engine.

In `pkg/steps/ai/openai/helpers.go:190-220`, Geppetto already checks `turns.PayloadKeyImages` and converts image entries into OpenAI chat `image_url` content parts. That logic already handles:

- remote URL images,
- inline base64 data URLs,
- default `detail: auto`.

Relevant snippet shape:

```go
if imgs, ok := b.Payload[turns.PayloadKeyImages].([]map[string]any); ok && len(imgs) > 0 {
    parts := []ChatMessagePart{{Type: "text", Text: text}}
    for _, img := range imgs {
        mediaType, _ := img["media_type"].(string)
        url, _ := img["url"].(string)
        ...
        if imageURL == "" && base64Content != "" {
            imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Content)
        }
        parts = append(parts, ChatMessagePart{
            Type: "image_url",
            ImageURL: &ChatMessageImageURL{URL: imageURL, Detail: "auto"},
        })
    }
}
```

This tells us three useful things:

1. Geppetto already has a working image payload convention.
2. We do not need to invent a new application-facing image format for the first implementation.
3. The biggest risk is serializer drift between the chat path and the Responses path.

---

## Turn-layer data model: what a multimodal block looks like today

The convenience constructor in `pkg/turns/helpers_blocks.go:24-36` defines the current contract:

```go
// NewUserMultimodalBlock creates a user block with text and optional images.
// images is a slice of maps with keys: "media_type" (string), and either "url" (string) or "content" ([]byte/base64).
func NewUserMultimodalBlock(text string, images []map[string]any) Block {
    payload := map[string]any{PayloadKeyText: text}
    if len(images) > 0 {
        payload[PayloadKeyImages] = images
    }
    ...
}
```

This is enough for a first image patch. It already supports:

- remote URL transport,
- inline bytes or base64 string transport,
- multiple images per message.

### Important limitation of the current turn model

The constructor comment only mentions `media_type`, `url`, and `content`.

It does **not** define canonical payload fields for:

- `detail`,
- `file_id`,
- `input_file`,
- generic media parts,
- audio parts.

That does not block the first implementation. It just means the first implementation should be explicit about what it is standardizing now versus what it is only tolerating opportunistically.

---

## Gap analysis

### Gap 1: OpenAI Responses supports image inputs, but Geppetto Responses does not emit them

Observed state:

- OpenAI docs support `input_image` and `image_url`.
- Geppetto Responses ordinary message serialization is text-only.

Impact:

- multimodal user turns are silently downgraded to text-only requests.

### Gap 2: The turn model already carries images, but only some providers consume them

Observed state:

- `turns.NewUserMultimodalBlock(...)` exists,
- OpenAI chat consumes `PayloadKeyImages`,
- Claude consumes `PayloadKeyImages`,
- OpenAI Responses does not.

Impact:

- provider behavior is inconsistent for the same `Turn` payload.

### Gap 3: There are no Responses multimodal regression tests yet

Observed state:

- `pkg/steps/ai/openai_responses/helpers_test.go` covers plain chat, reasoning, and tool chains.
- No current tests cover `PayloadKeyImages`, `input_image`, or data URLs.

Impact:

- a future patch could be incomplete, and there would be no focused test telling us where it regressed.

### Gap 4: Token-count behavior is easy to forget

Observed state:

- `token_count.go` reuses `buildResponsesRequest(...)`.

Impact:

- if we only test `RunInference`, we can miss a broken `/responses/input_tokens` path.

### Gap 5: Broader file/audio support lacks a canonical turn schema

Observed state:

- current turn helpers define images only.
- official docs also support `input_file`.
- current code comment mentions `image/audio`, but there is no first-class audio turn payload here.

Impact:

- the phrase “more media support” is larger than the first patch.
- the first patch should target image parity first and clearly separate later schema design.

---

## Proposed design

## Design goals

1. **Fix the real bug first**: make image-bearing user blocks serialize correctly for OpenAI Responses.
2. **Minimize churn**: do not redesign all provider-neutral multimodal modeling in the same patch.
3. **Reuse proven conventions**: keep using `PayloadKeyImages` plus `media_type`, `url`, and `content`.
4. **Preserve ordering**: reasoning and tool-call sequencing must remain untouched.
5. **Make token counting consistent automatically**: rely on the shared request builder.
6. **Leave a clean path for future file/media work**: do not paint the architecture into a corner.

## Recommended implementation slice (Phase 1)

### 1. Extend `responsesContentPart` to represent image-bearing input parts

Add fields to `responsesContentPart` that map to the official Responses request shape.

Recommended fields:

```go
type responsesContentPart struct {
    Type string `json:"type"`

    Text string `json:"text,omitempty"`

    ImageURL string `json:"image_url,omitempty"`
    FileID   string `json:"file_id,omitempty"`
    Detail   string `json:"detail,omitempty"`

    CallID    string `json:"call_id,omitempty"`
    Name      string `json:"name,omitempty"`
    Arguments string `json:"arguments,omitempty"`

    ToolCallID string `json:"tool_call_id,omitempty"`
    Content    string `json:"content,omitempty"`
}
```

Why this is enough for Phase 1:

- It supports the official `input_image` fields we actually need now.
- It supports opportunistic `file_id` on image parts because the official schema allows it.
- It does not force us to add `input_file` yet.

### 2. Refactor ordinary message serialization into image-aware part building

Instead of letting `appendMessage` only append text, factor it into two conceptual steps:

1. build text part if text is present,
2. for non-assistant roles, append image parts for every image payload.

Recommended helper shape:

```go
func buildResponsesMessageParts(role string, payload map[string]any) ([]responsesContentPart, error)
```

This helper should:

- emit `input_text` for system/user messages,
- emit `output_text` for assistant messages,
- inspect `PayloadKeyImages` only for non-assistant messages,
- normalize remote URLs and inline content to Responses-compatible `input_image` parts,
- default `detail` to `auto` when present logic needs a default.

### 3. Normalize image maps with a dedicated converter helper

Do **not** keep image conversion as ad-hoc inline code if it grows beyond a few lines. The behavior is specific and easy to get subtly wrong.

Recommended helper:

```go
func responsesImagePartFromMap(img map[string]any) (responsesContentPart, bool, error)
```

Expected behavior:

- If `url` is present and non-empty, use it as `image_url`.
- Else if `content` is present, convert bytes or base64 text into a `data:<media_type>;base64,...` URL.
- If `detail` is present and valid, pass it through.
- If `file_id` is present and non-empty, allow it as an alternative transport.
- If the image entry has no usable transport, return `false` or an error depending on the chosen strictness.

### 4. Be permissive about extra fields; be strict about broken transport

For the first patch, the safest policy is:

- ignore unknown keys in image maps,
- accept `detail` and `file_id` when present,
- skip malformed image entries only if they have neither usable URL nor usable inline content.

I recommend **logging malformed entries in tests through explicit failures** rather than silently hiding them in production logic. In practice, the helper can skip invalid entries if preserving partial requests is preferable, but the tests should pin the expected behavior so the choice is explicit.

### 5. Keep assistant multimodal replay out of Phase 1

Do not attempt to serialize assistant-side image history as part of this ticket.

Reasons:

- the current turn constructor is explicitly user-oriented,
- the official provider output model is not identical to input-image replay,
- product pressure right now is on user-supplied screenshots and similar evidence.

### 6. Let token counting inherit the fix automatically

No special token-count serializer is needed. Because `token_count.go` reuses `buildResponsesRequest(...)`, once the request builder becomes image-aware, the `/responses/input_tokens` request will also become image-aware.

This is a design feature. Keep it that way.

---

## Proposed architecture diagrams

### Current state

```text
User code
  |
  | builds turns.NewUserMultimodalBlock(text, images)
  v
Turn.Block.Payload
  text = "What changed?"
  images = [{media_type, url/content, ...}]
  |
  v
openai_responses.buildInputItemsFromTurn(...)
  |
  | only reads PayloadKeyText
  v
Responses request input
  [{role: "user", content: [{type: "input_text", text: ...}]}]
  |
  v
OpenAI never receives image parts
```

### Proposed Phase 1 state

```text
User code
  |
  | builds turns.NewUserMultimodalBlock(text, images)
  v
Turn.Block.Payload
  text = "What changed?"
  images = [{media_type, url/content, detail?, file_id?}]
  |
  v
openai_responses.buildResponsesMessageParts(...)
  |
  +--> input_text part
  +--> input_image part(s)
  v
Responses request input
  [{
    role: "user",
    content: [
      {type: "input_text", text: ...},
      {type: "input_image", image_url: ..., detail: "auto"}
    ]
  }]
  |
  v
OpenAI receives the actual multimodal request
```

### Phase separation for broader media support

```text
Phase 1: image parity using existing PayloadKeyImages
    |
    +--> small patch, low risk, immediate user value

Phase 2: canonical file/media modeling
    |
    +--> requires new turn payload keys or typed content abstraction

Phase 3: cross-provider shared content-part normalizer
    |
    +--> cleanup / dedup after behavior is proven
```

---

## Pseudocode for the recommended implementation

### Core request-building change

```go
func appendMessage(b turns.Block) {
    role := roleFor(b.Kind)
    parts, err := buildResponsesMessageParts(role, b.Payload)
    if err != nil {
        // choose explicit behavior: return error or skip malformed parts
    }
    if len(parts) > 0 {
        items = append(items, responsesInput{Role: role, Content: parts})
    }
}
```

### Message part builder

```go
func buildResponsesMessageParts(role string, payload map[string]any) ([]responsesContentPart, error) {
    parts := []responsesContentPart{}

    text := extractText(payload)
    if text != "" {
        textType := "input_text"
        if role == "assistant" {
            textType = "output_text"
        }
        parts = append(parts, responsesContentPart{
            Type: textType,
            Text: text,
        })
    }

    if role == "assistant" {
        return parts, nil
    }

    images := extractImageMaps(payload)
    for _, img := range images {
        part, ok, err := responsesImagePartFromMap(img)
        if err != nil {
            return nil, err
        }
        if ok {
            parts = append(parts, part)
        }
    }

    return parts, nil
}
```

### Image normalization helper

```go
func responsesImagePartFromMap(img map[string]any) (responsesContentPart, bool, error) {
    detail := normalizeDetail(stringOr(img["detail"], "auto"))
    fileID := strings.TrimSpace(stringOr(img["file_id"], ""))
    url := strings.TrimSpace(stringOr(img["url"], ""))

    if url != "" {
        return responsesContentPart{
            Type:     "input_image",
            ImageURL: url,
            FileID:   fileID,
            Detail:   detail,
        }, true, nil
    }

    if raw := img["content"]; raw != nil {
        mediaType := strings.TrimSpace(stringOr(img["media_type"], ""))
        base64Content := normalizeBase64(raw)
        if mediaType == "" || base64Content == "" {
            return responsesContentPart{}, false, fmt.Errorf("invalid inline image payload")
        }
        return responsesContentPart{
            Type:     "input_image",
            ImageURL: fmt.Sprintf("data:%s;base64,%s", mediaType, base64Content),
            FileID:   fileID,
            Detail:   detail,
        }, true, nil
    }

    if fileID != "" {
        return responsesContentPart{
            Type:   "input_image",
            FileID: fileID,
            Detail: detail,
        }, true, nil
    }

    return responsesContentPart{}, false, nil
}
```

### Detail normalization

```go
func normalizeDetail(v string) string {
    switch strings.ToLower(strings.TrimSpace(v)) {
    case "low", "high", "auto", "original":
        return strings.ToLower(strings.TrimSpace(v))
    case "":
        return "auto"
    default:
        // safest option: fall back to auto
        return "auto"
    }
}
```

---

## Why this design is preferable to the main alternatives

## Chosen design: incremental image parity with small extensibility hooks

### Benefits

- Fastest path to a correct user-visible feature.
- Reuses current turn payload conventions.
- Low risk to reasoning/tool ordering logic.
- Automatically fixes token counting too.
- Leaves room for richer media modeling later.

### Costs

- Keeps `[]map[string]any` image payloads for now.
- Does not solve canonical file/audio modeling in the same patch.
- Still leaves some duplication risk between chat and Responses helpers.

## Alternative 1: redesign turns into a provider-neutral typed content-part model first

### Why it is attractive

- Cleaner long-term abstraction.
- Could unify text/image/file/audio modeling across providers.
- Could reduce duplicated serializer logic over time.

### Why I do not recommend it first

- Too large for the immediate bug.
- Touches many more packages and tests.
- Increases the odds of introducing new regressions before we even have correct Responses image parity.

## Alternative 2: patch only the product app to use a different engine

### Why it is attractive

- Faster short-term escape hatch for one product.

### Why it is the wrong Geppetto fix

- Leaves the engine bug in place.
- Keeps cross-provider semantics inconsistent.
- Pushes an engine defect into application workarounds.

## Alternative 3: add `input_file` and generic media in the same patch

### Why it is attractive

- Feels more complete.

### Why I would split it

- The turn layer does not yet expose canonical file/media payload keys analogous to `PayloadKeyImages`.
- Official Responses support for `input_file` is real, but Geppetto needs an explicit turn-level representation before we can claim provider-neutral file support.

---

## File-by-file implementation plan

## Phase 0: no-code prep (already done in this ticket)

- Review official docs.
- Audit current serializer behavior.
- Document current-state architecture and risks.

## Phase 1: add image support to the Responses request builder

### File 1: `pkg/steps/ai/openai_responses/helpers.go`

Primary changes:

- extend `responsesContentPart`,
- add image/detail normalization helpers,
- update `appendMessage` to include image parts,
- preserve existing reasoning/tool behavior.

Checklist:

- [ ] Add image-related fields to `responsesContentPart`
- [ ] Add helper for extracting/normalizing image maps
- [ ] Default detail to `auto`
- [ ] Support remote URLs
- [ ] Support inline bytes/base64 content via data URLs
- [ ] Optionally accept `file_id`
- [ ] Do not disturb assistant reasoning/tool-call order logic

### File 2: `pkg/steps/ai/openai_responses/helpers_test.go`

Add tests like:

- `TestBuildInputItemsFromTurn_UserMessageWithImageURL`
- `TestBuildInputItemsFromTurn_UserMessageWithInlineImageBytes`
- `TestBuildInputItemsFromTurn_UserMessageWithInlineImageBase64`
- `TestBuildInputItemsFromTurn_UserMessageWithTextAndMultipleImages`
- `TestBuildInputItemsFromTurn_ImageDetailDefaultsToAuto`
- `TestBuildInputItemsFromTurn_ImageDetailPassesThroughWhenValid`

Important testing principle: keep the existing reasoning/tool-order tests untouched and passing.

### File 3: `pkg/steps/ai/openai_responses/token_count_test.go`

Add a targeted test proving that `CountTurn(...)` sends image-bearing request bodies too.

Example assertion strategy:

- build a `Turn` with `turns.NewUserMultimodalBlock(...)`,
- call the token counter against an `httptest.Server`,
- inspect the raw request body,
- assert that it contains `"type":"input_image"` and the expected transport field.

## Phase 2: small documentation cleanup after code lands

### File 4: `pkg/doc/topics/06-inference-engines.md`

Update the Responses-engine section so it no longer implies a narrower modality surface than the provider actually supports.

Recommended wording to add:

- the Responses engine supports text plus user image inputs when images are present on turn blocks,
- token counting uses the same request-building path.

### File 5: `pkg/turns/helpers_blocks.go` (optional comment-only update)

If the implementation accepts optional `detail` and `file_id` in image maps, update the constructor comment so the informal payload contract is not misleading.

Example revised comment idea:

```go
// images is a slice of maps with keys:
// - media_type (string) for inline content,
// - either url (string), content ([]byte/base64), or file_id (string),
// - optional detail (low|high|auto|original) for providers that support it.
```

## Phase 3: broader media design (follow-up unless needed immediately)

Potential future files if canonical file/media support is added:

- `pkg/turns/keys_gen.go`
- `pkg/turns/helpers_blocks.go`
- `pkg/steps/ai/openai_responses/helpers.go`
- provider-specific helpers in other engines if a generic media abstraction is introduced

This should likely be its own ticket unless an implementation session proves the delta is still small and well-contained.

---

## Testing and validation strategy

## Unit tests

The first line of defense should be focused unit tests in the Responses package.

### Must-have tests

1. **URL image serialization**
   - Input: user text + one image URL
   - Expect: one user message with `input_text` and `input_image`

2. **Inline bytes serialization**
   - Input: user text + inline `[]byte`
   - Expect: one `input_image.image_url` that starts with `data:<media_type>;base64,`

3. **Inline base64 string serialization**
   - Input: user text + base64 string content
   - Expect: same data-URL behavior

4. **Multiple images**
   - Input: one user block with two images
   - Expect: one message containing one text part and two `input_image` parts

5. **Detail behavior**
   - Input: explicit `detail=high`
   - Expect: `detail: "high"`
   - Input: no detail
   - Expect: `detail: "auto"` or omitted only if intentionally standardized that way

6. **No regression in reasoning/tool behavior**
   - Existing tests remain green without modification to their intent

## Request-shape integration tests

At least one integration-style test should inspect the serialized request body sent through `RunInference(...)` or `CountTurn(...)`.

Why this matters:

- unit tests on `buildInputItemsFromTurn(...)` are necessary,
- but inspecting actual HTTP JSON catches accidental marshal/tag issues.

## Suggested test commands

```bash
go test ./pkg/steps/ai/openai_responses/... -count=1
go test ./... -count=1
```

If a smaller inner loop is needed:

```bash
go test ./pkg/steps/ai/openai_responses -run 'TestBuildInputItemsFromTurn|TestTokenCounter' -count=1
```

---

## Risks, sharp edges, and review checklist

## Risk 1: accidentally breaking reasoning/tool ordering

Why it matters:

- `buildInputItemsFromTurn(...)` has nuanced handling for reasoning and tool-call chains.
- A sloppy refactor could reorder content or change when messages are emitted.

How to control it:

- confine the patch mostly to `appendMessage` and related helpers,
- keep existing reasoning/tool tests untouched,
- avoid restructuring the entire loop unless necessary.

## Risk 2: malformed image maps silently disappearing

Why it matters:

- silent drops can make debugging very hard.

How to control it:

- decide explicitly whether malformed image maps should error or skip,
- pin that behavior in tests,
- prefer explicit comments over accidental permissiveness.

## Risk 3: divergence between OpenAI chat and OpenAI Responses paths

Why it matters:

- both engines consume similar image payloads,
- future changes could drift if the conversion logic is duplicated carelessly.

How to control it:

- keep the normalization logic small and readable,
- consider a follow-up shared helper only after the Responses behavior is proven,
- document the matching fields in both implementations.

## Risk 4: broadening scope too early to files/audio

Why it matters:

- the provider supports more than the turn model currently standardizes.

How to control it:

- ship image parity first,
- create a follow-up turn-schema ticket if broader media modeling is needed.

## Reviewer checklist

- [ ] Does `responsesContentPart` now support the official image fields?
- [ ] Does `appendMessage` preserve existing text behavior?
- [ ] Are user image blocks now serialized as `input_image` parts?
- [ ] Do inline bytes become valid `data:` URLs?
- [ ] Are `detail` and optional `file_id` handled correctly?
- [ ] Do existing reasoning/tool-chain tests still pass unchanged?
- [ ] Does token counting see the same image-bearing request shape?

---

## Open questions

1. **Should invalid image entries fail the whole request or be skipped?**
   - Recommendation: choose one explicit behavior and lock it in with tests. My bias is to fail helper-level unit tests for malformed local construction so bugs are visible early.

2. **Should we accept `file_id` immediately even though the current constructor comment does not mention it?**
   - Recommendation: yes, if implemented as optional support and documented as an accepted extension field. This is small and aligns with the official Responses schema.

3. **Should file support be added now?**
   - Recommendation: not unless there is a concrete immediate caller. The turn layer lacks a canonical file payload path, so that is better handled intentionally, not incidentally.

4. **Should image normalization be shared with the OpenAI chat path now?**
   - Recommendation: probably not in the first patch. Get the Responses path correct first, then evaluate a small shared helper if duplication becomes painful.

---

## Recommended next actions

### If you are the intern implementing this

1. Read this document once end-to-end.
2. Read these files in order:
   - `pkg/turns/helpers_blocks.go`
   - `pkg/steps/ai/openai/helpers.go`
   - `pkg/steps/ai/openai_responses/helpers.go`
   - `pkg/steps/ai/openai_responses/helpers_test.go`
   - `pkg/steps/ai/openai_responses/token_count.go`
3. Start with a failing test for one URL image case.
4. Implement the minimal serializer patch.
5. Add inline-image and detail tests.
6. Run package tests, then full repo tests.
7. Update the Geppetto inference-engine docs only after behavior is validated.

### Suggested work order

```text
1. Add failing helper test
2. Patch responsesContentPart + appendMessage path
3. Add inline-content tests
4. Add token-count request-shape test
5. Run focused package tests
6. Run broader repo tests
7. Update docs
```

---

## References

### Official OpenAI docs used in this analysis

- `/home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/02-openai-images-and-vision-guide.md:104-172`
- `/home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/03-openai-create-model-response-reference.md:15885-15917`
- `/home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/03-openai-create-model-response-reference.md:47549-47563`
- `/home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/04-openai-input-token-count-reference.md:23-119`

### Geppetto files used in this analysis

- `pkg/doc/topics/06-inference-engines.md:713-723`
- `pkg/inference/engine/engine.go:9-14`
- `pkg/turns/types.go:8-26`
- `pkg/turns/helpers_blocks.go:24-36`
- `pkg/steps/ai/openai_responses/engine.go:48-60`
- `pkg/steps/ai/openai_responses/helpers.go:15-31`
- `pkg/steps/ai/openai_responses/helpers.go:70-80`
- `pkg/steps/ai/openai_responses/helpers.go:113-119`
- `pkg/steps/ai/openai_responses/helpers.go:299-333`
- `pkg/steps/ai/openai_responses/token_count.go:34-52`
- `pkg/steps/ai/openai/helpers.go:190-220`
- `pkg/steps/ai/openai_responses/helpers_test.go:19-201`

### Final recommendation in one sentence

Implement image support first in `pkg/steps/ai/openai_responses/helpers.go` using the existing `PayloadKeyImages` contract, prove it with focused serializer and token-count tests, and treat broader file/audio/general-media support as a deliberately separate design step rather than hidden scope creep.
