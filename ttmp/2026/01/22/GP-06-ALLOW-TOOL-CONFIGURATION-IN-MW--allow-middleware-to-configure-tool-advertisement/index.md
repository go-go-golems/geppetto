---
Title: Allow tool configuration in middleware (tool advertisement)
Ticket: GP-06-ALLOW-TOOL-CONFIGURATION-IN-MW
Status: active
Topics:
  - geppetto
  - inference
  - tools
  - middleware
  - openai
  - claude
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
  - Path: geppetto/pkg/inference/toolcontext/context.go
    Note: Where tool registry is carried on context
  - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
    Note: Builds OpenAI request tools from registry
  - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
    Note: Builds Responses request tools from registry
  - Path: geppetto/pkg/steps/ai/claude/engine_claude.go
    Note: Builds Claude request tools from registry
  - Path: geppetto/pkg/inference/toolhelpers/helpers.go
    Note: Tool loop attaches registry + config at runtime
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-22T17:40:05-05:00
WhatFor: ""
WhenToUse: ""
---

# Allow tool configuration in middleware (tool advertisement)

## Intent

Enable **middleware-driven configuration of which tool descriptions are passed to the provider inference call** (OpenAI / OpenAI Responses / Claude / Gemini).

Concretely:

- Providers currently derive the tool schema list from the `tools.ToolRegistry` stored on `context.Context`.
- We want a middleware to be able to:
  - restrict the *advertised* tool set (e.g. per mode, per user, per run),
  - optionally rewrite/augment tool descriptions/schemas before the provider sees them,
  - without changing the actual tool execution mechanism.

This is about **advertisement/configuration** (what the model is told it can call), not about executing tool calls. Tool execution remains the responsibility of:

- `toolhelpers.RunToolCallingLoop(...)`, and
- `tools.ToolExecutor` (policy/enforcement/execution).

## Why

We need a clean way for application middleware (e.g. agent-mode) to influence which tools the model is allowed to call *at inference time*, without requiring every provider engine to understand application-specific keys (like `turns.KeyAgentModeAllowedTools`) directly.

## Non-goals

- Do not re-introduce “tool execution as middleware”.
- Do not design/implement tool execution allow-list enforcement here (that can live in executor policy).
