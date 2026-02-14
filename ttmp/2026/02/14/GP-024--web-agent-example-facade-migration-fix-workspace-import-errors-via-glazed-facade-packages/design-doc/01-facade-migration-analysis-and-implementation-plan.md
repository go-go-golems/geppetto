---
Title: Facade Migration Analysis and Implementation Plan
Ticket: GP-024
Status: active
Topics:
  - web-agent-example
  - glazed
  - facade
  - migration
  - build
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/web-agent-example/cmd/web-agent-example/main.go
    Note: Uses removed legacy layers/parameters APIs
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/pinocchio/cmd/web-chat/main.go
    Note: Reference implementation already migrated to fields/values APIs
  - Path: /home/manuel/workspaces/2026-02-13/mv-debug-ui-geppetto/geppetto/ttmp/2026/02/14/GP-024--web-agent-example-facade-migration-fix-workspace-import-errors-via-glazed-facade-packages/sources/glaze-help-migrating-to-facade-packages.txt
    Note: Full migration guidance captured from `glaze help`
ExternalSources: []
Summary: Detailed migration plan to fix web-agent-example compile errors by applying the full Glazed facade package migration playbook.
---

# Executive Summary

`web-agent-example` currently fails to compile in this workspace because it imports removed Glazed legacy packages (`cmds/layers`, `cmds/parameters`) and a removed Geppetto package path (`pkg/layers`).

Using the full guidance from `glaze help migrating-to-facade-packages`, this ticket migrates `web-agent-example` to the current `schema/fields/values/sources` model and Geppetto `sections` package, then adds focused tests and validation to prevent regression.

# Problem Statement

Current compile failures:

- `github.com/go-go-golems/geppetto/pkg/layers` not found
- `github.com/go-go-golems/glazed/pkg/cmds/layers` not found
- `github.com/go-go-golems/glazed/pkg/cmds/parameters` not found

These are not dependency download issues; they are API removals after the breaking migration described in the `glaze help` playbook.

# Source-of-Truth Migration Guidance (Applied)

From `glaze help migrating-to-facade-packages` (captured in `sources/glaze-help-migrating-to-facade-packages.txt`):

- `cmds/layers` -> `cmds/schema` and parsed values -> `cmds/values`
- `cmds/parameters` -> `cmds/fields`
- command API should use `cmds.NewCommandDescription` + `cmds.WithSections(...)` / `cmds.WithFlags(...)`
- parsed values should use `values.Values.DecodeSectionInto(...)`

# Proposed Solution

## 1) Import and type migration in `main.go`

- Replace imports:
  - `geppetto/pkg/layers` -> `geppetto/pkg/sections`
  - `glazed/pkg/cmds/layers` -> `glazed/pkg/cmds/values`
  - `glazed/pkg/cmds/parameters` -> `glazed/pkg/cmds/fields`
- Replace function and constructor usage:
  - `CreateGeppettoLayers()` -> `CreateGeppettoSections()`
  - `cmds.WithLayersList(...)` -> `cmds.WithSections(...)`
  - `parameters.NewParameterDefinition(...)` -> `fields.New(...)`
  - `RunIntoWriter(... parsed *layers.ParsedLayers ...)` -> `RunIntoWriter(... parsed *values.Values ...)`
  - `parsed.InitializeStruct(layers.DefaultSlug, s)` -> `parsed.DecodeSectionInto(values.DefaultSlug, s)`

## 2) Keep runtime behavior unchanged

No product behavior changes are intended. Only API-surface migration:

- same flags
- same root mounting behavior
- same webchat router wiring
- same middlewares and sink wrapper wiring

## 3) Add targeted tests for resolver behavior in web-agent-example

Add focused tests for `noCookieRequestResolver` to cover:

- WS requires `conv_id` and returns runtime key `default`
- chat request parses prompt/text alias, generates conv_id when missing
- unsupported methods return typed `RequestResolutionError`

# Design Decisions

1. **No compatibility wrappers**
- per user direction and migration guide intent (breaking change accepted).

2. **Single-file main migration**
- Scope kept tight to `cmd/web-agent-example/main.go` because compile errors originate there.

3. **Targeted resolver tests**
- Add tests in app repo to lock runtime selection semantics independent of core.

# Alternatives Considered

## A) Pin old module versions
Rejected: conflicts with workspace-goal and ongoing migration direction.

## B) Add local shim package for legacy imports
Rejected: adds technical debt and contradicts no-backcompat requirement.

## C) Delay tests until after wider app migration
Rejected: this ticket is specifically meant to restore correctness/compile confidence.

# Implementation Plan

## Phase 1: Ticket prep and task decomposition

1. Capture `glaze help migrating-to-facade-packages` output in ticket sources.
2. Convert guidance into file-specific migration checklist.

## Phase 2: Code migration

1. Migrate imports/types in `cmd/web-agent-example/main.go`.
2. Update section/flag definitions and parsed values decoding.
3. Run `gofmt`.

## Phase 3: Validation

1. Run `go test ./cmd/web-agent-example`.
2. Run broader `go test ./...` in `web-agent-example` if module state allows.

## Phase 4: Tests

1. Add resolver tests in `cmd/web-agent-example/engine_from_req_test.go`.
2. Run tests again and fix any regressions.

## Phase 5: Documentation and closure

1. Update ticket `tasks.md` checkboxes.
2. Update diary with step-by-step progress and exact command outputs.
3. Update changelog with commits and related files.

# Acceptance Criteria

- `web-agent-example/cmd/web-agent-example/main.go` no longer imports removed legacy APIs.
- `go test ./cmd/web-agent-example` passes.
- Resolver behavior tests exist and pass.
- GP-024 tasks/diary/changelog are updated for each implementation slice.
