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


## 2026-06-01

Added analysis/design guide for extracting Pinocchio inline inference profile registry/config-document behavior into Geppetto so goja JS can resolve inference settings without importing Pinocchio.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/02-reusable-geppetto-inference-profile-registry-extraction-guide.md — New profile registry extraction design doc


## 2026-06-01

Uploaded updated reMarkable bundle including the new reusable Geppetto inference profile registry extraction guide.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/02-reusable-geppetto-inference-profile-registry-extraction-guide.md — Included in updated bundle
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Delivery note for updated bundle


## 2026-06-01

Uploaded the current design+diary bundle as reMarkable v2 under /ai/2026/06/01/GP-GOJA-API-2026-06-01.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v2 upload details


## 2026-06-01

Updated the primary JS API plan to use Geppetto registry sources directly through gp.inferenceProfiles.load(...), added a detailed phased implementation task list, and uploaded just that document as reMarkable v3.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated v3 JS API plan and task list
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v3 upload and design pivot


## 2026-06-01

Updated the primary JS API plan to remove gp.chat(), agent.ask(), and agent.system(); system/user/multimodal content now belongs in explicit Turn objects, agent.run(turn) is the execution API, tasks.md now contains detailed phased implementation tasks, and the doc was uploaded as reMarkable v4.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/design-doc/01-geppetto-go-go-goja-api-review-and-builder-design-guide.md — Updated v4 explicit-turn API plan
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/reference/01-investigation-diary.md — Recorded v4 update and upload
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/01/GP-GOJA-API-2026-06-01--review-and-redesign-geppetto-go-go-goja-api-and-javascript-bindings/tasks.md — Detailed phased implementation task list

