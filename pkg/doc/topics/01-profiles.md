---
Title: Profile Registry in Geppetto
Slug: profiles
Short: Registry-first profile model for selecting runtime defaults, policy, and persistence across apps.
Topics:
- configuration
- profiles
- registry
- migration
Commands:
- geppetto
Flags:
- profile
- profile-file
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Profile Registry in Geppetto

Geppetto now treats profiles as a first-class domain object (`ProfileRegistry`) instead of a loose map loaded only for flag overlays. This gives you one consistent model for:

- selecting runtime defaults by profile slug,
- storing profiles in memory, YAML, or SQLite,
- enforcing profile policy (for example read-only or override restrictions),
- exposing profile CRUD APIs from applications.

This page documents the canonical, registry-first model used by current pinocchio and go-go-os integrations.

## Why Registry-First Profiles

Registry-first profiles make profile state explicit and reusable across CLI flows and HTTP services.

Key benefits:

- **Reusable domain model**: `Profile`, `ProfileRegistry`, `RuntimeSpec`, `PolicySpec`.
- **Typed slugs**: `RegistrySlug`, `ProfileSlug`, `RuntimeKey` reduce stringly-typed errors.
- **Store abstraction**: same service API over in-memory, YAML file, or SQLite stores.
- **Policy + versioning**: profile metadata and optimistic concurrency are built in.
- **Application integration**: pinocchio/go-go-os can list, select, create, and update profiles through APIs.

## Core Data Model

The registry domain lives in `geppetto/pkg/profiles`.

```go
type Profile struct {
    Slug        ProfileSlug
    DisplayName string
    Description string
    Runtime     RuntimeSpec
    Policy      PolicySpec
    Metadata    ProfileMetadata
}

type ProfileRegistry struct {
    Slug               RegistrySlug
    DefaultProfileSlug ProfileSlug
    Profiles           map[ProfileSlug]*Profile
    Metadata           RegistryMetadata
}
```

`RuntimeSpec` carries runtime defaults such as:

- `system_prompt`
- `middlewares`
- `tools`
- `step_settings_patch`

`PolicySpec` controls profile mutability and request override behavior:

- `allow_overrides`
- `allowed_override_keys`
- `denied_override_keys`
- `read_only`

## Slug Types and Validation

Use typed slug constructors at boundaries:

```go
registrySlug, err := profiles.ParseRegistrySlug("default")
profileSlug, err := profiles.ParseProfileSlug("agent")
```

Or panic-on-invalid helpers in trusted bootstrap code:

```go
registrySlug := profiles.MustRegistrySlug("default")
profileSlug := profiles.MustProfileSlug("agent")
```

Typed slugs are serialized to JSON/YAML and normalize values before storage, which keeps API and storage behavior consistent.

## Store and Service Layers

### Store Implementations

- `InMemoryProfileStore` for tests and embedded defaults.
- YAML file codec/store for local file workflows.
- `SQLiteProfileStore` for durable multi-process usage.

### Service API

Applications typically use the `profiles.Registry` service interface:

- `ListRegistries`
- `GetRegistry`
- `ListProfiles`
- `GetProfile`
- `ResolveEffectiveProfile`
- `CreateProfile`
- `UpdateProfile`
- `DeleteProfile`
- `SetDefaultProfile`

For storage-backed services, use `profiles.NewStoreRegistry(...)`.

## Resolution Flow

`ResolveEffectiveProfile` merges profile runtime defaults with optional request overrides and returns canonical output used by runtime composition.

Resolution output includes:

- selected registry/profile/runtime key
- effective runtime fields
- effective step settings
- fingerprint input metadata

Request overrides are policy-gated. If a profile denies overrides or marks keys as denied, resolution returns a policy violation error.

## Hard-Cutover Model

The runtime model is registry-first and profile-first:

- profiles are stored and resolved through `profiles.Registry`,
- profile CRUD is the write path for runtime defaults,
- middleware configuration is profile-scoped and validated before persistence in API surfaces.

There is no environment-variable toggle for old middleware selection paths.

## Registry YAML Example

```yaml
slug: default
default_profile_slug: agent
profiles:
  agent:
    slug: agent
    display_name: Agent
    runtime:
      system_prompt: You are an assistant.
      tools:
        - calculator
    policy:
      allow_overrides: true
```

## Typed Extension Keys

`profile.extensions` use namespaced typed keys:

```text
<namespace>.<feature>@v<version>
```

Examples:

- `webchat.starter_suggestions@v1`
- `inventory.starter_suggestions@v1`

Keys are normalized and validated; malformed keys fail profile create/update.

## Middleware Config Ownership and Validation Timing

Middleware configuration lives inside the profile runtime:

```yaml
runtime:
  middlewares:
    - name: agentmode
      id: default
      config:
        default_mode: financial_analyst
```

Authoritative behavior:

- middleware identity/order/enable state are profile runtime concerns,
- middleware config payloads are validated against middleware JSON schema,
- unknown middleware names are hard errors in write-time validated APIs.

## CLI and Environment Selection

Profile selection remains available through:

- config (`profile-settings` section),
- environment (`PINOCCHIO_PROFILE`, `PINOCCHIO_PROFILE_FILE`),
- flags (`--profile`, `--profile-file`).

Precedence remains:

**flags > env > config > profiles > defaults**

Profile-first selection is the recommended path. Direct engine/provider flags (`--ai-engine`, `--ai-api-type`) are migration escape hatches, not the default operator workflow.

Registry-backed middleware resolution is now the only path; there is no environment toggle for legacy middleware switching.

## Application Patterns

Use this baseline in apps:

1. Bootstrap a registry store (in-memory/YAML/SQLite).
2. Build a `profiles.Registry` service.
3. Resolve request profile selection through the service.
4. Feed resolved runtime and profile version into runtime composition.
5. Optionally mount profile CRUD HTTP endpoints.

This keeps profile logic centralized and avoids app-specific shadow profile structures.

## Schema Discovery in Application APIs

Application APIs can expose schema catalogs for frontend form generation:

- `GET /api/chat/schemas/middlewares`
- `GET /api/chat/schemas/extensions`

Middleware schema items include UI metadata and the JSON Schema payload:

```json
[
  {
    "name": "agentmode",
    "version": 1,
    "display_name": "Agent Mode",
    "description": "Injects and parses mode control output.",
    "schema": {
      "type": "object",
      "properties": {
        "default_mode": { "type": "string" }
      }
    }
  }
]
```

Extension schema items are keyed by typed extension key:

```json
[
  {
    "key": "middleware.agentmode_config@v1",
    "schema": {
      "type": "object",
      "properties": {
        "instances": {
          "type": "object",
          "additionalProperties": { "type": "object" }
        }
      },
      "required": ["instances"],
      "additionalProperties": false
    }
  }
]
```

For extension discovery, the merge order is deterministic:

1. explicit extension schema docs supplied by the application,
2. middleware-derived extension schemas (`middleware.<name>_config@v1`),
3. codec-discovered schemas from `ExtensionCodecRegistry` entries that implement:
   - `ExtensionCodecLister` (registry supports listing codecs),
   - `ExtensionSchemaCodec` (codec exposes `JSONSchema()`).

This keeps the endpoint extensible without hardcoding all extension keys in app code. Profile mutations still go through profile CRUD endpoints.

## Typed-Key Middleware Config Examples

Middleware config values are persisted in `profile.extensions` under middleware typed keys.

JSON profile excerpt:

```json
{
  "slug": "analyst",
  "runtime": {
    "middlewares": [
      { "name": "agentmode", "id": "default", "enabled": true }
    ]
  },
  "extensions": {
    "middleware.agentmode_config@v1": {
      "instances": {
        "id:default": {
          "default_mode": "financial_analyst"
        }
      }
    }
  }
}
```

YAML profile excerpt:

```yaml
slug: analyst
runtime:
  middlewares:
    - name: agentmode
      id: default
      enabled: true
extensions:
  middleware.agentmode_config@v1:
    instances:
      "id:default":
        default_mode: financial_analyst
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `profile not found` | Unknown slug or wrong registry | Verify selected profile and registry slugs; check default profile on registry |
| `registry not found` | Store missing bootstrap registry | Ensure bootstrap upsert runs before service start |
| policy violation on update/delete | Profile is read-only or blocked by policy | Adjust policy intentionally or mutate a writable profile |
| stale update conflict | `expected_version` does not match latest | Re-fetch profile, then retry update with current version |
| `validation error (runtime.middlewares[*].name)` | Unknown middleware definition | Fix middleware name or register definition in application runtime |
| `validation error (runtime.middlewares[*].config)` | Middleware payload fails schema validation | Correct payload shape/types based on middleware schema endpoint |
| runtime did not switch after profile change | Resolver/composer not using profile version/fingerprint inputs | Ensure request plan includes `ProfileVersion` and runtime composer fingerprint includes version-sensitive inputs |

## See Also

- [Geppetto Documentation Index](00-docs-index.md)
- [Operate SQLite-backed profile registry](../playbooks/06-operate-sqlite-profile-registry.md)
- [Middlewares](09-middlewares.md)
- [Session Management in Geppetto](10-sessions.md)
- `geppetto/pkg/profiles/*` (implementation package)
