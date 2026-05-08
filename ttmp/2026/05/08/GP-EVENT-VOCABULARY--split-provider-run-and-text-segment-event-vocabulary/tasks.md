---
Title: Tasks
Ticket: GP-EVENT-VOCABULARY
Status: active
Topics:
  - geppetto
  - pinocchio
  - streaming
  - observability
  - events
DocType: tasks
Intent: short-term
Owners:
  - manuel
Summary: Phased hard-cutover task list for replacing overloaded inference/text events with explicit run, provider-call, text-segment, reasoning, and tool lifecycle events plus typed correlation IDs.
LastUpdated: 2026-05-08T07:20:00-04:00
---

# Tasks

This is the implementation checklist for the hard cutover. It intentionally does **not** preserve legacy runtime aliases for `EventFinal`, `EventPartialCompletion`, `ChatInferenceStarted`, `ChatTokensDelta`, or `ChatInferenceFinished`.

## Done

- [x] Create `GP-EVENT-VOCABULARY` ticket workspace.
- [x] Add primary design document and investigation diary.
- [x] Capture line-numbered source evidence for Geppetto events, provider engines, observability, Pinocchio runtime sink, protobufs, plugins, and SQLite reconcile code.
- [x] Write intern-facing design and implementation guide for a clean provider/run/text segment event vocabulary.
- [x] Include typed correlation-ID design to avoid relying on `metadata.Extra` heuristics.
- [x] Validate ticket with `docmgr doctor`.
- [x] Upload design bundle to reMarkable and verify remote listing.
- [x] Revise primary design guide to assume a hard cutover with no legacy runtime aliases.
- [x] Upload hard-cutover revision as a new reMarkable copy.
- [x] Replace the short TODO list with this detailed phased migration checklist.

## Phase 0 — Workspace, branch, and baseline gates

Goal: start from a known-good multi-repo workspace and make the hard cutover measurable.

- [ ] Create coordinated work branches in all touched repos:
  - [ ] `geppetto`
  - [ ] `pinocchio`
  - [ ] `2026-03-16--gec-rag` / CoinVault
  - [ ] optional `sessionstream` only if transport schemas or event wrappers require changes.
- [x] Confirm `go.work` points at local repos used by CoinVault:
  - [x] `./geppetto`
  - [x] `./pinocchio`
  - [x] `./sessionstream`
  - [x] `./2026-03-16--gec-rag`
- [x] Record baseline validation before changing code:
  - [x] `cd geppetto && go test ./pkg/steps/ai/... -count=1`
  - [x] `cd pinocchio && go test ./pkg/chatapp/... -count=1`
  - [x] `cd 2026-03-16--gec-rag && go test ./internal/webchat ./cmd/coinvault/cmds -count=1`
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run typecheck`
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run test:unit -- src/ws/parsing.test.ts src/ws/wsManager.test.ts`
- [x] Inventory all legacy symbol references and save the output under this ticket's `sources/` or `various/` directory:
  - [x] `rg "EventFinal|EventPartialCompletion|EventTypeFinal|EventTypePartialCompletion" geppetto`
  - [x] `rg "ChatInferenceStarted|ChatInferenceFinished|ChatTokensDelta|ChatInferenceStopped" pinocchio 2026-03-16--gec-rag`
  - [x] `rg "response_id|correlation_key|metadata.Extra|choice_index|tool_call_index" geppetto pinocchio 2026-03-16--gec-rag`
- [ ] Decide whether to do the implementation as one large branch or chained commits; even with separate commits, do not merge until the entire cutover compiles end-to-end.
- [x] Protect unrelated working-tree changes:
  - [x] keep existing CoinVault `SQLITE-TRACE-VERBS` modifications out of this migration unless intentionally included;
  - [x] keep browser-run artifacts untracked unless explicitly required.

## Phase 1 — Geppetto canonical event and correlation contracts

Goal: define the new vocabulary and typed identity envelope before provider adapters are migrated.

- [x] Add `pkg/events/correlation.go` with `events.Correlation`.
- [ ] Include these runtime-scope fields:
  - [x] `SessionID`
  - [x] `RunID`
  - [x] `InferenceID`
  - [x] `TurnID`
- [ ] Include these provider-call fields:
  - [x] `ProviderCallID`
  - [x] `ProviderCallIndex`
  - [x] `Provider`
  - [x] `Model`
  - [x] `ResponseID`
- [ ] Include these provider item/block fields:
  - [x] `ItemID`
  - [x] `OutputIndex`
  - [x] `SummaryIndex`
  - [x] `ChoiceIndex`
  - [x] `ContentBlockIndex`
- [ ] Include these transcript segment fields:
  - [x] `SegmentID`
  - [x] `SegmentIndex`
  - [x] `SegmentType`
  - [x] `StreamKind`
- [ ] Include these tool fields:
  - [x] `ToolCallID`
  - [x] `ToolCallIndex`
- [ ] Include these join fields:
  - [x] `CorrelationKey`
  - [x] `ParentCorrelationKey`
- [x] Add a `CorrelatedEvent` interface or equivalent helper API so consumers can retrieve typed correlation without reading `metadata.Extra`.
- [x] Add run lifecycle events:
  - [x] `EventRunStarted`
  - [x] `EventRunFinished`
  - [x] `EventRunStopped`
  - [x] `EventRunFailed`
- [x] Add provider-call lifecycle events:
  - [x] `EventProviderCallStarted`
  - [x] `EventProviderCallMetadataUpdated`
  - [x] `EventProviderCallFinished`
- [x] Add text segment events:
  - [x] `EventTextSegmentStarted`
  - [x] `EventTextDelta`
  - [x] `EventTextSegmentFinished`
  - [ ] optional `EventTextSegmentStopped` if interruption semantics need segment-level stop.
- [x] Add reasoning segment events:
  - [x] `EventReasoningSegmentStarted`
  - [x] `EventReasoningDelta`
  - [x] `EventReasoningSegmentFinished`
- [x] Add tool lifecycle events:
  - [x] `EventToolCallStarted`
  - [x] `EventToolCallArgumentsDelta`
  - [x] `EventToolCallRequested`
  - [x] `EventToolExecutionStarted`
  - [x] `EventToolResultReady`
  - [x] `EventToolCallFinished` if host-side completion needs a distinct event.
- [x] Remove or make inaccessible the legacy constructors/types from active provider code:
  - [x] `NewStartEvent`
  - [x] `NewPartialCompletionEvent`
  - [x] `NewFinalEvent`
  - [x] `NewThinkingPartialEvent`
  - [x] legacy `EventToolCall` if replaced by `EventToolCallRequested`.
- [x] Update `NewEventFromJson` / event decoding to use only canonical event type strings.
  - [x] Add `NewEventFromJson` decoding support for canonical event type strings while legacy events still exist during staged implementation.
  - [x] Remove legacy event decoding branches after provider/runtime cutover.
- [x] Add canonical validation guard for typed-correlation invariants (`ValidateCanonicalEvent`).
- [x] Update event printer/structured sinks so canonical events render clearly.
- [x] Add Geppetto event serialization tests:
  - [x] every event round-trips through JSON;
  - [x] every event preserves `Correlation`;
  - [x] provider-call events do not have text payloads in the new event structs;
  - [x] text segment events require `SegmentID` and `CorrelationKey` via `ValidateCanonicalEvent`.
- [x] Run `go test ./pkg/events/... -count=1`.

## Phase 2 — Geppetto correlation builders and invariants

Goal: centralize identity construction so providers do not each invent ad hoc metadata maps.

- [x] Add a provider-call correlation builder package/helper, e.g. `pkg/events/correlationbuilder` or provider-local helpers with shared tests.
- [x] Implement provider-call ID generation:
  - [x] stable across all events in one provider API call;
  - [x] independent of provider response ID because some providers reveal response IDs late;
  - [x] includes `InferenceID` or `RunID` plus `ProviderCallIndex`.
- [x] Implement segment ID generation:
  - [x] stable across start/delta/finish for the segment;
  - [x] includes provider call identity and stream object identity;
  - [x] does not depend on rendered Pinocchio message IDs.
- [ ] Implement normalized correlation-key builders for:
  - [x] Claude provider calls;
  - [x] Claude text blocks;
  - [x] Claude tool-use blocks;
  - [x] OpenAI Responses provider calls;
  - [x] OpenAI Responses output items and summaries;
  - [x] OpenAI-compatible Chat Completions content streams;
  - [x] OpenAI-compatible Chat Completions reasoning streams;
  - [x] OpenAI-compatible Chat Completions tool streams by ID or index.
- [ ] Add invariant helpers/tests:
  - [x] provider-call events must have `ProviderCallID` and provider-call `CorrelationKey`;
  - [x] text events must have `SegmentID`, `SegmentType=text`, and text `CorrelationKey`;
  - [x] tool events must have `ToolCallID` when provider supplies it, or `ToolCallIndex` fallback;
  - [x] no canonical event requires `metadata.Extra` for routing.
- [x] Decide whether `RunID` aliases `InferenceID` initially or becomes a new generated ID.
- [x] Document the decision in the implementation diary; design guide unchanged because this preserves the optional RunID plan.

## Phase 3 — Migrate Claude/Anthropic first

Goal: use Claude's clean envelope/content-block distinction as the first proof of the new vocabulary.

- [x] Update `pkg/steps/ai/claude/content-block-merger.go` to maintain provider-call state:
  - [x] active `ProviderCallID`;
  - [x] `ProviderCallIndex`;
  - [x] provider `response_id` from `message_start.message.id`;
  - [x] current stop reason;
  - [x] current stop sequence;
  - [x] latest usage;
  - [x] duration.
- [x] Emit `EventProviderCallStarted` for `message_start`.
- [x] Emit `EventProviderCallMetadataUpdated` for `message_delta`.
- [x] Emit `EventProviderCallFinished` for `message_stop`.
- [x] Ensure `message_delta` emits no text event.
- [x] Ensure `message_stop` emits no text event.
- [x] Update text block handling:
  - [x] `content_block_start type=text` emits `EventTextSegmentStarted`;
  - [x] `content_block_delta type=text_delta` emits `EventTextDelta`;
  - [x] `content_block_stop type=text` emits `EventTextSegmentFinished`.
- [x] Update tool-use block handling:
  - [x] `content_block_start type=tool_use` emits `EventToolCallStarted`;
  - [x] `content_block_delta type=input_json_delta` emits `EventToolCallArgumentsDelta`;
  - [x] `content_block_stop type=tool_use` emits `EventToolCallRequested`.
- [x] Ensure Claude correlation fields are populated on every event:
  - [x] `provider=claude`;
  - [x] `model`;
  - [x] `provider_call_id`;
  - [x] `provider_call_index`;
  - [x] `response_id` when known;
  - [x] `content_block_index` for block events;
  - [x] `segment_id` for text/reasoning segments;
  - [x] `tool_call_id` for tool-use blocks;
  - [x] `correlation_key`;
  - [x] `parent_correlation_key`.
- [x] Update `pkg/steps/ai/claude/engine_claude.go` to publish canonical events only.
- [x] Preserve canonical inference-result persistence from provider-call metadata:
  - [x] tool-use call persists/exports `stop_reason=tool_use`;
  - [x] final call persists/exports `stop_reason=end_turn` or provider equivalent.
- [x] Update Claude tests:
  - [x] text-only stream;
  - [x] text -> tool_use -> `message_stop`;
  - [x] text -> tool_use -> next provider call -> text -> end_turn;
  - [x] metadata-only `message_delta` retains usage and stop reason;
  - [x] no `EventTextSegmentFinished` is emitted from `message_stop`.
- [x] Run `go test ./pkg/steps/ai/claude -count=1`.

## Phase 4 — Migrate OpenAI Responses

Goal: map Responses item lifecycle to text/reasoning/tool segments and provider response lifecycle to provider-call events.

- [x] Update `pkg/steps/ai/openai_responses/streaming.go` to create a provider-call correlation before processing stream events.
- [x] Emit `EventProviderCallStarted` at response creation / first provider event.
- [x] Preserve provider `response_id` once known and update subsequent correlation.
- [x] Map text items:
  - [x] `response.output_item.added type=message` emits `EventTextSegmentStarted` when it represents text output;
  - [x] `response.output_text.delta` emits `EventTextDelta`;
  - [x] `response.output_item.done type=message` emits `EventTextSegmentFinished`.
- [x] Map reasoning items:
  - [x] reasoning item start emits `EventReasoningSegmentStarted`;
  - [x] reasoning text/summary deltas emit `EventReasoningDelta`;
  - [x] reasoning item/summary completion emits `EventReasoningSegmentFinished`.
- [x] Map function calls:
  - [x] function-call argument deltas emit `EventToolCallArgumentsDelta` when available;
  - [x] completed function-call item emits `EventToolCallRequested`.
- [x] Map `response.completed` to `EventProviderCallFinished` only.
- [x] Ensure `response.completed` never emits text segment finished unless an actual text item remains unclosed due to provider stream shape; if such a fallback is necessary, make it explicit and test it as text-item closure, not provider-final closure.
- [x] Update `pkg/steps/ai/openai_responses/observability.go` to fill typed correlation rather than map-only provider data.
- [x] Update Responses tests for:
  - [x] text-only response;
  - [x] reasoning response;
  - [x] function-call response;
  - [x] text + function-call response;
  - [x] `response.completed` no duplicate text finalization;
  - [x] correlation key stability by item ID and output index.
- [x] Run `go test ./pkg/steps/ai/openai_responses -count=1`.

## Phase 5 — Migrate OpenAI-compatible Chat Completions

Goal: synthesize explicit segment/provider lifecycles for the less-structured Chat Completions stream.

- [x] Update `pkg/steps/ai/openai/engine_openai.go` to create a provider-call correlation before reading chunks.
- [x] Emit `EventProviderCallStarted` on first chunk or before stream processing.
- [x] Track `response_id` as soon as chunks expose it.
- [x] Track `choice_index` for all content/reasoning/tool streams.
- [x] Text stream handling:
  - [x] on first non-empty `delta.content` for a choice, emit `EventTextSegmentStarted`;
  - [x] for each content delta, emit `EventTextDelta`;
  - [x] on `finish_reason=stop`, emit `EventTextSegmentFinished` for active text segments only.
- [x] Reasoning stream handling:
  - [x] on first non-empty reasoning delta for a choice, emit `EventReasoningSegmentStarted`;
  - [x] emit `EventReasoningDelta` for each reasoning delta;
  - [x] emit `EventReasoningSegmentFinished` when the choice/provider call ends.
- [x] Tool stream handling:
  - [x] preserve tool-call IDs across argument-only deltas;
  - [x] on first tool-call delta for `(choice_index, tool_call_index)`, emit `EventToolCallStarted`;
  - [x] emit `EventToolCallArgumentsDelta` for argument deltas;
  - [x] on `finish_reason=tool_calls`, emit `EventToolCallRequested` for each completed tool call.
- [x] Provider-call finalization:
  - [x] emit `EventProviderCallMetadataUpdated` when usage/finish reason metadata arrives;
  - [x] emit `EventProviderCallFinished` at EOF/final chunk;
  - [x] never emit text segment finished from EOF unless an active text segment exists and is being explicitly closed.
- [x] Update `pkg/steps/ai/openai/observability.go` to make existing normalized keys feed typed `Correlation`.
- [x] Update Chat Completions tests for:
  - [x] text-only answer;
  - [x] reasoning stream;
  - [x] tool-call stream with ID on first delta only;
  - [ ] parallel tool calls;
  - [ ] `finish_reason=tool_calls` does not produce duplicate text;
  - [ ] correlation keys match documented shape.
- [x] Run `go test ./pkg/steps/ai/openai -count=1`.

## Phase 5B — Migrate Gemini and local tool execution events

Goal: cover the remaining active Geppetto runtime paths before deleting legacy text/tool events.

- [x] Add dedicated Gemini migration analysis document:
  - [x] `analysis/01-gemini-canonical-event-migration-analysis.md`.
- [x] Capture source evidence:
  - [x] `sources/geppetto-gemini-engine-legacy-events.lines.txt`;
  - [x] `sources/geppetto-tool-executor-legacy-events.lines.txt`;
  - [x] `various/gemini-and-tool-executor-legacy-event-inventory.txt`.
- [x] Update `pkg/steps/ai/gemini/engine_gemini.go` to create a provider-call correlation before `GenerateContentStream`.
- [x] Emit `EventProviderCallStarted` before processing the Gemini stream.
- [x] Emit `EventProviderCallMetadataUpdated` when Gemini usage or finish reason metadata is observed.
- [x] Emit `EventProviderCallFinished` at stream EOF with stop reason, finish class, usage, duration, and `has_tool_calls`.
- [x] Text stream handling:
  - [x] emit `EventTextSegmentStarted` only when the first non-empty `genai.Text` arrives;
  - [x] emit `EventTextDelta` for text deltas with monotonically increasing sequence;
  - [x] emit `EventTextSegmentFinished` only for an actually-started text segment;
  - [x] never use provider EOF/final metadata to manufacture a text segment.
- [x] Function-call handling:
  - [x] preserve existing generated Gemini tool-call IDs when provider-native IDs are absent;
  - [x] emit `EventToolCallStarted` for each observed `genai.FunctionCall`;
  - [x] emit `EventToolCallRequested` with JSON input when the complete function call is available;
  - [x] skip `EventToolCallArgumentsDelta` unless Gemini exposes partial argument chunks;
  - [x] preserve `turns.NewToolCallBlock` output behavior.
- [x] Preserve current return-turn and inference-result persistence behavior.
- [x] Update/add Gemini tests:
  - [ ] text-only stream fixture;
  - [ ] function-call-only stream fixture;
  - [ ] mixed text and function-call stream fixture;
  - [ ] empty/safety response fixture;
  - [ ] stream error path fixture;
  - [x] no legacy start/partial/final/tool-call constructors remain in Gemini engine source;
  - [x] Gemini canonical correlation helpers validate provider-call/text/tool events.
- [x] Update `pkg/inference/tools/base_executor.go`:
  - [x] replace `EventToolCallExecute` with `EventToolExecutionStarted`;
  - [x] replace `EventToolCallExecutionResult` with `EventToolResultReady` plus `EventToolCallFinished`;
  - [x] carry typed tool correlation via context where available, or build minimal execution-only tool correlation without using `metadata.Extra` routing.
- [x] Update tool-executor tests for canonical execution/result events.
- [x] Update JS event collector payload encoding/test expectations for canonical tool execution/result events.
- [x] Save validation output:
  - [x] `various/gemini-canonical-migration-validation.log`.
- [x] Run `go test ./pkg/steps/ai/gemini ./pkg/inference/tools -count=1`.
- [x] Run `go test ./pkg/steps/ai/... -count=1`.
- [x] Run `go test ./pkg/inference/... -count=1`.
- [x] Run `go test ./pkg/js/modules/geppetto -count=1`.
- [x] Run `go test ./... -count=1`.

## Phase 6 — Update Geppetto inference result and observability output

Goal: expose provider-call results and canonical segment records for SQLite/debugging without guessing from emitted transcript events.

- [x] Extend `pkg/observability/observer.go` or add a typed observability record for provider-call inference results.
- [x] Add observability stages or record kinds for:
  - [x] provider event received;
  - [x] canonical event emitted;
  - [x] provider-call result finalized;
  - [x] segment started/updated/finished if needed for SQLite.
- [x] Ensure provider-call result records include:
  - [x] all run/provider-call IDs;
  - [x] provider/model/response ID;
  - [x] stop reason;
  - [x] finish class;
  - [x] usage;
  - [x] duration;
  - [x] has tool calls;
  - [x] `correlation_key`.
- [x] Ensure segment records include:
  - [x] `segment_id`;
  - [x] `segment_type`;
  - [x] stream kind;
  - [x] provider-call IDs;
  - [x] provider-native IDs/indexes;
  - [x] text length/status;
  - [x] start/finish record IDs if known.
- [x] Update inference result builder if needed so `tool_use` maps to `tool_calls_pending` for provider-call result rows.
- [x] Add tests for inference result and segment observability records.
- [x] Run `go test ./pkg/observability ./pkg/inference/engine ./pkg/steps/ai/... -count=1`.

## Phase 7 — Replace Pinocchio protobuf contract

Goal: make `CorrelationInfo` canonical and remove old event payload names.

- [x] Update `../pinocchio/proto/pinocchio/chatapp/v1/chat.proto`.
- [x] Add `CorrelationInfo` message with all required correlation fields.
- [x] Add canonical run payloads:
  - [x] `ChatRunStarted`
  - [x] `ChatRunFinished`
  - [x] `ChatRunStopped`
  - [x] `ChatRunFailed`
- [x] Add canonical provider-call payloads:
  - [x] `ChatProviderCallStarted`
  - [x] `ChatProviderCallMetadataUpdated`
  - [x] `ChatProviderCallFinished`
- [x] Add canonical text payloads:
  - [x] `ChatTextSegmentStarted`
  - [x] `ChatTextDelta`
  - [x] `ChatTextSegmentFinished`
- [x] Add canonical reasoning payloads:
  - [x] `ChatReasoningSegmentStarted`
  - [x] `ChatReasoningDelta`
  - [x] `ChatReasoningSegmentFinished`
- [x] Add canonical tool payloads:
  - [x] `ChatToolCallStarted`
  - [x] `ChatToolCallArgumentsDelta`
  - [x] `ChatToolCallRequested`
  - [x] `ChatToolExecutionStarted`
  - [x] `ChatToolResultReady`
  - [x] `ChatToolCallFinished`
- [x] Remove or stop generating active payloads for:
  - [x] `ChatMessageUpdate` as the universal text delta/final payload, if fully replaced;
  - [x] old top-level provider fields duplicated outside `CorrelationInfo`;
  - [x] `ChatInferenceStarted`;
  - [x] `ChatTokensDelta`;
  - [x] `ChatInferenceFinished`;
  - [x] `ChatInferenceStopped`.
- [x] Regenerate Pinocchio Go protobufs.
- [x] Regenerate Pinocchio web/chat TypeScript protobufs if that repo owns any.
- [x] Update protobuf registration code/tests.
- [x] Run Pinocchio protobuf generation/check commands.

## Phase 8 — Replace Pinocchio runtime sink and projections

Goal: make Pinocchio consume only canonical Geppetto events and publish only canonical chatapp events.

- [x] Update `../pinocchio/pkg/chatapp/runtime_sink.go`:
  - [x] remove `EventFinal` branch;
  - [x] remove `EventPartialCompletion` branch;
  - [x] remove logic that creates text segments for provider-call events;
  - [x] add handlers for canonical run/provider/text/reasoning/tool events;
  - [x] map typed Geppetto `Correlation` to protobuf `CorrelationInfo`.
- [x] Update `../pinocchio/pkg/chatapp/runtime_inference.go`:
  - [x] publish `ChatRunStarted` instead of `ChatInferenceStarted`;
  - [x] publish `ChatRunFinished` when the whole run completes;
  - [x] publish `ChatRunStopped`/`ChatRunFailed` for stop/error;
  - [x] do not synthesize text finalization at run completion unless an explicit canonical text event exists.
- [x] Update `../pinocchio/pkg/chatapp/projections.go`:
  - [x] project `ChatTextDelta` into UI text append/update events;
  - [x] project `ChatTextSegmentFinished` into finished text segment UI state;
  - [x] project run/provider-call events into debug/status UI only if desired;
  - [x] preserve `CorrelationInfo` in timeline entities.
- [x] Update reasoning plugin:
  - [x] use canonical reasoning events;
  - [x] stop deriving correlation from `metadata.Extra` for new events;
  - [x] preserve `CorrelationInfo` in reasoning UI/timeline entities.
- [x] Update tool plugin:
  - [x] use canonical tool events;
  - [x] support `ToolCallArgumentsDelta` if rendered/debugged;
  - [x] preserve `CorrelationInfo` in tool call/result entities.
- [x] Delete or rewrite tests that assert old event names.
- [ ] Add tests proving provider-call events do not affect text segment state. (runtime no longer closes text on provider/tool boundary; dedicated assertion pending)
- [x] Run `cd ../pinocchio && go test ./pkg/chatapp/... -count=1`.

## Phase 9 — Update Pinocchio web-chat debug SQLite export

Goal: make the trace artifact prove provider-call and segment semantics directly.

- [ ] Update `../pinocchio/cmd/web-chat/app/debug_reconcile_schema.go`.
- [ ] Add `geppetto_inference_results` table.
- [ ] Add `geppetto_segments` table.
- [ ] Update existing `geppetto_records`, `geppetto_provider_events`, and `geppetto_emitted_events` schemas to include canonical correlation fields if needed.
- [ ] Update `../pinocchio/cmd/web-chat/app/debug_reconcile_geppetto.go` to insert:
  - [ ] provider event rows;
  - [ ] canonical emitted event rows;
  - [ ] provider-call inference result rows;
  - [ ] segment rows.
- [ ] Update `../pinocchio/cmd/web-chat/app/debug_reconcile_views.go`:
  - [ ] provider-to-emitted joins by `correlation_key`;
  - [ ] provider-call result view;
  - [ ] segment lifecycle view;
  - [ ] backend-to-frontend correlation view using `CorrelationInfo`;
  - [ ] delivery-gap views updated for new event names.
- [ ] Add SQL tests/fixtures if debug reconcile has test coverage.
- [ ] Verify with sample SQLite export that these queries work:
  - [ ] provider-call results by `provider_call_index`;
  - [ ] text segments by `segment_id`;
  - [ ] tool calls by `tool_call_id` and `correlation_key`;
  - [ ] frontend timeline joins by `correlation_key`.

## Phase 10 — Update CoinVault protobuf mirror and frontend parser

Goal: make CoinVault consume only canonical Pinocchio chatapp events.

- [ ] Regenerate or manually mirror external Pinocchio chatapp TypeScript protobufs into CoinVault:
  - [ ] `web/src/pb/external/pinocchio/chat_pb.ts`
- [ ] Update websocket parser modules:
  - [ ] `web/src/ws/protobuf.ts`
  - [ ] `web/src/ws/entityData.ts`
  - [ ] `web/src/ws/snapshotMapping.ts`
  - [ ] `web/src/ws/uiEventMapping.ts`
  - [ ] `web/src/ws/debugFrames.ts`
  - [ ] `web/src/ws/parsing.ts` barrel if needed.
- [ ] Replace old event-name handling:
  - [ ] remove `ChatInferenceStarted` handling;
  - [ ] remove `ChatTokensDelta` handling;
  - [ ] remove `ChatInferenceFinished` handling;
  - [ ] add `ChatRunStarted`/`ChatRunFinished` handling;
  - [ ] add `ChatTextDelta`/`ChatTextSegmentFinished` handling;
  - [ ] add provider-call debug/status handling if surfaced.
- [ ] Preserve `CorrelationInfo` on all frontend timeline entities.
- [ ] Update Redux/store/entity types as needed:
  - [ ] message/text entities;
  - [ ] reasoning entities;
  - [ ] tool call/result entities;
  - [ ] provider-call debug entities if added.
- [ ] Update frontend tests:
  - [ ] parsing tests for canonical text events;
  - [ ] parsing tests for provider-call events;
  - [ ] parsing tests for tool-call argument deltas;
  - [ ] correlation preservation assertions.
- [ ] Run:
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run typecheck`
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run test:unit -- src/ws/parsing.test.ts src/ws/wsManager.test.ts`

## Phase 11 — Update CoinVault backend/debug scripts and trace browser

Goal: keep all debugging tools useful after event names and schemas change.

- [ ] Update CoinVault debug analysis SQL scripts under the relevant observability ticket.
- [ ] Update Python analysis scripts that refer to old event names.
- [ ] Update `SQLITE-TRACE-VERBS` trace browser app:
  - [ ] overview page shows provider-call results;
  - [ ] conversation page uses canonical text segment events;
  - [ ] correlations page uses `CorrelationInfo` / `correlation_key`;
  - [ ] reasoning page uses canonical reasoning segments;
  - [ ] tool calls page includes tool argument deltas and requested events;
  - [ ] schema/raw pages include new tables.
- [ ] Update trace-browser CLI verbs:
  - [ ] provider-call summary;
  - [ ] segment lifecycle summary;
  - [ ] duplicate-text check using `geppetto_segments`;
  - [ ] provider metadata preservation check using `geppetto_inference_results`.
- [ ] Validate `db-browser serve` against a new SQLite artifact.
- [ ] Validate `db-browser verbs` against a new SQLite artifact.

## Phase 12 — Cross-repo compile and deletion gates

Goal: prove the hard cutover is complete and no old runtime vocabulary remains.

- [ ] Run Geppetto tests:
  - [ ] `cd geppetto && go test ./pkg/events/... -count=1`
  - [x] `cd geppetto && go test ./pkg/steps/ai/gemini ./pkg/inference/tools -count=1`
  - [x] `cd geppetto && go test ./pkg/steps/ai/... -count=1`
  - [x] `cd geppetto && go test ./...`
- [x] Run Pinocchio tests:
  - [x] `cd pinocchio && go test ./pkg/chatapp/... -count=1`
  - [x] `cd pinocchio && go test ./cmd/web-chat/... -count=1`
  - [x] `cd pinocchio && go test ./...` if practical.
- [ ] Run CoinVault tests:
  - [x] `cd 2026-03-16--gec-rag && go test ./internal/webchat ./cmd/coinvault/cmds -count=1`
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run typecheck`
  - [ ] `cd 2026-03-16--gec-rag/web && pnpm run test:unit`
- [x] Run hard deletion search and inspect every match:
  - [x] `rg "EventFinal|EventPartialCompletion|EventTypeFinal|EventTypePartialCompletion" geppetto`
  - [x] `rg "ChatInferenceStarted|ChatInferenceFinished|ChatTokensDelta|ChatInferenceStopped" pinocchio 2026-03-16--gec-rag`
  - [x] remaining Geppetto matches are only historical docs/tickets or explicit deletion notes; no active runtime code remains.
- [ ] Confirm no new routing logic reads correlation from `metadata.Extra`:
  - [ ] `rg "metadata\.Extra|Extra\[\"response_id\"|Extra\[\"correlation_key\"" geppetto pinocchio`
  - [ ] remaining uses are debug-only or old docs.
- [ ] Confirm generated protobufs are up to date.
- [ ] Confirm CoinVault external protobuf mirrors are up to date.

## Phase 13 — End-to-end browser/SQLite validation matrix

Goal: prove the new vocabulary works across real browser sessions and trace artifacts.

- [ ] Start CoinVault full-trace with local workspace:
  - [ ] `PROFILE_SLUG=haiku devctl up --profile full-trace`
- [ ] Run Haiku prompt that forces tool use.
- [ ] Save artifacts:
  - [ ] `debug.sqlite`
  - [ ] `frontend-records.json`
  - [ ] `final-ui.png`
  - [ ] `sqlite-counts.txt`
  - [ ] provider-call result query output;
  - [ ] segment lifecycle query output.
- [ ] Verify Haiku SQLite:
  - [ ] `geppetto_inference_results` has a `tool_use` provider call row;
  - [ ] later provider call row has `end_turn` or final stop reason;
  - [ ] `geppetto_segments` has no text segment created by `message_stop`;
  - [ ] frontend timeline entities retain `CorrelationInfo`.
- [ ] Repeat at least one OpenAI Responses profile.
- [ ] Repeat at least one OpenAI-compatible Chat Completions profile with tool calls.
- [ ] For Chat Completions, verify streamed argument-only deltas keep the same `tool_call_id` and `correlation_key`.
- [ ] Use trace browser to inspect at least one artifact.
- [ ] Save a run report under the CoinVault observability ticket.

## Phase 14 — Documentation and delivery

Goal: update docs to match the hard-cutover implementation, not the old design.

- [ ] Update Geppetto docs:
  - [ ] event vocabulary reference;
  - [ ] provider engine guide;
  - [ ] turns/inference result guide if provider-call results are documented there.
- [ ] Update Pinocchio docs:
  - [ ] chatapp protobuf plugin docs;
  - [ ] React chatapp tutorial;
  - [ ] debug SQLite/reconcile docs.
- [ ] Update CoinVault docs/scripts references that mention old event names.
- [ ] Update this ticket design doc if implementation deviates from the proposed API names.
- [ ] Update this ticket diary with:
  - [ ] what changed;
  - [ ] commands run;
  - [ ] what failed;
  - [ ] what was tricky;
  - [ ] review instructions.
- [ ] Run `docmgr doctor --ticket GP-EVENT-VOCABULARY --root ttmp --stale-after 30`.
- [ ] Upload final implementation guide/report bundle to reMarkable.
- [ ] Verify reMarkable remote listing.

## Phase 15 — Commit strategy

Suggested commit sequence. Keep each commit compiling if possible, but do not merge any subset until the hard cutover is complete.

- [ ] Geppetto commit 1: add canonical event/correlation types and tests.
- [ ] Geppetto commit 2: migrate Claude.
- [ ] Geppetto commit 3: migrate OpenAI Responses.
- [ ] Geppetto commit 4: migrate OpenAI Chat Completions.
- [ ] Geppetto commit 4B: migrate Gemini and local tool execution events.
- [ ] Geppetto commit 5: remove legacy event names and update docs.
- [x] Pinocchio commit 1: replace protobufs with `CorrelationInfo` and canonical messages. (`60f6d3c`, completed by full UI cutover `95fb755`)
- [x] Pinocchio commit 2: replace runtime sink/projections/plugins. (`c3e31da`, `c57559a`, completed by full UI cutover `95fb755`)
- [ ] Pinocchio commit 3: update SQLite reconcile schema/views.
- [ ] CoinVault commit 1: update protobuf mirror and frontend parser.
- [ ] CoinVault commit 2: update tests, debug scripts, and trace browser.
- [ ] Docs commit(s): update ticket diaries/changelogs and final reports.

## Final acceptance criteria

The migration is complete when all of these are true:

- [x] No active runtime code emits or consumes `EventFinal`.
- [x] No active runtime code emits or consumes `EventPartialCompletion`.
- [x] Gemini emits only canonical provider-call/text/tool events.
- [x] Local tool execution emits only canonical tool execution/result events.
- [ ] No active Pinocchio/CoinVault code emits or consumes `ChatInferenceStarted`, `ChatTokensDelta`, or `ChatInferenceFinished`.
- [ ] Every canonical event carries typed correlation.
- [x] Every Pinocchio protobuf payload carries `CorrelationInfo`.
- [x] Provider-call lifecycle events never create text rows.
- [x] Text rows are created/updated/finished only by text-segment events.
- [ ] Claude `message_delta` and `message_stop` emit provider-call events only.
- [ ] OpenAI Responses `response.completed` emits provider-call finished only.
- [ ] OpenAI Chat Completions EOF/final usage emits provider-call finished only.
- [ ] SQLite export can directly show provider-call results and text segment lifecycles.
- [ ] Haiku tool-use browser run has no duplicate assistant text.
- [ ] Haiku SQLite shows `stop_reason=tool_use` in `geppetto_inference_results` for the tool-use provider call.
- [ ] Frontend timeline entities preserve correlation IDs without metadata heuristics.
- [ ] `docmgr doctor` passes for this ticket.
