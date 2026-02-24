---
Title: Webchat Rename Implementation Plan
Ticket: GP-02-WEBCHAT-RENAME-SYMBOLS
Status: active
Topics:
    - architecture
    - backend
    - chat
    - pinocchio
    - migration
DocType: planning
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/profile_policy.go
      Note: |-
        Primary request resolver naming cleanup surface
        Planned resolver symbol renames and helper cleanup
    - Path: ../../../../../../../pinocchio/cmd/web-chat/runtime_composer.go
      Note: |-
        Runtime composer symbol and helper naming cleanup surface
        Planned composer symbol renames and helper cleanup
    - Path: ../../../../../../../pinocchio/pkg/inference/runtime/composer.go
      Note: |-
        Runtime compose contract naming cleanup boundary
        Planned runtime compose contract field renames
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: ConvManager compose call-sites impacted by rename
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation_service.go
      Note: |-
        Internal service request model naming cleanup boundary
        Planned service request DTO naming migration
    - Path: ../../../../../../../pinocchio/pkg/webchat/http/api.go
      Note: |-
        Request/plan DTO naming cleanup boundary
        Planned request/plan DTO naming migration
    - Path: ../../../../../../../pinocchio/pkg/webchat/stream_hub.go
      Note: Runtime request propagation call-sites impacted by rename
    - Path: pkg/sections/profile_registry_feature_flag.go
      Note: |-
        Cross-repo environment variable name clarity follow-up
        Planned env var rename and compatibility fallback
ExternalSources: []
Summary: Staged implementation plan for renaming confusing webchat symbols/APIs to clearer profile/runtime semantics with compatibility shims, test updates, and rollout safety.
LastUpdated: 2026-02-23T15:30:57.731698639-05:00
WhatFor: Normalize confusing webchat naming so contributors can reason about profile selection, runtime composition, and request flow without ambiguous symbols.
WhenToUse: Use when implementing or reviewing GP-02 rename PRs, planning compatibility windows, and validating naming migration coverage.
---


# Webchat Rename Implementation Plan

## 1. Objective

This ticket exists to rename confusing symbols in the Pinocchio webchat runtime path so the code reflects what it actually does:

- profile selection and policy resolution;
- runtime composition from resolved profile runtime;
- runtime lifecycle decisions in conversation manager.

The goal is readability and API clarity, not behavior changes.

## 2. Scope and Boundaries

### In scope

- symbol and API names in webchat/profile-runtime flow;
- internal DTO/contract names where ambiguity hurts comprehension;
- compatibility aliases/shims where needed to avoid high-risk breaking changes in one pass;
- tests, docs, and call-sites touched by renames.

### Out of scope

- changing runtime precedence semantics;
- changing policy behavior;
- introducing new profile CRUD behavior;
- removing compatibility endpoints in this ticket.

## 3. Naming Principles

Use these rules for every rename decision:

1. Prefer domain-accurate names over historical names.
2. Distinguish request input from effective/merged runtime data.
3. Avoid overloaded terms like `RuntimeKey` when value is profile slug.
4. Keep names stable across layers (resolver -> HTTP DTO -> service -> composer).
5. Use compatibility aliases where downstream breakage risk is high.

## 4. Proposed Rename Matrix

## 4.1 High Priority (core clarity)

| Current | Proposed | Notes |
|---|---|---|
| `webChatProfileResolver` | `ProfileRequestResolver` | Better expresses request-policy role |
| `webChatRuntimeComposer` | `ProfileRuntimeComposer` | Composer is profile-aware; name should say so |
| `ConversationRequestPlan` | `ResolvedConversationRequest` | Clarifies this is post-resolution output |
| `AppConversationRequest` | `ConversationRuntimeRequest` | Clarifies this is a runtime-resolution input model |
| `RuntimeComposeRequest.RuntimeKey` | `ProfileKey` | Value is profile selector in current flow |
| `RuntimeComposeRequest.ResolvedRuntime` | `ResolvedProfileRuntime` | Removes ambiguity around payload origin |
| `RuntimeComposeRequest.Overrides` | `RuntimeOverrides` | Explicit runtime scope |
| `resolveProfile` | `resolveProfileSelection` | More precise function intent |
| `runtimeKeyFromPath` | `profileSlugFromPath` | Path segment currently maps to profile slug |
| `baseOverridesForProfile` | `runtimeDefaultsFromProfile` | Reflects defaults generation behavior |
| `mergeOverrides` | `mergeRuntimeOverrides` | Scope and payload type clarity |

## 4.2 Medium Priority (ergonomics)

| Current | Proposed | Notes |
|---|---|---|
| `newInMemoryProfileRegistry` | `newInMemoryProfileService` | Current function returns service interface, not raw registry object |
| `registerProfileHandlers` | `registerProfileAPIHandlers` | Distinguishes API route registration from internal helpers |
| `runtimeFingerprint` | `buildRuntimeFingerprint` | Verb-driven utility naming |
| `runtimeFingerprintPayload` | `RuntimeFingerprintInput` | Struct intent clarity |
| `validateOverrides` | `validateRuntimeOverrides` | Scope clarity |
| `parseMiddlewareOverrides` | `parseRuntimeMiddlewareOverrides` | Scope clarity |
| `parseToolOverrides` | `parseRuntimeToolOverrides` | Scope clarity |
| `defaultWebChatRegistrySlug` | `defaultRegistrySlug` | Less noisy, still clear in package context |

## 4.3 Low Priority / Cross-Cut

| Current | Proposed | Notes |
|---|---|---|
| `chat_profile` cookie literal | `currentProfileCookieName` constant | Remove repeated magic string |
| `PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` | `GEPPETTO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE` | Better ownership semantics (requires migration guard) |

## 5. Compatibility Strategy

This rename should be staged, not big-bang:

1. Add new symbols first.
2. Keep old names as wrappers/type aliases with deprecation comments.
3. Migrate internal call-sites and tests.
4. Remove old names in a later cleanup PR after downstream consumers are moved.

Pseudocode strategy:

```go
// Phase A compatibility shim
type ConversationRequestPlan = ResolvedConversationRequest

// Deprecated: use ProfileRequestResolver.
type webChatProfileResolver = ProfileRequestResolver
```

For env var rename, support dual-read window:

```go
if readBool("GEPPETTO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE") {
  return true
}
return readBool("PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE")
```

Then log deprecation and remove old var in a follow-up release.

## 6. Implementation Sequence

## Phase A: DTO and contract renames

- rename request/plan/service/runtime contract fields and types;
- add aliases where exported API compatibility is required;
- update all compile-time call-sites.

## Phase B: Resolver and composer symbols

- rename resolver/composer types and helper methods;
- rename override helper family;
- keep wrappers for transitional stability.

## Phase C: Constants, literals, and env flags

- introduce constants for cookie and magic strings;
- add env var dual-read behavior for feature flag rename;
- add explicit deprecation comments and docs.

## Phase D: Cleanup removal

- remove deprecated aliases after downstream repos are migrated;
- run grep sweep to ensure old names do not remain.

## 7. Affected File Groups

Primary files:

- `pinocchio/cmd/web-chat/profile_policy.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/pkg/webchat/stream_hub.go`
- `pinocchio/pkg/webchat/conversation.go`

High-impact tests:

- `pinocchio/cmd/web-chat/profile_policy_test.go`
- `pinocchio/cmd/web-chat/runtime_composer_test.go`
- `pinocchio/pkg/webchat/conversation_service_test.go`
- `pinocchio/pkg/webchat/http_helpers_contract_test.go`
- `pinocchio/pkg/webchat/stream_hub_test.go`
- `pinocchio/pkg/webchat/chat_service_test.go`

## 8. Validation Plan

Compile/test baseline:

```bash
cd pinocchio
go test ./cmd/web-chat -count=1
go test ./pkg/webchat/... -count=1
go test ./pkg/inference/runtime -count=1
go test ./... -count=1
```

Static verification:

```bash
rg -n "webChatProfileResolver|webChatRuntimeComposer|ConversationRequestPlan|AppConversationRequest|RuntimeKey"
```

Acceptance checks:

- all tests pass;
- no old-name references in non-compatibility files;
- API behavior unchanged;
- docs updated for renamed public symbols.

## 9. Risks and Mitigations

Risk: accidental behavior changes during rename.  
Mitigation: no logic edits mixed with rename PRs unless required for compile.

Risk: breakage in downstream imports.  
Mitigation: aliases + deprecation window.

Risk: confusion during mixed-name period.  
Mitigation: central tracking checklist and explicit docs of old->new mapping.

Risk: env var rename silently disabling features.  
Mitigation: dual-read + warning log + staged removal.

## 10. Definition of Done

Ticket is done when:

- rename matrix items are implemented or explicitly deferred with rationale;
- compatibility policy is applied consistently;
- tests are green and old names are reduced to intentional shims only;
- docs/tasks/changelog capture migration outcomes and deprecations.

## 11. Review Checklist

- Are names domain-accurate and consistent across layers?
- Did we separate request overrides from effective runtime defaults?
- Are compatibility aliases clearly marked deprecated?
- Was env var migration handled safely?
- Are test updates complete and meaningful?
