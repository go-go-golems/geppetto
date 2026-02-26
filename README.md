# Geppetto

![Geppetto](geppetto.jpg)

Geppetto is the runtime core for building LLM applications in Go.

Current project direction is:

- provider-agnostic inference engines,
- first-class tool calling and tool-loop orchestration,
- middleware-based runtime composition,
- typed turn/block data model,
- hard-cutover profile registries with stack resolution and provenance,
- native JavaScript bindings through Goja (`require("geppetto")`).

This repository is primarily a library + examples repo. Downstream applications (for example pinocchio and go-go-os) consume Geppetto as runtime infrastructure.

## Current Model (Important)

Geppetto now uses a registry-first, profile-first runtime model:

- runtime config is resolved from profile registries (not ad-hoc legacy profile maps),
- registry sources are stackable (`yaml`, SQLite file, `sqlite-dsn`),
- top-of-stack precedence is deterministic,
- request-time overrides are policy-gated,
- resolved runtime includes stack lineage/trace metadata and runtime fingerprint.

There is no overlay abstraction in active runtime composition.
There is no runtime `registrySlug` selector in `engines.fromProfile(...)`.

## Core Runtime Building Blocks

- `pkg/turns`: canonical conversation model (`Turn`, `Block`, typed keys).
- `pkg/inference/engine`: provider engine interfaces and implementations.
- `pkg/inference/toolloop`: orchestration loop for tool calls/results/retries.
- `pkg/inference/middleware`: inference middleware pipeline.
- `pkg/inference/session`: session lifecycle and history.
- `pkg/profiles`: profile registries, stack resolution, policy, storage backends.
- `pkg/js/modules/geppetto`: JS module exposed through Goja.

## Profile Registries

Runtime registry sources are loaded via `profile-registries` (CLI flag, env, or config depending on host app).

Supported source entries:

- `path/to/registry.yaml`
- `path/to/profiles.db`
- `sqlite-dsn:file:./profiles.db?...`

Single YAML format is one registry per file:

```yaml
slug: team
profiles:
  default:
    slug: default
    runtime:
      step_settings_patch:
        ai-chat:
          ai-api-type: openai
          ai-engine: gpt-4o-mini
```

For details:

- [`pkg/doc/topics/01-profiles.md`](pkg/doc/topics/01-profiles.md)
- [`pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md`](pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md)
- [`pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`](pkg/doc/playbooks/06-operate-sqlite-profile-registry.md)

## JavaScript API

Geppetto exposes a native module inside Goja:

```javascript
const gp = require("geppetto");
```

Main namespaces:

- `gp.turns`
- `gp.engines`
- `gp.profiles`
- `gp.schemas`
- `gp.middlewares`
- `gp.tools`

`gp.profiles` supports both host-injected registries and runtime stack binding:

- `connectStack(sources)`
- `disconnectStack()`
- `getConnectedSources()`

Use the JS lab harness:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
```

Run full JS profile/registry examples:

```bash
./examples/js/geppetto/run_profile_registry_examples.sh
```

References:

- [`pkg/doc/topics/13-js-api-reference.md`](pkg/doc/topics/13-js-api-reference.md)
- [`pkg/doc/topics/14-js-api-user-guide.md`](pkg/doc/topics/14-js-api-user-guide.md)
- [`examples/js/geppetto/README.md`](examples/js/geppetto/README.md)

## Quick Start: Go Examples

```bash
# simple inference pipeline
go run ./cmd/examples/simple-inference

# streaming output
go run ./cmd/examples/simple-streaming-inference

# middleware + tools examples
go run ./cmd/examples/middleware-inference
go run ./cmd/examples/generic-tool-calling
go run ./cmd/examples/openai-tools
go run ./cmd/examples/claude-tools

# JS host harness
go run ./cmd/examples/geppetto-js-lab --list-go-tools
```

All runnable examples are under `cmd/examples/`.

## Documentation

Start with:

- [`pkg/doc/topics/00-docs-index.md`](pkg/doc/topics/00-docs-index.md)

High-value pages:

- profiles: [`pkg/doc/topics/01-profiles.md`](pkg/doc/topics/01-profiles.md)
- engines: [`pkg/doc/topics/06-inference-engines.md`](pkg/doc/topics/06-inference-engines.md)
- tools: [`pkg/doc/topics/07-tools.md`](pkg/doc/topics/07-tools.md)
- middlewares: [`pkg/doc/topics/09-middlewares.md`](pkg/doc/topics/09-middlewares.md)
- sessions: [`pkg/doc/topics/10-sessions.md`](pkg/doc/topics/10-sessions.md)
- JS API reference: [`pkg/doc/topics/13-js-api-reference.md`](pkg/doc/topics/13-js-api-reference.md)

## Repository Layout

- `pkg/` runtime packages
- `cmd/examples/` runnable binaries
- `examples/js/geppetto/` JS scripts for API coverage
- `pkg/doc/` docs, playbooks, tutorials
- `cmd/gen-meta/` codegen for constants/type artifacts

## Development

Requirements:

- Go `1.25.7` (see `go.mod`)

Common commands:

```bash
go test ./...
go generate ./...
go build ./...
```
