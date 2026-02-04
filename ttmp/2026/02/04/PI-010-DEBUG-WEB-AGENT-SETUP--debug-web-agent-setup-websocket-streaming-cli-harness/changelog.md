# Changelog

## 2026-02-04

- Initial workspace created


## 2026-02-04

Added CLI design doc for /chat + /ws streaming harness.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/04/PI-010-DEBUG-WEB-AGENT-SETUP--debug-web-agent-setup-websocket-streaming-cli-harness/analysis/01-cli-harness-for-chat-websocket-streaming.md — Design document with CLI requirements and flows.


## 2026-02-04

Step 1: add debug logging for /ws and /chat (commit e96e8c5)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/router.go — Log ws queries


## 2026-02-04

Step 2: scaffold web-agent-debug CLI skeleton (commit 36d3bfe)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/main.go — Initial subcommand dispatch.


## 2026-02-04

Step 3: implement chat command for CLI harness (commit d9c16c7)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/chat.go — /chat POST implementation.


## 2026-02-04

Step 4: implement ws command for CLI harness (commit 820a2a8)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/ws.go — /ws client with ping + SEM rendering.


## 2026-02-04

Step 5: implement timeline command for CLI harness (commit 38a269b)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/timeline.go — Timeline snapshot fetch + summary.


## 2026-02-04

Step 6: implement run command for CLI harness (commit 5b115e0)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/run.go — Combined ws/chat/timeline flow.


## 2026-02-04

Step 8: fix timeline summary parsing + validate run output (commit b7f54b1)

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/web-agent-example/cmd/web-agent-debug/timeline.go — Use camelCase JSON tags and raw scalar parsing.

