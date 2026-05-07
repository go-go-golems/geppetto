# Tasks

## Phase 0: Ticket and design

- [x] Create ticket `GP-OBSERVABILITY` in `geppetto/ttmp`.
- [x] Write intern-oriented analysis/design/implementation guide for Geppetto provider/event observability.
- [x] Relate key Geppetto and Pinocchio files to the design guide.
- [x] Upload design guide bundle to reMarkable.

## Phase 1: Geppetto observability API

- [ ] Add `geppetto/pkg/observability` package with neutral `Record`, `Observer`, trace-level config, and panic-safe `Notify` helper.
- [ ] Add redaction/capping helpers for raw/provider payloads.
- [ ] Add unit tests for observer panic safety and redaction behavior.

## Phase 2: Glazed configuration

- [ ] Add Geppetto-owned Glazed observability section, e.g. `NewInferenceObservabilitySection`.
- [ ] Add typed `InferenceObservabilitySettings` with `geppetto-trace-level`, max records, max payload bytes, and redaction flag.
- [ ] Validate trace levels: `off`, `events`, `provider`, `raw`.
- [ ] Wire section into apps without using raw `os.Getenv()`.

## Phase 3: OpenAI Responses instrumentation

- [ ] Instrument provider event routing in `pkg/steps/ai/openai_responses/engine.go`.
- [ ] Record reasoning summary deltas, reasoning text deltas, output deltas, and provider completion/failure events.
- [ ] Record Geppetto emitted events around `thinking-ended`, `reasoning-summary`, and high-frequency thinking partials.
- [ ] Preserve provider IDs (`response_id`, `item_id`, `output_index`, `summary_index`) in `EventInfo.Data` when available.
- [ ] Add tests proving provider IDs are captured/preserved.

## Phase 4: Pinocchio integration

- [ ] Mount Geppetto observability section in `pinocchio/cmd/web-chat/main.go`.
- [ ] Decode typed observability settings and pass into runtime/engine construction.
- [ ] Extend `StreamDebugRecorder` with Geppetto debug record kind and observer method.
- [ ] Add `GET /api/debug/sessions/{id}/geppetto`.

## Phase 5: SQLite reconcile export

- [ ] Add `geppetto_records`, `geppetto_provider_events`, and `geppetto_emitted_events` tables.
- [ ] Add views for reasoning sequence, summaries without item IDs, provider-to-timeline correlation, and publish errors.
- [ ] Include Geppetto trace counts in `meta`.
- [ ] Add integration tests opening returned SQLite and verifying Geppetto tables/views.

## Phase 6: Live validation

- [ ] Run devctl-managed Pinocchio web-chat with `--debug-api --geppetto-trace-level provider`.
- [ ] Enable frontend debug, send reasoning prompt, download SQLite.
- [ ] Query full chain: provider event → Geppetto event → Sessionstream event → browser frame → timeline entity → turn snapshot.
- [ ] Document findings and update diary/changelog.
