# Tasks

## TODO

- [x] Define snapshot transport protos (`TimelineUpsertV1`, `TimelineSnapshotV1`)
  - Add a transport schema in `pinocchio/proto/sem/timeline/` that can represent:
    - per-entity upserts (`TimelineUpsertV1`)
    - full/partial snapshots (`TimelineSnapshotV1`)
    - versioning (`version`, `since_version`) and server timestamps
  - Model snapshot payloads as **protobuf-first**:
    - either a `oneof` of snapshot payloads (message/tool/status/middleware/planning/etc), or
    - `google.protobuf.Any` (avoid unless necessary).
  - Acceptance:
    - `cd pinocchio && buf generate` produces Go + TS types
    - protojson encoding produces stable lowerCamelCase field names

- [x] Implement server-side timeline projection store (SQLite, multi-conversation, versioned)
  - Use SQLite as the canonical persistence layer (durable across restart).
  - Store must support multiple conversations (keyed by `conv_id`).
  - Schema requirements (minimal viable):
    - per-conversation monotonic `version`
    - entity upsert by stable `entity_id`
    - ability to query `since_version` with `limit`
  - Acceptance:
    - a single SQLite DB file can store multiple conversations concurrently
    - `version` is monotonic per `conv_id`
    - upserts are idempotent (same `entity_id` overwrites props without duplicating order)

- [x] Project engine SEM frames into `sem.timeline.*` snapshots
  - Implement server-side projector: `SEM frame -> projection updates -> sem.timeline snapshots`.
  - Initial projection coverage (minimum):
    - `llm.start|delta|final` -> `sem.timeline.MessageSnapshotV1`
    - `tool.start|result|done` -> `sem.timeline.ToolCallSnapshotV1` + `sem.timeline.ToolResultSnapshotV1`
    - `thinking.mode.*` -> `sem.timeline.ThinkingModeSnapshotV1`
    - `planning.*` / `execution.*` -> `sem.timeline.PlanningSnapshotV1` (add if missing)
  - IMPORTANT:
    - projection is **not** implemented as a “sink that mutates conversation state”
    - projection is a separate subsystem owned by the webchat backend, applied after SEM translation
  - Acceptance:
    - running a chat produces timeline entities in SQLite that can be reloaded after restart

- [x] Add `GET /timeline` endpoint (`since_version`, `limit`)
  - HTTP handler returns a protojson `TimelineSnapshotV1` (or equivalent).
  - Behavior:
    - `GET /timeline?conv_id=...` returns full snapshot (all entities)
    - `GET /timeline?conv_id=...&since_version=N&limit=L` returns only updates after `N`
  - Acceptance:
    - response includes `conv_id`, `version`, and entities/upserts
    - endpoint is safe to call repeatedly (idempotent)

- [x] Update frontend hydration to use `GET /timeline` (+ since_version WS gating)
  - Replace “replay buffered SEM frames” with “apply timeline snapshot entities”.
  - Keep the existing singleton WS manager + hydration gating pattern.
  - Acceptance:
    - refresh/reconnect renders immediately via `/timeline` even if the server restarted
    - no duplication when WS deltas arrive

- [x] Add projection/hydration tests (no duplicates, monotonic versions)
  - Unit tests for SQLite store:
    - upsert creates entity, increments version
    - repeated upsert updates entity, increments version
    - `since_version` returns correct subset and respects `limit`
  - Integration-ish test for projector:
    - apply a short SEM stream (llm/tool/thinking/planning) and assert snapshot contents
  - Acceptance:
    - `cd pinocchio && go test ./...` passes

- [x] Wire configuration: SQLite DSN / file path (no multi-backend)
  - Add a single configuration mechanism for the projection DB (Glazed parameters in `web-chat` command; supports config overlays).
  - Default should remain “no projection persistence” unless configured.
  - Acceptance:
    - docs explain how to enable it and where the DB lives
