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
Summary: Task list for the provider/run/text segment vocabulary design ticket.
LastUpdated: 2026-05-08T05:55:00-04:00
---

# Tasks

## Done

- [x] Create `GP-EVENT-VOCABULARY` ticket workspace.
- [x] Add primary design document and investigation diary.
- [x] Capture line-numbered source evidence for Geppetto events, provider engines, observability, Pinocchio runtime sink, protobufs, plugins, and SQLite reconcile code.
- [x] Write intern-facing design and implementation guide for a clean provider/run/text segment event vocabulary.
- [x] Include typed correlation-ID design to avoid relying on `metadata.Extra` heuristics.
- [x] Validate ticket with `docmgr doctor`.
- [x] Upload design bundle to reMarkable and verify remote listing.

## TODO

- [ ] Implement typed Geppetto `Correlation` envelope and explicit text/provider-call events.
- [ ] Add Pinocchio `CorrelationInfo` protobuf and explicit chat text/provider-call backend events.
- [ ] Add `geppetto_inference_results` and `geppetto_segments` SQLite export tables/views.
- [ ] Migrate Claude first, then OpenAI Responses, then OpenAI-compatible Chat Completions.
- [ ] Keep legacy aliases for `EventFinal` and `ChatInferenceFinished` during migration.
