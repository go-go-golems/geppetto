---
Title: Diary
Ticket: GP-70
Status: active
Topics:
    - geppetto
    - engine-profiles
    - inference
    - model-metadata
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/engineprofiles/types.go:Core EngineProfile types where ModelInfo will be consumed
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/steps/ai/settings/settings-inference.go:InferenceSettings where ModelInfo field will be added
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/steps/ai/settings/settings-chat.go:ChatSettings pattern for pointer-typed optional fields
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/inference/engine/inference_config.go:InferenceConfig merge pattern to follow
    - /home/manuel/workspaces/2026-05-06/add-model-metadata/geppetto/pkg/inference/engine/factory/factory.go:isReasoningModel heuristic to replace
ExternalSources: []
Summary: "Implementation diary for GP-70: adding model metadata to engine profiles"
LastUpdated: 2026-05-06T10:55:00.000000000-04:00
WhatFor: "Track the investigation, design, and implementation steps for model metadata"
WhenToUse: "When resuming work on GP-70 or reviewing the implementation"
---

# Diary

## Goal

Capture the step-by-step implementation journey for adding model metadata (reasoning, input modalities, context window with quality high-watermark, costs, metadata grab-bag) to the geppetto engine profile system.

## Step 1: Codebase Investigation and Design Document

This step covered the full investigation of the geppetto/pinocchio codebase to understand the existing engine profile system, inference settings, stack merge semantics, and extension key patterns. The output is a comprehensive design document written for a new intern.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add metadata to models regarding thinking, input types, costs, context window size (which is both quality high watermark and absolute size, to be able to indicate when the quality of a model degrades). It is for example data like this: {\"id\": \"DeepSeek-V4-Pro\", \"name\": \"DeepSeek V4 Pro\", \"reasoning\": true, \"input\": [\"text\"], \"contextWindow\": 262144, \"maxTokens\": 32768, \"cost\": {\"input\": 0, \"output\": 0, \"cacheRead\": 0, \"cacheWrite\": 0}}. It should be loaded from profiles as part of inference settings, and be both typed data for the above fields, and also allow a map[string]any{} metadata grab bag. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable. Keep a diary as you work."

**Assistant interpretation:** Create a docmgr ticket, write a detailed design/implementation guide for adding model metadata to engine profiles, maintain a diary, and upload to reMarkable.

**Inferred user intent:** Produce an exhaustive, intern-friendly design document that covers the full system context, proposed architecture, and phased implementation plan. Deliver it as both a docmgr ticket document and a reMarkable PDF.

### What I did

- Explored the three-repo workspace (geppetto, pinocchio, glazed) to understand the architecture
- Read and analyzed 15+ key source files covering engine profiles, inference settings, stack merge, extension keys, engine factory, inference config, JS module surface, and inference results
- Created docmgr ticket GP-70 with design-doc and diary documents
- Wrote a comprehensive design document (~45KB) covering:
  - Executive summary
  - Problem statement with gap analysis table
  - Current-state architecture (9 subsections explaining every relevant component)
  - Proposed architecture with type sketches, field tables, merge semantics, YAML format, and API reference
  - Data flow diagrams (profile resolution, cost computation, context budgeting)
  - 6-phase implementation plan with specific file references
  - Testing strategy
  - Risks and alternatives
  - Open questions
  - Full reference table of key files
- Revised the document to remove all full Go implementation code, keeping only pseudocode, type sketches, and prose descriptions — leaving freedom for the implementor to make their own decisions

### Why

The design needs to be self-contained enough for a new intern to understand the entire profile system, not just the new feature. Without the current-state walkthrough, the intern would have to reverse-engineer the architecture from the code.

### What worked

- Reading the actual source files gave concrete evidence for every claim in the design
- Following the ProfileRuntime extension example (pinocchio/pkg/inference/runtime/profile_runtime.go) provided a proven pattern for how to integrate new typed data
- The three-way decision on ModelInfo placement (InferenceSettings only vs extension only vs both) led to a clean decision (Option A) with explicit rationale

### What didn't work

- Initially considered making ModelInfo both an InferenceSettings field AND a profile extension, but realized this creates a synchronization problem. Settled on InferenceSettings as the single source of truth.

### What I learned

- The stack merge system already handles InferenceSettings field-by-field through YAML round-trip, so adding ModelInfo there means it merges "for free" without custom merge code
- The `isReasoningModel()` heuristic in factory.go is exactly the kind of hardcoding that model metadata replaces
- `InferenceResult.Usage` already has all the token fields needed for cost computation—no new tracking needed

### What was tricky to build

- The quality high-watermark concept required careful thought about semantics: it's not just a "recommended limit" but a "quality degradation threshold" that differs from the hard context window. The two-field approach (context_window + quality_high_watermark) is novel but maps to real provider behavior (e.g., Gemini 1M context with quality degradation beyond ~500K).
- Removing full Go implementation code while keeping enough detail to be actionable required balancing specificity vs. implementor freedom. The solution was to use field tables, merge rule pseudocode, and signature sketches instead of concrete struct definitions and function bodies.

### What warrants a second pair of eyes

- The decision to use value types (not pointers) for `ModelCost` fields—nil `Cost` means unknown, `Cost{}` means free. Make sure this doesn't create confusion when merging costs through the stack (wholesale replacement of Cost, not field-by-field).
- The validation that `quality_high_watermark <= context_window`—should this be a warning or an error?

### What should be done in the future

- Implement Phase 1 (core types) first, then Phase 2 (profile integration), then validate with YAML fixtures before moving to later phases
- Add model metadata to the existing example profile YAML files in the repo
- Consider a CLI command to dump model info for a resolved profile

### Code review instructions

- Start in `design-doc/01-model-metadata-design-and-implementation-guide.md`
- Key architectural decisions to validate: ModelInfo in InferenceSettings (not extension-only), ModelCost as value types, quality_high_watermark as separate field from context_window
- Cross-reference the proposed types against `InferenceConfig` and `ChatSettings` for pattern consistency

### Technical details

- docmgr ticket: GP-70
- Design doc path: `geppetto/ttmp/2026/05/06/GP-70--add-model-metadata-to-engine-profiles/design-doc/01-model-metadata-design-and-implementation-guide.md`
- Diary path: `geppetto/ttmp/2026/05/06/GP-70--add-model-metadata-to-engine-profiles/reference/01-diary.md`
- reMarkable upload: `/ai/2026/05/06/GP-70/GP-70 Model Metadata Design.pdf`
- docmgr doctor: passed (all checks clean)
- 9 tasks created
- 5 files related to ticket
- Vocabulary added: `engine-profiles`, `model-metadata`

## Step 2: Implement Model Metadata Across Geppetto and Pinocchio

This step implemented the model metadata design in the codebase. The core support now lives in Geppetto as `InferenceSettings.ModelInfo`, with typed model capabilities, context limits, pricing, merge semantics, YAML/profile integration, JS exposure, and post-inference cost stamping. Pinocchio now surfaces model metadata in the profile picker and profile API.

The implementation intentionally kept `ModelInfo` as part of `InferenceSettings` rather than adding a second profile-extension source of truth. This preserved the normal profile stack behavior and avoided synchronization problems between `inference_settings` and `extensions`.

### Prompt Context

**User prompt (verbatim):** "implement all of the tasks. Keep commits at appropriate intervals, keep a detailed diary (see skill)."

**Assistant interpretation:** Implement the GP-70 task list, commit at logical milestones, keep the docmgr diary updated, and validate with tests.

**Inferred user intent:** Move from design-only ticket to working code across Geppetto and Pinocchio, preserving reviewability and traceability.

**Commit (code):** `bfcb0d00ad4f3a2b69d6bbb35ab918a69dee858b` — "Geppetto: add model metadata to inference profiles"

**Commit (code):** `d43a21434d00f61fc266213d58f11c56f7c2ca9d` — "Geppetto: accept model metadata in JS engine config"

**Commit (code):** `d53ada144636d14afd7fcbf4f3b7a311e1f6284a` — "Pinocchio: expose profile model metadata"

### What I did

- Added `ModelInfo`, `ModelCost`, and `InputModality` in `geppetto/pkg/steps/ai/settings/model_info.go`.
- Added context limit helpers, validation, clone support, merge support, and cost computation helpers.
- Added `ModelInfo *ModelInfo` to `InferenceSettings` and updated clone, metadata flattening, and summary rendering.
- Updated profile stack merge to treat `model_info.cost` as a wholesale replacement instead of recursively merging partial cost maps.
- Added YAML/profile decode and stack merge tests for `model_info`.
- Added `model_info` to `geppetto/examples/js/geppetto/profiles/10-provider-openai.yaml`.
- Updated reasoning-model decisions in Geppetto factory/OpenAI/OpenAI Responses paths to prefer `ModelInfo.Reasoning` and fall back to name heuristics.
- Added `Cost *float64` to `turns.InferenceResult` and stamped computed costs in OpenAI, OpenAI Responses, Claude, and Gemini result persistence paths.
- Exposed `modelInfo` to the Geppetto JS module on resolved profiles and engine objects.
- Accepted `modelInfo` in `geppetto.engines.fromConfig()` JS options.
- Updated Geppetto TypeScript declaration template with `ModelInfo` and `ModelCost` interfaces.
- Updated Pinocchio profile switcher list items to carry model info and render capability summaries.
- Updated Pinocchio web-chat profile API list/detail responses to include `model_info`.
- Rewrote ticket tasks as checked implementation tasks with notes for intentional no-op decisions.

### Why

The implementation follows the ticket design: model metadata is profile-loaded, typed, merged with inference settings, visible to JS/UI consumers, and connected to cost computation through existing inference usage data. Pinocchio integration focuses on surfacing the metadata rather than introducing a new token-budgeting subsystem.

### What worked

- YAML round-trip profile loading picked up `model_info` through struct tags as expected.
- The existing `MergeInferenceSettings` map merge could be reused with a small special case for `model_info.cost` wholesale replacement.
- Existing provider engines already had access to their `InferenceSettings`, so cost stamping could happen in provider persistence paths without creating an import cycle from `inference/engine` back to `settings`.
- JS module tests were straightforward because resolved profiles already expose `inferenceSettings`; `modelInfo` is now a direct convenience property.

### What didn't work

- Directly adding model-aware cost computation to `BuildInferenceResultFromEventMetadata()` would have introduced an import cycle (`settings` already imports `inference/engine`). I avoided this by adding `settings.ApplyModelInfoCost()` and calling it from provider packages.
- Implementing real web-chat prompt trimming was not done in this pass because no token-budgeting/trimming layer was present at the web-chat resolver boundary. Instead, `ModelInfo` is exposed through the API/runtime path so a tokenizer-aware layer can use `EffectiveContextLimit()` and `HardContextLimit()`.

### What I learned

- `InferenceSettings.Clone()` previously assumed all nested settings pointers were non-nil. Updating it to be nil-safe made ModelInfo cloning safer and reduced future YAML edge cases.
- The profile API and TUI profile picker list raw profile documents, not fully resolved stacks. Metadata displayed there reflects the profile document being listed; resolved paths still carry merged `InferenceSettings.ModelInfo`.
- Cost should be stamped close to provider result persistence because those paths know both final usage metadata and the model settings used for the call.

### What was tricky to build

- Avoiding import cycles was the main sharp edge. The natural place to compute cost looked like `inference/engine`, but that package cannot import `steps/ai/settings` because settings already imports engine for `InferenceConfig`. The solution was to keep `ComputeCost` and `ApplyModelInfoCost` in the settings package and call them from provider packages.
- Merge semantics required care: recursive map merge is normally desired, but partial cost merge would preserve stale base output/cache costs. The implementation special-cases `model_info.cost` so overlay cost replaces base cost as a unit.

### What warrants a second pair of eyes

- Review `MergeInferenceSettings` special-casing for `model_info.cost` to ensure it is not too narrow and handles YAML-decoded map shapes consistently.
- Review cost stamping locations across OpenAI, OpenAI Responses, Claude, and Gemini to ensure every successful inference result path is covered.
- Review Pinocchio profile picker rendering for terminal compatibility with emoji badges.
- Review the decision not to add CLI flags for model metadata in v1.

### What should be done in the future

- Add a tokenizer-aware prompt budgeting/trimming layer in web-chat that uses `EffectiveContextLimit()` and `HardContextLimit()`.
- Consider a model catalog updater or profile generator for keeping provider pricing/context metadata current.
- Add frontend rendering for `model_info` returned by the web-chat profile API.

### Code review instructions

- Start with `geppetto/pkg/steps/ai/settings/model_info.go` and `settings-inference.go`.
- Then review merge behavior in `geppetto/pkg/engineprofiles/inference_settings_merge.go` and tests in `stack_merge_model_info_test.go`.
- Review provider result stamping in `pkg/steps/ai/{openai,openai_responses,claude,gemini}`.
- Review JS exposure in `geppetto/pkg/js/modules/geppetto/api_runtime_metadata.go` and `api_engines.go`.
- Review Pinocchio exposure in `pinocchio/pkg/ui/profileswitch/*` and `pinocchio/cmd/web-chat/profiles/*`.
- Validate with:
  - `cd geppetto && go test ./...`
  - `cd pinocchio && go test ./pkg/ui/profileswitch ./cmd/web-chat/profiles ./pkg/cmds/profilebootstrap`
  - Pinocchio pre-commit also ran `go generate ./...`, frontend build, `go build ./...`, golangci-lint, `go vet`, and `go test ./...` successfully during commit.

### Technical details

- Geppetto full test command passed: `go test ./...`.
- Pinocchio targeted test command passed: `go test ./pkg/ui/profileswitch ./cmd/web-chat/profiles ./pkg/cmds/profilebootstrap`.
- Pinocchio pre-commit hook ran and passed full lint/test pipeline during commit `d53ada1`.
