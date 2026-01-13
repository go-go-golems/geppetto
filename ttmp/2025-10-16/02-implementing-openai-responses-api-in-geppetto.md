# Implementing OpenAI Responses API for Thinking Models (o3, GPT‑5) in Geppetto

This document extends `01-adding-responses-openai-api-to-geppetto.md` with concrete, step-by-step implementation guidance, precise code surfaces, settings/layers wiring, tests, and rollout. It aligns with Geppetto’s engine/factory architecture and layers/parsed-layers system.

## Objectives
- Add Responses API support for o3/o4 and GPT‑5 reasoning models alongside existing Chat Completions.
- Preserve current behavior for non-reasoning models.
- Integrate with Geppetto’s layers/parsed-layers, engine/factory, sinks, and toolhelpers.

## Dependencies
- Keep: `github.com/sashabaranov/go-openai` (Chat Completions)
- Add: `github.com/openai/openai-go/v3` (Responses API)

```go
require (
    github.com/sashabaranov/go-openai vX.Y.Z
    github.com/openai/openai-go/v3 v3.x.x
)
```

Response path imports:
```go
openai   "github.com/openai/openai-go/v3"
responses "github.com/openai/openai-go/v3/responses"
"github.com/openai/openai-go/v3/option"
"github.com/openai/openai-go/v3/shared"
```

## Routing by Model
Decide at runtime whether to use Responses or Chat Completions:
```go
func requiresResponses(model string) bool {
    m := strings.ToLower(model)
    return strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") || strings.HasPrefix(m, "gpt-5")
}

model := deref(e.settings.Chat.Engine)
if requiresResponses(model) { return e.runResponses(ctx, t) }
return e.runChatCompletions(ctx, t)
```

## Engine: Responses Path
Create `pkg/steps/ai/openai/engine_openai_responses.go` with a streaming and non-streaming implementation using the official SDK:
```go
func (e *OpenAIEngine) runResponses(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
    client := openai.NewClient(
        option.WithAPIKey(e.settings.API.ApiKeyFor("openai")),
        option.WithBaseURL(e.settings.OpenAIBaseURL()),
    )
    params, err := MakeResponsesParamsFromTurn(e.settings, t)
    if err != nil { return nil, err }
    // streaming branch with event loop
    // non-streaming via client.Responses.New
}
```

## Helpers for Responses
Create `pkg/steps/ai/openai/helpers_responses.go` with:
- `MakeResponsesParamsFromTurn(s *settings.StepSettings, t *turns.Turn) (responses.ResponseNewParams, error)`
- `buildInputFromTurn(t *turns.Turn) ([]responses.InputItemParam, error)`
- `mapEffort(string) shared.ReasoningEffort`
- `PrepareToolsForResponses(defs []engine.ToolDefinition, cfg engine.ToolConfig) ([]responses.ToolUnionParam, error)`

Key mappings:
- Blocks→Input items (`system|user|assistant`) with content parts: text/image/audio
- Model: `shared.Model(*s.Chat.Engine)`
- Tokens: `MaxOutputTokens`
- Sampling: `Temperature`, `TopP`, `StopSequences`
- Reasoning: `Reasoning{Effort: low|medium|high}` if configured
- Tools: function tools with JSON Schema

## Settings and Layers
Extend OpenAI layer with two new params:
- `openai-reasoning-effort`: enum `low|medium|high` (optional)
- `openai-parallel-tool-calls`: bool (optional)

Wire via parsed layers into `StepSettings.OpenAI` so helpers can read them. Use `InitializeParameterDefaultsFromStruct` or YAML defaults per `13-layers-and-parsed-layers.md`.

## Tool Loop Compatibility
Engines emit tool call deltas/finals; `toolhelpers` continues to orchestrate execution and message appends. If needed, add a small adapter to expose tool calls in the existing DTO format.

## Tests
- Helpers: inputs, tools, params (reasoning/stop/tokens)
- Streaming accumulator: text deltas, tool deltas, completed
- Non-streaming: assistant content, usage, stop reason
- Back-compat: `gpt-4o*` continues Chat Completions
- Tool loop: model calls tool, execution appended, final answer produced

## Docs
- Update `06-inference-engines.md` with dual-stack routing and reasoning settings
- Add a short tutorial for o3/GPT‑5 using Responses with streaming and tools

## Rollout
- Add dependency → code → tests → docs
- Default behavior unchanged for existing models
- Optional settings; minimal migration needed

{
  "cells": [],
  "metadata": {
    "language_info": {
      "name": "python"
    }
  },
  "nbformat": 4,
  "nbformat_minor": 2
}