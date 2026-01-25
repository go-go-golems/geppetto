# Tasks

## TODO

- [ ] Define snapshot transport protos (`TimelineUpsertV1`, `TimelineSnapshotV1`)
- [ ] Implement server-side timeline projection store (per conversation, versioned)
- [ ] Project engine SEM frames into `sem.timeline.*` snapshots
- [ ] Add `GET /timeline` endpoint (`since_version`, `limit`)
- [ ] Update frontend hydration to use `GET /timeline` (+ since_version WS gating)
- [ ] Add projection/hydration tests (no duplicates, monotonic versions)
- [ ] Decide persistence strategy (in-memory vs SQLite vs Redis) and implement
