---
Title: Diary
Ticket: MO-002-FIX-UP-THINKING-MODELS
Status: active
Topics:
    - bug
    - geppetto
    - go
    - inference
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/reasoning_validator.go
      Note: Upstream reasoning model constraints (max tokens + sampling).
    - Path: ../../../../../../../go-go-mento/go/pkg/docs/05-webchat-middleware-ordering.md
      Note: Section-based ordering reference for unified design.
    - Path: ../../../../../../../pinocchio/pkg/ui/backend.go
      Note: Default chat UI handler drops thinking events.
    - Path: cmd/examples/openai-tools/main.go
      Note: Repro steps for GPT-5 runs.
    - Path: pkg/steps/ai/openai/helpers.go
      Note: Chat-mode request parameter gating for reasoning models.
    - Path: pkg/steps/ai/openai_responses/engine.go
      Note: Source of Responses SSE event emission.
    - Path: pkg/steps/ai/openai_responses/helpers.go
      Note: Responses request parameter gating (sampling params).
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/01-responses-thinking-stream-event-flow.md
      Note: Detailed analysis of Responses thinking stream event flow.
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/02-pinocchio-turns-and-responses-ordering.md
      Note: Detailed turn/block ordering analysis.
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/03-turn-mutation-across-pinocchio-and-moments.md
      Note: Cross-system turn mutation analysis.
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/04-moments-follow-up-plan.md
      Note: Deferred Moments migration plan.
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/design-doc/01-unified-conversation-handling-across-repos.md
      Note: Unified conversation handling design.
    - Path: ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/tasks.md
      Note: Task checklist for MO-002 execution.
ExternalSources: []
Summary: Track fixes for thinking-model parameter handling in chat vs responses engines.
LastUpdated: 2026-01-13T00:00:00Z
WhatFor: Capture investigation and code changes for GPT-5/o-series parameter gating.
WhenToUse: Use when validating reasoning model support and engine request building.
---









# Diary

## Goal

Document the investigation and fixes for GPT-5/o-series (thinking) model parameter handling across OpenAI chat and Responses engines.

## Step 1: Identify thinking-model parameter failures and start gating

I created the ticket workspace and traced the failures reported when running GPT-5. Chat-mode failed fast because go-openai rejects `max_tokens` for GPT-5, while Responses-mode rejected `temperature` for GPT-5. The plan is to explicitly detect reasoning-capable models (o1/o3/o4/gpt-5) and gate parameters accordingly: use `max_completion_tokens` for chat mode and omit sampling params for Responses.

I began implementing the first part by adding a reasoning-model detector in the OpenAI chat helper, using it to move `max_tokens` into `max_completion_tokens` and to reset sampling params to supported values. This keeps the request aligned with go-openai's validator and GPT-5 constraints while preserving standard behavior for non-reasoning models.

**Commit (code):** N/A (not committed)

### What I did
- Created ticket MO-002-FIX-UP-THINKING-MODELS and added a diary doc.
- Captured reported errors:
  - `this model is not supported MaxTokens, please use MaxCompletionTokens`
  - `Unsupported parameter: 'temperature' is not supported with this model.`
- Added a reasoning-model detector in `openai/helpers.go` and started gating max tokens + sampling parameters for chat-mode requests.

### Why
- GPT-5 and o-series models have stricter parameter requirements; chat-mode must use `max_completion_tokens` and omit unsupported sampling knobs.
- Responses-mode should omit sampling params for these models as well.

### What worked
- Identified that go-openai's reasoning validator flags GPT-5 when `max_tokens` is set.

### What didn't work
- `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5` failed with:
  - `this model is not supported MaxTokens, please use MaxCompletionTokens`
- `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses` failed with:
  - `responses api error: status=400 body=map[error:map[code:<nil> message:Unsupported parameter: 'temperature' is not supported with this model. param:temperature type:invalid_request_error]]`

### What I learned
- GPT-5 is treated as a reasoning model by go-openai and must follow the same parameter restrictions as o-series.

### What was tricky to build
- Ensuring we only change request parameters for reasoning-capable models without breaking defaults for standard chat models.

### What warrants a second pair of eyes
- Confirm the gating logic matches OpenAI's current parameter constraints for GPT-5/o-series across both chat and responses.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai/helpers.go` and review `isReasoningModel` and request parameter changes.
- Validate with `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses`.

### Technical details
- go-openai validator: `/home/manuel/go/pkg/mod/github.com/sashabaranov/go-openai@v1.41.1/reasoning_validator.go`
- Command that failed:
  - `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5`

## Step 2: Gate sampling params in Responses for GPT-5/o-series

I extended the Responses request builder to treat GPT-5 and o1/o3/o4 as reasoning models when deciding whether to include sampling parameters. This aligns the Responses path with the same model constraints that the chat engine now enforces, and directly addresses the 400 error about unsupported `temperature`.

This is a focused change to `openai_responses/helpers.go`, leaving the rest of the request shape intact. The aim is to keep tool and reasoning behavior unchanged while dropping parameters that the API rejects for these model families.

**Commit (code):** N/A (not committed)

### What I did
- Updated `allowSampling` to exclude `o1`, `o3`, `o4`, and `gpt-5` in the Responses helper.

### Why
- GPT-5 rejects `temperature` (and related sampling params) in the Responses API.

### What worked
- The request builder now omits sampling params for GPT-5/o-series models.

### What didn't work
- N/A (no new failures recorded yet).

### What I learned
- Responses and chat paths both need explicit reasoning-model gating; assumptions based on o3/o4 alone are insufficient.

### What was tricky to build
- Keeping model-family matching consistent between chat and Responses helpers without changing behavior for standard models.

### What warrants a second pair of eyes
- Confirm the model prefix list is complete and that omitting sampling params is correct for all GPT-5 variants.

### What should be done in the future
- N/A

### Code review instructions
- Review `allowSampling` in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go`.
- Validate with `go run ./cmd/examples/openai-tools test-openai-tools --mode server-tools --ai-engine gpt-5 --ai-api-type=openai-responses`.

### Technical details
- Responses helper: `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/steps/ai/openai_responses/helpers.go`

## Step 3: Validate GPT-5 Responses run via tmux

I ran the Responses-mode CLI example in tmux using GPT-5 and the server-tools mode to validate the parameter gating changes. The run completed successfully and produced tool calls, search results, and the final response, which suggests the previous `temperature` rejection is resolved. It did take about a minute to finish, so the prior “hang” likely reflected long tool activity rather than a deadlock.

This step focused on runtime behavior rather than code changes, confirming that the streaming loop can complete with GPT-5 and server tools enabled.

**Commit (code):** N/A (no code changes)

### What I did
- Ran `go run ./cmd/examples/openai-tools test-openai-tools --ai-api-type=openai-responses --ai-engine gpt-5 --mode server-tools --log-level info` inside tmux.
- Observed multiple reasoning summary phases, tool calls, and final output.

### Why
- Validate that GPT-5 Responses requests no longer fail on unsupported `temperature` and that the flow completes.

### What worked
- The run completed with a final response and tool results; total runtime ~1m17s.

### What didn't work
- N/A

### What I learned
- GPT-5 Responses with server tools can be slow but completes; the “hang” likely reflects long tool activity.

### What was tricky to build
- Distinguishing a slow Responses run from a genuine hang without explicit timeouts.

### What warrants a second pair of eyes
- Confirm if we should add a user-facing timeout or progress indicator for long server-tool runs.

### What should be done in the future
- N/A

### Code review instructions
- N/A (runtime validation only).

### Technical details
- tmux session: `gpt5-resp` (captured output shows completion).

## Step 4: Trace Responses thinking events through pinocchio chat UI

I reproduced the pinocchio chat flow with GPT-5-mini and Responses API while logging at DEBUG to a file. The logs showed the Responses engine emitting many thinking-summary deltas and info events before the assistant message stream, but the default pinocchio chat UI handler ignores those event types, so they never render in the timeline.

I then wrote a dedicated analysis doc that maps the event path from Responses SSE to the Bubble Tea UI, contrasts it with the agent tool-loop UI (which does render thinking events), and calls out the web chat forwarder dropping those events entirely. This explains why the frontend appears idle or "confused" when thinking streams are present.

**Commit (code):** N/A (docs and analysis only)

### What I did
- Ran `go run ./cmd/pinocchio code professional "hello" --ai-engine gpt-5-mini --chat --ai-api-type openai-responses --log-level DEBUG --log-file /tmp/pinocchio-gpt5-debug.log --with-caller` in tmux.
- Scanned `/tmp/pinocchio-gpt5-debug.log` for thinking and reasoning events.
- Read UI forwarders and response engine code to map the event flow.
- Wrote the analysis doc at `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/01-responses-thinking-stream-event-flow.md`.

### Why
- We needed to explain the mismatch between Responses thinking streams and what the pinocchio chat UI renders.

### What worked
- Logs confirmed the engine emits `EventInfo` and `EventThinkingPartial` events that reach the UI handler.
- The analysis doc captures where the events are dropped and why the UI appears idle.

### What didn't work
- I could not reproduce a user-visible error message in the log file; the debug log contained no error lines.

### What I learned
- The default pinocchio chat UI handler only renders partial completion events and ignores thinking/info events.
- The agent tool-loop UI already has the logic to render thinking streams, so it is a good reference for the fix.

### What was tricky to build
- Tracing handler selection in `runtime.NewChatBuilder()` to confirm the default event forwarder in chat mode.

### What warrants a second pair of eyes
- Validate the hypothesis that the reported transient error lives outside the `--log-file` output and confirm any additional event types that should be surfaced in the UI.

### What should be done in the future
- If we decide to fix the UX, add thinking/info handling to the default chat UI forwarder and web chat mapping (with a visibility toggle if needed).

### Code review instructions
- Start with `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/01-responses-thinking-stream-event-flow.md` for the event path map.
- Spot-check `pinocchio/pkg/ui/backend.go` and `geppetto/pkg/steps/ai/openai_responses/engine.go` to see where thinking events are emitted and dropped.

### Technical details
- Debug log: `/tmp/pinocchio-gpt5-debug.log`
- tmux session: `pinocchio-gpt5`

## Step 5: Render thinking events in the default pinocchio chat UI

I extended the default pinocchio chat UI event forwarder to surface Responses "thinking" streams as a separate timeline entity. This mirrors the behavior already present in the agent tool-loop UI and makes the GPT-5 reasoning summary deltas visible during chat runs.

The changes are limited to the UI event handler: thinking-started/ended info events now open/close a "thinking" entity, and EventThinkingPartial updates its text. This should prevent the UI from appearing idle while the Responses engine streams reasoning summary deltas.

**Commit (code):** 7b38883 — "Handle thinking events in chat UI"

### What I did
- Updated `pinocchio/pkg/ui/backend.go` to handle `EventInfo` thinking-started/ended and `EventThinkingPartial`.
- Committed the pinocchio UI change.

### Why
- The default chat UI was ignoring Responses reasoning summary events, making the UI look idle even though events were streaming.

### What worked
- The forwarder now creates and updates a dedicated thinking timeline entity during Responses runs.

### What didn't work
- N/A

### What I learned
- The default chat UI path lacked the same thinking-stream handling that the agent tool-loop UI already had.

### What was tricky to build
- Keeping the handling minimal without introducing duplicate entity creation across event types.

### What warrants a second pair of eyes
- Confirm the event ordering guarantees (thinking-started before partial deltas) across providers.

### What should be done in the future
- N/A

### Code review instructions
- Review `pinocchio/pkg/ui/backend.go` for the new thinking event cases.

### Technical details
- Commit: `7b38883`

## Step 6: Add web chat semantic mappings for thinking events

I extended the web chat semantic forwarder to emit dedicated thinking start/delta/final frames. This mirrors the new chat UI behavior and ensures the browser UI can observe reasoning summary streams coming from the Responses API.

The mapping uses a separate `:thinking` entity id suffix and treats the `reasoning-summary` info event as a delta update with cumulative text, so UIs can render the final summary even if they do not process the raw SSE events.

**Commit (code):** df87f75 — "Map thinking events for web chat"

### What I did
- Updated `pinocchio/pkg/webchat/forwarder.go` to map `EventThinkingPartial` and `EventInfo` thinking events into SEM frames.
- Committed the web chat forwarder change.

### Why
- Web chat UIs currently drop Responses reasoning summary events and never expose the thinking stream.

### What worked
- The forwarder now emits `llm.thinking.*` frames suitable for UI consumption.

### What didn't work
- N/A

### What I learned
- The web chat forwarder only had LLM text and tool events; it needed explicit thinking events to surface reasoning summary streams.

### What was tricky to build
- Choosing event naming that stays consistent with existing `llm.*` semantics while avoiding ID collisions with assistant output.

### What warrants a second pair of eyes
- Confirm the client UI can safely ignore or consume `llm.thinking.*` events without breaking existing flows.

### What should be done in the future
- N/A

### Code review instructions
- Review `pinocchio/pkg/webchat/forwarder.go` for the new `EventInfo` and `EventThinkingPartial` cases.

### Technical details
- Commit: `df87f75`

## Step 7: Re-run GPT-5 chat with stderr capture

I reran the GPT-5 chat command in tmux with DEBUG logging and explicit stderr capture to chase the fast-scrolling error. The first attempt failed because I ran it from the workspace root (wrong path), but the second run from the pinocchio module succeeded and displayed the thinking stream in the UI. The stderr log only contained the "Logging to file" line, so the previously observed error was not reproduced.

I updated the analysis doc to record the stderr capture outcome and to reflect the fact that thinking events now render in the default chat UI and web chat forwarder.

**Commit (code):** N/A (docs only)

### What I did
- Ran `go run ./cmd/pinocchio ... --log-file /tmp/pinocchio-gpt5-debug-3.log --with-caller 2> /tmp/pinocchio-gpt5-stderr-3.log` inside tmux.
- Captured tmux output confirming the thinking stream rendered.
- Updated the analysis doc with stderr capture results and current behavior.

### Why
- Confirm whether the transient error was coming from stderr and verify the new UI handling in a real run.

### What worked
- The chat UI rendered the thinking stream and completed normally.
- Stderr contained only the log-file initialization line; no errors reproduced.

### What didn't work
- First attempt failed with `stat /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/cmd/pinocchio: directory not found` because I ran from the wrong directory.

### What I learned
- The earlier "error scroll" is not reproduced when running from the correct module path and capturing stderr.

### What was tricky to build
- Managing tmux lifecycle for an interactive chat session that stays open until explicitly exited.

### What warrants a second pair of eyes
- Confirm whether any UI errors are printed directly to the terminal outside the log file during other prompts or tool calls.

### What should be done in the future
- N/A

### Code review instructions
- Review the updated analysis doc at `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/01-responses-thinking-stream-event-flow.md`.

### Technical details
- Debug log: `/tmp/pinocchio-gpt5-debug-3.log`
- Stderr log: `/tmp/pinocchio-gpt5-stderr-3.log`
- tmux session: `pinocchio-gpt5-debug`

## Step 8: Document turn construction and Responses ordering

I wrote a deep-dive analysis on how pinocchio chat and webchat construct `turns.Turn` blocks and how those blocks become Responses API input items. The doc maps out the CLI chat seed/flatten flow, the webchat `conv.Turn` flow, and the reasoning ordering rules enforced by `buildInputItemsFromTurn`, then ties those to the observed 400 error and the hanging "Generating" UI state.

The analysis is written as a reference guide with diagrams, pseudocode, and explicit file/symbol references so we can reason about block order and where to add fixes.

**Commit (code):** N/A (docs only)

### What I did
- Added a detailed analysis doc at `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/02-pinocchio-turns-and-responses-ordering.md`.
- Related key files covering turn construction, chat seeding, webchat flow, and Responses input conversion.

### Why
- The Responses API validates reasoning ordering strictly; we need a clear map of how our Turns and Blocks are built to fix the error path.

### What worked
- The document captures the full pipeline and highlights ordering constraints and history duplication risks.

### What didn't work
- N/A

### What I learned
- The CLI chat backend flattens all prior Turns, which can duplicate blocks and complicate reasoning adjacency rules.
- The Responses engine returns HTTP 400 without emitting `EventError`, leaving the UI in a streaming state.

### What was tricky to build
- Reconciling the CLI chat history flattening with the Responses input ordering expectations.

### What warrants a second pair of eyes
- Validate the reasoning adjacency assumptions against real request logs and debug taps.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/02-pinocchio-turns-and-responses-ordering.md`.

### Technical details
- N/A

## Step 9: Upload turn-ordering analysis to reMarkable

I uploaded the new turn/block ordering analysis to the reMarkable device using the ticket-aware upload workflow. The first upload attempt timed out in the CLI harness, but rerunning with a longer timeout succeeded and confirmed the PDF was placed under the mirrored ticket directory on-device.

**Commit (code):** N/A (docs only)

### What I did
- Ran a dry-run to confirm the destination path and PDF name.
- Uploaded `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/02-pinocchio-turns-and-responses-ordering.md` to reMarkable.

### Why
- The user requested the analysis document be delivered to the tablet.

### What worked
- Upload succeeded on the second attempt and reported the final remote path.

### What didn't work
- Initial upload attempt timed out in the tool harness:
  - `python3 /home/manuel/.local/bin/remarkable_upload.py --ticket-dir ... --mirror-ticket-structure ...`

### What I learned
- The upload can exceed the default command timeout; rerun with a longer timeout when needed.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- N/A (upload only)

### Technical details
- Remote path: `ai/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/02-pinocchio-turns-and-responses-ordering.pdf`

## Step 10: Map turn mutation across pinocchio and moments webchat

I produced a new analysis document that explains how Turns are mutated across inference runs in pinocchio CLI chat, pinocchio webchat, and moments backend webchat. The doc calls out the `reduceHistory` origin and usage, shows where user blocks and system prompts are appended, and highlights ordering and compression middlewares that can reorder or drop blocks.

This provides the basis for diagnosing block adjacency problems under the Responses API, especially when duplicated system/reasoning blocks or reordered sections create invalid sequences.

**Commit (code):** N/A (docs only)

### What I did
- Wrote the analysis document at `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/03-turn-mutation-across-pinocchio-and-moments.md`.
- Linked the key pinocchio and moments backend sources that mutate Turns across runs.

### Why
- We need a precise, cross-system map of how Turns evolve to fix strict Responses ordering errors on later prompts.

### What worked
- The doc captures the mutation paths and identifies where duplication, reordering, and compression occur.

### What didn't work
- N/A

### What I learned
- `reduceHistory` is local to pinocchio’s chat backend and only used by `EngineBackend.Start`.
- Moments webchat applies both idempotent system prompts and ordering/compression middlewares that can reorder or drop blocks.

### What was tricky to build
- Reconciling how the different webchat stacks mutate `Turn.Data` and reorder blocks before inference.

### What warrants a second pair of eyes
- Validate the ordering middleware’s section boundaries against Responses reasoning adjacency requirements.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/03-turn-mutation-across-pinocchio-and-moments.md`.

### Technical details
- N/A

## Step 11: Draft unified conversation handling design

I drafted a design document that unifies conversation handling across geppetto, pinocchio, and moments. It explains why `reduceHistory` is a bug, proposes a shared `ConversationState` API in geppetto, and outlines migration steps for CLI chat, pinocchio webchat, and moments webchat. The doc includes diagrams, pseudocode, and explicit file references.

**Commit (code):** N/A (docs only)

### What I did
- Wrote the design doc at `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/design-doc/01-unified-conversation-handling-across-repos.md`.
- Related the primary pinocchio and moments source files that drive turn mutation and ordering.

### Why
- We need a single, consistent conversation model to avoid Responses ordering failures and duplicated blocks.

### What worked
- The design articulates a shared ConversationState and a staged migration plan for each repo.

### What didn't work
- N/A

### What I learned
- The current divergence across repos is mostly a lack of shared state semantics, not engine behavior.

### What was tricky to build
- Balancing a unified abstraction with repo-specific middleware behavior (ordering, compression, prompt injection).

### What warrants a second pair of eyes
- Validate the proposed snapshot rules against non-Responses providers to ensure no regressions.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/design-doc/01-unified-conversation-handling-across-repos.md`.

### Technical details
- N/A

## Step 12: Incorporate go-go-mento webchat patterns into the unified design

I reviewed the cleaned up `go-go-mento` webchat stack to extract patterns that inform the unified conversation design. The update focuses on lifecycle ownership, engine config signatures, run guards, snapshot hooks, and section-based ordering as first-class concepts.

I then amended the design doc to include these insights and to extend the migration plan with a ConversationManager layer and ordering semantics aligned to the go-go-mento section model.

**Commit (code):** N/A (docs only)

### What I did
- Reviewed `go-go-mento` webchat files and the ordering/middleware doc.
- Updated the unified conversation handling design doc with new insights and an amended migration plan.

### Why
- The go-go-mento webchat implementation is a cleaner reference that already solved many lifecycle and ordering problems we need to standardize.

### What worked
- The design doc now includes concrete patterns (manager lifecycle, signature-based recomposition, ordering sections, snapshot hooks) drawn from a working system.

### What didn't work
- N/A

### What I learned
- go-go-mento treats ordering as a deterministic section-based transformation rather than a side effect of middleware order, which is a better fit for Responses strictness.

### What was tricky to build
- Mapping go-go-mento concepts onto the proposed shared ConversationState without expanding scope beyond the unified design goals.

### What warrants a second pair of eyes
- Validate that the section-based ordering contract aligns with all existing pinocchio and moments block metadata expectations.

### What should be done in the future
- N/A

### Code review instructions
- Review `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/design-doc/01-unified-conversation-handling-across-repos.md` for the go-go-mento additions.

### Technical details
- Reference: `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/go-go-mento/go/pkg/docs/05-webchat-middleware-ordering.md`

## Step 13: Add task plan and Moments follow-up document

I added a dedicated Moments follow-up plan doc and populated it with a phased migration outline, risks, and validation notes. This captures the deferred work we should do once the geppetto/pinocchio migration stabilizes.

I also created a concrete task list for MO-002 covering the conversation package implementation, tests, and the pinocchio migrations. This gives us a clear execution checklist while keeping Moments explicitly deferred.

**Commit (code):** N/A (docs only)

### What I did
- Created `analysis/04-moments-follow-up-plan.md` with the deferred Moments migration plan.
- Added MO-002 tasks for conversation package work, tests, and pinocchio migrations.

### Why
- We need a persistent plan for Moments work without starting the migration yet.
- The ticket needed actionable tasks to drive the next steps.

### What worked
- The follow-up doc provides an explicit roadmap and risks for the Moments migration.
- The task list now reflects the near-term implementation sequence.

### What didn't work
- N/A

### What I learned
- Keeping the Moments plan separate makes it easier to update once the shared conversation package exists.

### What was tricky to build
- Keeping the task list tight while preserving a separate track for deferred Moments work.

### What warrants a second pair of eyes
- Review the task ordering to ensure it matches dependencies between conversation package work and pinocchio migrations.

### What should be done in the future
- N/A

### Code review instructions
- Review `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/analysis/04-moments-follow-up-plan.md`.
- Review `/home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/13/MO-002-FIX-UP-THINKING-MODELS--fix-thinking-model-parameter-handling/tasks.md`.

### Technical details
- N/A
