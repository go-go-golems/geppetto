---
Title: Fix thinking model parameter handling
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: complete
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Investigate and fix GPT-5/o-series parameter handling for chat vs responses engines (max tokens + sampling params).
LastUpdated: 2026-01-25T09:04:54.712486196-05:00
WhatFor: ""
WhenToUse: ""
---


# Fix thinking model parameter handling

## Overview

We are hitting model validation errors when using GPT-5 or other reasoning models in chat-mode (`openai` engine) and in Responses mode. GPT-5 rejects `max_tokens` in chat and requires `max_completion_tokens`, and the Responses API rejects `temperature` for GPT-5 reasoning models. This ticket tracks the fixes needed to correctly gate parameters for "thinking"/reasoning models across both engines.

## Current Symptoms

- Chat-mode invocation fails with: `this model is not supported MaxTokens, please use MaxCompletionTokens`
- Responses-mode invocation fails with: `Unsupported parameter: 'temperature' is not supported with this model.`

## Scope

- Adjust chat-mode (openai) request building to use `max_completion_tokens` and omit unsupported sampling params for GPT-5/o-series.
- Adjust Responses request building to omit sampling params for GPT-5/o-series (temperature/top_p/etc.).
- Document how to select engine type for GPT-5 models.

## Non-Goals

- No backwards-compat adapters beyond parameter gating for reasoning models.
- No changes to unrelated provider engines.
