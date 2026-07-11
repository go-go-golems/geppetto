---
Title: Normalize missing provider settings in inference profiles
Ticket: GEP-PROFILE-DEFAULTS
Status: complete
Topics:
    - javascript
    - profiles
    - inference
DocType: index
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: Make sparse registry-resolved inference profiles safe for JavaScript agent execution by materializing runtime-owned API, client, and provider settings according to chat API type.
LastUpdated: 2026-07-10T19:16:04.49140878-04:00
WhatFor: Track the analysis, implementation, tests, and validation that make omitted provider blocks behave as initialized defaults in Geppetto JavaScript profiles.
WhenToUse: Start here when a resolved inference profile builds an agent but fails during session execution because a runtime settings pointer is nil.
---


# Normalize missing provider settings in inference profiles

## Overview

Registry profile YAML deliberately permits concise definitions such as `api` plus `chat`. The JavaScript agent API resolves and clones those settings, then delegates to provider engines. Several engines dereference pointer fields that sparse YAML leaves nil. This ticket establishes one explicit runtime-normalization invariant: when a valid chat provider is selected, omitted runtime-owned settings become their normal empty/default objects before an engine is constructed.

The concrete failure originated in a local Ollama experiment using Geppetto's OpenAI-compatible chat engine. The profile resolved and `agent().inference(settings).build()` succeeded, but `session.next().run()` failed first with missing `Client` settings and then with missing `OpenAI` settings. The defect is general, not specific to Ollama or to the transcript-RAG application.

## Key links

- [Analysis, design, API contract, implementation plan, and test matrix](./design-doc/01-provider-aware-inference-profile-default-normalization-guide.md)
- [Chronological implementation diary](./reference/01-implementation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **complete**

## Topics

- javascript
- profiles
- inference

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Binding scope

- Normalize only runtime-owned, provider-defaultable objects: `API`, `Client`, and the selected provider's settings object.
- Do not invent a chat provider, model, credentials, user-supplied base URL, or policy opt-ins.
- Preserve existing explicit settings exactly.
- Keep unsupported `chat.api_type: ollama` unsupported; default construction must not imply engine support.
- Run no network requests in regression tests. Tests stop at deterministic request construction or use a local test engine seam.
