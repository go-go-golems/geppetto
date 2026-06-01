# Changelog

## 2026-06-01

- Initial workspace created


## 2026-06-01

Created evidence-backed review and intern-facing design guide for a Go-backed fluent Geppetto JS API, including current-state analysis, gaps, proposed agent/turn/engine/embedding/schema builders, phased implementation plan, and diary.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Chronological investigation diary
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/scripts/01-collect-evidence.sh — Reproducible evidence collection script


## 2026-06-01

Validated ticket with docmgr doctor, committed docs (321fb82), and uploaded design+diary bundle to reMarkable at /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Included in uploaded bundle


## 2026-06-01

Revised the JS API design to a hard-cut ideal model: JavaScript manipulates direct Go-owned wrappers, hidden __geppetto_ref is only transitional implementation detail, map-first constructors leave the public contract, and unsafe raw-object import is explicitly isolated.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Hard-cut API model redesign
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Diary step recording redesign rationale


## 2026-06-01

Re-uploaded the updated hard-cut redesign bundle to reMarkable with --force at /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated hard-cut redesign included in reMarkable bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Delivery note for forced re-upload


## 2026-06-01

Clarified API naming and boundaries: profiles become inferenceProfiles only, gp.inferenceSettings builds provider/model settings, Pinocchio can back the default inference profile resolver, agents own system prompt/tools/middleware, and JS-side env/API-key methods are forbidden.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated naming and credential policy
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Diary step for naming clarification

