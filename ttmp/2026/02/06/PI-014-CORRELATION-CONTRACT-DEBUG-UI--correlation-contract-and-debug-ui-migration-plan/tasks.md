# Tasks

## Core

- [ ] Implement correlation envelope v2 (`correlation` object) for SEM frames consumed by debug UI
- [ ] Keep correlation inference-scoped (`conv_id`, `session_id`, `inference_id`, `turn_id`) with no `phase` or `snapshot_seq`
- [ ] Extend turn snapshot schema with deterministic join metadata (`source`, `inference_id`, `seq_hint`)
- [ ] Add `DistinctSessions` API/query path and replace session-summary derivation from limited list queries
- [ ] Migrate endpoints to `/debug/turns` and `/debug/timeline` with no backwards compatibility aliases
- [ ] Implement per-layer middleware tracing wrappers at engine composition point (including built-ins)
- [ ] Call `SnapshotHook` with `phase=final` in addition to final `TurnPersister` writes
- [ ] Build separate debug app scaffold in `web-agent-example/cmd/web-agent-debug`
- [ ] Use RTK/RTK Query and the same CSS/styling framework guidelines as existing webchat in the new debug app

## Validation

- [ ] Add acceptance tests for event/snapshot join determinism
- [ ] Add acceptance tests for seq ordering and migration correctness
- [ ] Add acceptance tests for middleware trace coverage/order

## Documentation

- [ ] Update PI-013 design document with implemented decisions once work is done
