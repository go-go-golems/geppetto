---
Title: 'Implementation plan: geppetto shared plugin contract module'
Ticket: GP-19-JS-PLUGIN-CONTRACT
Status: active
Topics:
    - geppetto
    - javascript
    - middleware
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-21T19:13:25.851671743-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation plan: geppetto shared plugin contract module

## Problem

Plugin-contract helpers (`defineExtractorPlugin`, `wrapExtractorRun`) currently
live in runner-local script files, which causes duplication and drift whenever
multiple plugin runners need the same conformance behavior.

## Goal

Move shared plugin-contract logic into geppetto so scripts can import it as:

```js
const { defineExtractorPlugin, wrapExtractorRun } = require("geppetto/plugins");
```

This makes geppetto the contract source of truth while runner repos keep only
domain-specific extraction logic.

## Scope

- Add `geppetto/plugins` native module export.
- Implement descriptor validation helper.
- Implement run-wrapper canonicalization helper.
- Migrate runner scripts to consume shared helper.
- Remove runner-local duplicate helper file.

## Non-goals

- redesigning relation-extraction schema
- introducing backwards-compatibility for legacy global script mode
- changing current runner host descriptor lifecycle

## API contract

### `defineExtractorPlugin(descriptor)`

Validates and returns frozen descriptor with defaults:

- `apiVersion` default: `"cozo.extractor/v1"`
- `kind` default: `"extractor"`
- requires non-empty string `id`, `name`
- requires function `create`

### `wrapExtractorRun(runImpl)`

Returns wrapper that:

- validates input is object
- validates `input.transcript` is non-empty string
- canonicalizes optional fields:
  - `prompt`: trimmed string or `""`
  - `profile`: trimmed string or `""`
  - `timeoutMs`: positive number or `120000`
  - `engineOptions`: shallow-copied plain object or `null`
- freezes canonical input before invoking `runImpl`

## Integration points

- Geppetto registration:
  - `pkg/js/modules/geppetto/module.go`
- Geppetto plugin helper module:
  - `pkg/js/modules/geppetto/plugins_module.go`
- Geppetto tests:
  - `pkg/js/modules/geppetto/module_test.go`
- Runner scripts consuming shared API:
  - `cozo-relationship-js-runner/scripts/relation_extractor_template.js`
  - `cozo-relationship-js-runner/scripts/relation_extractor_reflective.js`

## Risks

- Runtime differences between goja object semantics and JS expectations.
- Error-message drift if helper behavior changes without tests.

## Mitigations

- Add focused module test using `require("geppetto/plugins")`.
- Keep API names and defaults aligned with existing runner-local behavior.

## Final review checklist

- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/plugins_module.go`
- `geppetto/pkg/js/modules/geppetto/module_test.go`
- `2026-02-18--cozodb-extraction/cozo-relationship-js-runner/scripts/relation_extractor_template.js`
- `2026-02-18--cozodb-extraction/cozo-relationship-js-runner/scripts/relation_extractor_reflective.js`

## Implementation outcome

Delivered:

- `require("geppetto/plugins")` native module in geppetto.
- shared `defineExtractorPlugin` and `wrapExtractorRun` helpers.
- module test coverage for helper availability and validation/defaulting.
- runner-script migration off local helper into shared geppetto helper.

Reference commits:

- geppetto: `3f5320f`
- runner migration: `19ca200`
