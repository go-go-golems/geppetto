# Changelog

## 2026-01-24

- Initial workspace created


## 2026-01-24

Step 15: Pinocchio backend now emits registry-only SEM with protobuf-authored event.data + stable IDs (pinocchio commit 949beb9)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/design-doc/01-pinocchio-react-webchat-refactor-plan.md — Document stable ID rules and SEM casing decisions
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/sem_translator.go — Registry-only SEM translator; protobuf data envelope; stable IDs (commit 949beb9)


## 2026-01-24

Step 16: Remove legacy TL protocol from Pinocchio backend (pinocchio commit a407483)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/tasks.md — Checked off Task #7
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/forwarder.go — Deleted; TL protocol removed (commit a407483)


## 2026-01-24

Step 17: Backend-owned /chat send queue + idempotency (pinocchio commit 7afc7e8)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/design-doc/01-pinocchio-react-webchat-refactor-plan.md — Document concrete /chat queue + idempotency contract
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — /chat now queues on busy; idempotency response caching (commit 7afc7e8)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/send_queue.go — In-memory per-conversation queue and request records (commit 7afc7e8)


## 2026-01-24

Step 18: Add GET /hydrate (SEM frame hydration) for WS gating (pinocchio commit f696ce4)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/tasks.md — Checked off Task #9
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — GET /hydrate returns buffered SEM frames + since_seq/limit (commit f696ce4)
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/sem_buffer.go — In-memory SEM frame buffer used for hydration (commit f696ce4)


## 2026-01-24

Step 19: Scaffold React+TS+RTK frontend and add Storybook in pinocchio/cmd/web-chat/web (pinocchio commits 456e3e6, 7d5ee47)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/24/PI-003-PORT-TO-REACT--port-pinocchio-webchat-to-react-moments-parity/tasks.md — Checked off Task #10
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.stories.tsx — Storybook story with SEM scenario playback
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx — Single ChatWidget root component (React)


## 2026-01-24

Step 20: Web-chat frontend now decodes SEM payloads via protobuf TS schemas, hardens WS hydration gating, and expands Storybook stories (pinocchio commit 19438d2)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/buf.gen.yaml — Buf now generates TS schemas into the embedded web-chat React app
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/sem/registry.ts — Registry-only SEM routing with `fromJson` decoding
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts — Singleton WS manager: StrictMode-safe hydration gating and buffering
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/src/chat/ChatWidget.stories.tsx — Storybook: widget-only fixtures + streaming scenarios
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/cmd/web-chat/web/package.json — Adds `@bufbuild/protobuf`
