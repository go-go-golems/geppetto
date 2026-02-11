# Changelog

## 2026-01-20

- Initial workspace created


## 2026-01-20

Wrote lifecycle/cancellation analysis clarifying Conversation vs Inference and proposing API cleanup (StartInference handle, collapse RunInferenceStarted).

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/01-run-vs-conversation-vs-inference-lifecycle-cancellation-and-ownership.md — New analysis doc


## 2026-01-21

Added compendium doc consolidating sinks/session/state/tool-loop/cancellation Q&A with diagrams and file references.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/02-compendium-sinks-sessions-conversation-state-lifecycle-engines-tool-loops-q-a-diagrams.md — Norvig-style compendium


## 2026-01-21

Added local unit tests: Session sink injection + cancellation; toolhelpers.RunToolCallingLoop with a minimal echo tool. Linked compendium to MO-004 testing playbook.

### Related Files

- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/core/session_test.go — Session tests for sinks + cancellation
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/pkg/inference/toolhelpers/helpers_test.go — Tool loop test with fake engine + echo tool
- /home/manuel/workspaces/2025-10-30/implement-openai-responses-api/geppetto/ttmp/2026/01/20/MO-006-CLEANUP-CANCELLATION-LIFECYCLE--clarify-and-cleanup-cancellation-run-lifecycle-semantics/analysis/02-compendium-sinks-sessions-conversation-state-lifecycle-engines-tool-loops-q-a-diagrams.md — Playbook link


## 2026-01-25

Ticket closed

