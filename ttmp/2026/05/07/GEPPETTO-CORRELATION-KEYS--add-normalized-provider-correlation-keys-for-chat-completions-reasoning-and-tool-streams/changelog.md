# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

Added Geppetto normalized provider correlation fields for Chat Completions and Responses observability.

### Related Files

- pkg/observability/observer.go — Adds scalar correlation fields to Geppetto records
- pkg/steps/ai/openai/engine_openai.go — Propagates correlation metadata into reasoning/content/tool events
- pkg/steps/ai/openai/observability.go — Builds Chat Completions correlation keys from response id
- pkg/steps/ai/openai_responses/observability.go — Adds Responses correlation keys while preserving native item ids

## 2026-05-07

Propagated normalized provider correlation fields through Pinocchio chatapp, web-chat frontend decoding, timeline entities, and debug SQLite export.

### Related Files

- pinocchio/proto/pinocchio/chatapp/v1/chat.proto — Adds correlation fields to chat UI payloads
- pinocchio/pkg/chatapp/plugins/reasoning.go — Uses correlation metadata in reasoning segment keys and payloads
- pinocchio/pkg/chatapp/plugins/toolcall.go — Carries correlation metadata in tool call/result updates
- pinocchio/pkg/chatapp/runtime_sink.go — Carries correlation metadata in chat message updates
- pinocchio/cmd/web-chat/web/src/ws/timelineEvents.ts — Preserves correlation props in frontend timeline entities
- pinocchio/cmd/web-chat/app/debug_reconcile_schema.go — Adds correlation columns and indexes to Geppetto SQLite tables
- pinocchio/cmd/web-chat/app/debug_reconcile_views.go — Adds correlation-key SQLite joins from provider to frontend

