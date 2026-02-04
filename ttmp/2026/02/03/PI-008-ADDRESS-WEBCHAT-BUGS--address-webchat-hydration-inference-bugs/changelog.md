# Changelog

## 2026-02-03

- Initial workspace created


## 2026-02-03

Step 1: capture hydration ordering evidence via /timeline + SQLite inspection.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/scripts/inspect_timeline_db.py — Evidence gathering script
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/sources/timeline_cac78e87.json — Captured /timeline payload


## 2026-02-03

Step 2: make StreamCoordinator fallback seq time-based to fix hydration ordering (commit fd7c65c).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/stream_coordinator.go — Time-based seq fallback
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/stream_coordinator_test.go — Updated fallback seq test


## 2026-02-03

Step 3: validate hydration ordering via tmux + Playwright; capture post-fix timeline payload.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/02/03/PI-008-ADDRESS-WEBCHAT-BUGS--address-webchat-hydration-inference-bugs/sources/timeline_9a62c624.json — Post-fix hydration payload

