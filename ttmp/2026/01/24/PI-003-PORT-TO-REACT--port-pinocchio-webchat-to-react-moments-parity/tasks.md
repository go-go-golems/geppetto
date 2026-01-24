# Tasks

## TODO

- [ ] Read the required docs (1–2h) and align on invariants
  - Read (in order): `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/design-doc/01-pinocchio-react-webchat-refactor-plan.md`
  - Then: `geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/analysis/01-moments-react-chat-widget-architecture.md`
  - Then: `go-go-mento/docs/reference/webchat/frontend-integration.md`, `go-go-mento/docs/reference/webchat/sem-and-widgets.md`, `go-go-mento/docs/reference/webchat/backend-internals.md`
  - Then: `moments/web/docs/event-driven-widgets.md`, `moments/web/docs/component-development.md`
  - Then: `pinocchio/ttmp/2025-08-22/02-backend-semantic-event-mapping.md`
  - Acceptance: implementor can explain (a) stable IDs, (b) registry-only routing, (c) hydration gating, (d) protojson boundary, (e) “no sink-owned conversation state”.

- [x] Decide SEM envelope strategy (protobuf-first) and lock it
  - Option A (recommended): keep JSON envelope `{ sem: true, event: { type, id, data, metadata } }`, where `data` is protojson for a per-type payload message.
  - Option B: define a protobuf envelope `SemEventEnvelope { oneof payload { ... } }` and still ship JSON (protojson) over the wire.
  - Acceptance: written decision + updated plan doc + an example for `llm.delta` and `tool.start` showing exact JSON on the wire.

- [x] Define and document stable ID rules
  - For each family (`llm.*`, `tool.*`, `planning.*`, widgets), specify:
    - entity ID source,
    - correlation ID source (run/session/turn),
    - dedupe rules on replay/hydration.
  - Acceptance: a small “ID contracts” section exists and is referenced by both backend SEM translator and frontend handlers.

- [x] Create Pinocchio SEM protobuf source tree (baseline copy from go-go-mento)
  - Create `pinocchio/proto/sem/**` by copying/adapting from `go-go-mento/proto/sem/**` as a starting point.
  - Adjust `option go_package` to `github.com/go-go-golems/pinocchio/pkg/sem/pb/proto/sem/<ns>;<alias>`.
  - Acceptance: `pinocchio/proto/sem/**.proto` compiles with Buf.

- [x] Add Buf toolchain to Pinocchio (generate Go + TS)
  - Add `pinocchio/buf.yaml` and `pinocchio/buf.gen.yaml` (pattern: `go-go-mento/buf.yaml`, `go-go-mento/buf.gen.yaml`).
  - Define TS output path for the new React package (e.g., `pinocchio/web/src/sem/pb`).
  - Add a one-liner doc: `cd pinocchio && buf generate`.
  - Acceptance: `buf generate` produces Go types under `pinocchio/pkg/sem/pb/**` and TS schemas under the chosen React path.

- [x] Backend: replace the monolithic SEM switch with a registry + protobuf-authored payloads
  - Start from `pinocchio/pkg/webchat/sem_translator.go`.
  - Create registry-first mapping (reuse/adapt `moments/backend/pkg/sem/registry` or implement in Pinocchio).
  - For each SEM event, construct the payload as a protobuf message and convert via protojson at the boundary.
  - Acceptance: no large `switch ev := e.(type)` remains as the primary mapping mechanism.

- [x] Backend: remove legacy TL protocol (`{ tl: true, event: ... }`)
  - Delete the TL envelope implementation (`TimelineEventsFromEvent`) and ensure the websocket stream is SEM-only.
  - Acceptance: no `{ tl: true, ... }` frames are emitted by the backend; legacy TL code path is removed.

- [x] Backend: implement server-side “send serialization / queue semantics” + idempotency
  - One conversation executes at most one run at a time; additional user messages enqueue (or explicit server-side busy semantics).
  - Use an idempotency key and return a stable request identifier early (idempotency key and/or derived message ID).
  - Acceptance: the UI does not need a retry queue; backend semantics are documented and tested.

- [x] Backend: implement timeline hydration snapshot endpoint(s)
  - V1: serve recent **SEM frames** via `GET /hydrate` (frontend replays via handlers).
  - Provide `since_seq` support (or equivalent) for incremental hydration.
  - Future: consider `sem.timeline.*` snapshot messages as a durable canonical snapshot schema.
  - Acceptance: reload + reconnect reproduces a stable timeline without duplication (with WS hydration gating).

- [ ] Frontend: choose the new React package/app location and scaffold tooling
  - Decide: `pinocchio/web/` standalone vs embedded into existing server static pipeline.
  - Set up Vite + React + TS + RTK + Storybook.
  - Wire Buf TS generation into the React tree (no checked-in drift).
  - Acceptance: `pnpm dev` and `pnpm storybook` (or equivalent) run successfully.

- [ ] Frontend: implement RTK timeline store + widget registry + SEM registry (registry-only)
  - Implement `timelineSlice` with idempotent add/upsert/append semantics.
  - Implement `sem/registry.ts` and require all SEM types be handled there (no `switch (ev.type)` fallback).
  - Implement widget renderer registry by `entity.kind`.
  - Acceptance: core SEM stream updates entities and renders widgets without any fallback switch.

- [ ] Frontend: implement singleton WS manager + hydration gating
  - Ensure exactly one connection per conversation; handle StrictMode double-mount.
  - Gate application of WS deltas until hydration completes.
  - Acceptance: refresh/reconnect does not duplicate entities and does not miss early events.

- [ ] Frontend: implement a single `ChatWidget` root component
  - One integration surface; variants are layout-only props.
  - Acceptance: `ChatWidget` can be embedded in multiple pages without semantic drift.

- [ ] Storybook: add widget-only stories and end-to-end SEM scenario stories
  - Widget-only stories: each entity kind gets a representative fixture.
  - Scenario stories: “core streaming + tools”, “reconnect + hydrate”, “debug pause”, etc.
  - Acceptance: you can iterate on individual widgets without running the server.

- [ ] Remove/deprecate old Pinocchio Preact/Zustand web UI once parity exists
  - Deprecate `pinocchio/cmd/web-chat/web/**`.
  - Acceptance: one supported webchat UI remains; old code path is removed or explicitly archived.
