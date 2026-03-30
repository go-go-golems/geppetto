---
Title: Investigation diary
Ticket: GP-59-YAML-SANITIZATION
Status: active
Topics:
    - geppetto
    - events
    - streaming
    - yaml
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/events/structuredsink/filtering_sink.go
      Note: investigated ownership and sink behavior
    - Path: geppetto/pkg/events/structuredsink/parsehelpers/helpers.go
      Note: investigated missing sanitization in helper path
    - Path: geppetto/pkg/steps/ai/openai/engine_openai.go
      Note: investigated provider partial/final publication
    - Path: geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: investigated responses publication path
    - Path: glazed/pkg/helpers/yaml/yaml.go
      Note: identified reusable sanitizer
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: confirmed downstream translation only
ExternalSources: []
Summary: Investigation record for locating the correct ownership layer for YAML sanitization in streaming structured event extraction and for writing the implementation guide stored in this ticket.
LastUpdated: 2026-03-28T18:04:46.297857105-04:00
WhatFor: Preserve the reasoning, commands, evidence, and continuation notes behind the GP-59 design ticket.
WhenToUse: Use when continuing the implementation, reviewing why the change belongs in Geppetto, or checking which files and commands were used to build the design guide.
---


# Investigation diary

## Goal

Capture how I determined where YAML sanitization belongs in the structured-streaming stack, what evidence I gathered, and what implementation approach the ticket now recommends.

## Step 1: Trace ownership and write the design ticket

I started from the user's uncertainty about whether the change lived in Geppetto or Pinocchio. The main work in this step was to trace the real runtime path instead of trusting memory or doc wording. Once that path was clear, I created a new docmgr ticket and wrote an intern-oriented design document that explains both the architecture and the recommended implementation shape.

The decisive conclusion was that Pinocchio is downstream-only for this feature. Provider engines emit Geppetto text events, Geppetto's `FilteringSink` routes structured payload bytes, and Geppetto parsehelpers do the YAML parsing work. Pinocchio only translates already-emitted events into SEM frames for the UI. That makes Geppetto the correct home for default-on YAML sanitization.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to add YAML sanitization (optional, but on by default) to streaming structured event YAML extractions. (I think it's in geppetto, but it might be in pinocchio). 

That way we can sanitize YAML streaming from the LLM provider. 

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new Geppetto docmgr ticket, confirm the correct ownership layer for YAML sanitization in structured streaming, write a very detailed intern-facing design and implementation guide, store it in the ticket, and upload the ticket bundle to reMarkable.

**Inferred user intent:** Avoid placing the feature in the wrong repo, and leave behind enough architectural context that an intern can implement the change safely without rediscovering the stack.

**Commit (code):** N/A

### What I did

- Ran `docmgr status --summary-only` to confirm the active docmgr root.
- Searched `geppetto`, `pinocchio`, and `glazed` for structured streaming, filtering sink, YAML parsing, and sanitization references.
- Read the ticket/doc workflow skill, docmgr skill, diary skill, and ticket writing style reference.
- Inspected the following code paths in detail:
  - `geppetto/pkg/events/context.go`
  - `geppetto/pkg/events/chat-events.go`
  - `geppetto/pkg/inference/toolloop/enginebuilder/builder.go`
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/steps/ai/openai_responses/engine.go`
  - `geppetto/pkg/events/structuredsink/filtering_sink.go`
  - `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`
  - `pinocchio/pkg/webchat/sem_translator.go`
  - `glazed/pkg/helpers/yaml/yaml.go`
- Created ticket `GP-59-YAML-SANITIZATION`.
- Added the primary design doc and diary doc to the ticket.
- Replaced placeholder ticket docs with the actual design narrative and investigation notes.

### Why

- The user's repo-location uncertainty was the first problem to solve.
- YAML sanitization placement depends on runtime ownership, not on naming or intuition.
- A new intern needs both the "where" and the "why", plus warning about stale examples that do not match the current helper API.

### What worked

- Searching the actual codebase immediately showed that Geppetto owns `FilteringSink` and `parsehelpers`, while Pinocchio only translates `EventPartialCompletion` and `EventFinal`.
- The existing `glazed` YAML cleanup helper provides a realistic reuse path instead of inventing a second sanitizer.
- The docmgr workflow fit the task cleanly: create ticket, add docs, then write evidence-backed content directly into the ticket.

### What didn't work

- My first attempt to open helper/interface files in `geppetto/pkg/events/structuredsink` used incorrect filenames:
  - `sed -n '1,260p' geppetto/pkg/events/structuredsink/interfaces.go`
  - `sed -n '1,260p' geppetto/pkg/events/structuredsink/parsehelpers.go`
- Result:
  - `sed: can't read geppetto/pkg/events/structuredsink/interfaces.go: No such file or directory`
  - `sed: can't read geppetto/pkg/events/structuredsink/parsehelpers.go: No such file or directory`
- Resolution:
  - listed the package with `rg --files geppetto/pkg/events/structuredsink`
  - opened the real helper path `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`

### What I learned

- The important architectural boundary is not "Geppetto vs Pinocchio" in the abstract; it is "before or after extractor parsing." YAML sanitization must happen before typed extraction results are emitted, which means Geppetto.
- `FilteringSink` is intentionally generic and only tag-aware. That is a deliberate design worth preserving.
- The docs currently show stale helper APIs (`Feed(...)`, `DebouncedYAML`) compared with the actual code (`FeedBytes(...)`, `YAMLController`).
- `FilteringSink.Options.MaxCaptureBytes` exists but is not implemented yet; this is adjacent context that could confuse reviewers if not called out explicitly.

### What was tricky to build

- The trickiest part was not writing the docs. It was disentangling three similar but distinct layers:
  - provider streaming,
  - structured sink extraction,
  - Pinocchio SEM translation.
- These layers all deal with "streaming text," so it is easy to assume the UI layer might own the fix. The evidence pass made it clear that Pinocchio is already downstream of the extraction decision.

### What warrants a second pair of eyes

- The proposed `DisableSanitize bool` API is practical because it gives zero-value default-on behavior, but someone should still sanity-check whether the team prefers a policy enum or an exported options constructor.
- Reusing `glazed/pkg/helpers/yaml.Clean` is the most economical option, but a maintainer should confirm that cross-repo helper reuse is preferred over copying logic into Geppetto.
- The doc update plan should be reviewed carefully because the current docs are slightly stale and could create follow-up churn if only partially fixed.

### What should be done in the future

- Implement the parsehelpers change and tests.
- Update the structured-sink docs and tutorials to use the helper consistently.
- Consider a future follow-up for `MaxCaptureBytes` enforcement, but keep it out of this ticket unless explicitly expanded.

### Code review instructions

- Start with the design doc in `design-doc/01-intern-guide-to-adding-optional-by-default-yaml-sanitization-to-streaming-structured-event-extractions.md`.
- Then review these code files in order:
  - `geppetto/pkg/steps/ai/openai/engine_openai.go`
  - `geppetto/pkg/events/context.go`
  - `geppetto/pkg/events/structuredsink/filtering_sink.go`
  - `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`
  - `pinocchio/pkg/webchat/sem_translator.go`
  - `glazed/pkg/helpers/yaml/yaml.go`
- Validation commands for the future implementation:
  - `go test ./pkg/events/structuredsink/... -count=1`
  - `go test ./pkg/events/... -count=1`
  - `docmgr doctor --ticket GP-59-YAML-SANITIZATION --stale-after 30`

### Technical details

Commands used during investigation:

```bash
docmgr status --summary-only
rg -n "structured event|structured-event|yaml extraction|streaming yaml|sanitize yaml|sanitiz" -S glazed geppetto pinocchio
rg --files geppetto/pkg/events/structuredsink
nl -ba geppetto/pkg/events/structuredsink/filtering_sink.go | sed -n '1,260p'
nl -ba geppetto/pkg/events/structuredsink/filtering_sink.go | sed -n '300,520p'
nl -ba geppetto/pkg/events/structuredsink/parsehelpers/helpers.go | sed -n '1,220p'
nl -ba geppetto/pkg/steps/ai/openai/engine_openai.go | sed -n '300,455p'
nl -ba geppetto/pkg/steps/ai/openai_responses/engine.go | sed -n '270,330p;860,910p;970,1000p'
nl -ba pinocchio/pkg/webchat/sem_translator.go | sed -n '260,320p'
nl -ba glazed/pkg/helpers/yaml/yaml.go | sed -n '1,260p'
docmgr ticket create-ticket --ticket GP-59-YAML-SANITIZATION --title "Add YAML sanitization to streaming structured event extractions" --topics geppetto,events,streaming,yaml
docmgr doc add --ticket GP-59-YAML-SANITIZATION --doc-type design-doc --title "Intern guide to adding optional-by-default YAML sanitization to streaming structured event extractions"
docmgr doc add --ticket GP-59-YAML-SANITIZATION --doc-type reference --title "Investigation diary"
```
