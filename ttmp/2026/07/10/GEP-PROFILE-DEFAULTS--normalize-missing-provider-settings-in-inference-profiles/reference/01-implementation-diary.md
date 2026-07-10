---
Title: Implementation diary
Ticket: GEP-PROFILE-DEFAULTS
Status: active
Topics:
    - javascript
    - profiles
    - inference
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/js/modules/geppetto/api_agent.go
      Note: Agent builder propagates default-constructor errors (commit d0557f7f)
    - Path: repo://pkg/js/modules/geppetto/api_agent_profile_test.go
      Note: Offline JavaScript sparse-profile request-construction regression coverage (commit d0557f7f)
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: Engine builder propagates default-constructor errors (commit d0557f7f)
    - Path: repo://pkg/js/modules/geppetto/api_engines.go
      Note: |-
        Normalization defect and planned provider-aware repair
        Provider-aware default normalization implementation (commit d0557f7f)
ExternalSources: []
Summary: Chronological evidence, commands, design decisions, failures, validation, and commits for provider-profile default normalization.
LastUpdated: 2026-07-10T19:04:30.981617688-04:00
WhatFor: Record every research, implementation, validation, and commit checkpoint for provider-aware profile default normalization.
WhenToUse: Read before resuming this ticket and update after each completed task or diagnostic result.
---



# Implementation diary

## Goal

Make sparse, registry-resolved inference profiles safe for Geppetto JavaScript agent execution while preserving explicit configuration and provider support boundaries.

## Step 1: Capture failure and create implementation ticket

The work began from a real transcript-RAG live summarization experiment. A profile used Geppetto's OpenAI-compatible engine to call a local Ollama `/v1` endpoint. After explicit local HTTP and loopback policy opt-ins, request construction failed because sparse profile YAML had not materialized `Client` and `OpenAI` settings objects.

This ticket isolates the defect in Geppetto. The JavaScript layer already owns a function named `ensureInferenceSettingsProviderDefaults`, but it currently initializes only `API`. The design expands that function into a provider-aware normalizer and adds network-free regression coverage.

### Prompt Context

**User prompt (verbatim):** "ok, create a new docmgr ticket for geppetto in /home/manuel/workspaces/2026-07-10/fix-geppetto-inference-profiles (in geppetto/ttmp) to address this, as well as a analysis/design/implementation guide that explains the issue. then add tasks and work on them one by one, committing at appropriate intervals and keeping a detailed diary as you work"

**Assistant interpretation:** Create a Geppetto ticket in the specified checkout, document the analysis and implementation plan, then implement the provider-profile fix with task-level progress, commits, and diary evidence.

**Inferred user intent:** Make concise inference profiles reliable for JavaScript execution without forcing every profile author to add empty provider blocks.

### What I did

- Created ticket `GEP-PROFILE-DEFAULTS`, design guide, diary, and seven ordered tasks.
- Inspected the JS normalization boundary, both builder call sites, profile merge behavior, settings constructors, engine factory validation, and provider request construction.
- Recorded the observed errors: `missing client settings` and `no openai settings`.
- Identified the provider matrix and constrained defaults to runtime-owned objects.

### Why

- The defect is cross-provider configuration behavior and belongs in Geppetto, not in a consuming application profile.
- A design-first ticket makes it possible to test all affected providers rather than treating the OpenAI failure alone.

### What worked

- The requested ticket workspace was created successfully despite a malformed existing docmgr vocabulary file.
- Code inspection confirmed that `agent().inference()` and `engine().inference()` already call a shared normalizer after cloning settings.
- Existing test coverage supplied a direct place to add JavaScript regression assertions.

### What didn't work

- `docmgr vocab list --category topics` failed because `ttmp/vocabulary.yaml` contains unresolved conflict markers at line 132:

  ```text
  yaml: line 132: could not find expected ':'
  ```

- Ticket creation and document creation nevertheless succeeded. The unrelated vocabulary conflict is not changed by this ticket.

### What I learned

- Building an agent/session is insufficient regression coverage: classic OpenAI dereferences `Client` and `OpenAI` only when a request is built.
- `chat.api_type: ollama` remains unsupported by the standard engine factory; this normalization work must not disguise that limitation.

### What was tricky to build

- Profile YAML, profile stacking, Go-owned JS wrappers, and provider engines have separate ownership layers. The correct mutation point is the clone in the JS builder, not YAML decoding or the registry object.
- Some settings constructors can return errors, so the normalizer must return errors rather than remain a void helper.

### What warrants a second pair of eyes

- Confirm the full provider matrix and whether Responses should always receive `Client` defaults.
- Confirm tests can reach deterministic request construction without performing network I/O.
- Keep the unrelated malformed vocabulary file out of this ticket's commit.

### What should be done in the future

- Implement and test provider-aware default creation.
- Run focused and full validation, then update this diary with exact commit hashes.

### Code review instructions

- Start with the design guide and `pkg/js/modules/geppetto/api_engines.go`.
- Compare engine dereferences in OpenAI, Claude, and Gemini before reviewing the tests.
- Run the focused Go test commands in the design guide.

### Technical details

```text
GoError: missing client settings
GoError: no openai settings
```

The expected runtime sequence is:

```text
profile YAML -> registry resolve -> JS wrapper -> clone -> normalize -> engine -> session request construction
```

## Step 2: Normalize provider defaults and prove sparse-profile request construction

The implementation changed the existing JavaScript normalization hook into an error-returning provider-aware function. It now supplies complete API maps and a client object for a selected chat provider, then materializes only the provider settings object required by that provider. The agent and engine builders propagate constructor errors instead of proceeding with partial settings.

The test suite covers direct normalization for classic OpenAI-compatible providers, OpenAI Responses, Claude and its `anthropic` alias, Gemini, and unsupported Ollama. The JavaScript regression resolves a stacked sparse profile, builds an agent through `require("geppetto")`, obtains the Go-owned agent reference inside the module test, and constructs an OpenAI request without making a network call.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the profile-default repair task by task, validate it at the real JavaScript-to-provider boundary, and commit the result.

**Inferred user intent:** A consumer should be able to omit empty `openai`, `client`, `claude`, or `gemini` blocks from profile YAML without delayed nil-pointer-like failures during inference.

**Commit (code):** `d0557f7fba7afc7048a922cb707407775da93c67` — "Normalize sparse JavaScript inference profiles"

### What I did

- Changed `ensureInferenceSettingsProviderDefaults` to return `error`.
- Initialized missing API maps (`APIKeys`, `BaseUrls`, `AllowHTTP`, and `AllowLocalNetworks`) and `Client` defaults.
- Initialized `OpenAI` defaults for `openai`, `anyscale`, and `fireworks`.
- Initialized `Claude` defaults and a conditional standard Claude base URL for `claude` and `anthropic`.
- Initialized `Gemini` defaults for `gemini`.
- Added explicit no-op switch cases for Responses and unsupported provider types; the engine factory remains responsible for rejecting unsupported Ollama chat.
- Updated both `agent().inference(...).build()` and `engine().inference(...).build()` to propagate normalization errors.
- Added table-driven default, preservation, factory, unsupported-provider, direct request-construction, and JavaScript profile regression tests.

### Why

- The shared hook is the single point where settings are cloned for mutable engine construction and where the final selected API type is available.
- Provider constructors return errors, so a void helper could hide initialization failures.
- Explicit no-op cases satisfy exhaustive provider handling while preserving the existing support boundary.

### What worked

- Focused tests passed:

  ```text
  GOWORK=off go test ./pkg/js/modules/geppetto ./pkg/inference/engine/factory ./pkg/steps/ai/openai ./pkg/steps/ai/claude ./pkg/steps/ai/gemini -count=1
  ```

- The pre-commit hook passed both `go test ./...` and the full lint suite.
- The OpenAI sparse-profile regression reaches `MakeCompletionRequestFromTurn` and asserts the stacked profile's selected `gpt-5-mini` model, without an HTTP request.
- Explicit Claude client, provider settings, and base URL values remain unchanged.

### What didn't work

- The first commit attempt passed `go test ./...` but lint could not start because another golangci-lint process held the lock:

  ```text
  Error: parallel golangci-lint is running
  ```

  I waited for the process to finish and retried without killing it.

- The next hook run found an exhaustive-switch violation after the initial implementation:

  ```text
  missing cases in switch of type types.ApiType: types.ApiTypeOpenResponses, types.ApiTypeOpenAIResponses, types.ApiTypeOllama, types.ApiTypeMistral, types.ApiTypePerplexity, types.ApiTypeCohere
  ```

  I added explicit no-op cases, staged that correction, reran focused tests and `make lintmax`, and then the normal commit hook passed.

### What I learned

- Existing sparse-profile JavaScript coverage built an agent/session but did not construct a provider request, so it could not observe the missing `Client` and `OpenAI` pointers.
- A provider-aware normalizer should initialize runtime-owned containers but never infer credentials, model identity, base URLs other than established provider defaults, or security policy opt-ins.
- `chat.api_type: ollama` is not supported by the standard engine factory. Its settings omission must not be confused with a default-normalization defect.

### What was tricky to build

- The JavaScript wrapper intentionally hides Go-owned references. The regression test uses the module's hidden reference inside the same Go package only, allowing inspection of the constructed engine without broadening the JavaScript public API.
- Classic OpenAI, Responses, Claude, and Gemini do not have identical requirements. Initializing every provider block would be simpler but would hide ownership and make snapshots noisy.
- The project lint configuration enforces exhaustive enum switches, which made the unsupported-provider behavior explicit rather than accidental.

### What warrants a second pair of eyes

- Confirm whether `anyscale` and `fireworks` should continue to share classic OpenAI settings defaults; this follows their current engine-factory route.
- Confirm the desired default policy for `Client` on API types that are currently unsupported. It is created as generic runtime infrastructure, but support is still rejected by the factory.
- Confirm whether profile resolution itself should eventually materialize defaults for non-JavaScript Go callers; this ticket deliberately fixes the existing JavaScript builder boundary only.

### What should be done in the future

- Add matching normalization to another Go-facing public engine-construction boundary only if a concrete sparse-profile failure demonstrates the need.
- Consider adding a small public test seam for request construction if future JS bridge tests need this behavior without package-private reference access.

### Code review instructions

- Start with `pkg/js/modules/geppetto/api_engines.go` and inspect the provider switch and its scope limits.
- Review `api_agent.go` and `api_engine_builder.go` for error propagation.
- Review `api_agent_profile_test.go`, especially the sparse-profile request-construction test and unsupported Ollama assertion.
- Validate with `GOWORK=off go test ./pkg/js/modules/geppetto -count=1`, `make lintmax`, and `go test ./...`.

### Technical details

The invariant after normalization is:

```text
API and Client are non-nil for a selected chat API type.
OpenAI, Claude, or Gemini is non-nil only when the selected provider requires it.
Explicit profile objects and values take precedence.
Unsupported API types still fail in the engine factory.
```

## Related

- [Provider-aware default-normalization guide](../design-doc/01-provider-aware-inference-profile-default-normalization-guide.md)
- [Ticket index](../index.md)
