---
Title: ProfileRegistry Architecture and Migration Plan
Ticket: GP-01-ADD-PROFILE-REGISTRY
Status: active
Topics:
    - architecture
    - geppetto
    - pinocchio
    - chat
    - inference
    - persistence
    - migration
    - backend
    - frontend
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/sections/sections.go
      Note: Current Geppetto bootstrap and profile middleware chain for CLI layering
    - Path: geppetto/pkg/steps/ai/settings/settings-step.go
      Note: StepSettings decode and metadata path used for runtime fingerprinting
    - Path: go-go-os/packages/engine/src/chat/runtime/http.ts
      Note: Current chat request payload shape without profile fields
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: Current in-memory web-chat profile resolver and profile endpoints
    - Path: pinocchio/cmd/web-chat/runtime_composer.go
      Note: Current runtime composition and override validation path
    - Path: pinocchio/pkg/persistence/chatstore/timeline_store.go
      Note: Interface-first persistence pattern reused for profile store design
    - Path: pinocchio/pkg/webchat/http/api.go
      Note: Canonical ConversationRequestPlan contract for chat and WS handlers
ExternalSources: []
Summary: Detailed architecture and migration plan for introducing reusable multi-source ProfileRegistry support in geppetto and adopting it across pinocchio web-chat and go-go-os clients.
LastUpdated: 2026-02-23T13:48:00-05:00
WhatFor: Replace flag-heavy per-binary AI runtime configuration with reusable profile registries supporting disk and database stores and first-class web profile UX.
WhenToUse: Use when implementing or reviewing profile-driven runtime composition, registry persistence, and profile APIs across geppetto/pinocchio/go-go-os.
---


# ProfileRegistry Architecture and Migration Plan

## 1. Executive Summary

This document proposes a new `ProfileRegistry` architecture centered in Geppetto and integrated into Pinocchio and Go-Go-OS. The target state is:

1. AI runtime configuration is profile-driven instead of flag-driven.
2. Registries are reusable objects with multiple backends (memory, file, database).
3. Pinocchio web-chat can list/select/create/update profiles through a real registry API.
4. Go-Go-OS web clients can consume that API and choose profiles at runtime.
5. Existing commands continue to work during migration through compatibility layers.

The existing system already has strong primitives we can reuse:

- Geppetto already has layered config decoding and stable `StepSettings` objects.
- Pinocchio already has runtime-keyed conversation lifecycle (`RuntimeKey`, `RuntimeFingerprint`).
- Pinocchio already has persistence abstractions (`TimelineStore` with in-memory and SQLite variants).
- Go-Go-OS chat runtime already encapsulates HTTP/WS transport and is ready for profile fields.

The main missing piece is a shared profile domain model plus a registry service that is not tied to CLI profile-file parsing.

## 2. Problem Statement and Goals

### 2.1 Current Problem

Today, “profile” behavior exists in multiple forms:

- Geppetto/Glazed profile files that map to section flags (`sources.GatherFlagsFromProfiles(...)` in `geppetto/pkg/sections/sections.go:284`).
- Pinocchio web-chat in-memory `chatProfileRegistry` (`pinocchio/cmd/web-chat/profile_policy.go:22`) with hardcoded profile structs.
- Go-Go-OS inventory integration that pins one strict runtime (`go-go-os/go-inventory-chat/internal/pinoweb/request_resolver.go:12`).

This creates duplicated policy and many AI flags in each binary command spec.

### 2.2 Goals

- Build `ProfileRegistry` as a reusable runtime object (not a command-only parsing trick).
- Support multiple registries and sources (default built-ins + file + DB + app overlays).
- Reduce or remove most AI engine/provider flags from command surfaces over time.
- Provide profile CRUD and listing APIs for web clients.
- Keep compatibility with existing `profiles.yaml` workflows during migration.
- Enable persistence and governance (audit fields, versioning, policy).

### 2.3 Non-Goals (Phase 1)

- Not redesigning every `StepSettings` field.
- Not removing all runtime overrides immediately.
- Not forcing one single profile schema for every future domain extension; we design extensibility hooks.

## 3. Current State Analysis

### 3.1 Geppetto: Flag-Centric Profile Injection

Key files:

- `geppetto/pkg/sections/sections.go:123`
- `geppetto/pkg/steps/ai/settings/settings-chat.go:22`
- `geppetto/pkg/steps/ai/settings/flags/chat.yaml:5`
- `geppetto/pkg/steps/ai/settings/settings-step.go:271`

#### What happens today

1. Command schemas include all AI flags via Geppetto sections (`CreateGeppettoSections`, `sections.go:34`).
2. Middleware bootstrap resolves `profile-settings.profile` and `profile-settings.profile-file` from env/config/flags.
3. Profile flags are loaded via `sources.GatherFlagsFromProfiles(...)` (`sections.go:284`) and merged into parsed values.
4. `settings.NewStepSettingsFromParsedValues(...)` decodes values into a runtime object (`settings-step.go:113`).
5. Engine is created from `StepSettings` by provider-specific factory logic (`geppetto/pkg/inference/engine/factory/factory.go:50`).

#### Strengths

- Mature layer precedence model.
- Stable `StepSettings` object used across runtime stack.
- Existing profile file concept is familiar to users.

#### Weaknesses

- Profile loading is tightly coupled to command middlewares.
- AI provider/model flags are exposed everywhere (`ai-engine`, `ai-api-type`, etc.).
- No reusable profile service API for app code or web APIs.
- No first-class registry/store abstraction for DB-backed profiles.

### 3.2 Pinocchio: App-Owned Runtime and In-Memory Profiles

Key files:

- `pinocchio/cmd/web-chat/profile_policy.go:14`
- `pinocchio/cmd/web-chat/runtime_composer.go:15`
- `pinocchio/pkg/inference/runtime/composer.go:10`
- `pinocchio/pkg/webchat/conversation.go:40`
- `pinocchio/pkg/webchat/http/api.go:28`

#### What happens today

- Web-chat defines local `chatProfile` and `chatProfileRegistry` structs.
- Resolver chooses profile from path/query/cookie/default and returns `ConversationRequestPlan`.
- Runtime composer builds `StepSettings` from parsed values, applies per-profile overrides (`system_prompt`, `middlewares`, `tools`), and computes fingerprint.
- Conversation manager rebuilds runtime when fingerprint changes (`conversation.go:291`).
- Basic profile endpoints exist:
  - `GET /api/chat/profiles`
  - `GET/POST /api/chat/profile`

#### Strengths

- Runtime composition boundary is already clean (`RuntimeComposer`).
- Conversation lifecycle already understands runtime identity and hot swap by fingerprint.
- HTTP request resolution is explicit and testable.

#### Weaknesses

- Profile registry is local to one command package.
- No persistence for profiles.
- API surface only supports list/current-set, not CRUD with validation/versioning.
- Profile schema is ad-hoc and not shared with Geppetto profile files.

### 3.3 Go-Go-OS: Good Chat Runtime Shell, No Profile Transport Yet

Key files:

- `go-go-os/packages/engine/src/chat/runtime/http.ts:28`
- `go-go-os/packages/engine/src/chat/runtime/conversationManager.ts:71`
- `go-go-os/packages/engine/src/chat/ws/wsManager.ts:63`
- `go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx:77`
- `go-go-os/apps/inventory/src/App.tsx:176`

#### What happens today

- Chat send uses payload `{ prompt, conv_id }` with no profile fields.
- WS URL is `/ws?conv_id=...` with no profile query.
- Conversation hooks/components expose no profile state.
- Inventory app currently relies on server-side strict runtime key policy.

#### Strengths

- Frontend transport and chat state are modular and easy to extend.
- There is already a clear seam for adding profile APIs in runtime/http layer.

#### Weaknesses

- No profile listing/selection/create UX in generic engine chat stack.
- No mechanism for sending requested profile on chat send or connect.
- No local store slice for profile metadata.

### 3.4 Existing Store Patterns We Can Reuse

Key files:

- `pinocchio/pkg/persistence/chatstore/timeline_store.go:27`
- `pinocchio/pkg/persistence/chatstore/timeline_store_memory.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go`
- `geppetto/pkg/inference/tools/registry.go:9`

Both codebases already use interface-first patterns with memory + SQLite implementations. We should mirror this for profiles:

- Interface for read/write operations.
- In-memory implementation for tests/dev.
- SQLite implementation for durable environments.
- Optional file-backed implementation for compatibility and local workflows.

## 4. Fundamental Concepts (Intern Onboarding)

This section defines the conceptual model before APIs.

### 4.1 Profile vs Runtime vs Registry

- A `Profile` is a named runtime preset and policy.
- A `Runtime` is the concrete composed engine + middleware + tools for one conversation.
- A `Registry` is a container/source of profiles plus metadata and lifecycle operations.

### 4.2 Runtime Composition Layers

Current conceptual precedence can be represented as:

1. Hardcoded defaults.
2. Registry/default profile settings.
3. Selected profile settings.
4. App policy constraints (allow/disallow overrides).
5. Request-time overrides (if allowed).

We should preserve explicit precedence in code and tests.

### 4.3 Immutable Identity and Fingerprints

A profile slug is not enough for runtime identity because profile definitions can change. Keep both:

- `runtime_key`: human-readable slug (for routing and UI).
- `runtime_fingerprint`: stable hash/payload of effective runtime config.

Pinocchio already uses this pattern (`conversation.go:291`), and it should remain the source of truth for conversation runtime changes.

### 4.4 Registry Sources and Composition

We need layered registries, not one file:

- Built-in defaults (compiled).
- File-based user profiles.
- DB-backed org profiles.
- Optional app overlays (session or product-specific constraints).

Think of this as a chain/overlay with conflict rules.

## 5. Proposed Target Architecture

## 5.1 Package Placement and Boundaries

### In Geppetto (new package)

`geppetto/pkg/profiles/`

Recommended files:

- `types.go` (profile and registry domain types)
- `store.go` (store interfaces)
- `registry.go` (main service abstraction)
- `overlay.go` (composition and precedence)
- `codec_yaml.go` (file schema and parsing)
- `stepsettings_mapper.go` (profile -> StepSettings merge)
- `validation.go` (schema + policy validation)

### In Pinocchio

- Replace local web-chat registry structs with Geppetto `profiles.Registry` adapters.
- Add profile API handlers that talk to registry service.
- Keep `RuntimeComposer` boundary; feed it resolved effective profile runtime spec.

### In Go-Go-OS

- Add profile API client module.
- Add profile state slice and selector hooks.
- Extend send/connect transport payloads with optional profile key.

## 5.2 Data Model Proposal

```go
type ProfileID string
type RegistryID string

type Profile struct {
    ID          ProfileID
    Slug        string
    DisplayName string
    Description string

    Runtime RuntimeSpec
    Policy  PolicySpec

    Metadata ProfileMetadata
}

type RuntimeSpec struct {
    // Step settings patch. Canonical internal representation.
    StepSettingsPatch map[string]any

    // Optional convenience fields for web/API clients.
    SystemPrompt string
    Middlewares  []MiddlewareUse
    Tools        []string
}

type PolicySpec struct {
    AllowOverrides bool
    AllowedOverrideKeys []string
    DeniedOverrideKeys  []string
    ReadOnly bool
}

type ProfileMetadata struct {
    Source       string // builtin|file|db|api
    Version      uint64
    CreatedAtMs  int64
    UpdatedAtMs  int64
    CreatedBy    string
    UpdatedBy    string
    Tags         []string
}

type ProfileRegistry struct {
    ID          RegistryID
    Slug        string
    DisplayName string

    DefaultProfileSlug string
    Profiles           map[string]*Profile

    Metadata RegistryMetadata
}
```

### Why this shape

- Keeps one canonical runtime payload (`StepSettingsPatch`) while allowing web-friendly fields.
- Captures policy controls without hardcoding to one app.
- Adds source/version metadata needed for audit and optimistic concurrency.

## 5.3 Registry Service Interfaces

```go
type RegistryReader interface {
    ListRegistries(ctx context.Context) ([]RegistrySummary, error)
    GetRegistry(ctx context.Context, registrySlug string) (*ProfileRegistry, error)
    ListProfiles(ctx context.Context, registrySlug string) ([]Profile, error)
    GetProfile(ctx context.Context, registrySlug, profileSlug string) (*Profile, error)
    ResolveEffectiveProfile(ctx context.Context, in ResolveInput) (*ResolvedProfile, error)
}

type RegistryWriter interface {
    CreateProfile(ctx context.Context, registrySlug string, p Profile, opts WriteOptions) (*Profile, error)
    UpdateProfile(ctx context.Context, registrySlug, profileSlug string, patch ProfilePatch, opts WriteOptions) (*Profile, error)
    DeleteProfile(ctx context.Context, registrySlug, profileSlug string, opts WriteOptions) error
    SetDefaultProfile(ctx context.Context, registrySlug, profileSlug string, opts WriteOptions) error
}

type Registry interface {
    RegistryReader
    RegistryWriter
}

type ResolveInput struct {
    RegistrySlug string
    ProfileSlug  string
    RuntimeKeyFallback string

    BaseStepSettings *settings.StepSettings
    RequestOverrides map[string]any
}

type ResolvedProfile struct {
    RegistrySlug string
    ProfileSlug  string
    RuntimeKey   string

    EffectiveStepSettings *settings.StepSettings
    EffectiveSystemPrompt string
    EffectiveMiddlewares  []runtime.MiddlewareUse
    EffectiveTools        []string

    RuntimeFingerprint string
    Metadata map[string]any
}
```

### Key behavioral contract

- `ResolveEffectiveProfile` is the single source of precedence and validation.
- Runtime fingerprint is generated from resolved effective runtime, not just slug.
- Writers enforce policy and validation uniformly across storage backends.

## 5.4 Store Adapters

```go
type ProfileStore interface {
    LoadRegistries(ctx context.Context) ([]*ProfileRegistry, error)
    SaveRegistry(ctx context.Context, reg *ProfileRegistry, opts SaveOptions) error
    DeleteRegistry(ctx context.Context, slug string, opts SaveOptions) error
    Close() error
}
```

Implementations:

- `InMemoryProfileStore`
- `YAMLFileProfileStore`
- `SQLiteProfileStore`

### Store composition

Use an overlay reader + designated writer model:

- Read path: merge from multiple stores in order.
- Write path: send to primary mutable store (usually DB, fallback file).

## 5.5 Architecture Diagrams

### System Context

```text
+--------------------+        +-----------------------------+
| geppetto sections  |        | pinocchio web-chat server   |
| + cli middlewares  |        | request resolver + composer |
+---------+----------+        +---------------+-------------+
          |                                   |
          v                                   v
      +-------------------------------------------+
      |      geppetto/pkg/profiles Registry       |
      | resolve/list/create/update/delete profile |
      +-------------------+-----------------------+
                          |
         +----------------+------------------+
         |                                   |
         v                                   v
+-----------------------+          +-----------------------+
| YAML file profile store|          | SQLite profile store  |
+-----------------------+          +-----------------------+
                          |
                          v
              +--------------------------+
              | go-go-os web clients     |
              | list/select/create       |
              +--------------------------+
```

### Runtime Resolution Sequence (chat request)

```text
Client -> /chat {conv_id,prompt,profile?,overrides?}
  -> RequestResolver
     -> Registry.ResolveEffectiveProfile(...)
        -> load selected profile + policy + defaults
        -> merge request overrides (if allowed)
        -> produce EffectiveStepSettings + fingerprint
     -> ConversationRequestPlan{runtime_key, overrides, ...}
  -> RuntimeComposer.Compose(...)
     -> ComposeEngineFromSettings(EffectiveStepSettings)
  -> ConversationManager.GetOrCreate(...)
     -> compare fingerprint, rebuild if changed
```

## 6. Geppetto Implementation Plan

### 6.1 Introduce ProfileRegistry Core (No Breaking Changes Yet)

Phase-1 implementation in Geppetto should be additive.

1. Create `pkg/profiles` package with domain types and interfaces.
2. Add YAML parser that can read both:
   - legacy profile map format (`profile -> section -> flags`)
   - new registry document format.
3. Implement adapter: legacy profile shape -> `Profile.Runtime.StepSettingsPatch`.
4. Add resolver method that outputs `*settings.StepSettings` and runtime metadata.

### 6.2 Integrate with Existing Section Parsing

Current integration seam is `GetCobraCommandGeppettoMiddlewares` (`sections.go:123`).

Short-term strategy:

- Keep existing middleware chain.
- Replace direct `GatherFlagsFromProfiles` call with registry-backed middleware adapter:

```go
middlewares_ = append(middlewares_,
    profilesources.FromRegistry(registry, profilesources.Options{
        ProfileSlug: profileSettings.Profile,
        ProfileFile: profileSettings.ProfileFile,
        DefaultProfile: "default",
        SourceName: "profiles",
    }),
)
```

This preserves command behavior while moving profile semantics into reusable registry logic.

### 6.3 Gradual Flag Surface Reduction

Current AI flags in `ai-chat` section (`chat.yaml:5`) should be migrated in phases:

1. Phase A: keep flags, mark as advanced/deprecated in help text.
2. Phase B: hide most provider/model flags from common commands (retain profile selector flags only).
3. Phase C: keep low-level flags only in expert/debug commands.

Recommended retained default command flags:

- `--profile`
- `--profile-registry` (optional)
- `--profile-file` (compatibility)

### 6.4 Runtime Metadata Conventions

Enrich metadata from resolver output so debug tooling can trace profile provenance:

- `profile.registry.slug`
- `profile.slug`
- `profile.version`
- `profile.source`
- `profile.runtime_fingerprint`

This builds on existing `StepSettings.GetMetadata()` usage (`settings-step.go:127`).

## 7. Pinocchio Integration Plan

### 7.1 Replace Local Registry Structs

Replace `chatProfile` and `chatProfileRegistry` in `pinocchio/cmd/web-chat/profile_policy.go` with a small adapter around Geppetto registry interfaces.

Current resolver signatures can remain unchanged:

```go
type webChatProfileResolver struct {
    registry profiles.Registry
    registrySlug string
}
```

### 7.2 Keep RuntimeComposer Boundary, Change Input Source

`webChatRuntimeComposer` currently builds `StepSettings` from parsed command values (`runtime_composer.go:69`).

New behavior:

1. Start from parsed values as base defaults.
2. Ask registry for effective profile resolution.
3. Compose engine from resolved step settings.
4. Keep fingerprint generation but include registry/profile/version metadata.

### 7.3 Expand Profile API Surface

Current profile endpoints are read/set current cookie only. Add CRUD endpoints:

- `GET /api/chat/profiles`
- `GET /api/chat/profiles/{slug}`
- `POST /api/chat/profiles`
- `PATCH /api/chat/profiles/{slug}`
- `DELETE /api/chat/profiles/{slug}`
- `POST /api/chat/profile` (set current selection, keep compatibility)

#### Example response payload

```json
{
  "slug": "inventory-analyst",
  "display_name": "Inventory Analyst",
  "description": "Tool-first inventory helper",
  "runtime": {
    "system_prompt": "You are an inventory assistant.",
    "middlewares": [{"name":"inventory-artifact-policy","config":{}}],
    "tools": ["inventory.get", "inventory.list", "inventory.find"]
  },
  "policy": {
    "allow_overrides": false,
    "read_only": false
  },
  "metadata": {
    "version": 12,
    "source": "db",
    "updated_at_ms": 1771880000000
  }
}
```

### 7.4 Request Plan Extensions

`ChatRequestBody` currently includes `prompt`, `conv_id`, and `overrides` (`pinocchio/pkg/webchat/http/api.go:20`). Add optional explicit profile fields:

- `profile`
- `registry`

This allows API clients (including Go-Go-OS) to explicitly select a profile without cookie dependence.

### 7.5 Policy Enforcement

Move policy checks out of local `mergeOverrides` heuristics and into shared registry resolver policy evaluation.

This avoids drift between command apps.

## 8. Go-Go-OS Integration Plan

### 8.1 Transport Contract Changes

Update `submitPrompt` in `go-go-os/packages/engine/src/chat/runtime/http.ts:28`:

```ts
body: JSON.stringify({
  prompt,
  conv_id: convId,
  profile: selectedProfileSlug,
  registry: selectedRegistrySlug,
  overrides,
})
```

Update WS URL builder in `wsManager.ts:63`:

```ts
/ws?conv_id=...&profile=...&registry=...
```

(Keep server-side fallback to cookie/default for backward compatibility.)

### 8.2 Profile API Client in Engine Package

Add `packages/engine/src/chat/runtime/profileApi.ts`:

- `listProfiles(basePrefix, registry?)`
- `getCurrentProfile(basePrefix)`
- `setCurrentProfile(basePrefix, slug, registry?)`
- `createProfile(basePrefix, input)`
- `updateProfile(basePrefix, slug, patch, version)`
- `deleteProfile(basePrefix, slug)`

### 8.3 Profile State Slice

Introduce Redux slice:

- `availableProfiles`
- `selectedProfile`
- `registry`
- `loading/error`

Expose hooks for UI components.

### 8.4 UI/UX Updates

`ChatConversationWindow` currently accepts header actions and has no profile props (`ChatConversationWindow.tsx:32`). Add optional profile control props:

- `enableProfiles?: boolean`
- `profileMode?: "cookie" | "explicit"`

Inventory app can then embed a profile dropdown in `headerActions` while reusing generic engine logic.

## 9. Database Storage Design

### 9.1 SQL Schema Proposal (SQLite First)

```sql
CREATE TABLE IF NOT EXISTS profile_registries (
  slug TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  default_profile_slug TEXT,
  created_at_ms INTEGER NOT NULL,
  updated_at_ms INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS profiles (
  registry_slug TEXT NOT NULL,
  slug TEXT NOT NULL,
  display_name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  runtime_json TEXT NOT NULL,
  policy_json TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  created_at_ms INTEGER NOT NULL,
  updated_at_ms INTEGER NOT NULL,
  created_by TEXT NOT NULL DEFAULT '',
  updated_by TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (registry_slug, slug),
  FOREIGN KEY (registry_slug) REFERENCES profile_registries(slug) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_profiles_registry_updated
  ON profiles(registry_slug, updated_at_ms DESC);
```

### 9.2 Concurrency Model

Use optimistic concurrency:

- Reads return `version`.
- Updates include expected version.
- Reject with conflict if version mismatch.

### 9.3 Audit and Governance

Minimal first-phase audit:

- `created_by`, `updated_by`.
- `updated_at_ms`.
- Optionally append-only `profile_events` table later.

### 9.4 Mapping to Existing Patterns

Mirror the style of `TimelineStore` interfaces (`timeline_store.go:27`) to minimize cognitive load for maintainers.

## 10. API Suggestions (HTTP + Go)

### 10.1 HTTP Endpoints

#### List profiles

`GET /api/chat/profiles?registry=default`

Response:

```json
{
  "registry": "default",
  "default_profile": "default",
  "items": [
    {"slug":"default","display_name":"Default","read_only":false},
    {"slug":"agent","display_name":"Agent","read_only":false}
  ]
}
```

#### Create profile

`POST /api/chat/profiles`

Request:

```json
{
  "registry": "default",
  "slug": "ops-analyst",
  "display_name": "Ops Analyst",
  "runtime": {
    "system_prompt": "You are an operations analyst.",
    "tools": ["calculator"],
    "middlewares": []
  },
  "policy": {
    "allow_overrides": true,
    "allowed_override_keys": ["system_prompt"]
  }
}
```

#### Update profile

`PATCH /api/chat/profiles/{slug}`

Request:

```json
{
  "registry": "default",
  "expected_version": 3,
  "patch": {
    "display_name": "Ops Analyst v2",
    "runtime": {
      "system_prompt": "You are a concise operations analyst."
    }
  }
}
```

### 10.2 Go Resolver API for WebChat

```go
type RequestProfileSelector struct {
    Registry string
    Profile  string
}

type ProfileAwareRequestResolver struct {
    Registry profiles.Registry
    DefaultRegistry string
}

func (r *ProfileAwareRequestResolver) Resolve(req *http.Request) (webhttp.ConversationRequestPlan, error) {
    // Parse conv_id/prompt/profile/registry from body/query/cookie
    // Resolve effective profile via registry
    // Return RuntimeKey = profile slug
    // Return Overrides only when policy allows
}
```

### 10.3 Fingerprint API

```go
func FingerprintResolvedRuntime(res *profiles.ResolvedProfile) string {
    payload := map[string]any{
        "registry": res.RegistrySlug,
        "profile": res.ProfileSlug,
        "version": res.Metadata["version"],
        "settings": res.EffectiveStepSettings.GetMetadata(),
        "system_prompt": res.EffectiveSystemPrompt,
        "middlewares": res.EffectiveMiddlewares,
        "tools": res.EffectiveTools,
    }
    b, _ := json.Marshal(payload)
    return string(b)
}
```

## 11. Migration Strategy

## 11.1 Phased Plan

### Phase 0: Design and Compatibility Harness

- Add `pkg/profiles` types/interfaces in Geppetto.
- Add legacy YAML adapter loader.
- Add unit tests for precedence and policy validation.

### Phase 1: Wire Geppetto Middlewares to Registry Adapter

- Keep `--profile` behavior unchanged.
- Internally route through registry resolver.
- Add metadata fields for profile provenance.

### Phase 2: Pinocchio Web-Chat Registry Integration

- Replace local profile structs with shared registry service.
- Add CRUD profile endpoints.
- Keep existing `/api/chat/profile` cookie route for compatibility.

### Phase 3: Go-Go-OS Client Profile UX

- Add profile API client + state slice.
- Add dropdown/select workflow.
- Send explicit profile in chat and WS transport.

### Phase 4: DB-Backed Primary Store

- Add SQLite store implementation.
- Enable configured apps to read/write profiles from DB.
- Keep optional file fallback for local environments.

### Phase 5: Flag Surface Cleanup

- Deprecate most AI engine/provider flags on common commands.
- Keep advanced flags under expert/debug command groups.

## 11.2 Backward Compatibility Rules

- Legacy `profiles.yaml` still loads.
- Existing `--profile`, `PINOCCHIO_PROFILE`, and config-file profile selection still work.
- Existing chat endpoints continue to accept requests without explicit profile field.

## 11.3 Migration Risk Table

- Risk: precedence regression. Mitigation: golden tests comparing old/new merged values.
- Risk: runtime churn on profile update. Mitigation: fingerprint includes version; explicit rebuild logs.
- Risk: policy bypass via overrides. Mitigation: centralized resolver enforcement + deny-by-default keys.
- Risk: frontend/server contract mismatch. Mitigation: versioned API response schema + integration tests.

## 12. Test Plan

### 12.1 Geppetto Unit Tests

- Registry load from legacy YAML.
- Overlay conflict resolution across multiple stores.
- `ResolveEffectiveProfile` precedence matrix.
- Policy allow/deny override keys.
- Fingerprint stability tests.

### 12.2 Pinocchio Integration Tests

- Resolver behavior for GET `/ws` and POST `/chat` with profile field/cookie/default.
- Conversation runtime rebuild when profile version changes.
- CRUD API tests including optimistic concurrency conflict.

### 12.3 Go-Go-OS Tests

- `profileApi` runtime client tests with mock fetch.
- `submitPrompt` includes profile/registry payload fields.
- `resolveWsUrl` includes profile query when configured.
- UI reducer tests for selection/create/update state.

### 12.4 End-to-End Scenario Test Matrix

1. Select profile in UI -> send prompt -> server uses selected runtime key.
2. Create profile from UI -> immediate visibility in list.
3. Update profile version -> existing conversation rebuilds on next request.
4. Read-only profile rejects mutation with clear error.

## 13. Suggested Initial File Changes (Concrete)

### Geppetto

- Add `geppetto/pkg/profiles/*` (new).
- Modify `geppetto/pkg/sections/sections.go`:
  - replace direct `GatherFlagsFromProfiles` usage with registry-backed adapter.

### Pinocchio

- Modify `pinocchio/cmd/web-chat/profile_policy.go`:
  - replace local in-memory structs with registry adapter.
- Modify `pinocchio/cmd/web-chat/runtime_composer.go`:
  - accept resolved profile output.
- Add/modify handlers under `pinocchio/pkg/webchat/http` for profile CRUD.

### Go-Go-OS

- Modify `go-go-os/packages/engine/src/chat/runtime/http.ts`.
- Modify `go-go-os/packages/engine/src/chat/ws/wsManager.ts`.
- Add profile client/store files in `go-go-os/packages/engine/src/chat/runtime` and `.../state`.
- Update app integration in `go-go-os/apps/inventory/src/App.tsx`.

## 14. Pseudocode Walkthrough

```go
// Server startup
storeChain := profiles.NewOverlayStore(
    profiles.NewBuiltinStore(defaultProfiles),
    profiles.NewYAMLFileStore(userFile),
    profiles.NewSQLiteStore(db),
)
registrySvc := profiles.NewService(storeChain, profiles.ServiceOptions{
    DefaultRegistry: "default",
})

resolver := webchat.NewProfileAwareRequestResolver(registrySvc)
composer := webchat.NewProfileAwareRuntimeComposer(parsedValues, registrySvc, middlewareFactories)

srv := webchat.NewServer(...,
    webchat.WithRuntimeComposer(composer),
)

// Request flow
plan := resolver.Resolve(req)
resolved := registrySvc.ResolveEffectiveProfile(ctx, profiles.ResolveInput{
    RegistrySlug: plan.RegistrySlug,
    ProfileSlug: plan.RuntimeKey,
    BaseStepSettings: baseSettings,
    RequestOverrides: plan.Overrides,
})

runtime := composer.Compose(ctx, requestFromPlanAndResolved(plan, resolved))
conversation := convManager.GetOrCreate(plan.ConvID, runtime.RuntimeKey, plan.Overrides)
```

## 15. Operational Considerations

### 15.1 Security

- Validate middleware names against registered allowlist.
- Validate tool names against registered tool factories.
- Deny sensitive override keys by default (API keys, base URLs, provider credentials).
- Enforce read-only profiles for managed environments.

### 15.2 Observability

Log fields on resolve/compose:

- `profile.registry`
- `profile.slug`
- `profile.version`
- `runtime.key`
- `runtime.fingerprint_hash`

Expose metrics:

- profile resolve latency
- profile cache hit/miss (if caching added)
- profile mutation counts by registry/source

### 15.3 Caching

Optional phase-2 optimization:

- Cache resolved profiles by `(registry, slug, version, override hash)`.
- Invalidate on write or store notification.

## 16. Open Questions and Decisions Needed

1. Should runtime overrides be globally enabled by default or default-denied except explicit keys?
2. Should profile CRUD be universally available or gated by app role/auth mode?
3. Should `system_prompt/middlewares/tools` remain first-class fields long-term, or be represented only as `StepSettingsPatch` + extension fields?
4. Do we need cross-registry profile import/export endpoints in phase 1?
5. Is SQLite the primary persistent store for all deployments, or do we require Postgres in near-term scope?

## 17. Recommended Implementation Order (1-Week Sprint Breakdown)

### Sprint A (Core)

- Geppetto profile domain + legacy YAML adapter + resolver tests.
- Registry-backed middleware adapter for command parsing.

### Sprint B (Server)

- Pinocchio resolver/composer integration with new registry service.
- Profile CRUD endpoints with in-memory + SQLite store options.

### Sprint C (Client)

- Go-Go-OS profile API client and state slice.
- UI profile select/create/edit flows in inventory app.

### Sprint D (Cleanup)

- Deprecate low-level AI flags on user-facing commands.
- Documentation update and migration notes for existing operators.

## 18. Final Recommendation

Implement `ProfileRegistry` as a Geppetto-owned domain service with pluggable stores and shared resolver semantics, then consume it from Pinocchio runtime resolution and Go-Go-OS chat clients. Keep existing profile-file and command behaviors through adapters while migrating toward profile-first runtime composition. This yields a unified architecture with lower config surface area, better web UX, and direct path to DB-backed profile management.

