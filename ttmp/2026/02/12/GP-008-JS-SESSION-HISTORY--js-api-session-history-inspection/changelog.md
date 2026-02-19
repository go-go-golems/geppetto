# Changelog

## 2026-02-12

- Initial workspace created


## 2026-02-12

Implemented JS session history APIs (turns/turnCount/getTurn/turnsRange), added snapshot-safe session helpers, unit tests, and a ticket smoke script.

### Related Files

- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/geppetto/ttmp/2026/02/12/GP-008-JS-SESSION-HISTORY--js-api-session-history-inspection/scripts/test_session_history_smoke.js — Added ticket smoke script
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/inference/session/session.go — Added TurnCount/GetTurn/TurnsSnapshot with clone semantics
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/api.go — Exposed history methods on JS session wrapper
- /home/manuel/workspaces/2026-02-12/geppetto-js/geppetto/pkg/js/modules/geppetto/module_test.go — Added history inspection and immutability test

