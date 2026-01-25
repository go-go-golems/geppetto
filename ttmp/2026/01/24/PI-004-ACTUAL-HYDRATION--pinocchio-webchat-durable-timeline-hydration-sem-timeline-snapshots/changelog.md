# Changelog

## 2026-01-24

- Initial workspace created


## 2026-01-24

Step 2: choose SQLite multi-conversation projection store; expand tasks

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/design-doc/01-durable-hydration-via-sem-timeline-snapshots.md — Record SQLite persistence decision and schema
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-004-ACTUAL-HYDRATION--pinocchio-webchat-durable-timeline-hydration-sem-timeline-snapshots/tasks.md — Detailed implementation checklist


## 2026-01-24

Step 3: implement SQLite projection store + projector + GET /timeline (pinocchio commits d97efe1, 244757b, b1f908b)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — GET /timeline + store enablement
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/timeline_projector.go — SEM->snapshot projection
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/timeline_store_sqlite.go — Durable projection store
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/proto/sem/timeline/transport.proto — Snapshot transport schema


## 2026-01-24

Step 4: configure timeline store via Glazed params (no env) (pinocchio commit 4c27169)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/main.go — Add --timeline-dsn/--timeline-db parameters
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Decode timeline DSN/DB from ParsedLayers
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/server.go — Close sqlite store on shutdown


## 2026-01-24

Step 5: frontend uses GET /timeline and URL conv_id is source of truth (pending pinocchio commit)


## 2026-01-24

Step 5: frontend uses GET /timeline + URL conv_id is source of truth (pinocchio commit 110280b)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx — Update URL with conv_id after first message; read conv_id from URL on mount
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts — Hydrate timeline from /timeline snapshot (fallback to /hydrate)


## 2026-01-24

Step 6: Fix New conv reset, gate stub planning events, and fix planning Storybook crash (pinocchio@81f41c2)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx — New conv resets timeline and ws/app state
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/sem/registry.ts — Clear planning aggregates to avoid frozen-array mutation
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Gate stub planning/thinking-mode SEM emission behind --emit-planning-stubs

