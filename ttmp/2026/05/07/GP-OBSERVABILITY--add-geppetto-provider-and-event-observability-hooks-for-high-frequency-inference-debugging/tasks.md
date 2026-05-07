# Tasks

## Phase 0: Ticket and design

- [x] Create ticket `GP-OBSERVABILITY` in `geppetto/ttmp`.
- [x] Write intern-oriented analysis/design/implementation guide for Geppetto provider/event observability.
- [x] Relate key Geppetto and Pinocchio files to the design guide.
- [x] Upload design guide bundle to reMarkable.

## Phase 1: Geppetto observability API

- [x] Add `geppetto/pkg/observability` package with neutral `Record`, `Observer`, trace-level config, and panic-safe `Notify` helper.
- [x] Add redaction/capping helpers for provider `object_json`, Geppetto `event_json`, and `metadata_json` payloads.
- [x] Add unit tests for observer panic safety and redaction behavior.

## Phase 2: Glazed configuration

- [x] Add Geppetto-owned Glazed observability section, e.g. `NewInferenceObservabilitySection`.
- [x] Add typed `InferenceObservabilitySettings` with `geppetto-trace-level`, max records, max payload bytes, and redaction flag.
- [x] Validate trace levels for the first implementation: `off`, `events`, `provider`; reserve `raw` for future raw stream previews if needed.
- [x] Wire section into apps without using raw `os.Getenv()`.

## Phase 3: OpenAI Responses instrumentation

- [x] Instrument provider event routing in `pkg/steps/ai/openai_responses/engine.go`.
- [x] Record reasoning summary deltas, reasoning text deltas, output deltas, and provider completion/failure events.
- [x] Record Geppetto emitted events around `thinking-ended`, `reasoning-summary`, and high-frequency thinking partials.
- [x] Preserve provider IDs (`response_id`, `item_id`, `output_index`, `summary_index`) in `EventInfo.Data` when available.
- [x] Add tests proving provider IDs are captured/preserved.

## Phase 4: Pinocchio integration

- [x] Mount Geppetto observability section in `pinocchio/cmd/web-chat/main.go`.
- [x] Decode typed observability settings and pass into runtime/engine construction.
- [x] Extend `StreamDebugRecorder` with Geppetto debug record kind and observer method.
- [x] Add `GET /api/debug/sessions/{id}/geppetto`.

## Phase 5: SQLite reconcile export

- [x] Add `geppetto_records`, `geppetto_provider_events`, and `geppetto_emitted_events` tables.
- [x] Add views for reasoning sequence, summaries without item IDs, provider-to-timeline correlation, and publish errors.
- [x] Include Geppetto trace counts in `meta`.
- [x] Add integration tests opening returned SQLite and verifying Geppetto tables/views.

## Phase 6: Live validation

- [x] Run devctl-managed Pinocchio web-chat with `--debug-api --geppetto-trace-level provider`.
- [x] Enable frontend debug, send reasoning prompt, download SQLite.
- [x] Query full chain: provider event → Geppetto event → Sessionstream event → browser frame → timeline entity → turn snapshot.
- [x] Document findings and update diary/changelog.
- [ ] Maintain implementation diary after each work slice with commands, failures, validation, and next steps
- [x] Review and narrow observer record schema before Pinocchio/SQLite integration to avoid app-specific or prematurely normalized fields
- [ ] Add performance/retention validation for high-frequency provider object/event/metadata JSON trace paths before live validation
- [ ] Document provider trace privacy policy: default redaction, payload caps, retention behavior, and no raw string capture in the first implementation
- [x] Write textbook-style design assessment report explaining concepts, tradeoffs, risks, and recommended implementation slices
- [x] Preserve decoded provider `object_json`, emitted `event_json`, and `metadata_json` in provider trace mode so missing fields and enrichment/translation bugs can be diagnosed without storing raw stream strings
- [x] Implement first-slice trace levels off/events/provider only; keep raw stream string capture out of v1
- [x] Add observability Record payload fields object_json, event_json, metadata_json with capped/redacted JSON helpers
- [x] Add OpenAI Responses observer options WithObserver and WithObservabilityConfig without changing existing NewEngine callers
- [x] Instrument OpenAI Responses provider_routed_event records with decoded object_json in provider trace mode
- [x] Instrument Geppetto publish records in events/provider trace modes, with full event_json/metadata_json only on done/error records
- [x] Propagate provider IDs into reasoning-summary-started, reasoning-summary-ended, thinking-ended, and final reasoning-summary EventInfo.Data
- [x] Add tests for provider object_json capture, event_json/metadata_json capture, observer panic safety, redaction/capping, and no raw preview field
- [x] Extend Pinocchio SQLite reconcile export with geppetto_records columns for object_json, event_json, metadata_json, stable IDs, stage, and info_message
- [x] Add SQLite export tests verifying Geppetto records round-trip through /api/debug/sessions/{id}/reconcile/upload without raw stream string columns
- [x] Run lightweight Playwright smoke test that starts web-chat with --debug-api --geppetto-trace-level provider, loads UI, and verifies /api/debug/sessions/{id}/geppetto returns kind=geppetto
- [x] Run real provider-backed end-to-end validation: start web-chat with provider tracing, submit prompt, verify /geppetto records, export SQLite, and query Geppetto tables/views
- [x] Run browser-driven real chat session with frontend stream debug enabled, export SQLite with frontend records, and verify delivery_chain plus Geppetto tables
- [ ] Add provider-to-browser reasoning correlation playbook/view using Geppetto partial-thinking event_json deltas, Sessionstream ordinals, frontend parsed chunks, and timeline entity IDs
- [x] Extend Pinocchio ReasoningUpdate payloads with provider response/item/output/summary IDs for direct SQL joins instead of row-order/chunk matching
- [x] Add provider response_id, item_id, output_index, and summary_index fields to Pinocchio ReasoningUpdate protobuf/schema and populate them from Geppetto EventInfo.Data for direct provider-to-browser SQL joins
- [x] Store numbered SQL scripts and latest smoke query outputs under ticket scripts/ for provider-to-browser correlation review
- [x] Add provider-to-browser correlation playbook documenting browser run, SQLite export, SQL scripts, exit criteria, and limitations
