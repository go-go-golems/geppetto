# Changelog

## 2026-01-22

- Initial workspace created


## 2026-01-22

Created PI-001 ticket; analyzed Pinocchio getOrCreateConv closure wiring; proposed go-go-mento-style EngineBuilder + config signatures + subscriber factory; attempted reMarkable upload and generated PDF bundle.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/analysis/01-simplify-getorcreateconv-via-enginebuilder-pinocchio-webchat.md — Primary analysis doc
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/reference/01-diary.md — Investigation diary
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/various/PI-001 Webchat EngineBuilder Refactor.pdf — PDF bundle for reMarkable


## 2026-01-22

Reconcile PI-001 analysis with current code: EngineConfig/Signature + EngineBuilder + subscriber factory + signature-based rebuild implemented in pinocchio/pkg/webchat; fixed signature to use sanitized StepSettings metadata (avoid API key leakage) and added basic signature tests.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/22/PI-001-WEBCHAT-ENGINEBUILDER--pinocchio-webchat-refactor-getorcreateconv-to-enginebuilder-pattern/tasks.md — Mark tasks 1-8 complete
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_config.go — Signature now excludes secrets via StepSettings.GetMetadata
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/pinocchio/pkg/webchat/engine_config_test.go — Tests for deterministic


## 2026-01-22

Closing: core EngineBuilder refactor is implemented in pinocchio webchat; remaining items are follow-ups (tests for rebuild behavior; Moments migration sketch).

