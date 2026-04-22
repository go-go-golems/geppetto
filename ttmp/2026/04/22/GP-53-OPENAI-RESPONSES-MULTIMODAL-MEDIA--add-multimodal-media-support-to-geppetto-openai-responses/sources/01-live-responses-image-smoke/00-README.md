---
Title: GP-53 live Responses image smoke fixture
Ticket: GP-53-OPENAI-RESPONSES-MULTIMODAL-MEDIA
Status: active
Topics:
  - inference
  - open-responses
  - openai-compatibility
DocType: reference
Intent: source-bundle
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Synthetic image fixture used to prove that the local Geppetto OpenAI Responses engine actually transmits image content to the model during a live smoke test."
LastUpdated: 2026-04-22T01:25:00-04:00
WhatFor: "Provide a reusable local image with visually distinctive facts that are not present in the prompt text."
WhenToUse: "Use when reproducing the GP-53 live multimodal smoke or comparing local Geppetto behavior with other clients such as pinocchio."
---

# GP-53 live Responses image smoke fixture

This folder contains the synthetic image used for the GP-53 live multimodal smoke.

The fixture intentionally contains visual facts that are not present in the question prompt:

- passcode text: `4319`
- a blue triangle on the left

This makes it useful for proving that a model actually inspected the image rather than answering from prompt text alone.
