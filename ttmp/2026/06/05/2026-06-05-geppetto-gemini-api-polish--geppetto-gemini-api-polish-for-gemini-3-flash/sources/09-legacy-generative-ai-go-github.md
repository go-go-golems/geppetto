---
Title: Deprecated Google Generative AI Go SDK
SourceURL: https://github.com/google/generative-ai-go
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:49-04:00
Ticket: 2026-06-05-geppetto-gemini-api-polish
Topics:
  - geppetto
  - providers
  - reasoning
  - streaming
  - tools
DocType: source
Status: active
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Official Gemini or SDK reference captured for the Geppetto Gemini API polish ticket.
WhatFor: Use as source material for Gemini 3 Flash API compatibility, thinking, thought signatures, function calling, and SDK migration.
WhenToUse: Read before changing Geppetto's Gemini provider implementation.
---

## \[Deprecated\] Google AI Go SDK for the Gemini API

With Gemini 2.0, we took the chance to create a single unified SDK for all developers who want to
use Google's GenAI models (Gemini, Veo, Imagen, etc). As part of that process, we took all of the
feedback from this SDK and what developers like about other SDKs in the ecosystem to create the
[Google Gen AI SDK](https://github.com/googleapis/go-genai).

The Gemini API docs are fully updated to show examples of the new Google Gen AI SDK: [Get
started](https://ai.google.dev/gemini-api/docs/quickstart?lang=go).

We know how disruptive an SDK change can be and don't take this change lightly, but our goal is to
create an extremely simple and clear path for developers to build with our models so it felt
necessary to make this change.

Thank you for building with Gemini and [let us know](https://discuss.ai.google.dev/c/gemini-api/4)
if you need any help!

**Please be advised that this repository is now considered legacy.** For the latest features,
performance improvements, and active development, we strongly recommend migrating to the official
**[Google Generative AI SDK for Go](https://github.com/googleapis/go-genai)**.

**Support Plan for this Repository:**

- **Limited Maintenance:** Development is now restricted to **critical bug fixes only**. No new
features will be added.
- **Purpose:** This limited support aims to provide stability for users while they transition to
the new SDK.
- **End-of-Life Date:** All support for this repository (including bug fixes) will permanently end
on **November 30, 2025**.

We encourage all users to begin planning their migration to the [Google Generative AI
SDK](https://github.com/googleapis/go-genai) to ensure continued access to the latest capabilities
and support.
