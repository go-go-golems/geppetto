---
Title: Reusable Geppetto Inference Profile Registry Extraction Guide
Ticket: GP-GOJA-API-2026-06-01
Status: active
Topics:
    - geppetto
    - js-bindings
    - goja
    - inference
    - intern-onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/engineprofiles/registry.go
      Note: RegistryReader/Registry interfaces used by JS inferenceProfiles resolver
    - Path: geppetto/pkg/engineprofiles/source_chain.go
      Note: YAML/SQLite source chain support already in Geppetto
    - Path: geppetto/pkg/engineprofiles/types.go
      Note: Core reusable inference profile registry data model already in Geppetto
    - Path: geppetto/pkg/js/modules/geppetto/api_profiles.go
      Note: Current JS profile API to rename/wrap as inferenceProfiles returning Go-owned InferenceSettings
    - Path: pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
      Note: Current Pinocchio registry composition and default selection behavior
    - Path: pinocchio/pkg/configdoc/profiles.go
      Note: Inline profiles to Geppetto registry and composed registry logic to move
    - Path: pinocchio/pkg/configdoc/types.go
      Note: Pinocchio unified profile document shape to extract/genericize
ExternalSources: []
Summary: Analysis and implementation guide for moving the reusable Pinocchio inference profile registry/config-document pieces into Geppetto so goja JS can resolve inference settings without importing Pinocchio.
LastUpdated: 2026-06-01T11:30:00-04:00
WhatFor: Use when implementing Geppetto-owned inference profile registry resolution and exposing it to goja JavaScript.
WhenToUse: Before changing geppetto/pkg/engineprofiles, pinocchio/pkg/configdoc, profile bootstrap code, or the Geppetto JS inferenceProfiles API.
---


# Reusable Geppetto Inference Profile Registry Extraction Guide

## Executive Summary

Yes: in the redesigned JS API, `gp.inferenceSettings()` should return a **Go-owned object**. JavaScript should receive a wrapper with methods such as `.provider(...)`, `.model(...)`, `.credentialRef(...)`, `.temperature(...)`, `.build()`, `.toJSON()`, and `.clone()`. The live settings object should be a Go wrapper around `*settings.InferenceSettings`, not a JavaScript map. When JavaScript needs data, it asks for an explicit snapshot.

The profile registry question is related but separate. The API should expose `gp.inferenceProfiles.resolve("assistant")`, and that call should return a Go-owned `InferenceSettings` wrapper. It should not return a Pinocchio profile, a JavaScript object map, or a full agent preset. Prompt, tools, middleware, event sinks, and tool-loop behavior remain configured through `gp.agent()`.

The surprising finding is that Geppetto already owns most of the reusable registry machinery. `geppetto/pkg/engineprofiles` has typed profile registries, YAML/SQLite stores, chained sources, stack resolution, merge logic, validation, and CLI bootstrap helpers. Pinocchio adds a **unified app config document** layer that can contain:

- app repositories;
- selected active profile;
- imported profile registries;
- inline profile definitions under `profiles:`;
- merge/explain behavior over system/user/repo/cwd/explicit config files.

To avoid pulling Pinocchio into Geppetto JS, move only the reusable inference-profile document/resolution pieces from `pinocchio/pkg/configdoc` into Geppetto. Do **not** move Pinocchio application concepts such as `app.repositories` into the Geppetto JS module contract unless they are generalized and clearly optional. The intended extraction is medium-sized but not a rewrite: about 2-4 days for a careful intern-sized implementation, or 1-2 days for an experienced maintainer, plus tests and docs.

## Problem Statement

We want Geppetto’s goja JavaScript module to support:

```javascript
const settings = gp.inferenceProfiles.resolve("assistant");
const agent = gp.agent()
  .inference(settings)
  .system("Answer briefly.")
  .tool("lookup", t => t.input(schema).handler(fn))
  .build();
```

without importing `github.com/go-go-golems/pinocchio`.

The boundaries are strict:

- `inferenceProfiles` resolves only inference settings.
- `inferenceSettings` is a Go-owned settings object.
- `agent` owns system prompt, tools, middleware, events, and run behavior.
- Geppetto JS forbids raw API keys and environment-variable credential lookup.
- Hosts may provide a default profile resolver backed by Pinocchio profile documents, but Geppetto must not depend on Pinocchio.

## Current Architecture Map

```text
Current reusable core already in Geppetto

geppetto/pkg/engineprofiles
  ├── types.go             EngineProfileRegistry / EngineProfile / refs / metadata
  ├── registry.go          RegistryReader / Registry / ResolvedEngineProfile
  ├── service.go           StoreRegistry: list/get/resolve profiles
  ├── source_chain.go      YAML/SQLite source parsing and chained registry precedence
  ├── stack_resolver.go    stack expansion with cycle/depth validation
  ├── stack_merge.go       inference settings + extension merge
  ├── validation.go        slug/profile/registry validation
  ├── codec_yaml.go        registry YAML codec
  ├── sqlite_store.go      SQLite-backed registry store
  └── memory_store.go      in-memory registry store

Pinocchio-specific reusable-ish layer

pinocchio/pkg/configdoc
  ├── types.go             app/profile/profiles document shape
  ├── load.go              strict YAML decode and presence annotation
  ├── merge.go             config document merge by layer
  ├── resolved.go          load + merge resolved config files
  ├── profiles.go          inline profiles -> Geppetto registry, composed registry
  └── explain.go           provenance/explain entries

Pinocchio CLI wiring

pinocchio/pkg/cmds/profilebootstrap/profile_selection.go
  ├── resolves config files with Geppetto bootstrap plan
  ├── loads Pinocchio config documents
  ├── imports external registry sources through Geppetto bootstrap
  ├── builds inline registry from configdoc profiles
  └── composes imported + inline registries
```

## What Geppetto Already Has

### Typed profile registry model

`geppetto/pkg/engineprofiles/types.go` already defines the core reusable data model:

- `EngineProfileRegistry`
- `EngineProfile`
- `EngineProfileRef`
- metadata structs
- `InferenceSettings *settings.InferenceSettings`
- `Extensions map[string]any`

Despite the current name `EngineProfile`, the payload we need for JS is exactly inference settings. The redesign can rename the JS-facing API to `inferenceProfiles` while leaving internal Go package names alone initially.

### Registry service abstraction

`geppetto/pkg/engineprofiles/registry.go` defines:

```go
type RegistryReader interface {
    ListRegistries(ctx context.Context) ([]RegistrySummary, error)
    GetRegistry(ctx context.Context, registrySlug RegistrySlug) (*EngineProfileRegistry, error)
    ListEngineProfiles(ctx context.Context, registrySlug RegistrySlug) ([]*EngineProfile, error)
    GetEngineProfile(ctx context.Context, registrySlug RegistrySlug, profileSlug EngineProfileSlug) (*EngineProfile, error)
    ResolveEngineProfile(ctx context.Context, in ResolveInput) (*ResolvedEngineProfile, error)
}
```

This is almost the host-facing interface needed by goja JS. The JS module can wrap `ResolvedEngineProfile.InferenceSettings` as an `InferenceSettings` Go wrapper.

### Source chains

`geppetto/pkg/engineprofiles/source_chain.go` already supports source specs such as:

- YAML file paths;
- SQLite files;
- SQLite DSNs;
- comma-separated registry source entries;
- precedence where later sources override earlier sources for profile lookup.

This means Geppetto does not need Pinocchio to load standalone profile registry files.

### CLI bootstrap helpers

`geppetto/pkg/cli/bootstrap/profile_registry.go` and `profile_selection.go` already know how to:

- resolve profile registry source flags/config values;
- build a `ResolvedProfileRegistryChain`;
- compute default resolve input.

This is generic enough for Geppetto applications and should be reused where possible.

## What Pinocchio Adds

### Unified config document shape

`pinocchio/pkg/configdoc/types.go` defines:

```yaml
app:
  repositories:
    - ~/prompts/base
profile:
  active: assistant
  registries:
    - ~/.pinocchio/profiles.yaml
profiles:
  assistant:
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5-mini
```

The reusable parts for Geppetto are:

- `profile.active`
- `profile.registries`
- `profiles.<slug>.inference_settings`
- `profiles.<slug>.stack`
- `profiles.<slug>.extensions`
- strict decode and merge behavior

The Pinocchio/app-specific part is:

- `app.repositories`

For Geppetto JS profile resolution, `app.repositories` is not needed.

### Inline profiles

`pinocchio/pkg/configdoc/profiles.go` converts inline profiles into a Geppetto `EngineProfileRegistry`:

```go
func InlineProfilesToRegistry(doc *Document, registrySlug gepprofiles.RegistrySlug) (*gepprofiles.EngineProfileRegistry, error)
func NewInlineStoreRegistry(doc *Document, registrySlug gepprofiles.RegistrySlug) (*gepprofiles.StoreRegistry, error)
func ComposeRegistry(imported gepprofiles.Registry, inline *gepprofiles.StoreRegistry) gepprofiles.Registry
```

This is exactly the reusable piece we need in Geppetto. It currently lives in Pinocchio only because Pinocchio introduced unified local config documents.

### Layered config file loading

`pinocchio/pkg/cmds/profilebootstrap/profile_selection.go` uses Geppetto bootstrap config plans to find Pinocchio config files across layers:

```text
system app config
home app config
XDG app config
git-root .pinocchio.yml
git-root .pinocchio.override.yml
cwd .pinocchio.yml
cwd .pinocchio.override.yml
explicit file
```

It then:

1. Loads and merges config documents.
2. Reads `profile.active` and `profile.registries`.
3. Imports external registry sources.
4. Builds inline registry from `profiles:`.
5. Composes imported and inline registries.
6. Returns `ResolvedCLIProfileRuntime` containing registry chain and default resolve input.

This is the behavior we want to reuse without depending on Pinocchio.

## Design Recommendation

Create a new Geppetto-owned package, tentatively:

```text
geppetto/pkg/inferenceprofiles
```

or, if keeping everything under the existing package is preferred:

```text
geppetto/pkg/engineprofiles/profiledoc
```

Recommended package naming:

- Internal existing package can remain `engineprofiles` for now to minimize churn.
- New JS-facing and doc-facing names should say `inferenceProfiles` and `InferenceSettings`.
- New reusable config-doc package should avoid `pinocchio` and avoid full-agent semantics.

### Proposed package split

```text
geppetto/pkg/engineprofiles              existing core registry types/stores/resolution
geppetto/pkg/engineprofiles/profiledoc   new reusable config document overlay package
geppetto/pkg/js/modules/geppetto         JS wrappers around inference settings/profile resolution
pinocchio/pkg/configdoc                  thin compatibility alias/adaptor or app-specific wrapper
pinocchio/pkg/cmds/profilebootstrap      uses Geppetto profiledoc + adds Pinocchio app behavior
```

### New Geppetto profile document shape

Keep it focused on inference settings:

```yaml
profile:
  active: assistant
  registries:
    - ~/.config/pinocchio/profiles.yaml
profiles:
  assistant:
    display_name: Assistant
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-5-mini
```

Optional: allow application packages to embed this inside their own document. Pinocchio can keep:

```yaml
app:
  repositories:
    - ~/prompts/base
profile:
  active: assistant
profiles:
  assistant:
    inference_settings: ...
```

but the reusable Geppetto extractor should ignore or delegate `app` rather than own it.

## Proposed API Contracts

### Go package API

```go
package profiledoc

type Document struct {
    Profile  SelectionBlock             `yaml:"profile"`
    Profiles map[string]*InlineProfile  `yaml:"profiles"`
}

type SelectionBlock struct {
    Active     string   `yaml:"active,omitempty"`
    Registries []string `yaml:"registries,omitempty"`
}

type InlineProfile struct {
    DisplayName       string                         `yaml:"display_name,omitempty"`
    Description       string                         `yaml:"description,omitempty"`
    Stack             []engineprofiles.EngineProfileRef `yaml:"stack,omitempty"`
    InferenceSettings *settings.InferenceSettings   `yaml:"inference_settings,omitempty"`
    Extensions        map[string]any                 `yaml:"extensions,omitempty"`
}

func DecodeDocumentWithSource(source string, data []byte) (*Document, error)
func LoadDocument(path string) (*Document, error)
func MergeDocuments(low, high *Document) (*Document, error)
func InlineProfilesToRegistry(doc *Document, registrySlug engineprofiles.RegistrySlug) (*engineprofiles.EngineProfileRegistry, error)
func NewInlineStoreRegistry(doc *Document, registrySlug engineprofiles.RegistrySlug) (*engineprofiles.StoreRegistry, error)
func ComposeRegistry(imported engineprofiles.Registry, inline *engineprofiles.StoreRegistry) engineprofiles.Registry
```

### Resolver API

```go
type ResolverOptions struct {
    ImportedSources []string
    InlineDocument  *profiledoc.Document
    ActiveProfile   engineprofiles.EngineProfileSlug
    InlineRegistrySlug engineprofiles.RegistrySlug
}

type ResolvedInferenceProfileRuntime struct {
    Registry engineprofiles.Registry
    Reader engineprofiles.RegistryReader
    DefaultRegistrySlug engineprofiles.RegistrySlug
    DefaultResolve engineprofiles.ResolveInput
    Close func()
}

func ResolveRuntime(ctx context.Context, opts ResolverOptions) (*ResolvedInferenceProfileRuntime, error)
```

### JS API

```typescript
interface InferenceProfiles {
  listRegistries(): RegistrySummary[];
  list(profileOrRegistry?: string): InferenceProfileSummary[];
  resolve(name?: string | ResolveInferenceInput): InferenceSettings;
  get(name: string, registry?: string): InferenceProfile;
}

interface InferenceSettings {
  provider(): string;
  model(): string;
  clone(): InferenceSettings;
  toJSON(): object;
}
```

Important: `resolve(...)` returns a Go-owned `InferenceSettings` object, not a JS map. `toJSON()` returns a snapshot.

## How `inferenceSettings()` Should Work in JS

```javascript
const settings = gp.inferenceSettings()
  .provider("openai-responses")
  .model("gpt-5-mini")
  .credentialRef("openai-main")
  .temperature(0.2)
  .build();
```

Implementation model:

```text
JS method call
  -> Go wrapper validates argument
  -> Go wrapper mutates *settings.InferenceSettings or builder state
  -> build() returns immutable/copy-on-write InferenceSettingsJS wrapper
  -> engine().inference(settings) accepts only the wrapper/interface
```

Pseudo-Go:

```go
type InferenceSettingsBuilderJS struct {
    api *moduleRuntime
    settings *settings.InferenceSettings
    credentialRef string
}

func (b *InferenceSettingsBuilderJS) Provider(provider string) *InferenceSettingsBuilderJS {
    apiType, err := normalizeProvider(provider)
    if err != nil { panic(b.api.vm.NewTypeError(err.Error())) }
    b.settings.Chat.ApiType = &apiType
    return b
}

func (b *InferenceSettingsBuilderJS) CredentialRef(ref string) *InferenceSettingsBuilderJS {
    if strings.TrimSpace(ref) == "" { panic(b.api.vm.NewTypeError("credentialRef must not be empty")) }
    // Store symbolic ref only. Do not resolve raw secret in JS.
    b.credentialRef = ref
    return b
}

func (b *InferenceSettingsBuilderJS) Build() *InferenceSettingsJS {
    if err := validateInferenceSettingsForJS(b.settings); err != nil { panic(b.api.vm.NewGoError(err)) }
    return &InferenceSettingsJS{api: b.api, settings: b.settings.Clone(), credentialRef: b.credentialRef}
}
```

## Effort Estimate

### Smallest useful extraction: 2-3 days

Move only these from Pinocchio to Geppetto:

- `configdoc.Document` without `AppBlock` or with `AppBlock` ignored/extension-capable;
- strict decode/presence annotation for `profile` and `profiles`;
- `MergeDocuments` for profile selection and inline profiles;
- `InlineProfilesToRegistry`;
- `NewInlineStoreRegistry`;
- `ComposeRegistry`;
- focused tests from Pinocchio.

Then update Geppetto JS to use `engineprofiles.RegistryReader` already available through `Options.EngineProfileRegistry`.

### Recommended robust extraction: 4-6 days

In addition to the smallest extraction:

- Add `profiledoc.ResolveRuntime` for imported + inline composition.
- Add provenance/explain data or preserve a simplified version.
- Add default host-source wiring for Geppetto and Pinocchio.
- Add JS wrappers returning Go-owned `InferenceSettings` objects.
- Add integration tests using fake Pinocchio-style documents without importing Pinocchio.
- Update docs/examples and TypeScript declarations.

### Larger cleanup: 1-2 weeks

If also renaming internal Go `engineprofiles` to `inferenceprofiles`, changing YAML fields, migrating all docs, and updating all call sites, expect a larger effort. This is not recommended for the first pass. Use JS-facing naming first; internal package renaming can happen later.

## Implementation Plan

### Phase 1: Extract reusable profile document package

Create:

```text
geppetto/pkg/engineprofiles/profiledoc
```

Move/adapt from Pinocchio:

- `types.go` minus Pinocchio-only `AppBlock`, or keep app fields behind an extension map.
- `load.go` strict decode/presence annotation for supported fields.
- `merge.go` profile document merge logic.
- `profiles.go` inline registry conversion and composed registry.
- relevant tests from `pinocchio/pkg/configdoc`.

Acceptance criteria:

- Geppetto tests cover inline profile decode, merge, stack resolution, inline-over-imported precedence, imported fallback, and default profile selection.
- Package imports Geppetto and Glazed only; it must not import Pinocchio.

### Phase 2: Add Geppetto-owned resolver composition

Add resolver helper:

```go
func ResolveInferenceProfileRuntime(ctx context.Context, opts ResolveOptions) (*ResolvedRuntime, error)
```

It should:

1. Parse imported registry source specs with `engineprofiles.ParseRegistrySourceSpecs`.
2. Load a chained imported registry with `NewChainedRegistryFromSourceSpecs`.
3. Convert inline document profiles to an inline registry.
4. Compose inline + imported registry with inline precedence.
5. Compute default resolve input from active profile and default registry.
6. Return `RegistryReader` and `Close`.

### Phase 3: Update Pinocchio to consume Geppetto package

Change `pinocchio/pkg/configdoc` to either:

- become a thin wrapper over Geppetto `profiledoc` plus Pinocchio `app.repositories`; or
- keep Pinocchio document loading but call Geppetto `InlineProfilesToRegistry`, `ComposeRegistry`, and resolver pieces.

Best intern-safe approach: first make Pinocchio call the new Geppetto functions while leaving file names and public Pinocchio APIs intact. Delete duplicated Pinocchio logic only after tests pass.

### Phase 4: Add JS wrappers

In `geppetto/pkg/js/modules/geppetto`:

- add `InferenceSettingsJS` wrapper;
- add `InferenceSettingsBuilderJS` wrapper;
- add `inferenceProfiles` namespace wrapper;
- update `engine()` builder to accept only `InferenceSettingsJS` or direct Go settings interfaces;
- remove `apiKey` / `apiKeyEnv` methods from public API.

Example:

```javascript
const settings = gp.inferenceProfiles.resolve("assistant");
const engine = gp.engine().inference(settings).build();
```

### Phase 5: Documentation and tests

Add tests:

- `gp.inferenceSettings()` returns a Go wrapper, not a plain object.
- `settings.toJSON()` returns a serializable snapshot.
- `gp.inferenceProfiles.resolve("assistant")` returns `InferenceSettings` wrapper.
- spreading/cloning JS snapshots does not mutate the Go object.
- `apiKey`, `apiKeyEnv`, and `fromEnv` do not exist.
- profile resolution does not expose system prompt/tools/middleware.

Update docs:

- JS API reference.
- Getting started guide.
- xgoja provider documentation.
- Pinocchio migration notes.

## File-Level Change Guide for Interns

Start reading in this order:

1. `geppetto/pkg/engineprofiles/types.go` — core registry model.
2. `geppetto/pkg/engineprofiles/registry.go` — read/resolve interfaces.
3. `geppetto/pkg/engineprofiles/service.go` — default registry implementation.
4. `geppetto/pkg/engineprofiles/source_chain.go` — imported YAML/SQLite source behavior.
5. `pinocchio/pkg/configdoc/types.go` — current unified document shape.
6. `pinocchio/pkg/configdoc/load.go` — strict YAML decode and presence flags.
7. `pinocchio/pkg/configdoc/merge.go` — overlay merge logic.
8. `pinocchio/pkg/configdoc/profiles.go` — inline profiles -> Geppetto registry.
9. `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go` — current composition behavior.
10. `geppetto/pkg/js/modules/geppetto/api_profiles.go` — current JS profile namespace to replace/rename.
11. `geppetto/pkg/js/modules/geppetto/api_engines.go` — current engine construction path.

## Risks and Decisions

### Risk: dragging Pinocchio app semantics into Geppetto

Mitigation: move only inference-profile document semantics. Keep repositories and app-specific runtime behavior in Pinocchio wrappers.

### Risk: confusing Go internal names vs JS names

Mitigation: JS says `inferenceProfiles`; Go can keep `engineprofiles` until a separate rename is justified.

### Risk: credential leakage

Mitigation: JS APIs accept only symbolic credential refs. Host Go resolves secrets.

### Risk: overbuilding a full profile system

Mitigation: Geppetto profile documents contain only inference settings and registry selection. More elaborate agent profiles are application code.

### Risk: duplicate behavior during migration

Mitigation: port tests first, introduce Geppetto functions, make Pinocchio call them, then remove duplicated Pinocchio logic.

## Recommendation

Do the extraction. It is worth it because the core registry already lives in Geppetto, and only the inline/config-document composition layer is trapped in Pinocchio. The right move is not to import Pinocchio into Geppetto JS; it is to move the reusable inference-profile resolver into Geppetto and let Pinocchio become one host/application that supplies default sources.

The minimum valuable deliverable is:

1. `geppetto/pkg/engineprofiles/profiledoc` with inline profile documents and registry composition.
2. `gp.inferenceProfiles.resolve(...)` returning a Go-owned `InferenceSettings` wrapper.
3. `gp.inferenceSettings()` builder returning the same wrapper type.
4. JS credential policy that forbids env/API-key methods.
5. Pinocchio updated to use the Geppetto package for inline profile registry composition.

This is a medium extraction with low conceptual risk because most hard registry behavior already exists in Geppetto.

## References

- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/engineprofiles/types.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/engineprofiles/registry.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/engineprofiles/service.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/engineprofiles/source_chain.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/engineprofiles/stack_resolver.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/configdoc/types.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/configdoc/load.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/configdoc/merge.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/configdoc/profiles.go`
- `/home/manuel/workspaces/2026-06-01/geppetto-js/pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`
