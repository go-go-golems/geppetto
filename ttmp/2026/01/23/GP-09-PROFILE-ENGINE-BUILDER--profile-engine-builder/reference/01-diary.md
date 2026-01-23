---
Title: Diary
Ticket: GP-09-PROFILE-ENGINE-BUILDER
Status: active
Topics:
    - architecture
    - backend
    - go
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md
      Note: Primary analysis output produced by this diary
    - Path: go-go-mento/go/pkg/webchat/engine_builder.go
      Note: Primary code under analysis
    - Path: go-go-mento/go/pkg/webchat/router.go
      Note: Primary coupling site under analysis
    - Path: pinocchio/pkg/webchat/engine_builder.go
      Note: Reference implementation used for comparisons
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-23T08:43:34.531090665-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the investigation and design work to extract webchat engine building out of the Router into a real `EngineBuilder` abstraction (with a `profileEngineBuilder` implementation), with all findings stored in the GP-09 ticket.

## Step 1: Ticket setup + intent check

This step creates the docmgr ticket workspace and establishes a running diary format that makes it easy to verify intent, track findings, and continue the work later. It also confirms the target architecture change: move “build engine from request” logic out of the Router and into a dedicated engine-builder layer.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket GP-09-PROFILE-ENGINE-BUILDER.

Create a detailed analysis of how the go-go-mento/go/pkg/webchat/engine_builder.go go-go-mento/docs/reference/webchat/engine-builder.md can be further extracted out of the Router, so that it becomes a real EngineBuilder, and that the router then calls an interface BuildEngineFromReq which then calls the webchat engine builder which is really a profileEngineBuilder (takes a profileSlug and returns an engine).

Research everywhere and find docs and relevant code and understand the EngineBuilder package (you can make suggestions on how to improve this, if you find issues with it).
```

```text
Store the result in docmgr ticket.

Also keep a frequent diary and add entries frequently as you go along, including at the very top so I can verify your understanding of my intent.
```

**Assistant interpretation:** Create a GP-09 docmgr ticket and write a deep, code-informed design analysis describing how to refactor the current webchat engine builder so the Router depends on a `BuildEngineFromReq` interface; implement the existing webchat-specific builder as a `profileEngineBuilder` that takes a `profileSlug` and returns an engine.

**Inferred user intent:** Make engine construction a first-class, testable, reusable component (not Router glue), so profile-based routing/engine selection is clearer, easier to extend, and less coupled to HTTP/webchat concerns.

**Commit (code):** N/A (docs-only so far)

### What I did
- Created docmgr ticket `GP-09-PROFILE-ENGINE-BUILDER`.
- Created diary doc `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md`.

### Why
- Establish a durable workspace to store the analysis and track progress with clear provenance.
- Make it easy for you to verify my understanding before I go deep into the code.

### What worked
- `docmgr ticket create-ticket` created the expected ticket directory structure under `geppetto/ttmp`.

### What didn't work
- N/A

### What I learned
- This repo’s docmgr root is `geppetto/ttmp`, with existing vocabulary for `architecture`, `backend`, `go`, and `inference`.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- Once the analysis is written, decide whether to implement the refactor immediately or keep this ticket as design-only.

### Code review instructions
- Start at `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/index.md`.
- Review diary at `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/reference/01-diary.md`.

### Technical details
- Commands run:
  - `docmgr status --summary-only`
  - `docmgr vocab list`
  - `docmgr ticket create-ticket --ticket GP-09-PROFILE-ENGINE-BUILDER --title "Profile Engine Builder" --topics architecture,backend,go,inference`
  - `docmgr doc add --ticket GP-09-PROFILE-ENGINE-BUILDER --doc-type reference --title "Diary"`

## Step 2: Map current builder/router coupling (go-go-mento + newer patterns)

This step audits how engine construction currently flows through `Router` and `ConversationManager`, and identifies the exact seams where “engine building” is still mixed with transport/request concerns. I also compared go-go-mento’s implementation to the newer Pinocchio and Geppetto patterns already present in this monorepo so the GP-09 proposal can align with existing direction.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket GP-09-PROFILE-ENGINE-BUILDER.

Create a detailed analysis of how the go-go-mento/go/pkg/webchat/engine_builder.go go-go-mento/docs/reference/webchat/engine-builder.md can be further extracted out of the Router, so that it becomes a real EngineBuilder, and that the router then calls an interface BuildEngineFromReq which then calls the webchat engine builder which is really a profileEngineBuilder (takes a profileSlug and returns an engine).

Research everywhere and find docs and relevant code and understand the EngineBuilder package (you can make suggestions on how to improve this, if you find issues with it).
```

```text
Store the result in docmgr ticket.

Also keep a frequent diary and add entries frequently as you go along, including at the very top so I can verify your understanding of my intent.
```

**Assistant interpretation:** Read through go-go-mento webchat engine builder + router codepaths, then map where Router still performs “engine/policy” work that should live behind a builder interface, using Pinocchio/Moments/Geppetto code and docs as reference points.

**Inferred user intent:** Reduce coupling: Router should mostly orchestrate HTTP/WS and call a request-level “engine build” interface; profile-based policy should live in a dedicated builder that can evolve without touching router code.

**Commit (code):** N/A (still docs-only)

### What I did
- Inspected go-go-mento webchat:
  - `go-go-mento/go/pkg/webchat/engine_builder.go`
  - `go-go-mento/go/pkg/webchat/engine_config.go`
  - `go-go-mento/go/pkg/webchat/conversation_manager.go`
  - `go-go-mento/go/pkg/webchat/router.go`
  - `go-go-mento/go/pkg/webchat/sink_wrapper.go`
  - `go-go-mento/go/pkg/webchat/engine.go`
  - `go-go-mento/go/pkg/webchat/types.go`
- Compared newer patterns in-repo:
  - Pinocchio: `pinocchio/pkg/webchat/engine_builder.go` and related PI-001 doc
  - Moments: `moments/backend/pkg/webchat/conversation.go` rebuild/signature pattern
  - Geppetto: `geppetto/pkg/inference/session/builder.go` and `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`

### Why
- The GP-09 refactor needs to land on the “right” seam: split request parsing from profile-policy from engine/sink composition, without inventing a one-off abstraction that doesn’t match the rest of the monorepo.

### What worked
- go-go-mento already has the core “config signature → rebuild” pattern implemented in `ConversationManager.GetOrCreate`.
- Pinocchio has already moved to an explicit `EngineBuilder` interface (`BuildConfig` / `BuildFromConfig`) and fixed signature secrecy concerns (sanitized signature).

### What didn't work
- N/A (investigation only)

### What I learned
- go-go-mento’s `EngineConfig.Signature()` currently JSON-marshals `StepSettings` directly, which is likely to embed secrets (API keys) in the signature string and logs/metrics.
- go-go-mento’s `EngineBuilder` has `toolFactories` but does not use them; tool registry building still happens in `Router.handleChatRequest`.
- `Profile.AllowOverrides` and `Profile.LoopName` exist but are unused; overrides are applied regardless of `AllowOverrides`, and loop selection is hard-coded.
- Router’s tool registry filtering uses `p.DefaultTools` and ignores `EngineConfig.Tools`, meaning `overrides["tools"]` is currently ineffective.
- `EngineBuilder.SetConversationManager` is a pragmatic circular-dependency break, but it makes engine building depend on the full `ConversationManager` instead of a smaller lookup interface.

### What was tricky to build
- The term “EngineBuilder” is overloaded in this monorepo:
  - webchat-level builder: returns `(engine.Engine, events.EventSink, config)`
  - geppetto `session.EngineBuilder`: returns an `InferenceRunner`
  - toolloop enginebuilder: builds a runner that can do tool-calling loops
  The GP-09 doc needs to be explicit about which layer each builder lives at and avoid conflating them.

### What warrants a second pair of eyes
- Whether changing go-go-mento’s `EngineConfig.Signature()` semantics (to avoid secrets) would have compatibility implications for any persisted signatures or logs that downstream systems rely on.

### What should be done in the future
- Define a request-facing interface (`BuildEngineFromReq`) that sits *above* profile engine building, so Router can stop owning the “where do we get profileSlug/overrides?” logic.

### Code review instructions
- Start at `go-go-mento/go/pkg/webchat/conversation_manager.go` to see the current signature-based rebuild seam.
- Then read `go-go-mento/go/pkg/webchat/engine_builder.go` + `go-go-mento/go/pkg/webchat/engine_config.go` to see what is/ isn’t centralized today.
- Compare with `pinocchio/pkg/webchat/engine_builder.go` to see an already-adopted interface-based shape.

### Technical details
- Key greps used:
  - `rg -n \"type (.*EngineBuilder|EngineBuilder)\" -S go-go-mento geppetto moments pinocchio`
  - `rg -n \"func \\(r \\*Router\\) handleChatRequest\" -n go-go-mento/go/pkg/webchat/router.go`

## Step 3: Gather “everywhere” references (docs + prior tickets)

This step collects the highest-signal internal documentation and adjacent implementations that already discuss engine-builder boundaries, so GP-09 can reuse the same vocabulary and avoid re-litigating settled decisions (especially around config signatures and secret leakage).

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket GP-09-PROFILE-ENGINE-BUILDER.

Create a detailed analysis of how the go-go-mento/go/pkg/webchat/engine_builder.go go-go-mento/docs/reference/webchat/engine-builder.md can be further extracted out of the Router, so that it becomes a real EngineBuilder, and that the router then calls an interface BuildEngineFromReq which then calls the webchat engine builder which is really a profileEngineBuilder (takes a profileSlug and returns an engine).

Research everywhere and find docs and relevant code and understand the EngineBuilder package (you can make suggestions on how to improve this, if you find issues with it).
```

```text
Store the result in docmgr ticket.

Also keep a frequent diary and add entries frequently as you go along, including at the very top so I can verify your understanding of my intent.
```

**Assistant interpretation:** Collect the most relevant existing docs and code in this monorepo that already cover EngineBuilder patterns, config signatures, and router coupling, so the final GP-09 analysis is grounded and consistent.

**Inferred user intent:** Avoid “fresh” design churn by leveraging existing decisions and learning from already-implemented refactors.

**Commit (code):** N/A

### What I did
- Read the primary go-go-mento long-term docs related to EngineBuilder:
  - `go-go-mento/docs/reference/webchat/engine-builder.md`
  - `go-go-mento/docs/architecture/webchat/README.md`
- Read the historically promoted ttmp reference backing the EngineBuilder doc:
  - `go-go-mento/ttmp/MEN-CONVERSATION-MANAGER-3083-webchat-conversation-manager-cleanup/reference/15-engine-builder-reference.md`
- Pulled in the most directly relevant “newer repo” EngineBuilder work:
  - `geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md`

### Why
- These documents already discuss the seam GP-09 is targeting (config signatures, extraction out of handlers) and include warnings (e.g., signature secrecy) that should influence the proposal.

### What worked
- PI-001’s “Status Update (2026-01-23)” captures that Pinocchio has already adopted an interface-based EngineBuilder and also already fixed signature secrecy by avoiding embedding raw StepSettings in the signature.

### What didn't work
- `go-go-mento/docs/architecture/webchat/README.md` still describes an older “buildEng closure” flow in parts; it needs to be read as conceptual rather than a line-by-line match for current go-go-mento code.

### What I learned
- There’s precedent for a two-stage split:
  - “build config / signature” (pure-ish policy)
  - “materialize engine + sink” (wiring)
  and the remaining extraction is mostly about removing request/transport dependencies from the policy layer.

### What was tricky to build
- Reconciling docs written at different refactor stages without assuming all diagrams are current.

### What warrants a second pair of eyes
- Whether the older go-go-mento docs should be updated as part of this workstream, or whether GP-09 is intended to produce a new “current state” doc without retrofitting old docs.

### What should be done in the future
- In the GP-09 analysis doc, explicitly call out “doc is conceptual vs exact current code” when referencing the older architecture diagrams.

### Code review instructions
- Read `go-go-mento/docs/reference/webchat/engine-builder.md` for the intended contract.
- Cross-check current behavior in `go-go-mento/go/pkg/webchat/engine_builder.go` and `go-go-mento/go/pkg/webchat/conversation_manager.go`.
- Use PI-001’s “Status Update (2026-01-23)” as the reference for how this pattern was recently implemented elsewhere in the monorepo.

### Technical details
- N/A

## Step 4: Write the GP-09 extraction analysis (BuildEngineFromReq + profileEngineBuilder)

This step turns the gathered evidence into a concrete, incremental extraction plan: introduce a request-facing `BuildEngineFromReq` interface, clarify the existing webchat builder as a profile engine builder, and identify specific coupling/bugs (tools overrides, AllowOverrides, signature secrecy) that should be addressed as part of making the abstraction “real”.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket GP-09-PROFILE-ENGINE-BUILDER.

Create a detailed analysis of how the go-go-mento/go/pkg/webchat/engine_builder.go go-go-mento/docs/reference/webchat/engine-builder.md can be further extracted out of the Router, so that it becomes a real EngineBuilder, and that the router then calls an interface BuildEngineFromReq which then calls the webchat engine builder which is really a profileEngineBuilder (takes a profileSlug and returns an engine).

Research everywhere and find docs and relevant code and understand the EngineBuilder package (you can make suggestions on how to improve this, if you find issues with it).
```

```text
Store the result in docmgr ticket.

Also keep a frequent diary and add entries frequently as you go along, including at the very top so I can verify your understanding of my intent.
```

**Assistant interpretation:** Produce a design writeup (stored in the GP-09 ticket) that explains the current state, identifies Router coupling, and proposes the new interface boundaries and migration plan.

**Inferred user intent:** Get to a refactorable, testable architecture where Router mostly wires HTTP/WS, and the “which profile/overrides/tools?” policy lives behind dedicated builders with stable signatures.

**Commit (code):** N/A

### What I did
- Wrote the main analysis doc: `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md`.

### Why
- The work needs a single, reviewable statement of “current state → target shape → incremental plan”, grounded in code and aligned with patterns already implemented in Pinocchio/Geppetto.

### What worked
- The existing go-go-mento structure already exposes the key seam (signature-based rebuild in `ConversationManager.GetOrCreate`), so the proposal can be incremental rather than a rewrite.

### What didn't work
- N/A

### What I learned
- The remaining extraction is mostly about removing *request/transport policy* from Router and making the config truly drive runtime behavior (tools, overrides enforcement, signature semantics).

### What was tricky to build
- Describing `BuildEngineFromReq` in a way that improves Router without accidentally re-introducing “fat Router” logic inside a different file (i.e., ensuring the interface boundary is meaningful and testable).

### What warrants a second pair of eyes
- Review the proposed interface names and layering to avoid future confusion with `session.EngineBuilder` and `toolloop/enginebuilder.Builder`.

### What should be done in the future
- Decide whether GP-09 is design-only or should immediately implement Phase 1 (request-facing builder extraction) in code.

### Code review instructions
- Read `geppetto/ttmp/2026/01/23/GP-09-PROFILE-ENGINE-BUILDER--profile-engine-builder/analysis/01-extract-profile-engine-builder-out-of-router.md` first.
- Cross-check the issues called out in:
  - `go-go-mento/go/pkg/webchat/engine_config.go`
  - `go-go-mento/go/pkg/webchat/router.go`

### Technical details
- N/A
