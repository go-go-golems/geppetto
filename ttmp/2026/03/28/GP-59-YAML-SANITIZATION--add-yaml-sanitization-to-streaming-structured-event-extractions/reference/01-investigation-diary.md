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
    - Path: sanitize/pkg/yaml/sanitize.go
      Note: sanitize-backed YAML cleanup used in the implementation
    - Path: pinocchio/pkg/webchat/sem_translator.go
      Note: confirmed downstream translation only
ExternalSources: []
Summary: Investigation record for locating the correct ownership layer for YAML sanitization in streaming structured event extraction and for writing the implementation guide stored in this ticket.
LastUpdated: 2026-03-30T18:30:00-04:00
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

## Step 2: Implement the sanitize-backed parsehelpers path and update the docs

After the earlier design pass, the user confirmed that the implementation should use `sanitize/`, not the old Glazed helper path. In this step I changed Geppetto itself: `parsehelpers` now defaults to sanitizing YAML before unmarshal, callers can explicitly disable that behavior, the new dependency is pinned in `geppetto/go.mod`, and the public structured-sink docs now match the shipped API instead of the stale examples.

The most important technical decision in this step was to keep the public helper shape small. I did not add a new exported `ParseYAMLBytes(...)` helper. Instead, I refactored the existing `YAMLController` so `FeedBytes(...)` and `FinalBytes(...)` share the same sanitize-aware normalization and parse path. That preserves the already-used surface area while still fixing the real bug.

### Prompt Context

**User prompt (verbatim):** "ok, let's work on that ticket then, task by task, committing at appropriate intervals, and then diary. 

Now that we have a properly working agentmode middleware to test things."

**Assistant interpretation:** Implement the remaining GP-59 tasks in Geppetto, commit the work in logical steps, and then update the ticket diary with the actual implementation details.

**Inferred user intent:** Finish the previously documented YAML sanitization work now that there is a real application path (`agentmode`) that benefits from the helper-level fix, and leave behind accurate ticket documentation rather than only a design plan.

**Commit (code):** `01041ff` — "Add YAML sanitization to structured sink parsehelpers"

### What I did

- Updated `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go` to:
  - add `DebounceConfig.SanitizeYAML *bool`
  - add `withDefaults`, `SanitizeEnabled`, and `WithSanitizeYAML`
  - centralize normalization in `normalizedYAML(...)`
  - run `yamlsanitize.Sanitize(...)` before `yaml.Unmarshal`
  - make `FinalBytes(...)` and `FeedBytes(...)` share the same parse path
- Added `geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go` covering:
  - sanitize-on default behavior
  - explicit sanitize-off behavior
  - progressive `FeedBytes(...)` parsing
  - valid fenced YAML pass-through
- Added `github.com/go-go-golems/sanitize v0.0.1` to `geppetto/go.mod` and materialized standalone `go.sum` entries with `GOWORK=off`.
- Updated the public docs:
  - `geppetto/pkg/doc/topics/04-events.md`
  - `geppetto/pkg/doc/topics/11-structured-sinks.md`
  - `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md`
  - `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md`
- Updated GP-59 ticket docs:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
  - `design-doc/01-intern-guide-...md`

### Why

- The code fix belonged in Geppetto, not in `agentmode`, because multiple extractors can use the same helper.
- The original GP-59 docs were stale in two ways:
  - they pointed at Glazed instead of `sanitize/`
  - they taught helper APIs that no longer matched the real code (`DebouncedYAML`, `Feed(...)`)
- Keeping one sanitize-aware parse path avoids the common regression where partial parsing and final parsing drift apart over time.

### What worked

- The same malformed YAML fixture already used in `agentmode` tests (`name:test`) is enough to prove the default-on sanitization contract.
- Focused tests (`go test ./pkg/events/structuredsink/... -count=1`) passed quickly in workspace mode.
- A standalone repo validation with `GOWORK=off` exposed the exact dependency pinning work needed for `go.sum`.
- The full pre-commit hooks on the code commit passed, including `go test ./...`, lint, and vet.

### What didn't work

- The first standalone test pass failed because `geppetto/go.sum` did not yet include the new dependency:

```text
missing go.sum entry for module providing package github.com/go-go-golems/sanitize/pkg/yaml
```

- After adding the direct module requirement, the next standalone pass still failed on transitive `tree-sitter` checksum entries:

```text
missing go.sum entry for module providing package github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go
missing go.sum entry for module providing package github.com/tree-sitter/go-tree-sitter
```

- Resolution:
  - `cd geppetto && GOWORK=off go get github.com/go-go-golems/sanitize@v0.0.1`
  - `cd geppetto && GOWORK=off go get github.com/go-go-golems/sanitize/pkg/yaml@v0.0.1`
  - reran `cd geppetto && GOWORK=off go test ./pkg/events/structuredsink/... -count=1`

### What I learned

- `go.work` can hide real standalone-module dependency gaps. For cross-repo workspace development, it is worth doing at least one `GOWORK=off` validation pass whenever a new module import is introduced.
- The `sanitize` module update also bumped `github.com/spf13/cobra` from `v1.10.1` to `v1.10.2` in `geppetto/go.mod`. That was a legitimate transitive effect of the new dependency pinning, not unrelated drift.
- The GP-59 ticket files had been staged in the repo already, so the first implementation commit also swept them in. I treated the next step as the intentional doc-follow-up rather than rewriting history.

### What was tricky to build

- The sharp edge was not the parser logic itself. It was preserving the smallest reasonable public API while still making the docs useful. The earlier design sketched an exported `ParseYAMLBytes(...)` helper, but the shipped code ended up cleaner without it because `FinalBytes(...)` already covers final-only parse flows.
- Another subtle edge was module hygiene. The code compiled fine in workspace mode because `go.work` pointed at `./sanitize`, but `geppetto` as its own module still needed full checksum materialization.

### What warrants a second pair of eyes

- The sanitize heuristic itself is intentionally permissive. A reviewer should sanity-check whether there are known extractor payloads where auto-repair would be less desirable than strict rejection.
- The docs now show `sanitize_yaml` as a config concept because the field has JSON/YAML tags, but there is not yet a higher-level end-to-end example of reading that option from a config file. That is acceptable for this ticket, but worth keeping in mind.

### What should be done in the future

- Optional: add a small end-to-end extractor smoke test through `FilteringSink` if future regressions suggest the helper tests are not enough.
- Optional: decide whether `parsehelpers` should expose richer sanitize metadata in the future.

### Code review instructions

- Start with `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`.
- Then read `geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go`.
- Then review the doc updates in:
  - `geppetto/pkg/doc/topics/11-structured-sinks.md`
  - `geppetto/pkg/doc/playbooks/03-progressive-structured-data.md`
  - `geppetto/pkg/doc/tutorials/04-structured-data-extraction.md`
- Validation commands:
  - `cd geppetto && go test ./pkg/events/structuredsink/... -count=1`
  - `cd geppetto && GOWORK=off go test ./pkg/events/structuredsink/... -count=1`

### Technical details

Commands used during implementation:

```bash
cd geppetto && go test ./pkg/events/structuredsink/... -count=1
cd geppetto && GOWORK=off go test ./pkg/events/structuredsink/... -count=1
cd geppetto && GOWORK=off go get github.com/go-go-golems/sanitize@v0.0.1
cd geppetto && GOWORK=off go get github.com/go-go-golems/sanitize/pkg/yaml@v0.0.1
git -C geppetto commit -m "Add YAML sanitization to structured sink parsehelpers"
```

## Step 3: Fix the block-scalar whitespace regression from review

After the initial GP-59 implementation landed, a review comment correctly pointed out that `normalizedYAML(...)` was trimming the entire YAML document before unmarshal. That looked harmless for malformed scalar cleanup, but it is semantically wrong for valid YAML block scalars because trailing newlines in `|` and `>` payloads are meaningful. The fix in this step was to separate "is this effectively empty?" from "what exact bytes should be unmarshalled?" and to add a regression test that proves `hello\n` survives intact.

This was a good example of why helper-layer mutations need semantic tests, not just parse-success tests. The original tests proved that sanitization could repair broken YAML, but they did not prove that the helper preserved already-valid YAML exactly enough for scalar semantics.

### Prompt Context

**User prompt (verbatim):** "not sure if this is valid?"

**Assistant interpretation:** Evaluate whether the code review comment about `strings.TrimSpace(...)` changing YAML semantics is correct, and fix it if it is.

**Inferred user intent:** Avoid overreacting to a possibly spurious review comment, but do not ship a subtle semantic regression in the generic YAML helper.

**Commit (code):** pending

### What I did

- Re-read `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go` to confirm that `strings.TrimSpace(string(body))` happened before unmarshal for both sanitize-on and sanitize-off paths.
- Changed `normalizedYAML(...)` so it:
  - uses a trimmed view only for the emptiness check
  - preserves the original body bytes for unmarshal when sanitization is disabled
  - preserves the sanitizer output verbatim when sanitization is enabled, unless the sanitized result is effectively empty
- Added `TestYAMLControllerFinalBytes_PreservesBlockScalarTrailingNewline` to `helpers_test.go`.
- Re-ran:
  - `cd geppetto && go test ./pkg/events/structuredsink/parsehelpers ./pkg/events/structuredsink -count=1`

### Why

- YAML block scalars are one of the clearest examples where document-level trimming mutates valid data before parse.
- The helper is generic infrastructure, so even a narrow semantic regression can affect multiple extractors at once.

### What worked

- The review comment was precise enough to point directly at the offending line and the right test shape.
- A tiny fixture was sufficient:

```yaml
message: |
  hello
```

- That fixture proves the intended decoded value is `hello\n`, which makes the regression easy to assert.

### What didn't work

- The first GP-59 tests only covered sanitize repair and strict opt-out failure. They did not cover exact semantic preservation for valid YAML, which is why this slipped through the first pass.

### What I learned

- "Parse succeeds" is not enough coverage for YAML helpers. The tests also need to protect semantics that are easy to damage with pre-parse normalization.
- Even when sanitization is enabled, it is safer to avoid unconditional trimming of the whole document and let the YAML parser see the original layout unless the sanitizer deliberately changes it.

### What was tricky to build

- The subtlety here is that trimming is reasonable for emptiness detection but not for actual parse input. Those are different questions, and the original implementation accidentally used one operation for both.

### What warrants a second pair of eyes

- A reviewer should quickly scan whether there are any other pre-unmarshal normalization steps in the codebase that make the same mistake with valid YAML or markdown-derived structured payloads.

### What should be done in the future

- Add more semantic-preservation tests if other YAML edge cases show up in review, especially around folded scalars and trailing blank lines.

### Code review instructions

- Start with `geppetto/pkg/events/structuredsink/parsehelpers/helpers.go`.
- Then review `geppetto/pkg/events/structuredsink/parsehelpers/helpers_test.go`.
- Validation command:
  - `cd geppetto && go test ./pkg/events/structuredsink/parsehelpers ./pkg/events/structuredsink -count=1`

### Technical details

Review comment summary:

```text
strings.TrimSpace on the whole YAML body changes payload semantics before parse,
which can corrupt extracted data for valid YAML block scalars.
```
