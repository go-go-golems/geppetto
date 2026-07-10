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
    - Path: repo://pkg/js/modules/geppetto/api_engines.go
      Note: Normalization defect and planned provider-aware repair
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

## Related

- [Provider-aware default-normalization guide](../design-doc/01-provider-aware-inference-profile-default-normalization-guide.md)
- [Ticket index](../index.md)
