# Geppetto

![Geppetto](geppetto.jpg)

Geppetto is a Go toolkit for building LLM applications with:

- provider-agnostic inference engines,
- tool calling and tool loop orchestration,
- middleware-based runtime composition,
- typed turn/block data structures,
- stackable profile registries (YAML + SQLite),
- a native JavaScript API (`require("geppetto")`) via Goja.

It is the runtime foundation used by downstream apps such as pinocchio and go-go-os.

## What Geppetto Focuses On

- Runtime correctness over prompt-template sugar.
- Strongly-typed inference/session/tooling primitives in Go.
- Profile-first runtime selection with deterministic merge/fingerprint semantics.
- Scriptable JS bindings for rapid prototyping and host embedding.

## Core Concepts

- `Turn` / `Block`: canonical conversation state model.
- `Engine`: provider-facing inference abstraction.
- `Session`: orchestration state (history + run lifecycle).
- `Middleware`: composable runtime behavior around inference.
- `ToolRegistry` + `toolloop`: tool calling execution and retries.
- `ProfileRegistry`: durable runtime defaults/policy/provenance.

## Profile Registries (Current Hard-Cut Model)

Runtime selection is registry-first:

- load one or more registry sources via `--profile-registries` (YAML, SQLite file, or sqlite-dsn),
- choose `--profile` slug,
- resolve through stack order (top of stack wins on collisions),
- enforce policy-gated request overrides.

Single YAML file format is **one registry per file** (`slug` + `profiles`).

See:

- [`pkg/doc/topics/01-profiles.md`](pkg/doc/topics/01-profiles.md)
- [`pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md`](pkg/doc/playbooks/05-migrate-legacy-profiles-yaml-to-registry.md)
- [`pkg/doc/playbooks/06-operate-sqlite-profile-registry.md`](pkg/doc/playbooks/06-operate-sqlite-profile-registry.md)

## JavaScript API

Geppetto exposes a native module for Goja:

```javascript
const gp = require("geppetto");
```

Key namespaces:

- `gp.turns`
- `gp.engines`
- `gp.profiles`
- `gp.schemas`
- `gp.middlewares`
- `gp.tools`

Run scripts with the lab harness:

```bash
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
```

Run the full JS profile/registry suite:

```bash
./examples/js/geppetto/run_profile_registry_examples.sh
```

See:

- [`pkg/doc/topics/13-js-api-reference.md`](pkg/doc/topics/13-js-api-reference.md)
- [`pkg/doc/topics/14-js-api-user-guide.md`](pkg/doc/topics/14-js-api-user-guide.md)
- [`examples/js/geppetto/README.md`](examples/js/geppetto/README.md)

## Quick Start (Go Examples)

```bash
# deterministic local inference
go run ./cmd/examples/simple-inference

# streaming output
go run ./cmd/examples/simple-streaming-inference

# provider + tools examples
go run ./cmd/examples/generic-tool-calling
go run ./cmd/examples/openai-tools
go run ./cmd/examples/claude-tools
```

Explore all example binaries under `cmd/examples/`.

## Repository Layout

- `pkg/` core runtime packages (engines, sessions, tools, profiles, JS module)
- `cmd/examples/` runnable examples and `geppetto-js-lab`
- `pkg/doc/` help pages, tutorials, and playbooks
- `examples/js/geppetto/` runnable JS API scripts

## Documentation Index

Start here:

- [`pkg/doc/topics/00-docs-index.md`](pkg/doc/topics/00-docs-index.md)

## Development

Requirements:

- Go `1.25.7` (see `go.mod`)

Common checks:

```bash
go test ./...
go generate ./...
go build ./...
```
