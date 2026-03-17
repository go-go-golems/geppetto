# Tasks

## Completed

- [x] Create ticket workspace and seed primary docs.
- [x] Map pinocchio engine->SEM->WS->timeline architecture with line-anchored evidence.
- [x] Map go-go-os WS ingest, SEM dispatch, timeline/render pipeline, and event viewer behavior.
- [x] Map go-go-gepa JS VM/plugin surfaces and identify script-event emission gaps.
- [x] Build and run ticket-local SEM envelope prototype (`scripts/sem-envelope-prototype.js`).
- [x] Produce comprehensive design doc with architecture, gaps, APIs, pseudocode, phases, tests, and risks.
- [x] Maintain chronological investigation diary.
- [x] Migrate Pinocchio timeline JS bindings to module API (`require("pinocchio")` + `require("pnocchio")`) and validate with real harness tests.

## Follow-up Implementation Tasks

- [ ] Add `gepa/events` JS host module in `go-go-gepa` with validated SEM emission.
- [ ] Bridge GEPA emitted SEM envelopes into Pinocchio stream transport.
- [ ] Decide initial projection strategy (`timeline.upsert` first vs custom `gepa.*` handlers).
- [ ] Register frontend renderers/handlers for GEPA-specific timeline entities (if custom kinds are used).
- [ ] Add end-to-end integration tests covering emit -> WS -> timeline -> UI.
