# Tasks

## TODO

- [x] Add tasks here

- [x] Add tests for ordering/merge: snapshot + deltas, in-memory store parity with SQLite
- [x] Define monotonic version contract for timeline entities (including user messages); decide version source with/without Redis
- [x] Implement in-memory TimelineStore that matches SQLite ordering semantics and supports size limits
- [x] Wire store selection so /timeline is always available (SQLite when configured, in-memory otherwise)
- [x] Update /timeline response to include snapshot_version watermark (max version)
- [x] Remove /hydrate endpoint or gate it behind explicit debug flag
- [x] Emit timeline-delta updates alongside SEM frames over WS (entity_id, kind, version, payload)
- [x] Update frontend merge logic to apply timeline deltas by version and ignore stale SEM effects
- [x] Add tests for ordering/merge: snapshot + deltas, in-memory store parity with SQLite
