# Tasks

## TODO

- [ ] Add follow-mode state/actions/selectors in debug-ui uiSlice
- [ ] Implement debug timeline websocket manager with conversation-scoped connect/disconnect
- [ ] Implement bootstrap (`/api/debug/timeline`) then buffered replay ordering for live attach
- [ ] Decode `timeline.upsert` and upsert generic timeline entities with dedupe by version/entity
- [ ] Add follow controls in SessionList and status badge in app shell
- [ ] Support pause/resume/reconnect UX for follow mode
- [ ] Ensure read-only behavior (no outbound control messages)
- [ ] Add websocket lifecycle and two-tab follow integration tests (timeline upsert path)
