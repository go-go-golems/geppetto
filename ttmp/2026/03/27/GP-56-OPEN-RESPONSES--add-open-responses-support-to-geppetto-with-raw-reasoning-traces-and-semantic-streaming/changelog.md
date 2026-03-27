# Changelog

## 2026-03-27

- Initial workspace created
- Added a detailed design doc describing how to generalize Geppetto's existing OpenAI Responses engine into provider-neutral Open Responses support
- Added a structured diary capturing the investigation, commands run, useful prior tickets, and the main architectural findings
- Expanded the task list into phased implementation slices with explicit commit and test boundaries
- Implemented Phase 1 provider plumbing so `open-responses` works as a first-class provider name while `openai-responses` remains a compatibility alias
- Extracted shared Responses provider identity and endpoint helpers so the engine and token counter are less OpenAI-name-specific
- Expanded reasoning block persistence to store raw reasoning text and summary payloads alongside encrypted content, and replay summaries into follow-up Responses requests
- Normalized legacy and alternate reasoning delta event names so Open Responses streams map onto Geppetto's existing reasoning-text and partial-thinking event model
- Added an operator-facing open-responses config/validation playbook, reran `docmgr doctor`, and refreshed the reMarkable bundle with the implemented state

## 2026-03-27

Wrote the intern-focused Open Responses design guide and investigation diary, grounded in the current openai_responses engine, turn model, event system, and prior reasoning bug tickets.

### Related Files

- /home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/ttmp/2026/03/27/GP-56-OPEN-RESPONSES--add-open-responses-support-to-geppetto-with-raw-reasoning-traces-and-semantic-streaming/design-doc/01-intern-guide-to-adding-open-responses-support-and-raw-reasoning-traces-in-geppetto.md — Primary analysis/design deliverable
- /home/manuel/workspaces/2026-03-27/use-open-responses/geppetto/ttmp/2026/03/27/GP-56-OPEN-RESPONSES--add-open-responses-support-to-geppetto-with-raw-reasoning-traces-and-semantic-streaming/reference/01-diary.md — Chronological record of research and documentation work
