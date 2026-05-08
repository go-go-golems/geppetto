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
LastUpdated: 2026-05-08T06:45:00-04:00
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
- [ ] Remove or make inaccessible the legacy constructors/types from active provider code:
  - [ ] `NewStartEvent`
  - [ ] `NewPartialCompletionEvent`
  - [ ] `NewFinalEvent`
  - [ ] `NewThinkingPartialEvent`
  - [ ] legacy `EventToolCall` if replaced by `EventToolCallRequested`.
- [ ] Update `NewEventFromJson` / event decoding to use only canonical event type strings.
  - [x] Add `NewEventFromJson` decoding support for canonical event type strings while legacy events still exist during staged implementation.
- [x] Add canonical validation guard for typed-correlation invariants (`ValidateCanonicalEvent`).
- [ ] Update event printer/structured sinks so canonical events render clearly.
- [x] Add Geppetto event serialization tests:
  - [x] every event round-trips through JSON;
  - [x] every event preserves `Correlation`;
  - [x] provider-call events do not have text payloads in the new event structs;
  - [x] text segment events require `SegmentID` and `CorrelationKey` via `ValidateCanonicalEvent`.
- [x] Run `go test ./pkg/events/... -count=1`.

## Phase 2 — Geppetto correlation builders and invariants

Goal: centralize identity construction so providers do not each invent ad hoc metadata maps.

- [ ] Add a provider-call correlation builder package/helper, e.g. `pkg/events/correlationbuilder` or provider-local helpers with shared tests.
- [ ] Implement provider-call ID generation:
  - [ ] stable across all events in one provider API call;
  - [ ] independent of provider response ID because some providers reveal response IDs late;
  - [ ] includes `InferenceID` or `RunID` plus `ProviderCallIndex`.
- [ ] Implement segment ID generation:
  - [ ] stable across start/delta/finish for the segment;
  - [ ] includes provider call identity and stream object identity;
  - [ ] does not depend on rendered Pinocchio message IDs.
- [ ] Implement normalized correlation-key builders for:
  - [ ] Claude provider calls;
  - [ ] Claude text blocks;
  - [ ] Claude tool-use blocks;
  - [ ] OpenAI Responses provider calls;
  - [ ] OpenAI Responses output items and summaries;
  - [ ] OpenAI-compatible Chat Completions content streams;
  - [ ] OpenAI-compatible Chat Completions reasoning streams;
  - [ ] OpenAI-compatible Chat Completions tool streams by ID or index.
- [ ] Add invariant helpers/tests:
  - [ ] provider-call events must have `ProviderCallID` and provider-call `CorrelationKey`;
  - [ ] text events must have `SegmentID`, `SegmentType=text`, and text `CorrelationKey`;
  - [ ] tool events must have `ToolCallID` when provider supplies it, or `ToolCallIndex` fallback;
  - [ ] no canonical event requires `metadata.Extra` for routing.
- [ ] Decide whether `RunID` aliases `InferenceID` initially or becomes a new generated ID.
- [ ] Document the decision in the design guide if it changes.

## Phase 3 — Migrate Claude/Anthropic first

Goal: use Claude's clean envelope/content-block distinction as the first proof of the new vocabulary.

- [ ] Update `pkg/steps/ai/claude/content-block-merger.go` to maintain provider-call state:
  - [ ] active `ProviderCallID`;
  - [ ] `ProviderCallIndex`;
  - [ ] provider `response_id` from `message_start.message.id`;
  - [ ] current stop reason;
  - [ ] current stop sequence;
  - [ ] latest usage;
  - [ ] duration.
- [ ] Emit `EventProviderCallStarted` for `message_start`.
- [ ] Emit `EventProviderCallMetadataUpdated` for `message_delta`.
- [ ] Emit `EventProviderCallFinished` for `message_stop`.
- [ ] Ensure `message_delta` emits no text event.
- [ ] Ensure `message_stop` emits no text event.
- [ ] Update text block handling:
  - [ ] `content_block_start type=text` emits `EventTextSegmentStarted`;
  - [ ] `content_block_delta type=text_delta` emits `EventTextDelta`;
  - [ ] `content_block_stop type=text` emits `EventTextSegmentFinished`.
- [ ] Update tool-use block handling:
  - [ ] `content_block_start type=tool_use` emits `EventToolCallStarted`;
  - [ ] `content_block_delta type=input_json_delta` emits `EventToolCallArgumentsDelta`;
  - [ ] `content_block_stop type=tool_use` emits `EventToolCallRequested`.
- [ ] Ensure Claude correlation fields are populated on every event:
  - [ ] `provider=claude`;
  - [ ] `model`;
  - [ ] `provider_call_id`;
  - [ ] `provider_call_index`;
  - [ ] `response_id` when known;
  - [ ] `content_block_index` for block events;
  - [ ] `segment_id` for text/reasoning segments;
  - [ ] `tool_call_id` for tool-use blocks;
  - [ ] `correlation_key`;
  - [ ] `parent_correlation_key`.
- [ ] Update `pkg/steps/ai/claude/engine_claude.go` to publish canonical events only.
- [ ] Preserve canonical inference-result persistence from provider-call metadata:
  - [ ] tool-use call persists/exports `stop_reason=tool_use`;
  - [ ] final call persists/exports `stop_reason=end_turn` or provider equivalent.
- [ ] Update Claude tests:
  - [ ] text-only stream;
  - [ ] text -> tool_use -> `message_stop`;
  - [ ] text -> tool_use -> next provider call -> text -> end_turn;
  - [ ] metadata-only `message_delta` retains usage and stop reason;
  - [ ] no `EventTextSegmentFinished` is emitted from `message_stop`.
- [ ] Run `go test ./pkg/steps/ai/claude -count=1`.

## Phase 4 — Migrate OpenAI Responses

Goal: map Responses item lifecycle to text/reasoning/tool segments and provider response lifecycle to provider-call events.

- [ ] Update `pkg/steps/ai/openai_responses/streaming.go` to create a provider-call correlation before processing stream events.
- [ ] Emit `EventProviderCallStarted` at response creation / first provider event.
- [ ] Preserve provider `response_id` once known and update subsequent correlation.
- [ ] Map text items:
  - [ ] `response.output_item.added type=message` emits `EventTextSegmentStarted` when it represents text output;
  - [ ] `response.output_text.delta` emits `EventTextDelta`;
  - [ ] `response.output_item.done type=message` emits `EventTextSegmentFinished`.
- [ ] Map reasoning items:
  - [ ] reasoning item start emits `EventReasoningSegmentStarted`;
  - [ ] reasoning text/summary deltas emit `EventReasoningDelta`;
  - [ ] reasoning item/summary completion emits `EventReasoningSegmentFinished`.
- [ ] Map function calls:
  - [ ] function-call argument deltas emit `EventToolCallArgumentsDelta` when available;
  - [ ] completed function-call item emits `EventToolCallRequested`.
- [ ] Map `response.completed` to `EventProviderCallFinished` only.
- [ ] Ensure `response.completed` never emits text segment finished unless an actual text item remains unclosed due to provider stream shape; if such a fallback is necessary, make it explicit and test it as text-item closure, not provider-final closure.
- [ ] Update `pkg/steps/ai/openai_responses/observability.go` to fill typed correlation rather than map-only provider data.
- [ ] Update Responses tests for:
  - [ ] text-only response;
  - [ ] reasoning response;
  - [ ] function-call response;
  - [ ] text + function-call response;
  - [ ] `response.completed` no duplicate text finalization;
  - [ ] correlation key stability by item ID and output index.
- [ ] Run `go test ./pkg/steps/ai/openai_responses -count=1`.

## Phase 5 — Migrate OpenAI-compatible Chat Completions

Goal: synthesize explicit segment/provider lifecycles for the less-structured Chat Completions stream.

- [ ] Update `pkg/steps/ai/openai/engine_openai.go` to create a provider-call correlation before reading chunks.
- [ ] Emit `EventProviderCallStarted` on first chunk or before stream processing.
- [ ] Track `response_id` as soon as chunks expose it.
- [ ] Track `choice_index` for all content/reasoning/tool streams.
- [ ] Text stream handling:
  - [ ] on first non-empty `delta.content` for a choice, emit `EventTextSegmentStarted`;
  - [ ] for each content delta, emit `EventTextDelta`;
  - [ ] on `finish_reason=stop`, emit `EventTextSegmentFinished` for active text segments only.
- [ ] Reasoning stream handling:
  - [ ] on first non-empty reasoning delta for a choice, emit `EventReasoningSegmentStarted`;
  - [ ] emit `EventReasoningDelta` for each reasoning delta;
  - [ ] emit `EventReasoningSegmentFinished` when the choice/provider call ends.
- [ ] Tool stream handling:
  - [ ] preserve tool-call IDs across argument-only deltas;
  - [ ] on first tool-call delta for `(choice_index, tool_call_index)`, emit `EventToolCallStarted`;
  - [ ] emit `EventToolCallArgumentsDelta` for argument deltas;
  - [ ] on `finish_reason=tool_calls`, emit `EventToolCallRequested` for each completed tool call.
- [ ] Provider-call finalization:
  - [ ] emit `EventProviderCallMetadataUpdated` when usage/finish reason metadata arrives;
  - [ ] emit `EventProviderCallFinished` at EOF/final chunk;
  - [ ] never emit text segment finished from EOF unless an active text segment exists and is being explicitly closed.
- [ ] Update `pkg/steps/ai/openai/observability.go` to make existing normalized keys feed typed `Correlation`.
- [ ] Update Chat Completions tests for:
  - [ ] text-only answer;
  - [ ] reasoning stream;
  - [ ] tool-call stream with ID on first delta only;
  - [ ] parallel tool calls;
  - [ ] `finish_reason=tool_calls` does not produce duplicate text;
  - [ ] correlation keys match documented shape.
- [ ] Run `go test ./pkg/steps/ai/openai -count=1`.

## Phase 6 — Update Geppetto inference result and observability output

Goal: expose provider-call results and canonical segment records for SQLite/debugging without guessing from emitted transcript events.

- [ ] Extend `pkg/observability/observer.go` or add a typed observability record for provider-call inference results.
- [ ] Add observability stages or record kinds for:
  - [ ] provider event received;
  - [ ] canonical event emitted;
  - [ ] provider-call result finalized;
  - [ ] segment started/updated/finished if needed for SQLite.
- [ ] Ensure provider-call result records include:
  - [ ] all run/provider-call IDs;
  - [ ] provider/model/response ID;
  - [ ] stop reason;
  - [ ] finish class;
  - [ ] usage;
  - [ ] duration;
  - [ ] has tool calls;
  - [ ] `correlation_key`.
- [ ] Ensure segment records include:
  - [ ] `segment_id`;
  - [ ] `segment_type`;
  - [ ] stream kind;
  - [ ] provider-call IDs;
  - [ ] provider-native IDs/indexes;
  - [ ] text length/status;
  - [ ] start/finish record IDs if known.
- [ ] Update inference result builder if needed so `tool_use` maps to `tool_calls_pending` for provider-call result rows.
- [ ] Add tests for inference result and segment observability records.
- [ ] Run `go test ./pkg/observability ./pkg/inference/engine ./pkg/steps/ai/... -count=1`.

## Phase 7 — Replace Pinocchio protobuf contract

Goal: make `CorrelationInfo` canonical and remove old event payload names.

- [ ] Update `../pinocchio/proto/pinocchio/chatapp/v1/chat.proto`.
- [ ] Add `CorrelationInfo` message with all required correlation fields.
- [ ] Add canonical run payloads:
  - [ ] `ChatRunStarted`
  - [ ] `ChatRunFinished`
  - [ ] `ChatRunStopped`
  - [ ] `ChatRunFailed`
- [ ] Add canonical provider-call payloads:
  - [ ] `ChatProviderCallStarted`
  - [ ] `ChatProviderCallMetadataUpdated`
  - [ ] `ChatProviderCallFinished`
- [ ] Add canonical text payloads:
  - [ ] `ChatTextSegmentStarted`
  - [ ] `ChatTextDelta`
  - [ ] `ChatTextSegmentFinished`
- [ ] Add canonical reasoning payloads:
  - [ ] `ChatReasoningSegmentStarted`
  - [ ] `ChatReasoningDelta`
  - [ ] `ChatReasoningSegmentFinished`
- [ ] Add canonical tool payloads:
  - [ ] `ChatToolCallStarted`
  - [ ] `ChatToolCallArgumentsDelta`
  - [ ] `ChatToolCallRequested`
  - [ ] `ChatToolExecutionStarted`
  - [ ] `ChatToolResultReady`
  - [ ] `ChatToolCallFinished`
- [ ] Remove or stop generating active payloads for:
  - [ ] `ChatMessageUpdate` as the universal text delta/final payload, if fully replaced;
  - [ ] old top-level provider fields duplicated outside `CorrelationInfo`;
  - [ ] `ChatInferenceStarted`;
  - [ ] `ChatTokensDelta`;
  - [ ] `ChatInferenceFinished`;
  - [ ] `ChatInferenceStopped`.
- [ ] Regenerate Pinocchio Go protobufs.
- [ ] Regenerate Pinocchio web/chat TypeScript protobufs if that repo owns any.
- [ ] Update protobuf registration code/tests.
- [ ] Run Pinocchio protobuf generation/check commands.

## Phase 8 — Replace Pinocchio runtime sink and projections

Goal: make Pinocchio consume only canonical Geppetto events and publish only canonical chatapp events.

- [ ] Update `../pinocchio/pkg/chatapp/runtime_sink.go`:
  - [ ] remove `EventFinal` branch;
  - [ ] remove `EventPartialCompletion` branch;
  - [ ] remove logic that creates text segments for provider-call events;
  - [ ] add handlers for canonical run/provider/text/reasoning/tool events;
  - [ ] map typed Geppetto `Correlation` to protobuf `CorrelationInfo`.
- [ ] Update `../pinocchio/pkg/chatapp/runtime_inference.go`:
  - [ ] publish `ChatRunStarted` instead of `ChatInferenceStarted`;
  - [ ] publish `ChatRunFinished` when the whole run completes;
  - [ ] publish `ChatRunStopped`/`ChatRunFailed` for stop/error;
  - [ ] do not synthesize text finalization at run completion unless an explicit canonical text event exists.
- [ ] Update `../pinocchio/pkg/chatapp/projections.go`:
  - [ ] project `ChatTextDelta` into UI text append/update events;
  - [ ] project `ChatTextSegmentFinished` into finished text segment UI state;
  - [ ] project run/provider-call events into debug/status UI only if desired;
  - [ ] preserve `CorrelationInfo` in timeline entities.
- [ ] Update reasoning plugin:
  - [ ] use canonical reasoning events;
  - [ ] stop deriving correlation from `metadata.Extra` for new events;
  - [ ] preserve `CorrelationInfo` in reasoning UI/timeline entities.
- [ ] Update tool plugin:
  - [ ] use canonical tool events;
  - [ ] support `ToolCallArgumentsDelta` if rendered/debugged;
  - [ ] preserve `CorrelationInfo` in tool call/result entities.
- [ ] Delete or rewrite tests that assert old event names.
- [ ] Add tests proving provider-call events do not affect text segment state.
- [ ] Run `cd ../pinocchio && go test ./pkg/chatapp/... -count=1`.

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
  - [x] `cd geppetto && go test ./pkg/steps/ai/... -count=1`
  - [ ] `cd geppetto && go test ./...`
- [ ] Run Pinocchio tests:
  - [x] `cd pinocchio && go test ./pkg/chatapp/... -count=1`
  - [ ] `cd pinocchio && go test ./cmd/web-chat/... -count=1`
  - [ ] `cd pinocchio && go test ./...` if practical.
- [ ] Run CoinVault tests:
  - [x] `cd 2026-03-16--gec-rag && go test ./internal/webchat ./cmd/coinvault/cmds -count=1`
  - [x] `cd 2026-03-16--gec-rag/web && pnpm run typecheck`
  - [ ] `cd 2026-03-16--gec-rag/web && pnpm run test:unit`
- [ ] Run hard deletion search and inspect every match:
  - [x] `rg "EventFinal|EventPartialCompletion|EventTypeFinal|EventTypePartialCompletion" geppetto`
  - [x] `rg "ChatInferenceStarted|ChatInferenceFinished|ChatTokensDelta|ChatInferenceStopped" pinocchio 2026-03-16--gec-rag`
  - [ ] remaining matches are only historical docs, archived tickets, or explicit deprecation notes; no active runtime code remains.
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
- [ ] Geppetto commit 5: remove legacy event names and update docs.
- [ ] Pinocchio commit 1: replace protobufs with `CorrelationInfo` and canonical messages.
- [ ] Pinocchio commit 2: replace runtime sink/projections/plugins.
- [ ] Pinocchio commit 3: update SQLite reconcile schema/views.
- [ ] CoinVault commit 1: update protobuf mirror and frontend parser.
- [ ] CoinVault commit 2: update tests, debug scripts, and trace browser.
- [ ] Docs commit(s): update ticket diaries/changelogs and final reports.

## Final acceptance criteria

The migration is complete when all of these are true:

- [ ] No active runtime code emits or consumes `EventFinal`.
- [ ] No active runtime code emits or consumes `EventPartialCompletion`.
- [ ] No active Pinocchio/CoinVault code emits or consumes `ChatInferenceStarted`, `ChatTokensDelta`, or `ChatInferenceFinished`.
- [ ] Every canonical event carries typed correlation.
- [ ] Every Pinocchio protobuf payload carries `CorrelationInfo`.
- [ ] Provider-call lifecycle events never create text rows.
- [ ] Text rows are created/updated/finished only by text-segment events.
- [ ] Claude `message_delta` and `message_stop` emit provider-call events only.
- [ ] OpenAI Responses `response.completed` emits provider-call finished only.
- [ ] OpenAI Chat Completions EOF/final usage emits provider-call finished only.
- [ ] SQLite export can directly show provider-call results and text segment lifecycles.
- [ ] Haiku tool-use browser run has no duplicate assistant text.
- [ ] Haiku SQLite shows `stop_reason=tool_use` in `geppetto_inference_results` for the tool-use provider call.
- [ ] Frontend timeline entities preserve correlation IDs without metadata heuristics.
- [ ] `docmgr doctor` passes for this ticket.
