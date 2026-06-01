---
Title: Investigation diary
Ticket: GP-GOJA-API-2026-06-01
Status: active
Topics:
    - geppetto
    - js-bindings
    - goja
    - inference
    - intern-onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api_runner.go
      Note: Current runtime/runner split and prepared run flow reviewed for agent facade design
    - Path: geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: Current builder/session/run handle implementation that new fluent API should reuse
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: Current JS and Go tool registry bridge reviewed for typed tool builder design
    - Path: geppetto/pkg/js/modules/geppetto/codec.go
      Note: Current loose JS map to Go turn/block codec and target for strict builder validation
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Native module export wiring and runtime state reviewed for builder API design
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: xgoja runtime module creation reviewed for Geppetto provider host-service integration
    - Path: pinocchio/pkg/chatapp/runtime_inference.go
      Note: Downstream application runtime consumption reviewed to shape agent/event API requirements
ExternalSources: []
Summary: Chronological diary for the Geppetto go-go-goja API review and fluent builder design ticket.
LastUpdated: 2026-06-01T10:26:25.438318096-04:00
WhatFor: Use to resume or review the investigation, commands, decisions, validation, and delivery evidence.
WhenToUse: Before implementing or reviewing the proposed fluent Geppetto JS API changes.
---


# Diary

## Goal

This diary records the investigation and documentation work for improving the Geppetto go-go-goja API and JavaScript bindings. It captures what was inspected, what conclusions were drawn, what artifacts were produced, and what remains risky for implementation.

## Step 1: Create the ticket and map the relevant repositories

I created a new docmgr ticket for the Geppetto go-go-goja API review and set up the primary design document, diary, task list, scripts directory, and sources directory. The first goal was to turn the user’s broad redesign request into a traceable ticket workspace before drawing conclusions.

I then inspected the workspace layout and confirmed that the current directory is a multi-repository workspace containing `geppetto`, `go-go-goja`, `pinocchio`, and `glazed`. The actual docmgr root is under the Geppetto repository, so all ticket artifacts live in `geppetto/ttmp/...`.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to improve the current gepetto go-go-goja api and js bindings. First, let's examine the current situation and do a proper review, what is good, what could be better, what is bad, what is missing, what is confusing. We want an elegant fluid builder based API where most operations and object constructions are done on the go side so that we can guarantee strong typing / runtime validation, instead of relying on javascript maps and other more dynamic features. The goal is to provide an elegant, opinionated (but customizable) API to do LLM inference to for example build agents, and just in general llm / embeddings based scripts (for example with xgoja, which allows creating standalone applications that bundle together a whole set of moduels)>

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

Keep a diary as you work, feel free to run experiments in the scripts/ folder of the ticket as well, commit at appropriate intervals."

**Assistant interpretation:** Create a docmgr ticket, investigate the existing Geppetto JS/goja/xgoja binding architecture, write a detailed intern-facing review and redesign guide, keep a diary, validate the ticket, commit appropriate artifacts, and upload the final bundle to reMarkable.

**Inferred user intent:** The user wants a clear implementation roadmap for making Geppetto’s JS API safer, more elegant, more Go-backed, and suitable for LLM/agent/embedding scripts in goja/xgoja environments.

### What I did

- Ran `pwd`, `ls -la`, and repository status checks to understand the workspace.
- Ran `docmgr status --summary-only` to confirm the docmgr root.
- Created ticket `GP-GOJA-API-2026-06-01` with topics `geppetto,js-bindings,goja,inference,intern-onboarding`.
- Added a design document titled `Geppetto go-go-goja API Review and Builder Design Guide`.
- Added this diary document.
- Added tasks for architecture mapping, API review, builder design, implementation guide, validation, and reMarkable upload.

### Why

- The task is broad enough to require a ticket workspace and structured docs.
- Creating the ticket first keeps evidence, scripts, design notes, tasks, and changelog together.
- The user explicitly requested a diary and ticket storage.

### What worked

- `docmgr status --summary-only` found the configured root at `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp`.
- `docmgr ticket create-ticket` created the expected dated ticket workspace.
- `docmgr doc add` created both the design doc and diary.

### What didn't work

- The first command attempted `git status --short` from `/home/manuel/workspaces/2026-06-01/geppetto-js`, but that directory is not itself a Git repository:
  - Command: `pwd && git status --short && rg --files | head -200 && docmgr status --summary-only`
  - Error: `fatal: not a git repository (or any of the parent directories): .git`
- I corrected this by checking each subrepository (`geppetto`, `go-go-goja`, `pinocchio`, `glazed`) separately.

### What I learned

- The workspace is a multi-repo checkout with a `go.work` file, not a single Git repository.
- Docmgr is configured to store tickets in the Geppetto repo.
- Ticket-related commits should be made in the `geppetto` repository unless code changes are made elsewhere.

### What was tricky to build

- The main sharp edge was repository locality. The top-level workspace contains multiple repositories, but Git commands must be scoped to the correct subrepo. The symptom was a fatal Git error from the workspace root. The solution was to use `git -C <repo>` for status checks and keep doc artifacts under the Geppetto ticket tree.

### What warrants a second pair of eyes

- Confirm that `GP-GOJA-API-2026-06-01` is the desired ticket ID and naming convention.
- Confirm whether future implementation should land in only Geppetto or also in `go-go-goja` for xgoja host-service propagation.

### What should be done in the future

- If implementation starts, create separate focused commits per repo when code changes cross repository boundaries.

### Code review instructions

- Start with the ticket index and task list under `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings`.
- Validate ticket presence with `docmgr doc list --ticket GP-GOJA-API-2026-06-01`.

### Technical details

- Ticket path: `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings`

## Step 2: Gather evidence from Geppetto, go-go-goja, and Pinocchio

I inspected the current Geppetto JavaScript module, runtime wrapper, older embeddings wrapper, public TypeScript declarations, docs, example scripts, xgoja provider mechanics, and downstream Pinocchio runtime usage. This step was intentionally evidence-first: the design recommendations should be anchored to files rather than assumptions.

I also created a ticket script, `scripts/01-collect-evidence.sh`, that writes a line-numbered evidence snapshot into `sources/01-code-evidence.md`. This gives future implementers a reproducible starting point for reviewing the relevant files.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Inspect the current implementation before recommending a new API.

**Inferred user intent:** Avoid speculative redesign; make the intern guide file-backed and technically credible.

### What I did

- Read key Geppetto files:
  - `pkg/js/modules/geppetto/module.go`
  - `pkg/js/modules/geppetto/api_types.go`
  - `pkg/js/modules/geppetto/codec.go`
  - `pkg/js/modules/geppetto/api_turns.go`
  - `pkg/js/modules/geppetto/api_engines.go`
  - `pkg/js/modules/geppetto/api_sessions.go`
  - `pkg/js/modules/geppetto/api_builder_options.go`
  - `pkg/js/modules/geppetto/api_tools_registry.go`
  - `pkg/js/modules/geppetto/api_runner.go`
  - `pkg/js/modules/geppetto/provider/provider.go`
  - `pkg/js/runtime/runtime.go`
  - `pkg/js/embeddings-js.go`
  - `pkg/doc/types/geppetto.d.ts`
  - `pkg/doc/topics/13-js-api-reference.md`
  - `pkg/doc/topics/14-js-api-user-guide.md`
  - `pkg/doc/tutorials/05-js-api-getting-started.md`
- Read key go-go-goja/xgoja files:
  - `pkg/xgoja/providerapi/module.go`
  - `pkg/xgoja/app/factory.go`
  - `pkg/xgoja/app/host.go`
  - `pkg/xgoja/doc/01-runtime-overview.md`
- Read key Pinocchio files:
  - `pkg/inference/runtime/engine.go`
  - `pkg/inference/runtime/composer.go`
  - `pkg/chatapp/runtime_inference.go`
  - `pkg/chatapp/runtime_sink.go`
- Created and ran `scripts/01-collect-evidence.sh`.
- Ran a line-count inventory over relevant files to identify large complexity centers.

### Why

- The API redesign needs to respect existing runtime-owner, session, toolloop, profile, event, and xgoja behavior.
- Pinocchio provides a real application example of how Geppetto runtime engines and event sinks are consumed.
- The old embeddings wrapper shows the migration need for first-class embeddings in `require("geppetto")`.

### What worked

- File discovery showed the Geppetto JS API is concentrated under `pkg/js/modules/geppetto`.
- The existing module already uses Go-backed refs via a non-enumerable hidden reference key.
- The public `.d.ts` file and example scripts give a clear current API baseline.
- The evidence script successfully generated `sources/01-code-evidence.md`.

### What didn't work

- One attempted read of `go-go-goja/pkg/xgoja/providerapi/provider.go` failed because that file does not exist:
  - Error: `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-01/geppetto-js/go-go-goja/pkg/xgoja/providerapi/provider.go'`
- I corrected this by locating and reading `go-go-goja/pkg/xgoja/providerapi/module.go`.

### What I learned

- The current design is better than a simple map bridge: objects such as engines, builders, sessions, tool registries, resolved profiles, and prepared runs carry Go references.
- The weak spot is that many inputs are still JavaScript object maps decoded through `map[string]any`.
- The xgoja Geppetto provider requires host services, while the generic runtime factory path needs a clear host injection story.
- Pinocchio’s runtime path reinforces the need to model app-owned runtime policy, event sinks, persistence, and profile provenance explicitly.

### What was tricky to build

- The evidence spans three repositories and several layers. The underlying cause is that the JS API is not isolated: it sits between Geppetto domain objects, go-go-goja runtime/module loading, and Pinocchio application runtime behavior. I approached this by grouping evidence into core module files, runtime/xgoja files, and downstream app files, then preserving a line-numbered snapshot in the ticket.

### What warrants a second pair of eyes

- The conclusion that xgoja host services need runtime factory changes should be reviewed against any generated-adapter code not covered in this first pass.
- The assessment of the old embeddings wrapper should be checked against current embedding provider interfaces before implementation.

### What should be done in the future

- Extend the evidence script if implementation uncovers additional provider, embedding, or generated xgoja files.

### Code review instructions

- Start with `sources/01-code-evidence.md` for line-numbered excerpts.
- Then read the source files directly to avoid relying on a stale evidence snapshot.
- Re-run `scripts/01-collect-evidence.sh` after substantial code changes.

### Technical details

- Evidence script: `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/scripts/01-collect-evidence.sh`
- Evidence output: `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/sources/01-code-evidence.md`

## Step 3: Write the intern-facing API review and builder design guide

I wrote the primary design document as a detailed review and implementation guide. It explains the current architecture, what is good, what could be better, what is bad or risky, what is missing, what is confusing, and how to implement a staged Go-backed fluent builder API.

The design recommends adding an opinionated facade (`gp.agent()`, `gp.chat()`, `gp.embeddings()`), typed builders (`gp.turn()`, `gp.engine()`, `gp.schema`), and preserving the current lower-level namespaces as advanced compatibility APIs.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Produce the requested detailed analysis/design/implementation guide in the ticket.

**Inferred user intent:** Give a new intern enough context to understand the current system and implement a safer, more elegant JS binding without breaking existing usage.

### What I did

- Replaced the generated design-doc template with a full design guide.
- Included diagrams, pseudocode, API sketches, migration tables, file references, implementation phases, and testing strategy.
- Proposed key API shapes:
  - `gp.agent().profile(...).system(...).tool(...).build().ask(...)`
  - `gp.turn().system(...).user(...).toolCall(...).build()`
  - `gp.engine().provider(...).model(...).timeoutMs(...).build()`
  - `gp.embeddings().provider(...).model(...).build().embed(...)`
  - `gp.schema.object({ ... })`
- Called out review-critical risks around runtime-owner threading, xgoja host services, validation drift, facade magic, TypeScript drift, and credentials policy.

### Why

- The user asked for a clear and technical intern-facing guide, not just a summary.
- The proposed design needs to be phased so a new engineer can ship small safe vertical slices.
- The system has enough existing pieces that the right approach is evolutionary, not a full rewrite.

### What worked

- The current implementation already provides several patterns the new API can reuse: module runtime state, hidden refs, engine/session/tool refs, runtime-owner bridge, tests, docs, and examples.
- The design could therefore recommend adding builders and facades without discarding the advanced API.

### What didn't work

- No code implementation was attempted in this step. The output is design-only.
- The current docs contain absolute paths from a prior workspace in `pkg/doc/topics/13-js-api-reference.md`; this was noted implicitly as documentation drift but not modified.

### What I learned

- The best path is to expand the Go-backed reference model to more domain objects, especially turns, blocks, schemas, embedding requests/results, and run results.
- The main API confusion is not that the low-level pieces are wrong; it is that the first-time user lacks one obvious happy path.

### What was tricky to build

- The tricky part was balancing an opinionated facade with the existing hard-cut separation between engine profiles and app-owned runtime policy. The design resolves this by keeping `profiles`, `runner`, and `engines` as advanced layers while adding `agent()` as a facade that composes those concepts transparently.

### What warrants a second pair of eyes

- API naming: `agent()`, `chat()`, `engine()`, `turn()`, and `embeddings()` should be reviewed for long-term ergonomics.
- Migration policy: reviewers should decide how long object-map APIs remain “advanced” before becoming deprecated.
- Strict-vs-loose behavior for turn/block decoding should be reviewed carefully because existing users may rely on permissive behavior.

### What should be done in the future

- Implement Phase 1 (`gp.turn()` and `gp.schema`) before the higher-level `agent()` facade.
- Update public docs and examples once the first implementation slice is merged.

### Code review instructions

- Review the design document from `Executive Summary` through `Implementation Plan` first.
- Pay special attention to:
  - `What Is Bad or Risky`
  - `Proposed Architecture`
  - `Internal Go Design Sketch`
  - `Implementation Plan`
  - `Risks and Review-Critical Points`

### Technical details

- Design doc: `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md`

## Step 4: Validate the current JS module tests

I ran the focused Geppetto JS module and runtime tests to make sure the review did not start from a broken baseline. This was not a full repository test run; it was a targeted validation of the packages most directly relevant to the design.

The tests passed, which strengthens the recommendation to evolve the current module rather than replace it wholesale.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the current state enough to support a credible review and handoff.

**Inferred user intent:** Avoid producing design guidance on top of an unknown or broken baseline.

### What I did

- Ran:
  - `cd geppetto && go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`

### Why

- These packages contain the native module and runtime wrapper that the redesign will modify.
- Passing tests provide a stable baseline for future implementation.

### What worked

- Test output:
  - `ok  	github.com/go-go-golems/geppetto/pkg/js/modules/geppetto	0.119s`
  - `ok  	github.com/go-go-golems/geppetto/pkg/js/runtime	0.011s`

### What didn't work

- N/A for this step.

### What I learned

- The focused JS module/runtime baseline is currently green.
- A future implementation can use these packages as the first validation loop.

### What was tricky to build

- N/A; this was a targeted validation command.

### What warrants a second pair of eyes

- A full `go test ./...` was not run across Geppetto, go-go-goja, and Pinocchio because this task focused on documentation and review.

### What should be done in the future

- Before code changes merge, run at least:
  - `cd geppetto && go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
  - `cd geppetto && go test ./... -count=1`
  - any relevant `go-go-goja` xgoja tests if host-service changes are made.

### Code review instructions

- Use the targeted test command as the minimum fast validation loop for JS binding changes.

### Technical details

- Command: `cd geppetto && go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`

## Step 5: Commit, validate, and upload the ticket bundle

I committed the initial ticket artifacts, validated the docmgr frontmatter and ticket health, fixed the vocabulary/source-frontmatter issues found by `docmgr doctor`, and uploaded the final design+diary bundle to reMarkable. This step converted the investigation from a local draft into a reviewable and delivered ticket package.

The first doctor run caught two issues: `goja` was not yet in the topic vocabulary, and the generated evidence markdown lacked docmgr frontmatter. I fixed both by adding the vocabulary entry and updating the evidence script to emit frontmatter before regenerating `sources/01-code-evidence.md`.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finalize the ticket deliverables, validate them, commit them, and upload them to reMarkable.

**Inferred user intent:** Make the design package durable, discoverable, and available on the user’s reMarkable device.

**Commit (code):** 321fb82ba9f09a5684d92805ca294783370ec1b9 — "Docs: review Geppetto goja JS API redesign"

### What I did

- Ran targeted Geppetto tests before finalization:
  - `cd geppetto && go test ./pkg/js/modules/geppetto ./pkg/js/runtime -count=1`
- Ran frontmatter validation for the design doc and diary.
- Ran `docmgr doctor --ticket GP-GOJA-API-2026-06-01 --stale-after 30`.
- Fixed doctor findings:
  - added vocabulary topic `goja`;
  - updated and reran `scripts/01-collect-evidence.sh` so `sources/01-code-evidence.md` has frontmatter.
- Re-ran `docmgr doctor`, which passed.
- Committed the initial docs package:
  - `321fb82ba9f09a5684d92805ca294783370ec1b9`
- Verified reMarkable status and account:
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
- Ran reMarkable dry-run bundle upload.
- Uploaded the real bundle.
- Verified remote listing:
  - `/ai/2026/06/01/GP-GOJA-API-2026-06-01`

### Why

- The ticket needed a clean doctor report before upload.
- The user explicitly requested reMarkable upload.
- Committing the docs creates a stable checkpoint before future implementation changes.

### What worked

- `docmgr doctor` passed after adding the vocabulary entry and source frontmatter.
- `remarquee status` returned `remarquee: ok`.
- `remarquee cloud account --non-interactive` returned `user=wesen@ruinwesen.com sync_version=1.5`.
- Dry-run upload printed the expected bundle plan.
- Real upload succeeded:
  - `OK: uploaded GP-GOJA-API-2026-06-01 Geppetto goja API review.pdf -> /ai/2026/06/01/GP-GOJA-API-2026-06-01`
- Remote listing showed:
  - `[f]	GP-GOJA-API-2026-06-01 Geppetto goja API review`

### What didn't work

- The first frontmatter validation command passed paths prefixed with `geppetto/ttmp/...`; docmgr interpreted them relative to its docs root and failed:
  - Error: `open /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/geppetto/ttmp/...: no such file or directory`
  - Fix: reran validation with absolute paths.
- The first `docmgr doctor` reported:
  - warning: unknown topic `goja`;
  - error: `sources/01-code-evidence.md` missing frontmatter delimiters.
- After removing the placeholder task, task IDs shifted; `docmgr task check --id 6` failed with `task id(s) not found: [6]`. I listed tasks and checked the correct final task ID, `5`.

### What I learned

- Docmgr path arguments are safest as absolute paths when validating docs from outside the docs root.
- Generated source markdown under a ticket still needs frontmatter unless ignored.
- Removing placeholder tasks changes docmgr task IDs.

### What was tricky to build

- The tricky part was keeping generated evidence useful while satisfying docmgr validation. The evidence script originally emitted plain markdown, which was good for humans but invalid for doctor. The fix was to make the script emit full frontmatter before the evidence body and then regenerate the source artifact.

### What warrants a second pair of eyes

- The first commit predates the final diary/changelog/task updates from upload completion, so reviewers should inspect the follow-up docs commit too.
- The uploaded bundle includes the design doc and diary, not the large generated evidence source, to keep the reMarkable PDF readable.

### What should be done in the future

- If the implementation begins, create a new code-focused step and commit separate from the research/design commits.

### Code review instructions

- Validate ticket health with:
  - `docmgr doctor --ticket GP-GOJA-API-2026-06-01 --stale-after 30`
- Verify upload with:
  - `remarquee cloud ls /ai/2026/06/01/GP-GOJA-API-2026-06-01 --long --non-interactive`

### Technical details

- reMarkable remote directory: `/ai/2026/06/01/GP-GOJA-API-2026-06-01`
- Bundle name: `GP-GOJA-API-2026-06-01 Geppetto goja API review`

## Step 6: Revise the design to a hard-cut Go-wrapper API model

The user pushed back on the earlier framing that hidden Go references were the right bridge pattern. I revised the design accordingly: hidden `__geppetto_ref` is now documented as a current transitional identity-preservation mechanism, not the target architecture.

The new target is a hard-cut public API where JavaScript manipulates Go-owned wrapper values directly through methods. Plain object/map construction is removed from the normal public API; explicit serialization and `gp.unsafe` import boundaries are the only places where raw JavaScript objects should appear.

### Prompt Context

**User prompt (verbatim):** "no, let's do ideal API model, we can do a hard cutover as well, we don't have legacy to support. 

Redesign."

**Assistant interpretation:** Rewrite the design away from compatibility and hidden-ref-first thinking, and define the ideal hard-cut API model.

**Inferred user intent:** The user wants the design to prioritize the cleanest long-term API rather than preserving current dynamic JavaScript maps or transitional facade mechanics.

### What I did

- Rewrote the executive summary to state that this is a hard-cut API redesign.
- Reframed `__geppetto_ref` as a current implementation detail used to preserve Go identity, not the desired conceptual API model.
- Added a pros/cons table comparing:
  - hidden `__geppetto_ref` facades;
  - direct Go wrapper values exposed to JS;
  - plain JS maps decoded later.
- Added an `Ideal object model` section:
  - Go constructs objects;
  - Go owns state;
  - Go validates every mutation;
  - JavaScript uses method calls and explicit `toJSON()`/`snapshot()` boundaries.
- Replaced the design principles with hard-cut principles:
  - Go-owned wrappers;
  - no map-first constructors;
  - strict by default;
  - explicit serialization;
  - unsafe APIs are named as unsafe.
- Added a final public module contract sketch with only `agent`, `chat`, `turn`, `engine`, `tool`, `toolRegistry`, `embeddings`, `schema`, `engineProfiles`, `events`, and `unsafe`.
- Replaced the migration strategy with a hard cutover strategy.
- Updated pseudocode to prefer direct Go wrapper values rather than conceptual hidden refs.

### Why

- The earlier version still treated hidden refs and compatibility as too central.
- The user clarified that there is no legacy API burden, so the design should optimize for the ideal API.
- A hard cut makes the implementation and documentation cleaner: the public contract can be small and strict from the start.

### What worked

- The existing document structure was still useful; the redesign could be made by changing the API model and cutover sections rather than starting from a blank document.
- The current `__geppetto_ref` mechanism became easier to explain once categorized as transitional evidence that identity matters.

### What didn't work

- The prior design language was too compatibility-oriented. It said to keep current namespaces and add preferred APIs next to them. That is no longer the recommendation.

### What I learned

- The clean model is not “JS object with hidden Go pointer”. The clean model is “Go wrapper value with a JS-friendly method surface”.
- `__geppetto_ref` answers a technical problem in the current facade implementation, but it should not define the new public mental model.

### What was tricky to build

- The tricky part was separating implementation mechanics from API semantics. goja may still require helper objects or hidden references in some places, but the public design should not expose or depend on that. The design now states that hidden refs are allowed only as low-level implementation fallback, while public objects are direct typed wrappers.

### What warrants a second pair of eyes

- The final public module contract should be reviewed carefully before implementation because it intentionally removes current map-first exports from the ordinary API.
- The `gp.unsafe` namespace should be kept minimal so it does not become the new dumping ground for dynamic behavior.

### What should be done in the future

- Update the TypeScript declaration generator to emit only the hard-cut public API.
- Add an API inventory test that fails if old map-first names remain public after the cutover.

### Code review instructions

- Start with these sections in the design doc:
  - `Current wrapper/reference pattern: useful evidence, not the target model`
  - `Pros and cons: hidden refs vs direct Go wrappers`
  - `Ideal object model`
  - `Final public module contract after hard cut`
  - `Hard Cutover Strategy`

### Technical details

- Updated design doc: `/home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md`

### Delivery note for Step 6

After committing the hard-cut redesign, I re-ran the reMarkable bundle dry-run and then uploaded with `--force` because the previous bundle already existed at the same remote path. The updated PDF is now present at `/ai/2026/06/01/GP-GOJA-API-2026-06-01`.

Exact upload behavior:

- First real upload attempt skipped with: `SKIP: GP-GOJA-API-2026-06-01 Geppetto goja API review already exists in /ai/2026/06/01/GP-GOJA-API-2026-06-01 (use --force to overwrite)`.
- Forced upload succeeded with: `OK: uploaded GP-GOJA-API-2026-06-01 Geppetto goja API review.pdf -> /ai/2026/06/01/GP-GOJA-API-2026-06-01`.
