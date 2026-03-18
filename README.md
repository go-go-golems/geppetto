# Geppetto

![Geppetto](geppetto.jpg)

Geppetto is a Go toolkit for building LLM applications with a clean runtime model:

- provider-backed inference engines
- tool calling and tool loops
- streaming events and event sinks
- middleware composition
- typed turns and blocks
- read-only profile registry resolution
- native JavaScript bindings through Goja

It is the runtime core used by downstream apps such as Pinocchio, but it is also usable directly as a Go library plus example suite.

## What Geppetto Is Good At

Geppetto is strongest when you want to keep application policy in your app while reusing solid inference plumbing.

That means:

- your app decides final `StepSettings`
- your app decides which tools exist and which are exposed
- your app decides prompts, middleware policy, caching, and transport
- Geppetto builds engines, runs inference, executes tool loops, and emits events

The recommended entry point for new Go applications is [`pkg/inference/runner`](pkg/inference/runner), which provides a smaller app-facing API on top of the lower-level engine, session, middleware, and toolloop packages.

## Mental Model

The current architecture is intentionally simpler than the older profile-driven layering:

- profiles are read-only registry data
- profiles select prompt, tool names, and middleware metadata
- applications own final `StepSettings`
- applications pass resolved runtime input into Geppetto
- Geppetto runs inference from those resolved inputs

In other words:

```text
app config / profile selection
  -> final StepSettings + prompt + tools + middlewares
  -> Geppetto runner / engine / session
  -> events, turns, and final output
```

Not:

```text
app
  -> Geppetto mutates runtime config internally
  -> engine
```

## Start Here

If you are new to the repository, this is the shortest useful path:

1. Read [Turns and Blocks](pkg/doc/topics/08-turns.md) to understand the core data model.
2. Read [Opinionated Runner API](pkg/doc/topics/10-runner.md).
3. Run the three runner demos.
4. Only after that, look at the lower-level advanced examples.

## Quick Start

Requirements:

- Go `1.25.7`
- provider API keys for the engines you want to use

Common environment variables:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GOOGLE_API_KEY`
- local Ollama at `http://localhost:11434`

Common commands:

```bash
go test ./...
go build ./...
make lint
make gosec
```

## Recommended Go Examples

These are the examples most people should start with.

```bash
# smallest blocking run
go run ./cmd/examples/runner-simple

# same runner API, but with a function tool
go run ./cmd/examples/runner-tools

# same runner API, but async / streaming with event sinks
go run ./cmd/examples/runner-streaming
```

There are also two simpler non-runner examples that show the direct blocking and streaming shapes:

```bash
go run ./cmd/examples/inference
go run ./cmd/examples/streaming-inference
```

## Advanced Examples

These examples stay in the tree because they demonstrate lower-level or provider-specific control, not because they are the default API.

```bash
go run ./cmd/examples/advanced/middleware-inference
go run ./cmd/examples/advanced/generic-tool-calling
go run ./cmd/examples/advanced/openai-tools
go run ./cmd/examples/advanced/claude-tools
```

Use those when you explicitly want to study:

- low-level `session.Session` wiring
- `enginebuilder.Builder`
- provider-native tool behavior
- manual event router assembly

## JavaScript API

Geppetto also exposes a native Goja module:

```javascript
const gp = require("geppetto");
```

Main namespaces include:

- `gp.turns`
- `gp.profiles`
- `gp.middlewares`
- `gp.events`
- `gp.tools`
- `gp.schemas`

Use the JS lab harness to try it:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
```

References:

- [JS API Reference](pkg/doc/topics/13-js-api-reference.md)
- [JS API User Guide](pkg/doc/topics/14-js-api-user-guide.md)
- [examples/js/geppetto/README.md](examples/js/geppetto/README.md)

## Profile Registries

Geppetto still includes profile registry resolution, but the scope is narrower than before.

Profiles are now for read-only runtime selection metadata:

- system prompt
- tool names
- middleware declarations
- provenance and stack lineage

Applications are expected to own final engine configuration and `StepSettings`.

Supported registry sources include:

- YAML files
- SQLite files
- SQLite DSNs

Example registry shape:

```yaml
slug: team
profiles:
  default:
    slug: default
    runtime:
      system_prompt: You are the default assistant.
      tools:
        - search
```

References:

- [Profiles](pkg/doc/topics/01-profiles.md)
- [Migrate legacy profiles YAML to registry](pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md)
- [Operate SQLite-backed profile registry](pkg/doc/playbooks/06-operate-sqlite-profile-registry.md)
- [Bootstrap final StepSettings from defaults, config, registries, and profile](pkg/doc/playbooks/08-bootstrap-binary-step-settings-from-defaults-config-registries-profile.md)

## Core Packages

The main packages to know are:

- [`pkg/turns`](pkg/turns): typed turn and block data model
- [`pkg/inference/runner`](pkg/inference/runner): recommended app-facing API
- [`pkg/inference/engine`](pkg/inference/engine): provider engines and execution contracts
- [`pkg/inference/middleware`](pkg/inference/middleware): middleware pipeline
- [`pkg/inference/toolloop`](pkg/inference/toolloop): tool-call orchestration
- [`pkg/inference/session`](pkg/inference/session): session lifecycle and history
- [`pkg/events`](pkg/events): event model, printers, and routing support
- [`pkg/profiles`](pkg/profiles): read-only registry resolution
- [`pkg/js/modules/geppetto`](pkg/js/modules/geppetto): Goja integration

## Repository Layout

- `cmd/examples/`: runnable demos
- `examples/js/geppetto/`: JavaScript examples
- `pkg/`: library packages
- `pkg/doc/`: docs, playbooks, and tutorials
- `cmd/gen-meta/`: code generation support

## Documentation Map

Start with:

- [Documentation Index](pkg/doc/topics/00-docs-index.md)

High-value pages:

- [Turns and Blocks](pkg/doc/topics/08-turns.md)
- [Opinionated Runner API](pkg/doc/topics/10-runner.md)
- [Inference Engines](pkg/doc/topics/06-inference-engines.md)
- [Events and Streaming](pkg/doc/topics/04-events.md)
- [Tools](pkg/doc/topics/07-tools.md)
- [Middlewares](pkg/doc/topics/09-middlewares.md)
- [Profiles](pkg/doc/topics/01-profiles.md)

## Architecture Sketch

```text
           application-owned resolution

  profile selection / config / env / app policy
        -> StepSettings + prompt + tools + middlewares
        -> runner / engine / session / toolloop
        -> events + final Turn


              lower-level Geppetto building blocks

  turns      engine      middleware      toolloop      events
```

## Development Notes

- `make lint` uses the pinned `golangci-lint` version from `.golangci-lint-version`
- `make lintmax` and CI now lint the same package scope: `./cmd/... ./pkg/...`
- `ttmp/` is intentionally excluded from lint and security scanning

If you are touching behavior rather than only docs, run at least:

```bash
go test ./...
make lint
```
