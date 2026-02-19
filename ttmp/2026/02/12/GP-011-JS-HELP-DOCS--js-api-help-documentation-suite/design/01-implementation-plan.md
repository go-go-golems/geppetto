---
Title: Implementation Plan
Ticket: GP-011-JS-HELP-DOCS
Status: active
Topics:
    - geppetto
    - goja
    - javascript
    - documentation
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan for JS API reference, user guide, and getting-started glazed help pages
LastUpdated: 2026-02-12T11:30:06.336046217-05:00
WhatFor: Author and validate complete JS API help pages in Glazed format with executable examples.
WhenToUse: Use when implementing or reviewing JS API documentation quality and command example correctness.
---

# Implementation Plan

## Goal

Create a complete JS documentation suite for Geppetto in `pkg/doc` with three coordinated help entries that are centered on writing and running JS scripts directly:

1. API Reference (contract-level, exhaustive)
2. User Guide (workflow patterns)
3. Getting Started Tutorial (step-by-step execution path)

All runnable examples must be validated against current code.

## Scope

In scope:

- Author three new markdown help entries with valid Glazed frontmatter.
- Include API signatures, configuration semantics, and practical composition examples.
- Add links in docs index for discoverability.
- Add a doc-load test to confirm new slugs are discoverable via help system.
- Add a JS execution harness command and runnable example scripts.
- Execute documented script commands and provider smoke script.
- Record verification results in ticket diary.

Out of scope:

- Reworking non-JS legacy documentation.
- Fixing unrelated legacy section type warnings.

## Files to Add/Modify

- `cmd/examples/geppetto-js-lab/main.go`
- `examples/js/geppetto/*.js`
- `pkg/doc/topics/13-js-api-reference.md`
- `pkg/doc/topics/14-js-api-user-guide.md`
- `pkg/doc/tutorials/05-js-api-getting-started.md`
- `pkg/doc/topics/00-docs-index.md`
- `pkg/doc/doc_test.go`
- `geppetto/ttmp/2026/02/12/GP-011-JS-HELP-DOCS--js-api-help-documentation-suite/scripts/test_doc_examples.sh`

## Authoring Standards

- Use valid Glazed frontmatter with unique `Slug`.
- Section types:
  - reference: `GeneralTopic`
  - user guide: `Application`
  - tutorial: `Tutorial`
- Keep examples copy/paste runnable with commands that exist in repo.
- Mark provider-backed commands as smoke checks where behavior is probabilistic.

## Validation Plan

Deterministic checks:

```bash
go test ./pkg/doc -count=1 -v
go run ./cmd/examples/geppetto-js-lab --list-go-tools
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/01_turns_and_blocks.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/02_session_echo.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/03_middleware_composition.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/04_tools_and_toolloop.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/05_go_tools_from_js.js
go run ./cmd/examples/geppetto-js-lab --script examples/js/geppetto/06_live_profile_inference.js
```

Automation wrapper:

```bash
bash geppetto/ttmp/2026/02/12/GP-011-JS-HELP-DOCS--js-api-help-documentation-suite/scripts/test_doc_examples.sh
```

## Risks and Mitigations

- Risk: examples drift from implementation.
  - Mitigation: enforce runnable JS script section and validate via script runner.
- Risk: provider behavior varies run to run.
  - Mitigation: classify provider commands as smoke checks and document nondeterminism.

## Exit Criteria

- Three new help entries authored and indexed.
- New slugs load in `pkg/doc` tests.
- All deterministic example commands pass.
- Provider smoke commands execute successfully with configured profile.
- Diary captures exact commands and outcomes.
