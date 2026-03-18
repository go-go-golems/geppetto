---
Title: Investigation diary
Ticket: GP-44-REMOVE-RUNTIMEKEYFALLBACK
Status: active
Topics:
    - geppetto
    - profile-registry
    - architecture
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: |-
        Verified runtimeKey does not affect engine metadata or construction.
        Diary evidence that runtimeKey does not affect engine creation
    - Path: geppetto/pkg/js/modules/geppetto/api_profiles.go
      Note: |-
        Verified JS aliasing and output encoding.
        Diary evidence for JS runtimeKeyFallback aliasing
    - Path: geppetto/pkg/profiles/registry.go
      Note: |-
        Verified public RuntimeKeyFallback API shape.
        Diary evidence for public RuntimeKeyFallback API
    - Path: geppetto/pkg/profiles/service.go
      Note: |-
        Verified output-only RuntimeKey logic.
        Diary evidence for output-only runtime key behavior
ExternalSources: []
Summary: Chronological record of the RuntimeKeyFallback investigation and the resulting ticket design package.
LastUpdated: 2026-03-17T16:50:00-04:00
WhatFor: Use this diary to review how the RuntimeKeyFallback removal recommendation was established.
WhenToUse: Use when continuing the ticket or reviewing the evidence collection process.
---


# Diary

## Goal

Capture how the repository investigation established that `RuntimeKeyFallback` is an output-only compatibility field and how the removal ticket was prepared.

## Step 1: Trace RuntimeKeyFallback through the resolver and JS APIs

I started by tracing every in-repo use of `RuntimeKeyFallback` and then widened the search to related profile-resolution and JS entrypoints. The goal was to determine whether the field affects real runtime behavior or only reflected metadata. Once the call graph was clear, I validated the conclusion against tests so the cleanup recommendation would be evidence-based rather than stylistic.

The result was consistent across the resolver, the chained registry wrapper, and the JS surfaces: `RuntimeKeyFallback` is not a selector, not a merge input, not a fingerprint input, and not an engine-construction input. It is only a label that survives as `ResolvedProfile.RuntimeKey`.

### Prompt Context

**User prompt (verbatim):** "ok, make a docmgr ticket to remove RuntimeKeyFallback, with an implementation plan and a detailed list of tasks, and then a second one for removing legacy / unused functionality, and do a detailed write up of the other things you found. 

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a full ticket workspace and long-form design package for removing `RuntimeKeyFallback`, including analysis, implementation guidance, tasks, and validation notes.

**Inferred user intent:** Turn an initial cleanup recommendation into a durable, intern-friendly engineering artifact that can be executed later with minimal rediscovery work.

**Commit (code):** N/A

### What I did

- Ran repository searches for `RuntimeKeyFallback`, `runtimeKeyFallback`, and `runtimeKey`.
- Read the resolver API in `pkg/profiles/registry.go`.
- Read the resolver implementation in `pkg/profiles/service.go`.
- Read the chained-registry forwarding logic in `pkg/profiles/source_chain.go`.
- Read the JS resolver and engine APIs in `pkg/js/modules/geppetto/api_profiles.go` and `pkg/js/modules/geppetto/api_engines.go`.
- Read tests and examples that still assert runtime-key behavior.
- Ran:

```bash
go test ./pkg/profiles
go test ./pkg/js/modules/geppetto
```

### Why

- I needed to determine whether `RuntimeKeyFallback` was a real control surface or only a naming shim.
- I wanted the ticket to separate architectural impact from documentation and compatibility fallout.

### What worked

- `rg` searches gave a compact, high-signal call-site list.
- The resolver implementation made the field’s role unambiguous.
- The JS engine API confirmed that runtime key never participates in engine metadata or engine construction.
- Both focused test suites passed, which increased confidence in the current behavior model before proposing removal.

### What didn't work

- My first broad grep for “fallback” produced too much unrelated output across historical ticket docs and old investigations.
- `docmgr ticket list --prefix GP --limit 15` failed because that flag does not exist.

Exact error:

```text
Error: unknown flag: --prefix
```

### What I learned

- The true profile-runtime identity in this subsystem is `registry slug + profile slug + merged runtime + fingerprint`.
- `RuntimeKeyFallback` survives mostly because JS tests, examples, and docs still mention it.
- The cleanest removal is likely a hard cut that removes both input and output fields, not just the fallback input.

### What was tricky to build

- The main tricky part was avoiding overstatement. There is a difference between “not needed inside this repo” and “no one anywhere could possibly use it.” I handled that by documenting the real residual risk as downstream out-of-repo consumers that may display `ResolvedProfile.RuntimeKey` even though no in-repo runtime flow depends on it.

### What warrants a second pair of eyes

- A repo-wide or monorepo-wide search outside `geppetto/` for consumers of `ResolvedProfile.RuntimeKey`.
- Any downstream UI code that may expose runtime key separately from profile slug.

### What should be done in the future

- Implement the hard-cut cleanup in one change that updates Go types, JS bindings, tests, examples, and docs together.

### Code review instructions

- Start in `pkg/profiles/registry.go` and `pkg/profiles/service.go`.
- Confirm that runtime-key logic appears only after real resolution work is already finished.
- Validate with:

```bash
go test ./pkg/profiles ./pkg/js/modules/geppetto
rg -n "RuntimeKeyFallback|runtimeKeyFallback|runtimeKey" geppetto/pkg geppetto/examples/js geppetto/pkg/doc
```

### Technical details

- Key commands run:

```bash
rg -n "RuntimeKeyFallback" geppetto
rg -n "ResolveInput\{|ResolveEffectiveProfile\(|RuntimeKeyFallback|runtimeKeyFallback|runtimeKey" geppetto/pkg geppetto/cmd geppetto/examples
nl -ba geppetto/pkg/profiles/service.go | sed -n '121,183p'
nl -ba geppetto/pkg/js/modules/geppetto/api_engines.go | sed -n '235,285p'
```
