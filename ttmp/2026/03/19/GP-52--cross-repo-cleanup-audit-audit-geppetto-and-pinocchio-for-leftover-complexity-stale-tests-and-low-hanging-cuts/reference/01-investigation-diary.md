---
Title: Investigation diary
Ticket: GP-52
Status: active
Topics:
    - cleanup
    - architecture
    - geppetto
    - pinocchio
    - investigation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary of the cross-repo cleanup audit, including the evidence-gathering process, high-signal commands, and the reasoning behind the final recommendations."
LastUpdated: 2026-03-19T14:20:00-04:00
WhatFor: "Use this diary to reconstruct how the audit was performed, what commands and repository reads produced the findings, and which observations are strongest versus more tentative."
WhenToUse: "Use when validating the audit, continuing the cleanup work later, or checking whether a recommendation came from direct evidence or from a higher-level architectural judgment."
---

# Investigation diary

## Goal

Produce a cross-repo audit of Geppetto and Pinocchio that answers a practical maintenance question rather than just an aesthetic one: what in these two historically grown repositories looks like leftover residue, duplicated migration complexity, or low-value test burden that could realistically be simplified or removed today?

The output needed to serve two audiences at once. First, it had to be readable by a tired new intern who does not yet have the shared historical context behind these repositories. Second, it had to be concrete enough for a maintainer to turn into cleanup tickets without redoing the investigation. That is why the audit focused on ownership boundaries, duplicated bootstrap logic, deprecated wrappers, and test triage rather than attempting a full architectural rewrite proposal.

## Scope

In scope:

- current repository structure of `geppetto/` and `pinocchio/`
- likely leftover compatibility or migration code
- duplication between app-owned and library-owned bootstrap logic
- very large or suspicious tests, especially around web-chat
- low-hanging cleanup candidates that appear safe or nearly safe

Out of scope:

- major feature redesign
- performance tuning
- correctness audit of every runtime path
- frontend UX review of the web chat application

## Method

The audit used a layered approach instead of diving directly into suspicious files.

1. Confirm the doc workspace and create a shared ticket.
2. Map top-level repository structure.
3. Identify largest tests and largest non-test files.
4. Read architecture spine files for each repository.
5. Use targeted search to confirm whether suspicious symbols were actually used.
6. Group findings by ownership and cleanup risk.
7. Convert findings into an intern-facing narrative and phased cleanup plan.

This method was chosen because “what is junk?” is a dangerous question when asked too early. Large files often look guilty even when they are simply central. Smaller compatibility leftovers are easier to miss unless you first understand the intended architecture.

## Ticket setup

Created shared ticket workspace under the Geppetto `ttmp` root:

- ticket: `GP-52`
- title: `CROSS-REPO-CLEANUP-AUDIT -- audit geppetto and pinocchio for leftover complexity stale tests and low-hanging cuts`

Created:

- `design-doc/01-geppetto-and-pinocchio-cleanup-audit-and-intern-guide.md`
- `reference/01-investigation-diary.md`

Reason for using the Geppetto `ttmp` root: the workspace already had `.ttmp.yaml` configured there, and the request was explicitly cross-repo rather than Pinocchio-only.

## Repo inventory snapshots

High-level repository size check:

- Geppetto: about 1,554 files
- Pinocchio: about 961 files

Largest Geppetto test files observed:

- `pkg/js/modules/geppetto/module_test.go` at about 1,823 lines
- `pkg/steps/ai/openai_responses/engine_test.go` at about 890 lines
- `pkg/events/structuredsink/filtering_sink_test.go` at about 851 lines
- `pkg/engineprofiles/sqlite_store_test.go` at about 579 lines
- `pkg/inference/middlewarecfg/resolver_test.go` at about 526 lines

Largest Pinocchio test files observed:

- `cmd/web-chat/app_owned_chat_integration_test.go` at about 580 lines
- `cmd/web-chat/profile_policy_test.go` at about 521 lines
- `pkg/webchat/router_debug_api_test.go` at about 510 lines
- `cmd/web-chat/runtime_composer_test.go` at about 351 lines
- `pkg/webchat/conversation_service_test.go` at about 332 lines

Largest non-test files that seemed most relevant:

- Geppetto: `pkg/events/chat-events.go`, `pkg/steps/ai/openai_responses/engine.go`, `pkg/inference/middlewarecfg/resolver.go`, `pkg/js/modules/geppetto/api_sessions.go`, `pkg/js/modules/geppetto/api_runner.go`
- Pinocchio: `pkg/cmds/cmd.go`, `pkg/webchat/router_debug_routes.go`, `cmd/web-chat/profile_policy.go`, `pkg/webchat/http/profile_api.go`, `cmd/pinocchio/cmds/js.go`

## Architecture reads

To avoid mistaking core complexity for junk, I first read the architecture spine files that define the intended ownership model.

Geppetto reads:

- `geppetto/README.md`
- `geppetto/pkg/doc/topics/00-docs-index.md`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/engineprofiles/service.go`
- `geppetto/pkg/engineprofiles/source_chain.go`
- `geppetto/pkg/inference/runner/prepare.go`

Pinocchio reads:

- `pinocchio/README.md`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/router_deps.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/inference/runtime/profile_runtime.go`

Main conclusion from that pass: the fundamental architectural split still makes sense. Geppetto owns reusable runtime/profile resolution. Pinocchio owns app composition. The messiness comes from incomplete cleanup at the boundary, not from a completely broken architecture.

## Evidence pass: Geppetto

### Duplicate and app-specific bootstrap logic in `pkg/sections`

The highest-confidence Geppetto finding came from comparing:

- `geppetto/pkg/sections/profile_sections.go`
- `geppetto/pkg/sections/sections.go`

Searches showed `GetProfileSettingsMiddleware(...)` appearing only as a definition inside the repository. Reading the function and the adjacent helper code showed substantial overlap with `GetCobraCommandGeppettoMiddlewares(...)`.

The most important observation was not simple duplication but Pinocchio leakage:

- use of `PINOCCHIO`-prefixed environment resolution
- Pinocchio-flavored default config path conventions
- helper naming that assumes Pinocchio semantics

This produced the strongest architecture-level judgment in the audit: some Pinocchio bootstrap behavior still lives inside Geppetto, which weakens the intended library/app boundary.

### Legacy event shapes in `pkg/events`

Searches around `EventTypeStatus`, `EventText`, `NewTextEvent(...)`, and `ChatEventHandler` suggested these are either dead or close to dead internally. The files also contain long-lived TODO-style comments that sound like remnants of an unfinished event-model cleanup.

This is a strong but slightly more cautious finding than the `pkg/sections` one, because public library surface requires more care. The internal evidence strongly suggests residue, but external usage still needs to be checked before deletion.

### Large JS module tests are maintenance-heavy, not useless

`pkg/js/modules/geppetto/module_test.go` clearly stood out by size. I specifically searched for the test function list before judging it. The file covers many namespaces, which is why the resulting recommendation was “split” rather than “cut.”

This distinction matters because it creates a reusable review heuristic for the rest of the audit: a big test is not automatically junk if it is protecting a broad public contract.

### Engine profile negative tests have some duplication

Searches for the strict rejection of the old legacy profile-map format showed similar cases at codec, file-store, and source-chain levels. That is a typical sign of safety-by-repetition rather than sharply layered test intent.

This became a smaller recommendation: rationalize duplicated negative coverage, but do not weaken the important “hard cut” behavior itself.

## Evidence pass: Pinocchio

### `pkg/cmds/cmd.go` still behaves like a historical accumulation file

I read the mode branching and execution sections rather than just looking at line count. The file mixes:

- setup and seeding
- run-mode selection
- blocking mode
- chat mode
- profile-switching behavior
- UI/event handling
- persistence wiring

That pattern strongly suggests a file that grew by accretion as the product gained features. The recommendation here is extraction and decomposition, not immediate feature removal.

### `pkg/geppettocompat` appears unused in-repo

Targeted repository search for the exported compatibility symbols only returned self-references. That makes this a good low-hanging candidate, pending a downstream import check.

### `simple-chat-agent` debug commands look close to abandoned

The `debugcmds`-gated debug command file is very large, yet the visible registration path is effectively inert and repository search did not show active in-repo usage. This is the kind of subsystem that produces high onboarding cost for very low product value unless someone still intentionally uses it.

### Web-chat bootstrap logic is duplicated

This was the strongest Pinocchio architecture finding. I compared the helper package:

- `pinocchio/pkg/cmds/helpers/profile_runtime.go`

with the bespoke logic in:

- `pinocchio/cmd/web-chat/main.go`

The duplication is not perfect line-for-line duplication, but it is clearly duplicated ownership. Both areas are trying to answer “how does Pinocchio resolve profile/bootstrap settings,” which means one of them should eventually disappear.

### `profile_policy.go` mixes current policy and migration compatibility

This file includes current selection precedence logic plus legacy cookie parsing and explicit rejection of older selector inputs. That does not make it wrong, but it does mean the file’s complexity must be interpreted historically. The important recommendation here is to give the legacy behavior an explicit expiration condition.

### Deprecated web-chat constructors still have active callers

The repository already contains newer dependency-based web-chat construction APIs, but production code and tests still use deprecated wrappers. This is a classic stalled migration and therefore a good cleanup candidate.

### Web-chat debug routes and tests are large enough to deserve boundary review

This area did not look obviously dead, so I did not classify it as junk. The stronger claim is that if the debug surface remains valuable, it probably deserves extraction or explicit isolation from core web-chat routing.

## Test judgments

The investigation repeatedly led to the same conclusion: test cleanup has to distinguish between contract tests, migration tests, and scaffolding tests.

Contract tests:

- often big for legitimate reasons
- should be preserved, though often split for readability
- example: Geppetto JS module contract suite

Migration tests:

- important only while the migration lives
- should be grouped so they can be removed together later
- example: legacy cookie or selector behavior in web-chat

Scaffolding or wrapper tests:

- often low value once ownership is consolidated
- should usually disappear when duplicate helpers or deprecated wrappers disappear
- example: tests of web-chat main wrapper helpers

This classification drove the final test triage recommendations in the design doc.

## Confidence notes

Highest-confidence recommendations:

- Geppetto `pkg/sections` needs cleanup and clearer ownership boundaries.
- Pinocchio duplicates runtime/bootstrap logic across helper and web-chat main paths.
- Deprecated `pkg/webchat` wrappers should eventually be removed after caller migration.
- `pkg/cmds/cmd.go` should be decomposed.

Moderate-confidence recommendations pending public/downstream checks:

- remove dead-looking Geppetto event types/interfaces
- delete `pkg/geppettocompat`
- remove `simple-chat-agent` debug command subsystem

Moderate-confidence recommendations pending product-policy decisions:

- shrink legacy-cookie/profile-policy behavior and tests
- reduce debug-route test surface

## Practical commands used

Representative command categories used during the investigation:

- repository-wide symbol search with `rg`
- file inventory with `find`/line counting
- targeted `sed -n` reads of architecture and suspicious files
- `docmgr` commands to create and manage the ticket workspace

Representative search intent:

```text
rg -n 'GetProfileSettingsMiddleware\(' geppetto -g '!ttmp/**'
rg -n 'EventTypeStatus|NewTextEvent|ChatEventHandler' geppetto pinocchio -g '!ttmp/**'
rg -n 'defaultPinocchioProfileRegistriesIfPresent\(' geppetto pinocchio -g '!**/ttmp/**'
rg -n 'NewRouter\(|NewServer\(' pinocchio -g '!ttmp/**'
rg -n 'geppettocompat|EnsureTurnID|WrapEngineWithMiddlewares' pinocchio -g '!ttmp/**'
```

These were not copied verbatim as an exact shell transcript from every pass, but they capture the actual search style used to confirm whether suspicious code was isolated or broadly referenced.

## Outcome

The investigation produced one main design document:

- `design-doc/01-geppetto-and-pinocchio-cleanup-audit-and-intern-guide.md`

That document contains:

- intern-oriented architecture explanation
- cleanup candidates
- test triage
- phased implementation plan
- review questions and file reading order

The most important conceptual output of the audit is this ownership rule:

- Geppetto should own reusable runtime/profile machinery.
- Pinocchio should own app defaults, app UX, and app compatibility policy.

Most of the recommended cleanup work follows directly from restoring that rule.

## Follow-up suggestions

Best immediate follow-up ticket:

- consolidate Pinocchio runtime/bootstrap helpers and remove duplicate web-chat resolution paths

Best low-risk cleanup ticket:

- investigate orphaned compatibility and debug subsystems (`pkg/geppettocompat`, dead event shapes, `simple-chat-agent` debug commands)

Best test-focused follow-up:

- split current-vs-legacy web-chat policy tests, and split Geppetto JS module contract tests by namespace

## Related

- `../design-doc/01-geppetto-and-pinocchio-cleanup-audit-and-intern-guide.md`
