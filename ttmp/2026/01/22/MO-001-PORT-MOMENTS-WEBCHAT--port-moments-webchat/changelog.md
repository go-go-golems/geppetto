# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Step 1-3: Create ticket + deep analysis of porting go-go-mento webchat to MO-007 Session/ExecutionHandle model; propose convergence plan with pinocchio webchat (adopt ConnectionPool/StreamCoordinator + add SessionManager; fix SessionID vs run_id semantics).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/analysis/01-port-go-go-mento-webchat-to-geppetto-session-design.md — In-depth delta analysis + migration plan
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/reference/01-diary.md — Detailed diary of survey and decisions


## 2026-01-22

Step 4: Ticket hygiene: add index links, replace placeholder tasks with actionable list, and relate key files to analysis+diary docs.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/index.md — Add quick links to analysis/diary
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/tasks.md — Replace placeholder with task breakdown


## 2026-01-22

Step 5: Add design docs for (a) event versioning+ordering and (b) step controller integration; update tasks to reflect: move go-go-mento good parts into pinocchio, reverse middleware order, ToolExecutor injection in toolhelpers, no DB persistence yet.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/design-doc/01-event-versioning-ordering-from-go-go-mento-to-pinocchio.md — Ordering/versioning plan and diagrams
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/design-doc/02-step-controller-integration-from-go-go-mento-to-pinocchio.md — Step mode integration plan
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/MO-001-PORT-MOMENTS-WEBCHAT--port-moments-webchat/tasks.md — Update task breakdown with new decisions


## 2026-01-22

Step 7: Start implementation: ToolExecutor injection in geppetto toolhelpers; pinocchio webchat now uses ConnectionPool+StreamCoordinator with cursor (stream_id or seq) and ws hello/ping/pong; middleware application order reversed.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolhelpers/helpers.go — Add ToolExecutor to tool loop config
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/connection_pool.go — Centralize ws connection management
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/conversation.go — Refactor to use pool+coordinator
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine.go — Reverse middleware application order
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Add ws.hello and ws.ping/pong
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/stream_coordinator.go — Consume events in-order and inject cursor fields


## 2026-02-25

Ticket closed

