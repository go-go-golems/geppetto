# Tasks

## TODO

- [x] Seed webchat sessions with a system prompt block derived from profile config.
- [x] Set default profile system prompt to "You are an assistant" and ensure other profiles have non-empty prompts.
- [x] Add/update tests for empty prompt run (seed turn non-empty).
- [x] Run backend checks (go test / lint) and document results in diary.

- [x] Add tasks here
- [x] Fix Router.Mount subpath handling with http.StripPrefix + base-path redirect; add tests for mounting under /api/webchat.
- [x] Split UI serving from API: introduce API-only handler + optional UI handler; keep existing Router by composing both; update docs/tests.
- [x] Extract ConversationManager to own get/create + add/remove connections; update router to use manager methods and centralize conversation lifecycle.
- [x] Move run/idempotency/queue logic out of router.go into conversation/queue helpers; add unit tests for queue drain and idempotency replay.
- [x] Update StreamCoordinator to derive seq from Redis stream IDs when available (xid metadata) with fallback to local counter; add tests.
- [x] Add eviction loop for idle conversations (no conns, no running/queued work) with configurable idle timeout + interval; add tests.
- [ ] Make ConnectionPool non-blocking with per-connection writer goroutines + backpressure strategy; add tests.
- [ ] Add optional TimelineStoreV2 UpsertWithVersion hook to use stream-derived version hints; update projector to prefer v2 when available; tests.
