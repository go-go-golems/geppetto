---
Title: Investigation diary
Ticket: GEP-JS-RENEWABLE-BEARER-INJECTION
Status: active
Topics:
    - javascript
    - oauth
    - inference
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: Observed engine construction path
    - Path: repo://pkg/js/modules/geppetto/module_hardcut_test.go
      Note: Existing native module test harness
ExternalSources: []
Summary: Chronological implementation record for host-only bearer source injection into JavaScript-created Geppetto engines.
LastUpdated: 2026-07-13T20:21:52.303500037-04:00
WhatFor: Continue, review, and validate the implementation without exposing credential material.
WhenToUse: When resuming or reviewing this ticket.
---


# Diary

## Goal

Record the analysis, design, implementation, validation, and delivery of a Go-host-only renewable bearer source for engines built through Geppetto's JavaScript module.

## Step 1: Map the JavaScript construction gap and publish the design

The initial investigation traced JavaScript's fluent engine builder through its native Go implementation and confirmed that it discards the host's bearer-source capability. The task is not to expose OAuth to JavaScript: it is to let the Go host attach an existing `BearerTokenSource` while JavaScript continues to handle only non-secret inference settings and engine wrappers.

A detailed intern-facing design guide now documents the source-to-request path, security invariants, APIs, diagrams, pseudocode, test plan, and phased implementation plan. The ticket was created before code changes so the capability boundary can be reviewed independently.

### Prompt Context

**User prompt (verbatim):** "ok, create a ticket for the JS integration of the bearer source, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


then implement task, by task, keep a detailed diary, and commit at appropriate intervals"

**Assistant interpretation:** Create a ticket and an exhaustive, evidence-backed guide; upload the documentation; then implement the source injection in small committed steps with diary updates.

**Inferred user intent:** Make a security-sensitive JavaScript integration understandable to a new engineer while preserving host ownership of renewable credentials and producing an auditable implementation trail.

### What I did

- Created ticket `GEP-JS-RENEWABLE-BEARER-INJECTION` and its design and diary documents.
- Added four tracked tasks covering design, wiring, regression tests, and delivery.
- Read the JS engine builder, module option/runtime initialization, factory helpers, standard factory source option, and existing Goja module test harness.
- Wrote `design-doc/01-host-owned-renewable-bearer-source-injection-for-javascript-engines.md` with line-anchored evidence and a proposed Go-only API.

### Why

The JavaScript builder currently invokes the no-options factory helper, so it cannot attach the existing `factory.WithBearerTokenSource` capability. A design-first step makes clear that the correct fix is host injection rather than a token or callback exposed to scripts.

### What worked

- `pkg/js/modules/geppetto/api_engine_builder.go:42-49` shows the direct no-options call.
- `pkg/inference/engine/factory/helpers.go:9-14` confirms that helper constructs a factory with no options.
- `pkg/inference/engine/factory/factory.go:82-88,136-148,221-225` already supplies the needed option, provider propagation, and static-key bypass.
- `pkg/js/modules/geppetto/module_hardcut_test.go` provides an established native-module test harness.

### What didn't work

No implementation or validation failure occurred in this step. The initial test-file search using `rg -l 'engine\\(\\).*inference|NewLoader\\(Options|DefaultInferenceSettings|engineBuilder' pkg/js/modules/geppetto --glob '*_test.go'` returned no matches because the engine-builder path did not yet have focused coverage; broader inspection found the shared harness.

### What I learned

`Options` is already the registration-time dependency-injection boundary for Go-owned registries, middleware, stores, and event infrastructure. A `credentials.BearerTokenSource` belongs there, while `InferenceSettings.API.APIKeys` and JavaScript objects do not.

### What was tricky to build

The factory already makes a source authoritative over static keys, but its convenient `NewEngineFromSettings` helper cannot receive options. The safe approach is to preserve that helper for the nil-source path and create a standard factory with `WithBearerTokenSource` only inside the native module when a host configured one. This keeps provider-specific behavior in the factory and avoids adding credential state to cloned settings.

### What warrants a second pair of eyes

- Confirm that a single registration-level source is the correct first multi-tenant boundary and that source selection must remain host-authorized in a future design.
- Confirm tests do not accidentally include the test bearer in JavaScript errors, metadata, or diagnostics.
- Confirm a source-enabled JavaScript engine can be integrated by Pinocchio without exposing its profile-owned refresh state.

### What should be done in the future

- Implement the documented Go-only source plumbing.
- Add focused source/no-source and no-JavaScript-exposure tests.
- Upload the completed ticket bundle to reMarkable after validation.

### Code review instructions

- Start with the design document, then compare its current-flow diagram to `pkg/js/modules/geppetto/api_engine_builder.go:26-56` and `pkg/inference/engine/factory/helpers.go:9-14`.
- Review the factory option and validation behavior in `pkg/inference/engine/factory/factory.go:82-148,221-225`.
- Validate the final design with `docmgr doctor --ticket GEP-JS-RENEWABLE-BEARER-INJECTION --stale-after 30` and the focused Go test commands listed in the design document.

### Technical details

The proposed host API is:

```go
geppetto.Register(registry, geppetto.Options{
    BearerTokenSource: source, // Go interface; never exposed to JavaScript
})
```

The JavaScript API remains unchanged:

```javascript
require("geppetto").engine().inference(settings).build()
```
