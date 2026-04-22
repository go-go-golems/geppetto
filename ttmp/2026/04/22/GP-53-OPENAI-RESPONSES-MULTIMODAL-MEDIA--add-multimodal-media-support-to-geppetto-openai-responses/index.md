---
Title: Add multimodal media support to Geppetto openai-responses
Ticket: GP-53-OPENAI-RESPONSES-MULTIMODAL-MEDIA
Status: active
Topics:
    - inference
    - open-responses
    - openai-compatibility
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/steps/ai/openai/helpers.go
      Note: Comparison file showing how the OpenAI chat path already serializes image-bearing user messages
    - Path: geppetto/pkg/steps/ai/openai_responses/helpers.go
      Note: Primary file where OpenAI Responses request content parts are built and where image support is missing today
    - Path: geppetto/pkg/turns/helpers_blocks.go
      Note: Defines the current multimodal block constructor and image payload shape used by callers
ExternalSources:
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/02-openai-images-and-vision-guide.md
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/03-openai-create-model-response-reference.md
    - /home/manuel/workspaces/2026-04-21/hair-v2/hair-booking/ttmp/2026/04/21/HAIR-020--integrate-geppetto-llm-review-with-pinocchio-geppetto-profile-registry-bootstrap-in-css-visual-diff/sources/02-openai-responses-api-docs/04-openai-input-token-count-reference.md
Summary: Research ticket for adding OpenAI Responses multimodal media support to Geppetto, centered on request building, test coverage, and a staged design for image support first and broader media support second.
LastUpdated: 2026-04-22T00:30:00-04:00
WhatFor: Track the analysis and implementation plan for OpenAI Responses multimodal media support in Geppetto.
WhenToUse: Use this ticket when implementing or reviewing image and future file/media support in the openai_responses engine.
---


# Add multimodal media support to Geppetto openai-responses

## Overview

This ticket captures the research and design work needed to add real multimodal media support to Geppetto's OpenAI Responses engine. The current implementation already supports text, reasoning, and tools, but it still drops image payloads when converting Geppetto `Turn` blocks into Responses API request items.

The primary deliverable is a detailed intern-friendly design document that explains the current architecture, the OpenAI contract, the exact gap in `pkg/steps/ai/openai_responses/helpers.go`, and a phased implementation plan.

## Key Links

- Design doc: `design-doc/01-openai-responses-multimodal-media-support-analysis-design-and-implementation-guide.md`
- Diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Scope Summary

### Completed in this ticket

- Created a Geppetto-local docmgr ticket workspace under `geppetto/ttmp`
- Reviewed official OpenAI Responses API docs already downloaded in HAIR-020
- Audited the Geppetto openai-responses request-building path
- Compared the Responses path against the existing OpenAI chat-completions image path
- Wrote a detailed implementation guide and investigation diary

### Planned follow-up work

- Add image request-part support to `pkg/steps/ai/openai_responses/helpers.go`
- Add regression tests for URL, base64 data URL, and `detail`
- Decide whether to extend the provider-neutral turn schema for files and other media
- Update Geppetto docs/examples after implementation lands

## Structure

- `design-doc/` - primary design and implementation guidance
- `reference/` - chronological diary of research and decisions
- `scripts/` - ticket-local scripts if implementation work starts
- `sources/` - optional source notes if local copies of external references are needed later
- `various/` - scratch artifacts if needed later
