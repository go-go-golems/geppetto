---
Title: 'Extensible Profile Metadata via Typed Keys: Architecture and Implementation Plan'
Ticket: GP-20-PROFILE-REGISTRY-EXTENSIONS
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - frontend
    - persistence
    - migration
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/profiles/types.go
      Note: Add extension payload fields and clone/deep-copy support.
    - Path: geppetto/pkg/profiles/validation.go
      Note: Add extension key/payload validation and error surfacing.
    - Path: geppetto/pkg/profiles/service.go
      Note: Resolve extensions into runtime context and middleware/tool instantiation hooks.
    - Path: geppetto/pkg/profiles/store.go
      Note: Store contract extension-safety expectations.
    - Path: geppetto/pkg/profiles/file_store_yaml.go
      Note: YAML round-trip behavior for extension payloads.
    - Path: geppetto/pkg/profiles/sqlite_store.go
      Note: JSON payload persistence and compatibility behavior for extension fields.
    - Path: geppetto/pkg/turns/key_families.go
      Note: Existing typed-key generic pattern reused for profile extension keys.
    - Path: geppetto/pkg/turns/types.go
      Note: Canonical namespaced key string shape to reuse.
    - Path: pinocchio/pkg/webchat
      Note: Consumer boundary where extension data can drive UI/behavior.
    - Path: go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Application integration point for extension-aware registry usage.
ExternalSources: []
Summary: Deep design for pattern 2: typed-key extension payloads on profiles/registries, enabling app-defined profile capabilities without core schema churn and without app flags in binaries.
LastUpdated: 2026-02-24T23:40:00-05:00
WhatFor: Provide implementation-grade guidance and API proposals for extensible profile metadata across model, validation, persistence, runtime resolution, and app integration.
WhenToUse: Use when implementing typed profile extensions, reviewing architecture tradeoffs, or onboarding contributors to the extension model and migration path.
---

# Extensible Profile Metadata via Typed Keys: Architecture and Implementation Plan

## Executive Summary

Pattern 2 introduces a typed-key extension layer to `profiles.Profile` and optionally `profiles.ProfileRegistry`, so application-specific profile data can live in registries instead of hard-coded flags and app-only structs.

The extension layer is designed with four constraints:

1. Core Geppetto must remain stable and opinionated only about shared concepts.
2. Applications must be free to add typed data without editing Geppetto every time.
3. Registry payloads must persist cleanly in both YAML and SQLite store backends.
4. Runtime behavior (middlewares, starters, routing, tool policies, etc.) should be derivable from profile registry content and extension resolvers, not from environment toggles.

The recommended implementation is:

- add an extension map to profile/registry models (`map[ProfileExtKey]any`),
- provide a generic typed-key API (`ExtK[T]`, `Get/Set/Decode`) modeled after `turns.DataKey[T]`,
- introduce an extension codec/registry contract for safe decode/validation,
- keep unknown keys as data (forward-compatible), validate known keys strictly,
- expose extension payloads through CRUD APIs and runtime resolution so `pinocchio` and `go-go-os` can consume them without app-specific branching in core binaries.

## Problem Statement

Current profile registry support is already a major step forward from binary flags, but runtime customizability is still bounded by fixed core fields in `RuntimeSpec` and related structs:

- `RuntimeSpec` currently has fixed fields (`step_settings_patch`, `system_prompt`, `middlewares`, `tools`).
- When an app wants profile-level data beyond this set (for example starter suggestions, capability panels, UI hints, external provider routing, per-app policy toggles), there is no first-class extension channel.
- The default fallback is to add app-specific flags/env variables or modify core structs, both of which increase coupling and maintenance cost.

This creates long-term friction:

- each new app feature pushes schema changes into Geppetto core;
- third-party users cannot safely carry custom profile data through registry CRUD without forking;
- webchat/go-go-os UI feature growth risks drift into app-local ad-hoc fields.

Pattern 2 addresses this by introducing typed extension keys and payloads that are persistable, discoverable, and runtime-consumable, while preserving compatibility for unknown keys.

### Fundamental Concept: Schema Core vs Extension Edge

The profile model should separate:

- **Core fields**: stable, cross-app semantics Geppetto can interpret directly.
- **Extension fields**: app-defined semantics Geppetto transports and optionally validates through typed codecs.

That separation keeps the core small and avoids repeating "add one more field to `RuntimeSpec`" cycles.

## Proposed Solution

### 1. Add Extension Bags to Domain Types

Add optional extension maps:

```go
// geppetto/pkg/profiles/types.go
type Profile struct {
    Slug        ProfileSlug
    DisplayName string
    Description string
    Runtime     RuntimeSpec
    Policy      PolicySpec
    Metadata    ProfileMetadata

    // New: app-defined profile extension payloads.
    Extensions map[ProfileExtensionKey]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

type ProfileRegistry struct {
    Slug               RegistrySlug
    DisplayName        string
    Description        string
    DefaultProfileSlug ProfileSlug
    Profiles           map[ProfileSlug]*Profile
    Metadata           RegistryMetadata

    // Optional: registry-level extension payloads.
    Extensions map[ProfileExtensionKey]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}
```

`Clone()` must deep-copy these maps recursively to avoid aliasing bugs.

### 2. Typed Extension Keys (Pattern 2 Core)

Create a typed-key wrapper analogous to `turns.DataKey[T]`:

```go
// geppetto/pkg/profiles/ext_keys.go
type ProfileExtensionKey string

func NewProfileExtensionKey(namespace, value string, version uint16) ProfileExtensionKey {
    return ProfileExtensionKey(turns.NewKeyString(namespace, value, version))
}

type ExtKey[T any] struct { id ProfileExtensionKey }

func ExtK[T any](namespace, value string, version uint16) ExtKey[T] {
    return ExtKey[T]{id: NewProfileExtensionKey(namespace, value, version)}
}

func (k ExtKey[T]) ID() ProfileExtensionKey { return k.id }
func (k ExtKey[T]) String() string          { return string(k.id) }
```

And typed accessors:

```go
func (k ExtKey[T]) Get(m map[ProfileExtensionKey]any) (T, bool, error) { ... }
func (k ExtKey[T]) Set(m *map[ProfileExtensionKey]any, v T) error      { ... }
func (k ExtKey[T]) Decode(raw any) (T, error)                          { ... } // JSON round-trip decode
```

This gives app authors compile-time typing at call sites while preserving storage as generic JSON/YAML.

### 3. Extension Codec Registry

Define optional codec registration for known keys:

```go
type ExtensionCodec interface {
    Key() ProfileExtensionKey
    Validate(raw any) error
    Normalize(raw any) (any, error)
    JSONSchema() map[string]any // optional for API/docs/UI generation
}

type ExtensionRegistry interface {
    Register(codec ExtensionCodec) error
    Lookup(key ProfileExtensionKey) (ExtensionCodec, bool)
    Keys() []ProfileExtensionKey
}
```

Behavior:

- For known keys: run `Normalize` + `Validate` during create/update.
- For unknown keys: pass-through by default (for forward compatibility), with optional strict mode in tests/tooling.

### 4. Validation Model

`ValidateProfile` and `ValidateRegistry` evolve:

- validate extension key syntax (canonical namespaced key string),
- reject empty/invalid keys,
- ensure extension payload is JSON-serializable,
- if codec exists, validate by codec.

Pseudo:

```text
for each (key, payload) in profile.extensions:
  assert valid canonical key
  assert json.Marshal(payload) succeeds
  if codec exists:
      payload' = codec.Normalize(payload)
      codec.Validate(payload')
      store normalized payload'
```

### 5. Persistence Implications

No schema migration is required in SQLite because registry payloads are persisted as JSON blobs (`payload_json`). YAML likewise accepts unknown maps naturally.

Required work:

- ensure encode/decode tests cover extension fields,
- ensure round-trip preserves unknown key payloads,
- ensure deep copy semantics are correct.

### 6. Runtime Consumption Hooks

Add an extension resolver phase in `ResolveEffectiveProfile`:

```go
type ExtensionResolver interface {
    // Called after core runtime merge, before runtime is finalized.
    Resolve(ctx context.Context, profile *Profile, registry *ProfileRegistry, effective *ResolvedProfile) error
}
```

`StoreRegistry` can accept optional resolvers:

```go
type StoreRegistry struct {
    store ProfileStore
    defaultRegistrySlug RegistrySlug
    extensionResolvers []ExtensionResolver
}
```

This allows app packages to register resolvers without polluting core packages. Example: webchat starter suggestions, app-specific UI hints, tool authorization augmentations.

### 7. CRUD/API Exposure

Profile CRUD payloads should include `extensions` with string keys. API contract example:

```json
{
  "slug": "inventory",
  "display_name": "Inventory",
  "runtime": { "system_prompt": "..." },
  "extensions": {
    "webchat.starter_suggestions@v1": {
      "suggestions": [
        "Show stock below reorder threshold",
        "Find inventory gaps for this week"
      ]
    },
    "inventory.capabilities@v1": {
      "panels": ["stock_alerts", "reorder", "suppliers"]
    }
  }
}
```

Apps can choose to interpret some keys and ignore others.

## Design Decisions

### Decision 1: Typed Keys Reuse Canonical Key Shape

Use `namespace.value@vN` key format (same pattern as `turns.NewKeyString`), instead of ad-hoc string constants.

Why:

- Namespacing avoids collisions.
- Versioning is explicit in key identity.
- Existing team mental model already uses this pattern.

### Decision 2: Keep Unknown Keys by Default

Unknown extension keys are not dropped or rejected in normal runtime/storage paths.

Why:

- forward compatibility across app versions,
- registry portability between pinocchio/go-go-os/custom apps,
- avoids accidental data loss on read-update-write cycles.

Strict rejection can be a linter/tooling mode, not default runtime behavior.

### Decision 3: Validation Is Layered

- Core validates syntax + serializability for all keys.
- Codec validates semantics for known keys.

Why:

- keeps core robust even without app codecs,
- allows progressive adoption by third parties.

### Decision 4: Runtime Extension via Resolvers, Not Switch Statements

Do not add app-specific `if key == ...` logic inside geppetto core service methods.

Why:

- keeps core package boundaries clean,
- enables app-local behavior with dependency injection.

### Decision 5: `Slug` Types Stay Strongly Typed

Continue typed `RegistrySlug`, `ProfileSlug`, `RuntimeKey` pattern and align extension key type with same philosophy.

Why:

- reduces stringly-typed mistakes,
- improves API readability and migration safety.

## Data Flow Diagram

```text
┌───────────────────────────────┐
│ profile-registry YAML/SQLite  │
│ - core fields                 │
│ - extensions map              │
└──────────────┬────────────────┘
               │ decode
               v
┌───────────────────────────────┐
│ geppetto profiles store       │
│ ValidateRegistry/Profile      │
│ - key syntax                  │
│ - payload serializability     │
│ - optional codec validation   │
└──────────────┬────────────────┘
               │ resolve
               v
┌───────────────────────────────┐
│ ResolveEffectiveProfile       │
│ - core runtime merge          │
│ - apply extension resolvers   │
│ - emit resolved metadata      │
└──────────────┬────────────────┘
               │ API/WS
               v
┌───────────────────────────────┐
│ pinocchio / go-go-os webchat  │
│ - list/select profile         │
│ - optional UI using ext keys  │
└───────────────────────────────┘
```

## API Suggestions

### A. Go: Extension Key and Helper Package

Recommended package: `geppetto/pkg/profiles/extensions`.

Suggested API:

```go
package extensions

type Key string
type TypedKey[T any] struct { ... }

func NewKey(namespace, value string, version uint16) Key
func K[T any](namespace, value string, version uint16) TypedKey[T]

func Get[T any](m map[Key]any, k TypedKey[T]) (T, bool, error)
func Set[T any](m *map[Key]any, k TypedKey[T], v T) error
```

### B. Go: Profile Convenience Methods

```go
func (p *Profile) GetExt[T any](k extensions.TypedKey[T]) (T, bool, error)
func (p *Profile) SetExt[T any](k extensions.TypedKey[T], v T) error
func (r *ProfileRegistry) GetExt[T any](k extensions.TypedKey[T]) (T, bool, error)
```

### C. CRUD JSON

- Keep wire format as `map[string]any`.
- Convert keys to typed form at domain boundary.

Example request:

```http
PATCH /api/chat/profiles/inventory?registry=default
Content-Type: application/json

{
  "extensions": {
    "webchat.starter_suggestions@v1": {
      "suggestions": ["What changed today?", "Show low stock SKUs"]
    }
  }
}
```

### D. Runtime Resolver Registration

```go
func NewStoreRegistry(
    store ProfileStore,
    defaultRegistrySlug RegistrySlug,
    opts ...StoreRegistryOption,
) (*StoreRegistry, error)

type StoreRegistryOption func(*StoreRegistry)

func WithExtensionResolver(r ExtensionResolver) StoreRegistryOption
```

This avoids constructor churn when new options appear.

## Pseudocode: End-to-End Flow

```pseudo
function UpdateProfile(registrySlug, profileSlug, patch):
  current = store.GetProfile(registrySlug, profileSlug)
  next = clone(current)
  applyPatch(next, patch)

  for (k, raw) in next.Extensions:
    assert IsCanonicalKey(k)
    assert IsJSONSerializable(raw)
    if codecRegistry.Has(k):
      normalized = codecRegistry[k].Normalize(raw)
      codecRegistry[k].Validate(normalized)
      next.Extensions[k] = normalized

  store.UpsertProfile(registrySlug, next)
  return next

function ResolveEffectiveProfile(input):
  registry = store.GetRegistry(input.RegistrySlug)
  profile = selectProfile(input.ProfileSlug, registry.default)

  effective = resolveCoreRuntime(profile.runtime, input.overrides, profile.policy)
  resolved = new ResolvedProfile(effective, profile, registry)

  for resolver in extensionResolvers:
    resolver.Resolve(ctx, profile, registry, resolved)

  return resolved
```

## Minimal Example: Starter Suggestions Extension

Typed key:

```go
type StarterSuggestionsExt struct {
    Suggestions []string `json:"suggestions" yaml:"suggestions"`
}

var StarterSuggestionsKey = extensions.K[StarterSuggestionsExt](
    "webchat", "starter_suggestions", 1,
)
```

Consumption in webchat integration:

```go
if ext, ok, err := profile.GetExt(StarterSuggestionsKey); err == nil && ok {
    response.StarterSuggestions = ext.Suggestions
}
```

No Geppetto core schema expansion is needed for this feature once extension plumbing exists.

## Migration Plan

### Phase 1: Domain and Validation Foundation

- Add `Extensions` fields to `Profile` and `ProfileRegistry`.
- Add key type + typed helpers.
- Extend `Clone`, validators, and tests.

### Phase 2: Persistence and Round-Trip Guarantees

- Add YAML/SQLite round-trip tests including unknown and known keys.
- Verify no data loss for unknown extension keys.

### Phase 3: CRUD/API and Client Contract

- Update API models to expose `extensions`.
- Add request validation with clear error field paths.
- Add API docs/examples for extension key formatting.

### Phase 4: Runtime Resolution Integration

- Add extension resolver interface and option plumbing.
- Provide first resolver(s) in app layers (pinocchio/go-go-os).

### Phase 5: Migration from Flags/App Config

- Map existing app toggles to profile extension keys.
- Keep migration tooling/docs (for old `profiles.yaml` shape or legacy env-driven behavior).
- Remove stale flags once parity is verified.

## Compatibility and Versioning

### Key Versioning Rule

- Bump key version only when payload semantics or required shape changes incompatibly.
- New optional fields within same payload can remain same key version if decoder tolerates them.

### Forward/Backward Compatibility

- Unknown keys: preserved.
- Unknown key versions for known namespaces: preserved unless strict mode.
- Decoder can optionally support multiple versions by registering multiple typed keys/codecs.

## Security and Robustness Considerations

- Enforce maximum payload size for extension maps on API ingress.
- Validate serializability to block unsupported Go values from entering storage.
- Protect resolver execution: resolver failures should produce explicit typed errors; avoid partial silent failures.
- Consider policy controls for writeability of extension namespaces:
  - e.g. `policy.allowed_extension_prefixes`.

## Alternatives Considered

### Alternative A: Keep Adding Fields to `RuntimeSpec`

Rejected because it causes core schema churn and couples app features directly to Geppetto release cadence.

### Alternative B: Single Unstructured `map[string]any` Without Typed Keys

Rejected because:

- weak discoverability,
- no version semantics,
- hard to validate and refactor safely,
- encourages accidental key collisions.

### Alternative C: Fully Pluginized Profile Runtime in Core

Partially viable but too heavy for immediate need. The typed-key + resolver model provides most benefits with lower complexity and faster adoption.

## Open Questions

1. Should registry-level extensions be enabled in phase 1 or deferred to phase 2?
2. Should unknown-key preservation be configurable globally (`strict`, `warn`, `passthrough`)?
3. Should extension schemas be exposed on an API endpoint for frontend form generation?
4. Do we need namespace ownership conventions (`webchat.*`, `inventory.*`, `thirdparty.<vendor>.*`) documented centrally?
5. Should policy include extension-write ACLs per actor/source?

## Implementation Checklist (Short Form)

- [ ] Add `Extensions` fields and deep-copy support.
- [ ] Add typed extension key package and tests.
- [ ] Wire validator for key shape + serializability + codec validation.
- [ ] Expose extensions in CRUD endpoints and DTOs.
- [ ] Add resolver interface and option-based injection.
- [ ] Implement first extension in app layer (starter suggestions recommended).
- [ ] Add migration playbook and update docs.

## References

- `geppetto/pkg/turns/key_families.go` for typed key API pattern (`DataKey[T]`, `DataK`).
- `geppetto/pkg/turns/types.go` for canonical key construction (`namespace.value@vN`).
- `geppetto/pkg/profiles/types.go` and `geppetto/pkg/profiles/service.go` for current profile model/resolution flow.

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
