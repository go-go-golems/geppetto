# Changelog

## 2026-05-07

- Initial workspace created


## 2026-05-07

- Created `GP-OBSERVABILITY` ticket for Geppetto provider/event observability.
- Added detailed intern-oriented implementation guide: `design/01-geppetto-provider-event-observability-implementation-guide.md`.
- Added task breakdown covering Geppetto observer API, Glazed configuration, OpenAI Responses instrumentation, Pinocchio integration, SQLite export, and live validation.
- Related key Geppetto and Pinocchio implementation files to the design guide.
- Uploaded ticket bundle to reMarkable: `/ai/2026/05/07/GP-OBSERVABILITY/GP-OBSERVABILITY Geppetto Provider Event Observability Guide.pdf`.

## 2026-05-07

Created implementation diary, added explicit diary-maintenance task, and recorded catch-up findings before code implementation.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — New diary document for implementation history


## 2026-05-07

Recorded design assessment and added follow-up tasks for schema narrowing, high-frequency performance/retention validation, and provider trace privacy policy.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Design assessment diary entry
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Added design-risk follow-up tasks


## 2026-05-07

Clarified report guidance: raw previews and decoded object JSON are required diagnostic evidence for provider/raw trace modes, not optional nice-to-have data, because many bugs involve missing fields or lower-level misinterpretation.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/analysis/01-textbook-report-geppetto-provider-event-observability-design-assessment.md — Clarified raw/object payload concepts and raw capture requirements
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Added raw evidence preservation task


## 2026-05-07

Wrote textbook-style design assessment report, narrowed first-slice diagnostic payloads to decoded provider object_json plus Geppetto event_json/metadata_json (no raw stream strings), and uploaded the report to reMarkable at /ai/2026/05/07/GP-OBSERVABILITY/GP-OBSERVABILITY Textbook Design Assessment.pdf.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/analysis/01-textbook-report-geppetto-provider-event-observability-design-assessment.md — Textbook report and revised diagnostic payload guidance
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Recorded diagnostic payload scope correction
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Checked report task and updated payload tasks


## 2026-05-07

Implemented first-slice Geppetto observability: neutral Record/Observer/config package, Glazed observability section, OpenAI Responses observer options, provider object_json records, publish event_json/metadata_json records, provider ID propagation into reasoning info events, and unit tests. Raw stream strings remain out of scope for v1.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/cli/bootstrap/inference_observability.go — Glazed observability section and typed settings
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/config.go — Trace levels off/events/provider and Config defaults
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/json.go — Capped/redacted evidence JSON helper
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/observer.go — Neutral Record/Observer and panic-safe Notify
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/observability/observer_test.go — Trace parsing
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/engine.go — OpenAI Responses instrumentation and provider ID propagation
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/observability.go — Observer options and record builders
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/steps/ai/openai_responses/observability_test.go — Object/event/metadata JSON and provider ID tests


## 2026-05-07

Wired first-slice Geppetto observability into Pinocchio web-chat: mounted Glazed observability settings, injected the debug recorder as OpenAI Responses observer through an engine-factory option, added Geppetto debug record storage, and exposed GET /api/debug/sessions/{id}/geppetto with endpoint coverage. SQLite export remains follow-up work.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/pkg/inference/engine/factory/factory.go — OpenAI Responses option hook for observer injection
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/debug_recorder.go — Geppetto debug record kind and observer adapter
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/server_debug.go — Geppetto debug endpoint
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/server_test.go — Geppetto endpoint test
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/main.go — Glazed observability settings decode and observer wiring
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/runtime_composer.go — Custom engine factory support in runtime composition


## 2026-05-07

Extended Pinocchio SQLite reconcile export with Geppetto tables/views and meta counts, added SQLite round-trip tests for object_json/event_json/metadata_json records, and ran a lightweight Playwright smoke against web-chat started with --debug-api --geppetto-trace-level provider; /api/debug/sessions/smoke/geppetto returned kind=geppetto.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Recorded SQLite implementation and Playwright smoke
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Checked SQLite export and lightweight smoke tasks
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/debug_reconcile_db.go — Geppetto SQLite tables
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/server_test.go — SQLite export assertions and test helpers


## 2026-05-07

Ran browser-driven real end-to-end chat validation with frontend stream debug enabled: typed and sent a prompt in Playwright, UI finished with answer, collected 837 frontend debug entries, fetched 1111 Geppetto records, exported /tmp/browser-chat-e2e.sqlite with frontend/backend/Geppetto records, and verified delivery_chain plus Geppetto reasoning/provider/emitted tables. Console warnings/errors were 0.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Recorded browser-driven real chat and SQLite correlation validation
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Checked browser-driven e2e validation tasks


## 2026-05-07

Analyzed /tmp/browser-chat-e2e.sqlite provider-to-browser correlation: 259 provider normalize deltas, 259 backend reasoning deltas, and 259 frontend ChatReasoningAppended frames; Geppetto partial-thinking event_json.delta matched frontend payload.chunk 259/259, mapping provider item rs_03990f2aaba1fe850069fcbe97c1a481909276888e7b2d8504 to browser/timeline message chat-msg-1:thinking:1. Added follow-up tasks for a formal correlation view/playbook and direct provider IDs in ReasoningUpdate.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Provider-to-browser correlation analysis
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/tasks.md — Added correlation follow-up tasks


## 2026-05-07

Saved numbered provider-to-browser SQL scripts under scripts/, added a correlation playbook, added the geppetto_reasoning_to_frontend SQLite view, added a follow-up task for provider ID fields in ReasoningUpdate, and reran a full browser-driven smoke. Latest run: session e4394b1d-4e89-47b7-9259-cd7adfafd07d, 1158 frontend records, 1539 Geppetto records, 359/359 Geppetto-to-frontend exact delta matches; outputs saved under scripts/outputs/.

### Related Files

- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/playbook/01-provider-to-browser-correlation-playbook.md — Repeatable validation playbook
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/reference/01-diary.md — Recorded SQL scripts
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/scripts/03-provider-to-browser-correlation.sql — Saved provider-to-browser SQL correlation query
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/geppetto/ttmp/2026/05/07/GP-OBSERVABILITY--add-geppetto-provider-and-event-observability-hooks-for-high-frequency-inference-debugging/scripts/outputs/04-correlation-quality-checks.out — Latest smoke output showing correlation match counts
- /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio/cmd/web-chat/app/debug_reconcile_db.go — Added geppetto_reasoning_to_frontend view

