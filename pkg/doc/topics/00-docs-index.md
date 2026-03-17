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

Geppetto is a Go toolkit for building AI-powered applications. It provides:

- **Streaming inference** with real-time token delivery
- **Tool calling** that works across providers (OpenAI, Claude, Gemini, Ollama)
- **Provider-agnostic architecture** — write once, switch providers via config
- **Event-driven design** for building responsive UIs and debugging
- **Profile registry stack resolution** (YAML + SQLite sources, policy-gated request overrides)
- **Native JS API** (`require("geppetto")`) for scripting and host embedding

## Start Here (New Users)

If you're new to Geppetto, read these docs in order:

1. **[Turns and Blocks](08-turns.md)** — The core data model. Understand this first.
2. **[Inference Engines](06-inference-engines.md)** — How inference is executed across providers.
3. **[Profiles](01-profiles.md)** — Registry-first runtime selection and stack resolution.
4. **[JS API Getting Started](../tutorials/05-js-api-getting-started.md)** — Step-by-step tutorial for the JavaScript API.
5. **[Streaming Tutorial](../tutorials/01-streaming-inference-with-tools.md)** — Build your first streaming command.

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
| [JS API Reference](13-js-api-reference.md) | Exhaustive contract for `require(\"geppetto\")` namespaces and options. |
| [JS API User Guide](14-js-api-user-guide.md) | Practical composition patterns for sessions, middlewares, tools, and hooks. |

## Configuration and Setup

| Doc | What It Covers |
|-----|----------------|
| [Profiles](01-profiles.md) | Registry-first profile model, persistence, policy, and migration from legacy profile maps. |
| [Embeddings](06-embeddings.md) | Vector embeddings for semantic search, including caching. |
| [Linting (turnsdatalint)](12-turnsdatalint.md) | Custom linter for Turn data key hygiene. |

## Tutorials

| Tutorial | What You'll Build |
|----------|-------------------|
| [Streaming Inference with Tools](../tutorials/01-streaming-inference-with-tools.md) | A Cobra command that streams output and supports tool calling. |
| [Event Routing and Structured Logging](../tutorials/02-event-routing-and-structured-logging.md) | Multiple event handlers with JSON logging and metrics. |
| [Embeddings Workflows](../tutorials/03-embeddings-workflows.md) | Semantic search with caching and batch processing. |
| [Structured Data Extraction](../tutorials/04-structured-data-extraction.md) | Extract structured data from streaming output in real-time. |
| [JS API Getting Started](../tutorials/05-js-api-getting-started.md) | End-to-end walkthrough of turns, sessions, tools, and hooks from JavaScript. |
| [Build Scoped JavaScript Eval Tools](../tutorials/07-build-scopedjs-eval-tools.md) | Detailed intern-friendly guide to `pkg/inference/tools/scopedjs`, runtime wiring, registration, and debugging. |

## Playbooks

Operational step-by-step guides for common tasks:

| Playbook | What You'll Do |
|----------|----------------|
| [Add a New Tool](../playbooks/01-add-a-new-tool.md) | Register a tool, attach to context, configure on Turn, run tool loop. |
| [Add a New Event Handler](../playbooks/02-add-a-new-event-handler.md) | Implement handler, subscribe to router, parse events. |
| [Progressive Structured Data](../playbooks/03-progressive-structured-data.md) | Extract structured payloads from streaming output with filtering sinks. |
| [Migrate to Session API](../playbooks/04-migrate-to-session-api.md) | Migrate from legacy lifecycle APIs to `session.Session` + `ExecutionHandle`. |
| [Migrate legacy profiles.yaml to registry](../playbooks/05-migrate-legacy-profiles-yaml-to-registry.md) | Convert legacy profile maps to canonical registry YAML and switch to registry-first profile workflows. |
| [Operate SQLite-backed profile registry](../playbooks/06-operate-sqlite-profile-registry.md) | Run profile registry with SQLite storage, plus backup/recovery and permissions guidance. |
| [Adopt Scoped JavaScript Eval Tools](../playbooks/09-adopt-scopedjs-eval-tools.md) | Package a prepared goja runtime as one `eval_xxx` tool using `scopedjs`. |
| [Bootstrap binary StepSettings from defaults, config, registries, and profile](../playbooks/08-bootstrap-binary-step-settings-from-defaults-config-registries-profile.md) | Build final runtime settings by separating hidden base parsing from profile resolution. |

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

- **Go 1.25+** (repository currently uses `go 1.25.7`)
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
- **Debug**: Use `--print-parsed-fields` to see resolved configuration

All public docs use the Turn-based architecture. Legacy conversation APIs are intentionally omitted.
