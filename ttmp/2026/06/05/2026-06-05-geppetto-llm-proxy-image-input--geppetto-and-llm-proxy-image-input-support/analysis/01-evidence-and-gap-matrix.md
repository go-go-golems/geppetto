---
Title: Evidence and Gap Matrix
Ticket: 2026-06-05-geppetto-llm-proxy-image-input
Status: active
Topics:
    - geppetto
    - llm-proxy
    - providers
    - multimodal
DocType: analysis
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: pkg/steps/ai/openai_responses/helpers_test.go
      Note: Existing Responses image fixture coverage
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-llm-proxy-image-input--geppetto-and-llm-proxy-image-input-support/design-doc/01-image-input-support-intern-guide.md
      Note: Detailed design and implementation guide
    - Path: ttmp/2026/06/05/2026-06-05-geppetto-llm-proxy-image-input--geppetto-and-llm-proxy-image-input-support/scripts/01-evidence-line-anchors.md
      Note: Evidence excerpt artifact
ExternalSources:
    - https://platform.openai.com/docs/api-reference/chat/create
    - https://platform.openai.com/docs/api-reference/responses/create
    - https://docs.anthropic.com/en/api/messages
    - https://pkg.go.dev/google.golang.org/genai
Summary: Evidence-backed matrix for image input support across llm-proxy and Geppetto providers.
LastUpdated: 2026-06-05T17:45:00-04:00
WhatFor: Use as a quick reference for current image support and exact files to inspect before implementation.
WhenToUse: Read when planning or reviewing multimodal image input changes.
---


# Evidence and Gap Matrix

## Purpose

This document records the concrete evidence gathered for image input support across `llm-proxy` and Geppetto's provider backends. The detailed implementation guidance is in `design-doc/01-image-input-support-intern-guide.md`; this document is the quick matrix and line-anchor reference.

## Current support matrix

| Component | Status | Evidence | Required change |
|---|---|---|---|
| Geppetto canonical turn model | Existing representation | `pkg/turns/helpers_blocks.go:24-40` defines `NewUserMultimodalBlock(text, images)` and documents `media_type`, `url`, `content`, `file_id`, and `detail`. | Keep this as the internal representation. Add shared normalization around the existing map shape. |
| `llm-proxy` Chat request decoding | Blocks images | `llm-proxy/pkg/openaichat/types.go:121-130` rejects content arrays with `unsupported_content_shape`. | Parse OpenAI-compatible content arrays and image parts. |
| `llm-proxy` Chat mapper | Text-only user mapping | `llm-proxy/pkg/openaichat/mapper.go:29-31` maps user content to `turns.NewUserTextBlock(text)`. | Map user messages with images to `turns.NewUserMultimodalBlock`. |
| Geppetto OpenAI Responses | Strong support | `pkg/steps/ai/openai_responses/helpers.go:604-637` maps images to `input_image`; tests in `helpers_test.go:40-132` cover URL, inline bytes, multiple images, and detail. | Use as baseline behavior; optionally refactor to shared normalization. |
| Geppetto OpenAI Chat | Partial support | `pkg/steps/ai/openai/helpers.go:237-266` builds `image_url` parts from `PayloadKeyImages`. | Add normalization, `image_url` alias support, detail preservation, and tests. |
| Geppetto Claude | Partial support | `pkg/steps/ai/claude/helpers.go:235-248` maps inline base64 image content with `media_type` into `api.NewImageContent`. | Add data URL/bytes normalization and explicit policy for plain remote URLs. |
| Geppetto Gemini modern path | Missing | `pkg/steps/ai/gemini/modern_adapter.go:241-253` only appends text parts for user/system/other blocks. | Add `InlineData` mapping for inline images and optional explicit `file_uri` support. |
| Gemini SDK capability | Available | `google.golang.org/genai` defines `Blob` and `Part.InlineData` in `types.go:1328-1431`. | Use `&moderngenai.Part{InlineData: &moderngenai.Blob{MIMEType, Data}}`. |

## Recommended canonical image descriptor

The existing map keys should be treated as the first-phase contract:

| Key | Type | Meaning |
|---|---|---|
| `url` | string | Provider-readable image URL or data URL. |
| `image_url` | string | OpenAI-compatible alias for `url`. |
| `content` | `[]byte` or string | Inline image bytes, base64 string, or data URL. |
| `media_type` | string | MIME type such as `image/png`; required for inline raw/base64 content. |
| `detail` | string | OpenAI detail hint: `auto`, `low`, or `high`. |
| `file_id` | string | OpenAI Responses file reference. |
| `file_uri` | string | Gemini/provider-side file URI. |

## Implementation priority

1. `llm-proxy` content-array parser and mapper.
2. Shared image normalization helper in Geppetto.
3. Gemini `InlineData` support in `modern_adapter.go`.
4. Test/refactor OpenAI Responses, OpenAI Chat, and Claude image mappings.
5. Direct Geppetto image smokes.
6. Proxy image smokes.

## Evidence artifact

The line-anchored evidence collected for this ticket is stored at:

```text
scripts/01-evidence-line-anchors.md
```

It includes excerpts for:

- Geppetto multimodal block constructor.
- OpenAI Responses image mapping and tests.
- OpenAI Chat image mapping.
- Claude image mapping.
- Gemini modern adapter user-block mapping.
- `llm-proxy` content validation and mapper.
- Gemini SDK `Blob` and `Part.InlineData` fields.

## Open questions

- Should `llm-proxy` accept image content arrays for assistant messages during replay, or should phase one reject them explicitly?
- Should Claude URL images return errors or be silently skipped? Explicit errors are easier to debug, but skipping may preserve compatibility with mixed payloads.
- Should Gemini support only inline data initially, or should explicit `file_uri` be accepted in the same patch?
- Should the map-based image descriptor remain long term, or should Geppetto eventually generate typed image payload keys?
