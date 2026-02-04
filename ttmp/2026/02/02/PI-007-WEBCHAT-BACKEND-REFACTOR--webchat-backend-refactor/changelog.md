# Changelog

## 2026-02-02

- Initial workspace created


## 2026-02-02

Seed webchat sessions with system prompt blocks, default prompts to 'You are an assistant', and add tests for empty-prompt runs.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/main.go — Default profile prompt
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go — Seed turn now includes system prompt
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation_test.go — Seed turn tests
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_builder.go — Fallback system prompt when profile is empty


## 2026-02-02

Fix chat payload mismatch: send prompt from UI and accept text as a backend alias.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx — Send prompt in chat payload
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_from_req.go — Map text -> prompt


## 2026-02-03

Disable remote Redux devtools to avoid socketcluster WS attempts.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/store/store.ts — Remove devtools enhancer


## 2026-02-03

Step 1: Fix Router.Mount to strip prefixes and add base-path redirect for subpath embedding (commit bf2c934).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — StripPrefix + redirect for mount
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router_mount_test.go — Mount tests


## 2026-02-03

Closed after seed-turn fix, prompt aliasing, and devtools disable.


## 2026-02-03

Reopened for continued work.


## 2026-02-03

Step 2: Split UI and API handlers, allow fs.FS static assets, and update docs/tests (commit 94f8d20).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — API/UI handler split
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router_handlers_test.go — Handler tests


## 2026-02-03

Step 3: Extract conversation lifecycle into ConvManager and update router delegation (commit 2a29380).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go — ConvManager lifecycle


## 2026-02-03

Step 4: Move queue/idempotency logic into conversation helpers with unit tests (commit 51929ea).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/send_queue.go — Queue helper refactor
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/send_queue_test.go — Queue helper tests


## 2026-02-03

Step 5: Derive stream seq from Redis IDs; remove legacy metadata keys; add tests and doc updates (commit 1828999).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/stream_coordinator.go — Derive seq
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/stream_coordinator_test.go — Sequencing tests


## 2026-02-03

Step 6: Add idle conversation eviction loop with configurable idle/interval and tests (commit 9c8adce).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conv_manager_eviction.go — Eviction loop
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conv_manager_eviction_test.go — Eviction tests


## 2026-02-03

Step 7: Make ConnectionPool non-blocking with backpressure + tests and doc updates (commit 011c824).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/connection_pool.go — Non-blocking pool
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/connection_pool_test.go — Backpressure test


## 2026-02-03

Step 8: switch timeline storage to seq-based versions and remove compatibility shims (commit 4964d10).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/doc/topics/webchat-sem-and-ui.md — Document seq-based versioning semantics
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/timeline_projector.go — Use event.seq to upsert timeline entities
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/timeline_store.go — Require explicit seq versions in timeline store


## 2026-02-03

Ticket closed

