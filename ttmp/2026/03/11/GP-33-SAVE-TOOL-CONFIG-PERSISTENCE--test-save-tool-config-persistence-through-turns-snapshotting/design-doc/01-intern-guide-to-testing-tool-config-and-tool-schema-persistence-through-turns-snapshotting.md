---
Title: intern guide to testing tool config and tool schema persistence through turns snapshotting
Ticket: GP-33-SAVE-TOOL-CONFIG-PERSISTENCE
Status: active
Topics:
    - geppetto
    - persistence
    - testing
    - tools
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/cmd/examples/geppetto-js-lab/main.go
      Note: Best insertion point for host-side persistence flags and YAML persister wiring
    - Path: geppetto/examples/js/geppetto/04_tools_and_toolloop.js
      Note: Minimal secondary smoke example
    - Path: geppetto/examples/js/geppetto/05_go_tools_from_js.js
      Note: Primary recommended fixture because Go tools provide stronger schema assertions
    - Path: geppetto/pkg/inference/engine/turnkeys_gen.go
      Note: Generated typed keys for ToolConfig and ToolDefinitions
    - Path: geppetto/pkg/inference/toolloop/loop.go
      Note: Primary runtime stamping path for persisted tool metadata
    - Path: geppetto/pkg/inference/tools/advertisement.go
      Note: Proves runtime advertisement still comes from the live context registry
    - Path: geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: JS builder/session surface already supports persister and snapshot hook injection
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Host module options include DefaultPersister and DefaultSnapshotHook
    - Path: geppetto/pkg/steps/ai/openai_responses/engine_test.go
      Note: Regression tests demonstrating persisted tool definitions are inspection-only
    - Path: geppetto/pkg/turns/serde/serde_test.go
      Note: Printed YAML proves exact on-disk key names and payload shape
    - Path: geppetto/pkg/turns/types.go
      Note: YAML key serialization explains why persisted files use canonical typed-key IDs
ExternalSources: []
Summary: End-to-end analysis and implementation guide for proving that Geppetto persists tool loop configuration and serializable tool definitions onto turns and that a host-side persister writes those fields to disk.
LastUpdated: 2026-03-11T08:00:00-04:00
WhatFor: Give a new engineer enough system context to add a persistence probe to a Geppetto example binary, run it locally, and verify the exact on-disk YAML fields for tool configuration and tool schema snapshots.
WhenToUse: Use when validating the March 10, 2026 `tool_definitions` persistence work, when wiring a host-side turn persister/snapshot hook, or when debugging why persisted turn files do or do not contain tool metadata.
---


# intern guide to testing tool config and tool schema persistence through turns snapshotting

## Executive Summary

The change from the last two Geppetto pull requests is real and locally visible in code:

1. the tool loop now stamps `engine.KeyToolConfig` and `engine.KeyToolDefinitions` onto `Turn.Data` before the first inference call,
2. the persisted tool-definition payload is intentionally JSON-safe (`Parameters map[string]any`), and
3. provider request-building still advertises tools from the live registry in `context.Context`, not from persisted turn data.

That means the verification task is not "does the engine use persisted tool definitions at runtime?" The correct question is:

> when a host application persists the completed turn to disk, does that serialized turn contain both `geppetto.tool_config@v1` and `geppetto.tool_definitions@v1`?

The best example binary to turn into that proof is `cmd/examples/geppetto-js-lab/main.go` together with `examples/js/geppetto/05_go_tools_from_js.js`.

This is the best fit because:

- it already exercises the canonical JS builder/session/tool-loop path,
- it already exposes built-in Go tools with generated JSON Schema,
- it does not require live provider network calls when used with `gp.engines.fromFunction(...)`,
- it is already documented as the standard JS harness in `README.md`,
- it can accept a host-side `DefaultPersister` and `DefaultSnapshotHook` with very small, isolated changes.

Observed local validation on March 11, 2026:

- `GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js` passed.
- `GOWORK=off GOCACHE=/tmp/geppetto-go-build go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js` passed.
- `GOWORK=off GOCACHE=/tmp/geppetto-go-build go test ./pkg/inference/toolloop ./pkg/turns/serde ./pkg/inference/tools` passed.
- `GOWORK=off go test ./pkg/steps/ai/openai_responses` also passed once rerun outside the original sandbox restriction.
- the only remaining reproducible environment issue is the workspace `go.work` version mismatch when commands are run without `GOWORK=off`.

## Problem Statement

You want a new intern to verify a very specific contract:

- tools configured through the Geppetto tool loop are written onto `Turn.Data`,
- the persisted tool schema snapshot survives turn serialization,
- a host-side turn persister writes those fields to disk,
- the chosen test uses an existing `@geppetto/cmd/` example instead of inventing a new ad hoc runtime.

The tricky part is that there are three closely related but different concepts:

1. live runtime tool registry,
2. in-memory turn snapshots during a run,
3. persisted turn files on disk.

If the intern mixes these up, they will likely test the wrong thing.

The codebase today makes a deliberate separation:

- live registry: authoritative for execution and provider advertisement,
- `Turn.Data`: authoritative for durable inspection metadata,
- snapshot hook: observes intermediate phases,
- persister: writes the completed turn.

So the implementation guide must teach the boundary first, then the commands.

## Scope

This ticket is scoped to verification and host-side test wiring.

In scope:

- tracing where `tool_config` and `tool_definitions` are written,
- tracing how they survive YAML serialization,
- identifying the best example binary and script pair,
- describing the smallest code change needed to persist a final turn file,
- defining exact assertions for the resulting YAML,
- documenting environment and sandbox caveats discovered during validation.

Out of scope:

- changing provider engines to read persisted tool definitions,
- redesigning the tool loop,
- redesigning the session model,
- changing the underlying persisted payload contract from GP-32.

## What Shipped In The Last Two PRs

Recent Git history in `geppetto/` shows:

- `0131ba2 Persist tool definitions on turns`
- `7fc41b1 Lock tool advertisement to runtime registry`
- `4c375a1 Expose tool definitions in JS codec`
- `8c21682 Sanitize persisted tool examples`
- `eadf1fe Add yaml export tags`
- merge PRs `#297` and `#298`

The implementation ticket for that work is:

- `ttmp/2026/03/10/GP-32-TURN-TOOL-DEFINITIONS--persist-serializable-tool-definitions-on-turn-data/`

That ticket is important context because it states the intent clearly:

- persist inspection-friendly tool definitions on turns,
- keep runtime execution and advertisement on the live registry,
- make YAML round-tripping safe.

## System Map

### Core Terms

`Turn`

- The durable unit of conversation state.
- Contains `Blocks`, `Metadata`, and `Data`.

`Turn.Data`

- Opaque typed-key map for persistable per-turn data.
- YAML serialization uses canonical key IDs like `geppetto.tool_config@v1`, not JS short names.

`ToolRegistry`

- Live runtime registry carrying executable tools and schemas.
- Stored in `context.Context`, not in `Turn.Data`.

`SnapshotHook`

- Callback for intermediate turn phases inside the tool loop.
- Observes in-memory phases such as `pre_inference`, `post_inference`, and `post_tools`.

`TurnPersister`

- Host-owned interface invoked after a successful run completes.
- This is the cleanest place to write the final turn YAML to disk.

### High-Level Flow

```text
JS script / Go host
    |
    v
gp.tools.createRegistry() / useGoTools()
    |
    v
builder.withTools(registry, toolLoopOpts)
    |
    v
session.run() / runner.RunInference()
    |
    v
toolloop.RunLoop()
    |- attach live registry to context
    |- set engine.KeyToolConfig on Turn.Data
    |- set engine.KeyToolDefinitions on Turn.Data
    |- snapshotHook("pre_inference")
    |- engine.RunInference(...)
    |- snapshotHook("post_inference")
    |- execute live tools from context registry
    |- snapshotHook("post_tools")
    `- return completed Turn
    |
    v
enginebuilder runner
    `- persister.PersistTurn(ctx, updatedTurn)
           |
           v
      serde.SaveTurnYAML(...)
           |
           v
      disk file with geppetto.tool_config@v1
      and geppetto.tool_definitions@v1
```

## Current-State Architecture

### 1. `Turn.Data` is an opaque typed-key store

Observed in `pkg/turns/types.go`:

- `Data` is an opaque wrapper over `map[TurnDataKey]any`.
- `MarshalYAML()` serializes each key using `k.String()`.
- `UnmarshalYAML()` keeps raw key strings and lets typed keys decode later.

Relevant files:

- `pkg/turns/types.go`
- `pkg/turns/key_families.go`
- `pkg/turns/keys_gen.go`
- `pkg/inference/engine/turnkeys_gen.go`

Why this matters:

- JS callers see short names such as `tool_config` and `tool_definitions`.
- persisted YAML uses canonical IDs such as `geppetto.tool_config@v1` and `geppetto.tool_definitions@v1`.
- your on-disk assertions must use the canonical keys, not the JS short names.

### 2. The typed keys are generated and package-owned

Observed in `pkg/inference/engine/turnkeys_gen.go`:

- `KeyToolConfig = turns.DataK[ToolConfig](...)`
- `KeyToolDefinitions = turns.DataK[ToolDefinitions](...)`

Observed in `pkg/turns/keys_gen.go`:

- the canonical value keys include `ToolConfigValueKey` and `ToolDefinitionsValueKey`.

Why this matters:

- these are the only supported access paths,
- the YAML key spelling is derived from these generated definitions,
- `turnsdatalint` exists specifically to prevent drift away from typed keys.

### 3. The persisted tool-definition shape is not the runtime tool definition shape

Observed in `pkg/inference/engine/types.go`:

- `ToolDefinition` uses `*jsonschema.Schema` and runtime-oriented fields,
- `ToolDefinitionSnapshot` uses `Parameters map[string]any`,
- `ToolDefinitions` is `[]ToolDefinitionSnapshot`.

This is a critical design decision from GP-32.

Why:

- a raw `*jsonschema.Schema` did not round-trip safely in the typed YAML/JSON decode path,
- a plain `map[string]any` is stable for serialization and inspection,
- persisted payloads are intentionally inspection metadata, not executable registry state.

### 4. The tool loop is where the persistence-relevant data gets stamped

Observed in `pkg/inference/toolloop/loop.go`:

- the live registry is attached to context via `tools.WithRegistry(ctx, l.registry)`,
- `engine.KeyToolConfig.Set(&t.Data, ...)` happens before the loop begins,
- `engine.KeyToolDefinitions.Set(&t.Data, persistedToolDefinitions(...))` also happens before the first engine inference,
- `persistedToolDefinitions(...)` sorts definitions by tool name and sanitizes example payloads.

Why this matters:

- by the time the engine sees the first turn, both persisted fields are already present,
- any snapshot or persister that serializes that evolving turn after this point should see them,
- deterministic sorting means on-disk YAML is stable for tests.

### 5. The runtime registry remains authoritative for execution and provider advertisement

Observed in:

- `pkg/inference/tools/advertisement.go`
- `pkg/steps/ai/openai/engine_openai.go`
- `pkg/steps/ai/openai_responses/engine.go`
- `pkg/steps/ai/openai_responses/engine_test.go`
- `pkg/doc/playbooks/01-add-a-new-tool.md`

Concrete rule:

- providers advertise tools from the live registry in `context.Context`,
- execution uses the live registry in `context.Context`,
- persisted `tool_definitions` is inspection-only.

This is not merely documentation. It is covered by tests:

- `TestAttachToolsToResponsesRequest_IgnoresPersistedToolDefinitionsWithoutRuntimeRegistry`
- `TestAttachToolsToResponsesRequest_UsesRuntimeRegistryInsteadOfPersistedSnapshots`

Why this matters:

- if an intern tries to verify persistence by checking provider request payload construction alone, they are testing the wrong layer,
- the correct verification target is persisted turn output, not provider request assembly.

### 6. Session history and host persistence are separate but complementary

Observed in `pkg/inference/session/session.go`:

- session turn history is append-only,
- each new user turn starts as a clone of the latest turn,
- `TurnsSnapshot()` returns cloned snapshots for safe inspection.

Observed in `pkg/inference/toolloop/enginebuilder/builder.go`:

- snapshot hooks are attached to the run context before execution,
- `PersistTurn(ctx, updated)` is invoked after successful completion.

Important distinction:

- snapshot hooks capture intermediate phases,
- the persister handles durable final-turn storage.

This is the main place new readers get confused.

### 7. The JS module and example harness already expose the host hooks

Observed in:

- `pkg/js/modules/geppetto/module.go`
- `pkg/js/modules/geppetto/api_sessions.go`
- `pkg/js/modules/geppetto/api_builder_options.go`
- `pkg/doc/topics/13-js-api-reference.md`
- `pkg/doc/topics/14-js-api-user-guide.md`

The JS-facing builder already supports:

- `withPersister(persister)`
- `withSnapshotHook(snapshotHook)`
- module defaults via `gp.Options.DefaultPersister`
- module defaults via `gp.Options.DefaultSnapshotHook`

This means the example binary does not need a JS API redesign. It only needs to supply concrete Go hook implementations.

## Snapshotting vs Persistence: The Exact Mental Model

Use this table when explaining the system to an intern.

| Layer | Owner | Purpose | Carries live executors? | Should be on disk? |
|---|---|---|---|---|
| `tools.ToolRegistry` in context | runtime | execution + provider advertisement | yes | no |
| `Turn.Data.geppetto.tool_config@v1` | turn | durable tool policy snapshot | no | yes |
| `Turn.Data.geppetto.tool_definitions@v1` | turn | durable tool schema snapshot | no | yes |
| `SnapshotHook` callbacks | host | observe intermediate phases | no | optional |
| `TurnPersister` | host | persist final completed turn | no | yes |

One nuance worth calling out explicitly:

- `pkg/doc/topics/08-turns.md` currently describes a `final` snapshot phase in a table.
- the code in `pkg/inference/toolloop/context.go` and `pkg/inference/toolloop/loop.go` only documents and emits `pre_inference`, `post_inference`, and `post_tools`.
- the final durable write today happens through `TurnPersister`, not through a built-in `"final"` snapshot callback.

For this ticket, treat the persister as the final-truth disk writer.

## Why `geppetto-js-lab` Is The Best Example Binary

Candidate binaries considered:

### `cmd/examples/geppetto-js-lab`

Pros:

- already the documented JS harness in `README.md`,
- already runs scripts under `examples/js/geppetto/`,
- already includes built-in Go tool registry setup,
- already uses `gp.Options`, where `DefaultPersister` and `DefaultSnapshotHook` exist,
- can run with `gp.engines.fromFunction(...)`, so no live model credentials are needed.

Cons:

- today it does not persist turns by default,
- it needs a small host-side CLI flag and hook implementation.

### `cmd/examples/generic-tool-calling`

Pros:

- pure Go example, direct tool-oriented example.

Cons:

- less aligned with the JS example scripts the repository already promotes,
- more provider/runtime coupling,
- not as convenient for reusing the existing deterministic JS scripts.

### `cmd/examples/openai-tools` / `cmd/examples/claude-tools`

Pros:

- directly exercise provider tool advertisement.

Cons:

- unnecessarily dependent on provider setup for this verification goal,
- harder to make deterministic,
- the verification target is persisted turn YAML, not provider network behavior.

Recommendation:

- use `cmd/examples/geppetto-js-lab` as the host harness,
- use `examples/js/geppetto/05_go_tools_from_js.js` as the primary script,
- keep `examples/js/geppetto/04_tools_and_toolloop.js` as the minimal secondary smoke script.

## Why `05_go_tools_from_js.js` Is The Best Script

`05_go_tools_from_js.js` is better than `04_tools_and_toolloop.js` for this ticket because:

1. it imports built-in Go tools from the host registry,
2. those tools are defined from typed Go structs in `cmd/examples/geppetto-js-lab/main.go`,
3. `tools.NewToolFromFunc(...)` generates real JSON Schema from those structs,
4. the persisted `tool_definitions.parameters` is therefore richer and easier to assert.

For example:

- `go_double` is backed by `goDoubleInput { N int \`json:"n" jsonschema:"required,..."\` }`
- `go_concat` is backed by `goConcatInput { A string ...; B string ... }`

This gives you concrete YAML assertions like:

- `parameters.type == "object"`
- `parameters.properties.n.type == "integer"`
- `required` includes `n`

That is a better proof than a generic `map[string]any` JS tool schema.

## Observed Local Validation

### Successful example runs

Command:

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go run ./cmd/examples/geppetto-js-lab \
  --script examples/js/geppetto/04_tools_and_toolloop.js
```

Observed result:

- completed successfully,
- printed `PASS: 04_tools_and_toolloop.js`.

Command:

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go run ./cmd/examples/geppetto-js-lab \
  --script examples/js/geppetto/05_go_tools_from_js.js
```

Observed result:

- completed successfully,
- printed `PASS: 05_go_tools_from_js.js`.

### Successful focused tests

Command:

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go test ./pkg/inference/toolloop ./pkg/turns/serde ./pkg/inference/tools
```

Observed result:

- all three packages passed.

### Full package retry after unrestricted access

Command:

```bash
GOWORK=off \
  go test ./pkg/steps/ai/openai_responses
```

Observed result:

- the package passed cleanly once rerun outside the original sandbox restriction.

Interpretation:

- the earlier `httptest.NewTLSServer` failure was a sandbox artifact, not a product failure,
- the persistence-relevant focused packages and the provider-specific `openai_responses` package now all have successful local verification,
- the real day-to-day local footgun remains the `go.work` version mismatch unless commands are run with `GOWORK=off`.

### Verified YAML key spelling

Command:

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go test ./pkg/turns/serde -run TestYAMLRoundTripTypedMaps -v
```

Observed YAML excerpt:

```yaml
data:
  geppetto.tool_config@v1:
    enabled: true
    tool_choice: auto
    max_iterations: 4
  geppetto.tool_definitions@v1:
    - name: echo
      parameters:
        properties:
          text:
            type: string
```

This is the strongest concrete evidence for the on-disk key spelling and payload shape.

## Proposed Verification Design

### Goal

Augment `cmd/examples/geppetto-js-lab` so an intern can run one existing JS example and obtain:

- one final persisted turn YAML file,
- optionally several phase snapshot YAML files,
- deterministic assertions over `geppetto.tool_config@v1` and `geppetto.tool_definitions@v1`.

### Minimal host changes

Add two CLI flags to `cmd/examples/geppetto-js-lab/main.go`:

1. `--persist-turn-dir <path>`
2. `--persist-snapshots` (optional)

Then add two small host implementations:

1. `yamlTurnPersister`
2. `yamlSnapshotWriter`

These are host utilities, not Geppetto core changes.

### Recommended file outputs

Use a directory structure like:

```text
/tmp/gp-save-tool-config/
  final_turn.yaml
  snapshots/
    01-pre_inference.yaml
    02-post_inference.yaml
    03-post_tools.yaml
```

This makes it obvious which file answers which question:

- `final_turn.yaml` proves durable persistence,
- `snapshots/*.yaml` proves the fields were present during intermediate phases.

## Proposed Host-Side Implementation

### `yamlTurnPersister`

Purpose:

- persist the final completed turn after successful execution.

Pseudocode:

```go
type yamlTurnPersister struct {
    outDir string
}

func (p *yamlTurnPersister) PersistTurn(ctx context.Context, t *turns.Turn) error {
    if t == nil {
        return nil
    }
    if err := os.MkdirAll(p.outDir, 0o755); err != nil {
        return err
    }
    path := filepath.Join(p.outDir, "final_turn.yaml")
    return serde.SaveTurnYAML(path, t, serde.Options{})
}
```

### `yamlSnapshotWriter`

Purpose:

- write one YAML file per snapshot phase.

Pseudocode:

```go
type yamlSnapshotWriter struct {
    dir string
    seq atomic.Int64
}

func (w *yamlSnapshotWriter) Hook(ctx context.Context, t *turns.Turn, phase string) {
    if t == nil {
        return
    }
    n := w.seq.Add(1)
    _ = os.MkdirAll(w.dir, 0o755)
    path := filepath.Join(w.dir, fmt.Sprintf("%02d-%s.yaml", n, phase))
    clone := t.Clone()
    _ = serde.SaveTurnYAML(path, clone, serde.Options{})
}
```

Recommendation:

- clone before writing, even though the hook is observational,
- do not block the main run on snapshot write failures unless you explicitly want test-hard failure semantics.

### Wiring into `geppetto-js-lab`

Pseudocode:

```go
var persistTurnDir = flag.String("persist-turn-dir", "", "Write final turn YAML to this directory")
var persistSnapshots = flag.Bool("persist-snapshots", false, "Write phase snapshot YAML files")

var defaultPersister enginebuilder.TurnPersister
var defaultSnapshotHook toolloop.SnapshotHook

if dir := strings.TrimSpace(*persistTurnDir); dir != "" {
    defaultPersister = &yamlTurnPersister{outDir: dir}
    if *persistSnapshots {
        writer := &yamlSnapshotWriter{dir: filepath.Join(dir, "snapshots")}
        defaultSnapshotHook = writer.Hook
    }
}

rt, err := jsruntime.NewRuntime(ctx, jsruntime.Options{
    ModuleOptions: gp.Options{
        GoToolRegistry:      goRegistry,
        ProfileRegistry:     profileRegistry,
        ProfileRegistryWriter: profileRegistryWriter,
        MiddlewareSchemas:   middlewareSchemas,
        ExtensionCodecs:     extensionCodecs,
        ExtensionSchemas:    extensionSchemas,
        DefaultPersister:    defaultPersister,
        DefaultSnapshotHook: defaultSnapshotHook,
    },
})
```

No JS script changes are required if you use module defaults.

## Recommended End-to-End Runbook

### Step 1. Build confidence with the existing example first

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go run ./cmd/examples/geppetto-js-lab \
  --script examples/js/geppetto/05_go_tools_from_js.js
```

Expected:

- the script prints `PASS: 05_go_tools_from_js.js`.

### Step 2. Add the host persistence flags and hook implementations

Change only:

- `cmd/examples/geppetto-js-lab/main.go`

Use:

- `pkg/turns/serde/serde.go`
- `pkg/inference/toolloop/enginebuilder/builder.go`
- `pkg/js/modules/geppetto/module.go`

as the integration references.

### Step 3. Run with persistence enabled

```bash
rm -rf /tmp/gp-save-tool-config

GOWORK=off GOCACHE=/tmp/geppetto-go-build \
  go run ./cmd/examples/geppetto-js-lab \
  --script examples/js/geppetto/05_go_tools_from_js.js \
  --persist-turn-dir /tmp/gp-save-tool-config \
  --persist-snapshots
```

### Step 4. Inspect the persisted YAML

```bash
sed -n '1,240p' /tmp/gp-save-tool-config/final_turn.yaml
ls -1 /tmp/gp-save-tool-config/snapshots
```

### Step 5. Assert the exact durable fields

The final YAML should contain:

- `data.geppetto.tool_config@v1`
- `data.geppetto.tool_definitions@v1`

For `05_go_tools_from_js.js`, expected useful assertions are:

1. `geppetto.tool_config@v1.enabled == true`
2. `geppetto.tool_config@v1.max_iterations == 3`
3. `geppetto.tool_config@v1.tool_choice == auto`
4. `len(geppetto.tool_definitions@v1) == 2`
5. tool definition names are sorted alphabetically: `go_concat`, `go_double`
6. `go_double.parameters.type == object`
7. `go_double.parameters.properties.n.type == integer`
8. `go_concat.parameters.properties.a.type == string`
9. `go_concat.parameters.properties.b.type == string`

### Step 6. Inspect the phase snapshots

What you should see:

- `01-pre_inference.yaml` already contains both `tool_config` and `tool_definitions`,
- `02-post_inference.yaml` adds the tool call block,
- `03-post_tools.yaml` adds the tool result block.

That proves the fields are stamped before the first engine inference, not retrofitted only at the end.

## Acceptance Criteria

The verification work is complete when all of the following are true:

1. `geppetto-js-lab` can persist a final turn YAML file without changing the JS example script.
2. The persisted file contains `geppetto.tool_config@v1`.
3. The persisted file contains `geppetto.tool_definitions@v1`.
4. The tool-definition payload reflects the runtime registry tools, not only the executed tool call.
5. Optional phase snapshots show the fields already present at `pre_inference`.
6. The example remains deterministic and does not depend on external APIs.

## Design Decisions

### Decision 1: Use `geppetto-js-lab`, not a provider-backed example

Reason:

- the persistence contract lives below the provider transport layer,
- deterministic local execution is more important than provider realism here.

### Decision 2: Prefer `05_go_tools_from_js.js` over `04_tools_and_toolloop.js`

Reason:

- richer schema assertions,
- still deterministic,
- exercises both direct Go tool import and tool-loop execution.

### Decision 3: Use `TurnPersister` for the final durable proof

Reason:

- `TurnPersister` is the existing post-success disk-write seam,
- it avoids ambiguity about whether snapshot hooks represent final truth,
- it matches the builder architecture already in `enginebuilder`.

### Decision 4: Keep snapshot files optional

Reason:

- final-turn persistence is the core requirement,
- intermediate snapshots are useful for debugging and for proving stamp timing,
- optional behavior keeps the example simple for normal use.

## Alternatives Considered

### Alternative A: Add a brand new test-only example binary

Rejected because the repository already promotes `geppetto-js-lab` as the JS example harness, and adding a second near-duplicate host would create documentation drift.

### Alternative B: Use `04_tools_and_toolloop.js` as the only fixture

Rejected as the primary choice because the JS-defined tool path does not produce as rich a schema contract as the Go tool path.

It remains a good secondary smoke script.

### Alternative C: Verify only via unit tests

Rejected because the ticket goal explicitly asks for end-to-end persistence through the snapshot/persister path and for a cmd/example-based validation workflow.

### Alternative D: Persist snapshots only, with no final-turn persister

Rejected because snapshots are observational and phase-based. The core proof needed here is that the completed turn written to disk contains the expected fields.

## Risks And Sharp Edges

### 1. `go.work` version mismatch in this workspace

Observed locally:

- `go.work` says `go 1.25`
- `geppetto/go.mod` requires `go 1.25.8`
- sibling modules require even newer patch versions

Workaround used during validation:

```bash
GOWORK=off GOCACHE=/tmp/geppetto-go-build ...
```

Document this in the runbook so the intern does not waste time on a false build failure.

### 2. On-disk YAML keys are canonical, not JS-short

If the intern greps for `tool_definitions:` in the persisted file, they may miss the real key and incorrectly conclude the feature is broken.

Correct key:

- `geppetto.tool_definitions@v1`

### 3. Snapshot hooks are not the same as a final durable write

If someone treats `snapshotHook` as the only persistence path, they may accidentally verify an intermediate file and skip the final completed turn.

## Open Questions

1. Should `geppetto-js-lab` keep the persistence flags permanently after this ticket, or are they only for verification work?
2. Should the example write snapshots synchronously and fail hard on I/O errors, or should snapshot writing be best-effort while final persistence remains strict?
3. Is the documentation table in `pkg/doc/topics/08-turns.md` that mentions a `final` snapshot phase intentionally aspirational, or should it be aligned to the current code that only emits `pre_inference`, `post_inference`, and `post_tools`?

## Implementation Plan

### Phase 1. Add host persistence hooks to `geppetto-js-lab`

Files:

- `cmd/examples/geppetto-js-lab/main.go`

Steps:

1. add CLI flags for output directory and optional snapshots,
2. add `yamlTurnPersister`,
3. add optional `yamlSnapshotWriter`,
4. thread them into `gp.Options.DefaultPersister` and `gp.Options.DefaultSnapshotHook`.

### Phase 2. Run the existing JS example as the fixture

Files:

- `examples/js/geppetto/05_go_tools_from_js.js`

Steps:

1. run the script without persistence first,
2. run again with persistence enabled,
3. inspect `final_turn.yaml` and snapshot files.

### Phase 3. Add a focused regression test if desired

Optional files:

- `cmd/examples/geppetto-js-lab/main_test.go` or a small focused harness test if this example gains enough host logic to justify it.

Possible assertions:

1. final YAML file exists,
2. canonical keys exist,
3. expected schema properties are present.

## References

Primary code references:

- `pkg/inference/engine/types.go`
- `pkg/inference/engine/turnkeys_gen.go`
- `pkg/inference/toolloop/loop.go`
- `pkg/inference/toolloop/loop_test.go`
- `pkg/inference/toolloop/context.go`
- `pkg/inference/toolloop/enginebuilder/builder.go`
- `pkg/inference/toolloop/enginebuilder/options.go`
- `pkg/inference/tools/advertisement.go`
- `pkg/inference/tools/definition.go`
- `pkg/turns/types.go`
- `pkg/turns/key_families.go`
- `pkg/turns/serde/serde.go`
- `pkg/turns/serde/serde_test.go`
- `pkg/js/modules/geppetto/module.go`
- `pkg/js/modules/geppetto/api_sessions.go`
- `pkg/js/modules/geppetto/api_builder_options.go`
- `pkg/js/modules/geppetto/codec.go`
- `pkg/js/modules/geppetto/module_test.go`
- `cmd/examples/geppetto-js-lab/main.go`
- `examples/js/geppetto/04_tools_and_toolloop.js`
- `examples/js/geppetto/05_go_tools_from_js.js`

Primary documentation references:

- `README.md`
- `pkg/doc/topics/08-turns.md`
- `pkg/doc/topics/10-sessions.md`
- `pkg/doc/topics/13-js-api-reference.md`
- `pkg/doc/topics/14-js-api-user-guide.md`
- `pkg/doc/playbooks/01-add-a-new-tool.md`

Upstream implementation ticket:

- `ttmp/2026/03/10/GP-32-TURN-TOOL-DEFINITIONS--persist-serializable-tool-definitions-on-turn-data/`

## Problem Statement

<!-- Describe the problem this design addresses -->

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
