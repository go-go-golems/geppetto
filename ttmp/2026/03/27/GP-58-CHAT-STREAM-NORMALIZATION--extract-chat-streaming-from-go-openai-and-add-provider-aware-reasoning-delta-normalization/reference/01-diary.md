---
Title: Diary
Ticket: GP-58-CHAT-STREAM-NORMALIZATION
Status: active
Topics:
    - inference
    - streaming
    - reasoning
    - geppetto
    - chat
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go
    - Path: pkg/embeddings/openai.go
      Note: Diary documents that embeddings stay on go-openai
    - Path: pkg/steps/ai/openai/engine_openai.go
      Note: Diary references the current stream decode loop
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Diary records request builder scope and phased migration choice
    - Path: pkg/steps/ai/openai/transcribe.go
      Note: Diary documents that transcription stays on go-openai
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Diary records the existing direct SSE reference pattern
    - Path: ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/raw-sse.txt
      Note: Diary cites the Together wire-format evidence reused in this ticket
ExternalSources:
    - https://github.com/openai/openai-go
    - https://docs.together.ai/docs/openai-api-compatibility
Summary: Chronological diary for the chat-streaming extraction design ticket.
LastUpdated: 2026-03-27T19:07:20.987625046-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture how the chat-streaming extraction ticket was created, what evidence was gathered, what design decisions were made, and how the ticket package was validated and delivered.

## Step 1: Create the new ticket and map the current chat-streaming boundary

I created a new ticket specifically for the narrow refactor the user requested: remove `go-openai` only from chat streaming, while keeping embeddings and transcription on the existing client. The first task was to avoid broad conclusions and instead re-establish the exact code boundaries in the current repo.

The investigation immediately confirmed that the chat path, the embeddings path, and the transcription path are already separated cleanly enough to document as independent scopes. That made it possible to recommend a focused refactor rather than a repo-wide SDK replacement.

### Prompt Context

**User prompt (verbatim):** "ok, remove it only for chat_streaming then, and keep for embeddings and transcription. Make a new ticket that shows all that in detail. Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new `docmgr` ticket focused on extracting the chat-streaming path from `go-openai`, document the architecture and implementation plan in detail, keep a diary, and upload the resulting bundle to reMarkable.

**Inferred user intent:** The user wants a concrete engineering package that can hand off the scoped refactor to another engineer or intern without ambiguity.

**Commit (code):** N/A

### What I did

- Ran `docmgr ticket create-ticket --ticket GP-58-CHAT-STREAM-NORMALIZATION --title "Extract chat streaming from go-openai and add provider-aware reasoning delta normalization" --topics ai,inference,streaming,openai,together`
- Ran `docmgr doc add --ticket GP-58-CHAT-STREAM-NORMALIZATION --doc-type design-doc --title "Intern guide to extracting chat streaming from go-openai and normalizing provider reasoning deltas"`
- Ran `docmgr doc add --ticket GP-58-CHAT-STREAM-NORMALIZATION --doc-type reference --title "Diary"`
- Inspected:
  - `pkg/steps/ai/openai/engine_openai.go`
  - `pkg/steps/ai/openai/helpers.go`
  - `pkg/embeddings/openai.go`
  - `pkg/steps/ai/openai/transcribe.go`
  - `pkg/steps/ai/openai_responses/engine.go`
  - `pkg/events/chat-events.go`
  - `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go`

### Why

- The user asked for a new ticket, not an edit to the previous Together investigation ticket.
- The new ticket needed to explain both "what to replace" and "what not to replace."
- Line-anchored evidence was necessary because this is an architecture ticket, not a brainstorming note.

### What worked

- `docmgr` ticket creation and document creation succeeded immediately.
- The codebase layout made scope separation straightforward:
  - chat streaming is in `pkg/steps/ai/openai/`
  - embeddings are in `pkg/embeddings/openai.go`
  - transcription is in `pkg/steps/ai/openai/transcribe.go`

### What didn't work

- I initially tried to inspect `pkg/turns/turn.go`, but that file does not exist in this repo layout.
- The correct file is `pkg/turns/types.go`.

### What I learned

- The repo already has a strong reference implementation for direct HTTP + SSE decoding in `pkg/steps/ai/openai_responses/engine.go`.
- That existing engine reduces the amount of new design invention needed for the chat-completions refactor.

### What was tricky to build

- The main tricky part was keeping the scope narrow. The code invites a larger conversation about replacing `go-openai` everywhere, but the user explicitly asked for chat streaming only.
- To keep the design honest, I had to separate "runtime stream decoding" from "request-building types." Those are related but not identical concerns.

### What warrants a second pair of eyes

- Whether Phase 1 should keep `go_openai.ChatCompletionRequest` as the request type or introduce a new internal request type immediately.
- Whether reasoning blocks produced from chat-completions streams should exactly mirror the Open Responses payload shape or use a reduced payload limited to text-only reasoning.

### What should be done in the future

- Implement the actual refactor described in this ticket.
- After that lands, evaluate whether the remaining chat request structs should also move away from `go-openai`.

### Code review instructions

- Start with `pkg/steps/ai/openai/engine_openai.go` and confirm where the existing stream loop loses provider-specific fields.
- Compare that with `pkg/steps/ai/openai_responses/engine.go` to understand the direct SSE pattern already used in the repo.
- Validate the scope boundary by checking `pkg/embeddings/openai.go` and `pkg/steps/ai/openai/transcribe.go`.

### Technical details

- Key commands:

```bash
docmgr ticket create-ticket --ticket GP-58-CHAT-STREAM-NORMALIZATION --title "Extract chat streaming from go-openai and add provider-aware reasoning delta normalization" --topics ai,inference,streaming,openai,together
docmgr doc add --ticket GP-58-CHAT-STREAM-NORMALIZATION --doc-type design-doc --title "Intern guide to extracting chat streaming from go-openai and normalizing provider reasoning deltas"
docmgr doc add --ticket GP-58-CHAT-STREAM-NORMALIZATION --doc-type reference --title "Diary"
nl -ba pkg/steps/ai/openai/engine_openai.go | sed -n '1,360p'
nl -ba pkg/steps/ai/openai/helpers.go | sed -n '1,760p'
nl -ba pkg/embeddings/openai.go | sed -n '1,220p'
nl -ba pkg/steps/ai/openai/transcribe.go | sed -n '300,620p'
```

## Step 2: Connect the Together failure to the new ticket scope

After the code-path mapping, I pulled evidence from the previous Together ticket to make sure this new design ticket was driven by observed wire-format behavior instead of assumptions. That prior work was important because the new ticket is not only about software architecture; it is about a real provider mismatch already reproduced locally.

The raw Together SSE capture showed that the provider emits reasoning tokens in `delta.reasoning` while `delta.content` stays empty. That is the key observation that justifies replacing the chat-streaming decode boundary rather than just tweaking event publication.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Bring the prior Together reasoning investigation into the new design ticket so the recommendations stay evidence-based.

**Inferred user intent:** The user wants the new ticket to explain why the refactor is needed, not only how to do it.

**Commit (code):** N/A

### What I did

- Read the prior raw SSE capture:
  - `ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/raw-sse.txt`
- Confirmed that the captured stream contains repeated `delta.reasoning` chunks and empty `delta.content`.
- Rechecked the `go-openai` stream delta type in `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/chat_stream.go`.

### Why

- This establishes that the provider is not the missing link.
- It also proves that a simple SDK upgrade is not enough if the field name mismatch persists.

### What worked

- The prior ticket already had the raw evidence in a reusable location.
- The mismatch was easy to express in the new design doc as a concrete "wire format vs typed struct" gap.

### What didn't work

- Nothing blocked this step technically.

### What I learned

- The real bug is two-stage:
  - the library drops `reasoning`
  - Geppetto ignores reasoning even if it were present

### What was tricky to build

- The tricky part was wording the conclusion precisely. It would be easy but inaccurate to say "Together is unsupported." The more accurate statement is "Together reasoning is present on the wire and lost by our current decoding stack."

### What warrants a second pair of eyes

- The raw Together stream example should probably be converted into a sanitized fixture during implementation so future tests do not depend on the prior ticket artifact.

### What should be done in the future

- Promote the relevant raw Together stream into `pkg/steps/ai/openai/testdata/chat-stream/`.

### Code review instructions

- Open the raw SSE file and compare one chunk to the `ChatCompletionStreamChoiceDelta` struct.
- Confirm the mismatch:
  - provider field: `reasoning`
  - typed field: `reasoning_content`

### Technical details

- Key command:

```bash
sed -n '1,220p' ttmp/2026/03/27/GP-57-TOGETHER-THINKING--investigate-missing-together-qwen-thinking-stream-in-openai-compatible-chat-completions/sources/experiments/raw-sse.txt
```

## Step 3: Write the design package and explain the implementation strategy

With the evidence assembled, I wrote the primary design document. I focused the document on onboarding value: an intern should be able to read it, understand why the current system fails, identify the correct files to change, and see a staged plan that minimizes risk.

The design document deliberately recommends a narrow first implementation: keep the existing request builder mostly intact, replace the streaming transport and decode logic, normalize reasoning fields, and reuse the event semantics already proven in the Open Responses engine.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce a highly detailed intern guide with prose, diagrams, pseudocode, API references, and file references.

**Inferred user intent:** The user wants the new ticket to be actionable by someone unfamiliar with this subsystem.

**Commit (code):** N/A

### What I did

- Updated the ticket `index.md`
- Wrote `tasks.md` with:
  - documentation deliverables
  - implementation phases
  - testing plan
  - deferred work
- Wrote the primary design doc with:
  - current-state analysis
  - root-cause analysis
  - architecture diagrams
  - pseudocode for the SSE reader and normalizer
  - phased implementation plan
  - risks and alternatives

### Why

- The user explicitly asked for an intern-oriented design and implementation guide.
- The plan needed to be sufficiently specific that another engineer could implement it without rediscovering the underlying reasoning.

### What worked

- The existing Open Responses engine gave the design doc a strong concrete reference pattern.
- The clean separation between chat, embeddings, and transcription made the scoped plan easier to explain.

### What didn't work

- The generated design doc template was too skeletal, so I replaced it completely rather than editing small placeholder sections.

### What I learned

- Geppetto already has the event vocabulary needed for reasoning in the shared events package.
- That means the implementation can focus on transport, normalization, and turn persistence instead of new event types.

### What was tricky to build

- The tricky part was specifying the right "middle" abstraction. If the design stopped at raw maps, the engine would stay messy. If it introduced too many new abstractions, the implementation would become over-engineered.
- The chosen compromise was a small normalized event struct that captures only the fields the engine actually needs.

### What warrants a second pair of eyes

- The proposed file split under `pkg/steps/ai/openai/` should be reviewed to make sure it fits the repo's preferred package style.
- The "minimal-change request builder" recommendation is pragmatic, but a reviewer may prefer a cleaner full decoupling immediately.

### What should be done in the future

- Convert the design into implementation tasks in code.
- After implementation, update the docs with the actual file layout that lands.

### Code review instructions

- Read the design doc from top to bottom once without opening code.
- On the second pass, open each referenced file and verify that the evidence claims match the code.
- Pay special attention to the sections on:
  - current-state architecture
  - design decisions
  - risks and sharp edges

### Technical details

- Main outputs written:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
  - `design-doc/01-intern-guide-to-extracting-chat-streaming-from-go-openai-and-normalizing-provider-reasoning-deltas.md`
  - `reference/01-diary.md`

## Step 4: Relate files, validate the ticket, and upload to reMarkable

Once the docs were written, I linked the key code files to the design doc and the diary so the ticket is navigable from both directions. After that I ran `docmgr doctor` and then the standard reMarkable bundle flow with a dry run first and a real upload second.

This step matters because the user asked for a stored ticket artifact, not just a local markdown file. The delivery is only complete when the ticket is validated and the reMarkable upload is confirmed remotely.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket bookkeeping and upload the package to reMarkable.

**Inferred user intent:** The user wants the analysis package available both in the repo ticket workspace and on the reMarkable.

**Commit (code):** N/A

### What I did

- Ran `docmgr doc relate` updates for the design doc, diary, and ticket index
- Ran `docmgr doctor --ticket GP-58-CHAT-STREAM-NORMALIZATION --stale-after 30`
- Fixed doctor warnings by:
  - changing the frontmatter topics to vocabulary-backed values
  - adding file notes to the ticket index relations
- Ran `remarquee status`
- Ran `remarquee cloud account --non-interactive`
- Ran a dry-run bundle upload
- Ran the real bundle upload
- Verified the remote listing with `remarquee cloud ls /ai/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION --long --non-interactive`

### Why

- Ticket quality depends on relations and validation, not just file creation.
- The reMarkable upload is an explicit part of the user request.

### What worked

- `docmgr doctor` passed cleanly after the topic and related-file-note fixes.
- The dry-run bundle upload succeeded without surprises.
- The real upload succeeded:
  - `OK: uploaded GP-58 Chat Streaming Extraction Design Package.pdf -> /ai/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION`
- Remote verification succeeded:
  - `[f] GP-58 Chat Streaming Extraction Design Package`

### What didn't work

- The first `remarquee cloud ls` check happened before the upload had finished, so it returned:

```text
Error: no matches for 'GP-58-CHAT-STREAM-NORMALIZATION'
```

- Waiting for the upload session to finish and then rerunning `cloud ls` resolved the issue.

### What I learned

- The ticket workflow benefits from doing file relations after the main design doc is written, because the final relation set becomes much cleaner.

### What was tricky to build

- The only tricky part in this step is keeping the frontmatter and related-file sets tight enough that `docmgr doctor` stays clean while still linking the evidence-bearing files.

### What warrants a second pair of eyes

- Check that the final bundle upload includes the design doc and diary in the intended order.

### What should be done in the future

- If this ticket becomes the implementation ticket later, continue the diary in this same file with code-step entries and commit hashes.

### Code review instructions

- Verify `docmgr doctor` passes.
- Verify the reMarkable upload path matches the ticket id and date.

### Technical details

- Commands used:

```bash
docmgr doc relate --doc /abs/path/to/design-doc.md --file-note "/abs/path/to/file.go:why it matters"
docmgr doc relate --doc /abs/path/to/reference/01-diary.md --file-note "/abs/path/to/file.go:why it matters"
docmgr doctor --ticket GP-58-CHAT-STREAM-NORMALIZATION --stale-after 30
remarquee status
remarquee cloud account --non-interactive
remarquee upload bundle --dry-run <docs...> --name "GP-58 Chat Streaming Extraction Design Package" --remote-dir "/ai/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION" --toc-depth 2
remarquee upload bundle <docs...> --name "GP-58 Chat Streaming Extraction Design Package" --remote-dir "/ai/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION" --toc-depth 2
remarquee cloud ls /ai/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION --long --non-interactive
```

## Step 5: Start implementation and convert the ticket plan into commit-sized work

The user then moved the ticket from design into implementation and asked that the tasks be worked off one by one with commits at appropriate intervals. Before changing code, I re-opened the ticket tasks and the repo state to make sure the implementation work could proceed in clean slices and without accidentally staging unrelated ticket directories.

This step deliberately did not start coding immediately. The point was to create a stable execution order: first the transport and normalization layer, then the engine refactor, then the regression tests, then the final ticket bookkeeping.

### Prompt Context

**User prompt (verbatim):** "Ok, create detailed tasks for the ticket, and work them off one by one. Commit at appropriate intervals and keep a detailed diary as you work."

**Assistant interpretation:** Break the implementation into ordered tasks, keep the ticket diary current during coding, and make focused commits instead of one large final commit.

**Inferred user intent:** The user wants an auditable implementation trail, not just the final code state.

**Commit (code):** N/A

### What I did

- Checked the repo state with `git status --short`
- Reopened the ticket task list
- Expanded the implementation plan into explicit ordered steps:
  - endpoint/config helper
  - SSE reader
  - normalized streaming types
  - reasoning/tool delta normalization
  - engine refactor
  - reasoning event publication
  - reasoning block persistence
  - regression tests

### Why

- The refactor touches both transport and engine semantics.
- Without an ordered task list, it would be easy to mix partial transport work with unfinished event behavior and make the intermediate commits hard to review.

### What worked

- The working tree only showed the ticket directories as untracked, so there was no unrelated source change to work around.
- The existing task list was already close to the right shape and only needed finer granularity.

### What didn't work

- N/A

### What I learned

- The cleanest first implementation slice is the new streaming transport plus normalization layer, because the engine can then be refactored against a stable internal contract.

### What was tricky to build

- The tricky part was choosing the commit boundaries. The transport layer alone is not user-visible, but it is the right first checkpoint because it isolates the riskiest replacement boundary away from the event logic.

### What warrants a second pair of eyes

- Review whether the chosen step order is the best one, especially if a reviewer would rather land parser tests before transport helpers.

### What should be done in the future

- After each implementation slice lands, update this diary with the exact files changed, tests run, and commit hash.

### Code review instructions

- Review the updated `tasks.md` first. It now represents the intended commit order.
- Then compare each future code commit back against those steps rather than reviewing the final diff as one blob.

### Technical details

- Commands used:

```bash
git -C /home/manuel/workspaces/2026-03-27/use-open-responses/geppetto status --short
sed -n '1,240p' ttmp/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION--extract-chat-streaming-from-go-openai-and-add-provider-aware-reasoning-delta-normalization/tasks.md
sed -n '1,260p' ttmp/2026/03/27/GP-58-CHAT-STREAM-NORMALIZATION--extract-chat-streaming-from-go-openai-and-add-provider-aware-reasoning-delta-normalization/reference/01-diary.md
```

## Step 6: Add the chat streaming transport and normalization layer

I implemented the first code slice by adding a Geppetto-owned chat streaming transport and parser under `pkg/steps/ai/openai/`. This slice deliberately stopped short of changing the engine loop. The goal was to replace the risky boundary first, prove that it compiles, and lock down the transport semantics with isolated tests before mixing in event publication or turn mutation.

This step created a stable internal contract for the later engine refactor: the engine will no longer need to understand raw SSE framing or provider-specific field aliases directly. It will receive normalized stream events instead.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Start implementing the ticket in small, reviewable slices and commit them as they land.

**Inferred user intent:** The user wants a disciplined migration where the transport replacement can be reviewed independently from the event and turn-behavior changes.

**Commit (code):** pending at the time of writing this entry

### What I did

- Added `pkg/steps/ai/openai/chat_stream.go`
- Added `pkg/steps/ai/openai/chat_stream_test.go`
- Implemented:
  - chat streaming config resolution from API settings
  - direct `/chat/completions` POST transport
  - raw SSE frame reading
  - normalized stream event types
  - normalization for:
    - `delta.content`
    - `delta.reasoning`
    - `delta.reasoning_content`
    - fragmented `tool_calls`
    - usage fields including `reasoning_tokens`
- Ran `go test ./pkg/steps/ai/openai -count=1`

### Why

- The current bug is caused by losing provider-specific fields at the stream decode boundary.
- The fastest safe fix is to own that boundary directly.
- Testing this layer in isolation makes later engine changes much easier to debug.

### What worked

- The package compiled cleanly on the first pass.
- The standalone tests passed.
- The new normalizer can already represent both Together-style and DeepSeek-style reasoning deltas.

### What didn't work

- N/A

### What I learned

- The minimal viable internal contract is small. The engine does not need a huge transport model; it mainly needs normalized text, reasoning, tool calls, usage, and finish reason.

### What was tricky to build

- The sharp edge here was deciding what to normalize now versus later. I kept the transport contract intentionally narrow to avoid prematurely modeling provider-specific noise that the engine does not consume.
- Another subtle point was keeping the stream replacement independent from the existing request-builder logic, so the blast radius stays small.

### What warrants a second pair of eyes

- Review whether `resolveChatStreamConfig(...)` should also be reused by the old `MakeClient(...)` path later, or whether keeping them separate is clearer.
- Review whether the normalized tool-call representation should remain `go_openai.ToolCall` for now or move to a fully internal type in a later pass.

### What should be done in the future

- Refactor the engine loop to consume `chatStreamEvent`.
- Add engine-level tests that prove the reasoning events and reasoning blocks are emitted correctly.

### Code review instructions

- Start in `pkg/steps/ai/openai/chat_stream.go`.
- Verify the order:
  - config resolution
  - raw HTTP request
  - SSE frame parsing
  - payload normalization
- Then read `pkg/steps/ai/openai/chat_stream_test.go` to confirm the coverage of:
  - multiline SSE data
  - Together reasoning alias
  - DeepSeek reasoning-content fallback
  - tool-call normalization

### Technical details

- Files changed:
  - `pkg/steps/ai/openai/chat_stream.go`
  - `pkg/steps/ai/openai/chat_stream_test.go`
  - `ttmp/.../GP-58.../tasks.md`
  - `ttmp/.../GP-58.../reference/01-diary.md`

- Command used:

```bash
go test ./pkg/steps/ai/openai -count=1
```

## Step 7: Switch the OpenAI engine to the new stream client and add fixture-driven regressions

After the transport layer was stable, I rewired `OpenAIEngine.RunInference(...)` to consume normalized stream events instead of the `go-openai` streaming client. This was the main behavioral change in the ticket: the engine now receives text deltas, reasoning deltas, tool-call fragments, usage, and finish reasons from Geppetto-owned code rather than from a third-party typed stream struct.

I paired that refactor with fixture-driven regression tests. That kept the implementation honest and covered the exact provider behaviors the ticket is about: Together-style `delta.reasoning`, DeepSeek-style `delta.reasoning_content`, plain text-only chat streams, and fragmented tool calls with final usage.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Complete the main refactor, prove the reasoning events and turn persistence behavior, and keep the implementation traceable in the diary.

**Inferred user intent:** The user wants the ticket worked through to real code, not just scaffolding.

**Commit (code):** pending at the time of writing this entry

### What I did

- Updated `pkg/steps/ai/openai/engine_openai.go` to:
  - replace `CreateChatCompletionStream(...)` with the new custom stream client
  - accumulate normalized reasoning deltas
  - publish:
    - `reasoning-text-delta`
    - `partial-thinking`
    - `reasoning-text-done`
  - persist a reasoning block onto the output turn when reasoning text is present
  - preserve tool-call merge behavior and final usage metadata
  - add `thinking_text`, `saying_text`, and `reasoning_tokens` to final metadata extras
- Added fixture files under `pkg/steps/ai/openai/testdata/chat-stream/`
- Added `pkg/steps/ai/openai/engine_openai_test.go`
- Ran:
  - `go test ./pkg/steps/ai/openai -count=1`
  - `go test ./... -count=1`

### Why

- The transport layer by itself does not fix the user-visible behavior; the engine must publish the normalized reasoning data and persist it to turns.
- The regression fixtures ensure the new path remains provider-aware instead of drifting back toward a single-schema assumption.

### What worked

- The engine refactor compiled cleanly after the test harness was adjusted to inject the default HTTP client explicitly.
- The package test suite passed.
- The full repository test suite passed.
- The fixture-driven tests covered all the planned scenarios for this ticket.

### What didn't work

- The first engine test attempt failed because `EnsureHTTPClient(...)` built a fresh client instead of reusing the temporarily replaced `http.DefaultClient`, which caused real DNS lookup attempts against `example.test`.

The exact failure was:

```text
Post "https://example.test/v1/chat/completions": dial tcp: lookup example.test: no such host
```

- I fixed that by setting `ClientSettings{HTTPClient: http.DefaultClient}` in the test engine setup so the fixture transport was actually used.

### What I learned

- The test harness detail around `EnsureHTTPClient(...)` matters when the engine is moved from a wrapped SDK client to direct HTTP. The engine tests now explicitly pin the injected client instead of relying on the implicit default-client fast path.

### What was tricky to build

- The main sharp edge was preserving existing behavior while introducing reasoning-specific behavior. The engine needed to add a second live stream for thinking without accidentally mixing reasoning text into the assistant output buffer.
- Another tricky point was preserving tool-call merge behavior. The new normalizer emits tool-call fragments in a format the existing merger can still consume, which minimized the amount of tool-calling code that had to change in the same patch.

### What warrants a second pair of eyes

- Review the final block ordering to confirm that appending the reasoning block immediately before the assistant text block is the desired long-term invariant.
- Review the metadata extras contract for `thinking_text`, `saying_text`, and `reasoning_tokens` to ensure downstream consumers expect those fields on chat-completions final events.

### What should be done in the future

- Consider sanitizing or shrinking the fixture corpus if additional providers are added later so the test matrix stays easy to review.
- Evaluate whether the remaining `go_openai` request and tool structs in the chat package should also move to internal types in a follow-up ticket.

### Code review instructions

- Start with `pkg/steps/ai/openai/engine_openai.go` and look only at the streaming path changes first.
- Then read `pkg/steps/ai/openai/engine_openai_test.go` together with the fixture files under `pkg/steps/ai/openai/testdata/chat-stream/`.
- Validate that the four scenarios map directly to the tasks in `tasks.md`.

### Technical details

- Files changed:
  - `pkg/steps/ai/openai/engine_openai.go`
  - `pkg/steps/ai/openai/engine_openai_test.go`
  - `pkg/steps/ai/openai/testdata/chat-stream/together_reasoning.sse`
  - `pkg/steps/ai/openai/testdata/chat-stream/deepseek_reasoning_content.sse`
  - `pkg/steps/ai/openai/testdata/chat-stream/text_only.sse`
  - `pkg/steps/ai/openai/testdata/chat-stream/tool_calls_fragmented.sse`
  - `ttmp/.../GP-58.../tasks.md`
  - `ttmp/.../GP-58.../changelog.md`
  - `ttmp/.../GP-58.../reference/01-diary.md`

- Commands used:

```bash
gofmt -w pkg/steps/ai/openai/chat_stream.go pkg/steps/ai/openai/chat_stream_test.go pkg/steps/ai/openai/engine_openai.go pkg/steps/ai/openai/engine_openai_test.go
go test ./pkg/steps/ai/openai -count=1
go test ./... -count=1
```
