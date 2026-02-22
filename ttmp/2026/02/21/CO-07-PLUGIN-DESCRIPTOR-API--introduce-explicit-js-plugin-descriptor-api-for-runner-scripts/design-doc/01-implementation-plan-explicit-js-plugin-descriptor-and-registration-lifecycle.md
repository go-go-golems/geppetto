---
Title: 'Implementation plan: explicit JS plugin descriptor and registration lifecycle'
Ticket: CO-07-PLUGIN-DESCRIPTOR-API
Status: active
Topics:
    - javascript
    - architecture
    - middleware
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-21T18:34:43.188753406-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation plan: explicit JS plugin descriptor and registration lifecycle

## Executive Summary

`cozo-relationship-js-runner` currently discovers script entrypoints through
global names (`extractRelations`, `extract`, `run`, `main`). This works but is
implicit, brittle, and hard to evolve. This ticket introduces an explicit plugin
descriptor export model with a versioned contract and a host-side lifecycle:

1. load module
2. validate descriptor
3. create plugin instance with host context
4. execute `run(...)`

Legacy global discovery remains temporarily as a compatibility fallback.

## Problem Statement

Current pain points with global-entrypoint convention:

- Hidden contract: script authors must know magic names.
- No versioning: hard to evolve safely over time.
- Weak validation: failure mode is often "callable not found" late in execution.
- Low discoverability: no structured metadata (`id`, `kind`, capabilities).
- Hard to compose plugin types beyond extraction.

Goals:

- Explicit, validated, versioned script API.
- Clear host/plugin lifecycle boundaries.
- Smooth migration from current global convention.
- Better error messages and introspection.

## Proposed Solution

Introduce a descriptor-based plugin API.

### Descriptor format (`cozo.extractor/v1`)

```javascript
const { defineExtractorPlugin } = require("./lib/plugin_api");

module.exports = defineExtractorPlugin({
  apiVersion: "cozo.extractor/v1",
  kind: "extractor",
  id: "relation-extractor/base",
  name: "Relation Extractor (Base)",
  description: "Extract relationship graph entities from transcript text.",
  create(ctx) {
    return {
      run(input, options) {
        // input: transcript/prompt/profile/engine options
        // options: timeout/tags/runtime hints
        return extractRelations(input.transcript, options);
      },
    };
  },
});
```

### Host loading lifecycle

```text
script path
  -> require(module)
  -> exported descriptor?
      yes: validate + create + run
      no: legacy global fallback lookup (temporary)
```

### Minimal validation rules (v1)

- `apiVersion` exact match: `cozo.extractor/v1`
- `kind === "extractor"`
- non-empty `id` and `name`
- `create` function
- `create(...)` return value has callable `run`

### Host context (`ctx`) proposed fields

- `gp` (optional future handle, read-only)
- `env` (safe env snapshot)
- `runtime` metadata:
  - `runId`
  - `profile`
  - `recordEnabled`
  - `scriptRoot`

### Runtime metadata additions

Output metadata should include:

- `plugin_mode`: `descriptor` or `legacy-global`
- `plugin_id` when descriptor mode is used
- `plugin_api_version`

## Design Decisions

1. `module.exports` descriptor over `globalThis` registration.
Reason: standard JS module semantics and explicit contract.

2. `defineExtractorPlugin(...)` helper for script authors.
Reason: centralized validation and consistent ergonomics.

3. Keep legacy fallback during migration window.
Reason: avoid breaking existing scripts abruptly.

4. Versioned `apiVersion` string.
Reason: enables future contract evolution.

5. One plugin instance per runner invocation.
Reason: predictable lifecycle and no cross-run state leakage.

## Alternatives Considered

1. Keep global convention and document better.
Rejected: still implicit and unversioned; weak tooling.

2. Register plugin via `globalThis.registerPlugin(...)`.
Rejected: still global-magic driven and ordering-sensitive.

3. Implement only host-side heuristics, no SDK helper.
Rejected: inconsistent script authoring and weaker validation ergonomics.

4. Remove legacy mode immediately.
Rejected: unnecessary migration risk; phased deprecation is safer.

## Implementation Plan

1. Add script-side SDK:
  - `scripts/lib/plugin_api.js`
  - descriptor/instance validation helpers
2. Add host loader:
  - resolve module export descriptor
  - validate contract
  - instantiate and execute plugin
3. Keep and tag legacy fallback mode:
  - existing global function lookup path
4. Migrate existing scripts:
  - base extractor
  - reflective extractor
5. Add ticket-local experiments:
  - valid plugin load
  - invalid descriptor diagnostics
  - legacy fallback behavior
6. Update docs/diary/changelog and finalize migration timeline.

## Open Questions

1. Should `ctx` include direct `require` helper for dynamic tool plugins?
2. Should host validate `inputSchema`/`outputSchema` in v1 or defer to v2?
3. How long should legacy global mode remain enabled before removal?
4. Should plugin descriptor be discoverable via a dedicated `inspect-plugin` command?

## References

- `cozo-relationship-js-runner/main.go`
- `cozo-relationship-js-runner/scripts/relation_extractor_template.js`
- `cozo-relationship-js-runner/scripts/relation_extractor_reflective.js`
- `geppetto/pkg/doc/topics/13-js-api-reference.md`
- `geppetto/pkg/doc/topics/14-js-api-user-guide.md`
