---
Title: 'Migration Analysis: Old Glazed to Facade Packages (Geppetto then Pinocchio)'
Ticket: GP-001-UPDATE-GLAZED
Status: active
Topics:
    - migration
    - glazed
    - geppetto
    - pinocchio
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/layers/layers.go
      Note: Highest-impact geppetto migration hotspot
    - Path: geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/09-count-breakdown.txt
      Note: Quantified migration surface and top symbol counts
    - Path: geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/14-failure-extracts.txt
      Note: Consolidated build/lint failure evidence
    - Path: glazed/pkg/cli/cobra-parser.go
      Note: Cobra middleware signature and parse flow used in migration planning
    - Path: glazed/pkg/cmds/cmds.go
      Note: Authoritative command/runtime interfaces now based on values
    - Path: glazed/pkg/doc/tutorials/migrating-to-facade-packages.md
      Note: Primary migration playbook analyzed
    - Path: pinocchio/pkg/cmds/helpers/parse-helpers.go
      Note: Highest-impact pinocchio middleware migration hotspot
ExternalSources: []
Summary: Exhaustive migration analysis and execution plan for moving geppetto first, then pinocchio, from legacy glazed layers/parameters/middlewares APIs to schema/fields/values/sources facade packages.
LastUpdated: 2026-02-12T08:20:10.684228939-05:00
WhatFor: Migration planning, risk management, and implementation sequencing for the glazed facade package transition across both repositories.
WhenToUse: Use before and during implementation to drive file-by-file migration, verify expected breakages, and validate completion criteria.
---


# Migration Analysis: Old Glazed to Facade Packages (Geppetto then Pinocchio)

## Goal

Produce an implementation-ready migration plan from legacy Glazed APIs (`layers`, `parameters`, `cmds/middlewares`, `ParsedLayers`) to the current facade APIs (`schema`, `fields`, `cmds/sources`, `values`) with exact filename/symbol impact for:

1. `geppetto/` first (foundation)
2. `pinocchio/` second (depends on geppetto)

## Scope and Inputs

Primary migration guidance reviewed:

- `glazed/pkg/doc/tutorials/migrating-to-facade-packages.md`

Actual API surface validated against current code in this workspace:

- `glazed/pkg/cmds/cmds.go`
- `glazed/pkg/cmds/schema/schema.go`
- `glazed/pkg/cmds/schema/section-impl.go`
- `glazed/pkg/cmds/values/section-values.go`
- `glazed/pkg/cmds/sources/cobra.go`
- `glazed/pkg/cmds/sources/update.go`
- `glazed/pkg/cmds/sources/load-fields-from-config.go`
- `glazed/pkg/cmds/sources/whitelist.go`
- `glazed/pkg/settings/glazed_section.go`

Automated inventories and failure captures for this ticket:

- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/00-counts.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/01-legacy-imports.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/02-legacy-symbol-usage.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/03-legacy-tags.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/04-signature-hotspots.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/05-geppetto-make-test.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/06-pinocchio-make-test.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/07-geppetto-make-lint.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/08-pinocchio-make-lint.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/09-count-breakdown.txt`

## Baseline Status (Before Migration)

### Hard blockers from current build/test

`make test` and `make lint` fail in both repos. Primary blocker is removed Glazed legacy packages:

- `github.com/go-go-golems/glazed/pkg/cmds/layers`
- `github.com/go-go-golems/glazed/pkg/cmds/parameters`
- `github.com/go-go-golems/glazed/pkg/cmds/middlewares`

Evidence:

- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/05-geppetto-make-test.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/07-geppetto-make-lint.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/06-pinocchio-make-test.txt`
- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/08-pinocchio-make-lint.txt`

### Quantified migration surface

From `00-counts.txt` + `09-count-breakdown.txt`:

- Legacy import occurrences (code/docs, excluding `ttmp/`): `92`
- Legacy symbol occurrences in code: `114`
- Legacy tag occurrences (`glazed.parameter`, `glazed.layer`, etc.): `221`
- Signature hotspots (`ParsedLayers`, `ParameterLayer`, etc.): `91`
- Legacy imports by repo: geppetto `40`, pinocchio `52`
- Legacy tags by repo: geppetto `111`, pinocchio `110`

## Confirmed API Delta (What Must Change)

### Command/runtime API

Old runtime function signatures using `*layers.ParsedLayers` must become `*values.Values`.

- Old:
  - `Run(ctx context.Context, parsed *layers.ParsedLayers) error`
  - `RunIntoWriter(ctx context.Context, parsed *layers.ParsedLayers, w io.Writer) error`
  - `RunIntoGlazeProcessor(ctx context.Context, parsed *layers.ParsedLayers, gp middlewares.Processor) error`
- New (`glazed/pkg/cmds/cmds.go`):
  - `Run(ctx context.Context, parsedValues *values.Values) error`
  - `RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error`
  - `RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error`

### Command description / schema assembly

- Old `cmds.WithLayersList(...)` and layer-based APIs are removed.
- New:
  - `cmds.WithSections(...)`
  - `cmds.WithSchema(...)`
  - `cmds.WithSectionsMap(...)`
  - `description.Schema` is `*schema.Schema`

### Package replacements

- `pkg/cmds/layers` -> `pkg/cmds/schema`
- `pkg/cmds/parameters` -> `pkg/cmds/fields`
- `pkg/cmds/middlewares` -> `pkg/cmds/sources`
- parsed values move from `layers.ParsedLayers` to `values.Values`

### Middleware/source names

Common direct renames:

- `ParseFromCobraCommand` -> `FromCobra`
- `GatherArguments` -> `FromArgs`
- `UpdateFromEnv` -> `FromEnv`
- `SetFromDefaults` -> `FromDefaults`
- `LoadParametersFromFile` -> `FromFile`
- `LoadParametersFromFiles` -> `FromFiles`
- `ExecuteMiddlewares` -> `Execute`
- `WrapWithWhitelistedLayers` -> `WrapWithWhitelistedSections`

### Struct tags

Current field decoding in `glazed/pkg/cmds/fields/initialize-struct.go` reads only `glazed:"..."` tags.

- `glazed.parameter:"..."` must become `glazed:"..."`
- `glazed.layer:"..."` usage should be removed/replaced by explicit section decode logic
- `glazed.default` and `glazed.help` metadata-style tags are legacy; defaults/help must live in section/field definitions

## Geppetto Migration Plan (Do First)

Reason: pinocchio imports geppetto settings and middleware assembly. Geppetto must expose stable new-API entrypoints before pinocchio can complete.

### G1. Foundation: migrate geppetto settings/layers package to schema/fields/sources/values

Primary hotspot:

- `geppetto/pkg/layers/layers.go`

Required actions:

1. Change imports:
- `cmds/layers` -> `cmds/schema`
- `cmds/middlewares` -> `cmds/sources`
- `cmds/parameters` -> `cmds/fields`
- parsed command section type from `*layers.ParsedLayers` -> `*values.Values`

2. Rewrite `CreateGeppettoLayers` to return sections:
- old: `[]cmdlayers.ParameterLayer`
- new: `[]schema.Section`

3. Rewrite `GetCobraCommandGeppettoMiddlewares` signature:
- old: `func(parsedCommandLayers *cmdlayers.ParsedLayers, cmd *cobra.Command, args []string) ([]middlewares.Middleware, error)`
- new: `func(parsedCommandValues *values.Values, cmd *cobra.Command, args []string) ([]sources.Middleware, error)`

4. Replace middleware chain construction with sources equivalents:
- `sources.FromCobra`, `sources.FromArgs`, `sources.FromEnv`, `sources.FromFiles`, `sources.FromDefaults`, `sources.Execute`
- apply section whitelist with `sources.WrapWithWhitelistedSections`

5. Replace old parse options:
- `parameters.WithParseStepSource(...)` -> `fields.WithSource(...)`
- `parameters.WithParseStepMetadata(...)` -> `fields.WithMetadata(...)`

6. Replace struct decode calls:
- `parsed.InitializeStruct(slug, target)` -> `parsedValues.DecodeSectionInto(slug, target)`

### G2. Migrate all geppetto settings section definitions

Files:

- `geppetto/pkg/embeddings/config/settings.go`
- `geppetto/pkg/steps/ai/settings/settings-chat.go`
- `geppetto/pkg/steps/ai/settings/settings-client.go`
- `geppetto/pkg/steps/ai/settings/openai/settings.go`
- `geppetto/pkg/steps/ai/settings/claude/settings.go`
- `geppetto/pkg/steps/ai/settings/gemini/settings.go`
- `geppetto/pkg/steps/ai/settings/ollama/settings.go`

Required actions:

1. Replace custom `*ParameterLayer` wrappers with section wrappers (`*schema.SectionImpl` embedding or plain `schema.Section`).
2. Rename constructors from `New...ParameterLayer` to `New...Section` (or keep exported names with new return type if minimizing downstream diffs).
3. Replace defaults init:
- `InitializeStructFromParameterDefaults` -> `InitializeStructFromFieldDefaults`
4. Replace all struct tags:
- `glazed.parameter:"x"` -> `glazed:"x"`

### G3. Migrate step-settings decode helpers and engine helpers

Files:

- `geppetto/pkg/steps/ai/settings/settings-step.go`
- `geppetto/pkg/embeddings/settings_factory.go`
- `geppetto/pkg/inference/engine/factory/helpers.go`

Required actions:

- Replace `*layers.ParsedLayers` inputs with `*values.Values`.
- Replace `InitializeStruct(...)` calls with `DecodeSectionInto(...)`.
- Keep explicit per-section decode behavior; do not rely on legacy `glazed.layer` tags.

### G4. Migrate geppetto command entrypoints/examples

Files (runtime signatures and layer wiring):

- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`
- `geppetto/cmd/examples/claude-tools/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/llm-runner/main.go`
- `geppetto/cmd/llm-runner/serve.go`

Required actions:

- Replace imports (`layers`,`parameters`) with (`values/schema`,`fields`).
- Replace command assembly `cmds.WithLayersList(...)` -> `cmds.WithSections(...)`.
- Replace `Run/RunIntoWriter` signatures to `*values.Values`.
- Replace decode calls to `DecodeSectionInto(schema.DefaultSlug, &settings)`.
- Update command-specific setting struct tags from `glazed.parameter` to `glazed`.

### G5. Geppetto docs/examples cleanup

Files:

- `geppetto/pkg/doc/topics/06-embeddings.md`
- `geppetto/pkg/doc/topics/06-inference-engines.md`

Action:

- Update tutorial snippets to avoid teaching removed APIs.

## Pinocchio Migration Plan (After Geppetto)

### P1. Migrate pinocchio command-model integration

Files:

- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/pkg/cmds/cobra.go`
- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- `pinocchio/pkg/cmds/cmdlayers/helpers.go`

Required actions:

1. Replace type fields in `PinocchioCommandDescription`:
- `[]layers.ParameterLayer` -> `[]schema.Section`
- `[]*parameters.ParameterDefinition` -> `[]*fields.Definition`

2. Replace schema augmentation:
- `description.Layers.PrependLayers(...)` -> `description.Schema.PrependSections(...)`

3. Replace default variables extraction logic:
- old path via `GetDefaultParameterLayer().Parameters.ToMap()` must become `parsedValues.DefaultSectionValues().Fields.ToMap()` or explicit section decode.

4. Migrate helper middleware assembly to `sources` APIs and new middleware function signature:
- input should be `*values.Values`, return `[]sources.Middleware`.

5. Convert helper settings tags to `glazed:"..."`.

### P2. Migrate pinocchio command implementations (all command classes)

Main command files with legacy imports/signatures:

- `pinocchio/cmd/pinocchio/cmds/openai/openai.go`
- `pinocchio/cmd/pinocchio/cmds/openai/transcribe.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/decode.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/encode.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/list.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/enrich.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/fastgpt.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/summarize.go`
- `pinocchio/cmd/pinocchio/cmds/clip.go`
- `pinocchio/cmd/pinocchio/cmds/helpers/md-extract.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/stats.go`
- `pinocchio/cmd/pinocchio/cmds/catter/catter.go`

Required actions:

- `cmds.WithLayersList` -> `cmds.WithSections`.
- settings/new glazed section constructor:
  - `settings.NewGlazedParameterLayers()` -> `settings.NewGlazedSection()`.
- `*layers.ParsedLayers` -> `*values.Values` in `Run*` methods.
- `InitializeStruct` -> `DecodeSectionInto`.
- tags `glazed.parameter` -> `glazed`.
- catter middleware function: sources API replacements.

### P3. Migrate webchat + redis settings wiring

Files:

- `pinocchio/pkg/redisstream/redis_layer.go`
- `pinocchio/pkg/webchat/types.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/cmd/web-chat/main.go`

Required actions:

- convert redis layer construction to section+fields.
- convert router/server constructors from `*layers.ParsedLayers` to `*values.Values`.
- convert all decode and tag usage accordingly.

### P4. Update examples and root command wiring

Files:

- `pinocchio/cmd/examples/simple-chat/main.go`
- `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go`
- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- `pinocchio/cmd/pinocchio/main.go`

Action:

- update middleware hook signatures and command description composition to schema/values APIs.

## Ordered Implementation Sequence (Recommended)

1. Geppetto settings section definitions and tags (`pkg/steps/ai/settings/*`, `pkg/embeddings/config/settings.go`).
2. Geppetto middleware orchestration (`pkg/layers/layers.go`) to sources/values.
3. Geppetto runtime consumers (`settings-step`, `engine/factory`, `embeddings/settings_factory`).
4. Geppetto commands/examples to new signatures and section composition.
5. `make test` and `make lint` in `geppetto/`.
6. Pinocchio command-model core (`pkg/cmds/*`) to values/schema/sources.
7. Pinocchio command implementations (`cmd/pinocchio/cmds/*`, `cmd/web-chat`, examples).
8. Pinocchio webchat/redis packages.
9. `make test` and `make lint` in `pinocchio/`.
10. Final doc/example cleanup and re-run full workspace validations.

## Validation Strategy

Per phase:

- `make test` in changed repo
- `make lint` in changed repo
- targeted smoke run of at least:
  - one geppetto example command
  - one pinocchio command from `cmd/pinocchio/cmds`
  - `cmd/web-chat` startup parse path

End-state grep checks:

- no `pkg/cmds/layers` imports (outside historical `ttmp/`)
- no `pkg/cmds/parameters` imports
- no `pkg/cmds/middlewares` imports
- no `glazed.parameter`/`glazed.layer` tags in active code

## Risks and Non-Migration Blockers

Observed in pinocchio baseline that are not directly glazed-facade migration, but will still fail CI:

- missing geppetto packages in this workspace branch:
  - `github.com/go-go-golems/geppetto/pkg/inference/toolhelpers`
  - `github.com/go-go-golems/geppetto/pkg/inference/toolcontext`
  - `github.com/go-go-golems/geppetto/pkg/conversation`
- `pkg/middlewares/agentmode/middleware.go` references `RunID` fields not present in current event/turn types.

Recommendation: keep these as separate follow-up tickets so migration PR scope stays clear.

## High-Value Mechanical Rewrite Rules

Use these broad rewrites first to shrink compile errors quickly:

1. Imports
- `cmds/layers` -> `cmds/schema` + `cmds/values` (as needed)
- `cmds/parameters` -> `cmds/fields`
- `cmds/middlewares` -> `cmds/sources`

2. Runtime signatures
- `*layers.ParsedLayers` -> `*values.Values`
- `parsed.InitializeStruct(slug, &x)` -> `parsed.DecodeSectionInto(slug, &x)`

3. Command description
- `cmds.WithLayersList(a...)` -> `cmds.WithSections(a...)`

4. Sources
- `ParseFromCobraCommand` -> `FromCobra`
- `GatherArguments` -> `FromArgs`
- `UpdateFromEnv` -> `FromEnv`
- `SetFromDefaults` -> `FromDefaults`
- `LoadParametersFromFile(s)` -> `FromFile` / `FromFiles`
- `ExecuteMiddlewares` -> `Execute`

5. Tags
- `glazed.parameter:"x"` -> `glazed:"x"`

## Appendix A: Exhaustive Geppetto Legacy-Import File Inventory

- `geppetto/cmd/examples/claude-tools/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/llm-runner/main.go`
- `geppetto/cmd/llm-runner/serve.go`
- `geppetto/pkg/doc/topics/06-embeddings.md`
- `geppetto/pkg/doc/topics/06-inference-engines.md`
- `geppetto/pkg/embeddings/config/settings.go`
- `geppetto/pkg/embeddings/settings_factory.go`
- `geppetto/pkg/inference/engine/factory/helpers.go`
- `geppetto/pkg/inference/engine/factory/helpers_test.go`
- `geppetto/pkg/layers/layers.go`
- `geppetto/pkg/steps/ai/settings/claude/settings.go`
- `geppetto/pkg/steps/ai/settings/gemini/settings.go`
- `geppetto/pkg/steps/ai/settings/ollama/settings.go`
- `geppetto/pkg/steps/ai/settings/openai/settings.go`
- `geppetto/pkg/steps/ai/settings/settings-chat.go`
- `geppetto/pkg/steps/ai/settings/settings-client.go`
- `geppetto/pkg/steps/ai/settings/settings-step.go`

## Appendix B: Exhaustive Pinocchio Legacy-Import File Inventory

- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- `pinocchio/cmd/examples/simple-chat/main.go`
- `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go`
- `pinocchio/cmd/pinocchio/cmds/catter/catter.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/stats.go`
- `pinocchio/cmd/pinocchio/cmds/clip.go`
- `pinocchio/cmd/pinocchio/cmds/helpers/md-extract.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/enrich.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/fastgpt.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/summarize.go`
- `pinocchio/cmd/pinocchio/cmds/openai/openai.go`
- `pinocchio/cmd/pinocchio/cmds/openai/transcribe.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/decode.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/encode.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/list.go`
- `pinocchio/cmd/pinocchio/main.go`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/pkg/cmds/cmd.go`
- `pinocchio/pkg/cmds/cmdlayers/helpers.go`
- `pinocchio/pkg/cmds/cobra.go`
- `pinocchio/pkg/cmds/helpers/parse-helpers.go`
- `pinocchio/pkg/cmds/loader.go`
- `pinocchio/pkg/redisstream/redis_layer.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/webchat/types.go`

## Appendix C: Legacy Tag File Inventory (must be converted to `glazed:"..."`)

### Geppetto

- `geppetto/cmd/examples/claude-tools/main.go`
- `geppetto/cmd/examples/generic-tool-calling/main.go`
- `geppetto/cmd/examples/middleware-inference/main.go`
- `geppetto/cmd/examples/openai-tools/main.go`
- `geppetto/cmd/examples/simple-inference/main.go`
- `geppetto/cmd/examples/simple-streaming-inference/main.go`
- `geppetto/cmd/llm-runner/main.go`
- `geppetto/cmd/llm-runner/serve.go`
- `geppetto/pkg/embeddings/config/settings.go`
- `geppetto/pkg/steps/ai/settings/claude/settings.go`
- `geppetto/pkg/steps/ai/settings/ollama/settings.go`
- `geppetto/pkg/steps/ai/settings/openai/settings.go`
- `geppetto/pkg/steps/ai/settings/settings-chat.go`
- `geppetto/pkg/steps/ai/settings/settings-client.go`
- `geppetto/pkg/steps/ai/settings/settings-step.go`

### Pinocchio

- `pinocchio/cmd/agents/simple-chat-agent/main.go`
- `pinocchio/cmd/examples/simple-chat/main.go`
- `pinocchio/cmd/examples/simple-redis-streaming-inference/main.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/print.go`
- `pinocchio/cmd/pinocchio/cmds/catter/cmds/stats.go`
- `pinocchio/cmd/pinocchio/cmds/clip.go`
- `pinocchio/cmd/pinocchio/cmds/helpers/md-extract.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/enrich.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/fastgpt.go`
- `pinocchio/cmd/pinocchio/cmds/kagi/summarize.go`
- `pinocchio/cmd/pinocchio/cmds/openai/openai.go`
- `pinocchio/cmd/pinocchio/cmds/openai/transcribe.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/count.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/decode.go`
- `pinocchio/cmd/pinocchio/cmds/tokens/encode.go`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/pkg/cmds/cmdlayers/helpers.go`
- `pinocchio/pkg/redisstream/redis_layer.go`
- `pinocchio/pkg/webchat/router.go`

## Appendix D: Symbol Hotspot Snapshot

Top-symbol counts used to prioritize migration order are recorded in:

- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/09-count-breakdown.txt`

## Appendix E: Raw Failure Extracts

Compile/lint failure extracts are captured in:

- `geppetto/ttmp/2026/02/12/GP-001-UPDATE-GLAZED--migrate-geppetto-and-pinocchio-to-glazed-facade-packages/sources/local/14-failure-extracts.txt`

