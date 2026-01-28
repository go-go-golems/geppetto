---
Title: Diary
Ticket: PI-004-WEBCHAT-ENGINE-BUILD
Status: active
Topics:
    - analysis
    - webchat
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/ttmp/2026/01/27/PI-004-WEBCHAT-ENGINE-BUILD--simplify-webchat-getorcreateconv-engine-building/analysis/01-webchat-getorcreateconv-vs-enginefromreqbuilder-analysis.md
      Note: Primary analysis output
ExternalSources: []
Summary: Implementation diary for webchat getOrCreateConv/engine builder simplification analysis
LastUpdated: 2026-01-27T22:50:19-05:00
WhatFor: Track analysis steps and findings for simplifying webchat engine building responsibilities.
WhenToUse: Update after each significant research or analysis step.
---


# Diary

## Goal

Track the investigation into how `getOrCreateConv`, `EngineFromReqBuilder`, and engine construction relate, to inform a refactor that isolates conversation retrieval from engine/profile selection.

## Step 1: Create PI-004 ticket and initialize analysis docs

I created the PI-004 ticket and added the analysis and diary documents so the upcoming research is tracked in a structured way. This establishes the scaffolding for a deeper technical write-up.

This step is purely organizational and does not change code.

### Prompt Context

**User prompt (verbatim):** "close it. then create a new ticket where we want to simplify getOrCreateConv / engine building in the webchat.\n\nAnalyze if getOrCreateConv's engine creation shouldn't be replace by EngineFromReqBuilder, or how the two relate. It seems to me that getOrCreateConv should only handle retrieving an existing converstaion, and not do anything with profiles and middlewares and engines, which is supposed to all come from BuildEngineFromReq.\n\nStore the deep detailed technical analysis document in the new docmgr ticket.\n\nALso keep a frequent detailed diary during your research and analysis, updating frequently as you learn things.\n\nThen upload to remarkable"

**Assistant interpretation:** Close PI-003, open a new ticket, and set up documentation for a deep analysis of webchat engine building responsibilities.

**Inferred user intent:** Establish a structured research path to simplify webchat engine building and produce a detailed analysis document.

**Commit (code):** N/A

### What I did
- Closed ticket PI-003.
- Created ticket PI-004 with analysis + diary documents.

### Why
- Provide a structured workspace for the requested analysis.

### What worked
- Ticket and docs were created successfully.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Review the new ticket structure under `geppetto/ttmp/2026/01/27/PI-004-WEBCHAT-ENGINE-BUILD--simplify-webchat-getorcreateconv-engine-building/`.

### Technical details
- Commands: `docmgr ticket create-ticket`, `docmgr doc add` for analysis and diary.

## Step 2: Inventory webchat engine-building responsibilities

I mapped where request parsing, profile selection, config building, and engine creation happen, focusing on `getOrCreateConv` and `EngineFromReqBuilder`. The goal was to clarify which layer currently owns profile/middleware selection and which layer owns conversation lifecycle.

This step collected the file-level evidence needed for the deep technical analysis.

### Prompt Context

**User prompt (verbatim):** "close it. then create a new ticket where we want to simplify getOrCreateConv / engine building in the webchat.\n\nAnalyze if getOrCreateConv's engine creation shouldn't be replace by EngineFromReqBuilder, or how the two relate. It seems to me that getOrCreateConv should only handle retrieving an existing converstaion, and not do anything with profiles and middlewares and engines, which is supposed to all come from BuildEngineFromReq.\n\nStore the deep detailed technical analysis document in the new docmgr ticket.\n\nALso keep a frequent detailed diary during your research and analysis, updating frequently as you learn things.\n\nThen upload to remarkable"

**Assistant interpretation:** Start the code-level investigation and capture how responsibilities are currently distributed.

**Inferred user intent:** Decide whether `getOrCreateConv` should stop building engines and defer to request builders.

**Commit (code):** N/A

### What I did
- Searched for `getOrCreateConv`, `EngineFromReqBuilder`, and engine configuration wiring in `pinocchio/pkg/webchat`.
- Reviewed `conversation.go`, `engine_from_req.go`, `engine_builder.go`, `router.go`, `types.go`, and `cmd/web-chat/README.md` to understand the flow.

### Why
- The analysis requires clear evidence of current responsibilities and call graph flow.

### What worked
- The codebase already separates request policy (conv/profile/overrides) from engine build configuration, but the conversation creator still rebuilds engines.

### What didn't work
- N/A

### What I learned
- `EngineFromReqBuilder` only resolves policy inputs, while `getOrCreateConv` composes engine/sink/subscriber and rebuilds on profile/config changes.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Confirm that no other entrypoints bypass `BuildEngineFromReq` and call `getOrCreateConv` directly with derived profile/overrides.

### What should be done in the future
- N/A

### Code review instructions
- Start with `pinocchio/pkg/webchat/conversation.go` and `pinocchio/pkg/webchat/engine_from_req.go`.
- Cross-check handler usage in `pinocchio/pkg/webchat/router.go`.

### Technical details
- Commands: `rg -n "getOrCreateConv|EngineFromReqBuilder|BuildEngineFromReq" pinocchio -S`.

## Step 3: Write deep analysis of getOrCreateConv vs EngineFromReqBuilder

I consolidated the findings into a deep, structured analysis that treats webchat as a layered system: request policy, engine configuration, and conversation lifecycle. The document explicitly maps current responsibilities and lays out refactor options with trade-offs and recommended direction.

This step produces the requested “textbook-style” analysis for the ticket and anchors the discussion in concrete file-level evidence.

### Prompt Context

**User prompt (verbatim):** "close it. then create a new ticket where we want to simplify getOrCreateConv / engine building in the webchat.\n\nAnalyze if getOrCreateConv's engine creation shouldn't be replace by EngineFromReqBuilder, or how the two relate. It seems to me that getOrCreateConv should only handle retrieving an existing converstaion, and not do anything with profiles and middlewares and engines, which is supposed to all come from BuildEngineFromReq.\n\nStore the deep detailed technical analysis document in the new docmgr ticket.\n\nALso keep a frequent detailed diary during your research and analysis, updating frequently as you learn things.\n\nThen upload to remarkable"

**Assistant interpretation:** Produce the detailed technical analysis document clarifying responsibilities and options.

**Inferred user intent:** Use a thorough analysis to guide a future refactor that cleans up ownership boundaries.

**Commit (code):** N/A

### What I did
- Wrote the analysis document with system model, call graphs, responsibility matrix, options, and recommendations.
- Added explicit references to the webchat files and their roles.

### Why
- The decision depends on clarifying the current architecture and pinpointing responsibility boundaries.

### What worked
- The analysis provides concrete, source-backed rationale for keeping request policy and engine composition separate.

### What didn't work
- N/A

### What I learned
- The current design already separates request policy from engine composition; the confusion stems from `getOrCreateConv` coupling lifecycle and engine wiring.

### What was tricky to build
- Presenting refactor options without collapsing distinct responsibilities into a single “builder” abstraction.

### What warrants a second pair of eyes
- Validate that Option A (EnginePlan) is minimal enough and doesn’t inadvertently widen the public API surface in undesirable ways.

### What should be done in the future
- N/A

### Code review instructions
- Review `geppetto/ttmp/2026/01/27/PI-004-WEBCHAT-ENGINE-BUILD--simplify-webchat-getorcreateconv-engine-building/analysis/01-webchat-getorcreateconv-vs-enginefromreqbuilder-analysis.md`.

### Technical details
- Sources reviewed: `pinocchio/pkg/webchat/conversation.go`, `engine_from_req.go`, `engine_builder.go`, `engine_config.go`, `router.go`, `types.go`, `cmd/web-chat/README.md`.
