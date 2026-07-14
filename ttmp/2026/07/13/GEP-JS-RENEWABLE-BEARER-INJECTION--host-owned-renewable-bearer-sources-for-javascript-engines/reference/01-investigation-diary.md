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
    - Path: repo://pkg/doc/playbooks/07-wire-provider-credentials-for-js-and-go-runner.md
      Note: Credential wiring guidance (commit 351f5cbb)
    - Path: repo://pkg/doc/playbooks/08-use-renewable-bearer-credentials.md
      Note: JavaScript host registration guidance (commit 351f5cbb)
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder.go
      Note: |-
        Observed engine construction path
        Source-aware factory construction (commit 13621922)
    - Path: repo://pkg/js/modules/geppetto/api_engine_builder_test.go
      Note: Host source, static-key fallback, and JavaScript exposure regression coverage (commit f962653d)
    - Path: repo://pkg/js/modules/geppetto/module.go
      Note: Host-only bearer source option (commit 13621922)
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

## Step 2: Wire the host source into native engine construction

The native module now accepts a host-configured `credentials.BearerTokenSource` and carries that interface only through Go runtime state. The JavaScript builder calls a private constructor that retains the old helper path when no source exists, or constructs the standard factory with `WithBearerTokenSource` when the host configured one.

This is deliberately capability injection rather than configuration mutation. The cloned `InferenceSettings` remains free of OAuth material, and the existing factory continues to decide which providers consume the source and when static API-key validation is bypassed.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the agreed host-only source path as a focused, independently reviewable commit.

**Inferred user intent:** Enable renewable credentials for JavaScript-created engines without enlarging the JavaScript credential surface.

**Commit (code):** `13621922` — "feat: inject host bearer sources into JS engines"

### What I did

- Added `Options.BearerTokenSource credentials.BearerTokenSource` in `pkg/js/modules/geppetto/module.go`.
- Copied the interface to the private `moduleRuntime.bearerTokenSource` during native-module runtime initialization.
- Added `moduleRuntime.newEngineFromSettings` in `pkg/js/modules/geppetto/api_engine_builder.go`.
- Changed the builder's final construction call to use the private source-aware helper.
- Ran `gofmt -w pkg/js/modules/geppetto/module.go pkg/js/modules/geppetto/api_engine_builder.go`.
- Committed the focused code change after the repository pre-commit hook passed full `go test ./...` and lint.

### Why

`factory.NewEngineFromSettings` always creates an unconfigured standard factory. Retaining it for a nil source preserves old behavior; using the existing factory option when configured applies the already-reviewed OpenAI-compatible request-time source path.

### What worked

- `go test ./pkg/js/modules/geppetto -count=1` passed before commit.
- The pre-commit hook passed `go test ./...`, `golangci-lint`, `go vet`, and Glazed lint checks.
- The diff contains no JavaScript exports, JavaScript-callable credential methods, profile fields, or static-key mutations.

### What didn't work

No build or test failure occurred in this step.

### What I learned

The existing no-options helper is useful as an explicit compatibility branch. Only the non-nil source branch needs a direct `NewStandardEngineFactory(...).CreateEngine(...)` call; provider capability and credential precedence remain centralized in the factory.

### What was tricky to build

The new field must be public enough for an embedding Go host but never become part of the JavaScript object graph. `Options` is copied into `moduleRuntime`, while `installExports` only attaches explicitly selected functions and namespaces. The implementation therefore stores the source in a private Go field and only passes it as a factory option during `build()`.

### What warrants a second pair of eyes

- Check the `Options` documentation makes the host-only nature unambiguous.
- Check future metadata/debug features do not reflect private runtime fields.
- Check OpenAI-compatible source-enabled builds are covered by a behavioral regression test, not only compilation.

### What should be done in the future

- Add source/no-source behavioral tests and an explicit JavaScript public-surface negative assertion.
- Update final ticket records and upload the completed bundle after tests pass.

### Code review instructions

- Start with `pkg/js/modules/geppetto/module.go` at `Options` and `newRuntime`.
- Follow `pkg/js/modules/geppetto/api_engine_builder.go` from `build()` to `newEngineFromSettings`.
- Verify provider-specific handling remains in `pkg/inference/engine/factory/factory.go`.
- Validate with `go test ./pkg/js/modules/geppetto -count=1` and the repository pre-commit hook output recorded above.

### Technical details

The new source-aware branch is equivalent to:

```go
factory := enginefactory.NewStandardEngineFactory(
    enginefactory.WithBearerTokenSource(m.bearerTokenSource),
)
return factory.CreateEngine(settings)
```

When `m.bearerTokenSource == nil`, it calls `enginefactory.NewEngineFromSettings(settings)` unchanged.

## Step 3: Lock the security boundary with behavioral regression tests

The test suite now creates the same OpenAI profile with an explicitly empty static-key map in two native-module runtimes. The host-source runtime successfully builds its JavaScript engine; the zero-value runtime fails with the existing static-key validation error. This verifies the new path is real rather than merely a field copied through the runtime.

The source test double intentionally returns an empty value and is never invoked during engine construction. It proves the source interface is enough for factory validation without putting even a synthetic bearer value into JavaScript, test diagnostics, or ticket artifacts.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add focused behavioral coverage for both the host-source path and the existing static-key fallback.

**Inferred user intent:** Prevent future regressions that would either break renewable JavaScript engines or expose a credential capability to scripts.

**Commit (code):** `f962653d` — "test: cover JS host bearer source injection"

### What I did

- Added `pkg/js/modules/geppetto/api_engine_builder_test.go`.
- Added a no-credential `hostOnlyTestBearerSource` that satisfies the Go interface.
- Added `TestEngineBuilderUsesHostBearerSourceWithoutStaticKey`.
- Added `TestEngineBuilderWithoutHostBearerSourceRequiresStaticKey`.
- Ran focused module, factory, and credentials tests, then committed after the full pre-commit test and lint hooks passed.

### Why

A compile-only test would not show whether the standard factory received the source. The factory's validation behavior gives a safe behavioral seam: source presence permits empty static key configuration, while source absence retains the old rejection.

### What worked

- `go test ./pkg/js/modules/geppetto -count=1` passed.
- `go test ./pkg/inference/engine/factory ./pkg/steps/ai/credentials -count=1` passed.
- The full pre-commit `go test ./...`, `golangci-lint`, `go vet`, and Glazed checks passed.
- The JavaScript test asserts the source is absent from module exports, engine properties, and engine metadata.

### What didn't work

No test or build failure occurred in this step.

### What I learned

The engine builder can be tested entirely through the native module and a temporary profile document. This keeps the test representative of JavaScript use while retaining source construction and inspection in Go.

### What was tricky to build

Testing a bearer source normally risks creating a string that could become visible in an assertion or error. The builder does not call `BearerToken`, so the test double returns an empty string and the test instead observes factory validation. This proves the wiring without creating credential-shaped test data.

### What warrants a second pair of eyes

- Consider whether a later HTTP-level integration test should assert request-time source use with a local server. It must keep the authorization value private to Go and should be added only if the existing provider tests do not already cover source propagation.
- Confirm error matching remains stable enough; the test intentionally checks the established missing-key fragment rather than a full error string.

### What should be done in the future

- Update public renewable-credential help with the now-available Go registration API.
- Run final ticket validation and upload the completed bundle.

### Code review instructions

- Review `pkg/js/modules/geppetto/api_engine_builder_test.go` alongside `api_engine_builder.go`.
- Run the two focused commands recorded above, then inspect the pre-commit result for complete repository coverage.

### Technical details

The success case uses `api_keys: {}` and a non-nil `Options.BearerTokenSource`; the failure case uses the same profile with zero-value `Options`. Neither path contains access, refresh, authorization-code, or client-secret data.

## Step 4: Publish operator guidance and complete ticket validation

The public renewable-credential playbook now documents the actual registration API rather than the former JavaScript limitation. It gives hosts a small Go snippet and emphasizes that JavaScript retains its existing fluent builder while never receiving the bearer source or credential values.

The ticket index, tasks, changelog, file relations, and this diary now close the implementation loop. The initial design bundle was uploaded before code work; the final bundle will include the completed diary and guidance after this documentation commit.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Publish the implemented API, validate the full ticket, and deliver the final intern-facing material to reMarkable.

**Inferred user intent:** Leave an intern and reviewer with an accurate guide, a safe public contract, concrete validation evidence, and a readable portable deliverable.

**Commit (code):** `351f5cbb` — "docs: describe JS host bearer source registration"

### What I did

- Updated `pkg/doc/playbooks/08-use-renewable-bearer-credentials.md` with `geppetto.Options.BearerTokenSource` registration and unchanged JavaScript usage.
- Updated `pkg/doc/playbooks/07-wire-provider-credentials-for-js-and-go-runner.md` to distinguish static profile credentials from host-owned renewable OAuth sources.
- Updated the ticket index and prepared final task, changelog, and file-relation bookkeeping.
- Ran `go test ./pkg/doc ./pkg/js/modules/geppetto -count=1`.
- Ran `go test -race ./pkg/js/modules/geppetto -count=1`.
- Ran `git diff --check` before committing the public documentation.

### Why

Documentation that says JavaScript has no injection path becomes unsafe once the host-only API exists because it encourages needless Go-prebuilt-engine workarounds. The updated playbook gives the only supported route while explicitly rejecting JavaScript token and callback APIs.

### What worked

- Package documentation and JavaScript module tests passed.
- The focused JavaScript race test passed.
- The final public API still has no JavaScript-visible bearer-related export.
- The initial bundle dry-run and upload succeeded at `/ai/2026/07/13/GEP-JS-RENEWABLE-BEARER-INJECTION`.

### What didn't work

No validation failure occurred in this step. The final bundle must overwrite the initial design-only upload so the device contains the completed implementation diary; this is safe here because no user annotations were created during the current session.

### What I learned

The best JavaScript documentation is not a JavaScript credential API. It documents the registration-time Go capability and makes clear that JavaScript's familiar `engine().inference(...).build()` call remains unchanged.

### What was tricky to build

The guidance needs to be actionable without accidentally encouraging credential copying. The Go example names only an opaque `credentials.BearerTokenSource`; it omits all token fields and places the source at module registration, not in settings or a JavaScript object.

### What warrants a second pair of eyes

- Verify the eventual Pinocchio embedding passes its profile-resolved source into this `Options` field rather than reconstructing a static key.
- Review any future multi-account API as a host authorization design, not as a script-facing selector.

### What should be done in the future

- Publish the branch and run downstream standalone Pinocchio validation after the Geppetto API is fetchable.
- Consider a local HTTP integration test only if existing OpenAI provider tests cease to cover source-to-request propagation.

### Code review instructions

- Read the public playbook update first, then compare it with `Options.BearerTokenSource` and `newEngineFromSettings`.
- Run `go test ./pkg/js/modules/geppetto -count=1` and `go test -race ./pkg/js/modules/geppetto -count=1`.
- Confirm `docmgr doctor --ticket GEP-JS-RENEWABLE-BEARER-INJECTION --stale-after 30` passes and review the uploaded bundle title and destination.

### Technical details

Validation commands completed successfully:

```bash
go test ./pkg/doc ./pkg/js/modules/geppetto -count=1
go test -race ./pkg/js/modules/geppetto -count=1
```

The complete code commit sequence is `13621922` (wiring), `f962653d` (tests), and `351f5cbb` (public docs). Ticket documentation commits record each phase separately.
