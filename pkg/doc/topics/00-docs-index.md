---
Title: Geppetto documentation index
Slug: geppetto-docs-index
Short: Task-based index for Geppetto docs and example programs.
Topics:
- geppetto
- documentation
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Geppetto Documentation Index

Geppetto is a Go library for building AI-powered applications. It provides:

- **Streaming inference** with real-time token delivery
- **Tool calling** that works across providers (OpenAI, Claude, Gemini, Ollama)
- **Provider-agnostic architecture** — write once, switch providers via config
- **Event-driven design** for building responsive UIs and debugging

## Start Here (New Users)

If you're new to Geppetto, read these docs in order:

1. **[Turns and Blocks](08-turns.md)** — The core data model. Understand this first.
2. **[Inference Engines](06-inference-engines.md)** — How to run inference against AI providers.
3. **[Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md)** — Build your first streaming command.

After that, explore based on what you need:

- Building a chat UI? → [Events and Streaming](04-events.md)
- Adding function calling? → [Tools](07-tools.md)
- Need semantic search? → [Embeddings](06-embeddings.md)

## Core Concepts

| Doc | What It Covers |
|-----|----------------|
| [Turns and Blocks](08-turns.md) | The Turn data model: `Run` → `Turn` → `Block`. How conversations are represented. |
| [Inference Engines](06-inference-engines.md) | The `Engine` interface, factory pattern, and provider implementations. |
| [Tools](07-tools.md) | Defining tools, registering them, and executing tool calls. |
| [Events and Streaming](04-events.md) | Real-time event delivery, Watermill routing, and printers. |
| [Middlewares](09-middlewares.md) | Adding cross-cutting behavior (logging, tool execution) around inference. |

## Configuration and Setup

| Doc | What It Covers |
|-----|----------------|
| [Profiles](01-profiles.md) | Switching between providers and environments via config. |
| [Embeddings](06-embeddings.md) | Vector embeddings for semantic search, including caching. |
| [Linting (turnsdatalint)](12-turnsdatalint.md) | Custom linter for Turn data key hygiene. |

## Tutorials

| Tutorial | What You'll Build |
|----------|-------------------|
| [Streaming Inference with Tools](../tutorials/01-streaming-inference-with-tools.md) | A Cobra command that streams output and supports tool calling. |

## Example Programs

These working examples are the source of truth for patterns. Run them to see Geppetto in action:

| Example | Description |
|---------|-------------|
| `simple-streaming-inference/` | Basic streaming inference without tools. Start here. |
| `generic-tool-calling/` | Provider-agnostic tool calling (works with any AI backend). |
| `openai-tools/` | Tool calling using OpenAI's native function calling. |
| `claude-tools/` | Tool calling using Claude's native tool_use format. |
| `middleware-inference/` | Using middlewares for logging and tool execution. |
| `citations-event-stream/` | Structured data extraction from streaming output. |

Find examples in: `geppetto/cmd/examples/`

## Prerequisites

- **Go 1.24+**
- **API keys** for your chosen provider(s):
  - OpenAI: `OPENAI_API_KEY`
  - Claude: `ANTHROPIC_API_KEY`
  - Gemini: `GOOGLE_API_KEY`
  - Ollama: running locally at `http://localhost:11434`

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐    ┌──────────┐    ┌─────────────────────────┐ │
│  │  Turn   │───▶│  Engine  │───▶│  Provider (OpenAI/...)  │ │
│  │ (Blocks)│◀───│          │◀───│                         │ │
│  └─────────┘    └────┬─────┘    └─────────────────────────┘ │
│                      │                                       │
│                      ▼                                       │
│               ┌──────────────┐                               │
│               │  Event Sink  │──▶ Router ──▶ Handlers        │
│               └──────────────┘                               │
├─────────────────────────────────────────────────────────────┤
│  Middleware: Logging │ Tools │ Validation │ Custom          │
└─────────────────────────────────────────────────────────────┘
```

## Getting Help

- **API docs**: Run `go doc github.com/go-go-golems/geppetto/pkg/...`
- **Examples**: Study the example programs in `geppetto/cmd/examples/`
- **Debug**: Use `--print-parsed-parameters` to see resolved configuration

All public docs use the Turn-based architecture. Legacy conversation APIs are intentionally omitted.
